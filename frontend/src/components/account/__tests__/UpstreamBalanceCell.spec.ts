import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import UpstreamBalanceCell from '../UpstreamBalanceCell.vue'
import type { Account } from '@/types'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => params ? `${key}:${Object.values(params).join(',')}` : key })
}))

vi.mock('@/components/common/HelpTooltip.vue', () => ({
  default: { template: '<div><slot name="trigger" /><slot /></div>' }
}))

const account = (snapshot?: Record<string, unknown>): Account => ({
  id: 1, name: 'test', platform: 'openai', type: 'apikey', proxy_id: null,
  concurrency: 1, priority: 1, status: 'active', error_message: null,
  last_used_at: null, expires_at: null, auto_pause_on_expired: false,
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z', schedulable: true,
  extra: snapshot ? { upstream_balance_probe: snapshot } : undefined
})

describe('UpstreamBalanceCell', () => {
  it.each([
    [{ status: 'unsupported' }, 'admin.accounts.upstreamBalance.unsupported'],
    [{ status: 'failed', last_error: 'newapi_user_balance_unavailable' }, 'admin.accounts.upstreamBalance.failed']
  ])('renders status %s', (snapshot, text) => {
    expect(mount(UpstreamBalanceCell, { props: { account: account(snapshot) } }).text()).toContain(text)
  })

  it('renders converted balance and tooltip values as unitless numbers with one decimal', async () => {
    const wrapper = mount(UpstreamBalanceCell, {
      props: { account: account({ status: 'ok', data: { source: 'newapi_user', mode: 'wallet', remaining: 12.56, used: 2, currency: 'USD' }, received_at: '2026-01-01T00:00:00Z' }) }
    })
    expect(wrapper.get('[data-testid="upstream-balance-value"]').text()).toBe('12.6')
    expect(wrapper.text()).toContain('admin.accounts.upstreamBalance.cumulativeUsed:2.0')
    expect(wrapper.text()).not.toContain('admin.accounts.upstreamBalance.usedLimit')
    expect(wrapper.text()).not.toMatch(/USD|CNY| quota/)
    await wrapper.get('[data-testid="upstream-balance-probe"]').trigger('click')
    expect(wrapper.emitted('probe')).toHaveLength(1)
  })

  it.each([
    [-1.24, '-1.2'],
    [12, '12.0']
  ])('formats remaining %s as %s', (remaining, expected) => {
    const wrapper = mount(UpstreamBalanceCell, {
      props: { account: account({ status: 'ok', data: { remaining }, received_at: '2026-01-01T00:00:00Z' }) }
    })
    expect(wrapper.get('[data-testid="upstream-balance-value"]').text()).toBe(expected)
  })

  it('renders raw quota as a unitless number when conversion is unavailable', () => {
    const wrapper = mount(UpstreamBalanceCell, {
      props: { account: account({ status: 'ok', data: { source: 'newapi', mode: 'token_quota', raw_remaining: 700000, raw_used: 200000, raw_limit: 900000, raw_unit: 'quota' }, received_at: '2026-01-01T00:00:00Z' }) }
    })
    expect(wrapper.get('[data-testid="upstream-balance-value"]').text()).toBe('700000.0')
    expect(wrapper.text()).toContain('admin.accounts.upstreamBalance.usedLimit:200000.0,900000.0')
  })

  it('never labels an unlimited snapshot as unlimited', () => {
    const wrapper = mount(UpstreamBalanceCell, {
      props: { account: account({ status: 'ok', data: { unlimited: true, remaining: -12 }, received_at: '2026-01-01T00:00:00Z' }) }
    })
    expect(wrapper.get('[data-testid="upstream-balance-value"]').text()).toBe('admin.accounts.upstreamBalance.failed')
    expect(wrapper.text()).not.toContain('admin.accounts.upstreamBalance.unlimited')
  })

  it('renders stale snapshots as expired', () => {
    const wrapper = mount(UpstreamBalanceCell, {
      props: { account: account({ status: 'ok', data: { remaining: 2 }, received_at: '2020-01-01T00:00:00Z', fresh_until: '2020-01-02T00:00:00Z' }), now: Date.parse('2020-01-03T00:00:00Z') }
    })
    expect(wrapper.text()).toContain('admin.accounts.upstreamBalance.stale')
  })
})
