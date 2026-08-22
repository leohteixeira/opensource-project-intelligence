import { keepPreviousData, useMutation, useQuery } from '@tanstack/react-query';
import { useMemo, useState } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router';

import {
  Banner,
  Button,
  Checkbox,
  ComparisonMatrix,
  DateRangeField,
  EmptyState,
  HealthDimensions,
  MetricValue,
  Panel,
  Skeleton,
  StatusBadge,
  Table,
  TextField,
  type MatrixRow,
  type StatusKey,
} from '../../design-system';
import {
  createComparison,
  fetchComparison,
  fetchContributors,
  fetchHealth,
  fetchMetrics,
  fetchProjects,
  type ComparisonDocument,
  type IntelligenceStatus,
  type MetricDocument,
} from '../api';
import { useApplication } from '../router';
import { routePath } from '../routes';

const grid = { display: 'grid', gap: 'var(--space-2)' } as const;

const copy = {
  en: {
    health: 'Project health',
    healthBody:
      'Seven independent, equally weighted dimensions. Missing evidence is never redistributed.',
    contributors: 'Contributor intelligence',
    contributorBody:
      'Only verified and analyst-confirmed identities contribute to people-level concentration.',
    compare: 'Compare projects',
    compareBody:
      'Choose two to five distinct projects. Every cell uses one identical window, cutoff and definition version.',
    create: 'Create comparison',
    noData: 'No materialized intelligence',
    retry: 'Retry',
    select: 'Select projects',
    cutoff: 'Evidence cutoff',
    identity: 'Contributor identity',
    commits: 'Eligible commits',
    status: 'Resolution status',
    metric: 'Metric',
    project: 'Project',
    value: 'Value',
  },
  'pt-BR': {
    health: 'Saúde do projeto',
    healthBody:
      'Sete dimensões independentes com pesos iguais. Evidências ausentes nunca são redistribuídas.',
    contributors: 'Inteligência de colaboradores',
    contributorBody:
      'Somente identidades verificadas e confirmadas por analista entram na concentração por pessoa.',
    compare: 'Comparar projetos',
    compareBody:
      'Escolha de dois a cinco projetos distintos. Cada célula usa a mesma janela, corte e versão de definição.',
    create: 'Criar comparação',
    noData: 'Nenhuma inteligência materializada',
    retry: 'Tentar novamente',
    select: 'Selecionar projetos',
    cutoff: 'Corte das evidências',
    identity: 'Identidade do colaborador',
    commits: 'Commits elegíveis',
    status: 'Status de resolução',
    metric: 'Métrica',
    project: 'Projeto',
    value: 'Valor',
  },
} as const;

export function ProjectHealthScreen() {
  const { projectId = '' } = useParams();
  const { locale } = useApplication();
  const labels = copy[locale];
  const [params, setParams] = useSearchParams();
  const window = params.get('window') ?? '90d';
  const custom = customBounds(window);
  const cutoff = params.get('cutoff') ?? undefined;
  const metrics = useQuery({
    queryKey: ['metrics', projectId, window, cutoff],
    queryFn: () => fetchMetrics(projectId, window, cutoff),
    enabled: Boolean(projectId),
    placeholderData: keepPreviousData,
  });
  const health = useQuery({
    queryKey: ['health', projectId, window, cutoff],
    queryFn: () => fetchHealth(projectId, window, cutoff),
    enabled: Boolean(projectId),
    placeholderData: keepPreviousData,
  });
  const resolvedWindow = health.data?.window ?? metrics.data?.window;

  return (
    <section style={grid} data-testid="project-health">
      <header>
        <h1 style={{ font: 'var(--type-page)' }}>{labels.health}</h1>
        <p>{labels.healthBody}</p>
      </header>
      <DateRangeField
        value={custom ? 'custom' : window}
        from={custom?.from ?? resolvedWindow?.from}
        to={custom?.to ?? resolvedWindow?.to}
        cutoff={resolvedWindow?.cutoff ?? cutoff}
        onChange={(value) => {
          const next = new URLSearchParams(params);
          next.set('window', value === 'custom' ? defaultCustomWindow(resolvedWindow) : value);
          setParams(next);
        }}
        onCustomChange={(bound, value) => {
          const current = custom ?? customBounds(defaultCustomWindow(resolvedWindow));
          if (!current) return;
          const next = new URLSearchParams(params);
          next.set(
            'window',
            `${bound === 'from' ? value : current.from}/${bound === 'to' ? value : current.to}`,
          );
          setParams(next);
        }}
      />
      {health.isPending || metrics.isPending ? <Skeleton height={320} /> : null}
      {health.isError && !health.data ? (
        <Failure title={labels.noData} retry={() => void health.refetch()} label={labels.retry} />
      ) : null}
      {health.data ? (
        <Panel
          title={labels.health}
          meta={`${health.data.version} · cutoff ${health.data.window.cutoff}`}
        >
          <HealthDimensions
            layout="grid"
            dimensions={health.data.dimensions.map((dimension) => ({
              name: dimension.name,
              score: dimension.score,
              status: visibleStatus(dimension.status),
              rubric: `${Math.round(dimension.weight * 10000) / 100}% · ${dimension.version}`,
            }))}
            overall={{
              calculable: health.data.overall_status === 'available',
              score: health.data.overall,
              version: health.data.version,
              reason: 'one or more required dimensions are unavailable',
            }}
          />
        </Panel>
      ) : null}
      {metrics.isError && metrics.data ? (
        <Banner tone="attention" title="Showing the last completed metric snapshot">
          The refresh failed; the prior immutable snapshot remains visible.
        </Banner>
      ) : null}
      {metrics.data ? <MetricCatalog metrics={metrics.data.items} /> : null}
    </section>
  );
}

