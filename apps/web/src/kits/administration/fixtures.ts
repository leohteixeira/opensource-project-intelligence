/**
 * Fixtures for the administration kit. Shapes follow the HTTP contract: Snowflake identifiers are
 * decimal strings, timestamps are UTC, and no value is ever a bare zero standing in for a missing
 * one. No credential, token or secret appears anywhere in this file, by design.
 */

export interface Applicant {
  readonly id: string;
  readonly name: string;
  readonly email: string;
  readonly subject: string;
  readonly requested: string;
}

export interface Member {
  readonly id: string;
  readonly name: string;
  readonly email: string;
  readonly role: 'viewer' | 'analyst' | 'admin';
  readonly state: 'active' | 'suspended';
  readonly lastSeen: string;
}

export interface ServiceAccount {
  readonly id: string;
  readonly name: string;
  readonly subject: string;
  readonly role: string;
  readonly scopes: string;
  readonly state: 'active' | 'suspended';
  readonly lastSeen: string;
}

export interface PolicyVersion {
  readonly version: string;
  readonly state: 'active' | 'superseded' | 'draft';
  readonly activated: string;
  readonly author: string;
  readonly reason: string;
}

export interface PolicyRule {
  readonly metric: string;
  readonly operator: string;
  readonly value: string;
  readonly window: string;
  readonly weight: string;
  readonly outcome: 'recommended' | 'conditional' | 'not_recommended';
}

export interface AuditEvent {
  readonly id: string;
  readonly at: string;
  readonly actor: string;
  readonly action: string;
  readonly resource: string;
  readonly result: 'succeeded' | 'coalesced';
  readonly detail: string;
}

export interface Provider {
  readonly name: string;
  readonly kind: 'model' | 'source';
  readonly state: 'ready' | 'stale' | 'failed' | 'not_applicable';
  readonly detail: string;
  readonly quota: string;
  readonly credential: string;
}

export const APPLICANTS: readonly Applicant[] = [
  {
    id: '732684514001',
    name: 'Bruno Lima',
    email: 'bruno.lima@example.com',
    subject: 'kc:1c8e…9a02',
    requested: '2026-08-20T13:41Z',
  },
  {
    id: '732684514002',
    name: 'Chen Wei',
    email: 'chen.wei@example.com',
    subject: 'kc:77b1…0e5f',
    requested: '2026-08-19T09:12Z',
  },
];

export const MEMBERS: readonly Member[] = [
  {
    id: '732684512957038592',
    name: 'Ana Silva',
    email: 'ana.silva@example.com',
    role: 'analyst',
    state: 'active',
    lastSeen: '2026-08-20T14:36Z',
  },
  {
    id: '732684514010',
    name: 'Rafael Costa',
    email: 'rafael.costa@example.com',
    role: 'admin',
    state: 'active',
    lastSeen: '2026-08-20T11:02Z',
  },
  {
    id: '732684514011',
    name: 'Marta Duarte',
    email: 'marta.duarte@example.com',
    role: 'viewer',
    state: 'active',
    lastSeen: '2026-08-18T16:20Z',
  },
  {
    id: '732684514012',
    name: 'Ivan Petrov',
    email: 'ivan.petrov@example.com',
    role: 'viewer',
    state: 'suspended',
    lastSeen: '2026-07-30T08:44Z',
  },
];

export const SERVICE_ACCOUNTS: readonly ServiceAccount[] = [
  {
    id: '732684514020',
    name: 'Portfolio exporter',
    subject: 'opi-exporter',
    role: 'viewer',
    scopes: 'projects:read, exports:write',
    state: 'active',
    lastSeen: '2026-08-20T14:00Z',
  },
  {
    id: '732684514021',
    name: 'Nightly registrar',
    subject: 'opi-registrar',
    role: 'analyst',
    scopes: 'projects:write, syncs:write',
    state: 'suspended',
    lastSeen: '2026-08-12T02:00Z',
  },
];

