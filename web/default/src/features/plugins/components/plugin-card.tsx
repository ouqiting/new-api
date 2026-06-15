import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import type { Plugin } from '../types'

interface PluginCardProps {
  plugin: Plugin
  onToggle: (pluginId: string, enabled: boolean) => void
  isPending: boolean
}

export function PluginCard(props: PluginCardProps) {
  const { t } = useTranslation()
  const { plugin, onToggle, isPending } = props

  return (
    <div className='bg-card text-card-foreground flex flex-col justify-between gap-4 rounded-lg border p-4 shadow-sm'>
      <div className='space-y-2'>
        <div className='flex items-start justify-between gap-2'>
          <h3 className='text-sm font-semibold leading-tight'>{plugin.title}</h3>
          {plugin.version && (
            <Badge variant='secondary' className='text-xs'>
              v{plugin.version}
            </Badge>
          )}
        </div>
        <p className='text-muted-foreground text-xs leading-relaxed'>
          {plugin.description}
        </p>
        {plugin.hooks && plugin.hooks.length > 0 && (
          <div className='flex flex-wrap gap-1'>
            {plugin.hooks.map((hook) => (
              <Badge key={hook} variant='outline' className='text-[10px]'>
                {hook}
              </Badge>
            ))}
          </div>
        )}
        {plugin.error && (
          <p className='text-destructive text-xs'>{plugin.error}</p>
        )}
      </div>
      <div className='flex items-center justify-end gap-2'>
        <span className='text-muted-foreground text-xs'>
          {plugin.enabled ? t('Enabled') : t('Disabled')}
        </span>
        <Switch
          checked={plugin.enabled}
          onCheckedChange={(checked) => onToggle(plugin.id, checked)}
          disabled={isPending || plugin.error !== undefined}
          size='sm'
        />
      </div>
    </div>
  )
}
