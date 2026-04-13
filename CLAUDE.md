# CLAUDE.md

## Project Overview

**go-webhook** — 可配置的 webhook 转发引擎。接收 JSON webhook，根据 YAML 规则用 expr 表达式引擎动态组装新 payload，HTTP 转发给目标服务。零代码实现 webhook 数据转换和路由。

## Tech Stack

Go 1.22+ | Gin | Cobra + Viper | expr-lang/expr | Zap | Prometheus | swaggo/swag + gin-swagger | testify | golangci-lint | Air (dev hot reload) | GoReleaser | Docker → ghcr.io | Helm chart → ghcr.io OCI

## Architecture

### 双端口设计

业务端口 `:8080`（`server.port`）— webhook 接收转发。管理端口 `:9090`（`admin.port`）— 独立 Gin engine，承载：

- `GET /admin/config` — 运行时配置 JSON（脱敏）
- `GET /admin/rules` — 当前规则列表（含编译状态）
- `GET /admin/healthz` — 健康检查
- `GET /metrics` — Prometheus
- `GET /swagger/*` — Swagger UI

两个 server 在 `serve` 命令中同时启动，graceful shutdown 同时关闭。

### 目录结构

```
go-webhook/
├── main.go
├── cmd/                         # Cobra: root.go, serve.go, validate.go
├── internal/
│   ├── config/                  # Viper 初始化 + 规则加载 + 热重载 + 类型定义
│   ├── engine/                  # matcher, transformer, dispatcher, functions
│   ├── handler/                 # webhook.go (业务), admin.go (管理)
│   ├── server/                  # webhook.go / admin.go — 两个 Gin engine 初始化
│   ├── middleware/              # auth, logging, metrics
│   ├── logger/                  # zap 封装，json/text 切换
│   └── metrics/                 # Prometheus 指标定义
├── docs/                        # swag init 自动生成，勿手动编辑
├── configs/                     # config.yaml + rules.yaml
├── test/fixtures/               # 测试 payload 样本
├── test/e2e/                    # E2E 测试（make e2e）
├── deploy/helm/go-webhook/      # Helm chart
├── .github/workflows/release.yml
├── .goreleaser.yaml
├── .air.toml
├── .vscode/launch.json          # 团队共享，纳入 git
├── Dockerfile
└── Makefile
```

## Key Commands

```bash
make dev          # Air hot reload 开发模式
make test         # 单元 + 集成测试 (-v -race -coverprofile)
make e2e          # E2E 测试 (-tags=e2e)
make build        # 构建 binary
swag init -g main.go -o docs/   # 重新生成 Swagger
go run . serve --config configs/config.yaml
go run . validate --rules configs/rules.yaml
```

## Development Rules

### Viper 配置
- key 命名点分小写：`server.port`, `admin.port`, `log.format`, `rules.path`
- 环境变量前缀 `GOWEBHOOK_`：`GOWEBHOOK_SERVER_PORT` → `server.port`
- 优先级：flag > env > yaml > default。所有配置项必须有 default 值
- 搜索路径：`./configs/`, `$HOME/.go-webhook/`, `/etc/go-webhook/`
- config.yaml 热重载用 Viper WatchConfig，rules.yaml 用 fsnotify 单独监听
- 重载原子替换规则集（atomic.Value），加载失败保留旧规则

### Gin
- 业务中间件顺序：Recovery → Logging → Metrics → Auth → Handler
- 管理端口中间件：Recovery → Logging（轻量），不挂 auth
- 统一错误响应：`{"error": "message", "code": "ERROR_CODE"}`
- 用 `ShouldBindJSON()` 解析请求体

### Admin API
- `/admin/config` 必须脱敏：含 `token`/`secret`/`password`/`key` 的字段值替换为 `"******"`
- admin 端口不暴露公网，通过网络策略限制访问

### Swagger
- swaggo 注解写在 handler 函数上，生成到 `docs/`，纳入 git
- CI 中校验 `swag init` 后无 diff
- Swagger UI 挂管理端口：`adminRouter.GET("/swagger/*any", ginSwagger.WrapHandler(...))`
- request/response 结构体定义在 `internal/handler/` 或 `internal/config/types.go`

