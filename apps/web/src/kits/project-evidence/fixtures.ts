/**
 * Fixtures for the project-evidence shell. ILLUSTRATIVE VALUES — they are shaped like the HTTP
 * contract (Snowflake identifiers as decimal strings, every result carrying a window and a cutoff,
 * statuses that are never numeric zero) but no number here was derived from a real repository.
 * They were chosen to make each specified state visible: 0.71 concentration against a 0.60
 * threshold, 34 days of coverage against a 90-day requirement, a withdrawn advisory that a
 * changelog still references. Replace this one file to see the shell against real collections.
 */
import type {
  CoverageDisclosureProps,
  EvidenceLinkProps,
  HealthDimension,
  IconName,
  StatusKey,
} from '../../design-system';

export const CUTOFF = '2026-08-20T14:35:00Z';
export const WINDOW = { label: '90d', from: '2026-05-22', to: '2026-08-19' } as const;
export const MEMBER = {
  displayName: 'Ana Silva',
  role: 'analyst',
  locale: 'en',
  timezone: 'America/Sao_Paulo',
} as const;

export const PROJECT = {
  id: '732684512931872768',
  name: 'Temporal',
  slug: 'temporal',
  description: 'Durable execution platform',
  state: 'active',
  registeredAt: '2025-11-04T09:12:44Z',
  registeredBy: 'Ana Silva',
  primaryRepository: 'github.com/temporalio/temporal',
  repositories: 3,
  sources: 8,
  license: 'MIT',
  languages: 'Go, TypeScript, Java',
  firstCommit: '2019-02-27',
  lastCollection: '2026-08-20T14:31:08Z',
} as const;

/* S9 — Overview */

export interface HeadlineMetric {
  readonly name: string;
  readonly label: string;
  readonly value?: string;
  readonly unit?: string;
  readonly status: StatusKey;
  readonly version: string;
  readonly delta?: string;
  readonly deltaDirection?: 'up' | 'down' | 'flat';
  readonly note?: string;
}

export const OVERVIEW = {
  overall: { calculable: true, score: 61, version: 'rubric v3' },
  dimensions: [
    { name: 'Activity', score: 0.82, status: 'available', rubric: 'activity v3' },
    { name: 'Community', score: 0.64, status: 'available', rubric: 'community v3' },
    { name: 'Maintenance', score: 0.47, status: 'available', rubric: 'maintenance v3' },
    { name: 'Concentration', score: 0.31, status: 'available', rubric: 'concentration v3' },
    { name: 'Stability', score: 0.71, status: 'available', rubric: 'stability v2' },
    {
      name: 'Security',
      status: 'unknown',
      rubric: 'security v2',
      note: 'no advisory source configured',
    },
    { name: 'Adoption', score: 0.58, status: 'available', rubric: 'adoption v2' },
  ] as readonly HealthDimension[],
  headline: [
    {
      name: 'release_frequency',
      label: 'Releases',
      value: '8',
      unit: 'releases',
      status: 'available',
      version: 'v2',
      delta: '+2',
      deltaDirection: 'up',
    },
    {
      name: 'active_contributors_30d',
      label: 'Active contributors',
      value: '119',
      unit: 'people',
      status: 'available',
      version: 'v3',
      delta: '−7',
      deltaDirection: 'down',
    },
    {
      name: 'median_pr_merge_time',
      label: 'Median PR merge time',
      value: '31.2',
      unit: 'hours',
      status: 'stale',
      version: 'v3',
    },
    {
      name: 'open_advisories',
      label: 'Public advisories',
      status: 'unknown',
      version: 'v2',
      note: 'no advisory source configured',
    },
  ] as readonly HeadlineMetric[],
  zeroValue: {
    name: 'breaking_changes_90d',
    label: 'Breaking changes claimed',
    value: '0',
    unit: 'claims',
    status: 'available',
    version: 'v1',
  } as HeadlineMetric,
  conditions: [
    'Review top-three contributor concentration before production adoption.',
    'Configure an advisory source before treating the security dimension as evaluated.',
  ],
  missing: ['regression_issue_share', 'nuget_download_change'],
  coverage: {
    requested: '90d',
    actual: '90d',
    ratio: 1,
    sources: [
      { name: 'github', value: '180d' },
      { name: 'npm', value: '180d' },
      { name: 'documentation', value: '142 documents' },
    ],
    missing: ['nuget'],
    cutoff: CUTOFF,
  } as CoverageDisclosureProps,
  partialCoverage: {
    requested: '90d',
    actual: '34d',
    ratio: 0.38,
    sources: [{ name: 'github', value: '34d' }],
    missing: ['npm', 'advisory', 'discussion'],
    cutoff: CUTOFF,
  } as CoverageDisclosureProps,
} as const;

