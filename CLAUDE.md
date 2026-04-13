# CLAUDE.md

## Project Overview

**go-webhook** — A configurable webhook forwarding engine. Receives JSON webhooks, matches against YAML rules using the expr expression engine to dynamically assemble new payloads, and forwards them via HTTP to target services. Zero-code webhook data transformation and routing.

## Tech Stack

Go 1.26+ | Gin | Cobra + Viper | expr-lang/expr | Zap | Prometheus | swaggo/swag + gin-swagger | testify | golangci-lint | Air (dev hot reload) | GoReleaser | Docker → ghcr.io | Helm chart → ghcr.io OCI

## Architecture

### Dual-Port Design

Business port `:8080` (`server.port`) — webhook receive and forward. Admin port `:9090` (`admin.port`) — separate Gin engine serving:

- `GET /admin/config` — runtime config JSON (sensitive fields redacted)
- `GET /admin/rules` — current rule list (with compile status)
- `GET /admin/healthz` — health check
- `GET /metrics` — Prometheus
- `GET /swagger/*` — Swagger UI

Both servers start together in the `serve` command and shut down gracefully together.

### Directory Structure

```
go-webhook/
├── main.go
├── cmd/                         # Cobra: root.go, serve.go, validate.go
├── internal/
│   ├── config/                  # Viper init + rule loading + hot reload + type definitions
│   ├── engine/                  # matcher, transformer, dispatcher, functions
│   ├── handler/                 # webhook.go (business), admin.go (admin)
│   ├── server/                  # webhook.go / admin.go — two Gin engine initializations
│   ├── middleware/              # auth, logging, metrics
│   ├── logger/                  # zap wrapper, json/text switching
│   └── metrics/                 # Prometheus metric definitions
├── docs/                        # swag init auto-generated, do not edit manually
├── configs/                     # config.yaml + rules.yaml
├── test/fixtures/               # test payload samples
├── test/e2e/                    # E2E tests (make e2e)
├── deploy/helm/go-webhook/      # Helm chart
├── .github/workflows/release.yml
├── .goreleaser.yaml
├── .air.toml
├── .vscode/launch.json          # shared across team, committed to git
├── Dockerfile
└── Makefile
```

## Key Commands

```bash
make dev          # Air hot reload dev mode
make test         # unit + integration tests (-v -race -coverprofile)
make e2e          # E2E tests (-tags=e2e)
make build        # build binary
swag init -g main.go -o docs/   # regenerate Swagger docs
go run . serve --config configs/config.yaml
go run . validate --rules configs/rules.yaml
```

## Development Rules

### Viper Configuration
- Key naming: dot-separated lowercase: `server.port`, `admin.port`, `log.format`, `rules.path`
- Env var prefix `GOWEBHOOK_`: `GOWEBHOOK_SERVER_PORT` → `server.port`
- Priority: flag > env > yaml > default. All config keys must have a default value
- Search paths: `./configs/`, `$HOME/.go-webhook/`, `/etc/go-webhook/`
- config.yaml hot reload via Viper WatchConfig; rules.yaml watched separately via fsnotify
- Reload atomically swaps rule set (atomic.Value); on load failure, retains previous rules

### Gin
- Business middleware order: Recovery → Logging → Metrics → Auth → Handler
- Admin port middleware: Recovery → Logging (lightweight), no auth
- Unified error response: `{"error": "message", "code": "ERROR_CODE"}`
- Use `ShouldBindJSON()` to parse request body

### Admin API
- `/admin/config` must redact sensitive fields: values of keys containing `token`/`secret`/`password`/`key` replaced with `"******"`
- Admin port must not be exposed publicly; restrict access via network policies

### Swagger
- swaggo annotations go on handler functions, generated to `docs/`, committed to git
- CI validates no diff after `swag init`
- Swagger UI mounted on admin port: `adminRouter.GET("/swagger/*any", ginSwagger.WrapHandler(...))`
- Request/response structs defined in `internal/handler/` or `internal/config/types.go`

### Zap Logging
- `log.format`: `json` (default, zap.NewProduction) or `text` (zap.NewDevelopment)
- Request logs include: method, path, status, latency, client_ip, request_id
- Engine logs use `logger.With(zap.String("rule", ruleName))`

