# go-webhook

Configurable webhook forwarding engine. Receives JSON webhooks, matches against YAML rules using [expr](https://github.com/expr-lang/expr) expressions, dynamically transforms the payload and target URL, and forwards via HTTP. Zero-code webhook data transformation and routing.

## Features

- **Expression-based rule engine** — match, transform URL, and build request body using expr expressions
- **Hot reload** — rules.yaml changes are picked up automatically via fsnotify (no restart needed)
- **Dual-port architecture** — business port (`:8080`) for webhooks, admin port (`:9090`) for ops
- **Retry with backoff** — exponential backoff retry on 5xx / network errors (configurable per rule)
- **Observability** — Prometheus metrics, structured JSON logging (Zap), Swagger UI
- **Deployment ready** — Docker (distroless), Helm chart, GoReleaser, GitHub Actions CI/CD

## Quick Start

### Install from source

```bash
go install github.com/shidaxi/go-webhook@latest
```

### Build locally

```bash
git clone https://github.com/shidaxi/go-webhook.git
cd go-webhook
make build
```

### Run

```bash
./go-webhook serve --config configs/config.yaml
```

The webhook server starts on `:8080` and the admin server on `:9090`.

### Validate rules

```bash
./go-webhook validate --rules configs/rules.yaml
```

## Configuration

### config.yaml

```yaml
server:
  port: 8080

admin:
  port: 9090

log:
  format: json   # json or text

rules:
  path: configs/rules.yaml
```

All settings support environment variable override with prefix `GOWEBHOOK_`:

```bash
GOWEBHOOK_SERVER_PORT=3000 ./go-webhook serve
```

Config search paths: `./configs/`, `$HOME/.go-webhook/`, `/etc/go-webhook/`.

### rules.yaml

Rules define how incoming webhooks are matched, transformed, and forwarded:

```yaml
rules:
  - name: alertmanager-to-lark
    match: 'len(payload.alerts) > 0'
    target:
      url: '"https://open.larksuite.com/open-apis/bot/v2/hook/" + payload.alerts[0].labels.lark_bot_id'
      method: POST
      timeout: 10s
      headers:
        Content-Type: application/json
    body: |
      {
        "msg_type": "interactive",
        "card": {
          "header": {
            "title": {
              "content": (payload.alerts[0].status == "firing" ? "fire " : "check ") + payload.alerts[0].labels.alertname,
              "tag": "plain_text"
            },
            "template": payload.alerts[0].status == "firing" ? "red" : "green"
          },
          "elements": [
            {
              "tag": "div",
              "text": {
                "content": "**Instance:** " + payload.alerts[0].labels.instance + "\n**Severity:** " + payload.alerts[0].labels.severity + "\n**Summary:** " + payload.alerts[0].annotations.summary,
                "tag": "lark_md"
              }
            }
          ]
        }
      }
```

Each rule has:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Rule identifier |
| `match` | expr (bool) | Expression evaluated against `payload`; rule fires when `true` |
| `target.url` | expr (string) | Target URL expression |
| `target.method` | string | HTTP method (default: `POST`) |
| `target.timeout` | duration | Request timeout (default: `10s`) |
| `target.headers` | map | Custom HTTP headers |
| `body` | expr (map) | Expression that builds the JSON body to send |

### Built-in expr functions

| Function | Example | Description |
|----------|---------|-------------|
| `now()` | `now()` | Current time (time.Time) |
| `env(name)` | `env("API_KEY")` | Read environment variable |
| `lower(s)` | `lower("ABC")` → `"abc"` | Lowercase string |
| `upper(s)` | `upper("abc")` → `"ABC"` | Uppercase string |
| `join(arr, sep)` | `join(["a","b"], ",")` → `"a,b"` | Join array with separator |
| `split(s, sep)` | `split("a,b", ",")` → `["a","b"]` | Split string by separator |

## API Endpoints

### Business port (default `:8080`)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/webhook` | Receive JSON payload, match rules, transform and forward |

### Admin port (default `:9090`)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/healthz` | Health check |
| GET | `/admin/rules` | List loaded rules with compile status |
| GET | `/admin/config` | Runtime config (sensitive fields redacted) |
| GET | `/metrics` | Prometheus metrics |
| GET | `/swagger/*` | Swagger UI |

## Example: Alertmanager to Lark

Send a firing alert:

```bash
curl -X POST http://localhost:8080/webhook \
  -H 'Content-Type: application/json' \
  -d '{
    "version": "4",
    "status": "firing",
    "alerts": [{
      "status": "firing",
      "labels": {
        "alertname": "HighCPU",
        "severity": "warning",
        "instance": "web-01",
        "lark_bot_id": "your-bot-id"
      },
      "annotations": {
        "summary": "CPU usage > 80%"
      }
    }]
  }'
```

Response:

```json
{
  "matched": 1,
  "dispatched": 1,
  "results": [
    {
      "RuleName": "alertmanager-to-lark",
      "TargetURL": "https://open.larksuite.com/open-apis/bot/v2/hook/your-bot-id",
      "StatusCode": 200,
      "Success": true,
      "Error": null
    }
  ]
}
```

## Deployment

### Docker

```bash
docker build -t go-webhook .
docker run -p 8080:8080 -p 9090:9090 -v $(pwd)/configs:/app/configs go-webhook
```

### Helm

```bash
helm install go-webhook oci://ghcr.io/<owner>/charts/go-webhook --version <ver>
```

Or from source:

```bash
helm install go-webhook deploy/helm/go-webhook/ \
  --set config.server.port=8080 \
  --set config.admin.port=9090
```

### CI/CD

Tag `v*` triggers GitHub Actions:
1. Run tests
2. GoReleaser builds binaries (darwin/linux x amd64/arm64) → GitHub Release
3. Docker multi-arch image (linux/amd64, linux/arm64) built and pushed to ghcr.io
4. Helm chart packaged and pushed as OCI to ghcr.io

## Development

```bash
make dev          # Hot reload dev server (Air)
make test         # Unit + integration tests (-race -cover)
make e2e          # E2E tests (-tags=e2e)
make bench        # Benchmark suite (-benchmem -count=3 -cpu=1,4)
make build        # Build binary
make lint         # golangci-lint
make swagger      # Regenerate Swagger docs
make clean        # Remove binary and coverage
```

## Prometheus Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `webhook_requests_total` | counter | status, path |
| `webhook_request_duration_seconds` | histogram | |
| `webhook_dispatch_total` | counter | rule_name, target, status |
| `webhook_dispatch_duration_seconds` | histogram | rule_name |
| `webhook_rules_loaded` | gauge | |

## License

MIT