/* S10 — Contributor intelligence */

export interface Contributor {
  readonly handle: string;
  readonly name: string;
  readonly role: string;
  readonly commits: number;
  readonly share: string;
  readonly firstSeen: string;
  readonly lastSeen: string;
  readonly kind: 'person' | 'bot' | 'service';
  readonly identities: number;
}

export interface Cohort {
  readonly cohort: string;
  readonly joined: number;
  readonly active90d: number | null;
  readonly retained: string | null;
  readonly status: StatusKey;
}

export interface UnresolvedIdentity {
  readonly id: string;
  readonly display: string;
  readonly suggestion: string | null;
  readonly confidence: string | null;
  readonly basis: string;
}

export const CONTRIBUTORS = {
  concentration: {
    topOne: '0.34',
    topThree: '0.71',
    threshold: '0.60',
    version: 'v3',
    authors: 341,
  },
  people: [
    {
      handle: 'maru-sama',
      name: 'Maru Tanaka',
      role: 'maintainer',
      commits: 412,
      share: '0.19',
      firstSeen: '2019-06-11',
      lastSeen: '2026-08-19',
      kind: 'person',
      identities: 2,
    },
    {
      handle: 'j-hoffmann',
      name: 'Jonas Hoffmann',
      role: 'maintainer',
      commits: 388,
      share: '0.18',
      firstSeen: '2020-01-08',
      lastSeen: '2026-08-18',
      kind: 'person',
      identities: 1,
    },
    {
      handle: 'aparna-r',
      name: 'Aparna Rao',
      role: 'committer',
      commits: 297,
      share: '0.14',
      firstSeen: '2021-03-22',
      lastSeen: '2026-08-20',
      kind: 'person',
      identities: 3,
    },
    {
      handle: 'dependabot[bot]',
      name: 'Dependabot',
      role: 'automation',
      commits: 1204,
      share: '—',
      firstSeen: '2019-08-02',
      lastSeen: '2026-08-20',
      kind: 'bot',
      identities: 1,
    },
    {
      handle: 'temporal-ci',
      name: 'Temporal CI',
      role: 'service account',
      commits: 806,
      share: '—',
      firstSeen: '2020-05-14',
      lastSeen: '2026-08-20',
      kind: 'service',
      identities: 1,
    },
    {
      handle: 'lgomes',
      name: 'Lucas Gomes',
      role: 'contributor',
      commits: 44,
      share: '0.02',
      firstSeen: '2025-11-30',
      lastSeen: '2026-07-04',
      kind: 'person',
      identities: 1,
    },
  ] as readonly Contributor[],
  cohorts: [
    { cohort: '2026-Q1', joined: 61, active90d: 38, retained: '0.62', status: 'available' },
    { cohort: '2025-Q4', joined: 74, active90d: 41, retained: '0.55', status: 'available' },
    { cohort: '2025-Q3', joined: 58, active90d: 26, retained: '0.45', status: 'available' },
    {
      cohort: '2026-Q2',
      joined: 49,
      active90d: null,
      retained: null,
      status: 'insufficient_data',
    },
  ] as readonly Cohort[],
  unresolved: [
    {
      id: '732684514001',
      display: 'maru.tanaka@example.org',
      suggestion: 'maru-sama',
      confidence: '0.91',
      basis: 'commit e-mail matches 214 commits authored by maru-sama',
    },
    {
      id: '732684514002',
      display: 'M. Tanaka',
      suggestion: 'maru-sama',
      confidence: '0.58',
      basis: 'display-name similarity only; no shared commit e-mail',
    },
    {
      id: '732684514003',
      display: 'aparna@build.internal',
      suggestion: null,
      confidence: null,
      basis: 'no evidence links this address to a known contributor',
    },
  ] as readonly UnresolvedIdentity[],
  conflict: {
    identity: 'M. Tanaka',
    a: { handle: 'maru-sama', by: 'Ana Silva', at: '2026-08-19T10:04Z' },
    b: { handle: 'm-tanaka-2', by: 'Rafael Costa', at: '2026-08-19T10:06Z' },
  },
} as const;

/* S11 — Adoption and security */

export interface Registry {
  readonly registry: string;
  readonly pkg: string | null;
  readonly unit: string | null;
  readonly value: string | null;
  readonly window?: string;
  readonly change?: string | null;
  readonly direction?: 'up' | 'down' | 'flat';
  readonly normalized?: string | null;
  readonly status: StatusKey;
  readonly version: string;
  readonly incomparable?: string;
  readonly note?: string;
  readonly collectedAt: string | null;
}

