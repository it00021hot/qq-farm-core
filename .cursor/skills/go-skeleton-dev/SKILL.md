---
name: go-skeleton-dev
description: Develop APIs and features in the go-skeleton Fiber/GORM backend scaffold. Covers layered workflow (model/dao/types/service/controller/router), coding conventions, Make/CLI codegen, multi-tenant rules, Swagger, and verification (lint/build/docs/security script). Use when working under go-skeleton/, adding backend endpoints, genController/genService/genModel, make run/docs/lint, or tenant/Casbin/RBAC changes.
---

# go-skeleton Development

Work from the `go-skeleton/` repo root. Go `>=1.26`. Config: `configs/config.{dev|test|prod}.yaml`.

Defaults: `ENV=dev`, `PORT=9528`. Seed admin: `admin` / `admin888`. DB auto-creates schema and seeds on boot (`database.pgsql.autoMigrate`); see [database/README.md](../../../database/README.md).

## Feature workflow

Do not skip layers. Order:

1. **Model** — `internal/app/model/`; register in `pkg/database/migrate` AutoMigrate list
2. **DAO** — `go run ./cmd/cli genModel [-m=table] [-e=dev] [-a=default]`; optional Querier in `internal/app/entity/` + `entity.go` methodMaps
3. **Types** — DTOs in `internal/types/{domain}/`
4. **Service** — `go run ./cmd/cli genService -n=Name [-d=dir]`；按域落盘：`service/platform/`（租户/角色/菜单/权限）、`service/system/`（用户/附件）、`service/auth/`
5. **Controller** — 与 service 同域：`controller/platform|system|auth`（`ping` 仍可在 `controller/backend`）；`Validate` + `pkg/response`
6. **Router** — register in `internal/router/routes/backend.go` (or frontend/common) with `.Name("中文说明")`
7. **RBAC resource** — if API needs auth: menu/button in `cn_sys_resource` (`alias` / `b_url` / `methods`), then `POST /backend/role/auth` (syncs Casbin; no manual Casbin CRUD)
8. **Swagger** — swag comments → `make docs` → `http://127.0.0.1:9528/docs`
9. **Tenant** — business tables have `tenant_id`; middleware `Auth → Tenant → Cache → Casbin`; platform business calls need header `X-Tenant-ID` (platform config paths exempt — see database README)

## Coding conventions

### Format

- Formatter: `gofumpt` via `make lint` (`gofumpt -l -w .`). Run before finish.
- No extra linter configs. Optional deep checks (shadow/nilaway) are in [README.md](../../../README.md), not daily required.

### Layer duties (no leakage)

| Layer | Does | Does not |
|-------|------|----------|
| Controller | bind/validate, call service, HTTP response | business rules, direct DB |
| Service | business logic, tenant scope, quota/role checks | raw HTTP status crafting |
| Types | Req/Resp + `validate` tags | business methods |
| Model | table mapping, `TableName*`, tenant getters/setters | API DTOs |
| Router | register routes + `.Name(...)` | handlers |

### Naming and layout

- Paths by domain: `controller/{platform|system|auth}/{domain}`, `service/{platform|system|auth}/{domain}`, `types/{domain}`
  - **platform**：租户/角色/菜单/权限（不依赖 `X-Tenant-ID`）
  - **system**：用户/附件（租户上下文）
  - **auth**：登录鉴权与动态路由
- Singletons: `var Admin = &Controller{}` / `var Admin = &Service{}`
- Embed: `controller.Controller` / `service.Service`
- Tables: `cn_` prefix (`cn_sys_admin`); Go types drop prefix (`SysAdmin`)
- JSON/DB fields: `snake_case` (`nick_name`, `tenant_id`)
- Import aliases: `adminsvc`, `admintypes` when packages collide

### Controller pattern

Fiber v3 uses `ctx fiber.Ctx` (not `*fiber.Ctx`).

```go
var req admintypes.CreateReq
if err := c.Validate(ctx, &req); err != nil {
	return response.BadRequestException(ctx, err.Error())
}
info, err := adminsvc.Admin.Create(ctx, req)
if err != nil {
	return response.BadRequestException(ctx, err.Error())
}
return response.SuccessJSON(ctx, "", info)
```

Public APIs need full swag (`@Summary` `@Tags` `@Security ApiKeyAuth` `@Router`; document `X-Tenant-ID` when required).

### Types / validation

- Tags: `json` + `query` when needed; `validate:"required,..."` (Chinese messages via zh validator)
- Pagination: `current` / `size`
- Status enum: usually `oneof=1 2` (1 normal, 2 disabled)

### Service / data access

