# 首版发布前数据库、性能与安全审计

审计日期：2026-08-05

审计范围：SourceMod 采集器数据库结构、Dashboard Go 服务、React 前端、公开/管理 API、缓存、A2S、聚合与清理流程。

目标版本：当前 `dev` 分支（首个公开版本候选）。

## 执行摘要

- 未发现 P0/P1 级发布阻断问题。
- 本轮已修复 Stats DB 索引、Dashboard 聚合过滤、重复请求、代理地址识别、全站限流、月度/终身汇总、分批清理和前端依赖版本漂移问题。
- 当前索引适合首版目标负载。新建 SQLite 的关键查询计划已确认使用索引；MySQL/PostgreSQL 已通过结构与 sqlc 编译校验，但仍需真实数据库与真实多人负载验收。
- 正常页面请求量合理：主页 3 个公开请求，个人中心首次查询约 6 个统计请求加共享站点配置，排行榜第一页已从 2 个同类排行请求降为 1 个。
- 原 3 项 P2 容量风险均已处理；当前主要剩余边界是 MySQL/PostgreSQL 真实执行计划和并发验收。
- Go 可达依赖漏洞为 0。npm 审计报告 1 个 React Router 高危公告，但公告明确只影响未使用的 unstable RSC API；当前项目是纯浏览器路由 SPA，不存在对应执行路径。

## 本轮已完成的优化

| 项目 | 结果 |
|---|---|
| Stats DB 增量聚合 | 为各增量来源增加 `last_saved_at` 前导索引；`MAX(last_saved_at)` 和变更日检测可走覆盖/范围索引。 |
| Stats DB 日聚合 | 为 Session、Run、Round、Segment 增加 `started_at` 前导索引。 |
| Stats DB 清理 | 为 `ended_at`、`finalized_at` 增加索引；装备/职业明细删除可先通过 Segment 的结束时间索引定位。 |
| 启动恢复 | 为 Server Boot 和 Segment 的状态定位增加复合索引。 |
| Dashboard 聚合查询 | 将 SteamID、服务器、模式、日期和聚合类型过滤全部下推到 SQLite，并使用绑定参数。 |
| Dashboard 增量写入 | 新增 `aggregate_rows(day)`，避免每个变更日删除前扫描整张聚合表。 |
| 排行榜前端 | 第 1 页列表直接复用前 10 条生成图表，不再同时请求 20 条和 10 条相同排行。 |
| 反向代理限流 | 仅信任本机回环代理提供的 `X-Real-IP`，兼容文档中的同机 Nginx，同时拒绝远程客户端伪造。 |
| 测试稳健性 | bcrypt 在竞态检测下较慢，HTTP 测试改为显式 15 秒超时并先检查错误，避免空响应解引用。 |
| 构建版本 | Windows/Linux 构建脚本的默认版本从遗留的 `0.8.3-dev` 对齐到当前 `0.9.2-dev`。正式发布仍可通过 `L4D2_STATS_VERSION` 注入 tag。 |

## 索引审计

### Stats DB

首版新数据库共创建 31 个显式索引。主要访问路径如下：

| 工作负载 | 主要筛选/排序 | 使用索引 |
|---|---|---|
| 玩家 Session 分页 | `steam_id, started_at DESC` | `lps_idx_sessions_steam_started` |
| 玩家章节与聚合 | `steam_id, started_at` | `lps_idx_segments_steam_started` |
| 增量水位/变更日 | `last_saved_at` | 各表 `*_saved` 索引 |
| UTC 日聚合 | `started_at` 范围 | Session/Run/Round/Segment 的 `*_started` 索引 |
| Session 清理 | `ended_at < cutoff` | `lps_idx_sessions_ended` |
| 明细清理 | Segment `ended_at < cutoff` | `lps_idx_segments_ended` + 明细表主键 |
| 对抗结果清理 | `finalized_at < cutoff` | 两个 `*_finalized` 索引 |
| 崩溃恢复 | `server_key/status`、`session_id/status` | 两个状态复合索引 |

新建 SQLite 的 `EXPLAIN QUERY PLAN` 验证结果：

- 水位查询使用 `lps_idx_sessions_saved` 覆盖索引。
- 变更日查询使用 `last_saved_at` 范围搜索。
- 聚合范围使用 `started_at` 范围搜索。
- Session 清理使用 `lps_idx_sessions_ended`。
- 装备清理子查询使用 `lps_idx_segments_ended`，目标表使用主键定位。
- 玩家 Session 使用 `lps_idx_sessions_steam_started`。

索引会增加采集器每 5 分钟快照写入时的维护成本，但目标服务器规模下，批量保存的收益高于额外索引写放大。真实 MySQL/PostgreSQL 大型服仍应观察写入延迟和慢查询。

注意：项目尚未发布，因此索引直接加入初始迁移。已有的开发数据库仍显示 schema version 1，不会自动补齐这些索引；最终验收前应清空并让插件重新创建 Stats DB。

### Dashboard DB

