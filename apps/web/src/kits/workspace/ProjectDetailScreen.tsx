import { useState } from 'react';

import {
  Banner,
  Button,
  CoverageDisclosure,
  DefinitionList,
  Drawer,
  EmptyState,
  EvidenceLink,
  HealthDimensions,
  JobProgress,
  Menu,
  Panel,
  Recommendation,
  RunMetadata,
  StatusBadge,
  Table,
  Tabs,
  TrendChart,
  type TableColumn,
} from '../../design-system';
import {
  HEALTH,
  JOBS,
  METRICS,
  SOURCES,
  TREND_FORECAST,
  TREND_SERIES,
  type MetricRow,
  type SourceRow,
  type WorkspaceProject,
} from './fixtures';

const TABS = [
  { value: 'overview', label: 'Overview' },
  { value: 'health', label: 'Health' },
  { value: 'contributors', label: 'Contributors' },
  { value: 'adoption-security', label: 'Adoption & Security' },
  { value: 'trends', label: 'Trends' },
  { value: 'topics', label: 'Topics', count: 12 },
  { value: 'releases', label: 'Releases' },
  { value: 'knowledge', label: 'Knowledge' },
  { value: 'sources-jobs', label: 'Sources & Jobs', count: 8 },
];

const ELSEWHERE = ['overview', 'contributors', 'adoption-security', 'releases', 'knowledge'];

function MetricDrawer({
  metric,
  onClose,
}: {
  readonly metric: MetricRow;
  readonly onClose: () => void;
}) {
  return (
    <Drawer
      eyebrow={metric.name}
      title={metric.label}
      onClose={onClose}
      footer={
        <>
          <Button variant="secondary" iconStart="download">
            Export evidence
          </Button>
          <Button variant="secondary" iconStart="table-2">
            Compare projects
          </Button>
        </>
      }
    >
      <div
        style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-1)', flexWrap: 'wrap' }}
      >
        {metric.value ? (
          <>
            <span style={{ font: 'var(--type-metric-lg)', fontVariantNumeric: 'tabular-nums' }}>
              {metric.value}
            </span>
            <span style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
              {metric.unit}
            </span>
          </>
        ) : (
          <StatusBadge
            status={metric.status}
            detail={
              metric.status === 'insufficient_data'
                ? 'needs 90d, has 34d'
                : 'no NuGet package linked'
            }
          />
        )}
      </div>
      <DefinitionList
        columns={2}
        items={[
          {
            label: 'Formula',
            value: 'count(releases) where published_at in [from, to]',
            mono: true,
          },
          { label: 'Unit', value: metric.unit },
          { label: 'Definition version', value: `${metric.name} ${metric.version}`, mono: true },
          { label: 'Observation window', value: '2026-05-22 to 2026-08-19', mono: true },
          { label: 'Cutoff', value: '2026-08-20T14:35:00Z', mono: true },
          { label: 'Aggregation boundary', value: 'project · 3 repositories' },
          { label: 'Applicability', value: 'applies to core and sdk repositories' },
          {
            label: 'Missing-data treatment',
            value: 'excluded from the calculation, never counted as zero',
          },
        ]}
      />
      <CoverageDisclosure
        requested="90d"
        actual={metric.coverage}
        ratio={metric.coverage === '90d' ? 1 : 0.84}
        sources={[
          { name: 'github releases', value: '100%' },
          { name: 'changelog', value: '62%' },
        ]}
        missing={['rss']}
        cutoff="2026-08-20T14:35:00Z"
      />
      <div style={{ display: 'grid', gap: 'var(--space-075)' }}>
        <p style={{ font: 'var(--type-table-head)', color: 'var(--text-secondary)' }}>Evidence</p>
        <EvidenceLink
          kind="release"
          title="v1.24.0"
          source="github · temporalio/temporal"
          collectedAt="2026-08-18T09:02Z"
          href="#"
        />
        <EvidenceLink
          kind="release"
          title="v1.23.4"
          source="github · temporalio/temporal"
          collectedAt="2026-08-06T11:41Z"
          href="#"
        />
        <EvidenceLink
          kind="changelog"
          title="CHANGELOG.md — Unreleased section"
          source="github · temporalio/temporal"
          collectedAt="2026-08-20T02:00Z"
          href="#"
        />
      </div>
    </Drawer>
  );
}

