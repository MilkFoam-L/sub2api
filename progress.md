## 2026-06-26 - Task: 合并 v0.1.139 并发布腾讯云 Docker 镜像
### What was done
- 将本地 main 合并到上游 release/tag v0.1.139 内容，并保持运行时版本为 0.1.139。
- 解决合并后的前端 TypeScript 冲突问题，已提交修复提交 `34d011c5 fix: resolve v0.1.139 frontend merge issues`；合并主体提交为 `4a0b4036 chore: update branch to v0.1.139`。
- 构建并推送腾讯云 CCR 镜像：`ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.139-34d011c5-20260626223230` 与 `latest`。
- 本地记录当前发布 tag 到 `.docker-last-tag`，该文件保持未提交。

### Testing
- 后端关键包编译触达通过：`GOCACHE="/c/Users/MilkFoam/AppData/Local/go-build" go test ./internal/config ./internal/handler ./internal/handler/admin ./internal/handler/dto ./internal/server ./internal/service -run 'TestDoesNotExist'`。
- Docker 构建通过：前端 `pnpm run build`、后端 Go release build、最终镜像构建均成功。
- 腾讯云 CCR 推送完成：版本 tag 与 `latest` 均推送成功。
- 远端 manifest digest 已核验一致：`sha256:f840511301c38c8487189c9dfd407cd50692e7ae962c8031c2dadd7ba0a91456`。

### Notes
- `backend/cmd/server/VERSION`：同步运行时版本为 `0.1.139`。
- `frontend/src/components/account/ModelWhitelistSelector.vue`：修复 v0.1.139 合并后重复声明与缺失 `probeScopeKey` 的 TypeScript 错误。
- `frontend/src/views/admin/SettingsView.vue`：修复 Codex blacklist/whitelist/fingerprint 初始化逻辑的 TypeScript 参数错误。
- `.docker-last-tag`：记录本次镜像 tag `0.1.139-34d011c5-20260626223230`，仅作本地发布记录，不提交。
- 回滚方式：代码层面可回滚到提交 `9927a8c4` 或对 `4a0b4036`、`34d011c5` 执行 `git revert`；镜像层面可将部署端 tag 回切到上一个已知可用镜像。

## 2026-06-28 - Task: 记录账号调度优化启发并联网调研调度策略
### What was done
- 记录了 S2A-Manager 对当前账号调度的启发，明确其定位是外部治理层而非请求级调度器替代品。
- 联网调研 Envoy、LiteLLM、OpenRouter、Netflix concurrency-limits、Kubernetes Scheduler 等调度/负载均衡资料，并整理成当前项目可落地的调度优化方向。
- 形成保留现有 Weighted P2C 与 sticky 能力、补充健康摘除、慢启动、成本/延迟/错误率评分、分组维度策略和自适应并发的后续优化建议。

### Testing
- 本轮仅新增/更新调度优化文档，未修改运行时代码；已通过 `Read` 复核文档内容结构和中文记录格式。
- 未运行构建或单元测试，原因是本轮没有代码路径变更。

### Notes
- `docs/SCHEDULER_OPTIMIZATION_NOTES.md`：新增账号调度优化启发、联网调度器资料总结、当前项目差距判断和后续优先级建议。
- `.gitignore`：为调度优化文档增加精确例外，避免新文档被 `docs/*` 忽略规则覆盖。
- `progress.md`：追加本轮文档记录、验证说明和回滚方式。
- 回滚方式：删除 `docs/SCHEDULER_OPTIMIZATION_NOTES.md`，还原 `.gitignore` 中的 `!docs/SCHEDULER_OPTIMIZATION_NOTES.md` 例外，并从 `progress.md` 末尾移除本轮 `2026-06-28` 记录即可恢复到本轮文档变更前状态。

