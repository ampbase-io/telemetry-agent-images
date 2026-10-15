# The Ampbase supervisor managing upstream Fluent Bit. Nothing is compiled
# here: the supervisor's static binary is copied into Fluent Bit's own
# published image.
#
# Every FROM is pinned by tag and digest, and Renovate moves the pins.
# Keep the two stage lines below in their `FROM <image>:<tag>@<digest> AS
# <name>` form: scripts/image-versions reads the published image's tags
# off them, and supervisor-release.yml rewrites the supervisor line with
# a regex that matches only that form.
FROM ghcr.io/ampbase-io/supervisor:v0.17.0@sha256:3a37b7f9deba6e722f847081e51cda4b64a62e1f9b3e31355ff0729476e5ff20 AS supervisor

# ghcr.io rather than Docker Hub: the same index digest, without Hub's
# pull rate limit on CI runners.
FROM ghcr.io/fluent/fluent-bit:5.1.3@sha256:c5542543523c9678398dd78d927c05e8425ec15b038f226b67b3b019b1a70845 AS agent

# Writable directories for UID 10001, and the binary_path symlink. Runs on
# the build platform: it only creates directories and a link, so it needs
# no emulation for the other arch.
FROM --platform=$BUILDPLATFORM alpine:3.22@sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8 AS dirs
RUN mkdir /empty /links && ln -s /fluent-bit/bin/fluent-bit /links/fluent-bit

# Fluent Bit is dynamically linked against glibc and a dozen libraries
# that its image carries on a distroless Debian base, so that image is the
# base here and the static supervisor is what moves. Copying the binary out
# instead would mean copying its libraries and loader with it.
FROM agent

COPY --from=supervisor /usr/local/bin/ampbase /usr/local/bin/ampbase

# binary_path and config_path are the ones the supervisor's README gives
# for Fluent Bit. The binary stays where upstream installs it; the link is
# copied as a directory's contents, because COPY follows a symlink named
# as the source itself.
COPY --from=dirs /links/ /usr/local/bin/

# The supervisor writes the config with tmp + rename, so it must own the
# directory, not just the file. The mode has to be on the COPY: copying
# an empty directory to a new destination takes BuildKit's default mode
# (0755), whatever the source directory's mode was.
COPY --from=dirs --chown=10001:10001 --chmod=700 /empty /etc/fluent-bit
COPY --from=dirs --chown=10001:10001 --chmod=700 /empty /var/lib/opamp-supervisor

# fluent/fluent-bit v5.1.3 LICENSE, sha256 0d542e0c8804e39aa7f37eb00da5a762149dc682d7829451287e11b938e94594.
COPY LICENSE.fluent-bit /usr/share/doc/fluent-bit/LICENSE

# Upstream runs as root. 10001 is the UID every image in this repo runs
# as, so a host directory prepared for one prepares it for all of them.
USER 10001:10001

# 2020 is Fluent Bit's HTTP server (health and metrics); 9464 is the
# supervisor's metrics endpoint.
EXPOSE 2020 9464

ENTRYPOINT ["/usr/local/bin/ampbase"]
CMD ["--config", "/etc/ampbase/supervisor.yaml"]

ARG AGENT_VERSION
ARG SUPERVISOR_VERSION
LABEL org.opencontainers.image.title="Ampbase Fluent Bit" \
      org.opencontainers.image.description="Fluent Bit managed by the Ampbase supervisor" \
      org.opencontainers.image.vendor="Ampbase, LLC" \
      org.opencontainers.image.url="https://ampbase.io" \
      org.opencontainers.image.source="https://github.com/ampbase-io/telemetry-agent-images" \
      org.opencontainers.image.licenses="Apache-2.0 AND LicenseRef-Ampbase-CSA" \
      io.ampbase.agent.version="${AGENT_VERSION}" \
      io.ampbase.supervisor.version="${SUPERVISOR_VERSION}"
