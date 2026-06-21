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
import { useCallback, useEffect, useState } from 'react'
import { KeyRound, Loader2, Power, PowerOff, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { formatTimestampToDate } from '@/lib/format'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { GroupBadge } from '@/components/group-badge'
import { StatusBadge } from '@/components/status-badge'
import { API_KEY_STATUS, API_KEY_STATUSES } from '@/features/keys/constants'
import {
  deleteAdminUserApiKey,
  getAdminUserApiKeys,
  updateAdminUserApiKeyStatus,
} from '../../api'
import type { AdminUserApiKey, User } from '../../types'

interface UserApiKeysDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: User | null
}

type ConfirmAction =
  | { type: 'delete'; apiKey: AdminUserApiKey }
  | { type: 'status'; apiKey: AdminUserApiKey; nextStatus: number }

const PAGE_SIZE = 20

export function UserApiKeysDialog(props: UserApiKeysDialogProps) {
  const { t } = useTranslation()
  const [apiKeys, setApiKeys] = useState<AdminUserApiKey[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [acting, setActing] = useState(false)
  const [confirmAction, setConfirmAction] = useState<ConfirmAction | null>(
    null
  )

  const loadApiKeys = useCallback(
    async (targetPage: number) => {
      if (!props.user?.id) return
      setLoading(true)
      try {
        const res = await getAdminUserApiKeys(props.user.id, {
          p: targetPage,
          size: PAGE_SIZE,
        })
        if (res.success && res.data) {
          setApiKeys(res.data.items || [])
          setTotal(res.data.total || 0)
          setPage(res.data.page || targetPage)
        } else {
          toast.error(res.message || t('Failed to load user API keys'))
        }
      } catch {
        toast.error(t('Failed to load user API keys'))
      } finally {
        setLoading(false)
      }
    },
    [props.user?.id, t]
  )

  useEffect(() => {
    if (props.open && props.user?.id) {
      setPage(1)
      loadApiKeys(1)
    } else {
      setApiKeys([])
      setTotal(0)
      setConfirmAction(null)
    }
  }, [props.open, props.user?.id, loadApiKeys])

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const handleConfirmAction = async () => {
    if (!confirmAction || !props.user?.id) return
    setActing(true)
    try {
      if (confirmAction.type === 'delete') {
        const res = await deleteAdminUserApiKey(
          props.user.id,
          confirmAction.apiKey.id
        )
        if (res.success) {
          toast.success(t('API Key deleted successfully'))
          await loadApiKeys(page)
        } else {
          toast.error(res.message || t('Failed to delete API key'))
        }
      } else {
        const res = await updateAdminUserApiKeyStatus(
          props.user.id,
          confirmAction.apiKey.id,
          confirmAction.nextStatus
        )
        if (res.success) {
          toast.success(
            confirmAction.nextStatus === API_KEY_STATUS.ENABLED
              ? t('API Key enabled successfully')
              : t('API Key disabled successfully')
          )
          await loadApiKeys(page)
        } else {
          toast.error(res.message || t('Failed to update API key status'))
        }
      }
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setActing(false)
      setConfirmAction(null)
    }
  }

  const renderLastUsed = (accessedTime: number) => {
    if (!accessedTime) {
      return <span className='text-muted-foreground'>-</span>
    }
    return (
      <span className='text-muted-foreground font-mono text-xs tabular-nums'>
        {formatTimestampToDate(accessedTime)}
      </span>
    )
  }

  const renderStatus = (apiKey: AdminUserApiKey) => {
    const config = API_KEY_STATUSES[apiKey.status]
    if (!config) {
      return (
        <StatusBadge
          label={t('Unknown')}
          variant='neutral'
          copyable={false}
        />
      )
    }
    return (
      <StatusBadge
        label={t(config.label)}
        variant={config.variant}
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
            <KeyRound className='size-5' />
            {t('Manage API Keys')}
          </>
        }
        description={
          props.user
            ? t('Manage API keys created by {{username}}', {
                username: props.user.username,
              })
            : t('Manage API keys created by this user')
        }
        contentClassName='sm:max-w-4xl'
        titleClassName='flex items-center gap-2'
        bodyClassName='flex flex-col gap-4'
      >
        <div className='rounded-md border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Name')}</TableHead>
                <TableHead>{t('Group')}</TableHead>
                <TableHead>{t('Status')}</TableHead>
                <TableHead>{t('Last Used')}</TableHead>
                <TableHead className='text-right'>{t('Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={5} className='py-8 text-center'>
                    <span className='inline-flex items-center gap-2'>
                      <Loader2 className='size-4 animate-spin' />
                      {t('Loading...')}
                    </span>
                  </TableCell>
                </TableRow>
              ) : apiKeys.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={5}
                    className='text-muted-foreground py-8 text-center'
                  >
                    {t('This user has not created any API keys')}
                  </TableCell>
                </TableRow>
              ) : (
                apiKeys.map((apiKey) => {
                  const isEnabled = apiKey.status === API_KEY_STATUS.ENABLED
                  const nextStatus = isEnabled
                    ? API_KEY_STATUS.DISABLED
                    : API_KEY_STATUS.ENABLED

                  return (
                    <TableRow key={apiKey.id}>
                      <TableCell>
                        <span className='font-medium'>{apiKey.name}</span>
                      </TableCell>
                      <TableCell>
                        <GroupBadge group={apiKey.group || ''} />
                      </TableCell>
                      <TableCell>{renderStatus(apiKey)}</TableCell>
                      <TableCell>
                        {renderLastUsed(apiKey.accessed_time)}
                      </TableCell>
                      <TableCell className='text-right'>
                        <div className='flex justify-end gap-2'>
                          <Button
                            size='sm'
                            variant='outline'
                            onClick={() =>
                              setConfirmAction({
                                type: 'status',
                                apiKey,
                                nextStatus,
                              })
                            }
                          >
                            {isEnabled ? (
                              <PowerOff data-icon='inline-start' />
                            ) : (
                              <Power data-icon='inline-start' />
                            )}
                            {isEnabled ? t('Disable') : t('Enable')}
                          </Button>
                          <Button
                            size='sm'
                            variant='destructive'
                            onClick={() =>
                              setConfirmAction({ type: 'delete', apiKey })
                            }
                          >
                            <Trash2 data-icon='inline-start' />
                            {t('Delete')}
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
            </TableBody>
          </Table>
        </div>

        <div className='flex items-center justify-between gap-3'>
          <p className='text-muted-foreground text-sm'>
            {t('Total')}: {total}
          </p>
          <div className='flex items-center gap-2'>
            <Button
              size='sm'
              variant='outline'
              disabled={loading || page <= 1}
              onClick={() => loadApiKeys(page - 1)}
            >
              {t('Previous')}
            </Button>
            <span className='text-muted-foreground text-sm'>
              {page} / {totalPages}
            </span>
            <Button
              size='sm'
              variant='outline'
              disabled={loading || page >= totalPages}
              onClick={() => loadApiKeys(page + 1)}
            >
              {t('Next')}
            </Button>
          </div>
        </div>
      </Dialog>

      <ConfirmDialog
        open={!!confirmAction}
        onOpenChange={(open) => !open && setConfirmAction(null)}
        title={getConfirmTitle(confirmAction, t)}
        desc={getConfirmDescription(confirmAction, t)}
        confirmText={getConfirmText(confirmAction, t)}
        destructive={confirmAction?.type === 'delete'}
        handleConfirm={handleConfirmAction}
        isLoading={acting}
      />
    </>
  )
}

function getConfirmTitle(
  confirmAction: ConfirmAction | null,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  if (confirmAction?.type === 'delete') return t('Delete API Key')
  if (confirmAction?.nextStatus === API_KEY_STATUS.ENABLED) {
    return t('Enable API Key')
  }
  return t('Disable API Key')
}

function getConfirmDescription(
  confirmAction: ConfirmAction | null,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  if (!confirmAction) return ''
  if (confirmAction.type === 'delete') {
    return t('Delete API key "{{name}}"? This action cannot be undone.', {
      name: confirmAction.apiKey.name,
    })
  }
  if (confirmAction.nextStatus === API_KEY_STATUS.ENABLED) {
    return t('Enable API key "{{name}}"?', {
      name: confirmAction.apiKey.name,
    })
  }
  return t('Disable API key "{{name}}"?', {
    name: confirmAction.apiKey.name,
  })
}

function getConfirmText(
  confirmAction: ConfirmAction | null,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  if (confirmAction?.type === 'delete') return t('Delete')
  if (confirmAction?.nextStatus === API_KEY_STATUS.ENABLED) return t('Enable')
  return t('Disable')
}