function HealthTab({ onOpenMetric }: { readonly onOpenMetric: (metric: MetricRow) => void }) {
  const columns: readonly TableColumn<MetricRow>[] = [
    {
      key: 'label',
      header: 'Metric',
      wrap: true,
      render: (row) => (
        <span style={{ display: 'grid', gap: 1 }}>
          <span>{row.label}</span>
          <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>
            {row.name} · {row.version}
          </span>
        </span>
      ),
    },
    {
      key: 'value',
      header: 'Value',
      numeric: true,
      render: (row) =>
        row.value ? (
          <span>
            <span style={{ fontWeight: 'var(--weight-medium)' }}>{row.value}</span>{' '}
            <span style={{ color: 'var(--text-tertiary)' }}>{row.unit}</span>
          </span>
        ) : (
          <StatusBadge status={row.status} size="sm" />
        ),
    },
    { key: 'coverage', header: 'Coverage', numeric: true },
    {
      key: 'status',
      header: 'Status',
      render: (row) => <StatusBadge status={row.status} size="sm" />,
    },
  ];
  const firstMetric = METRICS[0];
  const concentrationMetric = METRICS[6];

  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'minmax(0,1fr) minmax(0,1fr)',
          gap: 'var(--space-2)',
          alignItems: 'start',
        }}
      >
        <Panel
          title="Health dimensions"
          meta="90d · cutoff 2026-08-20T14:35:00Z · rubrics v2–v3"
          footer={
            <CoverageDisclosure
              requested="90d"
              actual="90d"
              ratio={0.94}
              missing={['advisories']}
            />
          }
        >
          <HealthDimensions
            dimensions={HEALTH}
            overall={{ calculable: true, score: 61, version: 'overall v4' }}
            onOpen={firstMetric ? () => onOpenMetric(firstMetric) : undefined}
          />
        </Panel>
        <Panel
          title="Adoption recommendation"
          meta="evaluated 2026-08-20T14:35:00Z"
          actions={
            <Button size="sm" variant="ghost" iconEnd="arrow-right">
              Policy v4
            </Button>
          }
        >
          <Recommendation
            result="conditional"
            policy="Default adoption policy"
            version="v4"
            window="90d"
            cutoff="2026-08-20T14:35:00Z"
            conditions={[
              'Review top-three contributor concentration before production adoption.',
              'Confirm a maintainer rotation plan; maintainer count is 3.',
            ]}
            decisive={[
              { metric: 'release_frequency', rule: '>= 4 / 90d', value: '8', pass: true },
              {
                metric: 'median_time_to_first_response',
                rule: '<= 48 h',
                value: '9.4 h',
                pass: true,
              },
              { metric: 'top_three_author_share', rule: '<= 0.60', value: '0.71', pass: false },
            ]}
            missing={['nuget_download_change', 'regression_issue_share']}
            onEvidence={concentrationMetric ? () => onOpenMetric(concentrationMetric) : undefined}
          />
        </Panel>
      </div>
      <Panel
        title="Metric catalog results"
        meta="closed catalog · versioned definitions"
        padding="0"
      >
        <Table
          caption="Metric results for the 90 day window ending 2026-08-20"
          columns={columns}
          rows={METRICS}
          getRowKey={(row) => row.name}
          onRowClick={onOpenMetric}
          footer="Statuses are not values: unknown, not applicable and insufficient data are excluded from ranking and never rendered as 0."
        />
      </Panel>
    </div>
  );
}

const SOURCE_COLUMNS: readonly TableColumn<SourceRow>[] = [
  { key: 'kind', header: 'Kind', mono: true },
  { key: 'url', header: 'Source', mono: true, wrap: true },
  { key: 'role', header: 'Role', mono: true },
  {
    key: 'state',
    header: 'State',
    render: (row) => <StatusBadge status={row.state} size="sm" />,
  },
  { key: 'lastSuccess', header: 'Last success', mono: true },
  { key: 'next', header: 'Next run', mono: true },
  { key: 'coverage', header: 'Coverage', mono: true },
];

