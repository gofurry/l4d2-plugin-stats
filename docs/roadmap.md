# Roadmap

## Current Position

SourceMod 采集器已经完成当前计划内的数据库地基、身份与生命周期、PvE 统计、Versus
统计和第一版查询契约。采集器后续的多人、多数据库与发布加固由维护者独立推进，不再
列入本路线图。

Dashboard v0.8.0–v0.8.1 已完成首轮工程地基和可用首页。本路线图从 Go API 与内嵌 React 展示端继续推进。Go 服务只读采集器拥有的统计表，不修改
采集口径，也不执行采集器迁移；网页自身的服务器目录、管理员会话和展示配置使用独立
的 Dashboard 存储边界。

## Architecture Decisions

- Go 1.26、Fiber v3、Cobra、Zap/Lumberjack。
- React 19、TypeScript、Vite、Ant Design 6、Tailwind CSS 4、SCSS Modules。
- React 源码位于 `dashboard/frontend`，生产构建输出到 `dashboard/web/dist` 并嵌入 Go
  二进制；开发时由 Vite 代理 `/api/v1`。
- 统计数据库支持 SQLite、MySQL 和 PostgreSQL，并始终以只读身份连接；sqlc 为三种方言生成独立查询包。
- Dashboard 配置与统计数据库分离；前者由 Go 负责迁移，后者仍由 SourceMod 负责迁移。
- Dashboard DB 固定使用纯 Go SQLite 和 Goose；首次配置通过幂等 Bootstrap 导入。
- Steam OpenID 只用于验证并取得 SteamID64，不建立玩家账号、角色或密码系统。
- 浏览器可持久化最近查询的 SteamID64，但它只是公开查询偏好，不能作为权限凭据。
- 后台为全局唯一管理员，密码只保存 easyhash bcrypt 哈希；后续使用配置密钥签发 HS256 JWT，并放入 HttpOnly Cookie，不建立服务端 Session。
- A2S 由 Go 只查询管理员登记的服务器并短时缓存；浏览器不直接发送 UDP A2S 请求。
- 公共 API、Steam 身份流程与管理员 API 使用不同路由和权限边界。

## Version Plan

### v0.8.0 - Go and Embedded React Foundation

**Status:** Implemented

**Goal:** 建立可启动、可测试、可嵌入前端的 Dashboard 工程地基。

#### Tasks

- [x] 初始化 `dashboard` Go 模块、严格配置加载、Zap/Lumberjack 日志和优雅关闭
- [x] 初始化 React 19 + Vite + TypeScript + Ant Design 6 + Tailwind 4 + SCSS Modules
- [x] 建立开发代理和生产 `embed.FS` 静态资源服务
- [x] 建立 `/api/v1/health/live` 与 `/api/v1/health/ready`
- [x] 接入 Fiber request ID、recover、helmet、compress 和统一错误响应
- [x] 建立 Stats DB 与 Dashboard DB 两套独立连接配置
- [x] 为 SQLite、MySQL、PostgreSQL 建立只读 Stats DB 驱动边界
- [x] 增加 Go 单元测试、前端检查、Cobra/systemd 和单二进制构建任务

#### Acceptance Criteria

- 开发模式下 Vite 与 Go API 可分别启动且 `/api/v1` 代理正常
- 生产构建只需一个 Go 二进制即可提供 API 和 React SPA
- API 路由不存在时返回 JSON 404，前端路由才回退到 `index.html`
- Stats DB 无写权限时服务仍可正常查询，Go 不执行采集器迁移

---

### v0.8.1 - Public Dashboard and Server Status

**Status:** Implemented

**Goal:** 提供首页最重要的历史统计与主服务器实时状态。

#### Tasks

- [x] 定义公开 Dashboard DTO，确保不返回 IP、内部 ID和数据库细节
- [x] 实现总玩家、近期活跃、游玩时长、PvE 与 Versus 核心摘要查询
- [x] 建立 Dashboard 服务器目录，映射展示名称、`server_key`、连接地址和 A2S 查询地址
- [x] 支持唯一主服务器、启用状态和显示顺序
- [x] 使用 steam-go A2S 在 Go 侧查询管理员登记的服务器
- [x] 增加超时、并发上限、短缓存和最后一次成功结果回退
- [x] 完成首页状态卡、核心指标、模式分区、自定义页脚和加载/离线状态

#### Acceptance Criteria

- A2S 不接受访客提供的任意主机或端口
- 首页请求不会对每位访客同步发起 UDP 查询
- A2S 超时不会拖垮首页或阻塞统计数据库查询
- PvE 与 Versus 指标不会混合聚合

---

### v0.8.2 - Steam Identity and Player Center

**Status:** Planned

**Goal:** 允许访客通过 Steam OpenID 或手动 SteamID64 查询个人统计。

#### Tasks

- [ ] 使用 steam-go OpenID 实现登录跳转、state 校验和回调验证
- [ ] 将验证得到的 SteamID64 安全交给前端并持久化为最近查询偏好
- [ ] 支持手动填写、校验、保存、切换和清除 SteamID64
- [ ] 实现玩家摘要、PvE、Versus、Session 和章节成绩只读 API
- [ ] 增加分页、范围上限、无数据状态和模式隔离
- [ ] 完成个人中心概览及 PvE/Versus 分页展示

