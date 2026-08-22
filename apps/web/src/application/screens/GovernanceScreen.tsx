import { useMutation, useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router';

import {
  Banner,
  Button,
  EmptyState,
  Panel,
  RadarList,
  Recommendation,
  Select,
  StatusBadge,
  Table,
  Tabs,
  TextArea,
  TextField,
  TrendChart,
  type RadarEntry,
  type RecommendationResult,
  type StatusKey,
  type TableColumn,
} from '../../design-system';
import {
  fetchAlerts,
  createPolicy,
  fetchPolicies,
  fetchRadar,
  fetchRecommendation,
  fetchTrends,
  markAlertRead,
  overrideRadar,
  transitionAlert,
  type Document,
} from '../api';
import { useApplication } from '../router';

const layout = { display: 'grid', gap: 'var(--space-2)' } as const;
const string = (value: unknown, fallback = '') =>
  typeof value === 'string' || typeof value === 'number' ? String(value) : fallback;
const array = (value: unknown): Document[] => (Array.isArray(value) ? (value as Document[]) : []);
const strings = (value: unknown): string[] =>
  Array.isArray(value) ? value.map((item) => string(item)).filter(Boolean) : [];

const policyStatus = (state: unknown): StatusKey => {
  switch (string(state)) {
    case 'active':
      return 'active';
    case 'superseded':
      return 'stale';
    case 'retired':
      return 'archived';
    default:
      return 'queued';
  }
};

const alertStatus = (state: unknown): StatusKey => {
  switch (string(state)) {
    case 'acknowledged':
      return 'observed';
    case 'resolved':
      return 'succeeded';
    case 'dismissed':
      return 'archived';
    default:
      return 'conditional';
  }
};

function RequestState({ pending, failed }: { pending: boolean; failed: boolean }) {
  if (pending) return <p role="status">Loading immutable intelligence…</p>;
  if (failed)
    return (
      <Banner tone="critical" title="Intelligence unavailable">
        Deterministic results remain unchanged. Retry the same policy, window, and cutoff.
      </Banner>
    );
  return null;
}

export function TrendRecommendationScreen() {
  const { t } = useTranslation();
  const { projectId = '' } = useParams();
  const [kind, setKind] = useState<'observed' | 'forecast'>('observed');
  const trends = useQuery({
    queryKey: ['trends', projectId, kind],
    queryFn: () => fetchTrends(projectId, kind),
    enabled: Boolean(projectId),
  });
  const recommendation = useQuery({
    queryKey: ['recommendation', projectId],
    queryFn: () => fetchRecommendation(projectId),
    enabled: Boolean(projectId),
  });
  const items = trends.data?.items ?? [];
  const evaluation = recommendation.data;
  const factors = array(evaluation?.factors);

  return (
    <section style={layout} data-testid="trend-recommendation-screen">
      <header>
        <h1>{t('trendsTitle')}</h1>
        <p>
          Reproducible observations and predictive early warnings remain separate claims, each with
          its own method, window, evidence, coverage, and version.
        </p>
      </header>
      <Tabs
        value={kind}
        onChange={(value) => setKind(value as 'observed' | 'forecast')}
        items={[
          { value: 'observed', label: 'Observed trends' },
          { value: 'forecast', label: 'Forecast warnings' },
        ]}
      />
      <RequestState pending={trends.isPending} failed={trends.isError} />
      <Panel
        title={kind === 'observed' ? 'Observed trends' : 'Forecast warnings'}
        meta={
          kind === 'observed'
            ? 'Theil–Sen · Mann–Kendall · deterministic'
            : 'rolling backtest · predictive'
        }
      >
        {!trends.isPending && items.length === 0 ? (
          <EmptyState glyph="chart-line" title="Insufficient trend evidence">
            No direction is inferred from sparse data. The result is not a neutral zero.
          </EmptyState>
        ) : (
          <div className="opi-intelligence-grid">
            {items.map((item, index) => {
              const coverage = item.coverage as Document | undefined;
              const value = Number(item.magnitude ?? item.predicted ?? 0);
              const status = string(item.status, 'insufficient_data');
              return (
                <article className="opi-intelligence-card" key={string(item.id, String(index))}>
                  <div className="opi-action-row">
                    <strong>{string(item.metric_name, 'Metric signal')}</strong>
                    <StatusBadge
                      status={
                        (status === 'increase' || status === 'decrease'
                          ? 'observed'
                          : status) as StatusKey
                      }
                      label={status.replaceAll('_', ' ')}
                      size="sm"
                    />
                  </div>
                  {Number.isFinite(value) && status !== 'insufficient_data' ? (
                    <TrendChart
                      label={string(item.metric_name)}
                      series={[
                        { label: 'window start', value: 0 },
                        { label: 'cutoff', value },
                      ]}
                      forecast={kind === 'forecast' ? [{ label: 'horizon', value }] : []}
                    />
                  ) : null}
                  <dl className="opi-compact-definition-list">
                    <div>
                      <dt>Kind</dt>
                      <dd>{kind}</dd>
                    </div>
                    <div>
                      <dt>Method/model</dt>
                      <dd>
                        {string(item.method_version)}
                        {item.selected_model ? ` · ${string(item.selected_model)}` : ''}
                      </dd>
                    </div>
                    <div>
                      <dt>Observation window</dt>
                      <dd>
                        {string(item.window_from)} – {string(item.window_to)}
                      </dd>
                    </div>
                    {kind === 'observed' ? (
                      <div>
                        <dt>Baseline window</dt>
                        <dd>
                          {string(item.baseline_from)} – {string(item.baseline_to)}
                        </dd>
                      </div>
                    ) : null}
                    <div>
                      <dt>{kind === 'observed' ? 'Magnitude' : 'Predicted value'}</dt>
                      <dd>{string(item.magnitude ?? item.predicted, 'unavailable')}</dd>
                    </div>
                    <div>
                      <dt>Coverage</dt>
                      <dd>
                        {string(coverage?.observed, '0')} / {string(coverage?.eligible, '0')}
                      </dd>
                    </div>
                    <div>
                      <dt>Evidence</dt>
                      <dd>{strings(item.evidence_ids).length} immutable records</dd>
                    </div>
                    {kind === 'forecast' ? (
                      <>
                        <div>
                          <dt>Horizon</dt>
                          <dd>{string(item.horizon_days)} days</dd>
                        </div>
                        <div>
                          <dt>Confidence interval</dt>
                          <dd>
                            {string(item.interval_low)} – {string(item.interval_high)} · confidence{' '}
                            {string(item.confidence)}
                          </dd>
                        </div>
                        <div>
                          <dt>Calibration error</dt>
                          <dd>{string(item.backtest_error, 'unavailable')}</dd>
                        </div>
                      </>
                    ) : null}
                  </dl>
                </article>
              );
            })}
          </div>
        )}
      </Panel>
      <RequestState pending={recommendation.isPending} failed={recommendation.isError} />
      {evaluation ? (
        <Panel title="Adoption recommendation" meta={`evaluation ${string(evaluation.id)}`}>
          <Recommendation
            result={string(evaluation.outcome, 'insufficient_data') as RecommendationResult}
            policy={`Policy ${string(evaluation.policy_id)} · owner ${string(evaluation.policy_owner, 'unassigned')}`}
            version={`v${string(evaluation.policy_version)}`}
            window="90d"
            cutoff={string((evaluation.window as Document | undefined)?.cutoff)}
            conditions={strings(evaluation.conditions)}
            missing={strings(evaluation.missing_data)}
            stale={strings(evaluation.stale_inputs).join(', ') || undefined}
            decisive={factors.map((factor) => ({
              metric: string(factor.metric_name),
              rule: `${string(factor.label)} · threshold ${string(factor.threshold)} · weight ${string(factor.weight)}`,
              value: string(factor.value),
              pass: Boolean(factor.matched),
            }))}
          />
          <p className="opi-intelligence-meta">
            The four-state result is deterministic. Generated explanation availability cannot change
            it. {strings(evaluation.evidence_ids).length} immutable evidence records support this
            evaluation.
          </p>
        </Panel>
      ) : null}
    </section>
  );
}

export function PoliciesGovernanceScreen() {
  const { t } = useTranslation();
  const { session } = useApplication();
  const [state, setState] = useState('');
  const [draftOpen, setDraftOpen] = useState(false);
  const [draftName, setDraftName] = useState('');
  const [draftOwner, setDraftOwner] = useState('');
  const [draftRules, setDraftRules] = useState('[]');
  const [draftMapping, setDraftMapping] = useState('{}');
  const [draftError, setDraftError] = useState('');
  const policies = useQuery({
    queryKey: ['policies', state],
    queryFn: () => fetchPolicies(state || undefined),
  });
  const items = policies.data?.items ?? [];
  const create = useMutation({
    mutationFn: async () => {
      const rules = JSON.parse(draftRules) as Document[];
      const radarMapping = JSON.parse(draftMapping) as Document;
      return createPolicy(session, {
        name: draftName,
        description: 'Cloned and reviewed through policy governance',
        owner: draftOwner,
        rules,
        radar_mapping: radarMapping,
      });
    },
    onSuccess: () => {
      setDraftOpen(false);
      setDraftError('');
      void policies.refetch();
    },
    onError: (error) =>
      setDraftError(error instanceof Error ? error.message : 'Draft validation failed'),
  });
  const openDraft = () => {
    const source = items[0];
    setDraftName(source ? `${string(source.name)} copy` : 'Production adoption');
    setDraftOwner(source ? string(source.owner) : 'Architecture');
    setDraftRules(JSON.stringify(source?.rules ?? [], null, 2));
    setDraftMapping(JSON.stringify(source?.radar_mapping ?? {}, null, 2));
    setDraftOpen(true);
  };
  const columns: readonly TableColumn<Document>[] = [
    { key: 'name', header: 'Policy', render: (row) => string(row.name) },
    { key: 'version', header: 'Version', mono: true, render: (row) => `v${string(row.version)}` },
    { key: 'owner', header: 'Owner', render: (row) => string(row.owner) },
    {
      key: 'state',
      header: 'State',
      render: (row) => (
        <StatusBadge
          status={policyStatus(row.state)}
          label={string(row.state).replaceAll('_', ' ')}
          size="sm"
        />
      ),
    },
    { key: 'rules', header: 'Rules', numeric: true, render: (row) => array(row.rules).length },
    {
      key: 'rule_tree',
      header: 'Rule tree',
      render: (row) =>
        array(row.rules)
          .map(
            (rule) =>
              `${string(rule.label)}: ${string(rule.metric_name)} ${string(rule.operator)} ${string(rule.threshold)} · weight ${string(rule.weight)}`,
          )
          .join('; '),
    },
    {
      key: 'radar_mapping',
      header: 'Radar mapping',
      render: (row) =>
        Object.entries((row.radar_mapping as Document | undefined) ?? {})
          .map(([outcome, ring]) => `${outcome} → ${string(ring)}`)
          .join('; '),
    },
  ];
  return (
    <section style={layout} data-testid="policies-screen">
      <header>
        <h1>{t('policiesTitle')}</h1>
        <p>Validate typed rules, publish immutable versions, and preserve historical decisions.</p>
      </header>
      <Banner tone="info" title="Published policy versions are immutable">
        Typed catalog rules validate before activation. Historical recommendations retain their
        exact policy and metric versions.
      </Banner>
      <div className="opi-action-row">
        <Select
          id="policy-state"
          label="State"
          value={state}
          placeholder="All states"
          options={['draft', 'active', 'superseded', 'retired']}
          onChange={(event) => setState(event.target.value)}
        />
        {session.role === 'admin' ? <Button onClick={openDraft}>New policy draft</Button> : null}
      </div>
      {draftOpen ? (
        <Panel title="Editable policy draft" meta="server validation required before publication">
          <div style={layout}>
            <TextField
              id="policy-name"
              label="Policy name"
              value={draftName}
              required
              onChange={(event) => setDraftName(event.target.value)}
            />
            <TextField
              id="policy-owner"
              label="Policy owner"
              value={draftOwner}
              required
              onChange={(event) => setDraftOwner(event.target.value)}
            />
            <TextArea
              id="policy-rules"
              label="Typed rule tree (JSON)"
              value={draftRules}
              required
              rows={10}
              onChange={(event) => setDraftRules(event.target.value)}
            />
            <TextArea
              id="policy-radar-mapping"
              label="Four-state radar mapping (JSON)"
              value={draftMapping}
              required
              rows={6}
              onChange={(event) => setDraftMapping(event.target.value)}
            />
            {draftError ? (
              <Banner tone="critical" title="Draft validation failed">
                {draftError}
              </Banner>
            ) : null}
            <div className="opi-action-row">
              <Button onClick={() => create.mutate()} pending={create.isPending}>
                Create immutable draft v1
              </Button>
              <Button variant="ghost" onClick={() => setDraftOpen(false)}>
                Cancel
              </Button>
            </div>
          </div>
        </Panel>
      ) : null}
      <RequestState pending={policies.isPending} failed={policies.isError} />
      <Panel title="Policy families" meta="catalogued metrics and explicit operators" padding="0">
        {items.length === 0 && !policies.isPending ? (
          <EmptyState glyph="file-text" title="No policy versions">
            Create and validate a draft before activation.
          </EmptyState>
        ) : (
          <Table
            caption="Immutable policy versions"
            columns={columns}
            rows={items}
            getRowKey={(row) => string(row.id)}
          />
        )}
      </Panel>
    </section>
  );
}

export function RadarGovernanceScreen() {
  const { t } = useTranslation();
  const { session } = useApplication();
  const radar = useQuery({ queryKey: ['radar'], queryFn: () => fetchRadar() });
  const [selected, setSelected] = useState<Document>();
  const [ring, setRing] = useState('assess');
  const [reason, setReason] = useState('');
  const [owner, setOwner] = useState('');
  const [reviewOn, setReviewOn] = useState('');
  const source = array(radar.data?.items);
  const canOverride = session.role === 'analyst' || session.role === 'admin';
  const override = useMutation({
    mutationFn: () =>
      overrideRadar(session, string(selected?.project_id), ring, reason, owner, reviewOn),
    onSuccess: () => {
      setSelected(undefined);
      void radar.refetch();
    },
  });
  const entries: RadarEntry[] = source.map((item) => {
    const override = item.override as Document | undefined;
    return {
      project: `Project ${string(item.project_id)}`,
      policyRing: string(item.suggested_ring, 'unplaced') as RadarEntry['policyRing'],
      effectiveRing: string(item.effective_ring, 'unplaced') as RadarEntry['effectiveRing'],
      override: override
        ? { owner: string(override.owner), expired: Boolean(item.override_expired) }
        : undefined,
      reviewDue: override ? string(override.review_on) : undefined,
      reviewOverdue: Boolean(item.review_overdue),
    };
  });
  return (
    <section style={layout} data-testid="radar-screen">
      <header>
        <h1>{t('radarTitle')}</h1>
        <p>Review policy suggestions and attributed organizational overrides.</p>
      </header>
      <Banner tone="info" title="Suggestion and placement are separate facts">
        Every ring comes from an exact policy version. An attributed override changes only the
        effective placement and expires on its review lifecycle.
      </Banner>
      <RequestState pending={radar.isPending} failed={radar.isError} />
      <Panel
        title="Effective placements"
        meta={`${string(radar.data?.count, '0')} selected projects`}
      >
        {entries.length ? (
          <RadarList
            entries={entries}
            onSelect={
              canOverride
                ? (entry) => {
                    const item = source.find(
                      (candidate) => `Project ${string(candidate.project_id)}` === entry.project,
                    );
                    setSelected(item);
                    setOwner(
                      string((item?.override as Document | undefined)?.owner, 'Architecture'),
                    );
                  }
                : undefined
            }
          />
        ) : (
          <EmptyState glyph="radar" title="No projects selected">
            The unplaced state remains explicit until an Analyst selects a recommendation.
          </EmptyState>
        )}
      </Panel>
      {selected && canOverride ? (
        <Panel
          title={`Override Project ${string(selected.project_id)}`}
          meta="The policy suggestion remains immutable"
        >
          <div style={layout}>
            <Select
              id="override-ring"
              label="Effective ring"
              value={ring}
              options={['adopt', 'trial', 'assess', 'hold']}
              onChange={(event) => setRing(event.target.value)}
            />
            <TextArea
              id="override-reason"
              label="Justification"
              value={reason}
              required
              onChange={(event) => setReason(event.target.value)}
            />
            <TextField
              id="override-owner"
              label="Owner"
              value={owner}
              required
              onChange={(event) => setOwner(event.target.value)}
            />
            <TextField
              id="override-review"
              type="date"
              label="Review date"
              value={reviewOn}
              required
              onChange={(event) => setReviewOn(event.target.value)}
            />
            {override.isError ? (
              <Banner tone="critical" title="Override conflict">
                Refresh the current placement before applying another change.
              </Banner>
            ) : null}
            <div className="opi-action-row">
              <Button onClick={() => override.mutate()} pending={override.isPending}>
                Save attributed override
              </Button>
              <Button variant="ghost" onClick={() => setSelected(undefined)}>
                Cancel
              </Button>
            </div>
          </div>
        </Panel>
      ) : null}
    </section>
  );
}

export function AlertsGovernanceScreen() {
  const { t } = useTranslation();
  const { session } = useApplication();
  const [state, setState] = useState('');
  const alerts = useQuery({
    queryKey: ['alerts', state],
    queryFn: () => fetchAlerts(state || undefined),
  });
  const read = useMutation({
    mutationFn: (id: string) => markAlertRead(session, id),
    onSuccess: () => alerts.refetch(),
  });
  const transition = useMutation({
    mutationFn: (item: Document) =>
      transitionAlert(
        session,
        string(item.id),
        Number(item.revision),
        'acknowledged',
        'Analyst review',
      ),
    onSuccess: () => alerts.refetch(),
  });
  const items = alerts.data?.items ?? [];
  const canTransition = session.role === 'analyst' || session.role === 'admin';
  const columns: readonly TableColumn<Document>[] = [
    {
      key: 'read_at',
      header: 'Read',
      render: (row) => (row.read_at ? 'Read' : <strong>Unread</strong>),
    },
    {
      key: 'severity',
      header: 'Severity',
      render: (row) => (
        <StatusBadge
          status={string(row.severity) === 'critical' ? 'failed' : 'conditional'}
          label={string(row.severity)}
          size="sm"
        />
      ),
    },
    { key: 'project_id', header: 'Project', mono: true, render: (row) => string(row.project_id) },
    {
      key: 'state',
      header: 'Shared state',
      render: (row) => (
        <StatusBadge
          status={alertStatus(row.state)}
          label={string(row.state).replaceAll('_', ' ')}
          size="sm"
        />
      ),
    },
    {
      key: 'suppression_count',
      header: 'Deduplicated',
      numeric: true,
      render: (row) => string(row.suppression_count, '0'),
    },
    {
      key: 'rule_version',
      header: 'Rule / signal',
      render: (row) => `v${string(row.rule_version)} · ${string(row.signal_version)}`,
    },
    {
      key: 'window',
      header: 'Window / detected',
      render: (row) =>
        `${string(row.window_from)} – ${string(row.window_to)} · ${string(row.last_detected_at)}`,
    },
    {
      key: 'evidence_ids',
      header: 'Evidence',
      render: (row) => `${strings(row.evidence_ids).length} immutable records`,
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (row) => (
        <span className="opi-action-row">
          <Button size="sm" variant="ghost" onClick={() => read.mutate(string(row.id))}>
            Mark read
          </Button>
          {canTransition && row.state === 'open' ? (
            <Button size="sm" variant="secondary" onClick={() => transition.mutate(row)}>
              Acknowledge
            </Button>
          ) : null}
        </span>
      ),
    },
  ];
  return (
    <section style={layout} data-testid="alerts-screen">
      <header>
        <h1>{t('alertsTitle')}</h1>
        <p>Monitor evidence-backed occurrences without conflating personal and shared state.</p>
      </header>
      <Banner tone="info" title="Shared lifecycle, personal read state">
        Redelivery updates one deduplicated occurrence. Reading it is personal; acknowledging,
        resolving, dismissing, and reopening are attributed team transitions.
      </Banner>
      <Select
        id="alert-state"
        label="Shared state"
        value={state}
        placeholder="All states"
        options={['open', 'acknowledged', 'resolved', 'dismissed']}
        onChange={(event) => setState(event.target.value)}
      />
      <RequestState
        pending={alerts.isPending}
        failed={alerts.isError || read.isError || transition.isError}
      />
      <Panel
        title="Alert inbox"
        meta="versioned rules · evidence-backed · deduplicated"
        padding="0"
      >
        {items.length === 0 && !alerts.isPending ? (
          <EmptyState glyph="bell" title="No alert occurrences">
            No qualifying evidence has triggered a rule.
          </EmptyState>
        ) : (
          <Table
            caption="Shared alert occurrences and personal read state"
            columns={columns}
            rows={items}
            getRowKey={(row) => string(row.id)}
          />
        )}
      </Panel>
    </section>
  );
}
