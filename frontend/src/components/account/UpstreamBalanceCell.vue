<template>
  <div v-if="eligible" class="flex h-6 min-w-[7rem] items-center gap-1">
    <HelpTooltip class="-ml-1" width-class="w-max max-w-[calc(100vw-2rem)]" data-testid="upstream-balance-details">
      <template #trigger>
        <span
          class="cursor-help border-b border-dotted border-gray-300 text-sm font-medium dark:border-dark-600"
          :class="hasBalance ? 'font-mono text-gray-800 dark:text-gray-200' : statusClass || 'text-gray-400 dark:text-gray-500'"
          data-testid="upstream-balance-value"
        >{{ primaryValue }}</span>
      </template>
      <div class="space-y-1">
        <template v-if="hasBalance && data">
          <p v-if="data.source">{{ t('admin.accounts.upstreamBalance.source', { value: data.source }) }}</p>
          <p v-if="data.mode">{{ t('admin.accounts.upstreamBalance.mode', { value: data.mode }) }}</p>
          <p v-if="data.mode === 'wallet' && (data.used != null || data.raw_used != null)">
            {{ t('admin.accounts.upstreamBalance.cumulativeUsed', { used: formatNumber(data.used ?? data.raw_used) }) }}
          </p>
          <p v-else-if="data.used != null || data.limit != null">
            {{ t('admin.accounts.upstreamBalance.usedLimit', { used: formatNumber(data.used), limit: formatNumber(data.limit) }) }}
          </p>
          <p v-else-if="data.raw_used != null || data.raw_limit != null">
            {{ t('admin.accounts.upstreamBalance.usedLimit', { used: formatNumber(data.raw_used), limit: formatNumber(data.raw_limit) }) }}
          </p>
          <p>{{ t('admin.accounts.upstreamBalance.updatedAt', { value: formatDate(snapshot?.received_at) }) }}</p>
        </template>
        <template v-else-if="stale && lastDetectedBalance != null">
          <p data-testid="upstream-balance-last-value">
            {{ t('admin.accounts.upstreamBalance.lastDetectedBalance', { value: formatNumber(lastDetectedBalance) }) }}
          </p>
          <p data-testid="upstream-balance-last-time">
            {{ t('admin.accounts.upstreamBalance.lastDetectedAt', { value: formatDate(snapshot?.received_at) }) }}
          </p>
          <p data-testid="upstream-balance-elapsed">
            {{ t('admin.accounts.upstreamBalance.elapsedSince', { value: elapsedSinceLastSuccess }) }}
          </p>
        </template>
        <p v-else>{{ statusLabel || '-' }}</p>
        <p
          v-if="probeEnabled && globalProbeEnabled !== false && nextProbeAt"
          data-testid="upstream-balance-next-probe"
        >
          {{ t('admin.accounts.upstreamBalance.nextProbeAt', { value: formatDate(nextProbeAt) }) }}
        </p>
        <p class="mt-2 border-t border-white/15 pt-2" data-testid="upstream-balance-probe-state">
          {{ t('admin.accounts.upstreamBalance.accountProbeState') }}
          <span :class="probeEnabled ? 'text-emerald-400' : 'text-red-400'">
            {{ probeEnabled ? t('admin.accounts.upstreamBalance.enabled') : t('admin.accounts.upstreamBalance.disabled') }}
          </span>
        </p>
        <p
          v-if="globalProbeEnabled === false"
          class="mt-1"
          data-testid="upstream-balance-global-probe-state"
        >
          {{ t('admin.accounts.upstreamBalance.globalProbeState') }}
          <span class="text-red-400">{{ t('admin.accounts.upstreamBalance.disabled') }}</span>
        </p>
      </div>
    </HelpTooltip>
    <span v-if="hasBalance && statusLabel" :class="statusClass" class="whitespace-nowrap text-[10px] font-medium">
      {{ statusLabel }}
    </span>
    <button
      type="button"
      class="inline-flex h-6 w-6 flex-shrink-0 items-center justify-center rounded text-blue-600 transition-colors hover:bg-blue-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-blue-400 dark:hover:bg-blue-900/30"
      :disabled="probing"
      :aria-label="t('admin.accounts.upstreamBalance.manualProbe')"
      :title="t('admin.accounts.upstreamBalance.manualProbe')"
      data-testid="upstream-balance-probe"
      @click="$emit('probe')"
    >
      <Icon name="refresh" size="xs" :class="{ 'animate-spin': probing }" />
    </button>
  </div>
  <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Account, UpstreamBalanceProbeSnapshot } from '@/types'

const props = withDefaults(defineProps<{
  account: Account
  now?: number
  probing?: boolean
  globalProbeEnabled?: boolean
}>(), {
  now: () => Date.now(),
  probing: false,
  globalProbeEnabled: true
})

