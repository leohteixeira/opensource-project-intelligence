import { useState } from 'react';

import {
  Button,
  DefinitionList,
  FilterBar,
  Pagination,
  Panel,
  Select,
  StatusBadge,
  Table,
  TextField,
  type TableColumn,
} from '../../design-system';
import { AUDIT_EVENTS, type AuditEvent } from './fixtures';

const COLUMNS: readonly TableColumn<AuditEvent>[] = [
  { key: 'at', header: 'Time (UTC)', mono: true },
  { key: 'actor', header: 'Actor' },
  { key: 'action', header: 'Action', mono: true },
  { key: 'resource', header: 'Resource', mono: true },
  {
    key: 'result',
    header: 'Result',
    render: (row) => (
      <StatusBadge
        status={row.result === 'succeeded' ? 'succeeded' : 'available'}
        label={row.result === 'succeeded' ? 'Succeeded' : 'Coalesced'}
        size="sm"
      />
    ),
  },
];

export function AuditScreen() {
  const [selected, setSelected] = useState<AuditEvent | null>(AUDIT_EVENTS[0] ?? null);

  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <FilterBar
        resultLabel={`${AUDIT_EVENTS.length} events`}
        applied={[{ key: 'from', field: 'from', value: '2026-08-18' }]}
      >
        <TextField
          id="audit-search"
          type="search"
          iconStart="search"
          size="md"
          placeholder="Search actor, action or resource"
        />
        <Select
          id="audit-action"
          size="md"
          placeholder="All actions"
          options={[
            'member.suspended',
            'project.deleted',
            'sync.requested',
            'export.created',
            'association.corrected',
            'alert_rule.created',
          ]}
        />
        <Select
          id="audit-actor"
          size="md"
          placeholder="All actors"
          options={['people', 'service accounts', 'system']}
        />
      </FilterBar>
      <Panel
        title="Audit events"
        meta="immutable · no secrets or sensitive payloads"
        padding="0"
        actions={
          <Button size="sm" variant="secondary" iconStart="download">
            Export JSON
          </Button>
        }
      >
        <Table
          caption="Audit events"
          getRowKey={(row) => row.id}
          selectedKey={selected?.id}
          onRowClick={setSelected}
          columns={COLUMNS}
          rows={AUDIT_EVENTS}
        />
      </Panel>
      <Pagination page={1} hasMore total={AUDIT_EVENTS.length} label="Events" />
      {selected ? (
        <Panel title="Event detail" eyebrow={selected.action} meta={selected.id}>
          <DefinitionList
            columns={2}
            items={[
              { label: 'Time', value: selected.at, mono: true },
              { label: 'Actor', value: selected.actor },
              { label: 'Resource', value: selected.resource, mono: true },
              { label: 'Result', value: selected.result },
              { label: 'Safe state change', value: selected.detail },
              { label: 'Request id', value: 'req_TkRhQwW7KjvPGf6B0a2J9Q', mono: true },
            ]}
          />
        </Panel>
      ) : null}
    </div>
  );
}
