# Database

表结构与初始化数据由 GORM 在启动时自动完成。

- 建表：`pkg/database/migrate` → `AutoMigrate`
- 种子：仅写入 `admin` 账号（菜单由前端静态路由提供）
- 启动时 DROP 旧平台 RBAC 表：`cn_sys_role` / `cn_sys_resource` / `cn_sys_role_auth` / `cn_sys_casbin_rule`

默认管理员：`admin` / `admin888`

## 单机约定

- **鉴权**：JWT 登录即可；登录后全部农场 API 可用（无 Casbin / 无菜单权限矩阵）
- **菜单**：前端 `VITE_AUTH_ROUTE_MODE=static` 扁平侧栏
- 中间件：`Auth`（+ Cache）
- 用户表：仅 `cn_sys_admin`

### 从旧版升级

旧库启动时会自动 DROP 上述 RBAC 表。若仍异常，可停服后删除 `runtime/data/qq-farm.db*` 再启动。

### 冒烟

```bash
go run ./cmd/app -e=dev -p=9528
bash scripts/api_security_check.sh
```
