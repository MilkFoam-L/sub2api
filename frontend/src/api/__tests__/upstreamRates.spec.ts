import { describe, expect, it, vi } from 'vitest'

import upstreamRatesAPI from '@/api/admin/upstreamRates'

const { get, post, put, del } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
    put,
    delete: del,
  },
}))

describe('admin upstream rates API', () => {
  it('uses scheduling upstream-rates endpoints', async () => {
    get.mockResolvedValueOnce({ data: [] })
    post.mockResolvedValueOnce({ data: { id: 1 } })
    put.mockResolvedValueOnce({ data: { id: 1 } })
    del.mockResolvedValueOnce({ data: { deleted: true } })

    await upstreamRatesAPI.listOverview()
    await upstreamRatesAPI.createSource({
      name: 'NewAPI',
      source_type: 'newapi',
      base_url: 'https://example.com',
      auth_mode: 'none',
      recharge_multiplier: 1,
      sync_interval_seconds: 300,
      enabled: true,
      use_for_scheduling: false,
    })
    await upstreamRatesAPI.updateSource(1, {
      name: 'NewAPI',
      source_type: 'newapi',
      base_url: 'https://example.com',
      auth_mode: 'none',
      recharge_multiplier: 1,
      sync_interval_seconds: 300,
      enabled: true,
      use_for_scheduling: true,
    })
    await upstreamRatesAPI.deleteSource(1)

    expect(get).toHaveBeenCalledWith('/admin/scheduling/upstream-rates/overview')
    expect(post).toHaveBeenCalledWith('/admin/scheduling/upstream-rates/sources', expect.objectContaining({ source_type: 'newapi' }))
    expect(put).toHaveBeenCalledWith('/admin/scheduling/upstream-rates/sources/1', expect.objectContaining({ use_for_scheduling: true }))
    expect(del).toHaveBeenCalledWith('/admin/scheduling/upstream-rates/sources/1')
  })
})
