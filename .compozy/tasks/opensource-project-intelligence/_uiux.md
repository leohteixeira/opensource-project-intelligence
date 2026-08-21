# UI/UX Change Map: Open Source Project Intelligence

Every browser surface the product adds, its required states, and its production mapping. Reference
artboards land under `docs/design/opendesign/opensource-project-intelligence/` and become visual
contracts for implementation tasks.

Companions: `_spec.md` Part I is the behavior authority, `_user_stories.md` supplies acceptance and
edge states, and `_dx.md` freezes the non-visual browser and HTTP contract.

## Current UI baseline

The current application is one placeholder `<main>` with a heading and backend URL at
`apps/web/src/App.tsx:3`; it has no navigation, routing, forms, authentication, data fetching,
charts, tables, localization, responsive system, accessibility tests, or component inventory. Its
only web configuration is the API base URL at `apps/web/src/config.ts:5`, and React mounts directly
at `apps/web/src/main.tsx:6`.

The repository has no `DESIGN.md`, `COPY.md`, `packages/ui`, `@compozy/ui`, or token inventory.
Because this is an independent repository and cross-repository source sharing is forbidden, the
design pass must establish repository-local `apps/web/DESIGN.md`, `apps/web/COPY.md`, and
`apps/web/src/styles/tokens.css` before producing artboards. Generic primitives belong in
`apps/web/src/ui/`; domain composites belong in `apps/web/src/systems/<domain>/`. Artboard CSS is a
visual contract and is never imported into production.

## Design constraints

- The interface is mobile-first and every Viewer workflow remains complete at 320 CSS pixels.
- Meet WCAG 2.2 AA: keyboard access, visible focus, semantic landmarks, labeled controls, skip link,
  44-by-44 CSS-pixel touch targets, text resizing, reduced motion, and non-color status cues.
- Use flat depth. A page may contain sections and one level of disclosure; avoid cards nested in
  cards and horizontal page scrolling.
- Use compact density for evidence tables, but provide a row-detail view on narrow screens instead
  of shrinking text or hiding columns without access.
- Health dimensions never collapse into one traffic-light verdict. The overall score is always
  visible as a secondary summary when the active definition's evidence requirements are met; it is
  labeled with its version and absent when it cannot be calculated.
- Green communicates a satisfied deterministic condition, amber a condition requiring attention, red
  a verified failure or `not_recommended` result, blue an informational/running state, and gray
  unknown/not-applicable. Every signal also uses text and an icon.
- `pending`, `running`, `stale`, `unknown`, `not_applicable`, `insufficient_data`, `failed`, and
  `ready` remain visually and textually distinct.
- Fixed UI copy follows the selected English or Brazilian Portuguese locale. Evidence remains in its
  original language; generated translations are labeled. Dates render in the member's timezone and
  expose UTC on demand.
- Headings use concise nouns or actions, with no subtitle/helper paragraph beneath a heading. Empty
  states carry the next permitted action in the body.
- Permission-restricted controls are absent. Temporarily unavailable actions render only when the
  action exists and include the reason and recovery path.
- Destructive confirmations name the resource, disclose irreversible effects, and require exact
  typed confirmation. Assistant confirmations show one atomic action only.
- All list filters, selected tabs, comparison Projects, windows, and search terms are
  URL-addressable so refresh, browser history, and shared links preserve context.

## Information architecture

All browser routes are localized. `/` redirects to the saved locale or the best supported browser
locale. Changing language maps the current resource to its equivalent path and preserves query
state. English and Portuguese pages emit canonical and `hreflang` links. API paths, JSON fields,
metric names, and error codes remain stable in English.

### Anonymous shell

- `/en` · `/pt-br` — public catalog landing page.
- `/en/catalog/:projectId` · `/pt-br/catalogo/:projectId` — public Project teaser.
- `/en/access/pending` · `/pt-br/acesso/pendente` — authenticated Applicant status.

### Approved-member shell