## 2026-06-28 - Task: 实现账号多目标调度第一阶段优化
### What was done
- 将普通网关账号调度从单纯负载/等待/调度债务评分，扩展为健康、成本、延迟、额度和负载共同参与的多目标评分。
- 新增普通网关账号运行时统计，记录错误率 EWMA、延迟 EWMA、连续失败次数和最近成功/失败时间。
- 将运行时统计接入普通网关候选账号评分，使高错误率、高延迟、高倍率、额度接近耗尽的账号在同优先级层内获得更高 cost。
- 在 Messages、Gemini 兼容、Chat Completions 和 Responses 转发路径中回报账号运行时成功/失败和耗时，并在连续上游失败达到阈值时写入临时不可调度状态。
- 新增调度权重配置和示例配置，保持 `priority` 优先分层、`legacy_lru` 旧算法路径和现有硬过滤约束不变。

### Testing
- 红灯验证：新增测试后，`go test` 曾因 `GatewaySchedulingScoreWeights`、`schedulerAccountRuntimeSnapshot`、`runtimeStats` 等尚未实现而失败，确认测试先于实现生效。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/service -run 'TestSchedulerPolicy|TestSchedulerAccountRuntimeStats'`。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/config -run 'TestLoadDefaultConfig|TestLoadSchedulingConfigFromEnv|TestConfigValidation'`。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/handler -run 'Test.*Gateway|Test.*Responses|Test.*Chat'`。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/service ./internal/config ./internal/handler`。

### Notes
- `backend/internal/config/config.go`：新增多目标调度评分权重、延迟基准、额度风险阈值和最大软惩罚配置及校验。
- `backend/internal/config/config_test.go`：补充默认值、环境变量覆盖和非法调度权重配置测试。
- `backend/internal/service/account_scheduler_policy.go`：扩展 Weighted P2C cost，加入错误率、延迟、账号倍率和额度风险软惩罚。
- `backend/internal/service/account_scheduler_policy_test.go`：新增多目标 cost、额度风险、sticky soft escape 等调度行为测试。
- `backend/internal/service/scheduler_runtime_stats.go`：新增账号运行时健康统计。
- `backend/internal/service/scheduler_runtime_stats_test.go`：新增运行时统计 EWMA、冷启动和连续失败测试。
- `backend/internal/service/gateway_service.go`：为账号候选附加运行时快照，并提供请求结果回报和轻量被动摘除能力。
- `backend/internal/handler/gateway_handler.go`：在 Messages 和 Gemini 兼容转发路径回报账号运行时结果。
- `backend/internal/handler/gateway_handler_chat_completions.go`：在 Chat Completions 转发路径回报账号运行时结果。
- `backend/internal/handler/gateway_handler_responses.go`：在 Responses 转发路径回报账号运行时结果。
- `deploy/config.example.yaml`：补充多目标调度配置示例和中英文说明。
- `docs/SCHEDULER_OPTIMIZATION_NOTES.md`：更新为已落地第一阶段，并说明配置、边界和回滚方式。
- `progress.md`：追加本轮实现、验证和回滚说明。
- 回滚方式：将新增 `score_weights.error_rate`、`score_weights.latency`、`score_weights.rate_multiplier`、`score_weights.quota_risk` 设为 `0` 可接近旧调度；代码层面可回退上述调度、统计、handler 回报、配置和文档文件的本轮修改。

## 2026-06-28 - Task: 实现账号主动检测暂停与恢复慢启动
### What was done
- 复用已有账号定时测试计划作为主动探活来源，不新增全账号扫描 worker，避免额外不可控网络探测流量。
- 为定时测试 runner 增加连续失败自动临时暂停：同一计划连续失败达到阈值后写入 `temp_unschedulable_until`，暂停时长随连续失败次数递增并受最大时长封顶。
- 保持暂停期间继续探活：runner 不以账号当前可调度状态作为执行前置条件，已有计划到期仍会继续测试。
- 为定时测试成功恢复接入 slow start：`auto_recover=true` 且实际清理了运行时状态后，标记账号进入慢启动窗口，调度 cost 在窗口内增加逐步衰减的软惩罚。
- 新增主动检测暂停和慢启动配置，并同步示例配置、调度优化文档和依赖注入。