function MetricCatalog({ metrics }: { metrics: MetricDocument[] }) {
  return (
    <div
      style={{ ...grid, gridTemplateColumns: 'repeat(auto-fit,minmax(230px,1fr))' }}
      aria-label="Deterministic metric catalog"
    >
      {metrics.map((metric) => (
        <Panel
          key={metric.definition.name}
          title={humanize(metric.definition.name)}
          meta={`cutoff ${metric.window.cutoff}`}
        >
          <MetricValue
            name={metric.definition.name}
            label={humanize(metric.definition.name)}
            value={formatValue(metric.value)}
            unit={metric.definition.unit}
            status={visibleStatus(metric.status)}
            note={metric.stale_reason ?? metric.coverage.note}
            version={metric.definition.version}
            window={`${metric.window.from} – ${metric.window.to}`}
          />
          <p style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
            coverage {metric.coverage.observed}/{metric.coverage.eligible} (
            {Math.round(metric.coverage.ratio * 100)}%)
          </p>
        </Panel>
      ))}
    </div>
  );
}

export function ContributorIntelligenceScreen() {
  const { projectId = '' } = useParams();
  const { locale, narrow } = useApplication();
  const labels = copy[locale];
  const [params, setParams] = useSearchParams();
  const window = params.get('window') ?? '90d';
  const custom = customBounds(window);
  const contributors = useQuery({
    queryKey: ['contributors', projectId, window],
    queryFn: () => fetchContributors(projectId, window),
    enabled: Boolean(projectId),
    placeholderData: keepPreviousData,
  });

  return (
    <section style={grid} data-testid="contributor-intelligence">
      <header>
        <h1 style={{ font: 'var(--type-page)' }}>{labels.contributors}</h1>
        <p>{labels.contributorBody}</p>
      </header>
      <DateRangeField
        value={custom ? 'custom' : window}
        from={custom?.from ?? contributors.data?.window.from}
        to={custom?.to ?? contributors.data?.window.to}
        cutoff={contributors.data?.window.cutoff}
        coverage={
          contributors.data
            ? `${Math.round(contributors.data.summary.resolution_coverage * 100)}% resolved`
            : undefined
        }
        onChange={(value) => {
          const next = new URLSearchParams(params);
          next.set(
            'window',
            value === 'custom' ? defaultCustomWindow(contributors.data?.window) : value,
          );
          setParams(next);
        }}
        onCustomChange={(bound, value) => {
          const current = custom ?? customBounds(defaultCustomWindow(contributors.data?.window));
          if (!current) return;
          const next = new URLSearchParams(params);
          next.set(
            'window',
            `${bound === 'from' ? value : current.from}/${bound === 'to' ? value : current.to}`,
          );
          setParams(next);
        }}
      />
      {contributors.isPending ? <Skeleton height={320} /> : null}
      {contributors.isError && !contributors.data ? (
        <Failure
          title={labels.noData}
          retry={() => void contributors.refetch()}
          label={labels.retry}
        />
      ) : null}
      {contributors.data ? (
        <>
          <div style={{ ...grid, gridTemplateColumns: 'repeat(auto-fit,minmax(190px,1fr))' }}>
            <Panel title="Active contributors">
              <MetricValue
                label="Active"
                value={contributors.data.summary.active}
                unit="contributors"
                status={visibleStatus(contributors.data.summary.status)}
                version="contributors-v1"
                window={`${contributors.data.window.from} – ${contributors.data.window.to}`}
              />
            </Panel>
            <Panel title="Top contributor share">
              <MetricValue
                label="Top 1"
                value={percentage(contributors.data.summary.top_one_share)}
                unit="% commits"
                status={visibleStatus(contributors.data.summary.status)}
                version="concentration-v1"
                window={`cutoff ${contributors.data.window.cutoff}`}
              />
            </Panel>
            <Panel title="Top three share">
              <MetricValue
                label="Top 3"
                value={percentage(contributors.data.summary.top_three_share)}
                unit="% commits"
                status={visibleStatus(contributors.data.summary.status)}
                version="concentration-v1"
                window={`cutoff ${contributors.data.window.cutoff}`}
              />
            </Panel>
            <Panel title="Identity coverage">
              <MetricValue
                label="Resolved"
                value={percentage(contributors.data.summary.resolution_coverage)}
                unit="% eligible commits"
                version="identity-v1"
                window={`cutoff ${contributors.data.window.cutoff}`}
              />
            </Panel>
          </div>
          <Table
            caption="Eligible contributors and identity resolution provenance"
            rows={contributors.data.items}
            layout={narrow ? 'detail' : 'table'}
            getRowKey={(row) => row.key}
            columns={[
              { key: 'key', header: labels.identity, mono: true },
              { key: 'commits', header: labels.commits, numeric: true },
              {
                key: 'identity_status',
                header: labels.status,
                render: (row) => (
                  <StatusBadge
                    status={identityStatus(row.identity_status)}
                    label={row.identity_status}
                    size="sm"
                  />
                ),
              },
            ]}
          />
        </>
      ) : null}
    </section>
  );
}

