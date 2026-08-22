import { useMutation, useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';

import {
  Banner,
  Button,
  DefinitionList,
  EmptyState,
  Panel,
  StatusBadge,
  Table,
  type StatusKey,
  type TableColumn,
} from '../../design-system';
import {
  approveMember,
  fetchAudit,
  fetchMembers,
  fetchOperations,
  fetchServiceAccounts,
  type Document,
} from '../api';
import { queryClient } from '../query';
import { useApplication } from '../router';

export type AdminSurface = 'members' | 'serviceAccounts' | 'audit' | 'operations';

export function AdminScreen({ surface }: { surface: AdminSurface }) {
  if (surface === 'members') return <MembersScreen />;
  if (surface === 'serviceAccounts') return <ServiceAccountsScreen />;
  if (surface === 'audit') return <AuditScreen />;
  return <OperationsScreen />;
}

function MembersScreen() {
  const { t } = useTranslation();
  const { session, narrow } = useApplication();
  const members = useQuery({ queryKey: ['admin-members'], queryFn: fetchMembers });
  const decide = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: 'approve' | 'reject' }) =>
      approveMember(session, id, decision),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['admin-members'] }),
  });
  const columns = useMemo<readonly TableColumn<Document>[]>(
    () => [
      { key: 'display_name', header: t('name') },
      { key: 'email', header: t('email') },
      { key: 'role', header: t('role') },
      { key: 'status', header: t('state'), render: statusCell },
      {
        key: 'action',
        header: t('action'),
        render: (row) =>
          stringValue(row.status) === 'applicant' ? (
            <span style={{ display: 'flex', gap: 'var(--space-05)', flexWrap: 'wrap' }}>
              <Button
                size="sm"
                onClick={() => decide.mutate({ id: stringValue(row.id), decision: 'approve' })}
              >
                {t('approveViewer')}
              </Button>
              <Button
                size="sm"
                variant="secondary"
                onClick={() => decide.mutate({ id: stringValue(row.id), decision: 'reject' })}
              >
                {t('reject')}
              </Button>
            </span>
          ) : null,
      },
    ],
    [decide, t],
  );
  return (
    <AdminTablePage
      title={t('members')}
      rows={members.data?.items ?? []}
      columns={columns}
      loading={members.isPending}
      error={members.isError || decide.isError}
      success={decide.isSuccess ? t('memberActionComplete') : undefined}
      narrow={narrow}
      retry={() => void members.refetch()}
    />
  );
}

function ServiceAccountsScreen() {
  const { t } = useTranslation();
  const { narrow } = useApplication();
  const accounts = useQuery({
    queryKey: ['admin-service-accounts'],
    queryFn: fetchServiceAccounts,
  });
  const columns: readonly TableColumn<Document>[] = [
    { key: 'name', header: t('name') },
    { key: 'external_subject', header: t('subject'), mono: true },
    { key: 'role', header: t('role') },
    {
      key: 'scopes',
      header: t('scopes'),
      render: (row) => arrayValue(row.scopes).join(', ') || '—',
    },
    { key: 'status', header: t('state'), render: statusCell },
  ];
  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Banner tone="neutral" title={t('serviceAccounts')}>
        {t('serviceAccountHelp')}
      </Banner>
      <AdminTablePage
        title={t('serviceAccounts')}
        rows={accounts.data?.items ?? []}
        columns={columns}
        loading={accounts.isPending}
        error={accounts.isError}
        narrow={narrow}
        retry={() => void accounts.refetch()}
      />
    </div>
  );
}

function AuditScreen() {
  const { t } = useTranslation();
  const { narrow } = useApplication();
  const audit = useQuery({ queryKey: ['admin-audit'], queryFn: fetchAudit });
  const columns: readonly TableColumn<Document>[] = [
    { key: 'occurred_at', header: t('occurredAt'), mono: true },
    { key: 'action', header: t('action'), mono: true },
    { key: 'actor_kind', header: t('role') },
    { key: 'resource_type', header: t('resource') },
    { key: 'outcome', header: t('outcome'), render: statusCell },
  ];
  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Banner tone="neutral" title={t('audit')}>
        {t('auditImmutable')}
      </Banner>
      <AdminTablePage
        title={t('audit')}
        rows={audit.data?.items ?? []}
        columns={columns}
        loading={audit.isPending}
        error={audit.isError}
        narrow={narrow}
        retry={() => void audit.refetch()}
      />
    </div>
  );
}

function OperationsScreen() {
  const { t } = useTranslation();
  const operations = useQuery({ queryKey: ['admin-operations'], queryFn: fetchOperations });
  if (operations.isError) {
    return (
      <Banner
        tone="critical"
        title={t('requestFailed')}
        actions={<Button onClick={() => void operations.refetch()}>{t('retry')}</Button>}
      />
    );
  }
  const health = recordValue(operations.data?.health);
  const capabilities = recordValue(operations.data?.capabilities);
  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <h1 style={{ font: 'var(--type-page)' }}>{t('operations')}</h1>
      <Banner tone="neutral" title={t('operations')}>
        {t('redactedOperations')}
      </Banner>
      <Panel title={t('health')}>
        <DefinitionList
          items={Object.entries(health).map(([label, value]) => ({
            label,
            value: safeValue(value),
            mono: true,
          }))}
        />
      </Panel>
      <Panel title={t('capabilities')}>
        <DefinitionList
          items={Object.entries(capabilities).map(([label, value]) => ({
            label,
            value: value ? t('enabled') : t('disabled'),
          }))}
        />
      </Panel>
    </div>
  );
}

interface AdminTablePageProps {
  title: string;
  rows: Document[];
  columns: readonly TableColumn<Document>[];
  loading: boolean;
  error: boolean;
  success?: string;
  narrow: boolean;
  retry: () => void;
}

function AdminTablePage(props: AdminTablePageProps) {
  const { t } = useTranslation();
  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <h1 style={{ font: 'var(--type-page)' }}>{props.title}</h1>
      {props.success ? <Banner tone="positive" title={props.success} /> : null}
      {props.error ? (
        <Banner
          tone="critical"
          title={t('requestFailed')}
          actions={<Button onClick={props.retry}>{t('retry')}</Button>}
        />
      ) : null}
      {props.loading ? <p aria-live="polite">{t('loading')}</p> : null}
      {!props.loading ? (
        <Table
          caption={props.title}
          rows={props.rows}
          columns={props.columns}
          layout={props.narrow ? 'detail' : 'table'}
          getRowKey={(row, index) => stringValue(row.id) || String(index)}
          empty={<EmptyState title={t('noRows')} />}
        />
      ) : null}
    </div>
  );
}

function statusCell(row: Document) {
  const state = stringValue(row.status || row.outcome);
  const statusMap: Record<string, StatusKey> = {
    active: 'available',
    success: 'available',
    approved: 'available',
    applicant: 'queued',
    pending: 'queued',
    suspended: 'failed',
    rejected: 'failed',
    denied: 'failed',
    failure: 'failed',
  };
  return (
    <StatusBadge size="sm" status={statusMap[state] ?? 'unknown'} label={state || 'unknown'} />
  );
}

function stringValue(value: unknown): string {
  return typeof value === 'string' || typeof value === 'number' ? String(value) : '';
}

function arrayValue(value: unknown): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string')
    : [];
}

function recordValue(value: unknown): Record<string, unknown> {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}

function safeValue(value: unknown): string {
  return typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean'
    ? String(value)
    : 'unknown';
}