Primary navigation: **Portfolio**, **Projects**, **Compare**, **Radar**, **Alerts**. Its Portuguese
paths are `/pt-br/portfolio`, `/pt-br/projetos`, `/pt-br/comparar`, `/pt-br/radar`, and
`/pt-br/alertas`; English uses `/en/portfolio`, `/en/projects`, `/en/compare`, `/en/radar`, and
`/en/alerts`. Secondary account navigation: **Preferences**, **Sign out**. Analyst-only actions
appear contextually, not as a separate administration area. Admin navigation adds **Members**,
**Policies**, **Audit**, and **Operations**.

On desktop the primary navigation is a persistent left rail. On mobile it is a five-item bottom bar;
Admin pages, preferences, archive, and less-frequent destinations live in the top-right menu. The
Project detail uses a scrollable tab list with an adjacent overflow menu, not a second nested rail.

Project tabs use
`/en/projects/:id/{overview,health,contributors,adoption-security,trends,topics, releases,knowledge,sources-jobs}`
or
`/pt-br/projetos/:id/{visao-geral,saude,colaboradores, adocao-seguranca,tendencias,topicos,lancamentos,conhecimento,fontes-tarefas}`.
Their labels are **Overview**, **Health**, **Contributors**, **Adoption & Security**, **Trends**,
**Topics**, **Releases**, **Knowledge**, and **Sources & Jobs** in English. Project identity editing
and lifecycle actions live in the Project header menu. The assistant opens as a full-height side
panel on desktop and a dedicated full-screen localized route on mobile.

## Surface map

| #   | Surface                                          | Kind   | Core change                                                                                          | Stories                                |
| --- | ------------------------------------------------ | ------ | ---------------------------------------------------------------------------------------------------- | -------------------------------------- |
| S1  | Application shell and routing                    | modify | Replace the static foundation with anonymous/member shells and role-aware navigation                 | US-001–US-005, US-031                  |
| S2  | Public catalog and teaser                        | new    | Expose only approved public Project identity and source links                                        | US-001                                 |
| S3  | Sign-in, pending access, and account             | new    | Bridge shared Keycloak identity to local membership and preferences                                  | US-002–US-004                          |
| S4  | Portfolio overview                               | new    | Rank attention without hiding freshness, coverage, or panel failure                                  | US-005                                 |
| S5  | Project catalog, registration, and identity      | new    | Register from URL and curate Project identity                                                        | US-006, US-007                         |
| S6  | Project lifecycle                                | new    | Pause, archive, restore, and permanently purge with explicit effects                                 | US-009                                 |
| S7  | Repositories, sources, and associations          | new    | Curate typed repositories and correct explainable cross-source links                                 | US-007, US-008, US-012                 |
| S8  | Jobs, synchronization, and history               | new    | Show durable progress, coverage, quota, failure, retry, and cancellation                             | US-010, US-011                         |
| S9  | Metrics and health                               | new    | Present definitions, evidence, seven dimensions, and the secondary overall score whenever calculable | US-013                                 |
| S10 | Contributor intelligence                         | new    | Show activity, retention, maintainers, concentration, and identity coverage                          | US-014                                 |
| S11 | Adoption and security                            | new    | Keep registry signals contextual and public security evidence qualified                              | US-015                                 |
| S12 | Comparison workspace                             | new    | Compare two to five Projects over one exact window and cutoff                                        | US-016                                 |
| S13 | Trends and early warnings                        | new    | Visually separate observed results from forecasts                                                    | US-017                                 |
| S14 | Recommendation and policy governance             | new    | Explain four-state evaluation and let Admins version policies                                        | US-018, US-019                         |
| S15 | Technology radar                                 | new    | Show policy placement, attributed override, and review date                                          | US-020                                 |
| S16 | Issue/discussion topics                          | new    | Explore evidence-bearing topics and submit corrections                                               | US-021                                 |
| S17 | Release intelligence                             | new    | Present categorized, cited release claims without invented completeness                              | US-022                                 |
| S18 | Documentation knowledge                          | new    | Search explicit snapshots and cite original evidence                                                 | US-023                                 |
| S19 | Natural-language analysis and HITL assistant     | new    | Ask cited questions and confirm one safe typed action                                                | US-024, US-025                         |
| S20 | AI run governance                                | new    | Compare immutable versions, feedback, rerun, and selected result                                     | US-026, US-030                         |
| S21 | Alerts                                           | new    | Separate per-user read state from shared lifecycle                                                   | US-027                                 |
| S22 | Exports                                          | new    | Configure and retrieve asynchronous CSV/evidence packages                                            | US-028                                 |
| S23 | Members, service accounts, audit, and operations | new    | Govern access and inspect attributable actions/redacted runtime status                               | US-003, US-012, US-029, US-030, US-032 |

