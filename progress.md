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

## 2026-07-04 - Task: 新增管理员调度面板与优先账号调度
### What was done
- 新增管理员“调度面板”，将原系统设置中的调度策略迁移到独立页面，并在侧边栏“账号管理”下方新增入口。
- 新增独立调度配置 API，只更新调度相关 setting，避免系统设置页保存其他配置时误覆盖调度策略。
- 新增优先账号配置，优先账号只在通过硬过滤、匹配当前请求且位于当前 priority 候选层内生效，不改变账号自身 priority。
- 新增最近调度内存日志，记录成功选择、粘性命中/重绑、无候选和无可用槽位等关键结果，便于管理员排障观察。
- 更新调度文档，说明调度顺序、优先账号边界、内存日志边界、接口和回滚方式。

### Testing
- 通过：`GOCACHE="$PWD/.gocache" go test ./internal/service -run "GatewayScheduling|SchedulingLog|SchedulerPolicyPreferred"`。
- 通过：`GOCACHE="$PWD/.gocache" go test ./internal/handler/admin ./internal/server/routes`。
- 通过：`GOCACHE="$PWD/.gocache" go test ./...`。
- 通过：`pnpm run build`，仅保留 Vite 既有 dynamic import/chunk size 警告和 Browserslist 数据提示。
- 通过：`pnpm exec vitest run src/api/__tests__/settings.gatewayScheduling.spec.ts`。

### Notes
- `backend/internal/config/config.go`：为网关调度配置增加优先账号字段。
- `backend/internal/service/gateway_scheduling_settings.go`、`backend/internal/service/setting_service.go`、`backend/internal/service/domain_constants.go`：增加优先账号 setting、调度配置独立更新方法和默认值。
- `backend/internal/service/account_scheduler_policy.go`、`backend/internal/service/gateway_service.go`：接入同层优先账号排序和关键调度结果日志。
- `backend/internal/service/scheduling_log_service.go`、`backend/internal/service/scheduling_log_service_test.go`：新增内存环形调度日志服务和并发/容量测试。
- `backend/internal/handler/admin/scheduling_handler.go`、`backend/internal/handler/admin/setting_handler.go`、`backend/internal/handler/dto/settings.go`、`backend/internal/handler/handler.go`、`backend/internal/handler/wire.go`、`backend/internal/server/routes/admin.go`、`backend/cmd/server/wire_gen.go`：新增调度面板接口、DTO 字段、路由和注入注册。
- `frontend/src/views/admin/SchedulingView.vue`、`frontend/src/api/admin/scheduling.ts`、`frontend/src/api/admin/index.ts`、`frontend/src/router/index.ts`、`frontend/src/components/layout/AppSidebar.vue`：新增调度面板页面、API、路由和侧边栏入口。
- `frontend/src/views/admin/SettingsView.vue`、`frontend/src/api/admin/settings.ts`、`frontend/src/api/__tests__/settings.gatewayScheduling.spec.ts`：移除系统设置页调度卡片，补充优先账号类型并把调度保存测试迁移到独立接口。
- `docs/SCHEDULER_OPTIMIZATION_NOTES.md`：追加调度面板、优先账号和调度日志说明。
- `progress.md`：追加本轮实现、验证和回滚说明。
- 回滚方式：对本轮提交执行 `git revert <commit>`；若只需运行时关闭优先账号，可在调度面板清空优先账号并恢复默认调度参数后保存，或清空 settings 表中的 `gateway_scheduling_*` 设置项回退到配置默认值。

## 2026-07-04 - Task: 修复调度面板审查风险并二次验证
### What was done
- 修复系统设置页保存其他配置时仍可能回写调度 setting 的风险；默认系统设置保存不再写入或刷新 `gateway_scheduling_*`，只有请求显式携带调度配置时才兼容写入。
- 修复负载批量读取失败进入 legacy fallback 时未应用优先账号、未记录调度日志的问题，并补齐无负载信息时的默认 loadInfo，避免 fallback 排序空指针。
- 修复管理员侧边栏调度面板入口硬编码中文的问题，改为中英文多语言 key。
- 针对上述两个后端风险补充回归测试，并完成二次代码审查，未发现残留上线阻断风险。

### Testing
- 通过：`GOCACHE="$PWD/.gocache" go test ./internal/service -run "UpdateSettingsWithAuthSourceDefaultsSkipsGatewaySchedulingSettings|GatewayLegacyFallbackUsesPreferredAccountAndRecordsLog"`。
- 通过：`GOCACHE="$PWD/.gocache" go test ./internal/service -run "GatewayScheduling|SchedulingLog|SchedulerPolicyPreferred|GatewayLegacyFallback"`。
- 通过：`GOCACHE="$PWD/.gocache" go test ./internal/handler/admin ./internal/server/routes`。
- 通过：`GOCACHE="$PWD/.gocache" go test ./...`。
- 通过：`pnpm run build`，仅保留 Vite 既有 dynamic import/chunk size 警告、chunk size 警告和 Browserslist 数据提示。
- 通过：`pnpm exec vitest run src/api/__tests__/settings.gatewayScheduling.spec.ts src/i18n/__tests__/riskControlLocales.spec.ts`。
- 通过：`git diff --check`，未发现空白错误或冲突标记。

### Notes
- `backend/internal/service/setting_service.go`：为系统设置更新增加调度写入开关，默认跳过调度 setting 和调度缓存刷新，避免覆盖调度面板配置。
- `backend/internal/handler/admin/setting_handler.go`：仅当请求显式携带 `gateway_scheduling` 时才让系统设置接口兼容写入调度配置。
- `backend/internal/service/gateway_service.go`：legacy fallback 排序接入优先账号，并在成功选择时写入调度日志。
- `backend/internal/service/setting_service_gateway_scheduling_test.go`：新增系统设置保存不写调度 setting 的回归测试。
- `backend/internal/service/gateway_legacy_fallback_scheduling_test.go`：新增 legacy fallback 使用优先账号并记录日志的回归测试。
- `frontend/src/components/layout/AppSidebar.vue`、`frontend/src/i18n/locales/zh.ts`、`frontend/src/i18n/locales/en.ts`：调度面板菜单入口改为多语言展示。
- `progress.md`：追加本轮风险修复、验证和回滚说明。
- 回滚方式：对本轮未提交修复执行 `git restore` 回退上述文件；如已提交，则对修复提交执行 `git revert <commit>`。

## 2026-07-04 - Task: 新增上游倍率源、可用率检测与调度软信号
### What was done
- 新增上游倍率源能力，支持配置 Sub2API/NewAPI 源、Bearer Token 脱敏保存、手动测试、手动同步、倍率快照和最近 1 小时可用率展示。
- 新增上游分组到本地账号/分组的绑定数据结构，支持 first/avg/min/max、offset 和 clamp 规则，为后续精细成本信号提供基础。
- 将上游倍率接入调度配置，默认关闭；开启后只作为同 priority 层内 Weighted P2C 的软成本信号，不绕过硬过滤、不改账号 priority、不改账号倍率。
- 在调度面板新增“上游倍率源与可用率”管理区，以及“上游倍率软信号”配置区。
- 更新调度优化文档，说明上游倍率源、可用率检测、调度接入边界和回滚方式。

### Testing
- 通过：`GOCACHE="$PWD/.gocache" go test ./internal/service -run "UpstreamRate|SchedulerPolicy"`。
- 通过：`GOCACHE="$PWD/.gocache" go test ./internal/repository ./internal/handler/admin ./internal/server/routes -run "UpstreamRate|Scheduling"`。
- 通过：`pnpm exec vitest run src/api/__tests__/upstreamRates.spec.ts src/api/__tests__/settings.gatewayScheduling.spec.ts`。
- 通过：`pnpm run build`，仅保留 Vite 既有 dynamic import/chunk size 警告和 Browserslist 数据提示。
- Wire 生成完成：`GOCACHE="$PWD/.gocache" go generate ./cmd/server`。
- 通过：第一轮审查 `git diff --check`、后端定向测试、前端定向测试均通过，未发现 token 明文响应、请求热路径外部 HTTP 或默认开启调度影响。
- 通过：第二轮审查 `GOCACHE="$PWD/.gocache" go test ./...`、`pnpm run build`、`git diff --check` 均通过，未发现生成文件缺失、意外全量格式化或敏感信息写入。

### Notes
- `backend/migrations/159_add_upstream_rate_sources.sql`：新增上游倍率源、快照、绑定和健康检查表。
- `backend/internal/service/upstream_rate_*.go`：新增上游倍率源管理、采集解析、后台同步 runner 和调度信号提供器。
- `backend/internal/repository/upstream_rate_repo.go`：新增上游倍率 SQL 仓储、健康聚合和账号信号聚合。
- `backend/internal/handler/admin/upstream_rate_handler.go`、`backend/internal/server/routes/admin.go`：新增上游倍率管理 API 和调度子路由。
- `backend/internal/config/config.go`、`backend/internal/service/gateway_scheduling_settings.go`、`backend/internal/handler/dto/settings.go`、`backend/internal/handler/admin/setting_handler.go`：新增上游倍率调度软信号配置。
- `backend/internal/service/account_scheduler_policy.go`、`backend/internal/service/gateway_service.go`：调度成本接入上游倍率软惩罚，默认关闭且快照缺失/过期时中性。
- `backend/internal/repository/wire.go`、`backend/internal/service/wire.go`、`backend/internal/handler/wire.go`、`backend/cmd/server/wire_gen.go`：接入新增仓储、服务、runner 和 handler 注入。
- `frontend/src/api/admin/upstreamRates.ts`、`frontend/src/api/admin/index.ts`、`frontend/src/api/admin/settings.ts`：新增前端 API 与调度配置类型。
- `frontend/src/views/admin/SchedulingView.vue`、`frontend/src/views/admin/SettingsView.vue`：调度面板新增上游倍率管理和软信号配置，系统设置默认配置补齐新字段。
- `frontend/src/api/__tests__/upstreamRates.spec.ts`、`frontend/src/api/__tests__/settings.gatewayScheduling.spec.ts`、`backend/internal/service/upstream_rate_*_test.go`、`backend/internal/service/account_scheduler_policy_test.go`：新增/更新回归测试。
- `docs/SCHEDULER_OPTIMIZATION_NOTES.md`：追加上游倍率源与可用率检测落地说明。
- `progress.md`：追加本轮实现、验证和回滚说明。
- 回滚方式：代码层执行 `git revert <feature_commit>`；运行时可在调度面板关闭“上游倍率软信号”；数据侧可执行 `UPDATE upstream_rate_sources SET enabled = false, use_for_scheduling = false;`。