### Testing
- 红灯验证：新增测试后，`go test` 曾因 `ActiveProbe`、`SlowStart`、`HasSlowStart`、`SlowStartUntil`、runner 新依赖等尚未实现而失败，确认测试先于实现生效。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/service -run 'TestScheduledTestRunner|TestSchedulerPolicy|TestSchedulerAccountRuntimeStats'`。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/config -run 'TestLoadDefaultConfig|TestLoadSchedulingConfigFromEnv|TestConfigValidation'`。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./cmd/server -run 'TestDoesNotExist'`。

### Notes
- `backend/internal/config/config.go`：新增 `active_probe` 和 `slow_start` 调度配置、默认值和校验。
- `backend/internal/config/config_test.go`：补充主动检测和慢启动默认值、环境变量覆盖与非法配置测试。
- `backend/internal/service/scheduled_test_runner_service.go`：复用定时测试结果实现连续失败自动暂停、成功恢复慢启动标记，并抽象测试/恢复/暂停/慢启动接口。
- `backend/internal/service/scheduled_test_runner_service_test.go`：新增定时探活未达阈值不暂停、达阈值暂停、成功恢复并标记慢启动测试。
- `backend/internal/service/scheduler_runtime_stats.go`：新增 slow-start 标记和快照字段。
- `backend/internal/service/scheduler_runtime_stats_test.go`：新增 slow-start 运行时统计测试。
- `backend/internal/service/account_scheduler_policy.go`：在运行时统计惩罚中加入 slow-start 逐步衰减惩罚。
- `backend/internal/service/account_scheduler_policy_test.go`：新增 slow-start cost、过期窗口、strict/soft sticky 行为测试。
- `backend/internal/service/gateway_service.go`：实现 `MarkAccountSlowStart`，供定时测试恢复后标记账号慢启动。
- `backend/internal/service/wire.go`：为 scheduled test runner 接入账号暂停仓储和慢启动 marker。
- `backend/cmd/server/wire_gen.go`：同步手工生成的依赖注入调用。
- `deploy/config.example.yaml`：补充主动检测和慢启动示例配置及中英文说明。
- `docs/SCHEDULER_OPTIMIZATION_NOTES.md`：记录第二阶段已落地行为、边界和回滚方式。
- `progress.md`：追加本轮实现、验证和回滚说明。
- 回滚方式：配置层可将 `gateway.scheduling.active_probe.auto_pause_enabled=false`、`gateway.scheduling.slow_start.enabled=false`；代码层面可回退上述 runner、runtime stats、调度策略、GatewayService、wire、配置、测试和文档改动。

## 2026-06-28 - Task: 审查调度器优化代码并修复递增冷却问题
### What was done
- 对本轮 Go 改动执行自动检查和人工审查，重点检查多目标调度、主动探活暂停、slow-start、handler 运行时回报和依赖注入路径。
- 审查发现定时探活连续失败暂停只读取 `failure_threshold` 条结果，导致超过阈值后暂停时长无法继续递增到上限。
- 修复为按 `plan.MaxResults` 与阈值取较大值读取最近结果，使连续失败数能正确增长，并补充递增冷却回归测试。

### Testing
- 发现问题前的回归测试红灯：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/service -run 'TestScheduledTestRunnerActiveProbePauseDurationGrowsWithConsecutiveFailures'`，失败原因是第二次暂停仍约为 1 分钟，未递增。
- 修复后通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/service -run 'TestScheduledTestRunnerActiveProbePauseDurationGrowsWithConsecutiveFailures'`。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go vet ./internal/service ./internal/config ./internal/handler ./cmd/server`。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/service ./internal/config ./internal/handler ./cmd/server`。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go build ./cmd/server`。
- `staticcheck`、`golangci-lint`、`govulncheck` 本机未安装，未能执行。

### Notes
- `backend/internal/service/scheduled_test_runner_service.go`：修复连续失败统计读取范围，确保主动探活暂停时长可随连续失败次数递增。
- `backend/internal/service/scheduled_test_runner_service_test.go`：新增暂停时长递增回归测试。
- `progress.md`：追加本轮审查、修复和验证说明。
- 回滚方式：回退 `scheduled_test_runner_service.go` 中读取最近结果数量的修改，并删除 `TestScheduledTestRunnerActiveProbePauseDurationGrowsWithConsecutiveFailures` 回归测试；如需恢复审查前日志，从 `progress.md` 末尾移除本轮记录。