- Row-level tenant: `tenant.Scope(vars.DB, tenant.TenantCtx(ctx))`
- Cross-tenant / platform global: `tenant.Global(...)`
- Business errors: `errors.New("中文说明")` → controller maps to `BadRequestException`
- List shape: `map[string]any{"list", "total", "page"}` + `pagination.New().ParsePage`
- Clear secrets (`password`, `salt`) before return
- Timestamps are usually Unix `uint`; do not switch to `time.Time` unless the whole chain matches

### Model / migrate

- gorm tags include `column` and `comment:` (PG column comments)
- New tables: update both `internal/app/model` and `pkg/database/migrate`
- Multi-tenant business tables: `tenant_id` + index; implement `GetTenantID` / `SetTenantID` when isolated

### Response and logging

- Only `pkg/response`: `SuccessJSON`, `BadRequestException`, `UnauthorizedException`, `ForbiddenException`
- Success `errcode=0`; never bare `ctx.JSON` for API payloads
- Logging: `log/slog` (`Info` / `Error` / `Warn` / `Debug`)

### Auth style constraints

- New protected routes: resource (`b_url` + `methods`) → role auth sync; no hand-written Casbin CRUD
- Casbin `p` is exactly three fields: `sub,obj,act` (never write markers into `v3`)
- Platform config paths (`/platform`, `/auth`) skip `X-Tenant-ID`; `/system` business data paths require it for platform users

### HTTP route convention（强制）

- 前缀与 Go 包对齐：`/platform`（租户/角色/菜单/权限）、`/system`（用户/附件）、`/auth`（登录鉴权）
- 路径：`/<prefix>/<resource>/<action>`，例：`GET /platform/tenant/list`、`POST /platform/tenant/add`
- action：`list` / `add` / `modify` / `delete`；批量 `batch-delete` 等；只读派生 `detail` / `tree` / `assignable` / `status`
- **仅 GET、POST**；禁止 PUT / DELETE / PATCH
- `b_url` **禁止** `/*` 通配，每个按钮一条精确 path + methods（仅 GET/POST）
- 按钮 `alias` = 前端 `hasAuth` 码；菜单 `hide_in_menu`：`1` 显示 / `2` 隐藏（与空 `fUrl` 不同：空 `fUrl` 表示无页面挂载点）
- 路由注册：`routes/auth.go`、`routes/platform.go`、`routes/system.go`；login/refresh 在 `common.go` 的 `/auth/login|refresh`
## Commands

| Task | Command |
|------|---------|
| Deps | `go mod tidy` |
| Run (Agent/LAN) | `PORT=9528 ENV=dev make run` |
| Run direct | `go run ./cmd/app -e=dev -p=9528` |
| CLI help | `go run ./cmd/cli -h` |
| Codegen | `genController` / `genService` / `genMiddleware` / `genCommand` / `genModel` |
| Format | `make lint` |
| Swagger | `make docs` |
| Cross-build | `make windows` / `linux` / `darwin` |
| Hot reload | `air` (adjust `.air.toml` per env) |

Prefer `make run` in Agent terminals (incremental build + ad-hoc codesign so LAN works).

Codegen examples:

```bash
go run ./cmd/cli genController -n=foo -d=backend/foo
go run ./cmd/cli genService -n=foo -d=backend/foo
go run ./cmd/cli genMiddleware -n=foo
go run ./cmd/cli genCommand -n=foo -d=foo -s=mysql,redis
go run ./cmd/cli genModel -m=cn_foo -e=dev -a=default
```

## Verification checklist

After changes, in order:

1. `make lint`
2. `go build -o /dev/null ./cmd/app ./cmd/cli` (plus targeted `go test` when touching tested packages)
3. If API/swag changed: `make docs` and confirm new paths in swagger
4. `PORT=9528 ENV=dev make run`, then smoke (login / new endpoint)
5. Tenant/RBAC changes: `bash scripts/api_security_check.sh` (needs `curl`, `jq`; optional `BASE_URL=...`)
6. Manual checks in [database/README.md](../../../database/README.md)「验收要点 / 安全验收」

Codegen alone is incomplete: wire routes, add swag, run lint/docs.

## Hard rules

- Casbin `p`: three fields only; no `v3` markers
- Role auth replaces policies in full; do not dual-write Casbin
- Unified `pkg/response` + `slog`
- Follow middleware order and tenant header rules above

## References

- Project overview: [README.md](../../../README.md)
- Multi-tenant & RBAC: [database/README.md](../../../database/README.md)
- Security script: [scripts/api_security_check.sh](../../../scripts/api_security_check.sh)