## 2026-07-04 - Task: 修正调度面板布局和按分组优先账号
### What was done
- 修复调度面板未包裹后台通用布局的问题，恢复与其他后台页面一致的侧边栏和顶部栏展示。
- 将“优先调度账号”从全局单选改为按分组分别配置，未配置的分组继续使用常规调度，不再把一个优先账号应用到全部分组。
- 删除调度面板中的“上游倍率源与可用率”管理块，以及上游倍率软信号配置入口，避免暴露暂时无实际价值的界面。
- 保留后端上游倍率能力与接口，不在本轮做破坏性删除，避免影响已提交迁移和服务注入。

### Testing
- 通过：`GOCACHE="$PWD/.gocache" go test ./internal/service ./internal/handler/admin -run "GatewayScheduling|SchedulerPolicyPreferred|GatewayLegacyFallback"`。
- 通过：`pnpm exec vitest run src/api/__tests__/settings.gatewayScheduling.spec.ts`。
- 通过：`pnpm run build`，仅保留 Vite 既有 dynamic import/chunk size 警告和 Browserslist 数据提示。

### Notes
- `frontend/src/views/admin/SchedulingView.vue`：新增 `AppLayout` 包裹，移除上游可用率界面，按分组展示优先账号选择。
- `frontend/src/views/admin/SettingsView.vue`、`frontend/src/api/admin/settings.ts`、`frontend/src/api/__tests__/settings.gatewayScheduling.spec.ts`：补充分组优先账号配置字段和默认值。
- `backend/internal/config/config.go`、`backend/internal/service/gateway_scheduling_settings.go`、`backend/internal/service/domain_constants.go`、`backend/internal/service/setting_service.go`：新增 `preferred_account_by_group_id` setting 读写与校验。
- `backend/internal/service/account_scheduler_policy.go`、`backend/internal/service/gateway_service.go`、`backend/internal/service/account_scheduler_policy_test.go`、`backend/internal/service/setting_service_gateway_scheduling_test.go`：调度按当前 group_id 应用优先账号，并补充回归测试。
- `backend/internal/handler/dto/settings.go`、`backend/internal/handler/admin/setting_handler.go`：调度配置 DTO 支持按分组优先账号。
- `docs/SCHEDULER_OPTIMIZATION_NOTES.md`、`progress.md`：同步说明本轮界面和调度策略调整。
- 回滚方式：对本轮未提交改动执行 `git restore` 回退上述文件；如后续提交，则对该提交执行 `git revert <commit>`。

## 2026-07-05 - Task: 调度面板标题与配置说明修正
### What was done
- 删除调度面板页面内部重复标题和描述，将页面标题与说明统一交给后台顶部标题栏展示。
- 为调度路由补充国际化标题键，修复顶部栏显示英文 fallback 和 `admin.scheduling.description` 原始 key 的问题。
- 在调度策略配置中补充“影响范围”说明，明确评分权重、阈值与粘性、主动探活暂停、恢复慢启动分别影响哪些调度行为。

### Testing
- 通过：`pnpm run build`，仅保留 Vite 既有 dynamic import/chunk size 警告和 Browserslist 数据提示。

### Notes
- `frontend/src/router/index.ts`：调度路由改为使用 `admin.scheduling.title` 标题键。
- `frontend/src/i18n/locales/zh.ts`、`frontend/src/i18n/locales/en.ts`：新增调度面板顶部栏标题和描述文案。
- `frontend/src/views/admin/SchedulingView.vue`：移除页内重复标题区，补充各配置小标题的影响说明。
- `progress.md`：追加本轮修正、验证和回滚说明。
- 回滚方式：对本轮未提交改动执行 `git restore frontend/src/router/index.ts frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts frontend/src/views/admin/SchedulingView.vue progress.md`；如后续提交，则执行 `git revert <commit>`。

## 2026-07-05 - Task: OAuth 主池优先与 API Key 兜底调度策略
### What was done
- 新增凭据类型调度策略，支持 `balanced`、`oauth_first`、`api_key_first` 三种模式。
- 实现 OAuth/API Key 主池与兜底池排序：主池先按现有 priority、分组优先账号和 Weighted P2C 排序，兜底开启时主池满载或受限后继续尝试另一类账号。
- 调度日志新增凭据策略和最终账号凭据类型，调度面板日志表格同步展示。
- 调度面板在“调度策略配置”中新增“凭据类型策略”小卡片，不改动整体页面布局。
- 明确过期时间只用于账号可用性过滤和提醒，不做“越快过期越优先”。

### Testing
- 通过：`GOCACHE="$PWD/../.gocache" go test ./internal/service ./internal/handler/admin -run "GatewayScheduling|SchedulerPolicyCredential|SchedulerPolicyPreferred|GatewayLegacyFallback"`。
- 通过：`pnpm exec vitest run src/api/__tests__/settings.gatewayScheduling.spec.ts`。
- 通过：`GOCACHE="$PWD/../.gocache" go test ./...`。
- 通过：`pnpm run build`，仅保留 Vite 既有 dynamic import/chunk size 警告和 Browserslist 数据提示。
- 通过：`git diff --check`。

### Notes
- `backend/internal/config/config.go`：新增凭据类型策略配置结构和值常量。
- `backend/internal/service/account_scheduler_policy.go`：新增凭据主池/兜底池排序函数，默认 balanced 保持旧行为。
- `backend/internal/service/gateway_service.go`、`backend/internal/service/scheduling_log_service.go`：调度路径使用凭据感知排序，并记录凭据策略和最终账号类型。
- `backend/internal/service/domain_constants.go`、`backend/internal/service/gateway_scheduling_settings.go`：新增凭据策略 setting key、读写和校验。
- `backend/internal/handler/dto/settings.go`、`backend/internal/handler/admin/setting_handler.go`：调度配置 API 支持凭据策略字段。
- `frontend/src/views/admin/SchedulingView.vue`、`frontend/src/api/admin/settings.ts`、`frontend/src/api/admin/scheduling.ts`、`frontend/src/views/admin/SettingsView.vue`、`frontend/src/api/__tests__/settings.gatewayScheduling.spec.ts`：新增凭据策略配置、日志展示和前端类型/测试。
- `docs/SCHEDULER_OPTIMIZATION_NOTES.md`、`progress.md`：同步记录凭据类型主池策略和回滚说明。
- 回滚方式：执行 `git revert <commit>` 回退本轮功能提交；运行时可将“凭据类型策略”改回“均衡使用”并保存，立即恢复旧调度行为。

## 2026-07-06 - Task: 合并上游 v0.1.145 发布版
### What was done
- 在确认工作区干净后，抓取并合并 `Wei-Shaw/sub2api` 的 `v0.1.145` 发布标签内容到当前分支。
- 解决合并冲突，保留本分支调度面板能力，同时合入上游高级调度、EasyPay、支付配置、账号过滤、WebSocket 和多语言等更新。
- 同步运行时版本文件为 `0.1.145`，确保 Docker 镜像版本与本次合并发布版一致。
- 修复合并后 OpenAI WS ingress 调度兼容问题：`responses_websockets_v2_ingress` 可正确匹配 WSv2/passthrough 与 HTTP bridge 模式。
- 修复 simple 模式和测试场景下调度快照为空时的 repo 回退路径，避免误报 `no available account`。

### Testing
- 通过：`GOCACHE="C:/Users/MilkFoam/AppData/Local/go-build" go test ./...`。
- 通过：`npx vue-tsc --noEmit`。
- 通过：`npx tsc --noEmit -p tsconfig.node.json`。
- 通过：`npx vite build`，仅保留既有 Browserslist 过旧、dynamic import 和 chunk size 警告。
- 说明：`npm run build` 在 Git Bash 下仅返回退出码 1 且无详细错误；已用同等分段命令完成类型检查和 Vite 构建验证，后续 Docker 构建会继续覆盖真实发布链路。

### Notes
- `backend/cmd/server/VERSION`：同步运行时版本号为 `0.1.145`。
- `backend/internal/handler/admin/setting_handler.go`：合并设置响应字段，移除冲突残留重复字段块，保留调度与支付新增字段。
- `backend/internal/service/openai_account_scheduler.go`：补齐高级调度额度余量常量、影子账号母账号健康回退和 WS ingress 传输兼容逻辑。
- `backend/internal/service/openai_gateway_service.go`：simple 模式调度候选与账号刷新优先走 repo，避免空快照误淘汰账号。
- `backend/internal/service/openai_account_scheduler_test.go`：补回分组感知调度测试 stub。
- `README.md`、`README_CN.md`、`README_JA.md`、`backend/`、`frontend/`、`deploy/`：合并上游 v0.1.145 发布版主体改动。
- `progress.md`：追加本轮合并、验证和回滚说明。
- 回滚方式：对本轮合并提交执行 `git revert -m 1 <merge_commit>`；若只回退本轮冲突修复，可对后续修复提交执行 `git revert <commit>`，或按上述文件逐项恢复。