export interface Advisory {
  readonly id: string;
  readonly title: string;
  readonly severity: 'critical' | 'high' | 'moderate' | 'low' | 'unknown';
  readonly published: string;
  readonly affected: string;
  readonly fixedIn: string | null;
  readonly state: 'resolved' | 'conflicting';
  readonly source: string;
  readonly note?: string;
  readonly cutoff: string;
}

export const ADOPTION = {
  registries: [
    {
      registry: 'npm',
      pkg: '@temporalio/client',
      unit: 'downloads',
      value: '1,940,221',
      window: '90d',
      change: '+11.4%',
      direction: 'up',
      normalized: '21,558 / day',
      status: 'available',
      version: 'v2',
      collectedAt: '2026-08-20T12:00Z',
    },
    {
      registry: 'PyPI',
      pkg: 'temporalio',
      unit: 'downloads',
      value: '402,873',
      window: '90d',
      change: '+4.1%',
      direction: 'up',
      normalized: '4,476 / day',
      status: 'available',
      version: 'v2',
      collectedAt: '2026-08-20T12:00Z',
    },
    {
      registry: 'Maven Central',
      pkg: 'io.temporal:temporal-sdk',
      unit: 'resolutions',
      value: '88,140',
      window: '90d',
      change: null,
      direction: 'flat',
      normalized: null,
      status: 'available',
      version: 'v2',
      incomparable:
        'Maven Central reports unique-IP resolutions, not downloads. Not normalized against npm or PyPI.',
      collectedAt: '2026-08-19T12:00Z',
    },
    {
      registry: 'Docker Hub',
      pkg: 'temporalio/server',
      unit: 'pulls',
      value: null,
      status: 'unknown',
      version: 'v2',
      note: 'source unavailable since 2026-08-17T04:00Z; last successful collection returned 62,113,904 pulls',
      collectedAt: '2026-08-17T04:00Z',
    },
    {
      registry: 'NuGet',
      pkg: null,
      unit: null,
      value: null,
      status: 'not_applicable',
      version: 'v2',
      note: 'no NuGet package is linked to this project',
      collectedAt: null,
    },
  ] as readonly Registry[],
  security: [
    {
      id: 'GHSA-7v4x-9m2q-pp58',
      title: 'Server accepts unsigned workflow task tokens',
      severity: 'high',
      published: '2026-06-14',
      affected: '< 1.24.3',
      fixedIn: '1.24.3',
      state: 'resolved',
      source: 'GitHub advisories',
      cutoff: '2026-08-14T04:00Z',
    },
    {
      id: 'CVE-2026-24118',
      title: 'Denial of service through unbounded history replay',
      severity: 'moderate',
      published: '2026-03-02',
      affected: '< 1.22.0',
      fixedIn: '1.22.0',
      state: 'resolved',
      source: 'GitHub advisories',
      cutoff: '2026-08-14T04:00Z',
    },
    {
      id: 'GHSA-3c8p-r4jd-x21h',
      title: 'Advisory withdrawn by publisher',
      severity: 'unknown',
      published: '2026-05-08',
      affected: 'unknown',
      fixedIn: null,
      state: 'conflicting',
      source: 'GitHub advisories',
      note: 'GitHub lists this advisory as withdrawn; the project changelog for v1.23.1 still describes a security fix referencing it. Both records are retained.',
      cutoff: '2026-08-14T04:00Z',
    },
  ] as readonly Advisory[],
  securityRelease: {
    tag: 'v1.23.1',
    date: '2026-05-11',
    claim: 'Security fix for advisory GHSA-3c8p-r4jd-x21h',
    source: 'changelog',
  },
} as const;

/* S17 — Release intelligence */

export interface ReleaseRow {
  readonly tag: string;
  readonly date: string;
  readonly state: StatusKey;
  readonly prerelease: boolean;
  readonly duplicate?: boolean;
  readonly runId: string | null;
  readonly claims: number | null;
  readonly breaking: number | null;
  readonly security: number | null;
}

export interface ReleaseClaim {
  readonly text: string;
  readonly confidence?: string;
  readonly note?: string;
  readonly cites: readonly EvidenceLinkProps[];
}

export interface ClaimGroupData {
  readonly kind: 'breaking' | 'feature' | 'deprecation' | 'security' | 'performance' | 'dx';
  readonly label: string;
  readonly glyph: IconName;
  readonly claims: readonly ReleaseClaim[];
  readonly empty?: string;
}

