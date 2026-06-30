# 账号调度优化启发记录

## 背景

当前项目的账号调度已经具备运行时能力：按分组、平台、模型、账号状态、额度、RPM、窗口费用、粘性会话、并发槽位、等待队列和负载感知策略选择账号。调度核心偏向“请求进来时实时选账号”。

对比 S2A-Manager 后得到的核心启发是：不要用外部面板式逻辑替换当前运行时调度器，而应在现有调度器之上增加“账号自动治理层”和“成本/倍率驱动优先级层”。

## S2A-Manager 带来的有效启发

### 1. 账号治理层与运行时调度解耦

S2A-Manager 并不直接参与每次请求的账号选择，它通过 Admin API 修改目标 Sub2API 账号字段，例如：

- `schedulable`
- `priority`
- `concurrency`
- `load_factor`
- `rate_multiplier`
- `group_ids`
- `expires_at`
- `auto_pause_on_expired`

启发：当前项目可以保留请求级调度器，把自动检测、自动暂停、自动恢复、余额预警、倍率驱动优先级等能力作为前置治理层。

### 2. 调度不只追求平均分配

当前调度使用负载感知和 Weighted P2C，目标是避免账号过载并尽量均衡。但实际运营中，“平均”不一定最优，账号之间可能存在成本、稳定性、余额、限额、延迟、错误率、过期时间和模型能力差异。

启发：账号调度应从“平均分配”升级为“约束下的最优分配”：

```text
先过滤不可用账号
-> 再保证健康和并发安全
-> 再考虑成本/倍率/余额/重置窗口
-> 最后做负载均衡和防热点
```

### 3. 倍率驱动优先级

S2A-Manager 的优先级规则会按指定分组内账号倍率排序：倍率越低，写入的 `priority` 越小；相同倍率共享同一优先级档位。

启发：当前项目可以引入“自动优先级治理”，让低成本账号自动获得更高优先级，但运行时仍由现有调度器处理并发、sticky 和过载逃逸。

需要注意：`priority` 是账号全局字段，如果账号同时属于多个分组，不适合直接做每分组独立优先级。若未来要支持每分组独立成本策略，应新增分组维度的调度权重或覆盖规则，而不是只改账号全局 `priority`。

### 4. 主动健康检测与自动暂停

S2A-Manager 的上游检测逻辑是：定时测试账号，连续失败达到阈值后把账号 `schedulable=false`，暂停期间继续探活，成功后恢复 `schedulable=true`，暂停到期也会尝试恢复。

启发：当前项目可以新增主动探活机制，避免依赖真实用户请求来发现坏账号。这样能减少用户请求撞到故障账号的概率。

推荐方向：

```text
检测失败
-> 达到阈值
-> 进入临时不可调度或关闭 schedulable
-> 继续低频探活
-> 成功后恢复
-> 恢复初期设置试运行窗口，避免立刻吃满流量
```

### 5. 故障账号不应继续影响自动倍率

S2A-Manager 在账号暂停期间，可以临时排除该账号绑定的采集源分组参与目标分组倍率计算。

启发：若当前项目未来引入采集源倍率或账号成本联动，需要区分“数据存在”和“数据可信”。被暂停、连续失败、余额异常的账号，不应继续作为自动倍率或成本决策的有效输入。

### 6. 余额和成本应进入治理闭环

S2A-Manager 支持单账号余额阈值和余额 Webhook 预警。虽然这不直接改变运行时选择，但对运营调度很关键。

启发：余额低、即将过期、成本高、故障率高的账号，应在治理层改变调度参数，例如降低优先级、降低负载权重、进入降级池或暂停。

## 当前项目适合保留的能力

以下运行时能力应保留，不建议被 S2A-Manager 风格逻辑替换：

