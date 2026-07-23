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

const account = (
  snapshot?: Record<string, unknown>,
  extra: Record<string, unknown> = {}
): Account => ({
  id: 1, name: 'test', platform: 'openai', type: 'apikey', proxy_id: null,
  concurrency: 1, priority: 1, status: 'active', error_message: null,
  last_used_at: null, expires_at: null, auto_pause_on_expired: false,
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z', schedulable: true,
  extra: snapshot || Object.keys(extra).length
    ? { ...extra, ...(snapshot ? { upstream_balance_probe: snapshot } : {}) }
    : undefined
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

  it('renders stale snapshots as expired with the last successful balance details', () => {
    const wrapper = mount(UpstreamBalanceCell, {
      props: {
        account: account({
          status: 'ok',
          data: { remaining: 2 },
          received_at: '2020-01-01T00:00:00Z',
          fresh_until: '2020-01-02T00:00:00Z',
          next_probe_at: '2020-01-03T01:00:00Z'
        }, { upstream_balance_probe_enabled: true }),
        now: Date.parse('2020-01-03T00:00:00Z'),
        globalProbeEnabled: true
      }
    })
    expect(wrapper.get('[data-testid="upstream-balance-value"]').text()).toBe('admin.accounts.upstreamBalance.stale')
    expect(wrapper.text()).toContain('admin.accounts.upstreamBalance.lastDetectedBalance:2.0')
    expect(wrapper.text()).toContain('admin.accounts.upstreamBalance.lastDetectedAt:')
    expect(wrapper.text()).toContain('admin.accounts.upstreamBalance.elapsedSince:admin.accounts.upstreamBalance.daysAgo:2')
    expect(wrapper.find('[data-testid="upstream-balance-next-probe"]').exists()).toBe(true)
  })

  it('shows the next refresh time and mirrors account and global probe states', async () => {
    const wrapper = mount(UpstreamBalanceCell, {
      props: {
        account: account({
          status: 'ok',
          data: { remaining: 12.56 },
          received_at: '2026-07-23T00:00:00Z',
          fresh_until: '2026-07-23T02:00:00Z',
          next_probe_at: '2026-07-23T01:00:00Z'
        }, { upstream_balance_probe_enabled: true }),
        now: Date.parse('2026-07-23T00:30:00Z'),
        globalProbeEnabled: true
      }
    })

    expect(wrapper.get('[data-testid="upstream-balance-next-probe"]').text()).toContain(
      'admin.accounts.upstreamBalance.nextProbeAt:'
    )
    expect(wrapper.get('[data-testid="upstream-balance-probe-state"] span').classes()).toContain('text-emerald-400')
    expect(wrapper.find('[data-testid="upstream-balance-global-probe-state"]').exists()).toBe(false)

    await wrapper.setProps({ globalProbeEnabled: false })
    expect(wrapper.find('[data-testid="upstream-balance-next-probe"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="upstream-balance-global-probe-state"] span').classes()).toContain('text-red-400')

    await wrapper.setProps({
      globalProbeEnabled: true,
      account: account(undefined, { upstream_balance_probe_enabled: false })
    })
    expect(wrapper.find('[data-testid="upstream-balance-next-probe"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="upstream-balance-probe-state"] span').classes()).toContain('text-red-400')
  })

  it('keeps a fresh retained balance visible while marking the latest probe as failed', () => {
    const wrapper = mount(UpstreamBalanceCell, {
      props: {
        account: account({
          status: 'failed',
          data: { remaining: 12.56 },
          received_at: '2026-07-23T00:00:00Z',
          fresh_until: '2026-07-23T01:00:00Z',
          last_attempt_at: '2026-07-23T00:30:00Z',
          next_probe_at: '2026-07-23T01:00:00Z',
          last_error: 'http_error'
        }, { upstream_balance_probe_enabled: true }),
        now: Date.parse('2026-07-23T00:45:00Z'),
        globalProbeEnabled: true
      }
    })

    expect(wrapper.get('[data-testid="upstream-balance-value"]').text()).toBe('12.6')
    expect(wrapper.text()).toContain('admin.accounts.upstreamBalance.failed')
    expect(wrapper.text()).not.toContain('admin.accounts.upstreamBalance.stale')
  })

  it('hides an invalid next refresh timestamp', () => {
    const wrapper = mount(UpstreamBalanceCell, {
      props: {
        account: account({
          status: 'failed',
          last_attempt_at: '2026-07-23T00:00:00Z',
          next_probe_at: 'not-a-time'
        }, { upstream_balance_probe_enabled: true }),
        globalProbeEnabled: true
      }
    })

    expect(wrapper.find('[data-testid="upstream-balance-next-probe"]').exists()).toBe(false)
  })
})
