import { useState } from 'react';

import {
  Banner,
  Button,
  EmptyState,
  EvidenceLink,
  FilterBar,
  Menu,
  Panel,
  Select,
  StatusBadge,
  Table,
  TextField,
  type EvidenceKind,
  type StatusKey,
  type TableColumn,
} from '../../design-system';
import { ALERTS, type AlertRow } from './fixtures';

const SEVERITY_STATUS: Record<AlertRow['severity'], StatusKey> = {
  critical: 'failed',
  attention: 'conditional',
  info: 'observed',
};

const SEVERITY_LABEL: Record<AlertRow['severity'], string> = {
  critical: 'Critical',
  attention: 'Attention',
  info: 'Info',
};

const STATE_STATUS: Record<AlertRow['state'], StatusKey> = {
  open: 'failed',
  acknowledged: 'conditional',
  resolved: 'succeeded',
};

function evidenceKind(evidence: string): EvidenceKind {
  if (evidence.startsWith('release')) return 'release';
  if (evidence.startsWith('forecast')) return 'run';
  if (evidence.startsWith('metric')) return 'metric';

  return 'changelog';
}

export function AlertsScreen() {
  const [alerts, setAlerts] = useState<readonly AlertRow[]>(ALERTS);
  const [selectedId, setSelectedId] = useState<string | undefined>(ALERTS[0]?.id);

  const transition = (id: string, state: AlertRow['state']) =>
    setAlerts((current) =>
      current.map((alert) => (alert.id === id ? { ...alert, state, read: true } : alert)),
    );

  const current = alerts.find((alert) => alert.id === selectedId);
  const unread = alerts.filter((alert) => !alert.read).length;

  const columns: readonly TableColumn<AlertRow>[] = [
    {
      key: 'read',
      header: '',
      width: '18px',
      render: (row) =>
        row.read ? null : (
          <span
            aria-label="unread"
            style={{
              display: 'block',
              width: 7,
              height: 7,
              borderRadius: 'var(--radius-pill)',
              background: 'var(--info-solid)',
            }}
          />
        ),
    },
    {
      key: 'severity',
      header: 'Severity',
      render: (row) => (
        <StatusBadge
          status={SEVERITY_STATUS[row.severity]}
          label={SEVERITY_LABEL[row.severity]}
          size="sm"
        />
      ),
    },
    {
      key: 'title',
      header: 'Occurrence',
      wrap: true,
      render: (row) => (
        <span style={{ display: 'grid', gap: 1 }}>
          <span
            style={{
              fontWeight: row.read ? 'var(--weight-regular)' : 'var(--weight-semibold)',
            }}
          >
            {row.title}
          </span>
          <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>
            {row.project} · {row.rule} {row.ruleVersion} · {row.evidence}
          </span>
        </span>
      ),
    },
    {
      key: 'occurrences',
      header: 'Occurrences',
      numeric: true,
      render: (row) => (row.occurrences > 1 ? `${row.occurrences} (deduplicated)` : '1'),
    },
    { key: 'detected', header: 'Detected', mono: true },
    {
      key: 'state',
      header: 'State',
      render: (row) => (
        <StatusBadge
          status={STATE_STATUS[row.state]}
          label={row.state.charAt(0).toUpperCase() + row.state.slice(1)}
          size="sm"
        />
      ),
    },
  ];

  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Banner tone="info" title="Alerts are in-app only">
        Occurrences are shared by the workspace; read state is per member. Acknowledgement,
        resolution and dismissal are attributed transitions with a justification where they change
        team interpretation.
      </Banner>
      <FilterBar
        resultLabel={`${alerts.length} occurrences · ${unread} unread`}
        applied={[{ key: 'state', field: 'state', value: 'open or acknowledged' }]}
      >
        <TextField
          id="alert-search"
          type="search"
          iconStart="search"
          size="md"
          placeholder="Search occurrences"
        />
        <Select
          id="alert-severity"
          size="md"
          placeholder="All severities"
          options={['critical', 'attention', 'info']}
        />
        <Select
          id="alert-rule"
          size="md"
          placeholder="All rules"
          options={[
            'Public security evidence',
            'Contributor concentration',
            'Breaking change',
            'Early warning',
          ]}
        />
        <Button variant="secondary" size="md" iconStart="plus">
          New rule
        </Button>
      </FilterBar>
      <Panel
        title="Inbox"
        padding="0"
        meta="cooldown 24h · repeated qualifying evidence updates the occurrence"
      >
        <Table
          caption="Alert occurrences"
          columns={columns}
          rows={alerts}
          selectedKey={selectedId}
          getRowKey={(row) => row.id}
          onRowClick={(row) => {
            setSelectedId(row.id);
            transition(row.id, row.state);
          }}
        />
      </Panel>
      {current ? (
        <Panel
          title={current.title}
          status={
            <StatusBadge status={STATE_STATUS[current.state]} label={current.state} size="sm" />
          }
          meta={`${current.rule} ${current.ruleVersion} · detected ${current.detected} · ${current.project}`}
          actions={
            <>
              <Button
                size="sm"
                variant="secondary"
                onClick={() => transition(current.id, 'acknowledged')}
              >
                Acknowledge
              </Button>
              <Button
                size="sm"
                variant="secondary"
                onClick={() => transition(current.id, 'resolved')}
              >
                Resolve
              </Button>
              <Menu
                triggerLabel="More"
                items={[
                  { label: 'Dismiss', icon: 'circle-slash' },
                  { label: 'Reopen', icon: 'rotate-ccw' },
                  { separator: true },
                  { label: 'Edit rule', icon: 'pencil' },
                ]}
              />
            </>
          }
        >
          <div style={{ display: 'grid', gap: 'var(--space-075)' }}>
            <EvidenceLink
              kind={evidenceKind(current.evidence)}
              title={current.evidence}
              source={current.project}
              collectedAt={current.detected}
              href="#"
            />
            <p style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
              Deduplication key: rule {current.ruleVersion} · {current.project} · 90d window.
              Repeated qualifying evidence inside the cooldown updates this occurrence rather than
              creating another.
            </p>
          </div>
        </Panel>
      ) : (
        <EmptyState title="No occurrence selected">
          Select a row to see its evidence and lifecycle.
        </EmptyState>
      )}
    </div>
  );
}
