# L4D2 Player Stats 部署手册

本文面向第一次部署 L4D2 Player Stats 的服务器管理员。发布包同时包含 SourceMod 采集器和可选的 Web Dashboard；只需要游戏内统计时，可以只安装采集器。

## 1. 系统组成

```text
L4D2 专用服务器
  └─ SourceMod 采集器
       └─ 写入 Stats DB（SQLite / MySQL / PostgreSQL）
              └─ Go Dashboard 读取并聚合
                    └─ dashboard.db 保存站点设置、管理员、公告和聚合结果
```

- `Stats DB` 是游戏统计的事实来源，由采集器写入。
- `dashboard.db` 是 Dashboard 自己的 SQLite 数据库，不替代 Stats DB。
- Dashboard 前端已经嵌入二进制，不需要安装 Node.js、Go 或单独部署静态文件。

## 2. 选择数据库

| 部署方式 | 推荐数据库 | 说明 |
|---|---|---|
| 单台游戏服务器，Dashboard 与游戏同机 | SQLite | 配置最少，适合绝大多数小型服务器 |
| 多台游戏服务器共享统计 | MySQL 或 PostgreSQL | 每台采集器必须使用不同的 `server_key` |
| Dashboard 与游戏服务器分开部署 | MySQL 或 PostgreSQL | 不要通过网络共享目录访问 SQLite |

SQLite 文件只适合由同一台机器上的进程访问。不要把 `.sq3` 放在 SMB/NFS 等网络共享目录中。

## 3. 部署前准备

采集器需要：

- Left 4 Dead 2 Dedicated Server；
- Metamod:Source；
- SourceMod；
- SourceMod 对应的 SQLite、MySQL 或 PostgreSQL 数据库驱动。

Dashboard 支持发布包中提供的 Windows amd64 和 Linux amd64。A2S 状态查询还要求 Dashboard 所在机器能够通过 UDP 访问游戏服务器端口。

安装前建议备份：

```text
left4dead2/addons/sourcemod/configs/databases.cfg
left4dead2/cfg/sourcemod/l4d2_player_stats.cfg（如果已存在）
Stats 数据库
dashboard.db（升级已有 Dashboard 时）
```

## 4. 安装 SourceMod 采集器

### 4.1 复制完整运行文件

将发布包中的 `left4dead2` 文件夹合并到游戏服务端的同名目录。

最终至少应存在：

```text
left4dead2/addons/sourcemod/plugins/l4d2_player_stats.smx
left4dead2/addons/sourcemod/configs/l4d2_player_stats/migrations/sqlite/0001_initial.sql
left4dead2/addons/sourcemod/configs/l4d2_player_stats/migrations/mysql/0001_initial.sql
left4dead2/addons/sourcemod/configs/l4d2_player_stats/migrations/pgsql/0001_initial.sql
```

迁移文件是首次初始化数据库所必需的，不要只复制 `.smx`。

### 4.2 配置 SourceMod 数据库

打开发布包中的 `examples/databases.cfg.example`，选择一种数据库配置，将名为 `l4d2_player_stats` 的配置块合并进：

```text
left4dead2/addons/sourcemod/configs/databases.cfg
```

不要使用示例文件覆盖服务器原有的整个 `databases.cfg`，否则可能破坏其他插件的数据库配置。

SQLite 示例：

```text
"l4d2_player_stats"
{
    "driver"    "sqlite"
    "database"  "l4d2_player_stats"
}
```

数据库文件会创建在：

```text
left4dead2/addons/sourcemod/data/sqlite/l4d2_player_stats.sq3
```

MySQL 或 PostgreSQL 请修改示例中的主机、端口、数据库名、用户名和密码。首次迁移需要建表及创建索引权限；日常采集需要读写统计表的权限。

### 4.3 设置唯一服务器键

加载插件一次后会生成：

```text
left4dead2/cfg/sourcemod/l4d2_player_stats.cfg
```

编辑其中的：

```text
sm_lps_server_key "community-coop-01"
```

v1.3 默认启用低频 Incident 分析，并允许为规则环境填写可选标签：

