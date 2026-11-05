import api from './client'
export interface Key { id: string; name: string; enabled: boolean; active: boolean; expires_at: number; created_at: number; quota_bytes: number; used_bytes: number; xray_synced: boolean; sync_error: string; hysteria_link?: string; vless_link?: string; subscription_url?: string }
export interface UsagePoint { date: string; bytes: number }
export type UsageGranularity = 'minute' | 'hour' | 'day'
interface ApiResponse<T> { ok: boolean; data: T; error?: string }
export const keyApi = {
  async list(): Promise<Key[]> { const { data } = await api.get<ApiResponse<Key[]>>('/keys'); return data.data || [] },
  async usage(id: string, granularity: UsageGranularity): Promise<UsagePoint[]> { const { data } = await api.get<ApiResponse<UsagePoint[]>>(`/keys/${encodeURIComponent(id)}/usage`, { params: { granularity } }); return data.data || [] },
  async create(form: { name: string; days: number; quota_gb: number }): Promise<Key> { const { data } = await api.post<ApiResponse<Key>>('/keys', form); return data.data },
  async change(id: string, action: 'revoke' | 'enable' | 'renew', days?: number): Promise<Key> { const { data } = await api.post<ApiResponse<Key>>(`/keys/${encodeURIComponent(id)}/${action}`, days ? { days } : {}); return data.data },
  async reconcile(): Promise<void> { await api.post('/reconcile') },
}