## 2026-06-28 - Task: 新增后台调度策略设置
### What was done
- 在后台系统设置的网关页新增“调度策略”配置卡片，覆盖多目标评分权重、延迟/额度阈值、粘性会话、主动探活暂停和恢复慢启动参数。
- 扩展 Admin settings GET/PUT DTO 和服务层设置项，将调度策略保存到已有 settings 表，并让运行时调度优先读取后台覆盖值，未配置时回退配置文件/环境变量。
- 将 GatewayService 和 ScheduledTestRunnerService 接入后台调度策略读取，使请求级调度和主动探活暂停均可使用后台配置。
- 更新调度优化文档，说明后台调度策略入口、可配置项、运行时读取方式和回滚方式。

### Testing
- 红灯验证：新增服务测试后，`go test` 曾因 `GetGatewaySchedulingConfig`、`gateway_scheduling` DTO、运行时设置覆盖等尚未实现而失败。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/service -run 'TestSettingServiceGatewayScheduling|TestScheduler|TestScheduledTestRunner'`。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/handler/admin -run 'TestSettingHandler'`。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/service ./internal/config ./internal/handler ./cmd/server`。
- 通过：`pnpm run typecheck`。
- 通过：`pnpm run build`，仅保留 Vite 既有 chunk/dynamic import 警告。

### Notes
- `backend/internal/service/gateway_scheduling_settings.go`：新增后台调度策略 settings 读取、解析、校验和写入映射。
- `backend/internal/service/setting_service_gateway_scheduling_test.go`：新增调度策略默认值、DB 覆盖和非法值回退测试。
- `backend/internal/service/setting_service.go`、`settings_view.go`、`domain_constants.go`：扩展调度策略设置项、系统设置视图和缓存刷新。
- `backend/internal/handler/admin/setting_handler.go`、`backend/internal/handler/dto/settings.go`：暴露 `gateway_scheduling` GET/PUT 字段并做保存校验。
- `backend/internal/service/gateway_service.go`：普通网关调度优先读取后台调度策略覆盖值。
- `backend/internal/service/scheduled_test_runner_service.go`：主动探活暂停和 slow-start 配置优先读取后台调度策略覆盖值。
- `backend/internal/service/wire.go`、`backend/cmd/server/wire_gen.go`：同步设置服务依赖注入。
- `frontend/src/api/admin/settings.ts`：新增调度策略类型和设置请求/响应类型。
- `frontend/src/views/admin/SettingsView.vue`：新增后台“调度策略”配置卡片并接入保存 payload。
- `docs/SCHEDULER_OPTIMIZATION_NOTES.md`：补充后台调度策略设置说明。
- `progress.md`：追加本轮实现、验证和回滚说明。
- 回滚方式：后台层可清空或恢复 settings 表中的 `gateway_scheduling_*` 设置项；代码层面可回退上述后端 settings、handler、wire、前端设置页、文档和日志改动；运行时会在无后台设置时回退配置文件/环境变量。

## 2026-06-29 - Task: 构建并推送后台调度策略镜像
### What was done
- 基于当前未提交工作区构建 Docker release 镜像，镜像 commit 标识使用 `35349fc2-dirty`，避免与纯净提交镜像混淆。
- 推送腾讯云 CCR 镜像版本 tag：`ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.139-35349fc2-dirty-20260629052908`。
- 同步更新并推送 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:latest`。
- 本地 `.docker-last-tag` 已记录本次镜像 tag。

### Testing
- Docker 构建通过：前端 `pnpm run build`、后端 Go release build、最终镜像构建均成功。
- 腾讯云 CCR 推送完成：版本 tag 与 `latest` 均推送成功。
- 远端 manifest digest 已核验一致：`sha256:f081465b39a6176aca8fcf61f06d594ef9f7f5fc199d5fa81c493d0d42600005`。
- 构建过程中仅出现 Vite 既有 dynamic import/chunk size 警告、Browserslist 数据提示和 Docker legacy builder 提示，未阻断构建。

