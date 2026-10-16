// Package smoke boots a built image against a stand-in OpAMP server and
// waits for the supervisor to report that the agent is running the config
// the server delivered.
//
// Inputs, both required:
//
//	SMOKE_IMAGE  the image reference to run (already present locally)
//	SMOKE_CASE   the image's smoke directory, holding supervisor.yaml (a
//	             text/template over {{.Endpoint}}) and the remote config
//
// The remote config is one of two shapes, never both. A file
// remote-config.yaml is delivered as the agent's one config file, under
// the config-map key "". A directory remote-config/ is delivered as a
// map of named files: each <key>.yaml in it under the key <key>, for
// agents whose adapter takes more than one file (Refinery's
// refinery_config and refinery_rules).
package smoke

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server"
	"github.com/open-telemetry/opamp-go/server/types"
)

// How long the agent gets to apply its config and turn healthy. A cold
// start pulls nothing (the image is local) but does page in a large static
// binary; on a CI runner that is a few seconds, so this is generous rather
// than tight. On expiry the test fails with the last state it saw and the
// container's logs.
const healthyWithin = 90 * time.Second

func TestImage(t *testing.T) {
	t.Parallel()

	image := requireEnv(t, "SMOKE_IMAGE")
	caseDir := requireEnv(t, "SMOKE_CASE")

	opamp := newFakeServer(t, readRemoteConfig(t, caseDir))

	supervisorYAML := renderSupervisorConfig(t, filepath.Join(caseDir, "supervisor.yaml"), opamp.endpoint)
	name := runContainer(t, image, supervisorYAML)

	ctx, cancel := context.WithTimeout(t.Context(), healthyWithin)
	defer cancel()
	if err := opamp.waitApplied(ctx); err != nil {
		t.Fatalf("%v\n--- container logs ---\n%s", err, containerLogs(name))
	}
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Fatalf("%s is not set", key)
	}
	return v
}

// agentState is the latest status the supervisor has reported. OpAMP
// omits unchanged fields from a message, so each field keeps its last
// reported value rather than being reset by a message that leaves it out.
type agentState struct {
	configStatus *protobufs.RemoteConfigStatus
	health       *protobufs.ComponentHealth
}

func (s agentState) String() string {
	return fmt.Sprintf("remote config %x %s %q; health: %v",
		s.configStatus.GetLastRemoteConfigHash(), s.configStatus.GetStatus(),
		s.configStatus.GetErrorMessage(), s.health)
}

// readRemoteConfig reads the case's remote config as the config map the
// server delivers; see the package comment for the two shapes.
func readRemoteConfig(t *testing.T, caseDir string) map[string][]byte {
	t.Helper()
	single := filepath.Join(caseDir, "remote-config.yaml")
	files, err := filepath.Glob(filepath.Join(caseDir, "remote-config", "*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	_, statErr := os.Stat(single)
	switch {
	case statErr == nil && len(files) > 0:
		t.Fatalf("%s has both remote-config.yaml and remote-config/; keep one", caseDir)
	case statErr == nil:
		body, err := os.ReadFile(single)
		if err != nil {
			t.Fatal(err)
		}
		return map[string][]byte{"": body}
	case len(files) == 0:
		t.Fatalf("%s has neither remote-config.yaml nor remote-config/*.yaml", caseDir)
	}
	config := make(map[string][]byte, len(files))
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		config[strings.TrimSuffix(filepath.Base(f), ".yaml")] = body
	}
	return config
}

// configHash identifies a config map by its keys and bodies, each
// length-prefixed so no two maps share an encoding.
func configHash(config map[string][]byte) []byte {
	keys := make([]string, 0, len(config))
	for k := range config {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%d:%s%d:", len(k), k, len(config[k]))
		h.Write(config[k])
	}
	return h.Sum(nil)
}

type fakeServer struct {
	endpoint string
	hash     []byte
	config   map[string][]byte

	mu      sync.Mutex
	state   agentState
	changed chan struct{}
}

func newFakeServer(t *testing.T, config map[string][]byte) *fakeServer {
	t.Helper()

	// The container runs on the host network, so loopback reaches it.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	fs := &fakeServer{
		endpoint: fmt.Sprintf("http://%s/v1/opamp", lis.Addr()),
		hash:     configHash(config),
		config:   config,
		changed:  make(chan struct{}, 1),
	}

	srv := server.New(testLogger{t})
	err = srv.Start(server.StartSettings{
		Listener: lis,
		Settings: server.Settings{
			Callbacks: types.Callbacks{
				OnConnecting: func(*http.Request) types.ConnectionResponse {
					return types.ConnectionResponse{
						Accept:              true,
						ConnectionCallbacks: types.ConnectionCallbacks{OnMessage: fs.onMessage},
					}
				},
			},
		},
	})
	if err != nil {
		_ = lis.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })
	return fs
}