## 2026-07-06 - Task: 构建并推送 v0.1.145 腾讯云镜像
### What was done
- 基于当前提交 `f6fe8374` 构建 Docker release 镜像，运行时版本为 `0.1.145`。
- 推送腾讯云 CCR 镜像版本 tag：`ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.145-f6fe8374-20260706190204`。
- 同步更新并推送 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:latest`。

### Testing
- Docker 构建通过：前端 `pnpm run build`、后端 Go release build、最终镜像构建均成功。
- 腾讯云 CCR 推送完成：版本 tag 与 `latest` 均推送成功。
- 远端 manifest digest 已返回一致：`sha256:35c424bc479f08382e3933d5e5b202c50a187e6b0ca3659de341bd81daccbb0c`。
- 构建过程仅保留 legacy builder 弃用提示、Browserslist 数据过旧、Vite dynamic import/chunk size 警告和 Node 子进程弃用提示，未阻断构建。

### Notes
- `progress.md`：追加本轮 Docker 构建、推送、验证和回滚说明。
- 回滚方式：部署端可将镜像 tag 回切到上一次已知可用版本；代码层面可回退到提交 `f6fe8374` 之前的版本，或对本轮日志提交执行 `git revert`。

## 2026-07-07 - Task: 合并上游 v0.1.146 发布版
### What was done
- 在确认工作区干净后，抓取并合并 `Wei-Shaw/sub2api` 的 `v0.1.146` 发布标签内容到当前分支。
- 解决合并冲突，保留本分支导入账号时绑定 OpenAI 分组的能力，同时合入上游拖拽上传、多文件合并导入、API Key 并发统计、账号请求头覆写、账号批量导入和新模型计价等更新。
- 同步运行时版本文件为 `0.1.146`，确保后续镜像版本与本次合并发布版一致。

### Testing
- 通过：`GOCACHE="C:/Users/MilkFoam/AppData/Local/go-build" go test ./...`。
- 通过：`npx vue-tsc --noEmit`。
- 通过：`npx tsc --noEmit -p tsconfig.node.json`。
- 通过：`npx vitest run src/__tests__/integration/data-import.spec.ts`，导入弹窗 8 个回归测试通过。
- 通过：`npx vite build`，仅保留既有 Browserslist 过旧、dynamic import 和 chunk size 警告。
- 通过：`git diff --check`，未发现空白错误或冲突标记。

### Notes
- `backend/cmd/server/VERSION`：同步运行时版本号为 `0.1.146`。
- `backend/internal/service/concurrency_service.go`：合并账号调度 debt 缓存接口与 API Key 并发统计缓存接口。
- `backend/internal/service/pricing_service.go`：合并 GPT-5.6 回退计价逻辑并保留 GPT-5.5 专用计价。
- `frontend/src/components/admin/account/ImportDataModal.vue`：合并拖拽/多文件导入与导入分组选择能力。
- `frontend/src/__tests__/integration/data-import.spec.ts`：重写导入弹窗回归测试，覆盖批量合并、文件校验、部分成功刷新和分组提交。
- `backend/`、`frontend/`、`.github/`：合并上游 v0.1.146 发布版主体改动。
- `progress.md`：追加本轮合并、验证和回滚说明。
- 回滚方式：对本轮合并提交执行 `git revert -m 1 <merge_commit>`；若只回退本轮冲突修复，可对后续修复提交执行 `git revert <commit>`，或按上述文件逐项恢复。

## 2026-07-07 - Task: 构建并推送 v0.1.146 腾讯云镜像
### What was done
- 基于当前提交 `33668d7c` 构建 Docker release 镜像，运行时版本为 `0.1.146`。
- 推送腾讯云 CCR 镜像版本 tag：`ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.146-33668d7c-20260707170124`。
- 同步更新并推送 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:latest`。

### Testing
- Docker 构建通过：前端 `pnpm run build`、后端 Go release build、最终镜像构建均成功。
- 腾讯云 CCR 推送完成：版本 tag 与 `latest` 均推送成功。
- 远端 manifest digest 已返回一致：`sha256:9b28011a1947cae3ed729ba533b8a92b0de7564af315d850a3573e4fb0e4c89e`。
- 首次 Docker 构建在后端 Go release build 阶段达到工具 10 分钟超时；重试利用缓存后构建成功。
- 构建过程仅保留 legacy builder 弃用提示、Browserslist 数据过旧、Vite dynamic import/chunk size 警告和 Node 子进程弃用提示，未阻断构建。

### Notes
- `progress.md`：追加本轮 Docker 构建、推送、验证和回滚说明。
- 回滚方式：部署端可将镜像 tag 回切到上一次已知可用版本；代码层面可回退到提交 `33668d7c` 之前的版本，或对本轮日志提交执行 `git revert`。

## 2026-07-08 - Task: OpenAI Team OAuth 401team 可重试开关
### What was done
- 为 OpenAI OAuth 母账号新增“401team 可重试”显式开关，管理员可在账号管理页“更多”菜单手动开启或关闭。
- 后端新增专用接口，只增量更新 `credentials.openai_team_401_retryable` 单个键，不通过全量账号更新覆盖脱敏后的 credentials。
- OpenAI OAuth 账号在开关开启后，遇到 `no_matching_rule` 或“无 refresh_token + Unauthorized”这类特定 401 时，不再直接写入 error/冷却，而是进入现有 failover/换号流程。
- 保持非 OpenAI、非 OAuth、影子账号、未开启开关账号和其他 401 的原有处理不变。

### Testing
- 通过：`GOCACHE="$PWD/.gocache" go test -tags unit ./internal/service -run 'TestRateLimitService_HandleUpstreamError_OpenAITeam401RetryableEntersFailoverOnly|TestRateLimitService_HandleUpstreamError_NonOAuth401|TestSetOpenAITeam401Retryable'`。
- 通过：`GOCACHE="$PWD/.gocache" go test ./internal/handler/admin`。
- 通过：`./node_modules/.bin/vitest run src/components/admin/account/__tests__/AccountActionMenu.spec.ts --reporter verbose`（在 `frontend` 目录执行）。
- 通过：`./node_modules/.bin/vue-tsc --noEmit`（在 `frontend` 目录执行）。
- 通过：`GOCACHE="$PWD/.gocache" go test ./internal/service -run 'TestRateLimitService_HandleUpstreamError_OpenAITeam401RetryableEntersFailoverOnly|TestSetOpenAITeam401Retryable'`，用于无 unit tag 编译触达。
- 说明：`GOCACHE="$PWD/.gocache" go test -tags unit ./internal/service` 的全量 service 单测在既有 `TestOpenAISelectAccountWithLoadAwareness_HydratesSelectedAccountFromSchedulerSnapshot` 调度快照用例处 panic，堆栈落在 `openai_gateway_service.go:listSchedulableAccounts`，与本轮 401team 开关目标路径无直接关系；本轮新增和受影响目标测试已通过。
- 说明：`npm --prefix frontend run typecheck` 在当前 Git Bash 环境只返回失败码且无诊断输出，已使用同一项目本地二进制 `./node_modules/.bin/vue-tsc --noEmit` 完成等价类型检查并通过。

### Notes
- `backend/internal/service/account.go`：新增 `openai_team_401_retryable` 凭据键常量和 OpenAI OAuth 开关读取 helper。
- `backend/internal/service/ratelimit_service.go`：新增特定 Team 401 匹配逻辑，命中时返回 failover，不写入 error/冷却。
- `backend/internal/service/admin_service.go`、`backend/internal/handler/admin/account_handler.go`、`backend/internal/server/routes/admin.go`：新增专用服务方法、管理员接口和路由。
- `backend/internal/repository/account_repo.go`、`backend/internal/service/account_credentials_persistence.go`：新增 credentials JSONB 增量合并能力，避免全量覆盖凭据。
- `frontend/src/components/admin/account/AccountActionMenu.vue`、`frontend/src/views/admin/AccountsView.vue`、`frontend/src/api/admin/accounts.ts`：新增菜单开关、页面处理和专用 API 调用。
- `frontend/src/i18n/locales/zh.ts`、`frontend/src/i18n/locales/en.ts`：新增 401team 可重试相关文案。
- `backend/internal/service/ratelimit_service_401_test.go`、`backend/internal/service/admin_service_credentials_merge_test.go`、`frontend/src/components/admin/account/__tests__/AccountActionMenu.spec.ts`：新增回归测试。
- `backend/internal/handler/admin/admin_service_stub_test.go`、`backend/internal/service/admin_service_spark_shadow_test.go`：同步测试桩接口以恢复编译。
- `docs/SCHEDULER_OPTIMIZATION_NOTES.md`、`progress.md`：同步记录本轮行为边界、验证结果和回滚方式。
- 回滚方式：代码层执行 `git revert <feature_commit>`；未提交时可用 `git restore` 回退上述文件。运行时可先在账号“更多”菜单关闭“401team 可重试”，关闭后该标记为中性，不影响原 401 处理。

## 2026-07-08 - Task: 构建并推送 401team 开关腾讯云镜像
### What was done
- 将 401team 可重试开关代码提交并推送到 `origin/main`，提交为 `bed4513f Add OpenAI Team 401 retry switch`。
- 基于提交 `bed4513f` 构建 Docker release 镜像，运行时版本为 `0.1.146`。
- 推送腾讯云 CCR 镜像版本 tag：`ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.146-bed4513f-20260708145750`。
- 同步更新并推送 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:latest`。

### Testing
- 代码推送完成：`main -> origin/main`。
- Docker 构建通过：前端 `pnpm run build`、后端 Go release build、最终镜像构建均成功。
- 腾讯云 CCR 推送完成：版本 tag 与 `latest` 均推送成功。
- 远端 manifest digest 已返回一致：`sha256:ef5f774ee6fa13ff9634a3e5d490de3df738b5460d3ee58cf51a4cd2851049d9`。
- 首次 Docker 构建在后端阶段达到工具 10 分钟超时；重试利用缓存后构建成功。
- 构建过程仅保留 legacy builder 弃用提示、Browserslist 数据过旧、Vite dynamic import/chunk size 警告和 Node 子进程弃用提示，未阻断构建。

### Notes
- `progress.md`：追加本轮代码推送、Docker 构建、腾讯云推送、验证和回滚说明。
- 回滚方式：部署端可将镜像 tag 回切到上一次已知可用版本；代码层面可对提交 `bed4513f` 及本轮日志提交执行 `git revert`，或切回上一个已部署镜像 tag。

## 2026-07-09 - Task: 合并上游 v0.1.147 并准备腾讯云镜像
### What was done
- 从 `https://github.com/Wei-Shaw/sub2api.git` 拉取并合并 tag `v0.1.147` 到当前 `main` 分支。
- 解决上游大文件拆分带来的冲突，保留本分支 401team 可重试、账号批量删除、调度设置/日志、OpenAI images reasoning effort 等本地能力。
- 将上游拆分后的语言包结构与 401team 菜单文案合并，旧 `frontend/src/i18n/locales/{zh,en}.ts` 切换为拆分模块。
- 修正 ent runtime 合并索引错位，避免 `privacy_filter_enabled` 默认值读取到 `rpm_limit`。

