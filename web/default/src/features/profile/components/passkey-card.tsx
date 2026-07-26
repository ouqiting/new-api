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
import { useCallback, useMemo, useState } from 'react'
import {
  KeyRound,
  Loader2,
  Pencil,
  Plus,
  ShieldAlert,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import dayjs from '@/lib/dayjs'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { StatusBadge } from '@/components/status-badge'
import {
  usePasskeyManagement,
  type PasskeyCredentialInfo,
} from '@/features/auth/passkey'
import {
  SecureVerificationDialog,
  useSecureVerification,
  type VerificationMethod,
  type VerificationMethods,
} from '@/features/auth/secure-verification'

interface PasskeyCardProps {
  loading: boolean
}

export function PasskeyCard({ loading: pageLoading }: PasskeyCardProps) {
  const { t } = useTranslation()
  const [pendingDelete, setPendingDelete] =
    useState<PasskeyCredentialInfo | null>(null)
  const [renameTarget, setRenameTarget] =
    useState<PasskeyCredentialInfo | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const [renaming, setRenaming] = useState(false)
  const [restrictedMethod, setRestrictedMethod] =
    useState<VerificationMethod | null>(null)

  const {
    loading,
    registering,
    removing,
    supported,
    enabled,
    passkeys,
    register,
    remove,
    rename,
  } = usePasskeyManagement()

  const {
    open: verificationOpen,
    setOpen: setVerificationOpen,
    methods: verificationMethods,
    state: verificationState,
    startVerification,
    executeVerification,
    cancel: cancelVerification,
    setCode,
    switchMethod,
    fetchVerificationMethods,
  } = useSecureVerification({
    onSuccess: () => {
      setRestrictedMethod(null)
    },
  })

  const dialogMethods = useMemo<VerificationMethods>(() => {
    if (!restrictedMethod) return verificationMethods
    return {
      ...verificationMethods,
      has2FA: restrictedMethod === '2fa' && verificationMethods.has2FA,
      hasPasskey:
        restrictedMethod === 'passkey' && verificationMethods.hasPasskey,
    }
  }, [restrictedMethod, verificationMethods])

  const handleRegister = useCallback(async () => {
    if (!supported) {
      toast.info(t('This device does not support Passkey'))
      return
    }

    const methods = await fetchVerificationMethods()
    if (!methods.has2FA) {
      // Without 2FA enabled, register directly. The browser-level Passkey prompt
      // is itself a strong proof of presence, so no extra verification is needed.
      await register()
      return
    }

    setRestrictedMethod('2fa')
    await startVerification(register, {
      preferredMethod: '2fa',
      title: t('Security verification'),
      description: t(
        'Confirm your identity with Two-factor Authentication before registering a Passkey.'
      ),
    })
  }, [fetchVerificationMethods, register, startVerification, supported, t])

  const handleRemove = useCallback(
    async (id: number) => {
      const methods = await fetchVerificationMethods()
      const required: VerificationMethod | null = methods.has2FA
        ? '2fa'
        : methods.hasPasskey
          ? 'passkey'
          : null

      if (!required) {
        toast.error(
          t(
            'Please enable Two-factor Authentication or Passkey before proceeding'
          )
        )
        return
      }

      if (required === 'passkey' && !methods.passkeySupported) {
        toast.info(t('This device does not support Passkey'))
        return
      }

      setPendingDelete(null)
      setRestrictedMethod(required)
      await startVerification(() => remove(id), {
        preferredMethod: required,
        title: t('Security verification'),
        description: t(
          'Confirm your identity before removing this Passkey from your account.'
        ),
      })
    },
    [fetchVerificationMethods, remove, startVerification, t]
  )

  const handleRenameSubmit = useCallback(async () => {
    if (!renameTarget) return
    const trimmed = renameValue.trim()
    if (!trimmed) {
      toast.error(t('Please enter a name'))
      return
    }
    setRenaming(true)
    try {
      const ok = await rename(renameTarget.id, trimmed)
      if (ok) {
        setRenameTarget(null)
      }
    } finally {
      setRenaming(false)
    }
  }, [rename, renameTarget, renameValue, t])

  const handleVerificationCancel = useCallback(() => {
    setRestrictedMethod(null)
    cancelVerification()
  }, [cancelVerification])

  const handleVerificationOpenChange = useCallback(
    (next: boolean) => {
      if (!next) {
        setRestrictedMethod(null)
      }
      setVerificationOpen(next)
    },
    [setVerificationOpen]
  )

  // Adapt the hook's `Promise<unknown>` return into the dialog's
  // `void | Promise<void>` signature without losing error propagation
  // semantics (errors are surfaced via toast inside the hook).
  const handleDialogVerify = useCallback(
    async (method: VerificationMethod, code?: string) => {
      try {
        await executeVerification(method, code)
      } catch {
        // Errors are already surfaced by useSecureVerification via toast.
      }
    },
    [executeVerification]
  )

  if (pageLoading || loading) {
    return (
      <Card className='gap-0 overflow-hidden py-0'>
        <CardHeader className='p-3 sm:p-5'>
          <Skeleton className='h-6 w-48' />
          <Skeleton className='mt-2 h-4 w-64' />
        </CardHeader>
        <CardContent className='p-3 sm:p-5'>
          <Skeleton className='h-20 w-full' />
        </CardContent>
      </Card>
    )
  }

  const formatLastUsed = (lastUsed?: string | null) =>
    lastUsed && !Number.isNaN(Date.parse(lastUsed))
      ? dayjs(lastUsed).fromNow()
      : t('Not used yet')

  const showUnsupportedNotice = !supported && !enabled

  return (
    <>
      <Card className='gap-0 overflow-hidden py-0'>
        <CardHeader className='p-3 sm:p-5'>
          <CardTitle className='text-lg tracking-tight sm:text-xl'>
            {t('Passkey Login')}
          </CardTitle>
          <CardDescription className='text-xs sm:text-sm'>
            {t('Use Passkey to sign in without entering your password.')}
          </CardDescription>
        </CardHeader>

        <CardContent className='p-3 sm:p-5'>
          <div className='space-y-6'>
            <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between xl:flex-col 2xl:flex-row'>
              <div className='flex items-start gap-4'>
                <div className='bg-muted rounded-md p-2'>
                  <KeyRound className='h-5 w-5' />
                </div>
                <div className='space-y-1'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <p className='font-medium'>{t('Passkey Authentication')}</p>
                    <StatusBadge
                      label={enabled ? t('Enabled') : t('Disabled')}
                      variant={enabled ? 'success' : 'neutral'}
                      showDot
                      copyable={false}
                    />
                  </div>
                  <p className='text-muted-foreground text-sm'>
                    {enabled
                      ? t('{{count}} passkeys registered', {
                          count: passkeys.length,
                        })
                      : t(
                          'Add a Passkey to sign in with your fingerprint, face, or security key.'
                        )}
                  </p>
                </div>
              </div>

              <Button
                className='w-full sm:w-auto xl:w-full 2xl:w-auto'
                onClick={handleRegister}
                disabled={!supported || registering}
              >
                {registering ? (
                  <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                ) : (
                  <Plus className='mr-2 h-4 w-4' />
                )}
                {t('Add Passkey')}
              </Button>
            </div>

            {passkeys.length > 0 && (
              <div className='divide-y rounded-md border'>
                {passkeys.map((passkey) => (
                  <div
                    key={passkey.id}
                    className='flex flex-col gap-3 p-3 sm:flex-row sm:items-center sm:justify-between'
                  >
                    <div className='min-w-0 space-y-1'>
                      <div className='flex flex-wrap items-center gap-2'>
                        <p className='truncate font-medium'>
                          {passkey.name || t('Passkey')}
                        </p>
                        {passkey.backup_eligible !== undefined && (
                          <StatusBadge
                            label={
                              passkey.backup_eligible
                                ? passkey.backup_state
                                  ? t('Backed up')
                                  : t('Not backed up')
                                : t('No backup')
                            }
                            variant={
                              passkey.backup_eligible
                                ? passkey.backup_state
                                  ? 'success'
                                  : 'warning'
                                : 'neutral'
                            }
                            showDot
                            copyable={false}
                          />
                        )}
                      </div>
                      <p className='text-muted-foreground text-sm'>
                        {t('Last used:')} {formatLastUsed(passkey.last_used_at)}
                      </p>
                    </div>
                    <div className='flex shrink-0 items-center gap-2'>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => {
                          setRenameTarget(passkey)
                          setRenameValue(passkey.name || '')
                        }}
                      >
                        <Pencil className='mr-1.5 h-3.5 w-3.5' />
                        {t('Rename')}
                      </Button>
                      <Button
                        variant='destructive'
                        size='sm'
                        disabled={removing}
                        onClick={() => setPendingDelete(passkey)}
                      >
                        {removing ? (
                          <Loader2 className='mr-1.5 h-3.5 w-3.5 animate-spin' />
                        ) : (
                          <Trash2 className='mr-1.5 h-3.5 w-3.5' />
                        )}
                        {t('Remove')}
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {showUnsupportedNotice && (
              <div className='bg-muted/60 text-muted-foreground flex items-start gap-3 rounded-md p-4 text-sm'>
                <ShieldAlert className='mt-0.5 h-4 w-4 flex-shrink-0 text-amber-500' />
                <div>
                  <p className='text-foreground font-medium'>
                    {t('Passkey not supported on this device')}
                  </p>
                  <p>
                    {t(
                      'Use a compatible browser or device with biometric authentication or a security key to register a Passkey.'
                    )}
                  </p>
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Remove Passkey?')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'This Passkey will no longer be usable for signing in. You can re-register anytime.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={removing}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
              disabled={removing}
              onClick={(event) => {
                event.preventDefault()
                if (pendingDelete) {
                  handleRemove(pendingDelete.id)
                }
              }}
            >
              {t('Remove')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog
        open={renameTarget !== null}
        onOpenChange={(open) => {
          if (!open) setRenameTarget(null)
        }}
      >
        <DialogContent className='sm:max-w-sm'>
          <DialogHeader>
            <DialogTitle>{t('Rename Passkey')}</DialogTitle>
          </DialogHeader>
          <Input
            value={renameValue}
            maxLength={64}
            placeholder={t('Passkey name')}
            onChange={(event) => setRenameValue(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault()
                handleRenameSubmit()
              }
            }}
          />
          <DialogFooter>
            <Button
              variant='outline'
              disabled={renaming}
              onClick={() => setRenameTarget(null)}
            >
              {t('Cancel')}
            </Button>
            <Button disabled={renaming} onClick={handleRenameSubmit}>
              {renaming && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
              {t('Save')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <SecureVerificationDialog
        open={verificationOpen}
        onOpenChange={handleVerificationOpenChange}
        methods={dialogMethods}
        state={verificationState}
        onVerify={handleDialogVerify}
        onCancel={handleVerificationCancel}
        onCodeChange={setCode}
        onMethodChange={switchMethod}
      />
    </>
  )
}
