import {
  Banner,
  Button,
  DefinitionList,
  MetricValue,
  Panel,
  Progress,
  StatusBadge,
  Table,
  type TableColumn,
} from '../../design-system';
import { PROVIDERS, type Provider } from './fixtures';

const COLUMNS: readonly TableColumn<Provider>[] = [
  { key: 'name', header: 'Provider', mono: true },
  { key: 'kind', header: 'Kind', mono: true },
  {
    key: 'state',
    header: 'Health',
    render: (row) => <StatusBadge status={row.state} size="sm" />,
  },
  { key: 'detail', header: 'Detail', wrap: true },
  { key: 'quota', header: 'Quota', mono: true },
  { key: 'credential', header: 'Credential', mono: true },
];

export function OperationsScreen() {
  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Banner
        tone="attention"
        title="One model provider is degraded"
        actions={
          <Button size="sm" variant="secondary" iconStart="rotate-ccw">
            Re-check providers
          </Button>
        }
      >
        AI-dependent analysis is queued or served from the last successful run. Collection,
        deterministic metrics, health, policies, radar, comparison and observed trends are
        unaffected.
      </Banner>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))',
          gap: 'var(--space-2)',
        }}
      >
        <Panel title="Deterministic pipeline" status={<StatusBadge status="ready" size="sm" />}>
          <div style={{ display: 'flex', gap: 'var(--space-3)', flexWrap: 'wrap' }}>
            <MetricValue label="Jobs running" value="2" unit="jobs" />
            <MetricValue label="Queue depth" value="7" unit="jobs" />
            <MetricValue label="Snapshots today" value="384" unit="rows" />
          </div>
        </Panel>
        <Panel
          title="AI analysis"
          status={<StatusBadge status="stale" detail="1 provider degraded" size="sm" />}
        >
          <div style={{ display: 'grid', gap: 'var(--space-1)' }}>
            <Progress label="Monthly model budget" completed={82} total={100} unit="%" />
            <DefinitionList
              dense
              columns={1}
              items={[
                { label: 'Aggregate usage', value: '1.24M tokens · 312 runs · 30d', mono: true },
              ]}
            />
          </div>
        </Panel>
        <Panel title="Storage and streams" status={<StatusBadge status="ready" size="sm" />}>
          <DefinitionList
            dense
            columns={1}
            items={[
              { label: 'PostgreSQL', value: 'ready · 41 GB', mono: true },
              { label: 'S3 object storage', value: 'ready · 128 GB', mono: true },
              { label: 'NATS JetStream', value: 'ready · 0 stalled consumers', mono: true },
              { label: 'Valkey', value: 'ready', mono: true },
            ]}
          />
        </Panel>
      </div>
      <Panel
        title="Providers and quotas"
        meta="credentials are configured by the VPS Operator outside this surface"
        padding="0"
        footer={
          <span>
            Redacted by design: no credential value, token, header or secret is ever included in a
            product response.
          </span>
        }
      >
        <Table
          caption="Provider capability, quota and health"
          columns={COLUMNS}
          rows={PROVIDERS}
          getRowKey={(row) => row.name}
        />
      </Panel>
    </div>
  );
}
