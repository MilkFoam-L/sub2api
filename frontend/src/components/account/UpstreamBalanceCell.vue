<template>
  <div v-if="eligible" class="flex h-6 min-w-[7rem] items-center gap-1">
    <HelpTooltip class="-ml-1" width-class="w-max max-w-[calc(100vw-2rem)]" data-testid="upstream-balance-details">
      <template #trigger>
        <span
          class="cursor-help border-b border-dotted border-gray-300 text-sm font-medium dark:border-dark-600"
          :class="valueLabel ? 'font-mono text-gray-800 dark:text-gray-200' : statusClass"
          data-testid="upstream-balance-value"
        >{{ valueLabel || statusLabel || '-' }}</span>
      </template>
      <div class="space-y-1">
        <p v-if="data?.source">{{ t('admin.accounts.upstreamBalance.source', { value: data.source }) }}</p>
        <p v-if="data?.mode">{{ t('admin.accounts.upstreamBalance.mode', { value: data.mode }) }}</p>
        <p v-if="data?.mode === 'wallet' && (data.used != null || data.raw_used != null)">
          {{ t('admin.accounts.upstreamBalance.cumulativeUsed', { used: formatNumber(data.used ?? data.raw_used) }) }}
        </p>
        <p v-else-if="data && (data.used != null || data.limit != null)">
          {{ t('admin.accounts.upstreamBalance.usedLimit', { used: formatNumber(data.used), limit: formatNumber(data.limit) }) }}
        </p>
        <p v-else-if="data && (data.raw_used != null || data.raw_limit != null)">
          {{ t('admin.accounts.upstreamBalance.usedLimit', { used: formatNumber(data.raw_used), limit: formatNumber(data.raw_limit) }) }}
        </p>
        <p>{{ t('admin.accounts.upstreamBalance.updatedAt', { value: formatDate(snapshot?.received_at) }) }}</p>
      </div>
    </HelpTooltip>
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

const props = withDefaults(defineProps<{ account: Account; now?: number; probing?: boolean }>(), { now: () => Date.now(), probing: false })
defineEmits<{ (event: 'probe'): void }>()
const { t } = useI18n()
const eligible = computed(() => props.account.platform === 'openai' && props.account.type === 'apikey')
const snapshot = computed<UpstreamBalanceProbeSnapshot | undefined>(() => props.account.extra?.upstream_balance_probe)
const data = computed(() => snapshot.value?.data)
const receivedAt = computed(() => snapshot.value?.received_at ? Date.parse(snapshot.value.received_at) : Number.NaN)
const freshUntil = computed(() => {
  if (snapshot.value?.fresh_until) return Date.parse(snapshot.value.fresh_until)
  return Number.NaN
})
const stale = computed(() => {
  if (!snapshot.value) return false
  if (snapshot.value.status !== 'ok' || !snapshot.value.fresh_until) return false
  return !Number.isFinite(receivedAt.value) || !Number.isFinite(freshUntil.value) || props.now > freshUntil.value
})
const statusLabel = computed(() => {
  if (!snapshot.value) return t('admin.accounts.upstreamBalance.notProbed')
  if (snapshot.value.status === 'unsupported') return t('admin.accounts.upstreamBalance.unsupported')
  if (snapshot.value.status === 'failed') return t('admin.accounts.upstreamBalance.failed')
  if (stale.value) return t('admin.accounts.upstreamBalance.stale')
  if (data.value?.unlimited === true) return t('admin.accounts.upstreamBalance.failed')
  return t('admin.accounts.upstreamBalance.notProbed')
})
const valueLabel = computed(() => {
  if (snapshot.value?.status !== 'ok' || stale.value || data.value?.unlimited === true) return ''
  const remaining = data.value?.remaining
  if (typeof remaining === 'number' && Number.isFinite(remaining)) {
    return formatNumber(remaining)
  }
  const rawRemaining = data.value?.raw_remaining
  if (typeof rawRemaining === 'number' && Number.isFinite(rawRemaining)) {
    return formatNumber(rawRemaining)
  }
  return ''
})
const statusClass = computed(() => {
  if (!snapshot.value) return 'text-gray-400 dark:text-gray-500'
  if (snapshot.value.status === 'failed') return 'text-red-600 dark:text-red-400'
  if (snapshot.value.status === 'unsupported' || data.value?.unlimited === true) return 'text-gray-500 dark:text-gray-400'
  if (stale.value) return 'text-amber-600 dark:text-amber-400'
  return 'text-gray-400 dark:text-gray-500'
})
const formatNumber = (value?: number) => typeof value === 'number' && Number.isFinite(value) ? value.toFixed(1) : '-'
const formatDate = (value?: string) => value ? new Date(value).toLocaleString(undefined, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) : '-'
</script>
