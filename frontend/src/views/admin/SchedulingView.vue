<template>
  <AppLayout>
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
      <div class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">按分组优先调度账号</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">每个分组可单独选择优先账号；为空时该分组使用常规调度。</p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" @click="clearPreferredAccounts">清空全部</button>
      </div>
      <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <div v-for="group in groupOptions" :key="group.id" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
          <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ group.name }}</div>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">#{{ group.id }} · {{ group.platform || 'all' }} · {{ group.status }}</p>
          <select v-model.number="config.preferred_account_by_group_id[String(group.id)]" class="input mt-3">
            <option :value="0">不指定优先账号</option>
            <option v-for="account in accountsForGroup(group.id)" :key="account.id" :value="account.id">
              #{{ account.id }} · {{ account.name }} · {{ account.platform }} · {{ account.status }} · priority {{ account.priority }}
            </option>
          </select>
        </div>
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
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import Toggle from '@/components/common/Toggle.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminAPI } from '@/api/admin'
import type { Account, AdminGroup } from '@/types'
import type { GatewaySchedulingSettings } from '@/api/admin/settings'
import type { SchedulingLogEvent } from '@/api/admin/scheduling'

const defaultConfig = (): GatewaySchedulingSettings => ({
  preferred_account_id: 0,
  preferred_account_by_group_id: {},
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
const groupOptions = ref<AdminGroup[]>([])
const logs = ref<SchedulingLogEvent[]>([])
const loading = ref(false)
const saving = ref(false)
const logsLoading = ref(false)
const error = ref('')

function assignConfig(next: GatewaySchedulingSettings) {
  Object.assign(config, defaultConfig(), next)
  config.preferred_account_by_group_id = { ...(next.preferred_account_by_group_id || {}) }
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

async function loadGroups() {
  groupOptions.value = await adminAPI.groups.getAllIncludingInactive()
}

function accountsForGroup(groupID: number) {
  return accountOptions.value.filter((account) => {
    const ids = account.group_ids || account.groups?.map((group) => group.id) || []
    return ids.includes(groupID)
  })
}

function clearPreferredAccounts() {
  config.preferred_account_id = 0
  config.preferred_account_by_group_id = {}
}

async function loadLogs() {
  logsLoading.value = true
  try {
    logs.value = await adminAPI.scheduling.listLogs(100)
  } finally {
    logsLoading.value = false
  }
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    await Promise.all([loadConfig(), loadAccounts(), loadGroups(), loadLogs()])
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