```text
sm_lps_incidents_enabled "1"
sm_lps_incident_queue_limit "2048"
sm_lps_ruleset_name "vanilla-coop"
```

`sm_lps_incidents_enabled` 在 Round 开始时锁存，中途修改只影响下一 Round；分析队列溢出只会把该 Round 标记为分析不完整，不会阻断核心累计统计。`sm_lps_ruleset_name` 最多 64 个字符，留空也会采集其余可用规则参数。

规则：

- 长度为 1–64 个字符；
- 在同一个 Stats DB 内永久唯一；
- 发布后不要随意修改，否则同一台服务器会被统计成两个来源；
- 多台游戏服务器共享数据库时，每台服务器必须使用不同值。

### 4.4 加载与验收

在服务器控制台执行：

```text
sm plugins reload l4d2_player_stats
sm_lps_status
sm_lps_flush
```

正常结果应包含：

```text
version=1.3.4
state=ready
schema=6/6
```

进入受支持模式游玩一段时间，再执行 `sm_lps_flush`。确认数据库中出现 `lps_players`、`lps_sessions` 和相关统计表记录。

插件只记录 `coop`、`realism` 和 `versus`，其他模式不会创建统计数据。

## 5. 部署 Dashboard

### 5.1 准备目录

从发布包选择对应平台目录，将其复制到独立运行目录。例如：

```text
/opt/l4d2-stats/
├─ l4d2-stats
├─ config.yaml
├─ dashboard.db       启动后生成
└─ logs/              启动后生成
```

Windows 可使用：

```text
D:\Services\l4d2-stats\
├─ l4d2-stats.exe
└─ config.yaml
```

把同目录的 `config.example.yaml` 复制为 `config.yaml`，或从 `examples/` 选择对应数据库的完整示例。

相对路径均以 `config.yaml` 所在目录为基准，而不是以当前终端目录为基准。

Linux 首次使用时执行：

```sh
chmod +x ./l4d2-stats
```

### 5.2 SQLite 配置

SQLite 适用于 Dashboard 与游戏服务器同机部署：

```yaml
server:
  listen: "0.0.0.0:18848"

dashboard_database:
  path: "./dashboard.db"

stats_database:
  driver: "sqlite"
  dsn: "/absolute/path/left4dead2/addons/sourcemod/data/sqlite/l4d2_player_stats.sq3"

logging:
  file: "./logs/l4d2-stats.log"

monitor:
  enabled: true
```

Windows YAML 路径建议使用正斜杠：

```yaml
dsn: "E:/SteamLibrary/steamapps/common/Left 4 Dead 2/left4dead2/addons/sourcemod/data/sqlite/l4d2_player_stats.sq3"
```

不要把正在写入的 Stats SQLite 文件复制一份给 Dashboard 长期读取；Dashboard 必须指向采集器实际使用的文件。

### 5.3 MySQL 配置

```yaml
stats_database:
  driver: "mysql"
  dsn: "l4d2_dashboard:replace-me@tcp(127.0.0.1:3306)/l4d2_stats?parseTime=true"
```

### 5.4 PostgreSQL 配置

```yaml
stats_database:
  driver: "pgsql"
  dsn: "postgres://l4d2_dashboard:replace-me@127.0.0.1:5432/l4d2_stats?sslmode=disable"
```

生产环境跨机器连接数据库时应按实际环境启用 TLS。密码含有 URL 保留字符时，PostgreSQL URL 需要正确进行百分号编码。

Dashboard 的常规查询和聚合只需要 `SELECT` 权限。如果要在后台执行原始明细清理，数据库账号还需要对清理目标表的 `DELETE` 权限；不授予时不影响网页查询，但清理操作会失败。SQLite 清理要求运行用户对数据库文件及其所在目录具有写权限。

## 6. 首次启动和管理员初始化

先检查配置与数据库：

```sh
./l4d2-stats doctor --config ./config.yaml
```

启动：

```sh
./l4d2-stats serve --config ./config.yaml
```

首次没有管理员账号时，程序会在当前终端输出一次性设置令牌。令牌不会写入应用日志或数据库。