defineEmits<{ (event: 'probe'): void }>()

const { t } = useI18n()
const CLOCK_SKEW_TOLERANCE_MS = 5 * 60 * 1000
const eligible = computed(() => props.account.platform === 'openai' && props.account.type === 'apikey')
const snapshot = computed<UpstreamBalanceProbeSnapshot | undefined>(() => props.account.extra?.upstream_balance_probe)
const data = computed(() => snapshot.value?.data)
const probeEnabled = computed(() => props.account.extra?.upstream_balance_probe_enabled === true)
const nextProbeAt = computed(() => {
  const value = snapshot.value?.next_probe_at
  return typeof value === 'string' && Number.isFinite(Date.parse(value)) ? value : ''
})
const receivedAt = computed(() => typeof snapshot.value?.received_at === 'string' ? Date.parse(snapshot.value.received_at) : Number.NaN)
const freshUntil = computed(() => {
  if (typeof snapshot.value?.fresh_until === 'string') return Date.parse(snapshot.value.fresh_until)
  if (!['ok', 'failed'].includes(snapshot.value?.status ?? '') || !nextProbeAt.value || !Number.isFinite(receivedAt.value)) {
    return Number.NaN
  }
  const next = Date.parse(nextProbeAt.value)
  return Number.isFinite(next) && next > receivedAt.value
    ? receivedAt.value + 2 * (next - receivedAt.value)
    : Number.NaN
})
const hasFreshnessMetadata = computed(() =>
  typeof snapshot.value?.fresh_until === 'string' || Boolean(nextProbeAt.value)
)
const stale = computed(() => {
  if (!snapshot.value || !data.value) return false
  if (!hasFreshnessMetadata.value) return false
  if (!Number.isFinite(receivedAt.value) || receivedAt.value > props.now + CLOCK_SKEW_TOLERANCE_MS) return true
  return !Number.isFinite(freshUntil.value) || freshUntil.value <= receivedAt.value || props.now > freshUntil.value
})
const lastDetectedBalance = computed(() => {
  const remaining = data.value?.remaining
  if (typeof remaining === 'number' && Number.isFinite(remaining)) return remaining
  const rawRemaining = data.value?.raw_remaining
  return typeof rawRemaining === 'number' && Number.isFinite(rawRemaining) ? rawRemaining : null
})
const valueLabel = computed(() => {
  if (!['ok', 'failed'].includes(snapshot.value?.status ?? '') || stale.value || data.value?.unlimited === true) return ''
  return lastDetectedBalance.value == null ? '' : formatNumber(lastDetectedBalance.value)
})
const statusLabel = computed(() => {
  if (!snapshot.value) return t('admin.accounts.upstreamBalance.notProbed')
  if (snapshot.value.status === 'unsupported') return t('admin.accounts.upstreamBalance.unsupported')
  if (stale.value) return t('admin.accounts.upstreamBalance.stale')
  if (snapshot.value.status === 'failed' || data.value?.unlimited === true) return t('admin.accounts.upstreamBalance.failed')
  return valueLabel.value ? '' : t('admin.accounts.upstreamBalance.notProbed')
})
const statusClass = computed(() => {
  if (!snapshot.value) return 'text-gray-400 dark:text-gray-500'
  if (snapshot.value.status === 'unsupported' || data.value?.unlimited === true) return 'text-gray-500 dark:text-gray-400'
  if (stale.value) return 'text-amber-600 dark:text-amber-400'
  if (snapshot.value.status === 'failed') return 'text-red-600 dark:text-red-400'
  return ''
})
const hasBalance = computed(() => valueLabel.value !== '')
const primaryValue = computed(() => hasBalance.value ? valueLabel.value : statusLabel.value || '-')
const elapsedSinceLastSuccess = computed(() => {
  if (!Number.isFinite(receivedAt.value)) return '-'
  const elapsedMinutes = Math.max(0, Math.floor((props.now - receivedAt.value) / 60_000))
  if (elapsedMinutes < 1) return t('admin.accounts.upstreamBalance.justNow')
  if (elapsedMinutes < 60) return t('admin.accounts.upstreamBalance.minutesAgo', { count: elapsedMinutes })
  const elapsedHours = Math.floor(elapsedMinutes / 60)
  if (elapsedHours < 24) return t('admin.accounts.upstreamBalance.hoursAgo', { count: elapsedHours })
  return t('admin.accounts.upstreamBalance.daysAgo', { count: Math.floor(elapsedHours / 24) })
})
const formatNumber = (value?: number) => typeof value === 'number' && Number.isFinite(value) ? value.toFixed(1) : '-'
const formatDate = (value?: string) => value
  ? new Date(value).toLocaleString(undefined, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  : '-'
</script>
