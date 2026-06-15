import { api } from '@/lib/api'
import type { PluginListResponse, TogglePluginRequest, TogglePluginResponse } from './types'

export async function getPlugins(): Promise<PluginListResponse> {
  const res = await api.get('/api/plugin')
  return res.data
}

export async function togglePlugin(request: TogglePluginRequest): Promise<TogglePluginResponse> {
  const res = await api.put('/api/plugin', request)
  return res.data
}

export async function reloadPlugins(): Promise<TogglePluginResponse> {
  const res = await api.post('/api/plugin/reload')
  return res.data
}