### S1. Application shell and routing

- **Today**: `apps/web/src/App.tsx:3` renders one static `<main>`; `apps/web/src/main.tsx:6` mounts
  it without providers or router.
- **Change**: introduce the anonymous, Applicant, approved-member, and Admin shells; persistent
  URLs; skip navigation; locale/timezone application; session-expiry recovery; and responsive
  navigation.
- **States to design**: initial session check, anonymous, pending Applicant, approved
  Viewer/Analyst/ Admin, suspended, offline, session expired, route not found, unauthorized deep
  link, narrow/medium/ wide viewport, keyboard focus order, English, Portuguese, and 200% zoom
  (US-001 EC-4/8; US-002 AC-2/3 and EC-6/9; US-031 all ACs/ECs).
- **Artboard**: `opensource-project-intelligence-app-shell.html`.

### S2. Public catalog and teaser

- **Change**: searchable, cursor-paginated Project tiles and a teaser detail with only name,
  description, source links, and sign-in call to action.
- **States to design**: populated, empty, searching, no matches, cursor-backed numbered previous/
  current/next pages, retained stale page after failure, archived-between-pages, protected deep
  link, and 100-times catalog scale (US-001 AC-1–3, EC-1–10).
- **Artboard**: `opensource-project-intelligence-public-catalog.html`.

### S3. Sign-in, pending access, and account

- **Change**: sign-in handoff, callback recovery, approval status, preferences, sign-out, and exact
  account-deletion confirmation.
- **States to design**: redirecting, callback failure, invalid flow, pending, rejected, suspended,
  newly approved, unsupported locale/timezone, save conflict, delete confirmation, delete running,
  deleted, and last-Admin prevention (US-002, US-003 EC-9, US-004).
- **Artboard**: `opensource-project-intelligence-access-account.html`.

### S4. Portfolio overview

- **Change**: attention queue and independent panels for health, recommendations, alerts, trends,
  warnings, releases, freshness, and failures, each linked to evidence.
- **States to design**: populated, no Projects by role, filtered empty, mixed fresh/stale/unknown,
  partial panel error with retained valid panels, new-snapshot refresh, archived removal, and large
  portfolio aggregation (US-005 AC-1–3, EC-1–10).
- **Artboard**: `opensource-project-intelligence-portfolio.html`.

### S5. Project catalog, registration, and identity

- **Change**: protected Project list, repository-URL registration, inferred identity review, Project
  header, and editable public identity.
- **States to design**: empty/list/search/pagination, valid preview, malformed/unsupported/private/
  hostile URL, quota reached, duplicate active/archived Project, concurrent duplicate resolution,
  accepted initial sync, recoverable collection failure, stale edit, and high concurrent backfill
  (US-006, US-007 EC-1/5/8/9).
- **Artboard**: `opensource-project-intelligence-project-registration.html`.

### S6. Project lifecycle

- **Change**: Admin-only transition menu with effect summary and permanent deletion ceremony.
- **States to design**: active, pause confirmation/running/paused, archive confirmation/archived,
  restore target choice/running, conflicting transition, scheduled-work cancellation, exact typed
  deletion, purge running, tombstone outcome, repeated deletion, and forbidden Analyst/Viewer
  (US-009 AC-1–3, EC-1–10).
- **Artboard**: `opensource-project-intelligence-project-lifecycle.html`.

### S7. Repositories, sources, and associations

- **Change**: repository role table, exactly-one-primary control, source inventory and crawl scope,
  redacted capability status, and association evidence/correction workspace.
