import { useQuery } from '@tanstack/react-query'
import { getPlugins } from '../api'

export function usePlugins() {
  return useQuery({
    queryKey: ['plugins'],
    queryFn: getPlugins,
    staleTime: 30 * 1000,
  })
}
