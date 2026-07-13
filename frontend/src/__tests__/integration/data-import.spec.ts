import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ImportDataModal from '@/components/admin/account/ImportDataModal.vue'
import { adminAPI } from '@/api/admin'

const showError = vi.fn()
const showSuccess = vi.fn()
const showWarning = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showWarning
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      importData: vi.fn()
    }
  }
}))

const importDataMock = vi.mocked(adminAPI.accounts.importData)

const importTranslations: Record<string, string> = {
  'admin.accounts.dataImportTitle': '导入数据',
  'admin.accounts.dataImportHint': '上传导出的 JSON 文件以批量导入账号与代理。',
  'admin.accounts.dataImportWarning': '导入将创建新账号与代理，分组需手工绑定；请确认已有数据不会冲突。',
  'admin.accounts.dataImportFile': '数据文件',
  'admin.accounts.dataImportSelectFile': '请选择数据文件',
  'admin.accounts.dataImportFileHint': '支持多选 JSON 文件（.json）',
  'admin.accounts.dataImportGroups': '导入账号分组',
  'admin.accounts.dataImportGroupsHint': '仅显示 OpenAI 分组；导入成功的账号会自动绑定到选中分组。',
  'admin.accounts.dataImportButton': '开始导入',
  'admin.accounts.dataImporting': '导入中...',
  'admin.accounts.dataImportParseFailedFile': '文件 {name} 解析失败',
  'admin.accounts.dataImportInvalidFile': '文件 {name} 不是受支持的导出数据文件',
  'admin.accounts.dataImportIgnoredFiles': '已忽略 {count} 个非 JSON 文件',
  'admin.accounts.dataImportFailed': '数据导入失败',
  'admin.accounts.dataImportResult': '导入结果',
  'admin.accounts.dataImportResultSummary': '代理创建 {proxy_created}，复用 {proxy_reused}，失败 {proxy_failed}；账号创建 {account_created}，失败 {account_failed}',
  'admin.accounts.dataImportErrors': '失败详情',
  'admin.accounts.dataImportSuccess': '导入完成：账号 {account_created}，失败 {account_failed}',
  'admin.accounts.dataImportCompletedWithErrors': '导入完成但有错误：账号失败 {account_failed}，代理失败 {proxy_failed}',
  'admin.accounts.selectedCount': '已选 {count}',
  'common.chooseFile': '选择文件',
  'common.cancel': '取消'
}

const t = (key: string, params: Record<string, unknown> = {}) => {
  const message = importTranslations[key]
  if (!message) throw new Error(`Missing test translation: ${key}`)
  return message.replace(/\{(\w+)\}/g, (_match, name: string) => String(params[name] ?? `{${name}}`))
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t })
}))

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
      GroupSelector: {
        props: ['modelValue', 'groups', 'platform', 'label', 'hint'],
        emits: ['update:modelValue'],
        template: `
          <div>
            <div>{{ label }}</div>
            <div>{{ hint }}</div>
            <label v-for="group in groups.filter((item) => item.platform === platform)" :key="group.id">
              <input
                type="checkbox"
                :value="group.id"
                :checked="modelValue.includes(group.id)"
                @change="$emit('update:modelValue', $event.target.checked ? [...modelValue, group.id] : modelValue.filter((id) => id !== group.id))"
              />
              {{ group.name }}
            </label>
          </div>
        `
      }
    }
  }
})

const makeJsonFile = (name: string, content: string, type = 'application/json') => {
  const file = new File([content], name, { type })
  Object.defineProperty(file, 'text', {
    value: () => Promise.resolve(content),
    configurable: true
  })
  return file
}

const setInputFiles = (element: Element, files: File[]) => {
  Object.defineProperty(element, 'files', {
    value: files,
    configurable: true
  })
}

