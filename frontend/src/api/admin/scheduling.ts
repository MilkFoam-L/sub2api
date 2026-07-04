import { apiClient } from '../client'
import type { GatewaySchedulingSettings } from './settings'

export type SchedulingStickyStatus = 'hit' | 'miss' | 'rebound' | 'wait_plan' | string

export interface SchedulingLogEvent {
  created_at: string
  platform?: string
  model?: string
  group_id?: number
  candidate_count: number
  available_count: number
  account_id?: number
  account_name?: string
  preferred_account_id?: number
  preferred_hit: boolean
  sticky_status?: SchedulingStickyStatus
  reason: string
  filter_summary?: Record<string, number>
  request_id?: string
  client_request_id?: string
}

export async function getConfig(): Promise<GatewaySchedulingSettings> {
  const { data } = await apiClient.get<GatewaySchedulingSettings>('/admin/scheduling/config')
  return data
}

export async function updateConfig(config: GatewaySchedulingSettings): Promise<GatewaySchedulingSettings> {
  const { data } = await apiClient.put<GatewaySchedulingSettings>('/admin/scheduling/config', config)
  return data
}

export async function listLogs(limit = 100): Promise<SchedulingLogEvent[]> {
  const { data } = await apiClient.get<SchedulingLogEvent[]>('/admin/scheduling/logs', {
    params: { limit }
  })
  return data
}

export const schedulingAPI = {
  getConfig,
  updateConfig,
  listLogs
}

export default schedulingAPI