- session sticky 与 soft escape
- 模型路由和模型能力过滤
- Weighted P2C / legacy LRU
- 并发槽位抢占
- 等待队列和超时控制
- RPM、窗口费用、额度过滤
- 调度债务防热点
- 调度快照和 Redis 缓存机制

这些能力解决的是“某个具体请求此刻应该选谁”，外部治理层解决的是“账号参数应该维护成什么状态”。

## 可落地优化方向

### 阶段一：观测增强

目标：先确认当前平均分配为何不如预期。

建议增加或整理以下指标：

- 每账号请求数、成功数、失败数
- 每账号 TTFT / 总耗时 EWMA
- 每账号当前并发、等待队列、负载率
- 每账号被选中次数和调度债务
- 每账号 sticky 命中、sticky 逃逸次数
- 每账号因模型、RPM、窗口费用、额度、并发满被过滤次数
- 每账号单位成本或倍率

### 阶段二：成本/健康权重进入打分

目标：调度不只平均，还要优先使用低成本、健康、低延迟账号。

可考虑把运行时 cost 扩展为：

```text
cost = 负载成本
     + 等待队列成本
     + 调度债务成本
     + 错误率惩罚
     + 延迟惩罚
     + 成本/倍率惩罚
     + 余额或过期风险惩罚
```

注意：惩罚项应有上限，避免单一指标导致全部流量集中到少数账号。

### 阶段三：主动探活与自动恢复

目标：减少用户请求撞坏账号。

建议新增主动探活规则：

- 每账号可配置检测模型、检测间隔、失败阈值、暂停时长
- 失败后设置 `temp_unschedulable_until` 或 `schedulable=false`
- 暂停期间继续探活
- 成功后恢复，但先进入 warm-up 窗口，限制初期流量

### 阶段四：分组维度策略

目标：解决账号全局优先级无法满足不同分组策略的问题。

建议不要只复用账号全局 `priority`，而是设计分组维度策略：

- group-account 调度权重
- group-account 成本覆盖
- group-account 最大占比
- group-account 优先级覆盖

这样可以避免一个账号在 A 分组低成本、B 分组高风险时被同一个全局优先级误导。

## 不建议直接照搬的点

- 不建议把 S2A-Manager 的“按倍率直接写 priority”作为唯一调度逻辑。
- 不建议用定时任务替代请求级负载感知。
- 不建议只追求平均请求数，忽略错误率、延迟、成本、余额和限额。
- 不建议在没有观测数据前大改调度算法。

## 推荐总原则

```text
治理层负责让账号状态更准；
运行时调度器负责让每次请求更稳；
成本策略负责让长期资源使用更优。
```

因此，当前优化方向应是“保留现有调度器 + 增强健康/成本/观测/分组策略”，而不是替换为单纯轮询或单纯倍率排序。

## 联网调度器资料带来的补充启发

### 1. Envoy：保留 P2C，但补齐健康摘除和慢启动

Envoy 的 Least Request 在同权重节点下使用 Power of Two Choices：随机抽取 N 个候选，默认 N=2，再选择 active requests 最少的节点；这和当前项目的 Weighted P2C 方向一致。

Envoy 在权重不同时，会把配置权重和当前 active requests 合成动态权重：

```text
有效权重 = 配置权重 / (active_requests + 1) ^ active_request_bias
```

启发：当前 `cost = (当前并发 + 等待队列 + 调度债务) / capacity` 方向是合理的，但还可以引入 `active_request_bias` 这类可配置参数，控制“当前并发”对选路的影响强弱，避免高权重账号在已有并发时仍被过度命中。

Envoy 的 outlier detection 会把连续错误、网关错误、连接失败、超时等异常转成主机摘除，并用递增 ejection time 保护系统；慢启动会让新加入或恢复的节点逐步吃流量。

启发：当前项目已有 `rate_limit_reset_at`、`overload_until`、`temp_unschedulable_until` 等状态，但缺少统一的被动异常摘除策略和恢复慢启动。推荐新增：