#### Acceptance Criteria

- Steam OpenID 验证只在 Go 服务端完成
- 修改浏览器中的 SteamID 只能查询公开数据，不能获得后台权限
- 不创建玩家账号表、密码、角色、刷新令牌或公开写入接口
- 所有个人查询均能限制行数和时间范围

---

### v0.8.3 - Single Administrator and Server Management

**Status:** Planned

**Goal:** 让全局唯一管理员安全管理首页服务器与展示设置。

#### Tasks

- [ ] 使用 easyhash bcrypt 校验配置中的管理员密码哈希
- [ ] 使用配置中的 JWT 密钥签发 HS256 Token，登录后写入 HttpOnly Cookie，退出时清除
- [ ] 配置 HttpOnly、Secure、SameSite 和 8 小时 Token 有效期
- [ ] 增加登录限流、CSRF、防缓存和安全审计日志
- [ ] 实现服务器目录的新增、编辑、启停、排序和设为主服务器
- [ ] 实现 A2S 地址校验和管理员主动测试查询
- [ ] 管理接口与公开接口使用独立路由组

#### Acceptance Criteria

- 仓库和普通配置文件中不出现管理员明文密码
- 未登录访客无法读取或修改后台配置
- 任意时刻最多只有一个主服务器
- 管理员 JWT 与认证状态不写入统计数据库

---

### v0.9.0 - Full Statistics Experience

**Status:** Planned

**Goal:** 将已冻结的 PvE 与 Versus 契约完整转化为可浏览的统计体验。

#### Tasks

- [ ] 完成 PvE 战役、章节、武器、近战、投掷物、救援和技巧展示
- [ ] 完成 Versus 比分、半场、幸存者、感染者职业、控制和能力展示
- [ ] 增加服务器、模式、地图、日期和玩家筛选
- [ ] 增加排行、趋势图、明细表和可分享的只读 URL
- [ ] 为复杂查询建立明确的超时、分页和缓存策略
- [ ] 增加中英文界面资源与统一数字/时长格式

#### Acceptance Criteria

- 每项展示指标都可追溯到统计契约或已审核的聚合公式
- 页面不会把 Bot、真人、PvE 与 Versus 口径混淆
- 查询失败、数据库离线和部分数据缺失都有可理解的降级显示

---

### v0.9.1 - Aggregation and Production Hardening

**Status:** Planned

**Goal:** 在长期数据量和真实部署环境中保持查询稳定、安全且易运维。

#### Tasks

- [ ] 根据真实查询计划和数据量设计可重建的聚合读模型
- [ ] 保持原始 Session、Run、Round 和 Segment 数据永久保留
- [ ] 为三种数据库执行等价查询、索引和分页回归
- [ ] 增加缓存命中率、查询耗时、A2S 状态和连接池指标
- [ ] 完成反向代理、HTTPS、可信代理、备份恢复和密钥轮换文档
- [ ] 完成长时间运行、并发访问、数据库故障和 A2S 故障测试
- [ ] 建立 CI 的 Go、前端、嵌入资源和跨平台产物验证

#### Acceptance Criteria

- 常用首页和个人页查询有稳定上限，不随原始数据线性恶化
- 聚合表可以从原始数据完整重建，不能成为唯一事实来源
- 三种数据库在相同筛选条件下返回等价结果
- 日志、缓存、JWT Cookie 和后台数据不会无限增长

---

### v1.0.0-alpha.1 - Dashboard Stability Freeze

**Status:** Planned

**Goal:** 冻结首个公开版本的 API、页面、部署配置和安全边界。

#### Tasks

- [ ] 冻结公开 API v1 DTO、错误码和分页约定
- [ ] 冻结 Stats DB 只读兼容边界与 Dashboard DB 升级策略
- [ ] 完成 SQLite、MySQL、PostgreSQL 真实部署验收
- [ ] 完成 Steam OpenID、管理员登录和反向代理安全回归
- [ ] 完成安装、配置、升级、备份、隐私和故障排查文档
- [ ] 收集至少一轮真实服务器反馈

#### Acceptance Criteria

- 没有已知高优先级数据误读、越权、敏感信息泄露或资源无限增长问题
- 从干净仓库可以生成可部署的单二进制与配置示例
- Alpha 后破坏性 API 或配置变化必须进入兼容性审查

---

### v1.0.0 - First Stable Dashboard Release

**Status:** Planned

**Goal:** 发布第一版可长期部署的 Go + React 统计展示系统。

#### Acceptance Criteria

- 单二进制提供公开 Dashboard、个人中心、Steam OpenID 和单管理员后台
- 首页主服务器状态、PvE 与 Versus 展示通过真实环境验收
- 三种统计数据库均有清晰的只读部署方式
- 发布包、校验值、变更日志、升级和回滚说明完整

## Deferred Beyond v1

- 玩家账号、角色、密码、跨服务器账号体系和公开写入 API
- 多管理员、细粒度 RBAC 和组织空间
- 浏览器直接 A2S、任意访客地址探测和 UDP 代理接口
- 实时逐事件流水、比赛回放和逐时刻推进曲线
- 多实例 Dashboard 与跨实例密钥轮换/认证协调
- 在可靠聚合和归档工具完成前删除原始 Session、Run、Round 或 Segment
