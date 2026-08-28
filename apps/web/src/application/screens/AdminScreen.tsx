import { useMutation, useQuery } from '@tanstack/react-query';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  Banner,
  Button,
  DefinitionList,
  EmptyState,
  Panel,
  Select,
  StatusBadge,
  Table,
  TextField,
  type StatusKey,
  type TableColumn,
} from '../../design-system';
import {
  approveMember,
  createServiceAccount,
  fetchAudit,
  fetchMembers,
  fetchOperations,
  fetchServiceAccounts,
  type Document,
  updateMember,
  updateServiceAccount,
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
          ) : (
            <MemberUpdateActions
              row={row}
              session={session}
              onComplete={() => void queryClient.invalidateQueries({ queryKey: ['admin-members'] })}
            />
          ),
      },
    ],
    [decide, session, t],
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
  const { narrow, session } = useApplication();
  const [name, setName] = useState('');
  const [subject, setSubject] = useState('');
  const [role, setRole] = useState<'viewer' | 'analyst'>('viewer');
  const [scopes, setScopes] = useState('projects:read');
  const accounts = useQuery({
    queryKey: ['admin-service-accounts'],
    queryFn: fetchServiceAccounts,
  });
  const create = useMutation({
    mutationFn: () =>
      createServiceAccount(session, {
        name,
        external_subject: subject,
        role,
        scopes: scopes
          .split(',')
          .map((scope) => scope.trim())
          .filter(Boolean),
      }),
    onSuccess: () => {
      setName('');
      setSubject('');
      void queryClient.invalidateQueries({ queryKey: ['admin-service-accounts'] });
    },
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
    {
      key: 'action',
      header: t('action'),
      render: (row) => (
        <ServiceAccountActions
          row={row}
          session={session}
          onComplete={() =>
            void queryClient.invalidateQueries({ queryKey: ['admin-service-accounts'] })
          }
        />
      ),
    },
  ];
  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Banner tone="neutral" title={t('serviceAccounts')}>
        {t('serviceAccountHelp')}
      </Banner>
      <Panel title={t('create')}>
        <form
          onSubmit={(event) => {
            event.preventDefault();
            create.mutate();
          }}
          style={{ display: 'grid', gap: 'var(--space-1)' }}
        >
          <TextField
            id="service-name"
            label={t('name')}
            value={name}
            onChange={(event) => setName(event.target.value)}
            required
          />
          <TextField
            id="service-subject"
            label={t('externalSubject')}
            value={subject}
            onChange={(event) => setSubject(event.target.value)}
            required
          />
          <Select
            id="service-role"
            label={t('role')}
            value={role}
            onChange={(event) => setRole(event.target.value as 'viewer' | 'analyst')}
            options={[
              { value: 'viewer', label: 'Viewer' },
              { value: 'analyst', label: 'Analyst' },
            ]}
          />
          <TextField
            id="service-scopes"
            label={t('scopesHint')}
            value={scopes}
            onChange={(event) => setScopes(event.target.value)}
            required
          />
          {create.isError ? <Banner tone="critical" title={t('requestFailed')} /> : null}
          <Button type="submit" pending={create.isPending}>
            {t('create')}
          </Button>
        </form>
      </Panel>
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

function MemberUpdateActions({
  row,
  session,
  onComplete,
}: {
  row: Document;
  session: Parameters<typeof updateMember>[0];
  onComplete: () => void;
}) {
  const { t } = useTranslation();
  const [role, setRole] = useState<'viewer' | 'analyst' | 'admin'>(() => {
    const current = stringValue(row.role);
    return current === 'analyst' || current === 'admin' ? current : 'viewer';
  });
  const [state, setState] = useState<'active' | 'suspended'>(() =>
    stringValue(row.status) === 'suspended' ? 'suspended' : 'active',
  );
  const update = useMutation({
    mutationFn: () =>
      updateMember(session, stringValue(row.id), numberValue(row.version), role, state),
    onSuccess: onComplete,
  });
  return (
    <span style={{ display: 'flex', gap: 'var(--space-05)', flexWrap: 'wrap', alignItems: 'end' }}>
      <Select
        id={`member-role-${stringValue(row.id)}`}
        placeholder={t('role')}
        value={role}
        onChange={(event) => setRole(event.target.value as typeof role)}
        options={[
          { value: 'viewer', label: 'Viewer' },
          { value: 'analyst', label: 'Analyst' },
          { value: 'admin', label: 'Admin' },
        ]}
      />
      <Select
        id={`member-state-${stringValue(row.id)}`}
        placeholder={t('state')}
        value={state}
        onChange={(event) => setState(event.target.value as typeof state)}
        options={[
          { value: 'active', label: t('activate') },
          { value: 'suspended', label: t('suspend') },
        ]}
      />
      <Button
        size="sm"
        variant="secondary"
        pending={update.isPending}
        onClick={() => update.mutate()}
      >
        {t('update')}
      </Button>
    </span>
  );
}

function ServiceAccountActions({
  row,
  session,
  onComplete,
}: {
  row: Document;
  session: Parameters<typeof updateServiceAccount>[0];
  onComplete: () => void;
}) {
  const { t } = useTranslation();
  const suspended = stringValue(row.status) === 'suspended';
  const update = useMutation({
    mutationFn: () =>
      updateServiceAccount(
        session,
        stringValue(row.id),
        numberValue(row.version),
        stringValue(row.role) === 'analyst' ? 'analyst' : 'viewer',
        suspended ? 'active' : 'suspended',
        arrayValue(row.scopes),
      ),
    onSuccess: onComplete,
  });
  return (
    <Button
      size="sm"
      variant="secondary"
      pending={update.isPending}
      onClick={() => update.mutate()}
    >
      {suspended ? t('activate') : t('suspend')}
    </Button>
  );
}

function AuditScreen() {
  const { t } = useTranslation();
  const { narrow } = useApplication();
  const [draft, setDraft] = useState({
    actor: '',
    action: '',
    resource: '',
    outcome: '',
    from: '',
    to: '',
  });
  const [filters, setFilters] = useState(draft);
  const audit = useQuery({
    queryKey: ['admin-audit', filters],
    queryFn: () => fetchAudit(filters),
  });
  const columns: readonly TableColumn<Document>[] = [
    { key: 'occurred_at', header: t('occurredAt'), mono: true },
    { key: 'action', header: t('action'), mono: true },
    { key: 'actor_kind', header: t('role') },
    { key: 'actor_id', header: t('actor'), mono: true },
    { key: 'resource_type', header: t('resource') },
    { key: 'outcome', header: t('outcome'), render: statusCell },
  ];
  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Banner tone="neutral" title={t('audit')}>
        {t('auditImmutable')}
      </Banner>
      <Panel title={t('audit')}>
        <form
          onSubmit={(event) => {
            event.preventDefault();
            setFilters({ ...draft });
          }}
          style={{
            display: 'grid',
            gridTemplateColumns: narrow ? '1fr' : 'repeat(3, minmax(0, 1fr))',
            gap: 'var(--space-1)',
            alignItems: 'end',
          }}
        >
          <TextField
            id="audit-actor"
            label={t('actor')}
            value={draft.actor}
            onChange={(event) => setDraft((value) => ({ ...value, actor: event.target.value }))}
          />
          <TextField
            id="audit-action"
            label={t('action')}
            value={draft.action}
            onChange={(event) => setDraft((value) => ({ ...value, action: event.target.value }))}
          />
          <TextField
            id="audit-resource"
            label={t('resource')}
            value={draft.resource}
            onChange={(event) => setDraft((value) => ({ ...value, resource: event.target.value }))}
          />
          <Select
            id="audit-outcome"
            label={t('outcome')}
            placeholder={t('outcome')}
            value={draft.outcome}
            onChange={(event) => setDraft((value) => ({ ...value, outcome: event.target.value }))}
            options={['succeeded', 'failed', 'denied', 'stale']}
          />
          <TextField
            id="audit-from"
            label={t('from')}
            type="date"
            value={draft.from}
            onChange={(event) => setDraft((value) => ({ ...value, from: event.target.value }))}
          />
          <TextField
            id="audit-to"
            label={t('to')}
            type="date"
            value={draft.to}
            onChange={(event) => setDraft((value) => ({ ...value, to: event.target.value }))}
          />
          <Button type="submit" variant="primary">
            {t('applyFilters')}
          </Button>
        </form>
      </Panel>
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
  const { locale } = useApplication();
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
  const healthItems: Array<[string, unknown]> = Object.keys(health).length
    ? Object.entries(health)
    : [['status', operations.data?.status ?? 'unavailable']];
  const capabilities = arrayRecordValue(operations.data?.capabilities);
  const modelProvider = recordValue(operations.data?.model_provider);
  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <h1 style={{ font: 'var(--type-page)' }}>{t('operations')}</h1>
      <Banner tone="neutral" title={t('operations')}>
        {t('redactedOperations')}
      </Banner>
      <Panel title={t('health')}>
        <DefinitionList
          items={healthItems.map(([label, value]) => ({
            label,
            value: safeValue(value),
            mono: true,
          }))}
        />
      </Panel>
      <Panel title={t('capabilities')}>
        <DefinitionList
          items={capabilities.map((capability) => ({
            label: stringValue(capability.name),
            value: `${capability.configured ? t('enabled') : t('disabled')} · ${safeValue(capability.status)}`,
          }))}
        />
      </Panel>
      {Object.keys(modelProvider).length ? (
        <Panel title={locale === 'pt-BR' ? 'Provedor de modelo' : 'Model provider'}>
          <DefinitionList
            items={[
              'provider',
              'model',
              'health',
              'enabled',
              'capabilities',
              'revision',
              'usage',
              'redacted',
            ].map((label) => ({
              label,
              value:
                label === 'capabilities'
                  ? arrayValue(modelProvider[label]).join(', ') || '—'
                  : label === 'usage'
                    ? JSON.stringify(recordValue(modelProvider[label]))
                    : safeValue(modelProvider[label]),
              mono: true,
            }))}
          />
        </Panel>
      ) : null}
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

function numberValue(value: unknown): number {
  return typeof value === 'number' ? value : Number(value) || 0;
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

function arrayRecordValue(value: unknown): Record<string, unknown>[] {
  return Array.isArray(value)
    ? value.filter(
        (item): item is Record<string, unknown> =>
          Boolean(item) && typeof item === 'object' && !Array.isArray(item),
      )
    : [];
}

function safeValue(value: unknown): string {
  return typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean'
    ? String(value)
    : 'unknown';
}