```text
连续失败/高错误率/超时
-> 临时摘除账号
-> 摘除时长随连续摘除次数递增
-> 恢复后进入 slow start，逐步恢复 load_factor
```

### 2. LiteLLM：LLM 调度要把 RPM/TPM、冷却和 fallback 当成一等能力

LiteLLM Router 支持 `simple-shuffle`、`least-busy`、`usage-based-routing`、`latency-based-routing`、`cost-based-routing` 等策略，并把 `rpm`、`tpm`、`order`、fallback、cooldown、Redis 多实例共享用量放在调度配置里。

启发：当前项目已经有 RPM 检查和等待队列，但调度权重主要仍围绕并发/等待/优先级，建议把以下信号提升为统一调度输入：

- 每账号 RPM 剩余量
- 每账号 TPM 或 token 预算剩余量
- 429 后 cooldown
- order/priority 分层 fallback
- 失败类型差异：429、超时、5xx、模型不支持、内容策略、上下文超限应分开处理

不建议直接照搬 usage-based-routing 的重 Redis 热路径，但可以先做轻量版：在候选过滤阶段排除已耗尽 RPM/窗口预算账号，在评分阶段偏向剩余额度更健康的账号。

### 3. OpenRouter：价格、吞吐、延迟和隐私策略都应是路由约束

OpenRouter provider routing 支持 `order`、`allow_fallbacks`、`only`、`ignore`、`sort=price/throughput/latency`、`preferred_min_throughput`、`preferred_max_latency`、`max_price`、数据保留策略等字段。默认会优先低价且近期没有明显故障的 provider。

启发：当前项目可以把调度策略拆成“硬过滤 + 软排序”：

硬过滤：

- 模型能力
- 分组和平台
- 隐私要求
- 数据保留/合规要求
- 最大价格或倍率
- 账号状态、额度、RPM、窗口费用

软排序：

- 成本更低
- TTFT 更低
- 吞吐更高
- 错误率更低
- 余额更健康
- 负载更低

这能解释为什么“平均分配”不是最优：如果账号价格、速度和稳定性不同，应该优先选低价且健康的账号，而不是让每个账号请求数完全平均。

### 4. Netflix concurrency-limits：并发上限应可自适应

Netflix concurrency-limits 强调用 inflight 并发数而不是静态 RPS 来保护服务，并通过延迟、超时、拒绝等信号动态调整并发上限。

启发：当前账号 `concurrency` 是静态配置，`load_factor` 是静态容量权重。可以增加“有效并发上限”概念：

```text
effective_concurrency = min(静态 concurrency, 自适应并发上限)
```

当账号延迟升高或超时增加时，自动降低有效并发；恢复稳定后逐步升回去。这比简单平均分配更适合不稳定上游。

### 5. Kubernetes：调度流程应明确分层

Kubernetes 调度明确分为 Filter 和 Score，并在 Reserve/Permit/Bind 阶段处理资源预留和最终绑定。

启发：当前项目已经隐含类似流程，但建议文档化/结构化成：

```text
PreFilter：解析请求、模型、分组、sticky、路由规则
Filter：硬过滤不可用账号
Score：按负载、成本、健康、延迟、额度打分
Reserve：抢并发槽位，记录调度债务
Permit：等待队列、粘性会话限制、最终准入
Bind：绑定 session、返回账号、请求结束后释放槽位
```

这样后续添加健康分、成本分、慢启动和自适应并发时，不容易把硬约束和软偏好混在一起。

## 当前项目与外部资料的差距判断

当前项目已经具备优秀基础：

- Weighted P2C
- `load_factor` 容量权重
- 当前并发和等待队列
- 调度债务防热点
- sticky soft escape
- OpenAI 高级调度中的错误率和 TTFT
- RPM、窗口费用、额度过滤

主要差距在：