describe('ImportDataModal', () => {
  beforeEach(() => {
    showError.mockReset()
    showSuccess.mockReset()
    showWarning.mockReset()
    importDataMock.mockReset()
  })

  it('未选择文件时提示错误', async () => {
    const wrapper = mountModal()

    await wrapper.find('form').trigger('submit')
    expect(showError).toHaveBeenCalledWith(t('admin.accounts.dataImportSelectFile'))
  })

  it('无效 JSON 时按文件名提示解析失败', async () => {
    const wrapper = mountModal()

    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile('data.json', 'invalid json')])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith(t('admin.accounts.dataImportParseFailedFile', { name: 'data.json' }))
    expect(importDataMock).not.toHaveBeenCalled()
  })

  it('不是导出数据的 JSON 按文件名拒绝', async () => {
    const wrapper = mountModal()

    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile('random.json', JSON.stringify({ name: 'test' }))])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith(t('admin.accounts.dataImportInvalidFile', { name: 'random.json' }))
    expect(importDataMock).not.toHaveBeenCalled()
  })

  it('无有效 JSON 的选择不清空已有选择', async () => {
    importDataMock.mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 0,
      errors: []
    })

    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')

    const valid = makeJsonFile(
      'valid.json',
      JSON.stringify({ exported_at: '2026-07-05T00:00:00Z', proxies: [], accounts: [{ name: 'a' }] })
    )
    setInputFiles(input.element, [valid])
    await input.trigger('change')

    setInputFiles(input.element, [new File(['hello'], 'notes.txt', { type: 'text/plain' })])
    await input.trigger('change')
    expect(showError).toHaveBeenCalledWith(t('admin.accounts.dataImportSelectFile'))

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(importDataMock).toHaveBeenCalledWith({
      data: expect.objectContaining({
        accounts: [{ name: 'a' }]
      }),
      skip_default_group_bind: true,
      group_ids: []
    })
  })

  it('支持选择多个 JSON 文件并合并后导入', async () => {
    importDataMock.mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 2,
      account_failed: 0,
      errors: []
    })

    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')
    const first = makeJsonFile(
      'first.json',
      JSON.stringify({ exported_at: '2026-07-05T00:00:00Z', proxies: [], accounts: [{ name: 'a' }] })
    )
    const second = makeJsonFile(
      'second.json',
      JSON.stringify({
        exported_at: '2026-07-05T00:00:01Z',
        proxies: [{ proxy_key: 'p' }],
        accounts: [{ name: 'b' }]
      })
    )
    setInputFiles(input.element, [first, second])

    await input.trigger('change')
    expect(wrapper.text()).toContain(t('admin.accounts.selectedCount', { count: 2 }))

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(importDataMock).toHaveBeenCalledTimes(1)
    expect(importDataMock).toHaveBeenCalledWith({
      data: expect.objectContaining({
        proxies: [{ proxy_key: 'p' }],
        accounts: [{ name: 'a' }, { name: 'b' }]
      }),
      skip_default_group_bind: true,
      group_ids: []
    })
    expect(showSuccess).toHaveBeenCalledWith(
      t('admin.accounts.dataImportSuccess', { account_created: 2, account_failed: 0 })
    )
  })

  it('部分成功时关闭弹窗仍通知父组件刷新', async () => {
    importDataMock.mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 1,
      errors: []
    })

    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [
      makeJsonFile(
        'mixed.json',
        JSON.stringify({
          exported_at: '2026-07-05T00:00:00Z',
          proxies: [],
          accounts: [{ name: 'a' }, { name: 'b' }]
        })
      )
    ])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith(
      t('admin.accounts.dataImportCompletedWithErrors', { account_failed: 1, proxy_failed: 0 })
    )
    expect(wrapper.emitted('imported')).toBeUndefined()

    await wrapper.findAll('button.btn-secondary')[1]!.trigger('click')

    expect(wrapper.emitted('imported')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('导入分组选择器只显示 OpenAI 分组并使用导入专用文案', () => {
    const wrapper = mountModal()

    expect(wrapper.text()).toContain(t('admin.accounts.dataImportFileHint'))
    expect(wrapper.text()).toContain(t('admin.accounts.dataImportGroups'))
    expect(wrapper.text()).toContain(t('admin.accounts.dataImportGroupsHint'))
    expect(wrapper.text()).not.toContain('admin.accounts.dataImport')
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
    setInputFiles(input.element, [makeJsonFile('data.json', JSON.stringify(payload))])

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
