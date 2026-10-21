# The Ampbase supervisor managing upstream Vector. Both binaries are
# copied out of their published, multi-arch images onto a distroless
# base; nothing is compiled here.
#
# Every FROM is pinned by tag and digest, and Renovate moves the pins.
# Keep the two stage lines below in their `FROM <image>:<tag>@<digest> AS
# <name>` form: scripts/image-versions reads the published image's tags
# off them, and supervisor-release.yml rewrites the supervisor line with
# a regex that matches only that form.
FROM ghcr.io/ampbase-io/supervisor:v0.18.0@sha256:bfa38eea7a5cd2b4f837305124c038d3ed68e98b5d6aad6faea7888d61fa2714 AS supervisor

# The -distroless-static variant, whose /usr/local/bin/vector is statically
# linked and so runs on distroless/static with no libc beneath it. ghcr.io
# rather than Docker Hub: the same index digest, without Hub's pull rate
# limit on CI runners.
FROM ghcr.io/vectordotdev/vector:0.59.0-distroless-static@sha256:2138c72ef15477d3a60126da851407a198ec61d2ff4cd600a4e69db5f6a29ff7 AS agent

# Writable directories for UID 10001. Runs on the build platform: it only
# creates empty directories, so it needs no emulation for the other arch.
FROM --platform=$BUILDPLATFORM alpine:3.22@sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8 AS dirs
RUN mkdir /empty

FROM gcr.io/distroless/static-debian12:latest@sha256:d75cdd72874d4790092fcb1b058493ecf6bb5bf2b2b897045b00ff01d91843f2

# binary_path and config_path are the ones the supervisor's README gives
# for Vector, and the binary's path in Vector's own image.
COPY --from=agent /usr/local/bin/vector /usr/local/bin/vector
COPY --from=supervisor /usr/local/bin/ampbase /usr/local/bin/ampbase

# The supervisor writes the config with tmp + rename, and writes the
# config it validates beside it, so it must own the directory, not just
# the file. /var/lib/vector is Vector's default data_dir, where disk
# buffers and checkpoints go. The mode has to be on the COPY: copying an
# empty directory to a new destination takes BuildKit's default mode
# (0755), whatever the source directory's mode was.
COPY --from=dirs --chown=10001:10001 --chmod=700 /empty /etc/vector
COPY --from=dirs --chown=10001:10001 --chmod=700 /empty /var/lib/vector
COPY --from=dirs --chown=10001:10001 --chmod=700 /empty /var/lib/opamp-supervisor

# vectordotdev/vector v0.59.0 LICENSE, sha256 60222c28c1a7f6a92c7df98e5c5f4459e624e6e285e0b9b94467af5f6ab3343d.
COPY LICENSE.vector /usr/share/doc/vector/LICENSE

# VECTOR_DANGEROUSLY_ALLOW_ENV_VAR_INTERPOLATION is deliberately not set.
# It lets config text read any variable in the container's environment,
# and the config here is delivered remotely; whether that is wanted is the
# operator's call, made with `docker run -e`, not the image's.

# Upstream runs as root. 10001 is the UID every image in this repo runs
# as, so a host directory prepared for one prepares it for all of them.
USER 10001:10001

# 8686 is Vector's API (health); 9464 is the supervisor's metrics
# endpoint.
EXPOSE 8686 9464

ENTRYPOINT ["/usr/local/bin/ampbase"]
CMD ["--config", "/etc/ampbase/supervisor.yaml"]

ARG AGENT_VERSION
ARG SUPERVISOR_VERSION
LABEL org.opencontainers.image.title="Ampbase Vector" \
      org.opencontainers.image.description="Vector managed by the Ampbase supervisor" \
      org.opencontainers.image.vendor="Ampbase, LLC" \
      org.opencontainers.image.url="https://ampbase.io" \
      org.opencontainers.image.source="https://github.com/ampbase-io/telemetry-agent-images" \
      org.opencontainers.image.licenses="MPL-2.0 AND LicenseRef-Ampbase-CSA" \
      io.ampbase.agent.version="${AGENT_VERSION}" \
      io.ampbase.supervisor.version="${SUPERVISOR_VERSION}"
