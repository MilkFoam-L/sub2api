<template>
  <div class="space-y-4">
    <div class="flex items-start gap-3">
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-primary-100 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
        <Icon name="shield" size="md" />
      </div>
      <div class="min-w-0 flex-1">
        <p class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('profile.privacyFilterTitle') }}
        </p>
        <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
          {{ t('profile.privacyFilterDescription') }}
        </p>
      </div>
    </div>

    <div class="flex items-center justify-between gap-4 rounded-2xl border border-gray-100 bg-white/80 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/50">
      <div>
        <p class="text-sm font-medium text-gray-900 dark:text-white">
          {{ enabled ? t('profile.privacyFilterEnabled') : t('profile.privacyFilterDisabled') }}
        </p>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ enabled ? t('profile.privacyFilterEnabledHint') : t('profile.privacyFilterDisabledHint') }}
        </p>
      </div>

      <button
        data-testid="profile-privacy-filter-toggle"
        type="button"
        role="switch"
        :aria-checked="enabled"
        :disabled="saving || !user"
        :class="[
          'relative inline-flex h-7 w-12 shrink-0 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60 dark:focus:ring-offset-dark-900',
          enabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-dark-600'
        ]"
        @click="togglePrivacyFilter"
      >
        <span
          :class="[
            'inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform',
            enabled ? 'translate-x-6' : 'translate-x-1'
          ]"
        />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { userAPI } from '@/api'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import type { User } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{
  user: User | null
}>()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const enabled = ref(Boolean(props.user?.privacy_filter_enabled))
const saving = ref(false)

watch(() => props.user?.privacy_filter_enabled, (value) => {
  enabled.value = Boolean(value)
})

async function togglePrivacyFilter() {
  if (!props.user || saving.value) {
    return
  }

  const previous = enabled.value
  const next = !previous
  enabled.value = next
  saving.value = true

  try {
    const updatedUser = await userAPI.updateProfile({
      privacy_filter_enabled: next
    })
    authStore.user = updatedUser
    appStore.showSuccess(next ? t('profile.privacyFilterEnableSuccess') : t('profile.privacyFilterDisableSuccess'))
  } catch (error: unknown) {
    enabled.value = previous
    appStore.showError(extractApiErrorMessage(error, t('profile.privacyFilterUpdateFailed')))
  } finally {
    saving.value = false
  }
}
</script>