### Notes
- `.docker-last-tag`：记录本次镜像 tag `0.1.139-35349fc2-dirty-20260629052908`，仅作本地发布记录。
- `progress.md`：追加本轮构建、推送、验证和回滚说明。
- 回滚方式：部署端可将镜像 tag 回切到上一次已知可用版本 `0.1.139-34d011c5-20260626223230`；如需恢复本地发布记录，可将 `.docker-last-tag` 改回该 tag，并从 `progress.md` 末尾移除本轮 `2026-06-29` 记录。

## 2026-06-29 - Task: 合并上游 v0.1.140
### What was done
- 将上游 `Wei-Shaw/sub2api` 的 `v0.1.140` release tag 合并到当前 `main` 分支。
- 在合并中保留本分支后台调度策略设置、图片 reasoning、Claude Code Codex 插件放行等既有定制，同时吸收上游 Grok、Count Tokens、支付回调、内容审核、系统日志、平台额度等更新。
- 解决 README、配置示例、网关路由、OpenAI/Grok 服务、支付订单、设置页和多语言文案等冲突。
- 修复合并后重复测试名导致的后端编译失败。

### Testing
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/config ./internal/handler ./internal/server ./internal/service ./cmd/server`。
- 通过：`pnpm run typecheck`。
- 通过：`pnpm run build`，仅保留 Vite 既有 dynamic import/chunk size 警告和 Browserslist 数据提示。
- 通过：`git diff --cached --check`，未发现冲突标记或空白错误。

### Notes
- `README.md`、`README_CN.md`、`README_JA.md`：同步 v0.1.140 文档更新并保留本分支说明。
- `backend/`：合并 v0.1.140 后端服务、路由、仓储、迁移、测试和调度相关更新。
- `frontend/`：合并 v0.1.140 前端 API、组件、页面、测试和 i18n 更新。
- `deploy/config.example.yaml`：合并上游配置示例并保留本分支普通网关调度策略示例。
- `progress.md`：追加本轮合并、验证和回滚说明。
- 回滚方式：对本轮合并提交执行 `git revert -m 1 <merge_commit>`；若只需回退上游同步，可回到合并前提交 `57ca70fa`。

## 2026-06-29 - Task: 同步运行时版本到 0.1.140
### What was done
- 将后端运行时版本号从 `0.1.139` 更新为 `0.1.140`，与已合并的上游 release tag 保持一致。

### Testing
- 通过：读取 `backend/cmd/server/VERSION` 确认为 `0.1.140`。
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./cmd/server -run 'TestDoesNotExist'`。

### Notes
- `backend/cmd/server/VERSION`：同步运行时版本号为 `0.1.140`。
- `progress.md`：追加本轮版本同步、验证和回滚说明。
- 回滚方式：将 `backend/cmd/server/VERSION` 改回 `0.1.139`，并从 `progress.md` 末尾移除本轮 `2026-06-29` 记录。

## 2026-06-29 - Task: 合并上游 v0.1.141 并同步版本
### What was done
- 将上游 `Wei-Shaw/sub2api` 的 `v0.1.141` release tag 合并到当前 `main` 分支。
- 同步运行时版本号为 `0.1.141`。
- 吸收上游 usage request type 统计、用户用量视图、支付页和相关后端 API 更新。

### Testing
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/config ./internal/handler ./internal/server ./internal/service ./cmd/server`。
- 通过：`pnpm run build`，仅保留 Vite 既有 dynamic import/chunk size 警告和 Browserslist 数据提示。

### Notes
- `backend/cmd/server/VERSION`：同步运行时版本号为 `0.1.141`。
- `backend/`：合并 v0.1.141 用量统计、DTO、仓储、服务和路由更新。
- `frontend/`：合并 v0.1.141 用量页、图表、支付页和 API 类型更新。
- `progress.md`：追加本轮合并、版本同步、验证和回滚说明。
- 回滚方式：对本轮合并提交执行 `git revert -m 1 <merge_commit>`；若只需回退版本号，将 `backend/cmd/server/VERSION` 改回 `0.1.140`。

## 2026-07-01 - Task: 构建并推送 v0.1.141 腾讯云镜像
### What was done
- 基于当前干净提交 `2842015b` 构建 Docker release 镜像，运行时版本为 `0.1.141`。
- 推送腾讯云 CCR 镜像版本 tag：`ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.141-2842015b-20260701031614`。
- 同步更新并推送 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:latest`。
- 本地 `.docker-last-tag` 记录本次镜像 tag；该文件不提交。