在自己的电脑浏览器中打开下面的地址，将 `服务器IP` 替换为游戏服务器或 Dashboard 所在机器的公网 IP/局域网 IP：

```text
http://服务器IP:18848/admin/setup
```

输入令牌并创建唯一管理员账号。初始化完成后，建议按以下顺序设置：

1. 站点外观、语言、浏览器标题和可选 SEO；
2. 添加游戏服务器，只需填写展示名称和连接地址；Dashboard 会从已持久化的 A2S 规则识别采集器的 `sm_lps_server_key`；
3. 配置页脚、更新公告和主页信息；
4. 需要 Steam 登录或游戏内页面时填写公开访问地址；Steam 登录再单独启用 Steam OpenID；
5. 在“游戏内页面”配置 MOTD 默认值、缓存和单服覆盖；
6. 在 Operations 页面检查聚合、数据库增长和保留策略。

## 7. 访问方式

### 7.1 直接通过 IP 访问

发布包的配置示例默认监听 `0.0.0.0:18848`。确认系统防火墙允许 TCP `18848` 后，访问：

```text
http://服务器IP:18848
```

### 7.2 使用 Nginx

需要使用域名或 HTTPS 时，参考 `examples/nginx.conf.example` 配置反向代理，并按自己的部署方式调整 Dashboard 监听地址。

### 7.3 配置 L4D2 原生 MOTD 游戏内页面

游戏内页面需要满足：

- 站点设置中的 `public_origin` 是玩家可访问的完整 HTTP/HTTPS 站点根地址；
- 采集器已设置唯一的 `sm_lps_server_key`；
- Dashboard 已对该服务器完成过一次成功 A2S 查询，并在规则中持久化这个 server key；
- 后台“游戏内页面”已启用。

在后台“游戏内页面”可以设置默认标题、描述、Banner、完整网站地址、首页模块、三个在线玩家生涯亮点指标，以及 Home、Player、Rankings、公告/文档的安全缓存预设。在“服务器管理”展开单台服务器后，可以让标题、描述、Banner、完整网站、亮点和三份文档继承全站、单独覆盖或隐藏（标题和亮点不能隐藏）。

回到“游戏内页面”的“MOTD 部署帮助”，选择服务器并复制生成内容到：

```text
left4dead2/motd.txt
```

格式如下：

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

`host.txt` 只能保留普通文本，不要把上述 HTML 放入 `host.txt`。Dashboard 和 Collector 不会远程写入这两个文件。

游戏内页面专门兼容旧 WebView：服务端直接输出 HTML，浏览器端没有 React、JavaScript 或 API 并发请求。“完整网站”按钮通过 Steam 外部浏览器入口打开普通系统浏览器，不会在 MOTD 内继续加载现代站点。部署后必须在生产所用的 Windows Steam + L4D2 客户端中实际验证按 H 打开页面、页面内导航和“完整网站”按钮。

缓存与 A2S 是两层独立机制：A2S 刷新周期继续在“站点设置 → 服务”配置；游戏内页面只读取已持久化快照，不会在玩家请求路径即时查服。Home 可选 10/30/60/120 秒，Player 可选 30/60/120/300 秒，Rankings 可选 60/120/300/600 秒，公告/文档可选 60/300/600/1800 秒，不支持关闭或任意数字。

Banner 与完整网站地址只接受不含账号密码的绝对 HTTP/HTTPS URL。Dashboard 不会下载、代理、探测或生成 Banner 缩略图；建议使用尺寸适中、支持 HTTP/HTTPS、无需 Cookie/Referer 的静态图片。服务器文档 Markdown 不加载外部图片，也不允许 raw HTML、脚本、iframe、音视频或嵌入组件。游戏内玩家资料始终按匿名访客的公开可见性渲染。

## 8. Linux systemd 安装

将二进制和配置放入最终目录并确认普通运行用户拥有：

- 二进制执行权限；
- `config.yaml` 读取权限；
- `dashboard.db` 所在目录写权限；
- 日志目录写权限；
- Stats SQLite 文件读取权限，以及使用清理功能时的写权限。

然后使用该用户通过 sudo 安装：

