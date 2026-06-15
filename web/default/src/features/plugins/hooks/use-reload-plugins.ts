import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { reloadPlugins } from '../api'

export function useReloadPlugins() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => reloadPlugins(),
    onSuccess: (data) => {
      if (data.success) {
        queryClient.invalidateQueries({ queryKey: ['plugins'] })
        toast.success(t('Plugins reloaded successfully'))
      } else {
        toast.error(data.message || t('Failed to reload plugins'))
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to reload plugins'))
    },
  })
}