### Testing
- Docker 构建通过：前端 `pnpm run build`、后端 Go release build、最终镜像构建均成功。
- 腾讯云 CCR 推送完成：版本 tag 与 `latest` 均推送成功。
- 远端 manifest digest 已核验一致：`sha256:9955e634d7a99b8b2279e65453287392d35a3fc482db21478a3bdc3a8124b8ad`。
- 构建过程中仅出现 Vite 既有 dynamic import/chunk size 警告、Browserslist 数据提示、Node deprecation 提示和 Docker legacy builder 提示，未阻断构建。

### Notes
- `.docker-last-tag`：记录本次镜像 tag `0.1.141-2842015b-20260701031614`，仅作本地发布记录。
- `progress.md`：追加本轮构建、推送、验证和回滚说明。
- 回滚方式：部署端可将镜像 tag 回切到上一次已知可用版本；代码层面可回退到提交 `2842015b` 之前的版本，或对本轮日志提交执行 `git revert`。

## 2026-07-01 - Task: 合并上游 v0.1.142 并同步版本
### What was done
- 将上游 `Wei-Shaw/sub2api` 的 `v0.1.142` release tag 合并到当前 `main` 分支。
- 解决 OpenAI 模型列表、账号仓储测试、GPT-5.5 计费兜底、系统默认设置和账号菜单的合并冲突。
- 同步运行时版本号为 `0.1.142`。
- 吸收上游 Spark shadow、dateline normalization、Grok media、账号影子凭据、订阅/账号相关更新。

### Testing
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/config ./internal/handler ./internal/server ./internal/service ./cmd/server`。
- 通过：`pnpm run build`，仅保留 Vite 既有 dynamic import/chunk size 警告和 Browserslist 数据提示。
- 通过：`git diff --check`，未发现冲突标记或空白错误。

### Notes
- `backend/cmd/server/VERSION`：同步运行时版本号为 `0.1.142`。
- `backend/`：合并 v0.1.142 后端模型、账号、影子路由、Grok media、订阅和配置更新。
- `frontend/`：合并 v0.1.142 前端账号管理、Spark shadow、登录/注册和多语言更新。
- `progress.md`：追加本轮合并、版本同步、验证和回滚说明。
- 回滚方式：对本轮合并提交执行 `git revert -m 1 <merge_commit>`；若只需回退版本号，将 `backend/cmd/server/VERSION` 改回 `0.1.141`。

## 2026-07-02 - Task: 合并上游 v0.1.143 并同步版本
### What was done
- 将上游 `Wei-Shaw/sub2api` 的 `v0.1.143` release tag 合并到当前 `main` 分支。
- 解决配置、API Key 认证缓存快照和 OpenAI OAuth 透传测试的合并冲突。
- 同步运行时版本号为 `0.1.143`。
- 吸收上游分组高峰倍率、IP 地理展示、OpenAI compact 模型降级配置、Anthropic API Key 认证、Grok media 分组开关和订阅/用量相关更新。

### Testing
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/config ./internal/handler ./internal/server ./internal/service ./cmd/server`。
- 通过：`pnpm run build`，仅保留 Vite 既有 dynamic import/chunk size 警告和 Browserslist 数据提示。
- 通过：`git diff --check`，未发现空白错误或未清理的冲突标记。

### Notes
- `backend/cmd/server/VERSION`：同步运行时版本号为 `0.1.143`。
- `backend/`：合并 v0.1.143 后端配置、分组高峰倍率、Anthropic API Key、用量统计、OpenAI/Grok/订阅和缓存快照更新。
- `frontend/`：合并 v0.1.143 前端分组高峰倍率、IP 地理展示、订阅、用量、账号与多语言更新。
- `deploy/`：合并示例环境变量和示例配置中的新增开关。
- `progress.md`：追加本轮合并、版本同步、验证和回滚说明。
- 回滚方式：对本轮合并提交执行 `git revert -m 1 <merge_commit>`；若只需回退版本号，将 `backend/cmd/server/VERSION` 改回 `0.1.142`。