```sh
cd /opt/l4d2-stats
sudo ./l4d2-stats install --config ./config.yaml
```

服务会使用执行 sudo 前的原始用户身份，而不是固定服务账号。安装时写入的二进制、配置和工作目录均为绝对路径，因此安装完成后不要随意移动文件。

常用命令：

```sh
systemctl status l4d2-stats
sudo systemctl restart l4d2-stats
journalctl -u l4d2-stats -n 100 --no-pager
journalctl -u l4d2-stats -f
sudo ./l4d2-stats uninstall
```

`uninstall` 只停止并删除 systemd unit，不会删除二进制、配置、数据库或日志。

首次设置令牌可通过以下命令查看：

```sh
journalctl -u l4d2-stats -n 100 --no-pager
```

## 9. 完整验收清单

### 采集器

- `sm plugins list` 显示 `L4D2 Player Stats`；
- `sm_lps_status` 显示 `version=1.3.4`、`state=ready` 和 `schema=6/6`；
- `sm_lps_server_key` 不再是默认占位值，并在共享 DB 中唯一；
- 真人进入受支持模式后，`sm_lps_flush` 能写入数据；
- 完成一个真人参与的 Round 后，`lps_round_contexts` 出现一行规则快照；启用 Incident 时，状态中的分析队列与 expected/dropped 计数合理；
- SourceMod `logs/errors_*.log` 没有持续数据库错误。

### Dashboard

```sh
./l4d2-stats version
./l4d2-stats doctor --config ./config.yaml
./l4d2-stats doctor --deep --config ./config.yaml
```

访问：

```text
GET /api/v1/health/live
GET /api/v1/health/ready
```

确认：

- `/admin/setup` 能完成首次初始化；
- 后台添加的服务器能出现在首页；
- A2S 查询测试能返回在线状态；
- 后台“游戏内页面”能识别 A2S server key 并生成 `motd.txt`；
- Windows Steam + L4D2 中按 H 能打开当前服务器的 `/ingame`，Home、Player、Rankings、文档和公告可用，且“完整网站”打开外部普通浏览器；
- 使用已有 SteamID64 能打开个人中心；
- 排行榜、更新公告和 Operations 页面能正常加载。

## 10. 常见问题

| 现象 | 检查项 |
|---|---|
| 插件提示必须修改服务器键 | 编辑 `cfg/sourcemod/l4d2_player_stats.cfg` 中的 `sm_lps_server_key` 后重连 |
| 找不到数据库配置 | 确认 `databases.cfg` 中存在名为 `l4d2_player_stats` 的块，且 KeyValues 大括号完整 |
| 数据库无法初始化 | 确认三个方言对应的 `0001_initial.sql` 已随插件部署，并检查数据库账号建表权限 |
| MySQL/PostgreSQL 连接失败 | 检查防火墙、监听地址、端口、账号授权范围、DSN 转义和 TLS 设置 |
| Dashboard 找不到 SQLite | 使用绝对路径或确认相对路径是相对于 `config.yaml`；不要指向旧副本 |
| 首次设置令牌找不到 | 前台启动查看 stderr；systemd 部署使用 `journalctl -u l4d2-stats` |
| A2S 查询失败 | 检查游戏服务器 UDP 端口、防火墙和地址；本地 listen server 可能不响应外部 A2S |
| 首页或游戏内页面无法关联统计 | 先确认 A2S 规则包含 `sm_lps_server_key`，再在后台对这台服务器执行查询测试；配置值必须与 Stats DB 中的 server key 完全一致 |
| MOTD 只显示空白或旧内容 | 确认 `motd.txt` 使用后台生成的绝对 HTTPS/HTTP 地址，检查反向代理和浏览器缓存，并在真实 L4D2 客户端重开 MOTD |
| 清理操作失败 | 数据库账号需要 `DELETE` 权限；SQLite 还需要文件和目录写权限 |
| 日志持续增长 | 检查 `logging.max_size_mb`、`max_backups`、`max_age_days`，默认会由 Lumberjack 轮转 |

升级已有安装请继续阅读 [UPGRADE.zh-CN.md](UPGRADE.zh-CN.md)。
