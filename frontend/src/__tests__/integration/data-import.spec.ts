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

const importGroups = [
  {
    id: 11,
    name: 'openai-default',
    description: 'OpenAI default group',
    platform: 'openai',
    rate_multiplier: 1,
    account_count: 2,
    is_exclusive: false,
    status: 'active',
    subscription_type: 'standard'
  },
  {
    id: 12,
    name: 'openai-plus',
    description: 'OpenAI plus group',
    platform: 'openai',
    rate_multiplier: 1.5,
    account_count: 1,
    is_exclusive: false,
    status: 'active',
    subscription_type: 'standard'
  },
  {
    id: 21,
    name: 'claude-default',
    description: 'Claude group',
    platform: 'anthropic',
    rate_multiplier: 1,
    account_count: 4,
    is_exclusive: false,
    status: 'active',
    subscription_type: 'standard'
  },
  {
    id: 31,
    name: 'gemini-default',
    description: 'Gemini group',
    platform: 'gemini',
    rate_multiplier: 1,
    account_count: 3,
    is_exclusive: false,
    status: 'active',
    subscription_type: 'standard'
  }
]

const mountModal = (props: Record<string, unknown> = {}) => mount(ImportDataModal, {
  props: { show: true, groups: importGroups, ...props },
  global: {
    stubs: {
      BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
      GroupBadge: { props: ['name'], template: '<span>{{ name }}</span>' },
      Icon: true,
      PlatformIcon: true
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
      skip_default_group_bind: true,
      group_ids: []
    })
    expect(importDataMock).toHaveBeenNthCalledWith(2, {
      data: secondPayload,
      skip_default_group_bind: true,
      group_ids: []
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

  it('导入分组选择器只显示 OpenAI 分组并使用导入专用文案', () => {
    const wrapper = mountModal()

    expect(wrapper.text()).toContain('admin.accounts.dataImportGroups')
    expect(wrapper.text()).toContain('admin.accounts.dataImportGroupsHint')
    expect(wrapper.text()).not.toContain('admin.users.groups')
    expect(wrapper.text()).toContain('openai-default')
    expect(wrapper.text()).toContain('openai-plus')
    expect(wrapper.text()).not.toContain('claude-default')
    expect(wrapper.text()).not.toContain('gemini-default')
  })

  it('勾选导入分组后提交 group_ids', async () => {
    const wrapper = mountModal()
    importDataMock.mockResolvedValueOnce({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 0,
      errors: []
    })

    const checkbox = wrapper.find('input[type="checkbox"][value="12"]')
    await checkbox.setValue(true)

    const payload = { version: 1, accounts: [{ name: 'a' }], proxies: [] }
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input, [createJsonFile(payload, 'data.json')])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(importDataMock).toHaveBeenCalledWith({
      data: payload,
      skip_default_group_bind: true,
      group_ids: [12]
    })
  })
})
