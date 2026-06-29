export interface PluginManifest {
  id: string
  title: string
  description: string
  version: string
  author?: string
  entry?: string
  hooks?: string[]
  capabilities?: string[]
  log?: boolean
  config?: Record<string, unknown>
}

export interface Plugin extends PluginManifest {
  enabled: boolean
  loaded: boolean
  error?: string
}

export interface PluginListResponse {
  success: boolean
  message: string
  data: Plugin[]
}

export interface TogglePluginResponse {
  success: boolean
  message: string
}

export interface TogglePluginRequest {
  plugin_id: string
  enabled: boolean
}
