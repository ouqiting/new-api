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
import * as z from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

const createApiKeySettingsSchema = (t: (key: string) => string) =>
  z.object({
    token_setting: z.object({
      random_key_prefix_enabled: z.boolean(),
      key_prefix: z
        .string()
        .trim()
        .regex(/^[A-Za-z0-9]*$/, {
          message: t('API key prefix can only contain letters and numbers'),
        }),
    }),
  })

type ApiKeySettingsFormValues = z.infer<
  ReturnType<typeof createApiKeySettingsSchema>
>

type ApiKeySettingsSectionProps = {
  defaultValues: ApiKeySettingsFormValues
}

export function ApiKeySettingsSection({
  defaultValues,
}: ApiKeySettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const schema = createApiKeySettingsSchema(t)

  const { form, handleSubmit } = useSettingsForm<ApiKeySettingsFormValues>({
    resolver: zodResolver(schema),
    mode: 'onChange',
    defaultValues,
    onSubmit: async (_data, changedFields) => {
      for (const [key, value] of Object.entries(changedFields)) {
        const optionValue =
          typeof value === 'boolean' ||
          typeof value === 'number' ||
          typeof value === 'string'
            ? value
            : ''
        await updateOption.mutateAsync({ key, value: optionValue })
      }
    },
  })

  const randomPrefixEnabled = form.watch(
    'token_setting.random_key_prefix_enabled'
  )

  return (
    <SettingsSection title={t('API Key Settings')}>
      <Form {...form}>
        <SettingsForm onSubmit={handleSubmit}>
          <SettingsPageFormActions
            onSave={handleSubmit}
            isSaving={updateOption.isPending}
            saveLabel='Save API key settings'
          />

          <FormField
            control={form.control}
            name='token_setting.random_key_prefix_enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable random key prefix')}</FormLabel>
                  <FormDescription>
                    {t(
                      'When enabled, new API keys use two random letters as the prefix.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='token_setting.key_prefix'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Key prefix')}</FormLabel>
                <FormControl>
                  <Input
                    {...field}
                    disabled={randomPrefixEnabled}
                    placeholder='sk'
                  />
                </FormControl>
                <FormDescription>
                  {t('New API keys will look like prefix-XXXX.')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
