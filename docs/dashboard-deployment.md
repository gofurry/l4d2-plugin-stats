# Dashboard 部署指南

生产形态是一个 Go 二进制和一个最小 `config.yaml`。React 资源和 Dashboard 初始结构已嵌入二进制；运行后才生成 Dashboard DB、SQLite sidecar 和轮转日志。

## 推荐目录

```text
/opt/l4d2-stats/
├─ l4d2-stats
├─ config.yaml
├─ dashboard.db
└─ logs/
```

Dashboard 的日常查询与聚合对 Stats DB 只读。只使用展示功能时，SQLite 文件及目录只需可读，MySQL/PostgreSQL 可使用只有 `SELECT` 权限的专用账号。若要在管理页执行原始数据清理，SQLite 文件及目录需要可写；MySQL/PostgreSQL 的同一 DSN 需要对清理目标表具备 `DELETE` 权限。Dashboard DB 和日志目录必须对实际运行用户可写。

## 最小配置

```yaml
server:
  listen: "127.0.0.1:18848"
dashboard_database:
  path: "./dashboard.db"
stats_database:
  driver: "sqlite"
  dsn: "/absolute/path/l4d2_player_stats.sq3"
logging:
  file: "./logs/l4d2-stats.log"

monitor:
  enabled: true
```

监听地址、数据库连接和日志属于运维配置；界面语言、背景图片、页脚链接、Steam 登录、管理员和游戏服务器全部在网页后台管理。

`monitor.enabled` 显式启用轻量运行监控。管理员登录后可从左侧入口打开；页面和 JSON 快照共用 `/api/v1/admin/monitor`，都会验证管理员 JWT。可选的 `monitor.refresh` 默认为 `2s`，`monitor.disk_paths` 可填写相对于配置文件的磁盘路径列表。

## 首次启动与网页设置

直接运行：

```sh
./l4d2-stats serve --config ./config.yaml
```

首次启动会把 30 分钟有效的一次性令牌输出到 stderr。打开 `http://127.0.0.1:18848/admin/setup` 创建唯一管理员。令牌不写入应用日志或数据库，过期后重启服务即可生成新令牌。

项目未发布，v0.9.1 不维护旧 Dashboard DB 升级包袱。测试过旧版时，停止进程后删除旧 `dashboard.db`、`dashboard.db-shm` 和 `dashboard.db-wal` 再启动；不要删除采集器的 Stats DB。

## systemd

以希望运行服务的普通用户登录后执行：

```sh
cd /opt/l4d2-stats
sudo ./l4d2-stats install --config ./config.yaml
```

安装程序使用 `SUDO_USER/SUDO_UID/SUDO_GID` 对应的原始用户，将二进制、配置和工作目录写成绝对路径，并验证所需读写权限。它不会创建固定服务账号或递归修改所有权。root 直接安装会收到警告。

查看状态和首次设置令牌：

```sh
systemctl status l4d2-stats
journalctl -u l4d2-stats -n 100 --no-pager
```

卸载只停用并删除 unit，不删除二进制、配置、数据库或日志：

```sh
sudo ./l4d2-stats uninstall
```

## 访问方式

没有域名时可监听 `0.0.0.0:18848` 并通过 `http://IP:18848` 访问。只有启用 Steam 登录时，才需要在后台填写玩家实际访问本站的完整地址，例如 `http://203.0.113.10:18848`，供 Steam 验证后返回；手动 SteamID64 查询不要求该地址或域名。

有域名时建议 Go 继续监听 `127.0.0.1:18848`，由 Nginx 终止 HTTPS：

```nginx
server {
    listen 443 ssl;
    server_name stats.example.com;

    location / {
        proxy_pass http://127.0.0.1:18848;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

后台公开地址填写 `https://stats.example.com`。使用 HTTPS 后建议重启一次服务，使 Fiber CSRF Cookie 的 Secure 初始设置与公开地址保持一致。页脚默认不展示；启用后可填写 ICP 备案等纯文本与 `http/https` 链接。

## 检查、日志与回滚

```sh
./l4d2-stats doctor --config ./config.yaml
./l4d2-stats migrate status --config ./config.yaml
curl -fsS http://127.0.0.1:18848/api/v1/health/live
curl -fsS http://127.0.0.1:18848/api/v1/health/ready
```

`ready` 检查两个数据库和 Stats schema；A2S 故障不影响 liveness。Lumberjack 按大小、份数和保留天数轮转应用日志，启动错误仍进入 journald。

升级前停止服务并备份二进制、配置和 Dashboard DB。首次执行清理前还应完整备份 Stats DB；清理后的旧逐条 Session/比赛结果不能通过回滚 Dashboard 二进制恢复。
