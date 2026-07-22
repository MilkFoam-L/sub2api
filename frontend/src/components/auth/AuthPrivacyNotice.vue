<template>
  <aside
    data-test="auth-privacy-notice"
    role="note"
    aria-labelledby="auth-privacy-title"
    class="rounded-2xl border border-primary/20 bg-primary/5 p-4"
  >
    <div class="flex items-start gap-3">
      <div
        class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary"
        aria-hidden="true"
      >
        <Icon name="shield" size="sm" />
      </div>
      <div class="min-w-0">
        <h3 id="auth-privacy-title" class="text-sm font-semibold text-foreground">
          {{ t('auth.privacyCommitment') }}
        </h3>
        <p class="mt-1.5 text-xs leading-5 text-muted-foreground sm:text-sm sm:leading-6">
          {{ t('auth.privacyDescription') }}
        </p>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
</script>
