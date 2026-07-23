# Goal 1 Tasks

> Goal 状态：已完成。
>
> 每轮只执行第一个未完成任务。每个任务完成后必须记录“做了什么、验证结果、剩余风险、下一步”；有代码修改时先验证、提交，再标记完成。

## Task 1：建立完整基线与边界清单

- [x] 状态：已完成
- 内容：读取仓库说明、Git 状态/diff/log、当前分支与远端、相关功能提交历史、构建测试入口、Docker/腾讯云配置；列出必须删除与必须保留的文件/符号。
- 独立验证：基线清单能解释当前所有工作树修改，且明确普通 gateway 调度、OpenAI 调度、401 Team 不属于删除范围。
- 做了什么：
  - 当前分支为 `main`，HEAD `787d6c87`，与 `origin/main` 一致；当前仅配置 `origin=https://github.com/bayma888/sub2api-bmai.git`。目标功能已经进入共享历史，后续只做前向删除提交，不 reset/rebase/force push。
  - 当前 15 个已跟踪修改全部属于已经开始但尚未闭环的清理：后端 Wire/handler/repository/service 断注入，admin 路由断开，gateway/setting 策略回退；前端 API、菜单、路由、SettingsView 断入口。未跟踪目录仅有 `goal-1/`；没有来源不明修改。现有 `stash@{0}: wip-before-removing-new-scheduling` 及更早升级 stash 保留不动。
  - 确认待下线功能主要来自 `57ca70fa`、`2ba4245a`、`b58a3f17`、`1f09607b`、`5c578797`、`d9a13fd1`、`d92910c9`：优先账号、按分组账号分配、凭据池、独立调度后台、调度日志、上游倍率采集/绑定/健康信号。
  - 必须删除的后端独立文件/测试：`backend/internal/handler/admin/{scheduling_handler.go,scheduling_dto_helpers.go,upstream_rate_handler.go}`、`backend/internal/repository/upstream_rate_repo.go`、`backend/internal/service/{gateway_scheduling_settings.go,scheduling_log_service.go,scheduling_log_service_test.go,setting_service_gateway_scheduling_test.go,gateway_legacy_fallback_scheduling_test.go,upstream_rate_collector.go,upstream_rate_collector_test.go,upstream_rate_runner.go,upstream_rate_service.go,upstream_rate_service_test.go,upstream_rate_types.go}`。
  - 必须删除的前端独立文件/测试：`frontend/src/views/admin/SchedulingView.vue`、`frontend/src/api/admin/{scheduling.ts,upstreamRates.ts}`、`frontend/src/api/__tests__/{settings.gatewayScheduling.spec.ts,upstreamRates.spec.ts}`。
  - 必须从既有文件删除：`/admin/scheduling` 与 `/admin/upstream-rates` 路由/handler/DI；`GatewaySchedulingSettings` DTO 与 `SystemSettings.gateway_scheduling`；全部 `SettingKeyGatewayScheduling*` DB 设置键及缓存/读写；`PreferredAccountID`/`PreferredAccountByGroupID`、credential strategy/fallback、外部 upstream-rate signal、调度日志记录；对应菜单/路由/API 类型/设置表单/专属文本与测试。
  - `docs/SCHEDULER_OPTIMIZATION_NOTES.md` 只清理上述新增功能段落，保留普通调度和 401 Team；`progress.md` 保留历史只追加。`backend/migrations/159_add_upstream_rate_sources.sql` 已进入共享历史，默认保留为历史惰性迁移，不新增破坏性 DROP，应用层停止引用并在最终记录残留表处置建议。
  - 必须保留普通网关调度基点 `35349fc2`：`GatewaySchedulingConfig`、weighted P2C/legacy LRU、priority/load factor、selection debt、queue penalty、sticky、runtime error/latency、slow start、active probe、quota risk、账号自身 `BillingRateMultiplier` 软成本、soonest-reset、负载快照/outbox/DB fallback 及测试。普通配置仅删除新增的 preferred、external upstream-rate、credential 字段；`ScoreWeights.RateMultiplier` 属于原调度，必须保留。
  - 必须保留 OpenAI 高级调度基点 `f26ca566`：`openai_advanced_scheduler_enabled`、OpenAI WS/HTTP scheduler 配置、API/UI/测试；必须保留 401 Team 基点 `bed4513f`：`openai_team_401_retryable` credential、admin API/UI、401 failover 与中英文文本/测试。
  - 保留其它普通“调度”文本变量，例如 API Key 分组调度、账号 schedulable、529/429 暂停、client affinity、OpenAI experimental scheduler，禁止按关键词批量删除。
