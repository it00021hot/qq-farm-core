# QQ 农场 · 后端（qq-farm-core）

QQ 农场智能助手后端：多账号托管、自动化种地/好友互动、活动与商城，以及管理端 API。

基于 [qq-farm-core](https://github.com/it00021hot/qq-farm-core)（Go ≥ 1.26 + Fiber）改造；业务核心在 `internal/farm`。

配套前端：[`../vue-framework`](../vue-framework)。  
桌面端（Wails）：[`../desktop`](../desktop)（通过 [`pkg/appserver`](pkg/appserver) 进程内启动本服务）。

## 功能概览

| 模块 | 说明 |
|------|------|
| 账号 | 多农场账号 CRUD、启停；微信扫码登录 |
| 运行时 | 巡田、种植、施肥、偷菜、帮忙、捣乱；可配置间隔与静默时段 |
| 个人面板 | 土地 / 背包（卖、用）/ 每日礼包与任务 |
| 好友 | 同步、土地查看、互动记录与访客 |
| 活动中心 | 千星游记、观星、星砂商店、节令 |
| 商城 | 游戏商城、神秘商人、钻石余额 |
| 配置 | 自动化开关、种植策略、游戏静态配置（种子/果实/道具） |
| 实时 | WebSocket 推送运行状态；可选异常 Webhook |
| 权限 | JWT 登录即可（单机自用，无 Casbin 菜单矩阵） |

默认开发库为 **SQLite**（`runtime/data/qq-farm.db`），Redis 默认关闭，可单机启动。

## 目录结构

```
cmd/
  app/                 # HTTP 服务入口
  cli/                 # 代码生成等 CLI
configs/               # config.{dev,test,prod}.yaml
database/              # 库表与权限约定说明
internal/
  app/                 # controller / service / model
  farm/                # 农场核心
    game/              # 游戏网关 RPC
    runtime/           # 账号会话与自动化循环
    logic/             # 土地/背包/配置等领域逻辑
    activitycenter/    # 活动快照
    proto/             # 游戏协议
    tsdk/              # WASM / TSDK
  middleware/          # Auth 等
  router/routes/       # auth / farm / system
pkg/                   # 配置、DB 迁移、响应等
resource/farm/         # gameConfig、tsdk.wasm
runtime/               # 日志、SQLite、tsdk 工作目录
scripts/               # proto 生成、安全检查等
```

## 快速开始

```shell
go mod tidy

# 推荐：增量编译 + 签名后启动（默认端口 9528）
make run

# 或直接
go run ./cmd/app -e=dev -p=9528
```

- 配置：`configs/config.dev.yaml`（网关、`farm.*`、JWT 密钥等按需修改）
- 默认管理员：`admin` / `admin888`（启动时 Seed）
- Swagger：`http://127.0.0.1:9528/docs`（`swagger.enabled`）
- 权限约定（登录即全权限 + 前端静态菜单）：见 [`database/README.md`](database/README.md)

前端默认代理到 `http://127.0.0.1:9528`（见 vue-framework `.env.test`）。

### 常用 Makefile

```shell
make help      # 查看目标
make run       # 本地启动（端口可用 PORT=9528）
make lint      # gofumpt
make proto     # 重新生成农场 protobuf
make docs      # swagger
make linux     # 交叉编译
```

热更新可用 [air](https://github.com/air-verse/air)：`air`（注意 `.air.toml` 按环境调整）。

### CLI（可选）

```shell
go run ./cmd/cli -h
go run ./cmd/cli genModel -h
go run ./cmd/cli genController -n=foo -d=backend
```

## 配置要点（`farm`）

```yaml
farm:
  gatewayUrl: 'wss://...'          # 游戏网关
  wasmPath: 'resource/farm/tsdk.wasm'
  gameConfigDir: 'resource/farm/gameConfig'
  clientVersion: '...'
  pushWebhook: ''                  # 可选：Bark / 企微机器人等
```

数据库：`database.sqlite.enabled: true` 为默认；`pgsql` 段保留便于回退。

## API 入口

农场业务挂在 `/farm/*`（需登录），例如：

- `GET /farm/status/detail`、`GET /farm/ws`
- `GET /farm/lands`、`POST /farm/operate`
- `GET /farm/bag`、`POST /farm/bag/sell|use`、`GET /farm/daily-gifts`
- `GET /farm/activity/snapshot`、活动领取/兑换
- `GET|POST /farm/automation/*`、`/farm/account/*`、`/farm/friend/*`

系统：`/system/admin`（用户）、`/auth/*`（登录改密）。

## 安全自检

```shell
go run ./cmd/app -e=dev -p=9528
bash scripts/api_security_check.sh
```

## License

MIT（上游 qq-farm-core LICENSE 仍适用）。