- **States to design**: single/many repositories, replace primary, duplicate URL, source limit,
  unsupported role/kind, stale derived results, automatic link by confidence, uncertain link,
  corrected/split/reassigned/constraint retained, concurrent correction, source unavailable,
  credential absent/redacted, hostile URL, pagination, and archived read-only (US-007, US-008,
  US-012 AC-1–3 and EC-1–10).
- **Artboard**: `opensource-project-intelligence-sources-associations.html`.

### S8. Jobs, synchronization, and history

- **Change**: source freshness table, sync request scope, history request, durable Job list/detail,
  factual progress, checkpoints, coalescing, retry, and permitted cancellation.
- **States to design**: SSE connected/reconnecting/resumed and polling fallback; queued, running
  known-total, running unknown-total, succeeded, partial source failure, failed retryable/permanent,
  cancelled, coalesced request, stale source, rate-limited with reset, quota rejected, worker
  restart/resumed checkpoint, requested-vs-actual coverage, older open item, no successful
  collection, concurrent recalculation, and high-volume pagination (US-010, US-011, US-012).
- **Artboard**: `opensource-project-intelligence-jobs-history.html`.

### S9. Metrics and health

- **Change**: health-dimension overview and metric detail drawer with value/status, formula, unit,
  window, cutoff, version, applicability, repositories, coverage, and evidence.
- **States to design**: all seven dimensions, secondary overall score visible whenever calculable
  and absent when requirements fail, available zero, unknown, not applicable, insufficient data,
  stale, mixed versions prevented, formula version comparison, custom window out of coverage,
  evidence page failure, and mobile metric table (US-013 AC-1–3, EC-1–10).
- **Artboard**: `opensource-project-intelligence-metrics-health.html`.

### S10. Contributor intelligence

- **Change**: activity and retention cohorts, maintainers, top-one/top-three concentration, identity
  association evidence, and unresolved coverage.
- **States to design**: no contributors, one contributor, unresolved identities, corrected identity,
  conflicting merge, concentration warning, unknown retention, bot/service-account treatment,
  paginated contributors, and translated labels without translating identities (US-014 AC-1–3,
  EC-1–10).
- **Artboard**: `opensource-project-intelligence-contributors.html`.

### S11. Adoption and security

- **Change**: separate per-registry signal panels and a public-security-evidence timeline with
  explicit non-scanner limitation.
- **States to design**: one/many/no registries, source-specific units, comparable normalization,
  incomparable signals, unavailable source, no public advisories as unknown evidence rather than
  safe, security release, conflicting advisory, stale cutoff, high-volume timeline, and mobile
  disclosure (US-015 AC-1–3, EC-1–10).
- **Artboard**: `opensource-project-intelligence-adoption-security.html`.

### S12. Comparison workspace

- **Change**: two-to-five Project selector, window/cutoff picker, comparable metric matrix, evidence
  drill-down, and saved shareable URL.
- **States to design**: two/three/five Projects, fewer-than-two/more-than-five, duplicate Project,
  preset/custom window, insufficient common coverage, numeric zero vs missing/not-applicable/
  incomparable, version mismatch rejection, one Project archived during selection, partial evidence
  failure, wide desktop matrix, and narrow row-detail mode (US-016 AC-1–3, EC-1–10).
- **Artboard**: `opensource-project-intelligence-comparison.html`.

### S13. Trends and early warnings

- **Change**: separate Observed and Early warnings views, each with evidence and method details;
  forecasts additionally show horizon, confidence, calibration/error, and minimum history.
- **States to design**: observed increase/decrease/stable, early warning, no signal, insufficient
  history, poor coverage, stale model, corrected evidence, AI explanation unavailable, overlapping
  signals, dense timeline, and screen-reader chart alternative (US-017 AC-1–3, EC-1–10).
- **Artboard**: `opensource-project-intelligence-trends-warnings.html`.

### S14. Recommendation and policy governance

- **Change**: four-state recommendation explanation for all members and Admin policy list/editor,
  immutable version history, validation, activation, and evaluation impact preview.
