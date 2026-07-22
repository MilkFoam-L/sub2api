import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import UpstreamBalanceCell from '../UpstreamBalanceCell.vue'
import type { Account } from '@/types'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => params ? `${key}:${Object.values(params).join(',')}` : key })
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
    [{ status: 'failed' }, 'admin.accounts.upstreamBalance.failed'],
    [{ status: 'ok', data: { unlimited: true } }, 'admin.accounts.upstreamBalance.unlimited']
  ])('renders status %s', (snapshot, text) => {
    expect(mount(UpstreamBalanceCell, { props: { account: account(snapshot) } }).text()).toContain(text)
  })

  it('renders remaining balance and emits manual probe', async () => {
    const wrapper = mount(UpstreamBalanceCell, {
      props: { account: account({ status: 'ok', data: { source: 'newapi', mode: 'token_quota', remaining: 12.5, used: 2, limit: 14.5 }, received_at: '2026-01-01T00:00:00Z' }) }
    })
    expect(wrapper.get('[data-testid="upstream-balance-value"]').text()).toContain('12.5')
    await wrapper.get('[data-testid="upstream-balance-probe"]').trigger('click')
    expect(wrapper.emitted('probe')).toHaveLength(1)
  })

  it('renders raw quota when NewAPI unit conversion is unavailable', () => {
    const wrapper = mount(UpstreamBalanceCell, {
      props: { account: account({ status: 'ok', data: { source: 'newapi', mode: 'token_quota', raw_remaining: 700000, raw_unit: 'quota' }, received_at: '2026-01-01T00:00:00Z' }) }
    })
    expect(wrapper.get('[data-testid="upstream-balance-value"]').text()).toContain('700000 quota')
  })

  it('renders stale snapshots as expired', () => {
    const wrapper = mount(UpstreamBalanceCell, {
      props: { account: account({ status: 'ok', data: { remaining: 2 }, received_at: '2020-01-01T00:00:00Z', fresh_until: '2020-01-02T00:00:00Z' }), now: Date.parse('2020-01-03T00:00:00Z') }
    })
    expect(wrapper.text()).toContain('admin.accounts.upstreamBalance.stale')
  })
})
