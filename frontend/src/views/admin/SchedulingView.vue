<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">调度面板</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          集中管理账号调度策略、优先账号和最近调度日志。
        </p>
      </div>
      <div class="flex gap-2">
        <button type="button" class="btn btn-secondary btn-sm" @click="loadAll" :disabled="loading">刷新</button>
        <button type="button" class="btn btn-primary btn-sm" @click="saveConfig" :disabled="saving || loading">
          {{ saving ? '保存中...' : '保存调度配置' }}
        </button>
      </div>
    </div>

    <div v-if="error" class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300">
      {{ error }}
    </div>

    <section class="card p-6">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">当前调度顺序</h2>
      <div class="mt-4 grid gap-3 md:grid-cols-5">
        <div v-for="step in schedulingSteps" :key="step.title" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
          <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ step.title }}</div>
          <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ step.description }}</p>
        </div>
      </div>
      <p class="mt-4 text-sm text-amber-600 dark:text-amber-300">
        优先账号只在已通过硬过滤、匹配当前分组和模型的候选中生效，不会绕过禁用、限流、过载、额度、RPM 或模型限制。
      </p>
    </section>

    <section class="card p-6">
      <div class="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div class="flex-1">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">优先调度账号</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">为空时使用常规调度；选择后仅在同 priority 候选层内临时置顶。</p>
          <select v-model.number="config.preferred_account_id" class="input mt-4 max-w-xl">
            <option :value="0">不指定优先账号</option>
            <option v-for="account in accountOptions" :key="account.id" :value="account.id">
              #{{ account.id }} · {{ account.name }} · {{ account.platform }} · {{ account.status }} · priority {{ account.priority }}
            </option>
          </select>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" @click="config.preferred_account_id = 0">清空</button>
      </div>
    </section>

    <section class="card p-6">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">调度策略配置</h2>
      <div class="mt-6 space-y-6">
        <div>
          <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">评分权重</h3>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-4">
            <label class="field">负载<input v-model.number="config.score_weights.load" type="number" min="0" step="0.1" class="input mt-1" /></label>
            <label class="field">等待队列<input v-model.number="config.score_weights.queue" type="number" min="0" step="0.1" class="input mt-1" /></label>
            <label class="field">调度债务<input v-model.number="config.score_weights.debt" type="number" min="0" step="0.1" class="input mt-1" /></label>
            <label class="field">错误率<input v-model.number="config.score_weights.error_rate" type="number" min="0" step="0.1" class="input mt-1" /></label>
            <label class="field">延迟<input v-model.number="config.score_weights.latency" type="number" min="0" step="0.1" class="input mt-1" /></label>
            <label class="field">账号倍率<input v-model.number="config.score_weights.rate_multiplier" type="number" min="0" step="0.1" class="input mt-1" /></label>
            <label class="field">额度风险<input v-model.number="config.score_weights.quota_risk" type="number" min="0" step="0.1" class="input mt-1" /></label>
            <label class="field">最大单项惩罚<input v-model.number="config.max_score_penalty" type="number" min="0" step="0.1" class="input mt-1" /></label>
          </div>
        </div>

        <div>
          <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">阈值与粘性</h3>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-4">
            <label class="field">延迟基准 ms<input v-model.number="config.latency_baseline_ms" type="number" min="1" step="1000" class="input mt-1" /></label>
            <label class="field">额度风险阈值<input v-model.number="config.quota_risk_threshold" type="number" min="0" max="1" step="0.01" class="input mt-1" /></label>
            <label class="field">粘性模式<select v-model="config.sticky_session_mode" class="input mt-1"><option value="soft">soft</option><option value="strict">strict</option><option value="off">off</option></select></label>
            <label class="field">逃逸倍率<input v-model.number="config.sticky_escape_score_ratio" type="number" min="1" step="0.05" class="input mt-1" /></label>
            <label class="field">逃逸负载率<input v-model.number="config.sticky_escape_load_rate" type="number" min="0" max="100" step="1" class="input mt-1" /></label>
          </div>
        </div>

        <div class="rounded-lg border border-amber-200 bg-amber-50/60 p-4 dark:border-amber-700 dark:bg-amber-900/10">
          <div class="mb-4 flex items-center justify-between gap-4">
            <div>
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">上游倍率软信号</h3>
              <p class="text-xs text-gray-600 dark:text-gray-300">默认关闭。开启后仅作为同 priority 层内 Weighted P2C 的软成本信号，不会绕过硬过滤。</p>
            </div>
            <Toggle v-model="config.upstream_rate.enabled" />
          </div>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-4">
            <label class="field">快照过期秒数<input v-model.number="config.upstream_rate.stale_ttl_seconds" type="number" min="1" step="60" class="input mt-1" /></label>
            <label class="field">倍率权重<input v-model.number="config.upstream_rate.rate_weight" type="number" min="0" step="0.1" class="input mt-1" /></label>
            <label class="field">可用率权重<input v-model.number="config.upstream_rate.health_weight" type="number" min="0" step="0.1" class="input mt-1" /></label>
            <label class="field">最低可用率<input v-model.number="config.upstream_rate.min_success_rate" type="number" min="0" max="1" step="0.01" class="input mt-1" /></label>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="mb-4 flex items-center justify-between gap-4">
              <div>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">主动探活暂停</h3>
                <p class="text-xs text-gray-500 dark:text-gray-400">连续失败后临时退出调度。</p>
              </div>
              <Toggle v-model="config.active_probe.auto_pause_enabled" />
            </div>
            <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <label class="field">失败阈值<input v-model.number="config.active_probe.failure_threshold" type="number" min="1" step="1" class="input mt-1" /></label>
              <label class="field">基础暂停<input v-model="config.active_probe.pause_duration" type="text" class="input mt-1" placeholder="10m" /></label>
              <label class="field">最大暂停<input v-model="config.active_probe.pause_duration_max" type="text" class="input mt-1" placeholder="1h" /></label>
            </div>
          </div>

          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="mb-4 flex items-center justify-between gap-4">
              <div>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">恢复慢启动</h3>
                <p class="text-xs text-gray-500 dark:text-gray-400">账号恢复后降低初期流量冲击。</p>
              </div>
              <Toggle v-model="config.slow_start.enabled" />
            </div>
            <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <label class="field">慢启动时长<input v-model="config.slow_start.duration" type="text" class="input mt-1" placeholder="5m" /></label>
              <label class="field">初始惩罚<input v-model.number="config.slow_start.penalty" type="number" min="0" step="0.1" class="input mt-1" /></label>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="card p-6">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">上游倍率源与可用率</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">类似渠道状态，检测上游倍率接口、同步快照和 1 小时可用率；默认只展示。</p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" @click="loadUpstreamRates" :disabled="upstreamLoading">刷新上游状态</button>
      </div>

      <div class="mt-4 grid grid-cols-1 gap-3 md:grid-cols-6">
        <label class="field md:col-span-2">源名称<input v-model="sourceForm.name" class="input mt-1" placeholder="例如 NewAPI A" /></label>
        <label class="field">类型<select v-model="sourceForm.source_type" class="input mt-1"><option value="sub2api">Sub2API</option><option value="newapi">NewAPI</option></select></label>
        <label class="field md:col-span-2">Base URL<input v-model="sourceForm.base_url" class="input mt-1" placeholder="https://example.com" /></label>
        <label class="field">Token<input v-model="sourceForm.token" class="input mt-1" type="password" placeholder="留空不修改" /></label>
        <label class="field">充值倍率<input v-model.number="sourceForm.recharge_multiplier" class="input mt-1" type="number" min="0" step="0.01" /></label>
        <label class="field">同步间隔秒<input v-model.number="sourceForm.sync_interval_seconds" class="input mt-1" type="number" min="15" step="60" /></label>
        <label class="field">认证<select v-model="sourceForm.auth_mode" class="input mt-1"><option value="bearer_token">Bearer Token</option><option value="none">无认证</option></select></label>
        <div class="flex items-end gap-3">
          <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300"><input v-model="sourceForm.enabled" type="checkbox" />启用</label>
          <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300"><input v-model="sourceForm.use_for_scheduling" type="checkbox" />可参与调度</label>
        </div>
        <div class="flex items-end gap-2">
          <button type="button" class="btn btn-primary btn-sm" @click="saveSource" :disabled="upstreamSaving">{{ editingSourceId ? '更新源' : '新增源' }}</button>
          <button v-if="editingSourceId" type="button" class="btn btn-secondary btn-sm" @click="resetSourceForm">取消</button>
        </div>
      </div>

      <div v-if="upstreamMessage" class="mt-4 rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm text-blue-700 dark:border-blue-800 dark:bg-blue-900/20 dark:text-blue-300">
        {{ upstreamMessage }}
      </div>

      <div class="mt-6 overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
          <thead class="text-left text-xs uppercase text-gray-500 dark:text-gray-400">
            <tr>
              <th class="px-3 py-2">源</th>
              <th class="px-3 py-2">状态</th>
              <th class="px-3 py-2">快照/绑定</th>
              <th class="px-3 py-2">1小时可用率</th>
              <th class="px-3 py-2">最后同步</th>
              <th class="px-3 py-2">最近错误</th>
              <th class="px-3 py-2">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="item in upstreamOverview" :key="overviewId(item)" class="text-gray-700 dark:text-gray-200">
              <td class="px-3 py-2">
                <div class="font-medium">{{ overviewName(item) }}</div>
                <div class="text-xs text-gray-500">{{ overviewType(item) }} · {{ overviewBaseUrl(item) }}</div>
              </td>
              <td class="px-3 py-2">
                <div>{{ overviewEnabled(item) ? '启用' : '停用' }} · {{ overviewUseForScheduling(item) ? '可参与调度' : '仅展示' }}</div>
                <div class="text-xs text-gray-500">Token：{{ overviewTokenConfigured(item) ? '已配置' : '未配置' }}</div>
              </td>
              <td class="px-3 py-2">{{ overviewSnapshotCount(item) }} / {{ overviewBindingCount(item) }}</td>
              <td class="px-3 py-2">{{ formatRate(overviewHealthRate(item)) }}<div class="text-xs text-gray-500">{{ overviewLatency(item) ?? '-' }} ms</div></td>
              <td class="px-3 py-2">{{ formatTime(overviewLastSyncAt(item) || '') }}<div class="text-xs text-gray-500">{{ overviewLastSyncStatus(item) || '-' }}</div></td>
              <td class="max-w-xs truncate px-3 py-2">{{ overviewLastError(item) || '-' }}</td>
              <td class="whitespace-nowrap px-3 py-2">
                <button type="button" class="btn btn-secondary btn-xs mr-2" @click="editSourceFromOverview(item)">编辑</button>
                <button type="button" class="btn btn-secondary btn-xs mr-2" @click="testUpstreamSource(overviewId(item))">测试</button>
                <button type="button" class="btn btn-secondary btn-xs mr-2" @click="syncUpstreamSource(overviewId(item))">同步</button>
                <button type="button" class="btn btn-danger btn-xs" @click="deleteUpstreamSource(overviewId(item))">删除</button>
              </td>
            </tr>
            <tr v-if="upstreamOverview.length === 0">
              <td colspan="7" class="px-3 py-8 text-center text-gray-500 dark:text-gray-400">暂无上游倍率源。</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="card p-6">
      <div class="flex items-center justify-between gap-4">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">最近调度日志</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">服务重启后清空，只展示最近内存日志。</p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" @click="loadLogs" :disabled="logsLoading">刷新日志</button>
      </div>

      <div v-if="logs.length === 0" class="mt-6 rounded-lg border border-dashed border-gray-300 p-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
        有请求经过网关后会出现调度日志。
      </div>
      <div v-else class="mt-6 overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
          <thead class="text-left text-xs uppercase text-gray-500 dark:text-gray-400">
            <tr>
              <th class="px-3 py-2">时间</th>
              <th class="px-3 py-2">结果</th>
              <th class="px-3 py-2">账号</th>
              <th class="px-3 py-2">模型</th>
              <th class="px-3 py-2">候选</th>
              <th class="px-3 py-2">优先命中</th>
              <th class="px-3 py-2">粘性</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="log in logs" :key="`${log.created_at}-${log.reason}-${log.account_id || 0}`" class="text-gray-700 dark:text-gray-200">
              <td class="whitespace-nowrap px-3 py-2">{{ formatTime(log.created_at) }}</td>
              <td class="px-3 py-2">{{ log.reason }}</td>
              <td class="px-3 py-2">{{ log.account_id ? `#${log.account_id} ${log.account_name || ''}` : '-' }}</td>
              <td class="px-3 py-2">{{ log.model || '-' }}</td>
              <td class="px-3 py-2">{{ log.available_count }}/{{ log.candidate_count }}</td>
              <td class="px-3 py-2">{{ log.preferred_hit ? '是' : '否' }}</td>
              <td class="px-3 py-2">{{ log.sticky_status || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import Toggle from '@/components/common/Toggle.vue'
import { adminAPI } from '@/api/admin'
import type { Account } from '@/types'
import type { GatewaySchedulingSettings } from '@/api/admin/settings'
import type { SchedulingLogEvent } from '@/api/admin/scheduling'
import type { UpstreamRateOverviewItem, UpstreamRateSourcePayload, UpstreamRateSourceType, UpstreamRateAuthMode } from '@/api/admin/upstreamRates'

const defaultConfig = (): GatewaySchedulingSettings => ({
  preferred_account_id: 0,
  score_weights: {
    load: 1,
    queue: 1,
    debt: 1,
    error_rate: 0.8,
    latency: 0.4,
    rate_multiplier: 0.6,
    quota_risk: 0.3,
  },
  latency_baseline_ms: 15000,
  quota_risk_threshold: 0.2,
  max_score_penalty: 5,
  sticky_session_mode: 'soft',
  sticky_escape_score_ratio: 1.25,
  sticky_escape_load_rate: 75,
  active_probe: {
    auto_pause_enabled: true,
    failure_threshold: 3,
    pause_duration: '10m',
    pause_duration_max: '1h',
  },
  slow_start: {
    enabled: true,
    duration: '5m',
    penalty: 1,
  },
  upstream_rate: {
    enabled: false,
    stale_ttl_seconds: 600,
    rate_weight: 0.6,
    health_weight: 0.4,
    min_success_rate: 0.8,
  },
})

const schedulingSteps = [
  { title: '硬过滤', description: '先排除禁用、不可调度、限流、过载、额度/RPM 不满足和模型不支持账号。' },
  { title: '粘性会话', description: '已有会话在 strict/soft 规则允许时继续使用原账号，保障上下文稳定。' },
  { title: 'Priority 分层', description: '仍按账号 priority 分层，数字越小越优先。优先账号不能跨层。' },
  { title: '同层优先账号', description: '指定账号在同层且合格时临时置顶，不改账号自身 priority。' },
  { title: 'Weighted P2C', description: '最后按负载、队列、债务、错误率、延迟、倍率和额度风险评分。' },
]

const config = reactive<GatewaySchedulingSettings>(defaultConfig())
const accountOptions = ref<Account[]>([])
const logs = ref<SchedulingLogEvent[]>([])
const upstreamOverview = ref<UpstreamRateOverviewItem[]>([])
const loading = ref(false)
const saving = ref(false)
const logsLoading = ref(false)
const upstreamLoading = ref(false)
const upstreamSaving = ref(false)
const error = ref('')
const upstreamMessage = ref('')
const editingSourceId = ref<number | null>(null)
const sourceForm = reactive<UpstreamRateSourcePayload>({
  name: '',
  source_type: 'sub2api',
  base_url: '',
  auth_mode: 'bearer_token',
  token: '',
  recharge_multiplier: 1,
  sync_interval_seconds: 300,
  enabled: true,
  use_for_scheduling: false,
})

function assignConfig(next: GatewaySchedulingSettings) {
  Object.assign(config, defaultConfig(), next)
  config.score_weights = { ...defaultConfig().score_weights, ...(next.score_weights || {}) }
  config.active_probe = { ...defaultConfig().active_probe, ...(next.active_probe || {}) }
  config.slow_start = { ...defaultConfig().slow_start, ...(next.slow_start || {}) }
  config.upstream_rate = { ...defaultConfig().upstream_rate, ...(next.upstream_rate || {}) }
}

async function loadConfig() {
  const next = await adminAPI.scheduling.getConfig()
  assignConfig(next)
}

async function loadAccounts() {
  const result = await adminAPI.accounts.list(1, 500, { lite: 'true', sort_by: 'priority', sort_order: 'asc' })
  accountOptions.value = result.items || []
}

async function loadLogs() {
  logsLoading.value = true
  try {
    logs.value = await adminAPI.scheduling.listLogs(100)
  } finally {
    logsLoading.value = false
  }
}

async function loadUpstreamRates() {
  upstreamLoading.value = true
  try {
    upstreamOverview.value = await adminAPI.upstreamRates.listOverview()
  } finally {
    upstreamLoading.value = false
  }
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    await Promise.all([loadConfig(), loadAccounts(), loadLogs(), loadUpstreamRates()])
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载调度面板失败'
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  saving.value = true
  error.value = ''
  try {
    const saved = await adminAPI.scheduling.updateConfig(JSON.parse(JSON.stringify(config)))
    assignConfig(saved)
    await loadLogs()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存调度配置失败'
  } finally {
    saving.value = false
  }
}

function sourcePayload(): UpstreamRateSourcePayload {
  return {
    ...sourceForm,
    name: sourceForm.name.trim(),
    base_url: sourceForm.base_url.trim(),
    token: sourceForm.token?.trim() || undefined,
  }
}

function resetSourceForm() {
  editingSourceId.value = null
  Object.assign(sourceForm, {
    name: '',
    source_type: 'sub2api' as UpstreamRateSourceType,
    base_url: '',
    auth_mode: 'bearer_token' as UpstreamRateAuthMode,
    token: '',
    recharge_multiplier: 1,
    sync_interval_seconds: 300,
    enabled: true,
    use_for_scheduling: false,
  })
}

async function saveSource() {
  upstreamSaving.value = true
  upstreamMessage.value = ''
  try {
    if (editingSourceId.value) {
      await adminAPI.upstreamRates.updateSource(editingSourceId.value, sourcePayload())
      upstreamMessage.value = '上游倍率源已更新。'
    } else {
      await adminAPI.upstreamRates.createSource(sourcePayload())
      upstreamMessage.value = '上游倍率源已新增。'
    }
    resetSourceForm()
    await loadUpstreamRates()
  } catch (err) {
    upstreamMessage.value = err instanceof Error ? err.message : '保存上游倍率源失败'
  } finally {
    upstreamSaving.value = false
  }
}

function editSourceFromOverview(item: UpstreamRateOverviewItem) {
  editingSourceId.value = overviewId(item)
  Object.assign(sourceForm, {
    name: overviewName(item),
    source_type: overviewType(item) as UpstreamRateSourceType,
    base_url: overviewBaseUrl(item),
    auth_mode: 'bearer_token' as UpstreamRateAuthMode,
    token: '',
    recharge_multiplier: 1,
    sync_interval_seconds: 300,
    enabled: overviewEnabled(item),
    use_for_scheduling: overviewUseForScheduling(item),
  })
}

async function testUpstreamSource(id: number) {
  const result = await adminAPI.upstreamRates.testSource(id)
  upstreamMessage.value = result.error || `测试完成：${result.status}，快照 ${result.snapshot_count} 个，耗时 ${result.latency_ms}ms。`
  await loadUpstreamRates()
}

async function syncUpstreamSource(id: number) {
  const result = await adminAPI.upstreamRates.syncSource(id)
  upstreamMessage.value = result.error || `同步完成：${result.status}，快照 ${result.snapshot_count} 个，耗时 ${result.latency_ms}ms。`
  await loadUpstreamRates()
}

async function deleteUpstreamSource(id: number) {
  if (!window.confirm('确认删除这个上游倍率源？相关快照和绑定会一起删除。')) return
  await adminAPI.upstreamRates.deleteSource(id)
  upstreamMessage.value = '上游倍率源已删除。'
  await loadUpstreamRates()
}

function overviewId(item: UpstreamRateOverviewItem) { return Number(item.source_id ?? item.SourceID ?? 0) }
function overviewName(item: UpstreamRateOverviewItem) { return String(item.source_name ?? item.SourceName ?? '') }
function overviewType(item: UpstreamRateOverviewItem) { return String(item.source_type ?? item.SourceType ?? '') }
function overviewBaseUrl(item: UpstreamRateOverviewItem) { return String(item.base_url ?? item.BaseURL ?? '') }
function overviewEnabled(item: UpstreamRateOverviewItem) { return Boolean(item.enabled ?? item.Enabled) }
function overviewUseForScheduling(item: UpstreamRateOverviewItem) { return Boolean(item.use_for_scheduling ?? item.UseForScheduling) }
function overviewTokenConfigured(item: UpstreamRateOverviewItem) { return Boolean(item.token_configured ?? item.TokenConfigured) }
function overviewSnapshotCount(item: UpstreamRateOverviewItem) { return Number(item.snapshot_count ?? item.SnapshotCount ?? 0) }
function overviewBindingCount(item: UpstreamRateOverviewItem) { return Number(item.binding_count ?? item.BindingCount ?? 0) }
function overviewHealthRate(item: UpstreamRateOverviewItem) { return Number(item.health_success_rate_1h ?? item.HealthSuccessRate1h ?? 0) }
function overviewLatency(item: UpstreamRateOverviewItem) { return item.health_avg_latency_ms_1h ?? item.HealthAvgLatencyMS1h ?? null }
function overviewLastSyncAt(item: UpstreamRateOverviewItem) { return item.last_sync_at ?? item.LastSyncAt ?? '' }
function overviewLastSyncStatus(item: UpstreamRateOverviewItem) { return String(item.last_sync_status ?? item.LastSyncStatus ?? '') }
function overviewLastError(item: UpstreamRateOverviewItem) { return String(item.last_error ?? item.LastError ?? '') }

function formatRate(value: number) {
  return `${Math.round((Number.isFinite(value) ? value : 0) * 100)}%`
}

function formatTime(value: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

onMounted(() => {
  loadAll()
})
</script>

<style scoped>
.field {
  display: block;
  font-size: 0.875rem;
  font-weight: 500;
  color: rgb(55 65 81);
}

:global(.dark) .field {
  color: rgb(209 213 219);
}
</style>