### Prometheus Metrics
- Required metrics: `webhook_requests_total` (counter, labels: status/path), `webhook_request_duration_seconds` (histogram), `webhook_dispatch_total` (counter, labels: rule_name/target/status), `webhook_dispatch_duration_seconds` (histogram), `webhook_rules_loaded` (gauge)
- Register with `promauto`; metric constants centralized in `internal/metrics/`

### Expr Expressions
- Compile at load time, only Run at runtime. Built-in functions: `now()`, `env()`, `lower()`, `upper()`, `join()`, `split()`
- Evaluation errors must not panic; log and skip the rule

### HTTP Forwarding
- Default timeout 10s (overridable per rule), exponential backoff retry 3 times (5xx/network errors only)
- Custom headers support `{{env.VAR}}` template

### TDD (ECC tdd-workflow)
- New features and bug fixes **must** follow RED → GREEN → REFACTOR
- RED: write tests first, `make test` confirms failure due to missing implementation (not env issues). **Do not write production code before RED is confirmed**
- GREEN: minimal implementation, `make test` confirms all pass
- REFACTOR: refactor, `make test` still green
- Git checkpoint per phase: `test: add <feature>` → `feat: implement <feature>` → `refactor: clean up <feature>`
- Coverage: overall 80%+, `internal/engine/` 90%+
- Unit tests (`*_test.go` colocated) + integration tests (`//go:build integration`) → `make test`
- E2E tests (`test/e2e/`, `-tags=e2e`) → `make e2e`
- **Do not run `go test` directly**; Makefile wraps the correct flags
- Mocks: HTTP via `httptest.NewServer`, Expr via `test/fixtures/` payloads

### CI/CD
- Tag `v*` triggers 4 parallel-after-test jobs: test → release (GoReleaser) + docker + helm
- **GoReleaser handles binaries only** — 4 binaries (darwin/linux × amd64/arm64) + GitHub Release + checksums. GoReleaser must NOT build or push Docker images (no `dockers` / `docker_manifests` sections in `.goreleaser.yaml`)
- **Docker image built in a separate CI job** using `docker/build-push-action` with multi-platform (`linux/amd64,linux/arm64`), GHA cache, and `docker/metadata-action` for tags
- **Dockerfile must use multi-stage build**: `golang:1.26-alpine` builder → `distroless/static-debian12` runtime. This ensures reproducible builds independent of GoReleaser's build context

### Helm Chart
- Source at `deploy/helm/go-webhook/`; CI auto-replaces Chart.yaml version/appVersion
- values.yaml defaults match config.yaml; ConfigMap mounts both config files
- Install: `helm install go-webhook oci://ghcr.io/<owner>/charts/go-webhook --version <ver>`

### Air + VSCode
- `make dev` = `air -c .air.toml`, watches .go/.yaml, builds with `-gcflags='all=-N -l'` for debug
- `.vscode/launch.json`: Launch directly + Attach to Air process. Includes `GIN_MODE=debug`

### README.md
- README.md is the project's public-facing entry document and must stay in sync with the code
- Covers: project overview, features, quick start (install/run), configuration, rule authoring examples, deployment (Docker/Helm), dev guide (make commands), API endpoint list
- Code changes **must** update README.md when:
  - CLI commands or flags are added/removed/changed
  - API endpoints are added/removed/changed
  - Config structure changes (config.yaml / rules.yaml)
  - New expr built-in functions are added
  - Deployment changes (Dockerfile / Helm chart / CI)
  - Make targets change
- Example code and commands in README must be copy-paste runnable

## Don'ts

- Expr logic belongs in the engine package only; handlers must not operate on expr directly
- Do not bypass Viper to read env (no `os.Getenv`); do not bypass zap to log (no `fmt.Println`)
- `/metrics`, `/swagger`, `/admin/*` only on admin port; never on business port
- `/admin/config` must not return plaintext secrets
- `docs/` is swag-generated; do not edit manually
- Helm chart must not hardcode image tag; use `{{ .Values.image.tag | default .Chart.AppVersion }}`
- `.goreleaser.yaml` must not hardcode version numbers or contain `dockers`/`docker_manifests` sections
- Dockerfile must not use single-stage (COPY binary only) pattern; always use multi-stage build from source
- `tmp/` must not be committed to git
- Do not write non-test files under `internal/` before RED phase is confirmed
- Tests must only use `make test` / `make e2e`; do not run `go test` directly
