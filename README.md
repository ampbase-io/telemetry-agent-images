# Ampbase telemetry agent images

Container images that bundle the Ampbase supervisor with a telemetry
agent, ready to run. Point the supervisor at your Ampbase channel and it
manages the agent's configuration from then on.

| Agent | Image |
|---|---|
| OpenTelemetry Collector (contrib) | `ghcr.io/ampbase-io/otelcol-contrib` *(coming soon)* |
| Fluent Bit | *planned* |
| Vector | *planned* |
| Telegraf | *planned* |
| Refinery | *planned* |

Images are rebuilt automatically when the agent, the supervisor, or the
base image publishes a new release, and every image is signed with
cosign and carries SLSA build provenance.

Setup and configuration: <https://ampbase.io/docs/install-supervisor>.