### Testing
- 通过：`GOCACHE="$PWD/.gocache" GOSUMDB=sum.golang.google.cn GOPROXY=https://goproxy.cn,direct go test ./internal/service ./internal/repository -run TestDoesNotExist`。
- 通过：`GOCACHE="$PWD/.gocache" GOSUMDB=sum.golang.google.cn GOPROXY=https://goproxy.cn,direct go test ./internal/handler/admin`。
- 通过：`GOCACHE="$PWD/.gocache" GOSUMDB=sum.golang.google.cn GOPROXY=https://goproxy.cn,direct go test -tags unit ./internal/service -run 'TestRateLimitService_HandleUpstreamError_OpenAITeam401RetryableEntersFailoverOnly|TestRateLimitService_HandleUpstreamError_NonOAuth401|TestSetOpenAITeam401Retryable|TestGatewayLegacyFallbackUsesPreferredAccountAndRecordsLog|TestAdminService_BulkDeleteAccounts'`。
- 通过：`./node_modules/.bin/vue-tsc --noEmit`（在 `frontend` 目录执行）。
- 通过：`./node_modules/.bin/vitest run src/components/admin/account/__tests__/AccountActionMenu.spec.ts --reporter verbose`（在 `frontend` 目录执行）。
- 通过：`git diff --check`。
- 说明：上游 tag `v0.1.147` 的 `backend/cmd/server/VERSION` 仍为 `0.1.146`，本轮不擅自改版本文件。

### Notes
- `backend/internal/service/*`、`backend/internal/handler/admin/*`、`frontend/src/i18n/locales/**`：合并上游 v0.1.147 拆分和本分支定制能力。
- `backend/ent/runtime/runtime.go`：修正合并后的 user field 索引错位。
- `progress.md`：记录本轮合并、冲突处理、验证与回滚说明。
- 回滚方式：代码层可对本轮 merge commit 执行 `git revert -m 1 <merge_commit>`；部署层可继续使用上一版镜像 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.146-bed4513f-20260708145750` 或上一已知可用 tag。

## 2026-07-09 - Task: 构建并推送 v0.1.147 合并版腾讯云镜像
### What was done
- 将上游 `v0.1.147` 合并结果推送到 `origin/main`：`b6667e13 Merge upstream v0.1.147 release`。
- 修复 release build 触达的 WebSocket ingress hook 签名残留冲突并推送：`1a7937c6 fix(openai): restore websocket payload hook mutation`。
- 使用临时 no-BuildKit Dockerfile 适配当前本机 Docker 环境缺少 buildx 的限制，构建完成后已删除临时文件且未提交。
- 推送腾讯云 CCR 镜像版本 tag：`ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.146-1a7937c6-20260709195307`。
- 按发布 tag 补打并推送 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:v0.1.147`。
- 同步更新并推送 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:latest`。

### Testing
- 通过：`GOCACHE="$PWD/.gocache" GOSUMDB=sum.golang.google.cn GOPROXY=https://goproxy.cn,direct go test ./internal/handler ./internal/service ./cmd/server -run TestDoesNotExist`。
- 通过：`./node_modules/.bin/vue-tsc --noEmit`（在 `frontend` 目录执行）。
- Docker 构建通过：前端 `pnpm run build`、后端 Go release build、最终 runtime 镜像构建均成功。
- 腾讯云 CCR 推送完成：`v0.1.147`、构建版本 tag 与 `latest` 均推送成功。
- 远端 manifest digest 已返回一致：`sha256:d84d13da308b2157f2e761ea11e63a7497fd88ae773421d68cbf82be0032b82e`。