- 验证结果：
  - `git status -sb` 显示 `main...origin/main`，全部工作树修改均已逐项归类，无未知业务修改。
  - 本机工具链：Go `1.26.0`、Node `22.18.0`、pnpm `10.28.0`、Docker client `29.5.3`。后端验证入口为 `make -C backend generate/test-unit/test-integration/build` 与最终 `go test ./...`；前端入口为 `make test-frontend`、全量 Vitest 和 `pnpm --dir frontend run build`。
  - Docker 入口为 `deploy/build_image.sh`/根 `Dockerfile`；腾讯云历史目标为 `ccr.ccs.tencentyun.com/apophis-chat/sub2api`，上一已验证回滚镜像 `v0.1.147` digest 为 `sha256:d84d13da308b2157f2e761ea11e63a7497fd88ae773421d68cbf82be0032b82e`。
- 剩余风险：
  - 当前半清理状态存在明确编译断点：`buildLegacyLRUSelectionOrder` 调用已恢复为两参数，但函数仍残留 `preferredAccountID` 第三参数及 preferred helpers，Task 2/4 必须修复。
  - 本机 Go `1.26.0` 低于仓库/CI 声明的 `1.26.5`，执行验证时需确认自动 toolchain 是否正常；Node/pnpm 也与 CI 版本不同。
  - 历史 migration 159 对已部署数据库可能已经执行，不能用破坏性 DROP 冒险回退。
- 下一步：执行 Task 2，完成后端独立调度与上游倍率运行入口断开并做残留搜索。

## Task 2：断开后端独立调度与上游倍率运行入口

- [x] 状态：已完成
- 内容：完成 route、handler 聚合、Wire、repository/service provider、runner/server 启动、settings runtime override 的断引用。
- 独立验证：残留搜索不再发现活动路由或 DI 构造链；普通服务启动依赖仍完整。
- 做了什么：
  - 从 admin 路由移除 `registerSchedulingRoutes`，彻底断开 `/admin/scheduling` 及其 `/upstream-rates` 子路由。
  - 从 `AdminHandlers`、`ProvideAdminHandlers` 和 handler provider set 移除 `SchedulingHandler`、`UpstreamRateHandler` 注入。
  - 从 repository/service provider set 移除 `NewUpstreamRateRepository`、`NewUpstreamRateService`、`ProvideUpstreamRateRunner`，生成的 `wire_gen.go` 同步断开实例化链。
  - 普通 gateway 与定时探测 runner 改为只读取文件配置 `cfg.Gateway.Scheduling`，移除 `SettingService.GetGatewaySchedulingConfig` 的运行时覆盖路径。
  - 从定时探测 runner 删除已无用途的 `SettingService` 字段和构造参数，同时保留 `GatewayService` 慢启动标记、账号暂停/恢复及普通 ActiveProbe 能力。
  - 从 `SettingService` 主结构移除独立调度配置缓存字段；具体 DB 设置实现文件留待 Task 4 连同 DTO/键/测试统一删除。
- 验证结果：
  - 静态搜索 `admin.NewSchedulingHandler|admin.NewUpstreamRateHandler|repository.NewUpstreamRateRepository|service.NewUpstreamRateService|service.ProvideUpstreamRateRunner|registerSchedulingRoutes|h.Admin.Scheduling|h.Admin.UpstreamRate`：无命中。
  - handler/repository/service `wire.go` 搜索独立调度与上游倍率 provider：无命中。
  - 定时探测 runner 与普通 gateway 热路径搜索 `GetGatewaySchedulingConfig`、`schedulingConfig(ctx...)`、`SettingService`：无命中。
  - `git diff --check` 通过；仅有仓库既存的 `progress.md` LF/CRLF 提示，无空白错误。
  - 相关 Go 测试已执行但包编译被 Task 4 已知残留阻断；改动前后错误集合一致：legacy LRU preferred 参数、credential-aware helper、独立调度缓存实现和相关测试符号未完成删除。本任务没有引入新的路由/DI/构造错误。
- 剩余风险：
  - `scheduling_handler.go`、`upstream_rate_handler.go`、upstream service/repository/runner 与 `gateway_scheduling_settings.go` 仍是孤立实现文件，Task 4 必须物理删除，否则 service 包继续无法编译。
  - `wire_gen.go` 当前为手工同步结果；删除阶段完成后需运行正式 Wire 生成并核对零差异。
- 下一步：执行 Task 3，大型全面检查 Task 1-2 的边界、调用链、编译错误和数据安全后再进入后端实现删除。

## Task 3：大型全面检查-debug循环（一）

