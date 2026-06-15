import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { togglePlugin } from '../api'
import type { TogglePluginRequest } from '../types'

export function useTogglePlugin() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: TogglePluginRequest) => togglePlugin(request),
    onSuccess: (data, variables) => {
      if (data.success) {
        queryClient.invalidateQueries({ queryKey: ['plugins'] })
        toast.success(
          variables.enabled ? t('Plugin enabled successfully') : t('Plugin disabled successfully')
        )
      } else {
        toast.error(data.message || t('Failed to update plugin'))
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to update plugin'))
    },
  })
}