## 2026-07-02 - Task: 构建并推送 v0.1.143 腾讯云镜像
### What was done
- 基于当前提交 `0688f39a` 构建 Docker release 镜像，运行时版本为 `0.1.143`。
- 推送腾讯云 CCR 镜像版本 tag：`ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.143-0688f39a-20260703010526`。
- 同步更新并推送 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:latest`。

### Testing
- Docker 构建通过：前端 `pnpm run build`、后端 Go release build、最终镜像构建均成功。
- 腾讯云 CCR 推送完成：版本 tag 与 `latest` 均推送成功。
- 远端 manifest digest 已返回一致：`sha256:febc15bfd4bdb3253dc8020092145fcf6c50118b35eefc7e2c9b7758f44a8d3c`。
- 构建过程中仅出现 Docker legacy builder 提示，未阻断构建；前端构建复用缓存。

### Notes
- `progress.md`：追加本轮构建、推送、验证和回滚说明。
- 回滚方式：部署端可将镜像 tag 回切到上一次已知可用版本；代码层面可回退到提交 `0688f39a` 之前的版本，或对本轮日志提交执行 `git revert`。

## 2026-07-04 - Task: 合并上游 v0.1.144 并同步版本
### What was done
- 将上游 `Wei-Shaw/sub2api` 的 `v0.1.144` release tag 合并到当前 `main` 分支。
- 解决 Ops 并发统计账号列表过滤参数的合并冲突，同时保留本分支既有筛选参数和上游新增分组过滤。
- 同步运行时版本号为 `0.1.144`。
- 吸收上游并发统计、账号用量、Fable 计费、OpenAI/Anthropic 限流窗口、Codex 导入、错误分类展示、IP 地理批量工具和部署配置更新。

### Testing
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/config ./internal/handler ./internal/server ./internal/repository ./internal/service ./internal/setup ./cmd/server`。
- 通过：`pnpm run build`，仅保留 Vite 既有 dynamic import/chunk size 警告和 Browserslist 数据提示。
- 通过：`git diff --check`，未发现空白错误或未清理的冲突标记。

### Notes
- `backend/cmd/server/VERSION`：同步运行时版本号为 `0.1.144`。
- `backend/`：合并 v0.1.144 后端配置、仓储、并发统计、账号用量、限流窗口、Codex 导入、用量记录和测试更新。
- `frontend/`：合并 v0.1.144 前端账号用量、用量筛选、错误分类展示、IP 地理批量工具和多语言更新。
- `deploy/`：合并 Docker Compose 与环境变量示例新增配置。
- `progress.md`：追加本轮合并、版本同步、验证和回滚说明。
- 回滚方式：对本轮合并提交执行 `git revert -m 1 <merge_commit>`；若只需回退版本号，将 `backend/cmd/server/VERSION` 改回 `0.1.143`。

## 2026-07-04 - Task: 构建并推送 v0.1.144 腾讯云镜像
### What was done
- 基于当前提交 `db494d32` 构建 Docker release 镜像，运行时版本为 `0.1.144`。
- 推送腾讯云 CCR 镜像版本 tag：`ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.144-db494d32-20260704122658`。
- 同步更新并推送 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:latest`。

### Testing
- Docker 构建通过：前端 `pnpm run build`、后端 Go release build、最终镜像构建均成功。
- 腾讯云 CCR 推送完成：版本 tag 与 `latest` 均推送成功。
- 远端 manifest digest 已返回一致：`sha256:ea489b0f1561570c904103c074ef5562f5db34fedd293cc21b9678127fe13e8c`。
- 首次构建因整体超时中断，重试时曾遇到 `goproxy.cn` 依赖下载 `unexpected EOF`，再次重试后构建成功。

### Notes
- `progress.md`：追加本轮构建、推送、验证和回滚说明。
- 回滚方式：部署端可将镜像 tag 回切到上一次已知可用版本；代码层面可回退到提交 `db494d32` 之前的版本，或对本轮日志提交执行 `git revert`。