- [x] 状态：已完成
- 内容：全面复核 Task 1-2 的需求边界、后端调用链、编译错误、数据/安全影响和回滚点，发现问题立即修复并重跑。
- 独立验证：相关后端包至少完成编译级验证；检查结果写入本文件。
- 做了什么：
  - 并行完成两路只读审查：一路追踪 scheduling/upstream-rate 路由、DI、启动链、热路径和 migration 159；一路核对普通 gateway 调度、OpenAI 高级调度、401 Team、分组隔离和 WebSocket 错误分类的保留证据。
  - 修复 `buildLegacyLRUSelectionOrder` 半清理签名：恢复双参数 legacy LRU，并删除优先账号选择 helper，保留 priority → load → LRU 原有行为。
  - 将普通 gateway legacy fallback 从已删除的 credential-aware helper 恢复为 `buildLegacyLRUSelectionOrder`。
  - 删除已无活动入口且直接导致生产编译失败的 `scheduling_handler.go` 和 `gateway_scheduling_settings.go`；其 DTO、键和专属测试留待 Task 4 成组删除。
  - 确认 `GatewayService.settingService` 仍服务于 OAuth metadata、beta policy、请求转发、cache rewrite 等普通能力，禁止随调度 DB override 一并删除。
  - 确认定时探测 runner 的计划仓储、测试执行、限流恢复、账号暂停、慢启动标记和配置依赖均完整。
  - 数据安全结论：保留共享历史中的 `159_add_upstream_rate_sources.sql`，不修改 checksum、不新增未经授权的 DROP；应用层停止使用，避免新旧数据库迁移状态分叉和加密 token 数据销毁。
- 验证结果：
  - `go build ./internal/service ./internal/handler ./internal/server/routes ./cmd/server`：通过，后端生产代码完成编译级验证。
  - `go test ./internal/service ./internal/handler ./internal/server/routes ./cmd/server -run TestDoesNotExist -count=1`：handler、routes、cmd/server 通过；service 测试包仅因 Task 4 待删旧测试引用 `schedulingConfigForGroup`、`buildCredentialAwareSelectionOrder`、`Get/UpdateGatewaySchedulingConfig` 而失败。
  - `git diff --check`：通过。
  - 保留证据已核对：普通配置 `GatewaySchedulingConfig`、weighted P2C/legacy LRU、sticky、load/queue/debt、ActiveProbe、SlowStart、账号自身 `BillingRateMultiplier`；全部 `openai_advanced_scheduler_*`；全部 `openai_team_401_retryable` API/UI/i18n/tests；`allow_ungrouped_key_scheduling`；精确业务错误字符串 `upstream_rate_limited`。
  - migration runner 使用完整文件名作为 `schema_migrations.filename` 主键并校验 checksum；直接删除/修改已发布 migration 会造成 schema 分叉或 checksum mismatch，因此保持历史文件不变。
- 剩余风险：
  - service 测试包尚未编译通过，Task 4 必须删除/改写只验证 preferred account、credential strategy、上游倍率信号和 DB scheduling override 的测试。
  - `scheduling_dto_helpers.go`、DTO、settings keys、调度日志热路径及全部 upstream-rate 文件仍是孤立残留，必须成组删除。
  - 普通 gateway 当前主 Layer 2 仍沿用历史的 priority → load → LRU；weighted P2C helper 主要由 OpenAI scheduler 使用。该语义在本轮清理前已存在，不属于本轮回归，禁止为“顺手修复”扩大范围。
  - migration 159 与 `159_batch_image_foundation.sql` 数字前缀重复，但 runner 按完整文件名排序与登记，当前不会主键冲突；后续新 migration 必须使用唯一新编号。
- 下一步：执行 Task 4，成组删除后端账号分配、凭据池、上游倍率、调度日志、DB settings DTO/键/测试残留，并把 service 测试包恢复到可编译状态。

## Task 4：删除后端账号分配、凭据池和上游倍率实现残留

- [x] 状态：已完成
- 内容：删除优先账号、分组账号指定、credential strategy/fallback、upstream-rate 采集/绑定/信号/日志字段及仅服务于它们的 DTO、配置键、测试和文件；保留原有 gateway 调度评分与 OpenAI 错误分类。
- 独立验证：目标符号残留搜索只允许明确列出的兼容/业务无关命中；Go 编译无未定义符号。
- 做了什么：删除上游倍率 handler/repository/service/runner、独立调度日志、DB scheduling override、优先账号和 credential-aware 选择逻辑及专属测试；保留普通 gateway、ActiveProbe、SlowStart、`RateMultiplier`、OpenAI DTO/高级调度和 401 Team。
- 验证结果：后端全包编译触达和相关调度/倍率/401 Team 定向测试通过；共享历史 migration 159 保持不变且运行时无引用。
- 剩余风险：数据库中可能保留闲置历史表，只能通过后续获授权的前向 migration 处置，当前不构成运行时依赖。
- 下一步：Task 5，清理前端入口和文本变量。

## Task 5：删除前端页面、API、路由、菜单与文本变量残留

- [x] 状态：已完成
- 内容：删除 Scheduling 页面、upstream-rates/scheduling API、路由、侧栏、设置类型和表单默认值；清理只属于已删功能的 i18n key/locale 文件，保留其他导航和设置文本。
- 独立验证：前端搜索无活动引用；所有现存 `t(...)` 关键路径有对应文本变量。
- 做了什么：删除独立调度页面、API、路由、侧栏入口、settings 类型/默认值及专属测试和文案。
- 验证结果：`nav.scheduling`/`admin.scheduling` 仅在历史 `progress.md` 中出现，活动代码无命中；Vue/TS 类型检查和 Vite 构建通过。
- 剩余风险：无已知前端死入口；保留的普通“调度”文本均属于其他有效功能。
- 下一步：Task 6，执行删除阶段综合检查。

