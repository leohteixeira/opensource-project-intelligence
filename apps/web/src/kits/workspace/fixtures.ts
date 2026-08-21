/**
 * Fixture data for the workspace. Shapes follow the HTTP contract: Snowflake identifiers are
 * decimal strings, statuses are never numeric zero, and every result carries a window and a cutoff.
 */
import type { HealthDimension, RadarEntry, SeriesPoint, StatusKey } from '../../design-system';

export interface WorkspaceProject {
  readonly id: string;
  readonly name: string;
  readonly slug: string;
  readonly description: string;
  readonly state: 'active' | 'paused';
  readonly repos: number;
  readonly sources: number;
  readonly recommendation: 'recommended' | 'conditional' | 'not_recommended' | 'insufficient_data';
  readonly overall: number | null;
  readonly freshness: 'fresh' | 'stale' | 'partial';
  readonly alerts: number;
  readonly attention: string | null;
}

export interface MetricRow {
  readonly name: string;
  readonly label: string;
  readonly value: string | null;
  readonly unit: string;
  readonly status: StatusKey;
  readonly version: string;
  readonly coverage: string;
}

export interface JobRow {
  readonly id: string;
  readonly kind: string;
  readonly state: StatusKey;
  readonly completed?: number;
  readonly total?: number;
  readonly unit: string;
  readonly checkpoint?: string;
  readonly startedAt: string;
  readonly updatedAt: string;
  readonly coalesced?: number;
  readonly retryAfter?: string;
  readonly failure?: string;
  readonly transport?: 'stream' | 'polling';
}

export interface SourceRow {
  readonly kind: string;
  readonly url: string;
  readonly role: string;
  readonly state: StatusKey;
  readonly lastSuccess: string;
  readonly next: string;
  readonly coverage: string;
}

export interface AlertRow {
  readonly id: string;
  readonly severity: 'critical' | 'attention' | 'info';
  readonly rule: string;
  readonly ruleVersion: string;
  readonly project: string;
  readonly title: string;
  readonly detected: string;
  readonly state: 'open' | 'acknowledged' | 'resolved';
  readonly read: boolean;
  readonly occurrences: number;
  readonly evidence: string;
}

export const CUTOFF = '2026-08-20T14:35:00Z';
export const WINDOW = { label: '90d', from: '2026-05-22', to: '2026-08-19' } as const;
export const MEMBER = {
  displayName: 'Ana Silva',
  role: 'analyst',
  locale: 'en',
  timezone: 'America/Sao_Paulo',
} as const;

export const PROJECTS: readonly WorkspaceProject[] = [
  {
    id: '732684512931872768',
    name: 'Temporal',
    slug: 'temporal',
    description: 'Durable execution platform',
    state: 'active',
    repos: 3,
    sources: 8,
    recommendation: 'conditional',
    overall: 61,
    freshness: 'fresh',
    alerts: 2,
    attention: 'Top-three contributor concentration rose to 0.71',
  },
  {
    id: '732684513124761600',
    name: 'Cadence',
    slug: 'cadence',
    description: 'Workflow orchestration engine',
    state: 'active',
    repos: 2,
    sources: 6,
    recommendation: 'not_recommended',
    overall: 34,
    freshness: 'stale',
    alerts: 1,
    attention: 'No release in 214 days; maintainer count fell to 1',
  },
  {
    id: '732684513351254016',
    name: 'OpenTelemetry Collector',
    slug: 'otel-collector',
    description: 'Vendor-agnostic telemetry pipeline',
    state: 'active',
    repos: 4,
    sources: 9,
    recommendation: 'recommended',
    overall: 88,
    freshness: 'fresh',
    alerts: 0,
    attention: null,
  },
  {
    id: '732684513489666048',
    name: 'Conductor',
    slug: 'conductor',
    description: 'Microservice orchestration platform',
    state: 'active',
    repos: 2,
    sources: 5,
    recommendation: 'insufficient_data',
    overall: null,
    freshness: 'partial',
    alerts: 0,
    attention: '34 days of coverage; policy needs 90',
  },
  {
    id: '732684513598718976',
    name: 'Argo Workflows',
    slug: 'argo-workflows',
    description: 'Kubernetes-native workflow engine',
    state: 'paused',
    repos: 1,
    sources: 4,
    recommendation: 'conditional',
    overall: 57,
    freshness: 'stale',
    alerts: 0,
    attention: null,
  },
];