export const RELEASES: readonly ReleaseRow[] = [
  {
    tag: 'v1.25.0',
    date: '2026-08-09',
    state: 'succeeded',
    prerelease: false,
    runId: '732684515221',
    claims: 11,
    breaking: 2,
    security: 0,
  },
  {
    tag: 'v1.24.4',
    date: '2026-07-22',
    state: 'succeeded',
    prerelease: false,
    runId: '732684515190',
    claims: 6,
    breaking: 0,
    security: 1,
  },
  {
    tag: 'v1.25.0-rc.2',
    date: '2026-07-30',
    state: 'queued',
    prerelease: true,
    runId: null,
    claims: null,
    breaking: null,
    security: null,
  },
  {
    tag: 'v1.24.3',
    date: '2026-06-14',
    state: 'stale',
    prerelease: false,
    runId: '732684515140',
    claims: 8,
    breaking: 0,
    security: 1,
  },
  {
    tag: 'v1.24.2',
    date: '2026-05-28',
    state: 'failed',
    prerelease: false,
    runId: '732684515101',
    claims: null,
    breaking: null,
    security: null,
  },
  {
    tag: 'v1.24.1',
    date: '2026-05-11',
    state: 'unknown',
    prerelease: false,
    runId: null,
    claims: null,
    breaking: null,
    security: null,
  },
];

export const RELEASE_DETAIL = {
  tag: 'v1.25.0',
  date: '2026-08-09T18:02:00Z',
  author: 'temporal-ci',
  run: {
    runId: '732684515221',
    provider: 'anthropic',
    model: 'claude-sonnet-4-5',
    promptVersion: 'release-claims v4',
    language: 'en',
    executedAt: '2026-08-09T18:44:12Z',
    versionLabel: 'v2 of 2',
    usage: '8.4k tokens',
  },
  groups: [
    {
      kind: 'breaking',
      label: 'Breaking changes',
      glyph: 'octagon-x',
      claims: [
        {
          text: 'The deprecated `DescribeWorkflowExecution` v1 response field `pendingActivities` is removed.',
          cites: [
            {
              kind: 'changelog',
              title: 'CHANGELOG.md · Removed',
              href: '#',
              source: 'github · temporalio/temporal',
              collectedAt: '2026-08-09T18:04Z',
            },
            {
              kind: 'pull_request',
              title: '#7412 remove pendingActivities from v1 response',
              href: '#',
              source: 'github',
              collectedAt: '2026-08-09T18:04Z',
            },
          ],
        },
        {
          text: 'Minimum supported Go version moves to 1.24.',
          cites: [
            {
              kind: 'changelog',
              title: 'CHANGELOG.md · Requirements',
              href: '#',
              source: 'github',
              collectedAt: '2026-08-09T18:04Z',
            },
          ],
        },
      ],
    },
    {
      kind: 'feature',
      label: 'Features',
      glyph: 'circle-dot',
      claims: [
        {
          text: 'Workflow update accepts a per-request timeout.',
          cites: [
            {
              kind: 'release',
              title: 'Release notes · Added',
              href: '#',
              source: 'github',
              collectedAt: '2026-08-09T18:04Z',
            },
          ],
        },
        {
          text: 'Nexus operations expose retry state in the describe API.',
          cites: [
            {
              kind: 'pull_request',
              title: '#7388 expose Nexus retry state',
              href: '#',
              source: 'github',
              collectedAt: '2026-08-09T18:04Z',
            },
          ],
        },
      ],
    },
    {
      kind: 'deprecation',
      label: 'Deprecations',
      glyph: 'clock',
      claims: [
        {
          text: '`StartWorkflowExecution` request field `retryPolicy.nonRetryableErrorTypes` is deprecated in favour of `nonRetryableErrors`.',
          cites: [
            {
              kind: 'changelog',
              title: 'CHANGELOG.md · Deprecated',
              href: '#',
              source: 'github',
              collectedAt: '2026-08-09T18:04Z',
            },
          ],
        },
      ],
    },
    {
      kind: 'security',
      label: 'Security',
      glyph: 'shield-alert',
      claims: [],
      empty:
        "No security claim was found in this release's evidence. This is not a statement that the release contains no vulnerability.",
    },
    {
      kind: 'performance',
      label: 'Performance',
      glyph: 'trending-up',
      claims: [
        {
          text: 'History replay allocates 18% less memory for workflows with more than 10,000 events.',
          confidence: 'low',
          cites: [
            {
              kind: 'release',
              title: 'Release notes · Performance',
              href: '#',
              source: 'github',
              collectedAt: '2026-08-09T18:04Z',
            },
          ],
          note: 'Single unquantified source; the 18% figure is quoted from the release notes and was not independently measured.',
        },
      ],
    },
    {
      kind: 'dx',
      label: 'Developer experience',
      glyph: 'book-open',
      claims: [
        {
          text: 'The CLI prints a machine-readable error envelope with `--output json`.',
          cites: [
            {
              kind: 'changelog',
              title: 'CHANGELOG.md · Changed',
              href: '#',
              source: 'github',
              collectedAt: '2026-08-09T18:04Z',
            },
          ],
        },
      ],
    },
  ] as readonly ClaimGroupData[],
} as const;