## Task 6：大型全面检查-debug循环（二）

- [x] 状态：已完成
- 内容：全面检查删除完整性、前后端接口一致性、UI/UX、i18n、权限、安全、数据库迁移处理和回滚方案；修复发现的问题。
- 独立验证：前后端目标包/typecheck 通过，残留清单为零或有明确合理例外。
- 做了什么：复核路由、DI、设置合同、迁移、普通调度和 OpenAI 边界；修复 legacy LRU 半清理签名与 fallback 调用。
- 验证结果：生产代码构建、目标测试、残留扫描和 `git diff --check` 通过；migration 159 被确认为必须保留的共享历史。
- 剩余风险：仅历史 migration/进度文档保留关键词，不存在活动功能入口。
- 下一步：Task 7，完成后端验证。

## Task 7：后端完整验证与修复

- [x] 状态：已完成
- 内容：运行相关单测、Go 编译、可承受范围内的全量测试/lint；逐个修复删除造成的回归。
- 独立验证：记录所有命令和结果；不得把未执行项写成通过。
- 做了什么：完成后端全包编译触达及普通调度、账号倍率、401 Team、配置包定向回归。
- 验证结果：删除提交前后端编译与目标测试通过；合并后又执行 `GOCACHE="$HOME/.cache/go-build" go test ./...` 并全量通过。
- 剩余风险：无已知后端删除回归。
- 下一步：Task 8，完成前端验证。

## Task 8：前端完整验证、构建与文本检查

- [x] 状态：已完成
- 内容：运行 TypeScript/Vue 检查、lint、生产构建和 i18n 残留检查；必要时浏览器验证管理侧栏、设置页和普通功能页面。
- 独立验证：前端构建成功，活跃页面无缺失翻译键、死路由或未定义变量。
- 做了什么：执行 Vue/TS 类型检查、Vite 生产构建、目标符号/i18n 搜索。
- 验证结果：类型检查和构建通过；仅保留既有 Browserslist、dynamic import 和 chunk size 警告。
- 剩余风险：组合 `npm run build` 在本机无诊断退出，已用相同本地二进制分段完成等价验证；Docker 发布构建将再次覆盖完整链路。
- 下一步：Task 9，完成删除阶段最终审查。

## Task 9：大型全面检查-debug循环（三）

- [x] 状态：已完成
- 内容：从需求偏离、代码、类型、构建、测试、UI/UX、安全、数据一致性、权限、错误处理、文档和回滚角度审查删除阶段；循环修复直到无已知高风险问题。
- 独立验证：形成删除阶段验收记录和剩余风险列表。
- 做了什么：确认删除范围严格限定于独立扩展，逐项验证普通 gateway、OpenAI 高级调度、401 Team、ActiveProbe、SlowStart 与 DTO 未缺失。
- 验证结果：残留搜索、后端编译/测试、前端类型/构建、文档与 migration 检查均通过。
- 剩余风险：仅闲置历史表及历史进度文字，不影响运行时。
- 下一步：Task 10，整理并提交删除结果。

## Task 10：整理生成代码、文档与删除阶段原子提交

- [x] 状态：已完成
- 内容：更新 Wire/生成产物、`progress.md` 和必要文档；检查 diff，只暂存本目标文件，创建删除阶段原子提交。
- 独立验证：提交后工作树中不存在误提交的密钥、日志、构建产物或无关用户修改。
- 做了什么：同步 Wire/文档/进度记录并创建删除提交 `c0aa6719 refactor(scheduling): 移除独立调度扩展`。
- 验证结果：提交后工作树干净，未提交密钥、日志或构建产物。
- 剩余风险：删除可通过 `git revert c0aa6719` 回滚。
- 下一步：Task 11，创建合并前备份分支。

## Task 11：创建并验证合并前备份分支

- [x] 状态：已完成
- 内容：基于清理提交创建唯一 `backup/<当前分支>-pre-v0.1.149-<时间戳>` 分支；远端可用时正常推送备份分支。
- 独立验证：本地备份分支指向预期提交；若推送，远端引用可查询。
- 做了什么：创建本地备份分支 `backup/pre-v0.1.149-20260710-c0aa6719`，指向删除提交 `c0aa6719`。
- 验证结果：本地分支可解析且与合并前 HEAD 一致。
- 剩余风险：备份分支尚未推送远端；将在 Task 17 正常推送代码时一并推送，不影响本地恢复能力。
- 下一步：Task 12，执行合并前门禁。

## Task 12：大型全面检查-debug循环（四，合并前门禁）

