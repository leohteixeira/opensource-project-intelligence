import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { Controller, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import { z } from 'zod';

import { Banner, Button, Panel, Select, TextField } from '../../design-system';
import { deleteAccount, savePreferences } from '../api';
import { queryClient } from '../query';
import { useApplication } from '../router';
import { routePath } from '../routes';

const preferencesSchema = z.object({
  locale: z.enum(['en', 'pt-BR']),
  timezone: z.string().trim().min(1),
});
type PreferencesValues = z.infer<typeof preferencesSchema>;

const deletionSchema = z.object({ confirmation: z.literal('DELETE MY ACCOUNT') });
type DeletionValues = z.infer<typeof deletionSchema>;

export function AccountScreen() {
  const { t } = useTranslation();
  const { session, locale } = useApplication();
  const navigate = useNavigate();
  const preferences = useForm<PreferencesValues>({
    resolver: zodResolver(preferencesSchema),
    defaultValues: {
      locale: session.member?.locale === 'pt-BR' ? 'pt-BR' : 'en',
      timezone: session.member?.timezone ?? 'UTC',
    },
  });
  const deletion = useForm<DeletionValues>({ resolver: zodResolver(deletionSchema) });
  const save = useMutation({
    mutationFn: (values: PreferencesValues) =>
      savePreferences(session, values.locale, values.timezone),
    onSuccess: (member) => {
      queryClient.setQueryData(['session'], { ...session, member });
    },
  });
  const remove = useMutation({
    mutationFn: (values: DeletionValues) => deleteAccount(session, values.confirmation),
    onSuccess: () => {
      queryClient.setQueryData(['session'], { state: 'anonymous', authenticated: false });
      window.setTimeout(() => navigate(routePath(locale, 'catalog')), 1200);
    },
  });

  return (
    <div style={{ display: 'grid', gap: 'var(--space-3)', maxWidth: 760 }}>
      <h1 style={{ font: 'var(--type-page)' }}>{t('account')}</h1>
      {save.isSuccess ? (
        <Banner tone="positive" title={t('saved')}>
          {t('accountUpdated')}
        </Banner>
      ) : null}
      {save.isError ? (
        <Banner
          tone="critical"
          title={errorStatus(save.error) === 412 ? t('conflict') : t('requestFailed')}
        />
      ) : null}
      <Panel title={t('preferences')}>
        <form
          onSubmit={preferences.handleSubmit((values) => save.mutate(values))}
          style={{ display: 'grid', gap: 'var(--space-2)' }}
        >
          <Controller
            control={preferences.control}
            name="locale"
            render={({ field, fieldState }) => (
              <Select
                id="account-locale"
                label={t('locale')}
                value={field.value}
                onChange={field.onChange}
                error={fieldState.error?.message}
                options={[
                  { value: 'en', label: t('localeEnglish') },
                  { value: 'pt-BR', label: t('localePortuguese') },
                ]}
              />
            )}
          />
          <TextField
            id="account-timezone"
            label={t('timezone')}
            error={preferences.formState.errors.timezone?.message}
            {...preferences.register('timezone')}
          />
          <Button type="submit" pending={save.isPending}>
            {t('save')}
          </Button>
        </form>
      </Panel>

      {remove.isSuccess ? <Banner tone="positive" title={t('deletionQueued')} /> : null}
      {remove.isError ? <Banner tone="critical" title={t('requestFailed')} /> : null}
      <Panel title={t('deletion')} tone="critical">
        <p style={{ marginBottom: 'var(--space-1)' }}>{t('deletionHelp')}</p>
        <Banner
          tone="critical"
          title={t('deletionWarning')}
          style={{ marginBottom: 'var(--space-2)' }}
        />
        <form
          onSubmit={deletion.handleSubmit((values) => remove.mutate(values))}
          style={{ display: 'grid', gap: 'var(--space-2)' }}
        >
          <TextField
            id="account-deletion-confirmation"
            label={t('deletionConfirmation')}
            error={deletion.formState.errors.confirmation?.message}
            {...deletion.register('confirmation')}
          />
          <Button type="submit" variant="danger" pending={remove.isPending}>
            {t('deleteAction')}
          </Button>
        </form>
      </Panel>
    </div>
  );
}

function errorStatus(error: Error): number | undefined {
  return (error as Error & { status?: number }).status;
}