`aggregate_rows` 日聚合及月度/终身汇总表均按玩家、类型、服务器和时间建立互补索引：

- `aggregate_rows_player_day (steam_id, day, kind)`：个人中心。
- `aggregate_rows_ranking (kind, mode, day, server_key)`：全服/玩法排行。
- `aggregate_rows_kind_server_day (kind, server_key, day, mode, steam_id)`：指定服务器排行。
- `aggregate_rows_day (day)`：增量重建时删除指定 UTC 日。

SQLite 查询计划已确认个人查询、全服排行、指定服务器排行和按日删除分别选择对应索引。全服总览和全周期排行榜读取终身汇总，长期趋势读取月度汇总，有限日期范围仍读取日聚合。

## 接口与前端请求量

| 页面/操作 | 正常请求 | 说明 |
|---|---:|---|
| 应用公共外壳 | 1 | `/site`；多个组件使用相同 TanStack Query key，会共享和合并请求，缓存 5 分钟。 |
| 主页首次加载 | 2 + 公共外壳 | `/dashboard/overview` 与 `/servers/status`。总览缓存 60 秒。 |
| 主页持续在线 | 1/刷新周期 | 只轮询 `/servers/status`；管理员可设 5–60 秒，后端 A2S 同期缓存、错峰和重试。 |
| 个人中心首次查询 | 6 + 公共外壳 | summary、activity、PvE、Versus、Session 首屏、章节首屏；并行加载。 |
| 个人筛选变化 | 3 | activity、PvE、Versus；Session/章节不重复请求。 |
| 排行榜第 1 页 | 1 + 服务器目录 | 20 条列表的前 10 条直接生成图表。 |
| 排行榜后续页 | 2 | 当前 20 条 + 缓存的 Top 10；Top 10 的 query key 稳定，通常不重复过网。 |
| 公告页 | 2 | 年份 + 首批 5 条；其余每次点击再加载 5 条。 |

请求数不是当前瓶颈。个人中心采用多个窄接口，可以并行、局部失败和独立缓存；暂时没有必要为了减少 HTTP 数量合并成一个超大接口。

前端已按路由懒加载。生产构建中 ECharts 和 Markdown 编辑器各约 0.86–0.90 MB（gzip 约 0.28–0.31 MB），只在对应页面加载，不进入主页首屏。静态 hash 资源使用一年 immutable 缓存。

## 缓存与并发边界

- 首页总览：60 秒缓存 + singleflight。
- 个人统计：60 秒缓存；最多 256 个 SteamID、1024 个缓存项 + singleflight。
- 排行榜：60 秒缓存；最多 128 个缓存项 + singleflight。
- A2S：只查询数据库中已登记且启用的服务器；最多 4 个并发；2 秒超时；管理员配置刷新周期、抖动和 1–3 次重试；失败可回退 5 分钟内的最后成功数据。
- Stats DB：默认最多 10 个连接、5 个空闲连接、5 秒查询超时。
- Dashboard SQLite：最多 4 个连接、2 个空闲连接。
- HTTP：1 MB 请求体上限；读取 10 秒、写入 15 秒、空闲 60 秒。
- 日志：Lumberjack 按大小、数量和天数轮转，不会无限增长。

## 风险发现

### P2-01：公开统计接口没有独立的请求限流

- 严重级别：P2
- 置信度：高
- 位置：`dashboard/internal/server/rate_limit.go`
- 修复：使用 Fiber 官方 limiter 的滑动窗口覆盖页面、静态资源和 API；每个可信客户端 IP 每分钟 300 次，API 超限返回统一 JSON 429。
- 边界：内存状态随进程重启清空，单机部署不需要跨实例共享；额度刻意宽松，不替代高风险公网场景的边缘防护。
- 状态：已修复；Fiber 官方滑动窗口中间件覆盖全站，每个可信客户端 IP 每分钟允许 300 次请求。

### P2-02：全服总览读取量随日聚合历史线性增长

- 严重级别：P2
- 置信度：高
- 位置：`dashboard/database/dashboard/migrations/00008_aggregate_rollups.sql`、`dashboard/internal/store/aggregate_rollup.go`
- 修复：新增月度和终身汇总；全服总览与全周期排行榜读取终身表，长期趋势读取月度表，最近范围仍读取日表。
- 一致性：全量构建原子替换三层数据；增量构建先扣除旧日值再加入新日值，旧 Dashboard DB 升级时自动补建汇总。
- 状态：已修复；新增月度和终身汇总表，增量聚合会差量修正受影响的汇总键。

### P2-03：大批量原始数据清理使用单事务

- 严重级别：P2
- 置信度：高
- 位置：`dashboard/internal/store/stats_aggregate.go`
- 修复：六类目标均先按主键读取最多 500 行，再在短事务中精确删除并提交，直至没有候选行。
- 边界：完整清理仍可能持续较久，因此超时上限为 10 分钟，管理员仍应在低峰期执行。
- 状态：已修复；各清理目标按 500 行短事务分批删除，仍只由管理员显式触发。