- [x] 状态：已完成
- 内容：检查备份、工作树、测试状态、分支跟踪、远端、上游来源和回滚命令，确认满足合并门禁。
- 独立验证：备份可恢复，工作树状态明确，无未记录高风险项。
- 做了什么：核验清理提交、工作树、备份分支、上游来源、测试结果和 merge/revert 回滚路径。
- 验证结果：合并前工作树干净，备份可恢复，官方 tag 可获取，未使用破坏性 Git 操作。
- 剩余风险：无合并门禁阻塞项。
- 下一步：Task 13，获取并核验官方 tag。

## Task 13：获取并核验官方 v0.1.149

- [x] 状态：已完成
- 内容：获取官方仓库 tag，核验 `v0.1.149` 指向、发布提交和相对当前分支的变更范围，制定冲突文件迁移清单。
- 独立验证：本地可解析 tag commit，来源为官方仓库，变更清单可复核。
- 做了什么：通过 `git fetch https://github.com/Wei-Shaw/sub2api.git tag v0.1.149` 获取官方标签并审阅相对当前分支的 72 文件变更。
- 验证结果：`v0.1.149` 可解析到提交 `19668b14`；范围包括用户角色/Token 排行、用量页、版本回滚、OpenAI/Grok 修复等。
- 剩余风险：用量排行与本地计费筛选、OpenAI Compact 路径存在潜在语义冲突，已纳入 Task 14/15。
- 下一步：Task 14，执行普通 merge 并迁移本地修改。

## Task 14：合并 v0.1.149 并迁移本地修改

- [x] 状态：已完成
- 内容：使用普通 merge 合并 tag；逐文件解决冲突，保留本地删除结果、OpenAI 高级调度、401 Team 和无关用户功能，同时吸收 1.149 修复。
- 独立验证：无未解决冲突；关键功能 diff 与迁移清单一致。
- 做了什么：非快进合并 `v0.1.149`，解决用量排行冲突，保留本地 `billing_mode`/成本排序并合入 Token 细分排序；迁移 Compact 非流式 keepalive 到当前拆分的标准 OAuth 与透传网关路径。
- 验证结果：无未解决冲突；创建 merge commit `88e11471 merge: 合并上游 v0.1.149 发布版`，提交后工作树干净。
- 剩余风险：代码尚未推送；Docker 发布链路尚未执行。
- 下一步：Task 15，执行合并后综合检查。

## Task 15：大型全面检查-debug循环（五，合并后）

- [x] 状态：已完成
- 内容：全面检查合并偏差、冲突误解、API/DB/权限/安全、前后端类型、文本变量和回滚；发现问题立即修复。
- 独立验证：合并状态干净或只剩明确待验证修改，关键包编译通过。
- 做了什么：发现并修复自动合并遗漏的 Compact keepalive 实现及隐藏的用量排序/筛选语义冲突；再次扫描已删除独立调度入口和 i18n 键。
- 验证结果：后端 `go test ./...` 全量通过；前端 151 个 Vitest 文件、955 个测试全部通过；Vue/TS 检查和 Vite 构建通过；Compact keepalive 与用户排行定向测试通过；`git diff --check` 和冲突检查通过。
- 剩余风险：Vite 仍有历史 Browserslist、dynamic import/chunk size 警告；不阻断发布。Docker 构建尚待 Task 16。
- 下一步：Task 16，执行 Docker 本地发布构建与必要健康验证。

## Task 16：合并后全量构建、测试与 Docker 本地验证

- [x] 状态：已完成
- 内容：运行仓库可用的后端测试/构建、前端检查/构建、生成代码检查、Docker build；可行时启动镜像做健康检查。
- 独立验证：记录命令、退出状态和关键结果，修复到无已知高风险失败。
- 做了什么：
  - 在前序后端全量测试、前端 955 个测试、Vue/TS 检查和 Vite 构建基础上，完成无缓存 Docker 多阶段 release 构建。
  - 修复 `.dockerignore` 未排除根目录及子目录 `.gocache` 的问题，避免约 6.2GB 无关缓存进入构建上下文；提交为 `99d2069d build(docker): 排除本地 Go 缓存构建上下文`。
  - 按用户要求执行 `docker system prune -a -f`，清理未使用镜像、停止容器、网络和构建缓存，未执行 volume prune；首次回收 19.29GB，健康验证后再清理依赖和中间层 1.104GB。
  - 从零构建 `sub2api:0.1.149-74b343b8-local`，最终仅保留 303MB 成品镜像。
- 验证结果：
  - 镜像 ID `sha256:ee73fbf3bb2e7b73363e1e5ef3a3b694800bdb8aa3f5c48197c80cdae0365a31`。
  - `--version` 输出 `Sub2API 0.1.149 (commit: 74b343b8)`；运行用户为 `uid=1000(sub2api)`、`gid=1000(sub2api)`。
  - 隔离 PostgreSQL/Redis 自动初始化、迁移和启动成功；`/health` 返回 `{"status":"ok"}`，Docker health status 为 `healthy`。
  - 最终 `docker system df`：1 个成品镜像、0 容器、0 构建缓存；3 个本轮匿名卷总占用 0B，按不删除数据卷边界保留。
