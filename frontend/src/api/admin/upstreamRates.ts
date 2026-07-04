import { apiClient } from '../client'

export type UpstreamRateSourceType = 'sub2api' | 'newapi'
export type UpstreamRateAuthMode = 'none' | 'bearer_token'
export type UpstreamRateTargetType = 'account' | 'group'
export type UpstreamRateRuleMode = 'first' | 'avg' | 'min' | 'max'

export interface UpstreamRateSource {
  id: number
  name: string
  source_type: UpstreamRateSourceType
  base_url: string
  auth_mode: UpstreamRateAuthMode
  token_configured: boolean
  recharge_multiplier: number
  sync_interval_seconds: number
  enabled: boolean
  use_for_scheduling: boolean
  last_sync_at?: string | null
  last_sync_status: string
  last_error: string
  created_at: string
  updated_at: string
}

export interface UpstreamRateSourcePayload {
  name: string
  source_type: UpstreamRateSourceType
  base_url: string
  auth_mode: UpstreamRateAuthMode
  token?: string
  clear_token?: boolean
  recharge_multiplier: number
  sync_interval_seconds: number
  enabled: boolean
  use_for_scheduling: boolean
}

export interface UpstreamRateSnapshot {
  ID?: number
  id?: number
  SourceID?: number
  source_id?: number
  UpstreamGroupKey?: string
  upstream_group_key?: string
  UpstreamGroupName?: string
  upstream_group_name?: string
  RawRateMultiplier?: number
  raw_rate_multiplier?: number
  EffectiveRateMultiplier?: number
  effective_rate_multiplier?: number
  Status?: string
  status?: string
  FetchedAt?: string
  fetched_at?: string
  ExpiresAt?: string
  expires_at?: string
}

export interface UpstreamRateBinding {
  ID?: number
  id?: number
  SourceID?: number
  source_id?: number
  SourceName?: string
  source_name?: string
  UpstreamGroupKey?: string
  upstream_group_key?: string
  TargetType?: UpstreamRateTargetType
  target_type?: UpstreamRateTargetType
  TargetID?: number
  target_id?: number
  TargetName?: string
  target_name?: string
  Mode?: UpstreamRateRuleMode
  mode?: UpstreamRateRuleMode
  Offset?: number
  offset?: number
  Enabled?: boolean
  enabled?: boolean
}

export interface UpstreamRateOverviewItem {
  SourceID?: number
  source_id?: number
  SourceName?: string
  source_name?: string
  SourceType?: string
  source_type?: string
  BaseURL?: string
  base_url?: string
  Enabled?: boolean
  enabled?: boolean
  UseForScheduling?: boolean
  use_for_scheduling?: boolean
  TokenConfigured?: boolean
  token_configured?: boolean
  LastSyncAt?: string | null
  last_sync_at?: string | null
  LastSyncStatus?: string
  last_sync_status?: string
  LastError?: string
  last_error?: string
  SnapshotCount?: number
  snapshot_count?: number
  HealthSuccessRate1h?: number
  health_success_rate_1h?: number
  HealthAvgLatencyMS1h?: number | null
  health_avg_latency_ms_1h?: number | null
  BindingCount?: number
  binding_count?: number
}

export interface UpstreamRateSyncResult {
  source_id: number
  status: string
  latency_ms: number
  snapshot_count: number
  error?: string
  snapshots?: UpstreamRateSnapshot[]
}

export async function listSources(): Promise<UpstreamRateSource[]> {
  const { data } = await apiClient.get<UpstreamRateSource[]>('/admin/scheduling/upstream-rates/sources')
  return data
}

export async function createSource(payload: UpstreamRateSourcePayload): Promise<UpstreamRateSource> {
  const { data } = await apiClient.post<UpstreamRateSource>('/admin/scheduling/upstream-rates/sources', payload)
  return data
}

export async function updateSource(id: number, payload: UpstreamRateSourcePayload): Promise<UpstreamRateSource> {
  const { data } = await apiClient.put<UpstreamRateSource>(`/admin/scheduling/upstream-rates/sources/${id}`, payload)
  return data
}

export async function deleteSource(id: number): Promise<void> {
  await apiClient.delete(`/admin/scheduling/upstream-rates/sources/${id}`)
}

export async function testSource(id: number): Promise<UpstreamRateSyncResult> {
  const { data } = await apiClient.post<UpstreamRateSyncResult>(`/admin/scheduling/upstream-rates/sources/${id}/test`)
  return data
}

export async function syncSource(id: number): Promise<UpstreamRateSyncResult> {
  const { data } = await apiClient.post<UpstreamRateSyncResult>(`/admin/scheduling/upstream-rates/sources/${id}/sync`)
  return data
}

export async function listSnapshots(sourceId: number): Promise<UpstreamRateSnapshot[]> {
  const { data } = await apiClient.get<UpstreamRateSnapshot[]>(`/admin/scheduling/upstream-rates/sources/${sourceId}/snapshots`)
  return data
}

export async function listBindings(): Promise<UpstreamRateBinding[]> {
  const { data } = await apiClient.get<UpstreamRateBinding[]>('/admin/scheduling/upstream-rates/bindings')
  return data
}

export async function listOverview(): Promise<UpstreamRateOverviewItem[]> {
  const { data } = await apiClient.get<UpstreamRateOverviewItem[]>('/admin/scheduling/upstream-rates/overview')
  return data
}

export const upstreamRatesAPI = {
  listSources,
  createSource,
  updateSource,
  deleteSource,
  testSource,
  syncSource,
  listSnapshots,
  listBindings,
  listOverview
}

export default upstreamRatesAPI
