import { describe, expect, it, vi } from 'vitest'

import { updateConfig } from '@/api/admin/scheduling'
import type { GatewaySchedulingSettings } from '@/api/admin/settings'

const { put } = vi.hoisted(() => ({
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    put,
  },
}))

describe('admin scheduling config payload', () => {
  it('saves gateway scheduling through the dedicated scheduling endpoint', async () => {
    const payload = {
      preferred_account_id: 0,
      preferred_account_by_group_id: { '1': 42 },
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
      sticky_escape_load_rate: 90,
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
      upstream_rate: {
        enabled: false,
        stale_ttl_seconds: 600,
        rate_weight: 0.6,
        health_weight: 0.4,
        min_success_rate: 0.8,
      },
    } satisfies GatewaySchedulingSettings

    put.mockResolvedValueOnce({ data: payload })

    await updateConfig(payload)

    expect(put).toHaveBeenCalledWith('/admin/scheduling/config', payload)
  })
})
