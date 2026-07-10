# Goal 1 Tasks

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

- [ ] 状态：未完成
- 内容：删除优先账号、分组账号指定、credential strategy/fallback、upstream-rate 采集/绑定/信号/日志字段及仅服务于它们的 DTO、配置键、测试和文件；保留原有 gateway 调度评分与 OpenAI 错误分类。
- 独立验证：目标符号残留搜索只允许明确列出的兼容/业务无关命中；Go 编译无未定义符号。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 5：删除前端页面、API、路由、菜单与文本变量残留

- [ ] 状态：未完成
- 内容：删除 Scheduling 页面、upstream-rates/scheduling API、路由、侧栏、设置类型和表单默认值；清理只属于已删功能的 i18n key/locale 文件，保留其他导航和设置文本。
- 独立验证：前端搜索无活动引用；所有现存 `t(...)` 关键路径有对应文本变量。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 6：大型全面检查-debug循环（二）

- [ ] 状态：未完成
- 内容：全面检查删除完整性、前后端接口一致性、UI/UX、i18n、权限、安全、数据库迁移处理和回滚方案；修复发现的问题。
- 独立验证：前后端目标包/typecheck 通过，残留清单为零或有明确合理例外。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 7：后端完整验证与修复

- [ ] 状态：未完成
- 内容：运行相关单测、Go 编译、可承受范围内的全量测试/lint；逐个修复删除造成的回归。
- 独立验证：记录所有命令和结果；不得把未执行项写成通过。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 8：前端完整验证、构建与文本检查

- [ ] 状态：未完成
- 内容：运行 TypeScript/Vue 检查、lint、生产构建和 i18n 残留检查；必要时浏览器验证管理侧栏、设置页和普通功能页面。
- 独立验证：前端构建成功，活跃页面无缺失翻译键、死路由或未定义变量。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 9：大型全面检查-debug循环（三）

- [ ] 状态：未完成
- 内容：从需求偏离、代码、类型、构建、测试、UI/UX、安全、数据一致性、权限、错误处理、文档和回滚角度审查删除阶段；循环修复直到无已知高风险问题。
- 独立验证：形成删除阶段验收记录和剩余风险列表。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 10：整理生成代码、文档与删除阶段原子提交

- [ ] 状态：未完成
- 内容：更新 Wire/生成产物、`progress.md` 和必要文档；检查 diff，只暂存本目标文件，创建删除阶段原子提交。
- 独立验证：提交后工作树中不存在误提交的密钥、日志、构建产物或无关用户修改。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 11：创建并验证合并前备份分支

- [ ] 状态：未完成
- 内容：基于清理提交创建唯一 `backup/<当前分支>-pre-v0.1.149-<时间戳>` 分支；远端可用时正常推送备份分支。
- 独立验证：本地备份分支指向预期提交；若推送，远端引用可查询。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 12：大型全面检查-debug循环（四，合并前门禁）

- [ ] 状态：未完成
- 内容：检查备份、工作树、测试状态、分支跟踪、远端、上游来源和回滚命令，确认满足合并门禁。
- 独立验证：备份可恢复，工作树状态明确，无未记录高风险项。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 13：获取并核验官方 v0.1.149

- [ ] 状态：未完成
- 内容：获取官方仓库 tag，核验 `v0.1.149` 指向、发布提交和相对当前分支的变更范围，制定冲突文件迁移清单。
- 独立验证：本地可解析 tag commit，来源为官方仓库，变更清单可复核。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 14：合并 v0.1.149 并迁移本地修改

- [ ] 状态：未完成
- 内容：使用普通 merge 合并 tag；逐文件解决冲突，保留本地删除结果、OpenAI 高级调度、401 Team 和无关用户功能，同时吸收 1.149 修复。
- 独立验证：无未解决冲突；关键功能 diff 与迁移清单一致。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 15：大型全面检查-debug循环（五，合并后）

- [ ] 状态：未完成
- 内容：全面检查合并偏差、冲突误解、API/DB/权限/安全、前后端类型、文本变量和回滚；发现问题立即修复。
- 独立验证：合并状态干净或只剩明确待验证修改，关键包编译通过。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 16：合并后全量构建、测试与 Docker 本地验证

- [ ] 状态：未完成
- 内容：运行仓库可用的后端测试/构建、前端检查/构建、生成代码检查、Docker build；可行时启动镜像做健康检查。
- 独立验证：记录命令、退出状态和关键结果，修复到无已知高风险失败。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 17：提交合并结果并推送代码

- [ ] 状态：未完成
- 内容：复核完整 diff 和提交历史，创建必要提交/完成 merge commit，正常推送当前分支；禁止 force push。
- 独立验证：远端分支 HEAD 与本地预期提交一致，备份分支仍可用。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 18：大型全面检查-debug循环（六）并推送腾讯云镜像

- [ ] 状态：未完成
- 内容：先做发布前综合门禁，再按项目惯例构建、标记并推送腾讯云镜像；核验远端 digest/manifest，确保不泄露凭据。
- 独立验证：代码远端、镜像标签和 digest 均可查询；部署回滚目标明确。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：

## Task 19：最终最大 Review、修缮与 Goal 完成

- [ ] 状态：未完成
- 内容：从 C 端体验、代码、依赖、安全、数据一致性、权限、错误处理、测试、构建、文档、Git、远端代码、镜像和回滚进行最终最大审查；修复所有已知高风险问题，更新 `progress.md`，标记 Goal 完成。
- 独立验证：所有验收证据齐全；没有已知高风险问题；仅保留明确披露的客观验证缺口。
- 做了什么：
- 验证结果：
- 剩余风险：
- 下一步：
