<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.dataImportTitle')"
    width="normal"
    close-on-click-outside
    @close="handleClose"
  >
    <form id="import-data-form" class="space-y-4" @submit.prevent="handleImport">
      <div class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.accounts.dataImportHint') }}
      </div>
      <div
        class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-600 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-400"
      >
        {{ t('admin.accounts.dataImportWarning') }}
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.dataImportFile') }}</label>
        <div
          class="flex items-center justify-between gap-3 rounded-lg border border-dashed border-gray-300 bg-gray-50 px-4 py-3 dark:border-dark-600 dark:bg-dark-800"
        >
          <div class="min-w-0">
            <div class="truncate text-sm text-gray-700 dark:text-dark-200">
              {{ fileLabel }}
            </div>
            <div class="text-xs text-gray-500 dark:text-dark-400">
              {{ t('admin.accounts.dataImportFileHint') }}
            </div>
          </div>
          <button type="button" class="btn btn-secondary shrink-0" @click="openFilePicker">
            {{ t('common.chooseFile') }}
          </button>
        </div>
        <input
          ref="fileInput"
          type="file"
          class="hidden"
          accept="application/json,.json"
          multiple
          @change="handleFileChange"
        />
      </div>

      <div
        v-if="result"
        class="space-y-2 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
      >
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.dataImportResult') }}
        </div>
        <div class="text-sm text-gray-700 dark:text-dark-300">
          {{ t('admin.accounts.dataImportResultSummary', result) }}
        </div>

        <div v-if="errorItems.length" class="mt-2">
          <div class="text-sm font-medium text-red-600 dark:text-red-400">
            {{ t('admin.accounts.dataImportErrors') }}
          </div>
          <div
            class="mt-2 max-h-48 overflow-auto rounded-lg bg-gray-50 p-3 font-mono text-xs dark:bg-dark-800"
          >
            <div v-for="(item, idx) in errorItems" :key="idx" class="whitespace-pre-wrap">
              {{ item.kind }} {{ item.name || item.proxy_key || '-' }} — {{ item.message }}
            </div>
          </div>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" type="button" :disabled="importing" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          class="btn btn-primary"
          type="submit"
          form="import-data-form"
          :disabled="importing"
        >
          {{ importing ? t('admin.accounts.dataImporting') : t('admin.accounts.dataImportButton') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { AdminDataImportResult, AdminDataPayload } from '@/types'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'close'): void
  (e: 'imported'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()
const appStore = useAppStore()

const importing = ref(false)
const files = ref<File[]>([])
const result = ref<AdminDataImportResult | null>(null)

const fileInput = ref<HTMLInputElement | null>(null)
const fileLabel = computed(() => {
  if (files.value.length === 0) {
    return t('admin.accounts.dataImportSelectFile')
  }
  if (files.value.length === 1) {
    return files.value[0].name
  }
  return t('admin.accounts.dataImportSelectedFiles', { count: files.value.length })
})

const errorItems = computed(() => result.value?.errors || [])

watch(
  () => props.show,
  (open) => {
    if (open) {
      files.value = []
      result.value = null
      if (fileInput.value) {
        fileInput.value.value = ''
      }
    }
  }
)

const openFilePicker = () => {
  fileInput.value?.click()
}

const handleFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  files.value = Array.from(target.files || [])
}

const handleClose = () => {
  if (importing.value) return
  emit('close')
}

const readFileAsText = async (sourceFile: File): Promise<string> => {
  if (typeof sourceFile.text === 'function') {
    return sourceFile.text()
  }

  if (typeof sourceFile.arrayBuffer === 'function') {
    const buffer = await sourceFile.arrayBuffer()
    return new TextDecoder().decode(buffer)
  }

  return await new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error || new Error('Failed to read file'))
    reader.readAsText(sourceFile)
  })
}

const createEmptyImportResult = (): AdminDataImportResult => ({
  proxy_created: 0,
  proxy_reused: 0,
  proxy_failed: 0,
  account_created: 0,
  account_failed: 0,
  errors: []
})

const mergeImportResult = (
  target: AdminDataImportResult,
  source: AdminDataImportResult,
  sourceFileName: string,
  shouldPrefixErrors: boolean
) => {
  target.proxy_created += source.proxy_created
  target.proxy_reused += source.proxy_reused
  target.proxy_failed += source.proxy_failed
  target.account_created += source.account_created
  target.account_failed += source.account_failed

  if (source.errors?.length) {
    target.errors = [
      ...(target.errors || []),
      ...source.errors.map((item) => ({
        ...item,
        message: shouldPrefixErrors ? `[${sourceFileName}] ${item.message}` : item.message
      }))
    ]
  }
}

const parseImportFiles = async (sourceFiles: File[]) => {
  const parsedFiles: Array<{ file: File; data: AdminDataPayload }> = []

  for (const sourceFile of sourceFiles) {
    const text = await readFileAsText(sourceFile)
    parsedFiles.push({
      file: sourceFile,
      data: JSON.parse(text) as AdminDataPayload
    })
  }

  return parsedFiles
}

const handleImport = async () => {
  if (!files.value.length) {
    appStore.showError(t('admin.accounts.dataImportSelectFile'))
    return
  }

  importing.value = true
  try {
    const parsedFiles = await parseImportFiles(files.value)
    const mergedResult = createEmptyImportResult()
    const shouldPrefixErrors = parsedFiles.length > 1

    for (const item of parsedFiles) {
      const res = await adminAPI.accounts.importData({
        data: item.data,
        skip_default_group_bind: true
      })
      mergeImportResult(mergedResult, res, item.file.name, shouldPrefixErrors)
    }

    result.value = mergedResult

    const msgParams: Record<string, unknown> = {
      account_created: mergedResult.account_created,
      account_failed: mergedResult.account_failed,
      proxy_created: mergedResult.proxy_created,
      proxy_reused: mergedResult.proxy_reused,
      proxy_failed: mergedResult.proxy_failed,
    }
    if (mergedResult.account_failed > 0 || mergedResult.proxy_failed > 0) {
      appStore.showError(t('admin.accounts.dataImportCompletedWithErrors', msgParams))
    } else {
      appStore.showSuccess(t('admin.accounts.dataImportSuccess', msgParams))
      emit('imported')
    }
  } catch (error: any) {
    if (error instanceof SyntaxError) {
      appStore.showError(t('admin.accounts.dataImportParseFailed'))
    } else {
      appStore.showError(error?.message || t('admin.accounts.dataImportFailed'))
    }
  } finally {
    importing.value = false
  }
}
</script>
