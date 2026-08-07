---
name: qq-farm-core-dev
description: Develop APIs in the qq-farm-core Fiber/GORM QQ Farm backend. Covers layered workflow (model/dao/types/service/controller/router), static JWT auth (no Casbin), Make/CLI codegen, Swagger, and verification. Use when working under qq-farm-core/, adding farm/system endpoints, or make run/docs/lint.
---

# qq-farm-core Development

Work from the `qq-farm-core/` repo root. Config: `configs/config.{dev|test|prod}.yaml`.

Defaults: `ENV=dev`, `PORT=9528`. Seed admin: `admin` / `admin888`. DB auto-migrates and seeds on boot; see [database/README.md](../../../database/README.md).

## Auth (single-operator)

- JWT login only; protected chain: `Auth` (+ Cache). **No Casbin / no platform role-menu-permission APIs.**
- Frontend uses `VITE_AUTH_ROUTE_MODE=static` flat menus.
- User table: `cn_sys_admin` only. Legacy RBAC tables are DROP'd on migrate.

## Feature workflow

1. **Model** — `internal/app/model/`; register in `pkg/database/migrate` AutoMigrate list
2. **DAO** — `go run ./cmd/cli genModel` when needed
3. **Types** — DTOs in `internal/types/{domain}/`
4. **Service** — farm under `service/farm/…`, admin under `service/system/admin`, auth under `service/auth/`
5. **Controller** — matching `controller/…`; `Validate` + `pkg/response`
6. **Router** — `internal/router/routes/*.go` with `.Name("中文说明")`
7. **Swagger** — swag comments → `make docs` → `http://127.0.0.1:9528/docs`

## Coding conventions

- Formatter: `gofumpt` via `make lint`
- Controller: bind/validate only; Service: business logic; Model: table mapping

## Verification

```bash
go build ./...
make lint   # if available
make docs   # after API comment changes
bash scripts/api_security_check.sh
```

Do not kill frontend processes; restart backend only on port **9528** when needed.
