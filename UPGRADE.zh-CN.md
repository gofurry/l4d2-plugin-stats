# L4D2 Player Stats 升级与回滚

## 升级前备份

至少备份：

```text
left4dead2/addons/sourcemod/plugins/l4d2_player_stats.smx
left4dead2/cfg/sourcemod/l4d2_player_stats.cfg
left4dead2/addons/sourcemod/configs/databases.cfg
Stats 数据库
Dashboard 运行目录中的 l4d2-stats、config.yaml、dashboard.db
```

SQLite 仍在写入时不要直接复制单个 `.sq3` 文件。先停止相关进程，或使用 SQLite 在线备份工具，确保 WAL 中的数据一并处理。

## 升级采集器

1. 将新发布包中的 `left4dead2` 文件夹合并到服务端同名目录；
2. 确认新迁移文件也已复制，不要只替换 `.smx`；
3. 保留已有 `cfg/sourcemod/l4d2_player_stats.cfg` 和 `databases.cfg`；
4. 在服务器控制台执行：

```text
sm plugins reload l4d2_player_stats
sm_lps_status
```

确认版本、数据库状态和 schema 版本符合新发布说明。

## 升级 Dashboard

Linux systemd：

```sh
sudo systemctl stop l4d2-stats
cp ./l4d2-stats ./l4d2-stats.previous
cp /path/to/new/l4d2-stats ./l4d2-stats
chmod +x ./l4d2-stats
./l4d2-stats doctor --config ./config.yaml
sudo systemctl start l4d2-stats
```

Windows：

1. 停止正在运行的 `l4d2-stats.exe`；
2. 将旧二进制保留为 `l4d2-stats.previous.exe`；
3. 替换为新二进制；
4. 使用原 `config.yaml` 执行 `doctor` 后再启动。

Dashboard 启动时会自动迁移自己的 `dashboard.db`。不要使用发布包中的示例配置覆盖现有 `config.yaml`。

升级后检查：

```sh
./l4d2-stats version
./l4d2-stats doctor --config ./config.yaml
```

并访问：

```text
/api/v1/health/live
/api/v1/health/ready
```

## 回滚

如果新版本仅修改二进制且没有执行不可逆数据库迁移，可停止服务并恢复上一份二进制：

```sh
sudo systemctl stop l4d2-stats
cp ./l4d2-stats.previous ./l4d2-stats
sudo systemctl start l4d2-stats
```

如果新版本已经升级 Stats DB 或 `dashboard.db` 结构，旧二进制未必兼容新 schema。此时应同时恢复升级前数据库备份，不要手工修改迁移版本号。

原始数据清理是不可逆操作。执行清理前应确认聚合完成并备份 Stats DB；回滚二进制不会恢复被删除的原始记录。