export function ComparisonWorkspaceScreen() {
  const { comparisonId } = useParams();
  const { locale, session, narrow } = useApplication();
  const labels = copy[locale];
  const navigate = useNavigate();
  const [selected, setSelected] = useState<string[]>([]);
  const [window, setWindow] = useState('90d');
  const [customFrom, setCustomFrom] = useState('2026-05-24');
  const [customTo, setCustomTo] = useState('2026-08-22');
  const [cutoffDate, setCutoffDate] = useState(() => new Date().toISOString().slice(0, 10));
  const projects = useQuery({
    queryKey: ['comparison-projects'],
    queryFn: () => fetchProjects('active'),
    enabled: !comparisonId,
  });
  const comparison = useQuery({
    queryKey: ['comparison', comparisonId],
    queryFn: () => fetchComparison(comparisonId ?? ''),
    enabled: Boolean(comparisonId),
  });
  const mutation = useMutation({
    mutationFn: () =>
      createComparison(
        session,
        selected,
        window === 'custom' ? `${customFrom}/${customTo}` : window,
        `${cutoffDate}T00:00:00.000Z`,
      ),
    onSuccess: (result) => navigate(routePath(locale, 'comparison', { comparisonId: result.id })),
  });

  if (comparisonId) {
    if (comparison.isPending) return <Skeleton height={420} />;
    if (comparison.isError || !comparison.data)
      return (
        <Failure
          title={labels.noData}
          retry={() => void comparison.refetch()}
          label={labels.retry}
        />
      );
    return <ComparisonResult value={comparison.data} narrow={narrow} labels={labels} />;
  }

  return (
    <section style={grid} data-testid="comparison-builder">
      <header>
        <h1 style={{ font: 'var(--type-page)' }}>{labels.compare}</h1>
        <p>{labels.compareBody}</p>
      </header>
      <DateRangeField
        value={window}
        from={customFrom}
        to={customTo}
        cutoff={`${cutoffDate}T00:00:00.000Z`}
        onChange={setWindow}
        onCustomChange={(bound, value) =>
          bound === 'from' ? setCustomFrom(value) : setCustomTo(value)
        }
      />
      <TextField
        id="comparison-cutoff"
        type="date"
        label={labels.cutoff}
        value={cutoffDate}
        onChange={(event) => setCutoffDate(event.target.value)}
      />
      <Panel title={labels.select} meta={`${selected.length}/5`}>
        {projects.isPending ? <Skeleton height={180} /> : null}
        {projects.data?.items.map((project) => (
          <Checkbox
            key={project.id}
            id={`compare-${project.id}`}
            label={project.name}
            description={project.slug}
            checked={selected.includes(project.id)}
            disabled={!selected.includes(project.id) && selected.length >= 5}
            onChange={(event) =>
              setSelected((current) =>
                event.target.checked
                  ? [...current, project.id]
                  : current.filter((id) => id !== project.id),
              )
            }
          />
        ))}
      </Panel>
      {mutation.isError ? <Banner tone="critical" title={mutation.error.message} /> : null}
      <Button
        disabled={selected.length < 2 || selected.length > 5 || mutation.isPending}
        onClick={() => mutation.mutate()}
      >
        {labels.create}
      </Button>
    </section>
  );
}

