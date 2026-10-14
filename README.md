# Ampbase telemetry agent images

Container images that bundle the Ampbase supervisor with a telemetry
agent, ready to run. Point the supervisor at your Ampbase channel and it
manages the agent's configuration from then on.

| Agent | Image |
|---|---|
| OpenTelemetry Collector (contrib) | `ghcr.io/ampbase-io/otelcol-contrib` |
| Fluent Bit | *planned* |
| Vector | *planned* |
| Telegraf | *planned* |
| Refinery | *planned* |

## Running

Write the `supervisor.yaml` the Ampbase install wizard gives you, then:

```sh
docker run -d --name ampbase --restart unless-stopped \
  -v "$(pwd)/supervisor.yaml:/etc/ampbase/supervisor.yaml:ro" \
  -v ampbase-state:/var/lib/opamp-supervisor \
  ghcr.io/ampbase-io/otelcol-contrib:0.162.0-ampbase0.17.0
```

Keep the `ampbase-state` volume. It holds the agent's identity and its
last good config, so a container recreated from a newer tag stays the
same agent rather than enrolling as a new one.

The collector is at `/usr/bin/otelcol` and its config at
`/etc/otelcol/config.yaml`, the paths the wizard's Linux snippet already
uses. The image runs as UID 10001.

## Tags

| Tag | Moves when |
|---|---|
| `0.162.0-ampbase0.17.0` | only when the image's other inputs change (a base-image patch) |
| `0.162.0` | the supervisor is released |
| `latest` | the agent or the supervisor is released |

Pin the first form, or a digest, in production.

## Keeping up to date

Images rebuild and republish without anyone touching them:

- **Agent and base-image releases**: Renovate opens a PR that moves the
  pin in the Dockerfile. It merges itself once the smoke test passes on
  amd64 and arm64. New agent versions wait three days first.
- **Supervisor releases**: the Ampbase release workflow notifies this
  repo. `supervisor-release.yml` waits for the supervisor image, commits
  the new pin to `main`, and builds.

## Verifying an image

Every image is signed with cosign (keyless) and carries SLSA build
provenance:

```sh
cosign verify ghcr.io/ampbase-io/otelcol-contrib:latest \
  --certificate-identity-regexp '^https://github.com/ampbase-io/telemetry-agent-images/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com

gh attestation verify oci://ghcr.io/ampbase-io/otelcol-contrib:latest \
  --repo ampbase-io/telemetry-agent-images
```

## Adding an agent

Each image is a directory under `images/` holding:

- `Dockerfile`, with the supervisor and the agent as tagged, digest-pinned
  stages named `AS supervisor` and `AS agent`.
  `scripts/image-versions` reads the image tags off those two lines.
- `smoke/supervisor.yaml`, a template over `{{.Endpoint}}`.
- `smoke/remote-config.yaml`, the config the smoke test's OpAMP server
  delivers.

To run the smoke test locally (Linux only: the container reaches the
test's OpAMP server over host networking, which Docker Desktop lacks):

```sh
docker build -t smoke:otelcol-contrib images/otelcol-contrib
cd smoke && SMOKE_IMAGE=smoke:otelcol-contrib \
  SMOKE_CASE=$PWD/../images/otelcol-contrib/smoke go test -v ./...
```

The test passes only when the supervisor reports that the delivered
config was applied and that the agent is healthy.