### Notes
- `backend/internal/service/openai_ws_forwarder.go`、`backend/internal/service/openai_ws_forwarder_ingress.go`、`backend/internal/service/openai_ws_v2_passthrough_adapter.go`：恢复 WS hook payload 替换能力以匹配 handler 隐私过滤链路。
- `progress.md`：追加本轮镜像构建、腾讯云推送、验证与回滚说明。
- 回滚方式：部署端可回切到上一版镜像 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.146-bed4513f-20260708145750`；代码层可 revert `1a7937c6`，并对合并提交 `b6667e13` 执行 `git revert -m 1 b6667e13`。

## 2026-07-10 - Task: 建立功能清理与 v0.1.149 合并发布基线

### What was done
- 完整读取 `goal-1` 目标、计划、任务和最新检查点，完成 Task 1 的 Git、功能边界、构建测试入口及腾讯云镜像基线审计。
- 逐项解释当前 15 个已跟踪修改，确认均为正在进行的独立调度/账号分配/上游倍率清理，没有来源不明的业务修改。
- 建立必须删除与必须保留清单，明确保留普通 gateway 调度、OpenAI 高级调度和 401 Team 可重试能力。

### Testing
- 只读验证：`git status -sb`、`git branch -vv`、`git stash list`、目标提交逐个 `git show --name-status`、关键符号残留搜索。
- 工具链基线：Go `1.26.0`、Node `22.18.0`、pnpm `10.28.0`、Docker client `29.5.3`。
- 本任务未修改业务代码，未运行编译或测试；已记录当前 `buildLegacyLRUSelectionOrder` 半清理签名不一致，交由后续任务修复并验证。

### Notes
- `goal-1/tasks.md`：标记 Task 1 完成并写入完整边界、验证入口、风险与下一步。
- `progress.md`：追加本轮基线审计记录。
- 回滚方式：本轮仅新增 Goal/进度记录，可删除 `goal-1/` 并回退本段 `progress.md`；未改动业务代码或 Git 历史。

## 2026-07-10 - Task: 断开独立调度与上游倍率运行入口

### What was done
- 删除 admin 独立调度与上游倍率路由、handler 聚合字段及 Wire provider，服务启动不再构造相关 repository/service/runner。
- 普通 gateway 和定时探测 runner 恢复为只读取文件中的 `gateway.scheduling` 配置，不再使用数据库运行时覆盖。
- 从定时探测 runner 移除无效的 `SettingService` 依赖，同时保留普通 ActiveProbe、慢启动、账号暂停与恢复能力。

### Testing
- 静态搜索活动路由、handler、repository/service provider、runner 启动和 settings override：无命中。
- `git diff --check` 通过，仅有 `progress.md` 的 LF/CRLF 工作区提示。
- 执行 `go test ./internal/service -run 'TestScheduledTestRunner|TestScheduledProbe' -count=1`；编译仍被后续待删的 preferred account、credential helper、gateway settings cache 及相关测试符号阻断，改动前后错误集合一致，无新增 Task 2 构造链错误。

### Notes
- `backend/internal/server/routes/admin.go`、`backend/internal/handler/{handler.go,wire.go}`：断开 HTTP 与 handler 注入入口。
- `backend/internal/repository/wire.go`、`backend/internal/service/wire.go`、`backend/cmd/server/wire_gen.go`：断开 repository/service/runner 启动链。
- `backend/internal/service/{gateway_scheduling.go,scheduled_test_runner_service.go,setting_service.go}`：停止 DB 调度配置热更新并移除无效依赖。
- 回滚方式：提交后执行 `git revert <Task 2 commit>`；不会触碰现有 stash、历史迁移或用户尚未提交的后续清理文件。

## 2026-07-10 - Task: 全面审查调度清理边界并恢复生产编译

### What was done
- 全面复核独立调度/上游倍率的路由、DI、启动链、热路径与历史迁移，并核对普通 gateway、OpenAI 高级调度、401 Team 和分组隔离的保留证据。
- 恢复 legacy LRU 双参数行为，删除优先账号 helper，并将普通 gateway fallback 从已删除的 credential-aware helper 切回 legacy LRU。
- 删除已无入口且导致生产编译失败的 scheduling handler 和 gateway DB scheduling settings 实现。
- 确认 migration 159 保持共享历史不变，不执行破坏性 DROP 或 checksum 修改。

### Testing
- 通过：`go build ./internal/service ./internal/handler ./internal/server/routes ./cmd/server`。
- 部分通过：`go test ./internal/service ./internal/handler ./internal/server/routes ./cmd/server -run TestDoesNotExist -count=1`；handler、routes、cmd/server 通过，service 仅被 Task 4 待删旧测试符号阻断。
- 通过：`git diff --check`。
- 静态核对通过：普通调度基础结构、OpenAI advanced scheduler、401 Team retryable、`allow_ungrouped_key_scheduling`、`upstream_rate_limited` 均保留。

### Notes
- `backend/internal/service/account_scheduler_policy.go`：移除 preferred/credential/upstream-rate 策略残留并恢复 legacy LRU 编译一致性。
- `backend/internal/service/gateway_scheduling.go`：legacy fallback 恢复调用普通 LRU helper。
- `backend/internal/handler/admin/scheduling_handler.go`、`backend/internal/service/gateway_scheduling_settings.go`：删除孤立实现。
- `backend/migrations/159_add_upstream_rate_sources.sql`：只读审查，保留历史文件；闲置表后续仅可通过有授权的新前向 migration 处置。
- 回滚方式：提交后执行 `git revert <Task 3 commit>`；不会修改 migration 159 或现有数据库数据。

## 2026-07-10 - Task: 删除独立调度面板与上游倍率残留

### What was done
- 完整删除独立调度面板、优先账号调度、OAuth/API Key 凭据池策略、内存调度日志和上游倍率采集/绑定/健康检查的前后端实现及专属测试。
- 清理 settings DTO、setting keys、运行时配置字段、前端 API/路由/侧边栏/设置类型，普通 gateway 恢复只使用文件或环境变量配置。
- 保留 Weighted P2C、legacy LRU、Sticky、ActiveProbe、SlowStart、账号自身倍率、OpenAI 高级调度和 401 Team 可重试功能。
- 保留已发布 migration 159，不删除历史 migration，不执行数据库 DROP。

### Testing
- 通过：设置临时 GOCACHE 后执行 `go test ./... -run TestNameThatDoesNotExist`，全部后端包与测试完成编译触达。
- 通过：`go test ./internal/service -run TestSchedulerPolicy`、`go test ./internal/service -run TestAccount_BillingRateMultiplier`、`go test ./internal/service -run TestSetOpenAITeam401Retryable`、`go test ./internal/config`。
- 通过：`./node_modules/.bin/vue-tsc --noEmit --pretty false`、`./node_modules/.bin/tsc --noEmit -p tsconfig.node.json --pretty false`、`./node_modules/.bin/vite build`；仅保留既有 Browserslist、dynamic import 和 chunk size 警告。
- 通过：专属符号残留搜索、`git diff --check` 和 migration 159 差异检查。

### Notes
- `backend/internal/{config,handler,repository,service}`：删除独立调度/上游倍率模块、数据库调度覆盖契约和专属测试，保留普通调度与 OpenAI 路径。
- `frontend/src/{api,components/layout,router,views/admin}`：删除调度页面、API、路由、菜单入口和 settings 类型/默认值。
- `docs/SCHEDULER_OPTIMIZATION_NOTES.md`：改为说明当前仍保留的普通调度能力、历史 migration 边界和 401 Team 功能。
- `progress.md`：追加本轮删除、验证和回滚说明。
- 回滚方式：提交后执行 `git revert <Task 4 commit>`；未提交时可按本轮 `git diff --name-status` 清单逐文件恢复。migration 159 未修改，无数据库回滚动作。

## 2026-07-10 - Task: 合并上游 v0.1.149 并迁移本地修改

### What was done
- 在合并前创建本地备份分支 `backup/pre-v0.1.149-20260710-c0aa6719`，随后将上游 tag `v0.1.149` 非快进合并到当前 `main`。
- 解决用量排行冲突，保留本地 `billing_mode`/账号成本筛选与排序，同时合入上游用户 Token 细分排序、用户角色、用量页重构和版本回滚等功能。
- 补齐上游 Compact 非流式 keepalive 在当前拆分网关结构中的实现，确保心跳提交后错误响应不再触发账号 failover，并覆盖标准 OAuth 与透传路径。
- 合并后再次扫描独立调度入口和多语言键，未恢复已删除的调度页面、路由、handler、repository 或 `nav.scheduling`/`admin.scheduling` 活动文案。

### Testing
- 通过：`GOCACHE="$HOME/.cache/go-build" go test ./...`，后端全量测试通过。
- 通过：Compact keepalive 定向测试，覆盖启用、禁用、首个 tick 前错误及响应已提交后禁止 failover。
- 通过：用户排行 `GroupIDFilter`/`SortBy` 定向测试。
- 通过：`./node_modules/.bin/vitest run --reporter=dot --no-color --pool=forks`，151 个测试文件、955 个测试全部通过。
- 通过：`./node_modules/.bin/vue-tsc --noEmit --pretty false --diagnostics`、`./node_modules/.bin/tsc --noEmit -p tsconfig.node.json`、`./node_modules/.bin/vite build`；仅保留既有 Browserslist、dynamic import 和 chunk size 警告。
- 通过：`git diff --check`、未解决冲突检查及独立调度残留扫描。

### Notes
- `backend/internal/handler/admin/dashboard_handler.go`、`backend/internal/repository/usage_log_repo_trend.go`：合并本地计费维度与上游 Token 排行排序合同。
- `backend/internal/service/openai_gateway_{service,forward,passthrough,response_handling}.go`：迁移 Compact 非流式 keepalive 生命周期、错误提交和停止等待逻辑。
- `backend/`、`frontend/`、`README*.md`、`assets/`：合并上游 v0.1.149 发布内容。
- `progress.md`：追加本轮备份、合并、迁移、验证与回滚说明。
- 回滚方式：代码层对本轮 merge commit 执行 `git revert -m 1 <merge_commit>`；也可从本地备份分支 `backup/pre-v0.1.149-20260710-c0aa6719` 恢复合并前状态。部署层在新镜像验证前继续使用上一已知可用镜像。

## 2026-07-10 - Task: v0.1.149 Docker 本地构建与健康验证

### What was done
- 修复 Docker 构建上下文包含本地 Go 缓存的问题，在 `.dockerignore` 排除根目录和子目录的 `.gocache`，避免约 6.2GB 无关缓存进入上下文。
- 按用户要求清理全部未使用 Docker 镜像、停止容器、网络和构建缓存，未执行全局 volume prune；Docker 报告实际回收 19.29GB。
- 切换新网络环境后，从零构建 `sub2api:0.1.149-74b343b8-local`，镜像 ID 为 `sha256:ee73fbf3bb2e7b73363e1e5ef3a3b694800bdb8aa3f5c48197c80cdae0365a31`。
- 使用隔离 PostgreSQL/Redis 完成自动初始化、迁移和应用健康验证；验证结束后清理测试容器、网络、依赖镜像和中间层，额外回收 1.104GB，最终仅保留 303MB 成品镜像。

### Testing
- 通过：无缓存 Docker 多阶段构建，前端 `pnpm run build`、后端 Go release build 和最终 Alpine 镜像均成功。
- 通过：`docker run --rm sub2api:0.1.149-74b343b8-local --version`，输出版本 `0.1.149`、commit `74b343b8`。
- 通过：镜像进程以 `uid=1000(sub2api)`、`gid=1000(sub2api)` 非 root 用户运行。
- 通过：隔离容器健康验证，`/health` 返回 `{"status":"ok"}`，Docker health status 为 `healthy`。
- 通过：清理后 `docker system df` 显示仅 1 个成品镜像、无容器、无构建缓存；3 个匿名卷总占用 0B，按“不删除数据卷”边界保留。

### Notes
- `.dockerignore`：新增 `.gocache/` 与 `**/.gocache/`，减少 Docker 上下文和 C 盘临时占用。
- 本机构建未安装 buildx，使用仓库现有 `deploy/Dockerfile` 和 legacy builder；最终腾讯云发布前仍需按 Task 18 复核生产根 Dockerfile 的 PostgreSQL 客户端层或使用等价 no-BuildKit 构建方式。
- `progress.md`：追加 Docker 清理、构建、健康验证和空间回收证据。
- 回滚方式：代码层对本轮 `.dockerignore` 提交执行 `git revert <commit>`；本地镜像可执行 `docker image rm sub2api:0.1.149-74b343b8-local` 删除，部署仍保持上一稳定镜像不变。

## 2026-07-10 - Task: 推送 v0.1.149 合并代码与备份分支

### What was done
- 将合并前备份分支 `backup/pre-v0.1.149-20260710-c0aa6719` 正常推送到 `origin`，保留清理提交 `c0aa6719` 的远端恢复点。
- 将包含独立调度扩展删除、上游 v0.1.149 合并、Compact keepalive 迁移、Docker 上下文修复和本地验证记录的 `main` 正常推送到 `origin`。
- 全程未使用 force push、rebase、reset 或覆盖 checkout。

### Testing
- 通过：`git ls-remote --heads origin main backup/pre-v0.1.149-20260710-c0aa6719`。
- 远端 `main` 首次推送后指向 `805ff424a25676983ea19d74305a07426374bd28`。
- 远端备份分支指向 `c0aa6719ea07e4e5e38ed90d5801d4bce8994971`，与本地合并前备份一致。
- 通过：推送后 `git status --short --branch` 显示本地 `main` 与 `origin/main` 同步，工作树干净。

### Notes
- `progress.md`、`goal-1/tasks.md`：记录代码推送、远端 SHA、剩余风险和下一步镜像发布任务。
- 回滚方式：如远端代码需要撤回，创建正常 `git revert` 提交并推送；合并整体可执行 `git revert -m 1 88e11471`，合并前代码可从远端备份分支恢复，禁止改写公共历史。

## 2026-07-10 - Task: 构建并推送 v0.1.149 腾讯云生产镜像

### What was done
- 以远端 `main` 提交 `f5cfb1b1` 为源码，使用生产根 Dockerfile 的等价 no-BuildKit 临时副本构建 release 镜像；临时副本仅移除 pnpm 缓存挂载语法，完整保留 PostgreSQL 18 客户端、resources、release ldflags、非 root 用户和健康检查。
- 推送腾讯云 CCR 可追溯标签：`ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.149-f5cfb1b1-20260710052405`。
- 同步推送 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:v0.1.149` 与 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:latest`。
- 发布后清理基础镜像、测试依赖和中间层，额外回收 1.171GB；本地仅保留可追溯成品镜像。

### Testing
- 通过：生产多阶段构建，前端 `pnpm run build`、后端 Go release build、PostgreSQL 客户端层和最终运行层均成功；首次 PostgreSQL manifest TLS 超时后单独拉取并复用缓存完成。
- 通过：镜像版本 `Sub2API 0.1.149 (commit: f5cfb1b1, built: 2026-07-10T05:24:05Z)`。
- 通过：镜像使用 `uid=1000(sub2api)` 非 root 用户，包含 `psql 18.4`、`pg_dump 18.4` 和 `/app/resources`。
- 通过：隔离 PostgreSQL/Redis 自动初始化、迁移和健康检查，`/health` 返回 `{"status":"ok"}`，Docker health status 为 `healthy`。
- 通过：三个远端 manifest 均为 `linux/amd64`，digest 一致：`sha256:1f6fc5ca31023e9d353848f593c79b7d749820247fb331a4ba92b82fcaa5e1a7`。
- 发布前 `latest` 回滚摘要：`sha256:d84d13da308b2157f2e761ea11e63a7497fd88ae773421d68cbf82be0032b82e`。
- 清理后本地仅保留一个 166MB 成品镜像、无容器、无构建缓存；C 盘可用约 95GB。

### Notes
- 腾讯云镜像仓库：`ccr.ccs.tencentyun.com/apophis-chat/sub2api`，未输出或写入任何仓库凭据。
- 本机构建未安装 buildx，临时 Dockerfile 位于系统 Temp，不在 Git 工作树中；仓库生产 `Dockerfile` 未被修改。
- `progress.md`、`goal-1/tasks.md`：记录镜像标签、远端摘要、健康验证和回滚目标。
- 回滚方式：部署端可回切到发布前 digest `sha256:d84d13da308b2157f2e761ea11e63a7497fd88ae773421d68cbf82be0032b82e`；代码层可使用远端备份分支或正常 `git revert`，禁止强推。

## 2026-07-10 - Task: 修复导航和模型广场语言包键显示

### What was done
- 补齐 `nav.tokenLeaderboard`、`nav.modelMarket`、模型广场页面和管理端 Token 排行榜中英文语言包文案，避免界面回退显示变量键。
- 新增语言包回归测试，覆盖导航项、模型广场页面和 Token 排行榜相关键，并复用现有键冲突测试防止语言包结构回归。

### Testing
- 通过：`pnpm exec vitest run src/i18n/__tests__/navigationFeatureLocales.spec.ts src/i18n/__tests__/localesNoKeyCollision.spec.ts`（70 passed）。
- 通过：`pnpm test:run`（152 files / 1019 tests passed）。
- 通过：`pnpm typecheck`。
- 通过：`pnpm lint:check`。
- 通过：`pnpm build`，仅有既有 Browserslist 过期、Vite chunk 与动态/静态 import 警告。

### Notes
- `frontend/src/i18n/locales/{zh,en}/common.ts`：补齐导航菜单文案。
- `frontend/src/i18n/locales/{zh,en}/dashboard.ts`：补齐模型广场页面文案。
- `frontend/src/i18n/locales/{zh,en}/admin/resources.ts`：在现有 `admin.usage` 结构下补齐 Token 排行榜文案。
- `frontend/src/i18n/__tests__/navigationFeatureLocales.spec.ts`：新增语言包关键键回归测试。
- 回滚方式：使用 `git revert <本次提交>` 正常回退语言包和测试变更，禁止改写公共历史。

## 2026-07-10 - Task: 备份并合并上游 v0.1.150

### What was done
- 在合并前创建并推送备份分支 `backup/main-before-v0.1.150-20260710-e15ae8cd`，保存本地 i18n 修复后的安全回滚点。
- 从 `https://github.com/Wei-Shaw/sub2api.git` 获取并合并标签 `v0.1.150`，手工解决 3 个冲突文件。
- 冲突解决保留了本地 `request_type` 数字兼容解析、OpenAI compact 非流式响应 stop 回调，同时合入上游 GPT-5.6 定价、缓存 token 解析与其他 v0.1.150 功能变更。