- **States to design**: recommended/conditional/not recommended/insufficient data, conditions,
  missing inputs, policy unavailable, stale evaluation, draft/superseded/active versions, empty rule
  tree, unknown metric/operator, unsaved changes, conflicting activation, prior evaluations
  retained, last policy protection, and large rule tree keyboard editing (US-018, US-019).
- **Artboard**: `opensource-project-intelligence-recommendations-policies.html`.

### S15. Technology radar

- **Change**: accessible radar list as primary data representation, optional spatial plot, policy
  suggestion versus effective placement, attributed override, reason, owner, and review date.
- **States to design**: adopt/trial/assess/hold, insufficient-data unplaced, policy placement,
  active/expired override, due review, removed override, changed policy suggestion, project
  archived, conflicting override, empty radar, 100-times scale, keyboard/list access, and
  print/export (US-020 AC-1–3, EC-1–10).
- **Artboard**: `opensource-project-intelligence-radar.html`.

### S16. Issue and discussion topics

- **Change**: known/emerging topic list, prevalence/change, evidence samples, confidence/version,
  and Analyst correction flow.
- **States to design**: known/emerging, no topics, low confidence, sparse discussion coverage,
  renamed/merged/split/reassigned, correction conflict, reanalysis stale/running/failed, repeated
  correction, large taxonomy, evidence in original language, and mobile detail (US-021 AC-1–3,
  EC-1–10).
- **Artboard**: `opensource-project-intelligence-topics.html`.

### S17. Release intelligence

- **Change**: release timeline and detail with feature/breaking/deprecation/security/performance/DX
  claims, each cited independently.
- **States to design**: analyzed/pending/stale/failed release, no changelog, sparse/conflicting
  evidence, no claims, prerelease, duplicate tag, corrected source, AI provider unavailable with raw
  release retained, long release, large history, and original/generated translation (US-022 AC-1–3,
  EC-1–10).
- **Artboard**: `opensource-project-intelligence-releases.html`.

### S18. Documentation knowledge

- **Change**: explicit-domain/source management entry, search box, snapshot results, citations, and
  original-document viewer.
- **States to design**: no configured docs, crawl queued/running/partial/failed, robots excluded,
  unsafe redirect, depth/size/type limit, empty query/no results, exact/cited result, stale
  snapshot, conflicting snapshots, AI/RAG unavailable with lexical results retained, pagination, and
  bilingual answer/evidence distinction (US-023 AC-1–3, EC-1–10).
- **Artboard**: `opensource-project-intelligence-knowledge.html`.

### S19. Natural-language analysis and HITL assistant

- **Change**: query composer with scope/window, structured cited response, clarifying question, and
  assistant proposal/confirmation/execution receipt.
- **States to design**: clear query, ambiguous scope clarification, insufficient data, analysis
  queued/running/succeeded/failed/stale, unsupported language, cited response, provider unavailable,
  one safe proposal, exact inputs/effect/quota/expiry, prohibited action, expired proposal,
  permission changed, resource changed, confirm once, duplicate confirm, execution failure, and
  mobile full-screen assistant (US-024, US-025).
- **Artboard**: `opensource-project-intelligence-assistant.html`.

### S20. AI run governance

- **Change**: immutable run metadata/evidence, side-by-side version comparison, selected-version
  control, feedback form, rerun request, usage/cost, and provider-degradation status.
- **States to design**: queued/running/succeeded/failed/stale, no successful version, multiple
  versions, selected older version, invalid selection, feedback submitted/repeated, rerun coalesced,
  provider disabled/unhealthy/quota-limited, structured-output invalid, partial evidence, long
  output, and Admin aggregate versus Analyst per-run visibility (US-026, US-030).
- **Artboard**: `opensource-project-intelligence-ai-runs.html`.

### S21. Alerts

- **Change**: alert inbox, unread count, filters, rule management, per-user read state, shared
  acknowledge/resolve/dismiss transitions, deduplication grouping, and evidence links.
