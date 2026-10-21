# The Ampbase supervisor managing the upstream OpenTelemetry Collector
# (contrib distribution). Both binaries are copied out of their published,
# multi-arch images onto a distroless base; nothing is compiled here.
#
# Every FROM is pinned by tag and digest, and Renovate moves the pins.
# Keep the two stage lines below in their `FROM <image>:<tag>@<digest> AS
# <name>` form: scripts/image-versions reads the published image's tags
# off them, and supervisor-release.yml rewrites the supervisor line with
# a regex that matches only that form.
FROM ghcr.io/ampbase-io/supervisor:v0.18.0@sha256:bfa38eea7a5cd2b4f837305124c038d3ed68e98b5d6aad6faea7888d61fa2714 AS supervisor

FROM otel/opentelemetry-collector-contrib:0.162.0@sha256:39923a8e431bd1f57be82411999d389fcfe40857492e4365456d97a4c1f74be6 AS agent

# Writable directories for UID 10001. Runs on the build platform: it only
# creates empty directories, so it needs no emulation for the other arch.
FROM --platform=$BUILDPLATFORM alpine:3.22@sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8 AS dirs
RUN mkdir /empty

FROM gcr.io/distroless/static-debian12:latest@sha256:d75cdd72874d4790092fcb1b058493ecf6bb5bf2b2b897045b00ff01d91843f2

# binary_path and config_path are the paths the Ampbase install wizard
# writes into supervisor.yaml for a Linux collector, so a snippet copied
# from the wizard runs against this image unedited.
COPY --from=agent /otelcol-contrib /usr/bin/otelcol
COPY --from=supervisor /usr/local/bin/ampbase /usr/local/bin/ampbase

# The supervisor writes the collector's config with tmp + rename, so it
# must own the directory, not just the file. The mode has to be on the
# COPY: copying an empty directory to a new destination takes BuildKit's
# default mode (0755), whatever the source directory's mode was.
COPY --from=dirs --chown=10001:10001 --chmod=700 /empty /etc/otelcol
COPY --from=dirs --chown=10001:10001 --chmod=700 /empty /var/lib/opamp-supervisor

COPY LICENSE.otelcol-contrib /usr/share/doc/otelcol-contrib/LICENSE

# The collector image's own UID, so volumes already owned by it carry over.
USER 10001:10001

# 13133 is the collector's health_check extension; 4317/4318 are OTLP.
# 9464 is the supervisor's metrics endpoint.
EXPOSE 4317 4318 9464 13133

ENTRYPOINT ["/usr/local/bin/ampbase"]
CMD ["--config", "/etc/ampbase/supervisor.yaml"]

ARG AGENT_VERSION
ARG SUPERVISOR_VERSION
LABEL org.opencontainers.image.title="Ampbase OpenTelemetry Collector (contrib)" \
      org.opencontainers.image.description="OpenTelemetry Collector contrib managed by the Ampbase supervisor" \
      org.opencontainers.image.vendor="Ampbase, LLC" \
      org.opencontainers.image.url="https://ampbase.io" \
      org.opencontainers.image.source="https://github.com/ampbase-io/telemetry-agent-images" \
      org.opencontainers.image.licenses="Apache-2.0 AND LicenseRef-Ampbase-CSA" \
      io.ampbase.agent.version="${AGENT_VERSION}" \
      io.ampbase.supervisor.version="${SUPERVISOR_VERSION}"