### Zap 日志
- `log.format`: `json`（默认，zap.NewProduction）或 `text`（zap.NewDevelopment）
- 请求日志含：method, path, status, latency, client_ip, request_id
- engine 日志用 `logger.With(zap.String("rule", ruleName))`

### Prometheus Metrics
- 必须定义：`webhook_requests_total` (counter, labels: status/path), `webhook_request_duration_seconds` (histogram), `webhook_dispatch_total` (counter, labels: rule_name/target/status), `webhook_dispatch_duration_seconds` (histogram), `webhook_rules_loaded` (gauge)
- 用 `promauto` 注册，指标常量集中在 `internal/metrics/`

### Expr 表达式
- 加载时 Compile，运行时只 Run。内置函数：`now()`, `env()`, `lower()`, `upper()`, `join()`, `split()`
- 求值错误不 panic，记日志跳过该规则

### HTTP 转发
- 默认超时 10s（规则可覆盖），指数退避重试 3 次（仅 5xx/网络错误）
- 自定义 headers 支持 `{{env.VAR}}` 模板

### TDD（ECC tdd-workflow）
- 新功能和 bug 修复**必须** RED → GREEN → REFACTOR
- RED：先写测试，`make test` 确认失败，原因是功能未实现而非环境问题。**RED 确认前不得写业务代码**
- GREEN：最小实现，`make test` 确认全绿
- REFACTOR：重构后 `make test` 仍绿
- 每阶段 git checkpoint：`test: add <feature>` → `feat: implement <feature>` → `refactor: clean up <feature>`
- 覆盖率：整体 80%+，`internal/engine/` 90%+
- 单元测试（`*_test.go` 同目录）+ 集成测试（`//go:build integration`）→ `make test`
- E2E 测试（`test/e2e/`，`-tags=e2e`）→ `make e2e`
- **不要直接 `go test`**，Makefile 封装了正确的 flags
- Mock：HTTP 用 `httptest.NewServer`，Expr 用 `test/fixtures/` payload

### CI/CD
- tag `v*` 触发 GoReleaser → 4 binary (darwin/linux × amd64/arm64) + Docker 多架构 + Helm chart → 全部推 ghcr.io
- Dockerfile：`golang:1.22-alpine` 构建 → `distroless/static-debian12` 运行

### Helm Chart
- 源码 `deploy/helm/go-webhook/`，CI 自动替换 Chart.yaml version/appVersion
- values.yaml 默认值与 config.yaml 一致，ConfigMap 挂载两份配置
- 安装：`helm install go-webhook oci://ghcr.io/<owner>/charts/go-webhook --version <ver>`

### Air + VSCode
- `make dev` = `air -c .air.toml`，监听 .go/.yaml，构建带 `-gcflags='all=-N -l'` 方便 debug
- `.vscode/launch.json`：Launch 直接启动 + Attach to Air 进程。含 `GIN_MODE=debug`

### README.md
- README.md 是项目对外的入口文档，必须保持与代码同步
- 内容包括：项目简介、功能特性、快速开始（安装/运行）、配置说明、规则编写示例、部署方式（Docker/Helm）、开发指南（make 命令）、API 端点列表
- 代码变更涉及以下情况时**必须同步更新 README.md**：
  - 新增/删除/变更 CLI 命令或 flag
  - 新增/删除/变更 API 端点
  - 配置项变更（config.yaml / rules.yaml 结构）
  - 新增 expr 内置函数
  - 部署方式变更（Dockerfile / Helm chart / CI）
  - Make target 变更
- README 中的示例代码和命令必须可直接复制运行

## Don'ts

- Expr 逻辑只在 engine 包，handler 不直接操作
- 不绕过 Viper 读 env（不用 `os.Getenv`），不绕过 zap 打日志（不用 `fmt.Println`）
- `/metrics`、`/swagger`、`/admin/*` 只在管理端口，不挂业务端口
- `/admin/config` 不返回明文 secret
- `docs/` 由 swag 生成，勿手动编辑
- Helm chart 不 hardcode image tag，用 `{{ .Values.image.tag | default .Chart.AppVersion }}`
- `.goreleaser.yaml` 不 hardcode 版本号
- `tmp/` 不提交 git
- RED 未确认前不写 `internal/` 下的非测试文件
- 测试只用 `make test` / `make e2e`，不直接 `go test`