- 剩余风险：本机缺少 buildx，本轮使用仓库现有 `deploy/Dockerfile`/legacy builder；Task 18 发布前需复核生产根 Dockerfile 的 PostgreSQL 客户端层，或使用等价 no-BuildKit 构建方式。legacy builder 弃用提示不影响本次构建结果。
- 下一步：Task 17，复核提交历史，推送 `main` 和合并前备份分支并验证远端引用。

## Task 17：提交合并结果并推送代码

- [x] 状态：已完成
- 内容：复核完整 diff 和提交历史，创建必要提交/完成 merge commit，正常推送当前分支；禁止 force push。
- 独立验证：远端分支 HEAD 与本地预期提交一致，备份分支仍可用。
- 做了什么：
  - 在工作树干净、`main` 相对远端 ahead 27、提交历史与备份指向核验通过后执行正常推送。
  - 推送远端备份分支 `backup/pre-v0.1.149-20260710-c0aa6719`，保留 `c0aa6719` 合并前恢复点。
  - 推送 `main`，包含独立调度删除、v0.1.149 合并、冲突迁移、全量验证和 Docker 上下文修复。
  - 未执行 force push、rebase、reset 或覆盖 checkout。
- 验证结果：
  - 首次推送后 `refs/heads/main` 为 `805ff424a25676983ea19d74305a07426374bd28`，与当时本地 HEAD 一致。
  - `refs/heads/backup/pre-v0.1.149-20260710-c0aa6719` 为 `c0aa6719ea07e4e5e38ed90d5801d4bce8994971`，与本地备份一致。
  - `git status --short --branch` 显示本地与 `origin/main` 同步、工作树干净；本任务记录提交完成后将再次推送并核对最终 HEAD。
- 剩余风险：代码远端已安全落盘；腾讯云生产镜像尚未构建和推送，生产根 Dockerfile/buildx 兼容问题留给 Task 18 处理。
- 下一步：Task 18，执行发布前综合门禁，构建并推送腾讯云镜像，核验远端 digest/manifest。

## Task 18：大型全面检查-debug循环（六）并推送腾讯云镜像

- [x] 状态：已完成
- 内容：先做发布前综合门禁，再按项目惯例构建、标记并推送腾讯云镜像；核验远端 digest/manifest，确保不泄露凭据。
- 独立验证：代码远端、镜像标签和 digest 均可查询；部署回滚目标明确。
- 做了什么：
  - 核验本地/远端 `main=f5cfb1b1`、工作树干净、Docker/磁盘/腾讯云 CCR 可用，读取发布前 `latest` digest 作为回滚点。
  - 使用生产根 Dockerfile 的等价 no-BuildKit 临时副本构建 release 镜像，仅移除 pnpm cache mount，保留 PostgreSQL 18 客户端、resources、release ldflags、非 root 用户和 healthcheck。
  - 推送 `0.1.149-f5cfb1b1-20260710052405`、`v0.1.149` 和 `latest` 三个腾讯云标签。
  - 发布后清理基础镜像、Redis/PostgreSQL 测试依赖、旧本地验证镜像和构建中间层，额外回收 1.171GB。
- 验证结果：
  - 生产镜像 ID/远端 digest：`sha256:1f6fc5ca31023e9d353848f593c79b7d749820247fb331a4ba92b82fcaa5e1a7`。
  - 版本输出为 `Sub2API 0.1.149 (commit: f5cfb1b1, built: 2026-07-10T05:24:05Z)`；运行用户为 `uid=1000(sub2api)`。
  - `psql`、`pg_dump` 均为 PostgreSQL `18.4`，`/app/resources` 存在。
  - 隔离 PostgreSQL/Redis 初始化、迁移和健康测试通过，`/health` 返回 `{"status":"ok"}`，Docker health status 为 `healthy`。
  - 三个远端 manifest 均为 `linux/amd64` 且 digest 完全一致；发布前回滚 digest 为 `sha256:d84d13da308b2157f2e761ea11e63a7497fd88ae773421d68cbf82be0032b82e`。
  - 清理后本地仅保留 166MB 可追溯成品镜像，无容器、无构建缓存，C 盘可用约 95GB。
- 剩余风险：本机 legacy builder 已弃用但本次构建成功；后续建议安装 buildx 恢复根 Dockerfile 原生缓存路径。仅发布 `linux/amd64`，与当前仓库历史腾讯云发布平台一致。
- 下一步：Task 19，执行最终最大 Review，复核代码、远端、镜像、回滚和文档证据后标记 Goal 完成。

## Task 19：最终最大 Review、修缮与 Goal 完成

