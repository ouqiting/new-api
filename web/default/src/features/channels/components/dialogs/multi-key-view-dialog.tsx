/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useState, useEffect, useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Loader2, RefreshCw, Copy, Check, Power, PowerOff, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { StatusBadge } from '@/components/status-badge'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import {
  getMultiKeyStatus,
  enableMultiKey,
  disableMultiKey,
  deleteMultiKey,
} from '../../api'
import {
  channelsQueryKeys,
  getMultiKeyStatusConfig,
  getMultiKeyConfirmMessage,
  isDestructiveAction,
} from '../../lib'
import type { KeyStatus, MultiKeyConfirmAction } from '../../types'
import { useChannels } from '../channels-provider'

type MultiKeyViewDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Full channel key string (newline-separated), already revealed via verification */
  fullKeyString: string
  isLoading: boolean
}

type KeyRow = {
  index: number
  status: number
  key: string
}

export function MultiKeyViewDialog(props: MultiKeyViewDialogProps) {
  const { t } = useTranslation()
  const { currentRow } = useChannels()
  const queryClient = useQueryClient()

  const [keysStatus, setKeysStatus] = useState<KeyStatus[]>([])
  const [isLoadingStatus, setIsLoadingStatus] = useState(false)
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null)
  const [confirmAction, setConfirmAction] =
    useState<MultiKeyConfirmAction | null>(null)
  const [isPerformingAction, setIsPerformingAction] = useState(false)

  const loadStatus = useCallback(async () => {
    if (!currentRow) return
    setIsLoadingStatus(true)
    try {
      const response = await getMultiKeyStatus(currentRow.id, 1, 500, undefined)
      if (response.success && response.data) {
        setKeysStatus(response.data.keys || [])
      } else {
        toast.error(response.message || t('Failed to load key status'))
      }
    } catch (error: unknown) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to load key status')
      )
    } finally {
      setIsLoadingStatus(false)
    }
  }, [currentRow, t])

  useEffect(() => {
    if (props.open && currentRow) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      loadStatus()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.open, currentRow?.id])

  // Build merged rows: full keys from revealed string + status from API
  const fullKeys = (props.fullKeyString || '')
    .split('\n')
    .map((k) => k.trim())
    .filter((k) => k.length > 0)

  const statusMap = new Map<number, KeyStatus>()
  for (const k of keysStatus) {
    statusMap.set(k.index, k)
  }

  const rows: KeyRow[] = fullKeys.map((key, idx) => ({
    index: idx,
    status: statusMap.get(idx)?.status ?? 1,
    key,
  }))

  const handleCopy = useCallback(
    async (key: string, index: number) => {
      try {
        await navigator.clipboard.writeText(key)
        setCopiedIndex(index)
        toast.success(t('Copied'))
        setTimeout(() => setCopiedIndex(null), 1200)
      } catch {
        toast.error(t('Copy failed'))
      }
    },
    [t]
  )

  const performAction = async () => {
    if (!confirmAction || !currentRow) return
    const { type, keyIndex } = confirmAction
    if (keyIndex === undefined) {
      setConfirmAction(null)
      return
    }

    setIsPerformingAction(true)
    try {
      let response
      if (type === 'enable') {
        response = await enableMultiKey(currentRow.id, keyIndex)
      } else if (type === 'disable') {
        response = await disableMultiKey(currentRow.id, keyIndex)
      } else if (type === 'delete') {
        response = await deleteMultiKey(currentRow.id, keyIndex)
      }

      if (response?.success) {
        toast.success(response.message || t('Operation successful'))
        queryClient.invalidateQueries({ queryKey: channelsQueryKeys.lists() })
        await loadStatus()
      } else {
        toast.error(response?.message || t('Operation failed'))
      }
    } catch (error: unknown) {
      toast.error(
        error instanceof Error ? error.message : t('Operation failed')
      )
    } finally {
      setIsPerformingAction(false)
      setConfirmAction(null)
    }
  }

  if (!currentRow) return null

  const renderStatusBadge = (status: number) => {
    const config = getMultiKeyStatusConfig(status)
    return (
      <StatusBadge
        label={t(config.label)}
        variant={config.variant}
        showDot
        copyable={false}
      />
    )
  }

  return (
    <>
      <Dialog
        open={props.open}
        onOpenChange={props.onOpenChange}
        title={
          <>
            {t('Multi-Key Channel Keys')}
            <StatusBadge
              label={currentRow.name}
              variant='neutral'
              copyable={false}
            />
          </>
        }
        description={t(
          'Click a key to copy. Use actions to enable, disable, or delete a key.'
        )}
        contentClassName='flex max-h-[90vh] max-w-3xl flex-col'
        titleClassName='flex items-center gap-2'
        contentHeight='min(72vh, 680px)'
        bodyClassName='space-y-4'
      >
        <div className='flex min-h-0 flex-1 flex-col space-y-4 overflow-hidden'>
          {/* Toolbar */}
          <div className='flex shrink-0 items-center justify-end'>
            <Button
              variant='outline'
              size='sm'
              onClick={loadStatus}
              disabled={isLoadingStatus}
            >
              <RefreshCw className='h-4 w-4' />
            </Button>
          </div>

          {/* Table */}
          <div className='min-h-0 flex-1 overflow-auto rounded-md border'>
            {props.isLoading || isLoadingStatus ? (
              <div className='flex items-center justify-center py-12'>
                <Loader2 className='text-muted-foreground h-8 w-8 animate-spin' />
              </div>
            ) : rows.length === 0 ? (
              <div className='text-muted-foreground py-12 text-center'>
                {t('No keys found')}
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className='w-20'>{t('Index')}</TableHead>
                    <TableHead className='min-w-[280px]'>{t('Key')}</TableHead>
                    <TableHead className='w-32'>{t('Status')}</TableHead>
                    <TableHead className='w-44 text-right'>
                      {t('Actions')}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rows.map((row) => (
                    <TableRow key={row.index}>
                      <TableCell className='font-mono text-sm'>
                        #{row.index + 1}
                      </TableCell>
                      <TableCell>
                        <button
                          type='button'
                          onClick={() => handleCopy(row.key, row.index)}
                          className='group flex w-full items-center gap-2 text-left'
                          title={t('Click to copy')}
                        >
                          <span className='font-mono text-sm break-all'>
                            {row.key}
                          </span>
                          {copiedIndex === row.index ? (
                            <Check className='h-3.5 w-3.5 shrink-0 text-success' />
                          ) : (
                            <Copy className='text-muted-foreground h-3.5 w-3.5 shrink-0 opacity-0 transition-opacity group-hover:opacity-100' />
                          )}
                        </button>
                      </TableCell>
                      <TableCell>{renderStatusBadge(row.status)}</TableCell>
                      <TableCell>
                        <div className='flex justify-end gap-2'>
                          {row.status === 1 ? (
                            <Button
                              variant='outline'
                              size='sm'
                              onClick={() =>
                                setConfirmAction({
                                  type: 'disable',
                                  keyIndex: row.index,
                                })
                              }
                            >
                              <PowerOff className='mr-1 h-3.5 w-3.5' />
                              {t('Disable')}
                            </Button>
                          ) : (
                            <Button
                              variant='outline'
                              size='sm'
                              onClick={() =>
                                setConfirmAction({
                                  type: 'enable',
                                  keyIndex: row.index,
                                })
                              }
                            >
                              <Power className='mr-1 h-3.5 w-3.5' />
                              {t('Enable')}
                            </Button>
                          )}
                          <Button
                            variant='destructive'
                            size='sm'
                            onClick={() =>
                              setConfirmAction({
                                type: 'delete',
                                keyIndex: row.index,
                              })
                            }
                          >
                            <Trash2 className='mr-1 h-3.5 w-3.5' />
                            {t('Delete')}
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </div>
        </div>
      </Dialog>

      <ConfirmDialog
        open={confirmAction !== null}
        onOpenChange={(open) => !open && setConfirmAction(null)}
        title={t('Confirm Action')}
        desc={t(getMultiKeyConfirmMessage(confirmAction))}
        destructive={isDestructiveAction(confirmAction)}
        isLoading={isPerformingAction}
        handleConfirm={performAction}
      />
    </>
  )
}