- **States to design**: unread/read, open/acknowledged/resolved/dismissed, deduplicated recurrence,
  invalid rule, no alerts, stale evidence, transition conflict, source recovery, large inbox, Viewer
  read-only lifecycle, Analyst rule editing, and mobile filters (US-027 AC-1–3, EC-1–10).
- **Artboard**: `opensource-project-intelligence-alerts.html`.

### S22. Exports

- **Change**: choose resource/filter/window and CSV or evidence JSON, track Job, then retrieve an
  expiring artifact with visible cutoff/version/locale.
- **States to design**: configuration, validation failure,
  queued/running/succeeded/failed/cancelled, snapshot consistency during concurrent updates,
  zero-row valid export, size quota, expired download after 24 hours, permission revoked, retry,
  duplicate request/coalescing, and mobile download handoff (US-028 AC-1–3, EC-1–10).
- **Artboard**: `opensource-project-intelligence-exports.html`.

### S23. Members, service accounts, audit, and operations

- **Change**: Admin applicant queue/member table, service-account binding with Viewer/Analyst role
  and scope subset, approval/role/suspension dialogs, immutable audit filters/detail, and redacted
  provider/quota/usage/health dashboard.
- **States to design**: no/many applicants, approve Viewer/Analyst/Admin, reject, stale membership,
  last Admin, suspended member or service account, absent Keycloak binding, invalid/overbroad
  service scope, bearer activity attributed to the service identity, audit empty/filter/no
  results/pagination/tombstone/pseudonymous actor, provider healthy/degraded/unavailable/not
  configured/rate-limited, credentials present but always redacted, aggregate model usage/cost,
  deterministic operation during AI outage, and partial status failure (US-003, US-012, US-029,
  US-030, US-032).
- **Artboard**: `opensource-project-intelligence-administration.html`.

## Component plan

### Rules

- Establish tokens and primitives once, then compose domain systems. Do not create a bespoke button,
  dialog, table, field, status chip, tooltip, or pagination control inside a domain system.
- Keep API data status in domain composites. Generic primitives receive explicit labels and semantic
  variants but never infer `healthy`, `recommended`, or `safe` from colors or numbers.
- Charts always have an equivalent table or textual summary in the same view.
- Use native HTML semantics before custom ARIA. Modal dialogs trap focus only while open and return
  focus to the invoking control. Toasts never carry the only record of an outcome.
- Virtualized data views must preserve keyboard and screen-reader access; use server pagination when
  the full collection is not loaded.

### New repository-local UI primitives

There is no existing primitive inventory to reuse. The design pass must define the smallest shared
set under `apps/web/src/ui/`: `AppShell`, `Button`, `IconButton`, `Link`, `TextField`, `TextArea`,
`Select`, `Checkbox`, `RadioGroup`, `DateRangeField`, `FormField`, `Banner`, `StatusBadge`,
`Progress`, `Tabs`, `Menu`, `Dialog`, `Drawer`, `Table`, `Pagination`, `FilterBar`,
`DefinitionList`, `EmptyState`, `Skeleton`, `Tooltip`, and `VisuallyHidden`. Each is justified
because it appears on at least three surfaces or encodes a cross-product accessibility/state rule.

Project-specific visualizations (`HealthDimensions`, `ComparisonMatrix`, `TrendChart`, `RadarPlot`)
are domain components, not generic primitives.

### New domain components