- [x] 状态：已完成
- 内容：从 C 端体验、代码、依赖、安全、数据一致性、权限、错误处理、测试、构建、文档、Git、远端代码、镜像和回滚进行最终最大审查；修复所有已知高风险问题，更新 `progress.md`，标记 Goal 完成。
- 独立验证：所有验收证据齐全；没有已知高风险问题；仅保留明确披露的客观验证缺口。
- 做了什么：
  - 复审发现余额自动刷新与倍率探测在同一周期内串行执行，慢倍率请求可能阻塞余额任务；已在 `74bcb6d8a` 将两条周期任务并发启动。
  - 复审发现设置页倍率/余额保存按钮共用 `upstreamBillingProbeSaving`，已拆分为独立 loading 状态，并新增前端回归测试。
  - NewAPI Dashboard PAT 已纳入统一脱敏和编辑保留；用户余额请求失败关闭，不回退模型 Token，不记录凭据；钱包累计用量不再伪装固定上限。
  - 已正常推送 `6eb9a817d`、`74bcb6d8a` 到 `origin/main`，未使用强推或改写历史。
  - 基于 `74bcb6d8` 构建并推送腾讯云可追溯镜像与 `latest`，完成隔离数据库/Redis 健康验证和远端摘要核验。
- 验证结果：
  - 后端 `go test -p 1 ./...` 全量通过；新增慢倍率不阻塞余额刷新测试通过。
  - 前端 `pnpm test:run` 通过，199 个测试文件、1414 项测试；设置页定向 27 项测试通过。
  - `pnpm run build` 类型检查和生产构建通过；`gofmt`、ESLint、`git diff --check` 通过。
  - 镜像版本为 `0.1.162`、commit `74bcb6d8`，运行用户为 `uid=1000`，PostgreSQL 客户端为 18.4，隔离 `/health` 返回 `{"status":"ok"}`。
  - 腾讯云标签 `0.1.162-74bcb6d8-20260722152041` 与 `latest` 摘要一致：`sha256:f73245417272a2a7f50c0e192447b32f746690d65ad31597fc72ca540528a905`。
- 剩余风险：未连接真实 NewAPI 多版本实例，部署后仍建议人工核对一次余额；Windows 环境缺少 GCC，Go race detector 未执行；legacy builder 和约 3GB 构建上下文仅影响构建效率，不构成运行时高风险。
- 下一步：Goal 完成。部署异常时回切发布前 `latest` 摘要 `sha256:1e1bfccd35b9ed035d4ff0f0079fcc0eb8bcd7db481cae9965312d88d1a494d2`，代码使用正常 `git revert` 回滚。

## Task 20：余额与倍率对齐回归测试（TDD）

- [x] 状态：已完成
- 内容：先补充余额组件、账号列表全局开关、余额失败快照与 due 查询的回归测试，并确认测试在实现前按预期失败。
- 独立验证：失败原因必须精确对应缺失的下一次刷新、账号/全局状态、失败保留和安全 due 排序能力。
- 做了什么：新增余额下一次刷新、账号/全局开关、失败快照保留、到期查询和前端展示回归测试，覆盖与倍率列需要一比一对齐的生命周期与交互。
- 验证结果：实现前失败点对应缺失能力；完成 Task 21/22 后相关测试转绿，并进入前后端全量测试集合。
- 剩余风险：真实 NewAPI 多版本实例仍需部署后人工核对一次余额换算。
- 下一步：Task 21，实现余额后端生命周期与安全 due 查询。

## Task 21：实现余额后端探测生命周期和 due 查询对齐

- [x] 状态：已完成
- 内容：对齐临时失败保留、确定性失败失效、next_probe_at 和安全到期排序。
- 独立验证：后端定向测试全部通过。
- 做了什么：实现余额探测的临时失败快照保留、确定性失败失效、下一次探测时间和安全到期排序，并保持凭据失败关闭与不回退 Token 额度。
- 验证结果：后端定向测试与 `go test -p 1 ./...` 全量通过；余额与倍率周期任务并发回归测试通过。
- 剩余风险：Windows 缺少 GCC，Go race detector 未执行；普通并发测试已覆盖本轮竞争路径。
- 下一步：Task 22，实现余额前端一比一展示。

## Task 22：实现余额前端一比一展示

- [x] 状态：已完成
- 内容：复刻倍率列的下一次刷新、账号开关、全局开关、失败/过期详情和视觉结构，并保留余额专属数值语义。
- 独立验证：组件与 AccountsView 定向测试全部通过。
- 做了什么：补齐余额下一次刷新、账号开关、全局开关、失败/过期详情和独立保存 loading；余额数值保持一位小数且不伪装固定上限。版本文件同步为 `0.1.163`。
- 验证结果：前端 199 个测试文件、1414 项测试通过；Vue/TypeScript 检查、Vite 生产构建和相关 ESLint 通过。实现提交 `66da7bfa0`、版本提交 `126984643` 已推送到 `origin/main`。
- 剩余风险：仅保留既有 Browserslist、动态/静态 import 和大 chunk 警告，不阻断发布。
- 下一步：Task 23，执行发布前大型全面检查。

