import { describe, expect, it, vi } from 'vitest'

import type { UpdateSettingsRequest } from '@/api/admin/settings'
import { updateSettings } from '@/api/admin/settings'

const put = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    put,
  },
}))

describe('admin settings gateway scheduling payload', () => {
  it('keeps gateway scheduling in update settings payload', async () => {
    const payload = {
      gateway_scheduling: {
        score_weights: {
          load: 1,
          queue: 0.8,
          debt: 0.5,
          error_rate: 2,
          latency: 1.5,
          rate_multiplier: 0.7,
          quota_risk: 2.5,
        },
        latency_baseline_ms: 1800,
        quota_risk_threshold: 0.85,
        max_score_penalty: 8,
        sticky_session_mode: 'soft',
        sticky_escape_score_ratio: 2,
        sticky_escape_load_rate: 0.9,
        active_probe: {
          auto_pause_enabled: true,
          failure_threshold: 4,
          pause_duration: '10m',
          pause_duration_max: '1h',
        },
        slow_start: {
          enabled: true,
          duration: '5m',
          penalty: 1.2,
        },
      },
    } satisfies UpdateSettingsRequest

    put.mockResolvedValueOnce({ data: payload })

    await updateSettings(payload)

    expect(put).toHaveBeenCalledWith('/admin/settings', payload)
  })
})
