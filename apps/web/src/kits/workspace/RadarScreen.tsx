import { useState } from 'react';

import {
  Banner,
  Button,
  DefinitionList,
  Dialog,
  Panel,
  RadarList,
  Select,
  StatusBadge,
  Table,
  Tabs,
  TextArea,
  TextField,
  type RadarEntry,
  type TableColumn,
} from '../../design-system';
import { RADAR } from './fixtures';

function OverrideDialog({
  entry,
  onClose,
}: {
  readonly entry: RadarEntry;
  readonly onClose: () => void;
}) {
  return (
    <Dialog
      title={`Override ring · ${entry.project}`}
      onClose={onClose}
      size="md"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" onClick={onClose}>
            Save override
          </Button>
        </>
      }
    >
      <DefinitionList
        columns={2}
        dense
        items={[
          { label: 'Policy suggestion', value: entry.policyRing, mono: true },
          { label: 'Policy version', value: 'Default adoption policy v4', mono: true },
          { label: 'Current placement', value: entry.effectiveRing, mono: true },
          { label: 'Owner', value: 'Ana Silva · analyst' },
        ]}
      />
      <Select
        id="override-ring"
        label="Effective ring"
        options={['adopt', 'trial', 'assess', 'hold']}
        defaultValue={entry.effectiveRing}
      />
      <TextArea
        id="override-reason"
        label="Justification"
        required
        maxLength={280}
        counter
        hint="Required. Stored with your name, the policy version and the review date."
      />
      <TextField
        id="override-review"
        label="Review on"
        type="date"
        defaultValue="2026-11-20"
        size="md"
      />
      <Banner tone="info" title="An override does not change the policy result">
        The recommendation, the suggested ring and this override remain three separate attributed
        facts. A later policy change expires the override and says so; it never moves it silently.
      </Banner>
    </Dialog>
  );
}

const COLUMNS: readonly TableColumn<RadarEntry>[] = [
  { key: 'project', header: 'Project' },
  { key: 'policyRing', header: 'Policy suggestion', mono: true },
  { key: 'effectiveRing', header: 'Effective placement', mono: true },
  {
    key: 'override',
    header: 'Override',
    render: (row) =>
      row.override ? (
        <StatusBadge
          status={row.override.expired ? 'stale' : 'available'}
          label={row.override.expired ? `Expired · ${row.override.owner}` : row.override.owner}
          size="sm"
        />
      ) : (
        <span style={{ color: 'var(--text-tertiary)' }}>—</span>
      ),
  },
  {
    key: 'reviewDue',
    header: 'Review on',
    mono: true,
    render: (row) =>
      row.reviewDue ? (
        <span style={{ color: row.reviewOverdue ? 'var(--critical-fg)' : undefined }}>
          {row.reviewDue}
          {row.reviewOverdue ? ' · overdue' : ''}
        </span>
      ) : (
        <span style={{ color: 'var(--text-tertiary)' }}>—</span>
      ),
  },
];

export function RadarScreen() {
  const [entry, setEntry] = useState<RadarEntry | null>(null);
  const [view, setView] = useState('list');

  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Banner tone="info" title="Rings are derived from the active policy version">
        Placement comes from the explicit ring mapping in Default adoption policy v4. Analysts
        choose which projects are displayed and may override a ring with a justification and a
        review date.
      </Banner>
      <Tabs
        value={view}
        onChange={setView}
        size="sm"
        items={[
          { value: 'list', label: 'Rings' },
          { value: 'table', label: 'Table' },
        ]}
      />
      {view === 'list' ? (
        <Panel
          title="Technology radar"
          meta="Default adoption policy v4 · 90d · cutoff 2026-08-20T14:35:00Z"
          actions={
            <Button size="sm" variant="secondary" iconStart="download">
              Export
            </Button>
          }
        >
          <RadarList entries={RADAR} onSelect={setEntry} />
        </Panel>
      ) : (
        <Panel title="Technology radar" meta="Accessible table representation" padding="0">
          <Table
            caption="Radar placements with policy suggestion and override"
            columns={COLUMNS}
            rows={RADAR}
            getRowKey={(row) => row.project}
            onRowClick={setEntry}
          />
        </Panel>
      )}
      {entry ? <OverrideDialog entry={entry} onClose={() => setEntry(null)} /> : null}
    </div>
  );
}