## Task 23：大型全面检查-debug循环（七）

- [x] 状态：已完成
- 内容：检查需求偏离、自动刷新、SQL 安全、UI/UX、i18n、类型、测试和数据一致性并修复。
- 独立验证：前后端全量测试、lint、构建通过。
- 做了什么：复核余额/倍率生命周期、周期并发、凭据脱敏、SQL 到期排序、前端开关与保存状态、i18n、版本和回滚路径；未发现需要扩大范围的高风险问题。
- 验证结果：后端全量测试、前端全量测试、类型检查、生产构建、gofmt、ESLint 与 `git diff --check` 均通过；本地与远端 `main` 一致。
- 剩余风险：真实 NewAPI 实例人工核对与 Windows race detector 仍是客观验证缺口。
- 下一步：Task 24，发布 `0.1.163` 腾讯云镜像。

## Task 24：修正 0.1.163 版本并发布腾讯云镜像

- [x] 状态：已完成
- 内容：修正 VERSION，提交并推送代码，构建并推送 0.1.163 可追溯/v0.1.163/latest 镜像，核验版本和远端摘要。
- 独立验证：镜像 `--version`、隔离健康检查和远端 digest 一致。
- 做了什么：确认 `backend/cmd/server/VERSION=0.1.163` 且 `origin/main=126984643`；因本机 buildx 缺失，使用生产根 Dockerfile 的等价 legacy-builder 临时副本构建 `linux/amd64` 镜像，未改仓库生产 Dockerfile。推送可追溯标签 `0.1.163-12698464-20260723040528`，并同步 `v0.1.163`、`latest`。
- 验证结果：镜像版本输出 `Sub2API 0.1.163 (commit: 12698464, built: 2026-07-23T04:05:28Z)`；默认入口运行用户为 `uid=1000(sub2api)`；`psql`/`pg_dump` 为 18.4；资源和 healthcheck 存在。隔离 PostgreSQL/Redis 自动初始化、迁移和启动成功，`/health` 返回 `{"status":"ok"}`。三个远端标签均为 `linux/amd64`，摘要一致为 `sha256:c489fb73d1fc1893ccf52c94bf3bc1a2edf2fb6787512fe012be27ca4387ee3a`。
- 剩余风险：本机仍缺少 buildx，legacy builder 构建上下文约 3.0GB，仅影响构建效率；发布前 `latest` 回滚摘要为 `sha256:f73245417272a2a7f50c0e192447b32f746690d65ad31597fc72ca540528a905`。
- 下一步：Task 25，执行最终最大 Review 与记录闭环。

## Task 25：最终最大 Review 与记录闭环

- [x] 状态：已完成
- 内容：复核代码、服务器升级说明、Git、镜像、回滚、文档和剩余风险，更新进度与任务状态。
- 独立验证：无已知高风险问题，所有证据可复核。
- 做了什么：完成后端安全/数据一致性与前端交互/脱敏独立复核；修复 NewAPI Dashboard PAT 可能继承通用 HTTP 许可的高风险问题，强制携带 PAT 的控制面请求使用 HTTPS 并拒绝 URL userinfo；提交并推送 `3bcb1486a`。基于该提交重建并推送 `0.1.163-3bcb1486-20260723051001`、`v0.1.163`、`latest`，补充服务器升级与回滚说明，修正 `progress.md` 的未来日期标题。
- 验证结果：安全测试先红后绿；目标测试、服务层全量测试、服务层 `go vet`、后端 `go test ./...` 通过；前端 199 个测试文件/1417 项测试、类型检查、生产构建和 ESLint 通过；独立复核确认无剩余高风险。隔离 PostgreSQL/Redis 环境启动 healthy，`/health` 返回 `{"status":"ok"}`；镜像版本为 `0.1.163 / 3bcb1486`，默认入口用户 `uid=1000`，PostgreSQL 客户端 18.4。腾讯云三个远端标签均为 `linux/amd64`，摘要一致为 `sha256:1a1249c87871419b03ae2348481d7e56931d60e09342816f57f3daea36786eb2`；隔离容器、网络和命名卷已清理。
- 剩余风险：未连接真实 NewAPI 多版本实例；Windows 缺少 GCC，Go race detector 未执行；本机缺少 buildx，legacy builder 构建上下文约 3.0GB。非阻断中低风险已记录到 `progress.md`，包括少见瞬态失败分类、批量凭据轮换快照、设置并发保存、长周期锁续租和 Tooltip 可访问性增强；无已知高风险或发布阻断问题。
- 下一步：Goal 完成。服务器优先部署不可变标签 `0.1.163-3bcb1486-20260723051001`；若异常，回切 `0.1.162-74bcb6d8-20260722152041`（摘要 `sha256:f73245417272a2a7f50c0e192447b32f746690d65ad31597fc72ca540528a905`）。