export const POLICY_VERSIONS: readonly PolicyVersion[] = [
  {
    version: 'v4',
    state: 'active',
    activated: '2026-07-01T09:00Z',
    author: 'Rafael Costa',
    reason: 'Quarterly governance update',
  },
  {
    version: 'v3',
    state: 'superseded',
    activated: '2026-04-01T09:00Z',
    author: 'Rafael Costa',
    reason: 'Raised concentration threshold',
  },
  {
    version: 'v5',
    state: 'draft',
    activated: '—',
    author: 'Rafael Costa',
    reason: 'Adds regression evidence rule',
  },
];

export const POLICY_RULES: readonly PolicyRule[] = [
  {
    metric: 'release_frequency',
    operator: '>=',
    value: '4',
    window: '90d',
    weight: '0.20',
    outcome: 'recommended',
  },
  {
    metric: 'median_time_to_first_response',
    operator: '<=',
    value: '48 h',
    window: '90d',
    weight: '0.15',
    outcome: 'recommended',
  },
  {
    metric: 'top_three_author_share',
    operator: '<=',
    value: '0.60',
    window: '90d',
    weight: '0.25',
    outcome: 'conditional',
  },
  {
    metric: 'maintainer_count',
    operator: '>=',
    value: '2',
    window: '90d',
    weight: '0.20',
    outcome: 'conditional',
  },
  {
    metric: 'public_advisories_unpatched',
    operator: '==',
    value: '0',
    window: '365d',
    weight: '0.20',
    outcome: 'not_recommended',
  },
];

export const AUDIT_EVENTS: readonly AuditEvent[] = [
  {
    id: '732684514101',
    at: '2026-08-20T14:38:11Z',
    actor: 'Ana Silva · analyst',
    action: 'alert_rule.created',
    resource: 'rule 732684513922121728',
    result: 'succeeded',
    detail: 'via assistant proposal 732684513917927424',
  },
  {
    id: '732684514102',
    at: '2026-08-20T14:31:02Z',
    actor: 'Ana Silva · analyst',
    action: 'sync.requested',
    resource: 'project temporal',
    result: 'coalesced',
    detail: '2 compatible requests folded into job 732684512948649984',
  },
  {
    id: '732684514103',
    at: '2026-08-20T13:02:44Z',
    actor: 'opi-exporter · service',
    action: 'export.created',
    resource: 'export 732684513844100608',
    result: 'succeeded',
    detail: 'csv · 412 rows · viewer scope',
  },
  {
    id: '732684514104',
    at: '2026-08-20T11:20:09Z',
    actor: 'Rafael Costa · admin',
    action: 'member.suspended',
    resource: 'member 732684514012',
    result: 'succeeded',
    detail: 'prior state active; new state suspended',
  },
  {
    id: '732684514105',
    at: '2026-08-19T22:12:37Z',
    actor: 'system',
    action: 'association.corrected',
    resource: 'project temporal',
    result: 'succeeded',
    detail: 'split npm:@temporalio/nexus; constraint retained',
  },
  {
    id: '732684514106',
    at: '2026-08-18T07:44:00Z',
    actor: 'deleted user 9f2a',
    action: 'project.deleted',
    resource: 'tombstone 732684513701470209',
    result: 'succeeded',
    detail: 'payload-free tombstone retained',
  },
];

export const PROVIDERS: readonly Provider[] = [
  {
    name: 'local-ollama',
    kind: 'model',
    state: 'ready',
    detail: 'qwen2.5:32b · structured output verified',
    quota: 'unbounded (local)',
    credential: 'not required',
  },
  {
    name: 'workspace-openai',
    kind: 'model',
    state: 'stale',
    detail: 'rate limited; retry after 1830s',
    quota: '82% of monthly budget',
    credential: 'present · redacted',
  },
  {
    name: 'github',
    kind: 'source',
    state: 'ready',
    detail: '5 000 req/h app token',
    quota: '31% used',
    credential: 'present · redacted',
  },
  {
    name: 'npm',
    kind: 'source',
    state: 'failed',
    detail: '429 from registry; resumes at 15:05Z',
    quota: '100% used',
    credential: 'not required',
  },
  {
    name: 'nuget',
    kind: 'source',
    state: 'not_applicable',
    detail: 'no NuGet source configured in this workspace',
    quota: '—',
    credential: '—',
  },
];