export const HEALTH: readonly HealthDimension[] = [
  { name: 'Activity', score: 0.82, status: 'available', rubric: 'activity v3' },
  { name: 'Community', score: 0.64, status: 'available', rubric: 'community v3' },
  { name: 'Maintenance', score: 0.47, status: 'available', rubric: 'maintenance v3' },
  { name: 'Concentration', score: 0.31, status: 'available', rubric: 'concentration v3' },
  { name: 'Stability', score: 0.71, status: 'available', rubric: 'stability v2' },
  {
    name: 'Security',
    status: 'unknown',
    note: 'no advisory source configured',
    rubric: 'security v2',
  },
  { name: 'Adoption', score: 0.58, status: 'available', rubric: 'adoption v2' },
];

export const METRICS: readonly MetricRow[] = [
  {
    name: 'release_frequency',
    label: 'Releases in previous 90 days',
    value: '8',
    unit: 'releases',
    status: 'available',
    version: 'v2',
    coverage: '90d',
  },
  {
    name: 'active_contributors_30d',
    label: 'Active contributors, previous 30 days',
    value: '119',
    unit: 'people',
    status: 'available',
    version: 'v3',
    coverage: '90d',
  },
  {
    name: 'issues_opened_closed_30d',
    label: 'Issues opened / closed, previous 30 days',
    value: '148 / 131',
    unit: 'issues',
    status: 'available',
    version: 'v3',
    coverage: '90d',
  },
  {
    name: 'median_time_to_first_response',
    label: 'Median time to first issue response',
    value: '9.4',
    unit: 'hours',
    status: 'available',
    version: 'v3',
    coverage: '90d',
  },
  {
    name: 'median_pr_merge_time',
    label: 'Median pull-request merge time',
    value: '31.2',
    unit: 'hours',
    status: 'stale',
    version: 'v3',
    coverage: '76d',
  },
  {
    name: 'backlog_change_30d',
    label: 'Backlog change, previous 30 days',
    value: '+17',
    unit: 'issues',
    status: 'available',
    version: 'v2',
    coverage: '90d',
  },
  {
    name: 'top_three_author_share_90d',
    label: "Top three commit authors' share",
    value: '0.71',
    unit: 'ratio',
    status: 'available',
    version: 'v3',
    coverage: '90d',
  },
  {
    name: 'time_since_last_release',
    label: 'Time since last release',
    value: '11',
    unit: 'days',
    status: 'available',
    version: 'v1',
    coverage: '90d',
  },
  {
    name: 'regression_issue_share',
    label: 'Regression-related issue share',
    value: null,
    unit: 'ratio',
    status: 'insufficient_data',
    version: 'v1',
    coverage: '34d',
  },
  {
    name: 'nuget_download_change',
    label: 'NuGet download change',
    value: null,
    unit: 'ratio',
    status: 'not_applicable',
    version: 'v2',
    coverage: '—',
  },
];

export const JOBS: readonly JobRow[] = [
  {
    id: '732684512948649984',
    kind: 'initial_sync',
    state: 'running',
    completed: 6,
    total: 8,
    unit: 'sources',
    checkpoint: 'github:pull_requests:2026-03-14T00:00:00Z',
    startedAt: '2026-08-20T14:31:08Z',
    updatedAt: '2026-08-20T14:34:18Z',
    coalesced: 2,
    transport: 'stream',
  },
  {
    id: '732684513191870464',
    kind: 'documentation_crawl',
    state: 'running',
    unit: 'documents',
    startedAt: '2026-08-20T14:22:01Z',
    updatedAt: '2026-08-20T14:34:02Z',
    transport: 'polling',
  },
  {
    id: '732684513200259072',
    kind: 'history_request',
    state: 'failed',
    unit: 'sources',
    startedAt: '2026-08-20T09:02:44Z',
    updatedAt: '2026-08-20T09:04:10Z',
    retryAfter: '1830s',
    failure:
      'npm registry rate limit reached. Quota resets at 15:05Z; the request will resume from its checkpoint.',
  },
  {
    id: '732684513258979328',
    kind: 'recalculation',
    state: 'succeeded',
    completed: 12,
    total: 12,
    unit: 'metrics',
    startedAt: '2026-08-19T22:10:00Z',
    updatedAt: '2026-08-19T22:12:37Z',
  },
];