/* S18 — Documentation knowledge */

export interface CrawlDomain {
  readonly domain: string;
  readonly scope: string;
  readonly state: StatusKey;
  readonly documents: number;
  readonly depth: string;
  readonly lastCrawl: string;
  readonly bytes: string;
  readonly failure?: string;
}

export interface CrawlLimit {
  readonly rule: string;
  readonly value: string;
  readonly hit: boolean;
  readonly detail?: string;
}

export interface KnowledgeResult {
  readonly title: string;
  readonly snapshot: string;
  readonly url: string;
  readonly exact: boolean;
  readonly answer: string;
  readonly cites: readonly EvidenceLinkProps[];
}

export interface LexicalMatch {
  readonly title: string;
  readonly url: string;
  readonly score: string;
  readonly terms: string;
}

export const KNOWLEDGE = {
  domains: [
    {
      domain: 'docs.temporal.io',
      scope: '/docs/**',
      state: 'ready',
      documents: 142,
      depth: '4 of 4',
      lastCrawl: '2026-08-13T04:00Z',
      bytes: '38.2 MB',
    },
    {
      domain: 'temporal.io',
      scope: '/blog/**',
      state: 'running',
      documents: 61,
      depth: '2 of 3',
      lastCrawl: 'in progress',
      bytes: '9.4 MB',
    },
    {
      domain: 'learn.temporal.io',
      scope: '/**',
      state: 'failed',
      documents: 0,
      depth: '—',
      lastCrawl: '2026-08-12T04:00Z',
      bytes: '—',
      failure: 'robots.txt disallows /; no documents were fetched.',
    },
    {
      domain: 'community.temporal.io',
      scope: '/t/**',
      state: 'stale',
      documents: 318,
      depth: '3 of 3',
      lastCrawl: '2026-06-02T04:00Z',
      bytes: '51.7 MB',
    },
  ] as readonly CrawlDomain[],
  limits: [
    { rule: 'Maximum crawl depth', value: '4 levels', hit: false },
    {
      rule: 'Maximum document size',
      value: '2 MB',
      hit: true,
      detail: '7 documents exceeded 2 MB and were skipped',
    },
    {
      rule: 'Accepted content types',
      value: 'text/html, text/markdown, application/pdf',
      hit: true,
      detail: '3 responses were application/zip and were skipped',
    },
    {
      rule: 'Redirect policy',
      value: 'same registrable domain only',
      hit: true,
      detail: '1 redirect to an unrelated host was refused',
    },
  ] as readonly CrawlLimit[],
  query: 'How is a workflow task timeout retried?',
  results: [
    {
      title: 'Workflow task timeouts',
      snapshot: '2026-08-13T04:00Z',
      url: 'docs.temporal.io/docs/concepts/workflow-task-timeout',
      exact: true,
      answer:
        'A workflow task timeout schedules a new workflow task rather than failing the workflow execution. The retry is unlimited by default and is not governed by the activity retry policy.',
      cites: [
        {
          kind: 'document',
          title: 'Workflow task timeout · paragraph 3',
          href: '#',
          source: 'docs snapshot 2026-08-13',
          collectedAt: '2026-08-13T04:00Z',
        },
        {
          kind: 'document',
          title: 'Workflow execution lifecycle · Timeouts',
          href: '#',
          source: 'docs snapshot 2026-08-13',
          collectedAt: '2026-08-13T04:00Z',
        },
      ],
    },
    {
      title: 'Retry policies',
      snapshot: '2026-08-13T04:00Z',
      url: 'docs.temporal.io/docs/concepts/retry-policies',
      exact: false,
      answer:
        'Retry policies apply to activities and child workflows. Workflow tasks are excluded.',
      cites: [
        {
          kind: 'document',
          title: 'Retry policies · Scope',
          href: '#',
          source: 'docs snapshot 2026-08-13',
          collectedAt: '2026-08-13T04:00Z',
        },
      ],
    },
  ] as readonly KnowledgeResult[],
  conflicting: {
    a: { snapshot: '2026-08-13T04:00Z', text: 'The retry is unlimited by default.' },
    b: { snapshot: '2026-06-02T04:00Z', text: 'The retry is capped at 10 attempts.' },
  },
  lexical: [
    {
      title: 'Workflow task timeouts',
      url: 'docs.temporal.io/docs/concepts/workflow-task-timeout',
      score: '18.4',
      terms: 'workflow, task, timeout, retried',
    },
    {
      title: 'Timeouts reference',
      url: 'docs.temporal.io/docs/references/timeouts',
      score: '14.1',
      terms: 'timeout, retry',
    },
    {
      title: 'Retry policies',
      url: 'docs.temporal.io/docs/concepts/retry-policies',
      score: '11.7',
      terms: 'retry',
    },
  ] as readonly LexicalMatch[],
} as const;