### P3-01：前端依赖声明包含 `latest`

- 严重级别：P3
- 置信度：高
- 位置：`dashboard/frontend/package.json`、`dashboard/frontend/pnpm-lock.yaml`
- 修复：所有直接运行时和开发依赖均固定到当前已验证的精确版本；CI 继续使用 frozen lockfile。
- 状态：已修复；所有直接前端依赖均固定到当前 lockfile 的精确版本。

### P3-02：npm 审计报告 React Router RSC 公告

- 严重级别：P3（对本项目的实际风险）
- 置信度：高
- 位置：`dashboard/frontend/package.json:26`、`pnpm-lock.yaml`
- 证据：`react-router-dom 7.18.2` 依赖 `react-router 7.18.2`，npm 报告 [GHSA-qwww-vcr4-c8h2](https://github.com/advisories/GHSA-qwww-vcr4-c8h2)。公告明确仅影响 unstable RSC API；本项目使用 `BrowserRouter`，未使用 RSC 或 React Router action 服务端执行。
- 影响：当前没有可达攻击路径，但自动审计仍会返回非零状态。
- 建议：不要强制覆盖到与 `react-router-dom` 不匹配的 8.x；等待兼容的 `react-router-dom` 修复版本后正常升级。
- 状态：已评估、暂不修复。

### P3-03：A2S 与总览 singleflight 的底层加载不随访客取消

- 严重级别：P3
- 置信度：高
- 位置：`dashboard/internal/a2s/provider.go:266`、`dashboard/internal/service/overview.go:40`
- 证据：共享加载使用 `context.Background()`，调用方断开只会停止等待，不会取消已合并的底层任务。
- 影响：少量任务会在客户端离开后继续完成；A2S 有 4 并发和 2 秒查询超时，统计有数据库超时，因此资源仍有界。
- 建议：保留当前 singleflight 语义；只有监控显示取消后工作明显浪费时，再引入引用计数式共享上下文。
- 状态：接受。

### P3-04：MySQL/PostgreSQL 尚未做真实执行计划与并发验收

- 严重级别：P3
- 置信度：高
- 位置：`database/migrations/mysql/0001_initial.sql`、`database/migrations/pgsql/0001_initial.sql`
- 证据：三方言迁移结构、sqlc 生成和 Go 编译均通过；本轮只有 SQLite 执行了真实迁移和 `EXPLAIN`。
- 影响：不同优化器、字符集、事务日志和网络延迟下可能出现 SQLite 测试无法发现的慢查询。
- 建议：发布说明标记 MySQL/PostgreSQL 为可用但需实服观察；在综合服务器试运行时采集慢查询和聚合/清理耗时。
- 状态：开放验收项。

## 已验证的安全边界

- 管理后台 JWT、Token 版本失效、HttpOnly/SameSite Cookie、CSRF 中间件均存在测试覆盖。
- 登录与首次设置有有界内存限流；同机 Nginx 的真实客户端地址可正确参与限流，且只信任回环代理。
- 首次设置令牌不写入轮转日志或数据库。
- 管理接口设置 `no-store`；API 404 不回退 React HTML。
- Markdown 不启用原始 HTML 渲染；外部链接使用安全 `rel`。
- 公开 DTO 与查询不选择 Session IP 地址。
- 动态 Dashboard 聚合查询全部使用绑定参数，未发现 SQL 注入路径。
- 未发现提交到仓库的私钥、JWT 密钥或密码哈希。

## 验证结果

| 检查 | 结果 |
|---|---|
| `go tool sqlc generate` | 通过，生成代码已同步。 |
| `go test ./...` | 通过。 |
| `go test -race ./...` | 通过。 |
| `go vet ./...` | 通过。 |
| 前端 ESLint | 通过。 |
| 前端 Vitest | 3 个文件、6 个测试通过。 |
| 前端生产构建 | 通过；仅有按路由懒加载的大 chunk 提示。 |
| 三方言迁移结构校验 | SQLite 47 条、MySQL 38 条、PostgreSQL 47 条通过。 |
| SQLite 集成迁移 | 16 张表、31 个索引，通过重复执行与契约测试。 |
| `govulncheck` | 项目可达漏洞 0。 |
| `pnpm audit --prod` | 1 个 RSC 专用公告；当前项目不可达，详见 P3-02。 |
| 敏感信息模式扫描 | 未发现候选。 |

## 发布建议

可以进入首版候选测试，但建议采用“先单服/小规模试运行，再公开推广”的方式：

1. 清空现有开发 Stats DB，让插件以最新初始迁移重新创建数据库并确认 31 个索引。
2. 运行一次完整聚合，记录耗时、聚合行数、Stats DB 与 Dashboard DB 大小。
3. 在低峰期用少量到期测试数据验证清理预览和删除，不要首次就清理大量历史数据。
4. 综合服务器试运行 7–14 天后复查：采集保存耗时、聚合耗时、Dashboard p95、数据库增长和 A2S 失败率。
