# L4D2 Player Stats

一个面向《求生之路 2》服务器的玩家身份、会话和玩法统计系统。

项目采用 monorepo。当前阶段只定义 SourceMod 采集插件的数据契约；插件稳定后再开发 Go 后端及其内嵌前端。

## 当前状态

项目处于契约设计阶段，尚未开始编写插件实现。

已经确认的基础边界：

- 数据库支持 SQLite、MySQL 和 PostgreSQL（SourceMod 驱动名为 `pgsql`）。
- 只采集 `coop`、`realism` 和 `versus`；其他模式完全不记录。
- `coop` 与 `realism` 属于 PvE 统计族，`versus` 使用独立的 PvP 统计模型。
- 只为通过 Steam 认证的真人玩家建立身份和统计，不为 Bot 建立玩家记录。
- 同时保存连接时间和有效参赛时间。
- 保存服务器观察到的玩家 IP 地址，不含端口；IP 不得由公开网页默认展示。
- 身份、会话和统计明细默认永久保留。
- 常规保存周期为 300 秒加可配置随机抖动，并在关键生命周期事件发生时保存。
- 第一阶段数据库故障只使用有界内存状态恢复，不创建无限增长的本地日志或队列文件。

## 契约

- [模式与生命周期](contracts/modes.md)
- [统计口径](contracts/statistics.md)
- [数据库结构](database/schema.md)

## 计划中的 monorepo 边界

```text
collector/     SourceMod 数据采集插件
database/      三种数据库的结构和迁移
contracts/     插件与未来 Go 服务共同遵守的行为定义
dashboard/     未来的 Go 后端与内嵌前端
deploy/        SourceMod 安装包布局
docs/          架构、部署和测试文档
scripts/       仓库级构建与发布脚本
```

实现阶段会逐步创建这些目录，不提前提交没有内容的占位目录。

## License

[MIT](LICENSE)