| System path            | Components                                                                                                                                         | Surfaces        |
| ---------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | --------------- |
| `systems/access`       | `SessionGate`, `PendingAccess`, `MemberForm`, `AccountDeletion`                                                                                    | S1, S3, S23     |
| `systems/catalog`      | `PublicProjectList`, `ProjectList`, `ProjectSummary`                                                                                               | S2, S4, S5      |
| `systems/projects`     | `ProjectHeader`, `ProjectForm`, `LifecycleDialog`, `RepositoryTable`, `SourceTable`, `AssociationReview`                                           | S5–S8           |
| `systems/jobs`         | `JobStatus`, `JobProgress`, `JobTimeline`, `CoverageSummary`, `QuotaStatus`                                                                        | S5, S8, S18–S22 |
| `systems/intelligence` | `EvidenceLink`, `CoverageDisclosure`, `MetricValue`, `MetricDetail`, `HealthDimensions`, `ContributorTable`, `AdoptionSignals`, `SecurityTimeline` | S4, S9–S11, S14 |
| `systems/comparison`   | `ProjectPicker`, `WindowPicker`, `ComparisonMatrix`                                                                                                | S12             |
| `systems/trends`       | `TrendChart`, `ObservedTrendList`, `EarlyWarningList`, `ForecastDisclosure`                                                                        | S4, S13         |
| `systems/policies`     | `Recommendation`, `PolicyRuleEditor`, `PolicyVersionHistory`, `EvaluationPreview`                                                                  | S4, S14         |
| `systems/radar`        | `RadarList`, `RadarPlot`, `RadarOverride`                                                                                                          | S15             |
| `systems/topics`       | `TopicList`, `TopicDetail`, `TopicCorrection`                                                                                                      | S16             |
| `systems/releases`     | `ReleaseTimeline`, `ReleaseClaims`                                                                                                                 | S4, S17         |
| `systems/knowledge`    | `KnowledgeSearch`, `SnapshotResult`, `EvidenceViewer`                                                                                              | S18, S19        |
| `systems/assistant`    | `QueryComposer`, `AnalysisResult`, `Clarification`, `ActionProposal`, `ActionReceipt`                                                              | S19             |
| `systems/analysis`     | `RunStatus`, `RunMetadata`, `RunComparison`, `RunFeedback`                                                                                         | S16–S20         |
| `systems/alerts`       | `AlertInbox`, `AlertRuleForm`, `AlertLifecycle`                                                                                                    | S4, S21         |
| `systems/exports`      | `ExportForm`, `ExportStatus`                                                                                                                       | S22             |
| `systems/admin`        | `ApplicantQueue`, `MemberTable`, `ServiceAccountTable`, `ServiceAccountForm`, `AuditTable`, `AuditDetail`, `OperationsStatus`                      | S23             |

### Signal and state mapping

Exact token names are established in `apps/web/src/styles/tokens.css`; the semantic contract below
is frozen and token spelling is not.

| Meaning                          | Visual contract                                        | Text contract                                      |
| -------------------------------- | ------------------------------------------------------ | -------------------------------------------------- |
| Available/ready/succeeded        | check glyph, neutral or positive signal                | Exact state plus cutoff when relevant              |
| Recommended                      | check-circle glyph, positive signal                    | `Recommended` / `Recomendado`                      |
| Conditional/attention            | triangle glyph, amber signal                           | Condition count and first required action          |
| Not recommended/verified failure | stop glyph, red signal                                 | Exact policy/failure reason; never `unhealthy`     |
| Insufficient data                | hollow-circle glyph, gray signal                       | Missing evidence and required coverage             |
| Unknown                          | question glyph, gray signal                            | `Unknown`; never `0` or blank                      |
| Not applicable                   | minus glyph, gray signal                               | `Not applicable` and applicability reason          |
| Queued/pending                   | clock glyph, blue-gray signal                          | Position only when factual; otherwise `Queued`     |
| Running                          | animated only when reduced motion permits; blue signal | Completed/total or `Total unknown`                 |
| Stale                            | history glyph, amber outline                           | Last valid cutoff and why stale                    |
| Failed                           | error glyph, red signal                                | Safe reason and retry/recovery action              |
| Paused                           | pause glyph, neutral signal                            | Collection stopped; retained data remains readable |
| Archived                         | archive glyph, neutral signal                          | Read-only and excluded from active views           |
| Forecast                         | dashed-line/forward glyph, violet informational signal | Horizon, confidence, calibration, model version    |
| Observed trend                   | solid-line/history glyph, blue informational signal    | Observation and baseline windows, method version   |
| AI-generated                     | sparkle glyph used sparingly                           | Provider/model, prompt version, cutoff, evidence   |
| Generated translation            | language glyph                                         | Source language and `Generated translation` label  |
