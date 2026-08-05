# Database

表结构与初始化数据由 GORM 在启动时自动完成，无需手工导入 SQL。

- 建库：连接时若不存在会自动创建 `go_skeleton`
- 建表：`pkg/database/migrate` → `AutoMigrate`（结构体 `comment:` 标签写入列注释）
- 种子数据：幂等写入超级管理员 / 角色树 / 菜单资源 / 角色授权 / casbin 规则

配置项：`database.pgsql.autoMigrate`（默认开启，设为 `false` 可跳过）

默认管理员账号：`admin` / `admin888`

## 多租户约定

### 模型

| 概念 | 说明 |
|------|------|
| `tenant_id = 0` | 平台账号 |
| `tenant_id > 0` | 租户账号（归属该租户） |
| 全局角色树 | 仅平台维护；`role_type=1` 平台专用，`=2` 可赋给租户用户 |
| `cn_sys_admin_tenant` | 平台用户可管理的多租户绑定；超管不写此表表示全部 |

### 中间件顺序

`Auth → Tenant → Cache → Casbin`

- 租户用户：JWT 固定 `tenant_id`，校验租户 status / expire_at
- 平台用户操作业务数据：请求头 `X-Tenant-ID`（须在授权集合内；超管任意）
- 平台配置接口（`/backend/tenant|role|menu|permission|platform-user`）不要求 `X-Tenant-ID`
- `/backend/auth/info`、`/backend/auth/routes`、`/backend/auth/logout` 同属平台配置路径，不要求 `X-Tenant-ID`

### 权限模型（菜单 / 按钮 / API）

```
cn_sys_resource (目录/菜单/按钮，alias + f_url + b_url + methods)
       ↑ resource_ids
cn_sys_role_auth  →  前端 menus/buttons 下发
       ↓ SyncRoleCasbin（v3=resource_sync）
cn_sys_casbin_rule →  Casbin 中间件 API 鉴权
```

| 字段 | 用途 |
|------|------|
| `resource_type` | 1 目录 / 2 菜单 / 3 操作按钮 |
| `alias` | 前端按钮权限码（`useAuth().hasAuth(alias)`） |
| `f_url` | 前端路由占位 |
| `b_url` + `methods` | 后端路径模式 + HTTP 方法；角色授权后同步为 Casbin `p` |

管理入口：

1. `POST/PUT /backend/menu` 维护资源（含按钮 `alias` / `b_url` / `methods`）
2. `POST /backend/role/auth` 授权资源 ID；服务端写 `role_auth` 并 `SyncRoleCasbin` + Reload
3. `GET /backend/role/auth?role_id=` 回显授权
4. `GET /backend/auth/info` 返回当前用户 `menus` + `buttons`（超管 `buttons=["*"]`，结果缓存 Redis `perms:{uuid}`）
5. `GET /backend/permission/apis` 列出可绑定路由；`GET /backend/permission/role/:id` 只读 Casbin；`POST /backend/permission/reload` 强制重载

Casbin 仍是运行时唯一 API 闸门；不要单独做 Casbin CRUD 双写。

`SetAuth` / 种子同步会对角色执行**全量替换** Casbin `p` 策略（`sub,obj,act` 三字段）。  
注意：不可把标记写入 `v3`——Casbin model 定义为 3 字段，多字段会导致 `LoadPolicy` panic。

### 租户侧能力

- 创建/更新用户、启停
- `GET /backend/role/assignable`：仅返回自身角色子树内、且 `role_type=租户` 的角色
- 分配角色时服务端再次校验「不高于自身」

### 平台侧能力

- 租户 CRUD、配额 `max_users`、过期 `expire_at`、启停、用量
- 开户可顺带创建主账号（默认角色 `role_tenant_admin` id=3）
- 角色树 / 菜单 / 角色菜单授权 / 创建平台用户并绑定租户
- 权限运维：API 列表、角色策略只读、Casbin reload

### 验收要点

1. 租户 A 无法读写租户 B 数据（即使篡改 body/`X-Tenant-ID`）
2. 租户用户调用 `/backend/tenant`、`/backend/role`（除 assignable）、`/backend/menu` 被拒
3. 员工不能把「租户管理员」等上级角色赋给他人
4. 过期/禁用租户无法登录；建用户受 `max_users` 限制
5. 平台超管（`server.superRoleId`）跳过 Casbin，可操作全部租户
6. `GET /backend/auth/info`：超管 `buttons=["*"]`；租户角色返回已授权 alias
7. `POST /backend/role/auth` 调整带 `b_url` 的按钮后，目标角色 API 访问立即变化

### 安全验收

```bash
# 先启动服务：go run ./cmd/app -e=dev -p=9528
bash scripts/api_security_check.sh
# 可选：BASE_URL=http://127.0.0.1:9528 bash scripts/api_security_check.sh
```