/* S22 — Exports */

export interface ExportJob {
  readonly id: string;
  readonly kind: string;
  readonly state: StatusKey;
  readonly completed?: number;
  readonly total?: number;
  readonly unit?: string;
  readonly startedAt: string;
  readonly updatedAt: string;
  readonly checkpoint?: string;
  readonly transport?: 'stream' | 'polling';
  readonly coalesced?: number;
  readonly retryAfter?: string;
  readonly failure?: string;
  readonly resource: string;
  readonly format: string;
}

export interface Artifact {
  readonly id: string;
  readonly resource: string;
  readonly format: string;
  readonly rows: string;
  readonly bytes: string;
  readonly cutoff: string;
  readonly locale: string;
  readonly version: string;
  readonly expiresAt: string;
  readonly expiresIn: string | null;
  readonly state: 'ready' | 'expired';
}

export const EXPORTS = {
  resources: [
    { value: 'metrics', label: 'Metric values' },
    { value: 'health', label: 'Health dimensions' },
    { value: 'contributors', label: 'Contributors' },
    { value: 'releases', label: 'Releases and claims' },
    { value: 'advisories', label: 'Public security evidence' },
    { value: 'audit', label: 'Audit events' },
  ],
  jobs: [
    {
      id: '732684516401',
      kind: 'export',
      state: 'running',
      completed: 1840,
      total: 4120,
      unit: 'rows',
      startedAt: '2026-08-20T14:33:02Z',
      updatedAt: '2026-08-20T14:34:51Z',
      checkpoint: 'metrics:cursor:eyJvIjoxODQwfQ',
      transport: 'stream',
      resource: 'Metric values',
      format: 'csv',
    },
    {
      id: '732684516388',
      kind: 'export',
      state: 'succeeded',
      completed: 4120,
      total: 4120,
      unit: 'rows',
      startedAt: '2026-08-20T13:02:00Z',
      updatedAt: '2026-08-20T13:03:44Z',
      coalesced: 2,
      resource: 'Releases and claims',
      format: 'json',
    },
    {
      id: '732684516301',
      kind: 'export',
      state: 'succeeded',
      completed: 0,
      total: 0,
      unit: 'rows',
      startedAt: '2026-08-19T09:10:00Z',
      updatedAt: '2026-08-19T09:10:12Z',
      resource: 'Public security evidence',
      format: 'csv',
    },
    {
      id: '732684516244',
      kind: 'export',
      state: 'failed',
      startedAt: '2026-08-18T22:41:00Z',
      updatedAt: '2026-08-18T22:43:19Z',
      retryAfter: '600s',
      resource: 'Contributors',
      format: 'json',
      failure:
        'Export exceeded the 200 MB artifact quota at 4,120 of 61,004 rows. Narrow the window or the resource and request again.',
    },
    {
      id: '732684516190',
      kind: 'export',
      state: 'cancelled',
      startedAt: '2026-08-18T11:00:00Z',
      updatedAt: '2026-08-18T11:00:38Z',
      resource: 'Audit events',
      format: 'csv',
    },
  ] as readonly ExportJob[],
  artifacts: [
    {
      id: '732684516388',
      resource: 'Releases and claims',
      format: 'evidence JSON',
      rows: '4,120',
      bytes: '18.4 MB',
      cutoff: '2026-08-20T13:02:00Z',
      locale: 'en',
      version: 'catalog v3',
      expiresAt: '2026-08-21T13:03:44Z',
      expiresIn: '22 h 29 min',
      state: 'ready',
    },
    {
      id: '732684516301',
      resource: 'Public security evidence',
      format: 'CSV',
      rows: '0',
      bytes: '412 B',
      cutoff: '2026-08-19T09:10:00Z',
      locale: 'en',
      version: 'catalog v3',
      expiresAt: '2026-08-20T09:10:12Z',
      expiresIn: null,
      state: 'expired',
    },
  ] as readonly Artifact[],
} as const;