export const SOURCES: readonly SourceRow[] = [
  {
    kind: 'github',
    url: 'github.com/temporalio/temporal',
    role: 'core',
    state: 'ready',
    lastSuccess: '2026-08-20T14:31Z',
    next: '2026-08-20T20:00Z',
    coverage: '180d',
  },
  {
    kind: 'github',
    url: 'github.com/temporalio/sdk-go',
    role: 'sdk',
    state: 'ready',
    lastSuccess: '2026-08-20T14:31Z',
    next: '2026-08-20T20:00Z',
    coverage: '180d',
  },
  {
    kind: 'npm',
    url: 'npmjs.com/package/@temporalio/client',
    role: 'package',
    state: 'ready',
    lastSuccess: '2026-08-20T12:00Z',
    next: '2026-08-21T00:00Z',
    coverage: '180d',
  },
  {
    kind: 'nuget',
    url: '—',
    role: 'package',
    state: 'not_applicable',
    lastSuccess: '—',
    next: '—',
    coverage: '—',
  },
  {
    kind: 'documentation',
    url: 'docs.temporal.io',
    role: 'documentation',
    state: 'running',
    lastSuccess: '2026-08-13T04:00Z',
    next: 'in progress',
    coverage: '142 documents',
  },
  {
    kind: 'advisory',
    url: 'GitHub advisories',
    role: 'advisory',
    state: 'stale',
    lastSuccess: '2026-08-14T04:00Z',
    next: '2026-08-21T04:00Z',
    coverage: '365d',
  },
  {
    kind: 'discussion',
    url: 'github.com/temporalio/temporal/discussions',
    role: 'discussion',
    state: 'failed',
    lastSuccess: '2026-08-18T04:00Z',
    next: 'retry 15:05Z',
    coverage: '97d',
  },
  {
    kind: 'rss',
    url: 'temporal.io/blog/rss',
    role: 'changelog',
    state: 'ready',
    lastSuccess: '2026-08-20T06:00Z',
    next: '2026-08-21T06:00Z',
    coverage: '365d',
  },
];

export const ALERTS: readonly AlertRow[] = [
  {
    id: '732684513601',
    severity: 'critical',
    rule: 'Public security evidence',
    ruleVersion: 'v2',
    project: 'Cadence',
    title: 'Security release referenced in changelog without an advisory',
    detected: '2026-08-20T11:02Z',
    state: 'open',
    read: false,
    occurrences: 1,
    evidence: 'changelog · v1.2.9',
  },
  {
    id: '732684513602',
    severity: 'attention',
    rule: 'Contributor concentration',
    ruleVersion: 'v3',
    project: 'Temporal',
    title: 'Top-three author share rose above 0.60',
    detected: '2026-08-20T08:41Z',
    state: 'acknowledged',
    read: true,
    occurrences: 3,
    evidence: 'metric · top_three_author_share',
  },
  {
    id: '732684513603',
    severity: 'info',
    rule: 'Breaking change',
    ruleVersion: 'v1',
    project: 'OpenTelemetry Collector',
    title: 'Breaking change claimed in v0.104.0',
    detected: '2026-08-19T17:20Z',
    state: 'open',
    read: true,
    occurrences: 1,
    evidence: 'release · v0.104.0',
  },
  {
    id: '732684513604',
    severity: 'attention',
    rule: 'Early warning · maintainer loss',
    ruleVersion: 'v2',
    project: 'Cadence',
    title: 'Forecast: maintainer count reaches 0 within 120 days',
    detected: '2026-08-18T06:00Z',
    state: 'open',
    read: false,
    occurrences: 2,
    evidence: 'forecast · maintainer_count v2',
  },
];

export const RADAR: readonly RadarEntry[] = [
  { project: 'OpenTelemetry Collector', policyRing: 'adopt', effectiveRing: 'adopt' },
  {
    project: 'Temporal',
    policyRing: 'trial',
    effectiveRing: 'adopt',
    override: { owner: 'A. Silva' },
    reviewDue: '2026-11-20',
  },
  { project: 'Argo Workflows', policyRing: 'trial', effectiveRing: 'trial' },
  {
    project: 'Conductor',
    policyRing: 'unplaced',
    effectiveRing: 'assess',
    override: { owner: 'R. Costa', expired: true },
    reviewDue: '2026-07-01',
    reviewOverdue: true,
  },
  { project: 'Cadence', policyRing: 'hold', effectiveRing: 'hold' },
];

export const TREND_SERIES: readonly SeriesPoint[] = [
  { label: '2026-04', value: 44 },
  { label: '2026-05', value: 41 },
  { label: '2026-06', value: 38 },
  { label: '2026-07', value: 31 },
  { label: '2026-08', value: 27 },
];

export const TREND_FORECAST: readonly SeriesPoint[] = [
  { label: '2026-09', value: 24 },
  { label: '2026-10', value: 21 },
];
