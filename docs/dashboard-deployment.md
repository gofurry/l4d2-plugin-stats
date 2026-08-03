# Dashboard deployment

Dashboard 的生产形态是一个 Go 二进制和一个 `config.yaml`。React 资源、Dashboard SQLite 迁移都已嵌入二进制；运行后产生的 `dashboard.db` 和日志与配置文件放在同一部署目录即可。

## 推荐目录

```text
/opt/l4d2-stats/
├─ l4d2-stats
├─ config.yaml
├─ dashboard.db              # 首次启动后生成
└─ logs/                     # 首次启动后生成并自动轮转
```

不要把 Stats DB 的写权限授予 Dashboard。SQLite 文件及其所在目录只需可读；MySQL/PostgreSQL 应创建只有 `SELECT` 权限的专用账号。

## 部署前检查

```sh
cd /opt/l4d2-stats
./l4d2-stats doctor --config ./config.yaml
./l4d2-stats migrate status --config ./config.yaml
```

`doctor` 会验证严格配置、Dashboard DB、Stats DB 连接和 Stats schema。Dashboard DB 会由嵌入的 Goose 迁移自动升级；Stats DB 永远不会由此程序迁移。

## 注册 systemd

以实际希望运行服务的普通用户登录，然后执行：

```sh
cd /opt/l4d2-stats
sudo ./l4d2-stats install --config ./config.yaml
```

`install` 会：

- 将二进制、配置和工作目录转换为绝对路径；
- 使用 `SUDO_USER/SUDO_UID/SUDO_GID` 对应的原始用户，而不是固定账号；
- 验证配置可读、二进制可执行，以及 Dashboard/日志目录可写；
- 只创建缺失的运行目录，不递归修改已有目录所有权；
- 生成 `l4d2-stats.service`，启用 `Restart=on-failure`、`NoNewPrivileges`、`PrivateTmp` 和有限文件系统保护。

root 直接调用时服务会以 root 运行并打印警告，不推荐这样做。

查看状态与启动失败信息：

```sh
systemctl status l4d2-stats
journalctl -u l4d2-stats -n 100 --no-pager
```

卸载只删除 systemd unit，不删除二进制、配置、数据库或日志：

```sh
sudo ./l4d2-stats uninstall
```

## 健康检查

```sh
curl -fsS http://127.0.0.1:18848/api/v1/health/live
curl -fsS http://127.0.0.1:18848/api/v1/health/ready
```

`live` 只表示进程可响应；`ready` 还会检查 Dashboard DB、Stats DB 和 schema。A2S 故障不会影响 `live`，也不会阻塞历史统计 API。

## 直接使用 IP

不使用域名时，可把监听地址改为：

```yaml
server:
  listen: "0.0.0.0:18848"
  public_base_url: ""
```

随后只在主机防火墙中开放确实需要的来源。Steam OpenID 尚未启用，因此 `public_base_url` 可以为空；未来启用 OpenID 时需要稳定的公开回调 URL。

## Nginx 反向代理

有域名时仍建议 Go 只监听 `127.0.0.1:18848`，由 Nginx 负责 HTTPS：

```nginx
server {
    listen 80;
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

启用管理员后台前必须配置 HTTPS 和可信代理边界。ICP备案等内容可在 `bootstrap.site.footer_text` 与 `footer_links` 中设置；文本支持换行，链接仅允许 `http/https`。

## 日志与空间上限

应用日志由 Lumberjack 按 `max_size_mb`、`max_backups` 和 `max_age_days` 轮转并可压缩，不会无限保留。启动阶段无法打开日志文件时，错误仍由 systemd 记录到 journald。

## 升级与回滚

升级前备份当前二进制、配置和 Dashboard DB：

```sh
systemctl stop l4d2-stats
cp l4d2-stats l4d2-stats.previous
cp config.yaml config.yaml.previous
cp dashboard.db dashboard.db.previous
```

替换新二进制后执行：

```sh
./l4d2-stats doctor --config ./config.yaml
systemctl start l4d2-stats
curl -fsS http://127.0.0.1:18848/api/v1/health/ready
```

若升级失败，停止服务，恢复 `.previous` 二进制与对应 Dashboard DB，再启动服务。Stats DB 是只读的，不参与 Dashboard 回滚。