function SourcesJobsTab() {
  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Banner
        tone="info"
        title="Collection uses public data only"
        actions={
          <Button size="sm" variant="secondary" iconStart="refresh-cw">
            Request sync
          </Button>
        }
      >
        Operator-managed read credentials authenticate requests for usable quotas. They never
        authorise private content, and no credential is ever returned to this surface.
      </Banner>
      <Panel
        title="Sources and coverage"
        meta="8 sources · initial history target 180d"
        padding="0"
      >
        <Table
          caption="Source coverage and freshness"
          columns={SOURCE_COLUMNS}
          rows={SOURCES}
          getRowKey={(row) => `${row.kind}:${row.url}`}
        />
      </Panel>
      <Panel title="Jobs">
        <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
          {JOBS.map((job) => (
            <div
              key={job.id}
              style={{
                paddingBottom: 'var(--space-15)',
                borderBottom: '1px solid var(--border-table)',
              }}
            >
              <JobProgress
                {...job}
                actions={
                  job.state === 'running' ? (
                    <Button size="sm" variant="ghost">
                      Cancel
                    </Button>
                  ) : job.state === 'failed' ? (
                    <Button size="sm" variant="secondary" iconStart="rotate-ccw">
                      Retry
                    </Button>
                  ) : null
                }
              />
            </div>
          ))}
        </div>
      </Panel>
    </div>
  );
}

function TrendsTab() {
  const [kind, setKind] = useState('observed');

  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Tabs
        value={kind}
        onChange={setKind}
        size="sm"
        items={[
          { value: 'observed', label: 'Observed' },
          { value: 'forecast', label: 'Early warnings' },
        ]}
      />
      {kind === 'observed' ? (
        <Panel
          title="Active contributors"
          status={<StatusBadge status="observed" label="Observed decrease" size="sm" />}
          meta="observation 2026-05-22 to 2026-08-19 · baseline 2026-02-21 to 2026-05-21 · method v2"
        >
          <div style={{ display: 'grid', gap: 'var(--space-15)' }}>
            <TrendChart
              label="Active contributors"
              unit="people"
              width={520}
              height={96}
              series={TREND_SERIES}
              showForecast={false}
            />
            <DefinitionList
              columns={3}
              dense
              items={[
                { label: 'Change', value: '−38.6%' },
                { label: 'Method', value: 'Theil-Sen slope, v2', mono: true },
                { label: 'Coverage', value: '90d of 90d' },
              ]}
            />
          </div>
        </Panel>
      ) : (
        <Panel
          title="Maintainer count"
          status={<StatusBadge status="forecast" label="Early warning" size="sm" />}
          meta="model maintainer_count v2 · generated 2026-08-20T06:00Z"
          tone="attention"
        >
          <div style={{ display: 'grid', gap: 'var(--space-15)' }}>
            <Banner tone="attention" title="This is a forecast, not an observation">
              The dashed segment is predicted. Forecasts do not change health dimensions, policy
              results or radar placement; they raise an early warning only.
            </Banner>
            <TrendChart
              label="Active contributors"
              unit="people"
              width={520}
              height={96}
              series={TREND_SERIES}
              forecast={TREND_FORECAST}
            />
            <DefinitionList
              columns={4}
              dense
              items={[
                { label: 'Horizon', value: '120 days' },
                { label: 'Confidence', value: '0.62' },
                { label: 'Known error', value: 'MAPE 0.19 (last 6 folds)' },
                { label: 'Minimum history', value: '180d (have 214d)' },
              ]}
            />
            <EvidenceLink
              kind="run"
              title="Forecast run v2 · maintainer_count"
              source="local-ollama · deterministic model, no LLM"
              collectedAt="2026-08-20T06:00Z"
              href="#"
              external={false}
            />
          </div>
        </Panel>
      )}
    </div>
  );
}

