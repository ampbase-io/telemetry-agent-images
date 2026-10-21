# The Ampbase supervisor managing upstream Honeycomb Refinery. Both
# binaries are copied out of their published, multi-arch images onto a
# distroless base; nothing is compiled here.
#
# Every FROM is pinned by tag and digest, and Renovate moves the pins.
# Keep the two stage lines below in their `FROM <image>:<tag>@<digest> AS
# <name>` form: scripts/image-versions reads the published image's tags
# off them, and supervisor-release.yml rewrites the supervisor line with
# a regex that matches only that form.
FROM ghcr.io/ampbase-io/supervisor:v0.18.0@sha256:bfa38eea7a5cd2b4f837305124c038d3ed68e98b5d6aad6faea7888d61fa2714 AS supervisor

# Docker Hub, as the collector image uses. Public ECR serves the same
# index digest but answers the publish job's concurrent two-platform
# resolution with 429 Too Many Requests.
FROM honeycombio/refinery:3.4.1@sha256:34da28eac1987da612f8c152570b22fc289ef9fb84c37e5d97870c6135d9431a AS agent

# Writable directories for UID 10001. Runs on the build platform: it only
# creates empty directories, so it needs no emulation for the other arch.
FROM --platform=$BUILDPLATFORM alpine:3.22@sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8 AS dirs
RUN mkdir /empty

FROM gcr.io/distroless/static-debian12:latest@sha256:d75cdd72874d4790092fcb1b058493ecf6bb5bf2b2b897045b00ff01d91843f2

# Upstream's ko-built binary is statically linked, so it runs on
# distroless/static with no libc beneath it. binary_path is the one the
# Ampbase install wizard writes for Refinery.
COPY --from=agent /ko-app/refinery /usr/local/bin/refinery
COPY --from=supervisor /usr/local/bin/ampbase /usr/local/bin/ampbase

# There is no config directory: the supervisor writes both of Refinery's
# files into versioned directories under its state dir and swaps a
# symlink between them, so the state dir is the one that must be
# writable. The mode has to be on the COPY: copying an empty directory to
# a new destination takes BuildKit's default mode (0755), whatever the
# source directory's mode was.
COPY --from=dirs --chown=10001:10001 --chmod=700 /empty /var/lib/opamp-supervisor

# honeycombio/refinery v3.4.1 LICENSE, sha256 c71d239df91726fc519c6eb72d318ec65820627232b2f796219e87dcf35d0ab4.
COPY LICENSE.refinery /usr/share/doc/refinery/LICENSE

# Upstream runs as 65532. 10001 is the UID every image in this repo runs
# as, so a host directory prepared for one prepares it for all of them.
USER 10001:10001

# 8080 is Refinery's ListenAddr (incoming events and /alive); 8081 its
# default PeerListenAddr. 9464 is the supervisor's metrics endpoint.
EXPOSE 8080 8081 9464

ENTRYPOINT ["/usr/local/bin/ampbase"]
CMD ["--config", "/etc/ampbase/supervisor.yaml"]

ARG AGENT_VERSION
ARG SUPERVISOR_VERSION
LABEL org.opencontainers.image.title="Ampbase Refinery" \
      org.opencontainers.image.description="Honeycomb Refinery managed by the Ampbase supervisor" \
      org.opencontainers.image.vendor="Ampbase, LLC" \
      org.opencontainers.image.url="https://ampbase.io" \
      org.opencontainers.image.source="https://github.com/ampbase-io/telemetry-agent-images" \
      org.opencontainers.image.licenses="Apache-2.0 AND LicenseRef-Ampbase-CSA" \
      io.ampbase.agent.version="${AGENT_VERSION}" \
      io.ampbase.supervisor.version="${SUPERVISOR_VERSION}"