function ComparisonResult({
  value,
  narrow,
  labels,
}: {
  value: ComparisonDocument;
  narrow: boolean;
  labels: (typeof copy)[keyof typeof copy];
}) {
  const rows = useMemo<MatrixRow[]>(
    () =>
      value.rows.map((row) => ({
        metric: humanize(row.metric),
        unit: row.unit,
        comparable: row.cells.every((cell) => cell.status === 'available'),
        cells: row.cells.map((cell) => ({
          value: cell.value,
          display: cell.value === undefined ? undefined : formatValue(cell.value),
          status: visibleStatus(cell.status),
          label: cell.status,
        })),
      })),
    [value.rows],
  );
  const detailRows = value.rows.flatMap((row) =>
    row.cells.map((cell) => ({
      id: `${row.metric}-${cell.project_id}`,
      metric: humanize(row.metric),
      project: cell.project_id,
      value: cell.status === 'available' ? `${formatValue(cell.value)} ${row.unit}` : cell.status,
    })),
  );

  return (
    <section style={grid} data-testid="comparison-result">
      <header>
        <h1 style={{ font: 'var(--type-page)' }}>{labels.compare}</h1>
        <p style={{ font: 'var(--type-mono-xs)' }}>
          {value.window.from} – {value.window.to} · cutoff {value.window.cutoff}
        </p>
      </header>
      {narrow ? (
        <Table
          caption="Comparison row details"
          layout="detail"
          rows={detailRows}
          columns={[
            { key: 'metric', header: labels.metric },
            { key: 'project', header: labels.project, mono: true },
            { key: 'value', header: labels.value, numeric: true },
          ]}
        />
      ) : (
        <ComparisonMatrix
          projects={value.project_ids}
          rows={rows}
          window={`${value.window.from} – ${value.window.to}`}
          cutoff={value.window.cutoff}
        />
      )}
    </section>
  );
}

function visibleStatus(status: IntelligenceStatus): StatusKey {
  if (status === 'incomparable' || status === 'unavailable') return 'unknown';
  return status;
}

function identityStatus(status: string): StatusKey {
  if (status === 'verified' || status === 'analyst_confirmed') return 'available';
  return 'unknown';
}

function humanize(value: string): string {
  return value.replaceAll('_', ' ').replace(/^./, (first) => first.toUpperCase());
}

function formatValue(value?: number): string {
  if (value === undefined) return '';
  return Number.isInteger(value) ? String(value) : value.toFixed(2);
}

function percentage(value?: number): string {
  return value === undefined ? '' : (value * 100).toFixed(1);
}

function customBounds(value: string): { from: string; to: string } | undefined {
  const [from, to, extra] = value.split('/');
  return from && to && !extra ? { from, to } : undefined;
}

function defaultCustomWindow(window?: { from: string; to: string }): string {
  const to = (window?.to ?? new Date().toISOString()).slice(0, 10);
  const fallbackFrom = new Date(`${to}T00:00:00.000Z`);
  fallbackFrom.setUTCDate(fallbackFrom.getUTCDate() - 90);
  const from = (window?.from ?? fallbackFrom.toISOString()).slice(0, 10);
  return `${from}/${to}`;
}

function Failure({ title, retry, label }: { title: string; retry: () => void; label: string }) {
  return (
    <EmptyState
      glyph="circle-dashed"
      title={title}
      action={<Button onClick={retry}>{label}</Button>}
    />
  );
}
