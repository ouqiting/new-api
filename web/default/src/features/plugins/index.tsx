import { useTranslation } from 'react-i18next'
import { RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { SectionPageLayout } from '@/components/layout'
import { PluginCard } from './components/plugin-card'
import { usePlugins } from './hooks/use-plugins'
import { useReloadPlugins } from './hooks/use-reload-plugins'
import { useTogglePlugin } from './hooks/use-toggle-plugin'

export function Plugins() {
  const { t } = useTranslation()
  const { data, isLoading } = usePlugins()
  const toggleMutation = useTogglePlugin()
  const reloadMutation = useReloadPlugins()

  const plugins = data?.data ?? []

  const handleToggle = (pluginId: string, enabled: boolean) => {
    toggleMutation.mutate({ plugin_id: pluginId, enabled })
  }

  const handleReload = () => {
    reloadMutation.mutate()
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Plugins')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          size='sm'
          onClick={handleReload}
          disabled={reloadMutation.isPending}
        >
          <RefreshCw className='mr-2 h-4 w-4' />
          {t('Reload')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        {isLoading ? (
          <div className='text-muted-foreground text-sm'>{t('Loading...')}</div>
        ) : plugins.length === 0 ? (
          <div className='text-muted-foreground text-sm'>{t('No plugins found')}</div>
        ) : (
          <div className='grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3'>
            {plugins.map((plugin) => (
              <PluginCard
                key={plugin.id}
                plugin={plugin}
                onToggle={handleToggle}
                isPending={toggleMutation.isPending}
              />
            ))}
          </div>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
