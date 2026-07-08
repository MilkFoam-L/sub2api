import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountActionMenu from '../AccountActionMenu.vue'
import type { Account } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function makeAccount(overrides: Partial<Account> = {}): Account {
  return {
    id: 1,
    name: 'account',
    platform: 'openai',
    type: 'apikey',
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: true,
    created_at: '2026-06-16T00:00:00Z',
    updated_at: '2026-06-16T00:00:00Z',
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    ...overrides,
  }
}

function mountMenu(account: Account) {
  return mount(AccountActionMenu, {
    props: {
      show: true,
      account,
      position: { top: 10, left: 20 }
    },
    global: {
      stubs: {
        Teleport: true,
        Icon: true,
      }
    }
  })
}

describe('AccountActionMenu', () => {
  it.each([
    ['standard insufficient balance', 'Upstream no balance (INSUFFICIENT_BALANCE): Insufficient account balance'],
    ['newapi insufficient user quota', 'API returned 403: {"error":{"message":"用户额度不足, 剩余额度: ＄-0.003692","type":"new_api_error","param":"","code":"insufficient_user_quota"}}'],
  ])('上游无余额账号显示恢复并启用快捷按钮：%s', (_name, errorMessage) => {
    const wrapper = mountMenu(makeAccount({
      status: 'error',
      schedulable: false,
      error_message: errorMessage,
    }))

    expect(wrapper.text()).toContain('admin.accounts.recoverUpstreamBalance')
  })

  it('普通错误账号不显示恢复并启用快捷按钮', () => {
    const wrapper = mountMenu(makeAccount({
      status: 'error',
      schedulable: false,
      error_message: 'API returned 500: upstream server error',
    }))

    expect(wrapper.text()).not.toContain('admin.accounts.recoverUpstreamBalance')
    expect(wrapper.text()).toContain('admin.accounts.recoverState')
  })

  it('点击恢复并启用快捷按钮会派发事件并关闭菜单', async () => {
    const account = makeAccount({
      status: 'error',
      schedulable: false,
      error_message: 'Upstream no balance (INSUFFICIENT_BALANCE): Insufficient account balance',
    })
    const wrapper = mountMenu(account)

    const button = wrapper.findAll('button').find((item) => item.text().includes('admin.accounts.recoverUpstreamBalance'))
    expect(button).toBeTruthy()

    await button!.trigger('click')

    expect(wrapper.emitted('recover-upstream-balance')).toEqual([[account]])
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('OpenAI OAuth 母账号显示 401team 可重试开关并派发下一状态', async () => {
    const account = makeAccount({
      platform: 'openai',
      type: 'oauth',
      parent_account_id: null,
      credentials: { openai_team_401_retryable: false },
    })
    const wrapper = mountMenu(account)

    const button = wrapper.findAll('button').find((item) => item.text().includes('admin.accounts.enableOpenAITeam401Retryable'))
    expect(button).toBeTruthy()

    await button!.trigger('click')

    expect(wrapper.emitted('toggle-openai-team-401-retryable')).toEqual([[account, true]])
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('已开启的 OpenAI OAuth 母账号显示关闭 401team 可重试', () => {
    const wrapper = mountMenu(makeAccount({
      platform: 'openai',
      type: 'oauth',
      parent_account_id: null,
      credentials: { openai_team_401_retryable: true },
    }))

    expect(wrapper.text()).toContain('admin.accounts.disableOpenAITeam401Retryable')
  })

  it('非 OpenAI OAuth 母账号不显示 401team 可重试开关', () => {
    expect(mountMenu(makeAccount({ platform: 'openai', type: 'apikey' })).text()).not.toContain('admin.accounts.enableOpenAITeam401Retryable')
    expect(mountMenu(makeAccount({ platform: 'antigravity', type: 'oauth' })).text()).not.toContain('admin.accounts.enableOpenAITeam401Retryable')
    expect(mountMenu(makeAccount({ platform: 'openai', type: 'oauth', parent_account_id: 42 })).text()).not.toContain('admin.accounts.enableOpenAITeam401Retryable')
  })
})