### Testing
- 通过：`go test ./internal/handler/admin ./internal/service ./internal/server`（使用 `.cache/go-build` 与 `.cache/go-mod` 本地忽略缓存）。
- 通过：`go test ./...`（backend，全量包测试通过）。
- 通过：`pnpm test:run`（153 files / 1027 tests passed）。
- 通过：`pnpm typecheck`。
- 通过：`pnpm lint:check`。
- 通过：`pnpm build`，仅有既有 Browserslist 过期、Vite chunk 与动态/静态 import 警告。

### Notes
- 冲突文件：`backend/internal/handler/admin/dashboard_handler.go`、`backend/internal/service/openai_gateway_response_handling.go`、`backend/internal/service/pricing_service.go`。
- `.cache/` 已在 `.gitignore` 中，用于本机 Go 测试缓存，不纳入提交。
- 回滚方式：代码层优先使用 `git revert -m 1 <合并提交>` 回退上游合并；也可从远端备份分支 `backup/main-before-v0.1.150-20260710-e15ae8cd` 恢复，禁止强推。

## 2026-07-10 - Task: 构建并推送 v0.1.150 腾讯云生产镜像

### What was done
- 以远端 `main` 合并提交 `2059b628` 为源码，构建并推送腾讯云 CCR 镜像。
- 因本机 Docker 缺少 buildx，使用 `.cache/Dockerfile.no-buildkit` 临时副本构建；临时副本只移除 pnpm BuildKit cache mount，其余生产 Dockerfile 内容保持一致。
- 推送可追溯标签：`ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.150-2059b628-20260710075827`。
- 同步推送：`ccr.ccs.tencentyun.com/apophis-chat/sub2api:v0.1.150` 与 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:latest`。

### Testing
- 通过：Docker 多阶段构建；首次 `postgres:18-alpine` 拉取因 Docker Hub OAuth 网络超时失败，单独拉取后复用缓存完成最终镜像。
- 通过：镜像版本 `Sub2API 0.1.150 (commit: 2059b628, built: 2026-07-10T07:58:27Z)`。
- 通过：默认入口运行用户为 `uid=1000(sub2api)`，包含 `psql 18.4`、`pg_dump 18.4` 与 `/app/resources`。
- 通过：隔离 PostgreSQL/Redis 安装验证，Docker health 为 `healthy`，`/health` 返回 `{"status":"ok"}`。
- 通过：三个远端标签均为 `linux/amd64`，manifest digest 一致：`sha256:b65a42505e72d66fc4c3e30435006149520fbf7146ccb92251ff5876dd4dec6e`。
- 发布前 `latest` 回滚参考：config digest `sha256:2b9046b777eda163c8ec0eb4770ef932621e67b53ddfd23aafe5d1862c5fc78e`。

### Notes
- 腾讯云镜像仓库：`ccr.ccs.tencentyun.com/apophis-chat/sub2api`，未输出或写入任何仓库凭据。
- 上游 `v0.1.150` 标签合入后 `backend/cmd/server/VERSION` 仍为 `0.1.149`，本次 Docker build 显式传入 `VERSION=0.1.150`，镜像内版本已验证正确。
- `.cache/Dockerfile.no-buildkit` 与 Go 测试缓存均为本地忽略临时产物，不纳入提交。
- 回滚方式：部署端可回切到发布前 `latest` 参考镜像，或改用可追溯旧标签；代码层使用 `git revert` 正常回退，禁止强推。

## 2026-07-10 - Task: 修复账号筛选批量删除文案未格式化

### What was done
- 补齐账号列表“删除筛选结果”流程缺失的中英文 i18n 文案，避免界面直接显示 `admin.accounts.bulkDeleteFilteredTitle`、`admin.accounts.bulkDeleteFilteredButton` 等原始 key。
- 同步补齐筛选删除确认提示和空结果提示，并将相关 key 纳入语言包可见文本回归测试。

### Testing
- 通过：`pnpm test:run src/i18n/__tests__/navigationFeatureLocales.spec.ts src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts`（2 files / 78 tests passed）。
- `pnpm typecheck` 未通过，错误来自本轮开始前已有改动 `frontend/src/components/layout/AppHeader.vue`：模板引用的 `formatHeaderMoney`、`availableBalance`、`frozenBalance`、`balanceFrozenText` 未定义；本轮未修改该文件。

### Notes
- `frontend/src/i18n/locales/zh/admin/accounts.ts`：新增筛选批量删除的中文按钮、标题、确认和空结果文案。
- `frontend/src/i18n/locales/en/admin/accounts.ts`：新增对应英文文案。
- `frontend/src/i18n/__tests__/navigationFeatureLocales.spec.ts`：新增四个筛选批量删除 key 的存在性与非空校验。
- 回滚方式：还原上述三个文件的本轮变更，并删除 `progress.md` 本节即可。

## 2026-07-10 - Task: 备份并合并上游 v0.1.151

### What was done
- 合并前复核 `AppHeader.vue` 与 `AppSidebar.vue` 均不存在相对 `HEAD` 的未提交差异；本地 `main` 与 `origin/main` 当时同步。
- 创建并推送保护分支 `backup/main-before-v0.1.151-20260710-cfab0bb3`，固定合并前提交 `cfab0bb3`。
- 通过临时 fetch ref 获取上游 `v0.1.151`，并以本地 merge commit `be070cd1` 合入 `main`；合并无冲突，未推送 `main`。
- 合入 OpenAI Fast/Flex 用户级策略、Codex identity pairing、GPT-5.6 计费与用量修复、setup-token 自动刷新、Grok reasoning effort 与 Codex image_gen 修复等上游发布内容。

### Testing
- 通过：`GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./internal/pkg/openai ./internal/pkg/apicompat ./internal/server/middleware ./internal/service ./internal/repository`。
- 通过：`pnpm typecheck`。
- 通过：`pnpm vitest run src/components/keys/__tests__/UseKeyModal.spec.ts src/composables/__tests__/useModelWhitelist.spec.ts`（2 files / 20 tests passed）。
- 通过：`git diff --check HEAD^1 HEAD`；并确认 `upstream-v0.1.151` 已是 merge commit 的祖先。

### Notes
- `backend/`、`frontend/src/`、`backend/migrations/173_allow_cyber_blocked_usage_request_type.sql`、`backend/resources/model-pricing/model_prices_and_context_window.json`：合入上游 v0.1.151 的 69 个文件变更。
- `backup/main-before-v0.1.151-20260710-cfab0bb3`：合并前安全回滚点，已推送至 `origin`。
- 回滚方式：使用 `git revert -m 1 be070cd1` 回退本次上游合并；或从远端备份分支恢复。禁止使用 reset 或 force push。

## 2026-07-10 - Task: 推送 v0.1.151 main 并发布腾讯云 Docker 镜像

### What was done
- 将已验证的 `main` 推送至 `origin`，远端提交为 `92400528`。
- 构建并推送腾讯云 CCR 可追溯镜像 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.151-92400528-20260710191539`。
- 同步推送 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:v0.1.151` 与 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:latest`。

