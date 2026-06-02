import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ImportDataModal from '@/components/admin/account/ImportDataModal.vue'
import { adminAPI } from '@/api/admin'

const showError = vi.fn()
const showSuccess = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      importData: vi.fn()
    }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === 'admin.accounts.dataImportSelectedFiles') {
        return `selected ${params?.count}`
      }
      return key
    }
  })
}))

const importDataMock = vi.mocked(adminAPI.accounts.importData)

const mountModal = () => mount(ImportDataModal, {
  props: { show: true },
  global: {
    stubs: {
      BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
    }
  }
})

const setInputFiles = (input: ReturnType<ReturnType<typeof mountModal>['find']>, files: File[]) => {
  Object.defineProperty(input.element, 'files', {
    value: files,
    configurable: true
  })
}

const createJsonFile = (content: unknown, name: string) => {
  const text = typeof content === 'string' ? content : JSON.stringify(content)
  const file = new File([text], name, { type: 'application/json' })
  Object.defineProperty(file, 'text', {
    value: () => Promise.resolve(text),
    configurable: true
  })
  return file
}

describe('ImportDataModal', () => {
  beforeEach(() => {
    showError.mockReset()
    showSuccess.mockReset()
    importDataMock.mockReset()
  })

  it('未选择文件时提示错误', async () => {
    const wrapper = mountModal()

    await wrapper.find('form').trigger('submit')
    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportSelectFile')
  })

  it('无效 JSON 时提示解析失败', async () => {
    const wrapper = mountModal()

    const input = wrapper.find('input[type="file"]')
    const file = createJsonFile('invalid json', 'data.json')
    setInputFiles(input, [file])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportParseFailed')
    expect(importDataMock).not.toHaveBeenCalled()
  })

  it('支持选择多个 JSON 文件并逐个导入后汇总结果', async () => {
    const wrapper = mountModal()

    importDataMock
      .mockResolvedValueOnce({
        proxy_created: 1,
        proxy_reused: 0,
        proxy_failed: 0,
        account_created: 2,
        account_failed: 0,
        errors: []
      })
      .mockResolvedValueOnce({
        proxy_created: 0,
        proxy_reused: 1,
        proxy_failed: 0,
        account_created: 3,
        account_failed: 0,
        errors: []
      })

    const firstPayload = { version: 1, accounts: [{ name: 'a' }], proxies: [] }
    const secondPayload = { version: 1, accounts: [{ name: 'b' }], proxies: [] }
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input, [
      createJsonFile(firstPayload, 'first.json'),
      createJsonFile(secondPayload, 'second.json')
    ])

    await input.trigger('change')
    expect(wrapper.text()).toContain('selected 2')

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(importDataMock).toHaveBeenCalledTimes(2)
    expect(importDataMock).toHaveBeenNthCalledWith(1, {
      data: firstPayload,
      skip_default_group_bind: true
    })
    expect(importDataMock).toHaveBeenNthCalledWith(2, {
      data: secondPayload,
      skip_default_group_bind: true
    })
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.dataImportSuccess')
    expect(wrapper.text()).toContain('admin.accounts.dataImportResultSummary')
  })

  it('多文件导入会汇总失败数量并为错误详情标记来源文件', async () => {
    const wrapper = mountModal()

    importDataMock
      .mockResolvedValueOnce({
        proxy_created: 0,
        proxy_reused: 0,
        proxy_failed: 0,
        account_created: 1,
        account_failed: 0,
        errors: []
      })
      .mockResolvedValueOnce({
        proxy_created: 0,
        proxy_reused: 0,
        proxy_failed: 0,
        account_created: 0,
        account_failed: 1,
        errors: [{ kind: 'account', name: 'broken', message: 'duplicate' }]
      })

    const input = wrapper.find('input[type="file"]')
    setInputFiles(input, [
      createJsonFile({ version: 1, accounts: [{ name: 'ok' }], proxies: [] }, 'ok.json'),
      createJsonFile({ version: 1, accounts: [{ name: 'broken' }], proxies: [] }, 'bad.json')
    ])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportCompletedWithErrors')
    expect(wrapper.text()).toContain('[bad.json] duplicate')
  })
})
