# L4D2 Player Stats

[![Go](https://github.com/gofurry/l4d2-plugin-stats/actions/workflows/go.yml/badge.svg)](https://github.com/gofurry/l4d2-plugin-stats/actions/workflows/go.yml)
[![React](https://github.com/gofurry/l4d2-plugin-stats/actions/workflows/frontend.yml/badge.svg)](https://github.com/gofurry/l4d2-plugin-stats/actions/workflows/frontend.yml)
[![License](https://img.shields.io/github/license/gofurry/l4d2-plugin-stats)](LICENSE)

面向《求生之路 2》服务器的玩家统计系统，由 SourceMod 数据采集插件和可选的 Web Dashboard 组成。采集器记录真人玩家在合作、写实和对抗模式中的身份、会话、玩法数据与低频战局事实；Dashboard 提供服务器状态、个人中心、排行榜、战局分析、更新公告和数据维护后台。

> 第一次部署请阅读[中文部署手册](INSTALL.zh-CN.md)；升级已有安装请阅读[升级与回滚](UPGRADE.zh-CN.md)。

## 功能

- 只统计通过 Steam 验证的真人玩家，不为 Bot 创建玩家档案；
- 合作与写实共用 PvE 统计口径，对抗模式使用独立的幸存者、感染者、半场和比赛模型；
- 记录连接时间、实际操作时间、章节与战役成绩、击杀、助攻、伤害、生存、救援、治疗、装备和技巧数据；
- 采集保护队友、挂边、Tank 石块命中、Hunter 空中击杀与 Charger 近战截停等可空高价值幸存者指标；
- 永久保存真人幸存者之间的扶起、解救、治疗和友伤定向事实，个人页可按时间、服务器和模式查看玩家关系；
- 按冻结的 Achievement Contract v1 自动确认生涯成就，支持公开/隐藏/彩蛋可见性、三枚展示徽章和历史自动补判，无需领取或手动刷新；
- 保存每个 Round 的规则环境，并以可验证、低频的 Incident 分析控制、倒地、死亡、救援、警报车、Witch 惊扰、医疗包、目标互动和 Boss 生命周期；
- 支持 SQLite、MySQL 和 PostgreSQL，三种数据库使用一致的结构与统计契约；
- 通过绝对快照、异步保存、有界队列、重试和日志抑制降低数据库故障对游戏线程的影响；
- 提供内嵌 React 的 Go 单二进制 Dashboard，无需单独部署前端；
- 提供专为 L4D2 原生 MOTD WebView 设计的服务端渲染游戏内页面，零 JavaScript 展示当前服务器、在线玩家、公开个人摘要、排行榜、公告和本服文档；
- 支持 Steam OpenID、手动 SteamID64 查询、全服排行榜、多服务器 A2S 状态和单管理员后台；
- 提供地图/战役、规则环境、标准化时间线、Boss 生存时间和玩家效率分析；
- 提供日、月度和终身聚合、数据库增长监控及管理员确认后的分批清理；
- 应用日志自动轮转，公开 API 不返回玩家 IP，管理写操作使用 JWT、CSRF 和请求限流保护。
- 默认审计真人聊天到独立的 `chat-audit.db`，并提供仅管理员可用的聊天检索、导出、连接审计和可选百度 GeoIP 城市级近似位置。

## 界面预览

### 服务器概览

![Dashboard 服务器概览](docs/assets/screenshots/dashboard-server-overview.webp)

### 玩家个人页

![Dashboard 玩家个人页](docs/assets/screenshots/dashboard-player-profile.webp)

### 数据运维

![Dashboard 数据运维](docs/assets/screenshots/dashboard-data-maintenance.webp)

## 运行结构

![L4D2 Player Stats 多服务器部署架构](docs/assets/l4d2-player-stats-architecture.svg)

每个逻辑服务器组使用一个稳定的 `server_key`；同一台机器或同一社区组下的多个 IP:PORT 实例可以有意共用该值，Dashboard 会把它们聚合成一个入口与排行榜范围。共享 Stats 数据库中的不同逻辑组必须使用不同的 `server_key`。每个 L4D2 实例仍各自运行一份采集器；采集插件可以独立运行，只有需要网页展示和数据维护时才部署 Dashboard。

## 支持范围

| 项目 | 支持内容 |
|---|---|
| 游戏模式 | `coop`、`realism`、`versus` |
| Stats 数据库 | SQLite、MySQL、PostgreSQL |
| Dashboard 系统 | Windows amd64、Linux amd64 |
| 玩家身份 | SteamID64 真人玩家 |
| 服务器形态 | 独立服务器、用于开发测试的本地服务器 |

其他游戏模式不会创建 gameplay 身份、会话或统计记录；启用默认的 Chat Audit 后，真人 `say`/`say_team` 仍会进入独立审计链路。PvE 与对抗数据严格分开，详细统计语义见[统计口径](contracts/statistics.md)和[对抗契约](contracts/versus-v1.md)。

## Release 包结构

官方构建产物同时包含采集器和 Dashboard：

```text
l4d2-plugin-stats-*.zip
├─ left4dead2/                         直接合并到游戏服务端目录
│  └─ addons/sourcemod/
│     ├─ plugins/l4d2_player_stats.smx
│     └─ configs/l4d2_player_stats/migrations/
├─ dashboard/
│  ├─ windows-amd64/
│  │  ├─ l4d2-stats.exe
│  │  └─ config.example.yaml
│  └─ linux-amd64/
│     ├─ l4d2-stats
│     └─ config.example.yaml
├─ examples/
│  ├─ databases.cfg.example
│  ├─ dashboard-sqlite.yaml
│  ├─ dashboard-mysql.yaml
│  ├─ dashboard-postgresql.yaml
│  └─ nginx.conf.example
├─ INSTALL.zh-CN.md
├─ UPGRADE.zh-CN.md
├─ CHANGELOG.md
├─ README.md
└─ LICENSE
```

Dashboard 前端已嵌入可执行文件，不需要单独部署 Node.js。发布包不包含开发期的 `docs/` 和 `contracts/` 目录；部署所需内容集中在两份离线手册和 `examples/`。Linux 解压后如缺少执行权限，请运行 `chmod +x dashboard/linux-amd64/l4d2-stats`。

## 快速开始

### 1. 安装采集器

服务器需要先安装 [Metamod:Source](https://www.sourcemm.net/) 和 [SourceMod](https://www.sourcemod.net/)。

将 release 压缩包内的 `left4dead2` 文件夹覆盖到服务器同名目录。它会安装：

```text
left4dead2/addons/sourcemod/plugins/l4d2_player_stats.smx
left4dead2/addons/sourcemod/configs/l4d2_player_stats/migrations/
```

将 `examples/databases.cfg.example` 中选定的数据库块合并到：

```text
left4dead2/addons/sourcemod/configs/databases.cfg
```

插件首次加载会生成：

```text
left4dead2/cfg/sourcemod/l4d2_player_stats.cfg
```

至少设置一个稳定的逻辑服务器组标识；同组实例填写相同值，不同组在共享数据库中填写不同值：

```text
sm_lps_server_key "my-l4d2-server-01"
```

重新加载并检查状态：

```text
sm plugins reload l4d2_player_stats
sm_lps_status
```

正常状态应显示数据库驱动、`schema=7/7` 和 `state=ready`。完整步骤见[中文部署手册](INSTALL.zh-CN.md)。

### 2. 启动 Dashboard

从压缩包的 `dashboard` 目录选择对应平台的二进制，并将 `config.example.yaml` 复制为 `config.yaml`：

```yaml
server:
  listen: "0.0.0.0:18848"

dashboard_database:
  path: "./dashboard.db"

chat_audit:
  database_path: "./chat-audit.db"

stats_database:
  driver: "sqlite"
  dsn: "/absolute/path/l4d2_player_stats.sq3"

logging:
  file: "./logs/l4d2-stats.log"

monitor:
  enabled: true
```

启动服务：

```sh
./l4d2-stats serve --config ./config.yaml
```

首次启动会在终端输出一次性设置令牌。打开 `/admin/setup` 创建管理员，然后在后台配置界面、服务器、Steam 登录、公告和数据维护策略。

Linux 可直接注册为 systemd 服务：

```sh
sudo ./l4d2-stats install --config ./config.yaml
```

生产部署、HTTPS、权限和首次设置说明见[中文部署手册](INSTALL.zh-CN.md)，备份与回滚说明见[升级与回滚](UPGRADE.zh-CN.md)。

### 3. 启用游戏内 MOTD 页面

先确认采集器已设置所属逻辑组的 `sm_lps_server_key`，并让 Dashboard 在“服务器管理”中至少成功读取一次各实例的 A2S 规则。然后进入后台“游戏内页面”：

1. 启用游戏内页面，设置默认标题、Banner、全页背景、首页模块、三个生涯亮点指标和缓存预设；
2. 在同页选择服务器组，可按组继承、覆盖或隐藏背景、Banner、描述和完整网站链接，并覆盖亮点指标和三份 Markdown 文档；
3. 在“MOTD 部署帮助”选择服务器组，可先预览目标页面，再把同一份 `motd.txt` 内容复制到组内各游戏实例。

生成内容使用原生 HTML 跳转，目标形如：

```html
<html>
<head>
<meta http-equiv="refresh"
content="0;url=https://stats.example.com/ingame?server=community-coop-01">
</head>
<body>
Loading...
</body>
</html>
```

`server` 参数来自已持久化的 A2S `sm_lps_server_key` 规则，代表整个服务器组而非单个端口；玩家打开页面时不会触发即时 A2S 查询。游戏内页面使用独立的 10–1800 秒安全缓存预设，和“站点设置 → 服务”中的 A2S 刷新周期不是同一个配置。Background、Banner 与完整网站链接只接受不含账号密码的绝对 HTTP/HTTPS 地址，Dashboard 不下载、代理或探测这些外部资源；换图时应使用新 URL 或 `?v=2` 一类查询版本。游戏内 Markdown 外链、“完整网站”、快速链接和加入游戏入口只打开零 JavaScript 的操作提示卡，显示需要在普通浏览器访问的 URL 或控制台 `connect` 命令，不声称 MOTD 能可靠跳出外部浏览器。游戏内个人资料始终按匿名访客的公开可见性渲染，不会因 Steam 或管理员 Cookie 获得额外权限。内嵌 CSS 与成就 Atlas 使用内容指纹 URL 和 immutable cache，资源内容变化后会自动换址，避免 L4D2 WebView 长期命中旧样式。

## 管理命令

| 命令 | 权限 | 用途 |
|---|---|---|
| `sm_lps_status` | SourceMod root | 查看连接、驱动、迁移和最近错误 |
| `sm_lps_reconnect` | SourceMod root | 立即重新连接并检查数据库 |
| `sm_lps_flush` | SourceMod root | 立即保存当前快照 |

Dashboard CLI 提供：

```text
l4d2-stats serve
l4d2-stats doctor
l4d2-stats doctor --deep
l4d2-stats aggregate status
l4d2-stats aggregate rebuild
l4d2-stats backup create
l4d2-stats backup restore <file>
l4d2-stats diagnostics export
l4d2-stats migrate status
l4d2-stats install
l4d2-stats uninstall
```

所有命令都可通过 `--config` 指定配置文件。

## 数据与隐私

- Stats DB 是游戏采集事实来源，Dashboard DB 保存网页配置、管理员和可重建的聚合数据；
- Chat Audit 最终历史保存在独立 `chat-audit.db`：采集默认开启，Stats outbox 只保留 72 小时传输缓冲，最终默认保留 30 天；
- 采集器保存服务器观察到的玩家 IP，用于会话审计，但公开 API 和网页不会查询或展示该字段；
- GeoIP 在未配置百度 AK 时不会请求 provider；配置后仅后台异步解析公网 IP，请求速率可在后台设置为 1-3 QPS（默认 2），管理测试与后台队列共用同一限速器；Dashboard 缓存只保存 HMAC 键和城市级近似结果，不复制原始 IP。位置筛选会有界扫描底层连接游标，缓存未命中时提示仍在解析并要求稍后刷新，不会同步请求百度；
- 常规 Dashboard 查询只读 Stats DB；执行原始数据清理时才使用具备 `DELETE` 权限的维护连接；
- 过期装备/职业明细、已关闭 Session 和比赛结果只有在聚合覆盖校验通过并由管理员确认后才会分批删除；Incident 使用独立的默认 180 天保留策略；
- 数据库密码只应保存在服务器本地配置中，不应提交到仓库。
- 常规备份不包含 `chat-audit.db`；SQLite Stats 备份副本会移除瞬时聊天 outbox 并清空 Session IP，Dashboard 备份副本会清空百度 AK、GeoIP cache secret 及其缓存，所有清理都不修改 live DB，恢复后自动生成新的 cache secret。MySQL/PostgreSQL Stats 原生备份可能包含 Session IP/outbox，管理员必须按自身隐私策略排除或单独到期；诊断包不包含聊天正文、原始 IP 列表或 GeoIP AK。

详细规则见[数据生命周期](docs/dashboard-data-lifecycle.md)。

## 从源码构建

采集器以生产环境稳定版 SourceMod 1.12 为兼容目标，需要 SourcePawn Compiler `1.12.0.7246`、对应版本的 include、Python 3 和本机路径配置。编译工具目录应与游戏服务器的 SourceMod 运行目录分开配置，构建脚本会拒绝 1.13 或其他版本的编译器：

```powershell
Copy-Item scripts/config.example.ps1 scripts/config.local.ps1
./scripts/build.ps1
```

Dashboard 构建需要 Go、Node.js 和 pnpm：

```powershell
./scripts/build-dashboard.ps1
```

Linux 可以使用：

```sh
./scripts/build-dashboard.sh
```

项目构建产物统一写入 `dist/`，不会被 Git 跟踪。

## 仓库结构

```text
collector/     SourceMod 采集器源码
database/      SQLite、MySQL、PostgreSQL 迁移与结构说明
contracts/     模式、统计口径和对抗数据契约
dashboard/     Go API、Dashboard DB、内嵌 React 前端
docs/          部署、数据生命周期和维护文档
scripts/       构建、校验、部署和打包脚本
```

## 参与贡献

欢迎提交 Issue 或 Pull Request。涉及数据库字段、统计含义或玩法边界的修改，请先阅读现有契约，并同步更新三种数据库迁移、采集器和 Dashboard 查询。实现已有 Issue 的功能 PR 请在描述中使用 `Fixes #<issue-number>`，让功能合并时同步关闭对应 Issue。

## License

[MIT](LICENSE)