### Testing
- 通过：Docker 多阶段生产构建；本机缺少 buildx，临时使用仅移除 pnpm BuildKit cache mount 的 `.cache/Dockerfile.no-buildkit`，正式 Dockerfile 未改动。
- 通过：镜像运行时输出 `Sub2API 0.1.151 (commit: 92400528)`。
- 通过：默认入口执行 `--version`；镜像包含 `/app/resources`、可执行 `/app/sub2api`、`psql 18.4` 与 `pg_dump 18.4`。
- 通过：三个远端标签均为 `linux/amd64`，manifest digest 一致：`sha256:ccb7a556ecd71c4b9b0839d674e42dbba2a2cd31c6348233efdea350c8a8a79d`。

### Notes
- `Dockerfile`：保持未修改；`.cache/Dockerfile.no-buildkit` 为已忽略的本机临时构建文件。
- 先推送可追溯标签和 `v0.1.151`，成功后才更新 `latest`。
- 回滚方式：部署端可回切到上一版 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:v0.1.150`（此前 manifest digest `sha256:b65a42505e72d66fc4c3e30435006149520fbf7146ccb92251ff5876dd4dec6e`）；代码层按需执行 `git revert`，禁止 force push。

## 2026-07-13 - Task: 全面修复网站缺失国际化文案

### What was done
- 恢复账号数据导入的文件提示、分组标签与分组说明，并补齐账号、模型探测、个人资料、渠道、运维和系统设置等全站缺失的中英文文案。
- 对齐中英文语言包叶子结构与插值占位符，补齐历史上仅存在于单一语言的文案。
- 新增生产代码静态翻译键扫描门禁，防止后续功能合并后再次显示 `admin.*` 等原始键；修正导入测试将原始键误当成功结果的问题。

### Testing
- `pnpm test:run`：通过，154 个测试文件、1039 项测试。
- `pnpm typecheck`：通过。
- `pnpm lint:check`：通过。
- `pnpm build`：通过；仅有既有动态/静态导入及大 chunk 警告。
- `pnpm exec vitest run src/i18n/__tests__/localeCompleteness.spec.ts`：通过，覆盖静态引用完整性、动态键命名空间、双语叶子对称和插值占位符对称。

### Notes
- `frontend/src/i18n/locales/{zh,en}`：补齐全站缺失及不对称文案，未修改业务请求和导入 API。
- `frontend/src/i18n/__tests__/localeCompleteness.spec.ts`：新增全站静态键与双语结构门禁；动态拼接键前缀不作为静态键误报。
- `frontend/src/__tests__/integration/data-import.spec.ts`：使用严格文案字典，未知键直接失败，并断言用户可见导入文案。
- 回滚方式：还原本轮语言包与测试文件增量，并删除本条 `progress.md` 记录；无需数据库或配置回滚。

## 2026-07-13 - Task: 合并上游 v0.1.152

### What was done
- 将上游 `v0.1.152` 合入本地 `main`，保留本地全站 i18n 完整性修复与自动检查。
- 解决 API Key 认证缓存快照版本及 Fast/Flex 用户搜索文案冲突，采用 v0.1.152 的缓存 v15 与邮箱搜索选择器实现。
- 引入上游 writer 生命周期防护、Codex/Grok 路由与协议兼容、缓存用量、网页搜索按次计费等修复。

### Testing
- `GOCACHE="C:/Users/MilkFoam/Desktop/AI/sub2api/.gocache" go test ./...`：通过。
- `pnpm test:run`：通过，157 个测试文件、1060 项测试。
- `pnpm typecheck && pnpm lint:check`：通过。
- i18n 完整性与 OpenAI Fast/Flex 文案专项测试：8 项通过。

### Notes
- `backend/internal/service/api_key_auth_cache_impl.go`：认证缓存快照版本更新为 v15。
- `frontend/src/i18n/locales/{zh,en}/admin/settings.ts`：保留上游 Fast/Flex 邮箱搜索选择器文案并通过本地完整性门禁。
- 回滚方式：使用 `git revert -m 1 <v0.1.152 merge commit>` 回退本次合并；不使用 reset 或 force push。

## 2026-07-13 - Task: 注入 v0.1.152 运行时版本

### What was done
- 将服务端版本文件从上游遗留的 `0.1.151` 更新为发布版本 `0.1.152`，确保本地构建和未显式传参的发布流程使用正确版本。

### Testing
- Docker 多阶段生产构建通过；本机缺少 buildx，使用已忽略的 `.cache/Dockerfile.no-buildkit` 移除 BuildKit cache mount，正式 `Dockerfile` 未修改。
- 镜像运行时输出 `Sub2API 0.1.152 (commit: f84d96f7, built: 2026-07-13T08:25:42Z)`。
- 镜像为 `linux/amd64`，包含 `/app/resources`、`psql 18.4`、`pg_dump 18.4` 和非 root 用户 `sub2api`。
- 三个腾讯云标签的远端 manifest digest 一致：`sha256:d7a5f26cde3cc905b77d1866677de81c7e474b00896dfdfdd5928101756c2518`。

### Notes
- `backend/cmd/server/VERSION`：更新为 `0.1.152`。
- 已发布 `0.1.152-f84d96f7-20260713082542`、`v0.1.152` 与 `latest`。
- 回滚方式：部署端回切 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:v0.1.151`；代码层回退版本提交，不影响数据库和业务数据。

## 2026-07-13 - Task: 修复上游余额不足错误处理回归

### What was done
- 恢复上游错误码解析对顶层 `code`、`detail.code` 及内嵌 JSON 同类字段的支持，修复 `INSUFFICIENT_BALANCE` 被普通 403 策略跳过的问题。
- 统一余额不足错误文案生成，并让 OpenAI 管理员 Responses、Chat Completions、Compact 与图片测试入口在命中余额不足时将账号标记为错误且停止调度；普通 403、401 与 429 行为保持不变。
- 扩展池模式、限流服务和管理员测试连接回归用例，覆盖顶层、嵌套、detail 与字符串内嵌错误格式。

### Testing
- 定向 TDD 回归测试：修复前按预期失败；修复后通过。
- `GOCACHE="$PWD/.cache/go-build" go test -tags=unit ./internal/service -run 'TestRateLimitService_HandleUpstreamError_OpenAIInsufficientBalanceDisablesPoolModeAccount|TestShouldRetryPoolModeOnSameAccount_SkipsInsufficientBalance|TestAccountTestService_OpenAI(InsufficientBalanceSetsPermanentError|ChatCompletionsInsufficientBalanceSetsPermanentError|Ordinary403DoesNotSetPermanentError|401SetsPermanentErrorOnly|429PersistsSnapshotAndRateLimitState)' -count=1`：通过。
- `GOCACHE="$PWD/.cache/go-build" go test ./...`：通过。
- `GOCACHE="$PWD/.cache/go-build" go test -tags=unit ./internal/service -count=1`：余额相关测试通过；全量存在 2 个与本轮无关的既有失败，分别为流式错误 failover 断言与图片 reasoning effort 设置校验。
- `golangci-lint run ./...`：未执行，当前环境未安装 `golangci-lint`。

### Notes
- `backend/internal/service/gateway_upstream_response.go`：仅补回上游拆分时遗漏的错误码解析分支，未回退或改写合并后的文件结构。
- `backend/internal/service/upstream_insufficient_balance.go`、`ratelimit_service.go`：复用统一错误文案，保持正常网关的调度阻断行为。
- `backend/internal/service/account_test_service.go`：各 OpenAI 管理员测试入口命中余额不足时复用同一逻辑写入专用错误状态。
- `backend/internal/service/*_test.go`：新增多格式余额不足与普通 403 不误判测试。
- 回滚方式：还原本条记录及上述 7 个后端文件的本轮增量；无需数据库迁移或配置回滚。

## 2026-07-13 - Task: 合并上游 v0.1.153

### What was done
- 将上游正式 Release `v0.1.153`（tag commit `a2bc1337`）合入本地 `main`，引入 Grok 视频编辑/延长、Apple 容器部署、OpenAI 订阅档位覆盖、调度与用量统计等上游修复。
- 解决中文管理概览文案冲突，并修正自动合并产生的 `claudeMaxSimulation` 错误嵌套和 `allowUserRefund` 重复键；保留本地中文文案及完整性门禁。
- 完整保留本地 OpenAI 上游余额不足解析、账号阻断、管理员测试连接状态写入及其回归测试。
- 将运行时版本文件同步为 `0.1.153`，确保后续本地及 Docker 构建使用正确版本。

### Testing
- `GOCACHE="$PWD/.cache/go-build" go test ./...`：通过。
- 余额不足专项 unit 回归测试：通过。
- `pnpm test:run`：通过，159 个测试文件、1083 项测试。
- `pnpm typecheck && pnpm lint:check`：通过。
- i18n 完整性与重复键专项测试：12 项通过。
- `pnpm build`：通过；仅有既有动态/静态导入与大 chunk 警告。

### Notes
- `frontend/src/i18n/locales/zh/admin/overview.ts`：冲突采用本地自然中文表达，同时保留上游其他新增内容。
- `frontend/src/i18n/locales/zh/misc.ts`：移除自动合并后的同名重复键，保留与英文一致的正确层级。
- `backend/cmd/server/VERSION`：从 `0.1.152` 更新为 `0.1.153`。
- 回滚方式：使用 `git revert -m 1 <本次 v0.1.153 merge commit>` 回退合并；迁移 `174_add_usage_logs_api_key_latest_ip_index_notx.sql` 仅新增索引，部署前回滚不涉及数据库。

## 2026-07-23 - Task: 接入 NewAPI 用户真实余额