/* S20 — AI run governance */

export interface RunVersion {
  readonly runId: string;
  readonly state: StatusKey;
  readonly provider: string;
  readonly model: string;
  readonly promptVersion: string;
  readonly language: string;
  readonly executedAt: string;
  readonly versionLabel: string;
  readonly selected: boolean;
  readonly usage: string;
  readonly claims: number;
  readonly cites: number;
}

export interface RunDiff {
  readonly field: string;
  readonly a: string;
  readonly b: string;
  readonly changed: boolean;
}

export interface ProviderStatus {
  readonly provider: string;
  readonly capability: string;
  readonly state: StatusKey;
  readonly quota: string;
  readonly health: string;
  readonly note: string | null;
}

export const RUNS = {
  subject: 'Release claims · v1.25.0',
  versions: [
    {
      runId: '732684515221',
      state: 'succeeded',
      provider: 'anthropic',
      model: 'claude-sonnet-4-5',
      promptVersion: 'release-claims v4',
      language: 'en',
      executedAt: '2026-08-09T18:44:12Z',
      versionLabel: 'v2 of 2',
      selected: false,
      usage: '8.4k tokens',
      claims: 11,
      cites: 19,
    },
    {
      runId: '732684515208',
      state: 'succeeded',
      provider: 'anthropic',
      model: 'claude-sonnet-4-5',
      promptVersion: 'release-claims v3',
      language: 'en',
      executedAt: '2026-08-09T18:12:07Z',
      versionLabel: 'v1 of 2',
      selected: true,
      usage: '7.9k tokens',
      claims: 9,
      cites: 14,
    },
  ] as readonly RunVersion[],
  diff: [
    { field: 'Claims extracted', a: '9', b: '11', changed: true },
    { field: 'Breaking changes', a: '2', b: '2', changed: false },
    { field: 'Security claims', a: '0', b: '0', changed: false },
    { field: 'Claims without a citation', a: '1', b: '0', changed: true },
    { field: 'Performance claim confidence', a: 'not reported', b: 'low', changed: true },
    { field: 'Prompt version', a: 'release-claims v3', b: 'release-claims v4', changed: true },
  ] as readonly RunDiff[],
  failedRun: {
    runId: '732684515101',
    state: 'failed' as StatusKey,
    provider: 'anthropic',
    model: 'claude-sonnet-4-5',
    promptVersion: 'release-claims v4',
    language: 'en',
    executedAt: '2026-05-28T22:41:09Z',
    versionLabel: 'v1 of 1',
    usage: '1.2k tokens',
  },
  providers: [
    {
      provider: 'anthropic',
      capability: 'claim extraction, topic analysis, assistant',
      state: 'running',
      quota: '68% of daily token budget used',
      health: 'healthy',
      note: null,
    },
    {
      provider: 'openai',
      capability: 'translation',
      state: 'paused',
      quota: '—',
      health: 'disabled by Admin',
      note: 'Generated translations are unavailable. Evidence is served in its original language.',
    },
    {
      provider: 'voyage',
      capability: 'documentation retrieval',
      state: 'failed',
      quota: 'quota exhausted 14:02Z',
      health: 'unhealthy',
      note: 'Retrieval falls back to lexical search. Deterministic metrics and health are unaffected.',
    },
  ] as readonly ProviderStatus[],
  aggregate: [
    { period: '2026-08 to date', runs: '1,204', tokens: '9.1M', failures: '31', coalesced: '88' },
    { period: '2026-07', runs: '4,880', tokens: '36.4M', failures: '142', coalesced: '331' },
  ],
} as const;

/* S7 — Repositories, sources and associations */

export interface Repository {
  readonly id: string;
  readonly url: string;
  readonly role: string;
  readonly state: StatusKey;
  readonly stars: string;
  readonly lastCollection: string;
  readonly derived: string;
}

export interface SourceInventoryRow {
  readonly kind: string;
  readonly url: string;
  readonly scope: string;
  readonly state: StatusKey;
  readonly credential: string;
  readonly limit: string;
}

export interface Association {
  readonly id: string;
  readonly source: string;
  readonly target: string;
  readonly confidence: string;
  readonly basis: string;
  readonly state: 'automatic' | 'uncertain' | 'corrected';
  readonly correctedBy?: string;
  readonly correctedAt?: string;
  readonly note?: string;
}

