<div align="center">
  <h1>webhookd</h1>
  <p>
    <img src="logo.png" alt="webhookd logo" width="320" />
  </p>
  <p><strong>Self-hosted webhook generator</strong> — create a webhook, get a URL, serve static JSON with custom headers.</p>

  <p>
    <a href="https://github.com/4thel00z/webhookd/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/4thel00z/webhookd/actions/workflows/ci.yml/badge.svg?branch=master" /></a>
    <a href="https://github.com/4thel00z/webhookd/actions/workflows/release-please.yml"><img alt="release-please" src="https://github.com/4thel00z/webhookd/actions/workflows/release-please.yml/badge.svg?branch=master" /></a>
    <a href="https://github.com/4thel00z/webhookd/actions/workflows/goreleaser.yml"><img alt="goreleaser" src="https://github.com/4thel00z/webhookd/actions/workflows/goreleaser.yml/badge.svg" /></a>
    <a href="https://github.com/4thel00z/webhookd/releases"><img alt="Release" src="https://img.shields.io/github/v/release/4thel00z/webhookd?sort=semver" /></a>
    <a href="COPYING"><img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-informational" /></a>
    <a href="https://github.com/4thel00z/webhookd/pkgs/container/webhookd"><img alt="GHCR" src="https://img.shields.io/badge/ghcr-webhookd-blue" /></a>
  </p>
</div>

## What is this?

`webhookd` is a small daemon that lets you:
- **create** a webhook (`POST /v1/webhooks`)
- **invoke** it at a stable URL (`/v1/hooks/{id}`) with the configured method
- **deactivate** it (`DELETE /v1/webhooks/{id}`)

The HTTP layer is built with [Fiber](https://github.com/gofiber/fiber) and documented with [Huma](https://github.com/danielgtaylor/huma) (OpenAPI + JSON Schema). The CLI uses [Fang](https://github.com/charmbracelet/fang).

## Quickstart

### Run locally (Go)

```bash
go run ./cmd/webhookd --port 1337
```

Or explicitly:

```bash
go run ./cmd/webhookd serve --port 1337
```

### Run with Docker (GHCR)

```bash
docker run --rm -p 1337:1337 ghcr.io/4thel00z/webhookd:latest --host 0.0.0.0 --port 1337
```

## API

### Create a webhook

```bash
curl -s -X POST http://localhost:1337/v1/webhooks \
  -H 'content-type: application/json' \
  -d '{"method":"GET","body":"hello","headers":{}}'
```

Response contains the `id` and `path` (Huma also adds `$schema`):

```json
{"$schema":"http://localhost:1337/schemas/Post-v1-webhooksResponse.json","id":"...","path":"/v1/hooks/..."}
```

### Invoke it

```bash
curl -s http://localhost:1337/v1/hooks/<id>
```

### Deactivate it

```bash
curl -s -X DELETE http://localhost:1337/v1/webhooks/<id>
```

### OpenAPI / docs

- **Docs UI**: `GET /docs`
- **OpenAPI**: `GET /openapi.json` and `GET /openapi.yaml`
- **JSON Schemas**: `GET /schemas/*`

## Configuration

By default, `webhookd` looks for `.webhookdrc.json`. If it doesn’t exist, it starts with defaults.

Example:

```json
{
  "enable_auth_on_options": false,
  "token_extractors": ["headers", "params"],
  "oauth_json_web_key_sets_url": "https://example.com/.well-known/jwks.json",
  "oauth_issuer": "https://example.com/",
  "oauth_audience": "my-audience"
}
```

## OpenTelemetry

Tracing is **disabled by default**. Enable it by setting either:
- `WEBHOOKD_OTEL_ENABLED=true`, or
- `OTEL_EXPORTER_OTLP_ENDPOINT` / `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`

Useful env vars:
- `WEBHOOKD_OTEL_ENABLED`: `true|false`
- `WEBHOOKD_OTEL_EXPORTER_OTLP_ENDPOINT`: OTLP HTTP endpoint (full URL or `host:port`). Example: `http://localhost:4318`
- `WEBHOOKD_OTEL_EXPORTER_OTLP_HEADERS`: comma-separated `k=v` pairs
- `WEBHOOKD_OTEL_EXPORTER_OTLP_INSECURE`: when using `host:port`, prefixes with `http://` instead of `https://`
- `WEBHOOKD_OTEL_TRACES_SAMPLER_RATIO`: `0..1` (default `1`)
- `WEBHOOKD_OTEL_SERVICE_NAME`: override service name (default `webhookd`)
- `WEBHOOKD_ENV` / `ENV`: sets `deployment.environment.name`


## Development

```bash
go test ./...
```

## License

MIT — see [`COPYING`](COPYING).