### What was done
- 为 OpenAI API Key 账号新增可选的 NewAPI Dashboard Access Token/PAT 与数字用户 ID 配置；Access Token 纳入统一敏感凭据脱敏与编辑保留机制，不会通过账号 DTO 返回前端。
- 余额探测配置 PAT 后并发请求同源 `/api/user/self` 与 `/api/status`，复用账号代理、TLS 指纹、超时、禁重定向和上游 URL 安全策略；认证、响应或换算失败时关闭失败，不回退模型 API Key 的 Token 额度。
- 将 NewAPI `quota` 按 `quota_per_unit` 换算为真实余额，支持负余额；钱包 Tooltip 仅显示累计已用，不把累计消费与当前余额相加误称固定上限。
- 管理端余额主值和 Tooltip 数值统一保留一位小数且不展示 USD、CNY、quota 等单位；无限 Token 不再显示为无限或负余额。

### Testing
- `GOCACHE="C:/Users/MilkFoam/AppData/Local/Temp/sub2api-go-cache" go test -p 1 ./...`：后端全量通过。
- `pnpm test:run`：前端全量 199 个测试文件、1413 项测试通过。
- NewAPI 余额、凭据脱敏/合并、创建/编辑表单、余额组件和 i18n 定向测试：通过。
- `pnpm run build`：Vue/TypeScript 类型检查与 Vite 生产构建通过；仅保留既有 Browserslist、动态/静态 import 和大 chunk 警告。
- 后端独立可执行文件已编译到系统临时目录；`git diff --check`、相关 ESLint 与独立安全审查通过。

### Notes
- `newapi_access_token` 只用于 `/api/user/self`；`/api/status` 不携带该凭据，响应体和错误快照不记录 token。
- 编辑时留空会保留已有 PAT；当前 UI 不提供显式删除 PAT 入口，可通过替换 PAT 轮换凭据。
- 未连接真实 NewAPI 多版本实例，协议兼容性由新旧 `/api/status` 结构和失败关闭测试覆盖；部署后仍建议用目标 NewAPI 实例做一次人工余额核对。
- 回滚方式：对本次功能提交执行 `git revert <commit>`；无数据库迁移、生产配置或镜像变更。

## 2026-07-23 - Task: 修复余额自动刷新与设置保存联动

### What was done
- 将同一周期内的上游倍率探测与上游余额探测改为并发启动，避免慢倍率请求串行阻塞余额自动刷新；两条任务仍保留各自的开关、到期筛选、分布式锁和共享并发上限。
- 将设置页“上游倍率自动探测”和“上游余额自动探测”的保存状态拆分，点击其中一个保存按钮时不再让另一个按钮同步禁用或显示加载。
- 新增慢倍率请求不阻塞余额刷新的后端回归测试，以及两个保存按钮 loading 状态互不影响的前端回归测试。

### Testing
- `go test -p 1 ./...`：后端全量通过。
- `pnpm test:run`：前端全量 199 个测试文件、1414 项测试通过。
- `pnpm run build`：Vue/TypeScript 类型检查与 Vite 生产构建通过；仅保留既有 Browserslist、动态/静态 import 和大 chunk 警告。
- 相关 ESLint、`gofmt`、`git diff --check` 与定向回归测试通过。
- `go test -race` 未能执行：当前 Windows 环境没有 GCC，Go race detector 要求 CGO；普通并发回归测试和后端全量测试均已通过。

### Notes
- 修复不改变余额探测间隔、账号启用条件、API 合同或凭据处理，仅消除两条周期任务之间的串行等待和前端共享状态。
- 回滚方式：对本次修复提交执行 `git revert <commit>`；无数据库迁移或配置回滚。

## 2026-07-23 - Task: 发布最新余额修复腾讯云镜像

### What was done
- 基于已推送且全量验证通过的提交 `74bcb6d8`，使用生产根 Dockerfile 的等价 no-BuildKit 文件构建 `linux/amd64` release 镜像。
- 推送腾讯云可追溯标签 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.162-74bcb6d8-20260722152041`，随后更新 `latest`。
- 发布前先记录旧 `latest` 摘要，并在隔离 PostgreSQL/Redis 环境完成自动初始化、迁移及健康检查。

### Testing
- 镜像版本输出：`Sub2API 0.1.162 (commit: 74bcb6d8, built: 2026-07-22T15:20:41Z)`。
- 运行用户为 `uid=1000(sub2api)`；`psql`、`pg_dump` 均为 PostgreSQL `18.4`；`/app/resources`、可执行文件与 Docker healthcheck 均存在。
- 隔离容器健康状态为 `healthy`，`/health` 返回 `{"status":"ok"}`；测试结束后仅清理本轮临时容器和网络。
- 可追溯标签与新 `latest` 推送均返回摘要 `sha256:f73245417272a2a7f50c0e192447b32f746690d65ad31597fc72ca540528a905`；可追溯标签重新拉取核验为同一摘要。

### Notes
- 发布前 `latest` 回滚摘要为 `sha256:1e1bfccd35b9ed035d4ff0f0079fcc0eb8bcd7db481cae9965312d88d1a494d2`。
- 本机仍使用 Docker legacy builder，构建上下文约 3.0GB；这些是后续构建效率优化项，不影响本次镜像完整性或运行正确性。
- 回滚方式：部署端回切到发布前摘要；代码层正常 `git revert 74bcb6d8a` 和 `git revert 6eb9a817d`，禁止强推。

## 2026-07-23 - Task: 最终发布审查与 NewAPI PAT 传输加固

### What was done
- 对余额后端协议、安全、并发、数据一致性及前端展示、交互、凭据脱敏、i18n 和设置保存执行独立只读复核；确认账号 DTO 不会返回 `newapi_access_token` 明文。
- 修复最终审查发现的发布阻断安全问题：当账号配置 NewAPI Dashboard PAT 时，控制面余额探测不再继承通用上游的 `allow_insecure_http`，而是强制使用 HTTPS，并在任何出站请求前拒绝包含 URL userinfo 的地址。
- 新增安全回归测试，证明 HTTP/userinfo 目标返回 `invalid_base_url`，且不会触发任何上游请求；修复提交 `3bcb1486a` 已推送到 `origin/main`。
- 基于安全修复提交重建 `linux/amd64` 镜像，推送可追溯标签 `0.1.163-3bcb1486-20260723051001`，并重新指向 `v0.1.163` 与 `latest`；Task 24 的旧 `0.1.163` 摘要已被本次安全版取代。
- 清理隔离验证容器、网络和命名卷；未修改生产配置、密钥、数据库或线上实例。

### Validation
- 新增安全测试先失败，确认明文 PAT 请求问题真实存在；修复后目标测试、`go test ./internal/service -count=1`、`go vet ./internal/service` 和后端 `go test ./...` 全量通过。
- 前端最终基线验证通过：199 个测试文件、1417 项测试通过；Vue/TypeScript 检查、Vite 生产构建和全量 ESLint 通过。构建仅保留既有 Browserslist、动态/静态 import 与大 chunk 警告。
- 独立安全复核确认原高风险已关闭，未发现仍存在的高风险或阻断发布问题。
- 镜像版本输出：`Sub2API 0.1.163 (commit: 3bcb1486, built: 2026-07-23T05:10:01Z)`；默认入口以 `uid=1000(sub2api)` 执行，`psql`/`pg_dump` 为 18.4，镜像包含健康检查并为 `linux/amd64`。
- 隔离 PostgreSQL 18 + Redis 8 环境自动初始化、迁移和启动成功，容器达到 healthy，`/health` 返回 `{"status":"ok"}`。
- 腾讯云远端三个标签的 manifest 均为 `linux/amd64`，摘要一致：`sha256:1a1249c87871419b03ae2348481d7e56931d60e09342816f57f3daea36786eb2`。
- Git 核验：安全修复基线 `3bcb1486aec961c8a766ef55668291770cfdfda7` 已同步到 `origin/main`；本次收尾记录单独提交，不改变镜像代码内容。

### Server upgrade
1. 升级前备份数据库，并记录当前应用镜像摘要；推荐保留 `0.1.162-74bcb6d8-20260722152041` 作为功能级回滚点。
2. 将部署镜像设置为不可变标签 `ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.163-3bcb1486-20260723051001`；也可使用 `v0.1.163`，但不可变标签更便于审计。
3. 执行 `docker pull ccr.ccs.tencentyun.com/apophis-chat/sub2api:0.1.163-3bcb1486-20260723051001`，随后用现有 Compose 配置重建应用服务，不需要数据库迁移参数或新增环境变量。
4. 升级后核对容器版本、`/health`、登录、账号列表倍率/余额列，以及目标 NewAPI 实例的真实余额换算。
5. 安全兼容提示：使用 Dashboard PAT 的 NewAPI 余额探测现在强制要求 HTTPS；仍使用 HTTP 控制面地址的账号会失败关闭且不会发送 PAT，升级前应先为 NewAPI 配置 HTTPS。

### Risks / rollback
- 无已知高风险或发布阻断问题。未连接真实 NewAPI 多版本实例，部署后仍需用目标实例人工核对一次余额；Windows 环境缺少 GCC，Go race detector 未执行。
- 非阻断审查项包括：极端临时故障下的 PAT 快照保留分类、批量轮换 PAT/用户 ID 的旧快照即时失效、极少见的设置并发保存竞态、长周期 leader lock 续租，以及触屏/键盘 Tooltip 等可访问性增强；这些不造成凭据明文响应泄露或当前发布阻断，留待后续独立任务处理。
- 本机缺少 buildx，使用生产根 Dockerfile 的等价 legacy-builder 临时副本构建；约 3.0GB 构建上下文仅影响构建效率。
- 如升级失败，优先回切不可变标签 `0.1.162-74bcb6d8-20260722152041`（摘要 `sha256:f73245417272a2a7f50c0e192447b32f746690d65ad31597fc72ca540528a905`）；不建议回切 Task 24 的旧 `0.1.163` 摘要，因为其尚未包含本次 PAT HTTPS 加固。代码回滚使用正常 `git revert`，不得重写公共历史。