export const REPOSITORIES: readonly Repository[] = [
  {
    id: '732684517001',
    url: 'github.com/temporalio/temporal',
    role: 'primary',
    state: 'ready',
    stars: '12.4k',
    lastCollection: '2026-08-20T14:31Z',
    derived: 'fresh',
  },
  {
    id: '732684517002',
    url: 'github.com/temporalio/sdk-go',
    role: 'sdk',
    state: 'ready',
    stars: '1.3k',
    lastCollection: '2026-08-20T14:31Z',
    derived: 'fresh',
  },
  {
    id: '732684517003',
    url: 'github.com/temporalio/documentation',
    role: 'documentation',
    state: 'stale',
    stars: '312',
    lastCollection: '2026-08-13T04:00Z',
    derived: 'stale since 2026-08-13T04:00Z',
  },
];

export const SOURCE_INVENTORY: readonly SourceInventoryRow[] = [
  {
    kind: 'github',
    url: 'github.com/temporalio/temporal',
    scope: 'issues, pull requests, releases, commits, discussions',
    state: 'ready',
    credential: 'installation token · redacted',
    limit: '4,412 of 5,000 requests / hour',
  },
  {
    kind: 'npm',
    url: 'npmjs.com/package/@temporalio/client',
    scope: 'downloads, versions',
    state: 'ready',
    credential: 'none required',
    limit: 'unmetered',
  },
  {
    kind: 'documentation',
    url: 'docs.temporal.io',
    scope: '/docs/**, depth 4',
    state: 'running',
    credential: 'none required',
    limit: '2 MB per document',
  },
  {
    kind: 'advisory',
    url: 'GitHub advisories',
    scope: 'temporalio/*',
    state: 'stale',
    credential: 'installation token · redacted',
    limit: 'shared with github',
  },
  {
    kind: 'nuget',
    url: '—',
    scope: '—',
    state: 'not_applicable',
    credential: '—',
    limit: '—',
  },
];

export const ASSOCIATIONS: readonly Association[] = [
  {
    id: '732684518001',
    source: 'npmjs.com/package/@temporalio/client',
    target: 'github.com/temporalio/sdk-typescript',
    confidence: '0.96',
    basis: 'package.json repository field resolves to the target',
    state: 'automatic',
  },
  {
    id: '732684518002',
    source: 'docs.temporal.io',
    target: 'github.com/temporalio/documentation',
    confidence: '0.71',
    basis: 'documentation footer links the repository; no reciprocal link found',
    state: 'uncertain',
  },
  {
    id: '732684518003',
    source: 'community.temporal.io',
    target: 'github.com/temporalio/temporal',
    confidence: '0.42',
    basis: 'domain similarity only',
    state: 'corrected',
    correctedBy: 'Ana Silva',
    correctedAt: '2026-08-14T16:20Z',
    note: 'Reassigned from github.com/temporalio/sdk-go. The constraint that a discussion source belongs to exactly one repository was retained.',
  },
];

/* S6 — Project lifecycle */

export interface LifecycleEvent {
  readonly at: string;
  readonly from: string | null;
  readonly to: string;
  readonly actor: string;
  readonly note: string;
}

export interface ScheduledJob {
  readonly id: string;
  readonly kind: string;
  readonly nextRun: string;
  readonly scope: string;
}

export const LIFECYCLE = {
  current: 'active',
  history: [
    {
      at: '2025-11-04T09:12:44Z',
      from: null,
      to: 'active',
      actor: 'Ana Silva',
      note: 'Project registered',
    },
    {
      at: '2026-02-18T13:40:02Z',
      from: 'active',
      to: 'paused',
      actor: 'Rafael Costa',
      note: 'Collection paused during registry migration',
    },
    {
      at: '2026-03-02T08:15:19Z',
      from: 'paused',
      to: 'active',
      actor: 'Rafael Costa',
      note: 'Collection resumed from checkpoint',
    },
  ] as readonly LifecycleEvent[],
  scheduled: [
    { id: '732684519001', kind: 'sync', nextRun: '2026-08-20T20:00:00Z', scope: '8 sources' },
    {
      id: '732684519002',
      kind: 'recalculation',
      nextRun: '2026-08-21T02:00:00Z',
      scope: '12 metrics',
    },
    {
      id: '732684519003',
      kind: 'documentation_crawl',
      nextRun: '2026-08-27T04:00:00Z',
      scope: '4 domains',
    },
  ] as readonly ScheduledJob[],
  tombstone: {
    id: '732684512931872768',
    slug: 'temporal',
    deletedAt: '2026-08-20T15:02:11Z',
    actor: 'Rafael Costa',
    retained: 'identifier, slug, deletion timestamp and actor',
    removed:
      'all repositories, sources, collected evidence, metric values, AI runs, alerts and exports',
  },
} as const;