func (fs *fakeServer) onMessage(_ context.Context, _ types.Connection, msg *protobufs.AgentToServer) *protobufs.ServerToAgent {
	fs.record(msg)

	resp := &protobufs.ServerToAgent{
		InstanceUid: msg.GetInstanceUid(),
		Capabilities: uint64(protobufs.ServerCapabilities_ServerCapabilities_AcceptsStatus |
			protobufs.ServerCapabilities_ServerCapabilities_OffersRemoteConfig |
			protobufs.ServerCapabilities_ServerCapabilities_AcceptsEffectiveConfig),
	}
	if !bytes.Equal(fs.reportedHash(), fs.hash) {
		files := make(map[string]*protobufs.AgentConfigObject, len(fs.config))
		for key, body := range fs.config {
			files[key] = &protobufs.AgentConfigObject{Body: body, ContentType: "text/yaml"}
		}
		resp.RemoteConfig = &protobufs.AgentRemoteConfig{
			ConfigHash: fs.hash,
			Config:     &protobufs.AgentConfigMap{ConfigMap: files},
		}
	}
	return resp
}

func (fs *fakeServer) record(msg *protobufs.AgentToServer) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if s := msg.GetRemoteConfigStatus(); s != nil {
		fs.state.configStatus = s
	}
	if h := msg.GetHealth(); h != nil {
		fs.state.health = h
	}
	select {
	case fs.changed <- struct{}{}:
	default:
	}
}

func (fs *fakeServer) reportedHash() []byte {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.state.configStatus.GetLastRemoteConfigHash()
}

func (fs *fakeServer) snapshot() agentState {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.state
}

// waitApplied returns once the supervisor reports this server's config as
// APPLIED and the agent as healthy, or with the last state seen when ctx
// ends. A FAILED status ends the wait immediately: the supervisor will not
// retry a hash it has refused.
func (fs *fakeServer) waitApplied(ctx context.Context) error {
	for {
		s := fs.snapshot()
		status := s.configStatus.GetStatus()
		ours := bytes.Equal(s.configStatus.GetLastRemoteConfigHash(), fs.hash)
		switch {
		case ours && status == protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED:
			return fmt.Errorf("supervisor refused the config: %s", s)
		case ours && status == protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED && s.health.GetHealthy():
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("agent not applied and healthy within %s; %s", healthyWithin, s)
		case <-fs.changed:
		}
	}
}

func renderSupervisorConfig(t *testing.T, path, endpoint string) string {
	t.Helper()
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct{ Endpoint string }{endpoint}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "supervisor.yaml")
	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	// The image runs as UID 10001, which reads the mounted file through
	// its own mode alone. Chmod rather than trust WriteFile's mode, which
	// the umask can narrow.
	if err := os.Chmod(out, 0o644); err != nil {
		t.Fatal(err)
	}
	return out
}

// runContainer starts the image the way a customer would: the supervisor
// config mounted read-only at the image's default --config path.
func runContainer(t *testing.T, image, supervisorYAML string) string {
	t.Helper()
	name := fmt.Sprintf("smoke-%d", time.Now().UnixNano())
	out, err := exec.CommandContext(t.Context(), "docker", "run", "--detach",
		"--name", name,
		"--network", "host",
		"--volume", supervisorYAML+":/etc/ampbase/supervisor.yaml:ro",
		image,
	).CombinedOutput()
	if err != nil {
		t.Fatalf("docker run: %v\n%s", err, out)
	}
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "--force", name).Run() })
	return name
}

func containerLogs(name string) string {
	out, err := exec.Command("docker", "logs", name).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("(docker logs failed: %v)\n%s", err, out)
	}
	return string(out)
}

type testLogger struct{ t *testing.T }

func (l testLogger) Debugf(_ context.Context, format string, v ...any) { l.t.Logf(format, v...) }
func (l testLogger) Errorf(_ context.Context, format string, v ...any) {
	l.t.Logf("ERROR "+format, v...)
}
