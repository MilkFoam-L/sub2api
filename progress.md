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
