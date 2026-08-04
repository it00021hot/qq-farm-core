# Database

表结构与初始化数据由 GORM 在启动时自动完成，无需手工导入 SQL。

- 建库：连接时若不存在会自动创建 `go_skeleton`
- 建表：`pkg/database/migrate` → `AutoMigrate`
- 种子数据：幂等写入超级管理员 / 角色 / casbin 规则

配置项：`database.pgsql.autoMigrate`（默认开启，设为 `false` 可跳过）

默认管理员账号：`admin` / `admin888`
