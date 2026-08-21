import { useState } from 'react';

import {
  Banner,
  DefinitionList,
  Menu,
  Panel,
  Recommendation,
  StatusBadge,
  Table,
  type TableColumn,
} from '../../design-system';
import { POLICY_RULES, POLICY_VERSIONS, type PolicyRule, type PolicyVersion } from './fixtures';

const VERSION_COLUMNS: readonly TableColumn<PolicyVersion>[] = [
  { key: 'version', header: 'Version', mono: true },
  {
    key: 'state',
    header: 'State',
    render: (row) => (
      <StatusBadge
        status={row.state === 'active' ? 'active' : row.state === 'draft' ? 'queued' : 'archived'}
        label={row.state === 'active' ? 'Active' : row.state === 'draft' ? 'Draft' : 'Superseded'}
        size="sm"
      />
    ),
  },
  { key: 'activated', header: 'Activated', mono: true },
];

const RULE_COLUMNS: readonly TableColumn<PolicyRule>[] = [
  { key: 'metric', header: 'Metric', mono: true },
  { key: 'operator', header: 'Operator', mono: true },
  { key: 'value', header: 'Threshold', mono: true, numeric: true },
  { key: 'window', header: 'Window', mono: true },
  { key: 'weight', header: 'Weight', numeric: true },
  {
    key: 'outcome',
    header: 'Drives',
    render: (row) => <StatusBadge status={row.outcome} size="sm" />,
  },
];

export function PoliciesScreen() {
  const [version, setVersion] = useState('v4');

  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Banner tone="info" title="Policy versions are immutable">
        Admins clone, validate, publish, activate and retire versions. Analysts select an existing
        version but cannot author one. Activating a new version never rewrites evaluations already
        attributed to an older one.
      </Banner>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'minmax(0,340px) minmax(0,1fr)',
          gap: 'var(--space-2)',
          alignItems: 'start',
        }}
      >
        <Panel
          title="Default adoption policy"
          meta="policy 732684512965427200"
          padding="0"
          actions={
            <Menu
              triggerLabel="Policy actions"
              items={[
                { label: 'Clone to draft', icon: 'copy' },
                { label: 'Validate draft', icon: 'circle-check' },
                { label: 'Activate draft', icon: 'play' },
                { separator: true },
                { label: 'Retire version', icon: 'archive' },
              ]}
            />
          }
        >
          <Table
            caption="Policy versions"
            getRowKey={(row) => row.version}
            selectedKey={version}
            onRowClick={(row) => setVersion(row.version)}
            columns={VERSION_COLUMNS}
            rows={POLICY_VERSIONS}
          />
        </Panel>
        <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
          <Panel
            title={`Rule tree · ${version}`}
            meta="catalogued metrics and explicit operators only"
            padding="0"
            footer={
              <span>
                An unknown metric is rejected with 422 unknown_metric. Arbitrary expressions and
                code are not accepted.
              </span>
            }
          >
            <Table
              caption={`Rules for version ${version}`}
              columns={RULE_COLUMNS}
              rows={POLICY_RULES}
              getRowKey={(row) => row.metric}
            />
          </Panel>
          <Panel title="Activation impact preview" meta="dry run against 5 projects · 90d">
            <div style={{ display: 'grid', gap: 'var(--space-15)' }}>
              <DefinitionList
                columns={4}
                dense
                items={[
                  { label: 'Recommended', value: '1 → 1' },
                  { label: 'Conditional', value: '2 → 2' },
                  { label: 'Not recommended', value: '1 → 1' },
                  { label: 'Insufficient data', value: '1 → 1' },
                ]}
              />
              <Recommendation
                result="conditional"
                policy="Default adoption policy"
                version={version}
                window="90d"
                cutoff="2026-08-20T14:35:00Z"
                conditions={[
                  'Temporal: review top-three contributor concentration before production adoption.',
                ]}
                decisive={[
                  {
                    metric: 'top_three_author_share',
                    rule: '<= 0.60',
                    value: '0.71',
                    pass: false,
                  },
                ]}
              />
            </div>
          </Panel>
        </div>
      </div>
    </div>
  );
}