1. 普通网关没有像 OpenAI 高级调度那样统一引入错误率、TTFT、成本和健康分。
2. 没有系统化的被动 outlier ejection，失败账号主要靠已有状态字段或手工/上游错误路径处理。
3. 账号恢复后缺少 slow start，可能一恢复就重新吃到正常流量。
4. 静态并发和静态 load factor 无法反映账号实时退化。
5. 成本/倍率只是账号字段，没有进入普通网关 cost 公式。
6. 缺少按分组维度的调度覆盖规则，账号全局 priority 很难表达不同分组的成本/稳定性差异。

## 后续优先级建议

### 优先级 1：先增强观测，不直接重写算法

先补全每账号调度决策日志和指标，确认“不够好”具体是：

- 热点账号过热
- 低成本账号没优先用
- 高延迟账号仍在被选
- sticky 导致局部不均
- 高权重账号被打满
- 429 后 cooldown 不够及时
- 恢复账号吃流量太快

### 优先级 2：普通网关引入健康/延迟/成本评分

把 OpenAI 高级调度中的 TTFT、错误率思路下沉到普通调度，形成统一 cost：

```text
cost = load_cost
     + queue_cost
     + debt_cost
     + error_penalty
     + ttft_penalty
     + rate_multiplier_penalty
     + quota_risk_penalty
```

推荐以配置开关灰度启用，默认权重保守。

### 优先级 3：增加 outlier ejection + slow start

把失败账号从“立刻重试/靠请求撞出来”改为“临时摘除 + 递增冷却 + 主动恢复 + 慢启动”。

### 优先级 4：再考虑自适应并发

自适应并发收益高，但会影响运行时行为，建议在指标稳定后再做。先从只调整 `EffectiveLoadFactor` 或评分权重开始，避免直接改 Redis 实际并发槽位。

## 已落地：多目标调度第一阶段

本轮已把普通网关从“只看负载/等待/调度债务”的 cost，扩展为“健康 + 成本 + 延迟 + 额度 + 负载”的多目标评分。

### 已实现内容

1. 新增普通网关账号运行时统计：
   - 每账号错误率 EWMA。
   - 每账号延迟 EWMA。
   - 连续失败次数。
   - 最近成功/失败时间。

2. 扩展 Weighted P2C 同优先级 cost：

```text
cost = load_cost
     + queue_cost
     + debt_cost
     + error_rate_penalty
     + latency_penalty
     + rate_multiplier_penalty
     + quota_risk_penalty
```

3. 保留原有硬约束：
   - `priority` 仍然先分层。
   - `legacy_lru` 仍走旧逻辑。
   - 已超额度、限流、暂停、过载等仍由原过滤逻辑处理。

4. 接入普通网关请求结果回报：
   - Messages 路径。
   - Gemini 兼容路径。
   - Chat Completions 路径。
   - Responses 路径。

5. 增加轻量被动摘除：
   - 连续上游失败达到阈值后写入 `temp_unschedulable_until`。
   - 冷却时间随连续失败次数递增，最多 10 分钟。
   - 用户侧取消请求不会惩罚账号。

### 新增配置

```yaml
gateway:
  scheduling:
    score_weights:
      load: 1.0
      queue: 1.0
      debt: 1.0
      error_rate: 0.8
      latency: 0.4
      rate_multiplier: 0.6
      quota_risk: 0.3
    latency_baseline_ms: 15000
    quota_risk_threshold: 0.2
    max_score_penalty: 5.0
```

### 当前边界

- 本阶段未新增前端 UI。
- 本阶段未新增数据库表。
- 主动探活和暂停期间继续恢复检测尚未实现。
- slow start 只作为后续阶段保留，当前先通过错误率/延迟/临时摘除减少故障账号命中。
- 自适应并发未改 Redis 槽位语义，避免影响现有并发控制。

### 回滚方式