export function ProjectDetailScreen({
  project,
  tab,
  onTab,
  onOpenKit,
}: {
  readonly project: WorkspaceProject;
  readonly tab: string;
  readonly onTab: (tab: string) => void;
  readonly onOpenKit?: (kit: string) => void;
}) {
  const [metric, setMetric] = useState<MetricRow | null>(null);

  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <div
        style={{
          display: 'flex',
          gap: 'var(--space-15)',
          alignItems: 'flex-start',
          flexWrap: 'wrap',
        }}
      >
        <div style={{ display: 'grid', gap: 'var(--space-05)', flex: 1, minWidth: 240 }}>
          <div
            style={{
              display: 'flex',
              gap: 'var(--space-1)',
              alignItems: 'center',
              flexWrap: 'wrap',
            }}
          >
            <StatusBadge status="active" size="sm" />
            <StatusBadge status="conditional" size="sm" />
            <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-secondary)' }}>
              {project.id} · {project.repos} repositories · {project.sources} sources
            </span>
          </div>
          <p style={{ font: 'var(--type-body)', color: 'var(--text-secondary)', maxWidth: '68ch' }}>
            {project.description}
          </p>
        </div>
        <div style={{ display: 'flex', gap: 'var(--space-1)' }}>
          <Button variant="secondary" size="md" iconStart="refresh-cw">
            Request sync
          </Button>
          <Menu
            triggerLabel="Project actions"
            items={[
              { label: 'Edit identity', icon: 'pencil' },
              { label: 'Manage repositories', icon: 'git-branch' },
              { label: 'Request longer history', icon: 'history' },
              { separator: true },
              { label: 'Pause collection', icon: 'pause' },
              { label: 'Archive', icon: 'archive' },
              { label: 'Delete permanently', icon: 'trash-2', danger: true },
            ]}
          />
        </div>
      </div>
      <Tabs value={tab} onChange={onTab} items={TABS} />
      {tab === 'health' ? <HealthTab onOpenMetric={setMetric} /> : null}
      {tab === 'sources-jobs' ? <SourcesJobsTab /> : null}
      {tab === 'trends' ? <TrendsTab /> : null}
      {tab === 'topics' ? (
        <Panel
          title="Issue and discussion topics"
          meta="known taxonomy v5 · emerging topics from 90d"
          footer={
            <span>
              Corrections remain attributed inputs to later reprocessing, not silent edits.
            </span>
          }
        >
          <div style={{ display: 'grid', gap: 'var(--space-15)' }}>
            <RunMetadata
              runId="732684513258979328"
              versionLabel="v3 of 3"
              selected
              provider="local-ollama"
              model="qwen2.5:32b"
              promptVersion="topics@7"
              language="en"
              executedAt="2026-08-20T09:12:44Z"
              usage="3.1k tokens"
              actions={
                <Button size="sm" variant="secondary" iconStart="rotate-ccw">
                  Rerun
                </Button>
              }
            />
            <EvidenceLink
              kind="issue"
              title="Emerging: worker task starvation under high concurrency — 24 issues, prevalence 6.1%, confidence 0.71"
              source="github issues · 90d"
              collectedAt="2026-08-20T09:12Z"
              href="#"
            />
            <EvidenceLink
              kind="discussion"
              title="Known: upgrade path clarity — 11 discussions, prevalence 2.8%, confidence 0.83"
              source="github discussions · 90d"
              collectedAt="2026-08-20T09:12Z"
              href="#"
            />
          </div>
        </Panel>
      ) : null}
      {ELSEWHERE.includes(tab) ? (
        <EmptyState
          glyph="arrow-right"
          title={`The ${tab.replace('-', ' and ')} surface is built in the project-evidence shell`}
          action={
            <Button variant="secondary" onClick={() => onOpenKit?.('project-evidence')}>
              Open project evidence
            </Button>
          }
        >
          This shell covers Health, Trends, Topics and Sources &amp; Jobs. Overview, Contributors,
          Adoption &amp; Security, Releases and Knowledge are built in the project-evidence shell,
          against the same components and the same fixture shapes.
        </EmptyState>
      ) : null}
      {metric ? <MetricDrawer metric={metric} onClose={() => setMetric(null)} /> : null}
    </div>
  );
}