- 将新增 `score_weights` 中的 `error_rate`、`latency`、`rate_multiplier`、`quota_risk` 设为 `0`，即可接近旧的负载调度行为。
- 如需代码回滚，回退 `scheduler_runtime_stats.go`、`account_scheduler_policy.go`、`gateway_service.go`、Gateway handler 回报挂接和配置改动。

## 已落地：主动检测暂停与恢复 slow start

本轮第二阶段已把 S2A-Manager 风格的“主动检测、失败暂停、恢复后慢启动”接入现有定时账号测试体系。

### 已实现内容

1. 复用已有 `ScheduledTestRunnerService`：
   - 只对已配置的 scheduled test plan 生效。
   - 不新增全账号自动扫描 worker。
   - 不会自动创建探活计划。

2. 连续失败自动临时暂停：
   - 定时测试失败会保存结果。
   - Runner 读取该计划最近结果并计算连续失败数。
   - 连续失败达到阈值后调用 `SetTempUnschedulable`。
   - 暂停时长随连续失败次数递增，并受最大暂停时长限制。

3. 暂停期间继续探活：
   - Runner 不以账号当前可调度状态作为执行前置条件。
   - 已有定时测试计划到期仍会继续运行。

4. 成功探活自动恢复并慢启动：
   - `auto_recover=true` 且测试成功时，继续复用 `RecoverAccountAfterSuccessfulTest` 清理 error/rate-limit/temp-unsched 等状态。
   - 如果确实恢复了运行时状态，会标记账号进入 slow-start 窗口。
   - 调度 cost 在 slow-start 窗口内增加逐步衰减的软惩罚。

### 新增配置

```yaml
gateway:
  scheduling:
    active_probe:
      auto_pause_enabled: true
      failure_threshold: 3
      pause_duration: 10m
      pause_duration_max: 1h
    slow_start:
      enabled: true
      duration: 5m
      penalty: 1.0
```

### 行为边界

- 主动检测只来自已有定时测试计划；没有计划的账号不会被自动探活。
- 自动暂停写入的是 `temp_unschedulable_until`，不是永久关闭 `schedulable`。
- slow start 只影响同优先级层内评分，不改变 Redis 并发槽位。
- `strict` sticky 仍保持强粘性；`soft` sticky 可因 slow-start cost 明显更高而逃逸。

### 回滚方式

```yaml
gateway:
  scheduling:
    active_probe:
      auto_pause_enabled: false
    slow_start:
      enabled: false
```

如需代码回滚，回退 scheduled test runner 的暂停逻辑、runtime stats slow-start 字段、scheduler cost slow-start 惩罚、GatewayService slow-start marker、wire 注入、配置和测试改动。

## 后台调度策略设置

已新增后台调度策略设置入口，位于管理后台“系统设置 -> 网关”页顶部的“调度策略”卡片。管理员可以直接配置：

- 多目标评分权重：负载、等待队列、调度债务、错误率、延迟、账号倍率、额度风险。
- 调度阈值：延迟基准、额度风险阈值、最大单项惩罚。
- 粘性会话：sticky 模式、逃逸评分倍率、逃逸负载率。
- 主动探活暂停：连续失败自动暂停、失败阈值、基础暂停时长、最大暂停时长。
- 恢复慢启动：是否启用、慢启动时长、初始惩罚。

后台保存后写入已有 settings 表；运行时调度会优先读取 settings 中的覆盖值，并回退到配置文件或环境变量默认值。为避免热路径频繁读库，调度配置读取使用短 TTL 缓存；后台保存会刷新缓存。

说明：主动探活配置只作用于已有 scheduled test plan，不会自动扫描所有账号，也不会自动创建测试计划。

### 后台设置回滚

- 后台层：在页面恢复默认值并保存，或清空 settings 表中的 `gateway_scheduling_*` 设置项。
- 运行时：无后台设置时会自动回退到配置文件或环境变量。
- 前端层：回退系统设置页中的“调度策略”卡片和相关类型。
