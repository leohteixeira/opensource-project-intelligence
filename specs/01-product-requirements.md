# Open Source Project Intelligence Specification

## Part I — Product

## Overview

Open Source Project Intelligence is a self-hosted, fullstack intelligence platform for evaluating
public open source technologies. It transforms repository activity, issues, pull requests, releases,
contributors, discussions, documentation, package activity, advisories, changelogs, feeds, and
project websites into traceable evidence for technical adoption and portfolio governance.

The primary user is a technical lead, staff engineer, architect, or other Analyst deciding whether a
technology is active, sustainable, maintainable, stable, secure enough for its context, and gaining
or losing adoption. Viewers consume the evidence and decisions; Admins govern access and policies;
VPS Operators manage the shared deployment boundaries.

The product is valuable because no single activity counter answers an adoption question. It combines
deterministic time-windowed metrics with source-aware qualitative intelligence, makes missing data
visible, separates observed facts from forecasts, and preserves evidence behind every score,
recommendation, alert, and AI-generated explanation. It is an intelligence layer over multiple
public sources, not a repackaged GitHub dashboard or a generic chatbot.

This specification covers the complete product vision described in the source proposal. The original
MVP, V2, and V3 labels are implementation-history context only; all selected capabilities form one
product acceptance boundary. Dependency-ordered internal checkpoints may guide delivery, but none is
a separately accepted product release.

## Goals

- Let a user discover the public catalog, register through the workspace-shared identity provider,
  obtain approved access, and use one bilingual mobile-first workspace safely.
- Let Analysts register a technology from a public repository URL, then expand it into an explicit
  multi-repository, multi-source Project without confusing Project identity with vendor identity.
- Collect public source history incrementally and repeatedly while showing actual coverage,
  freshness, progress, quotas, failures, and resumability.
- Preserve canonical facts, selected raw evidence, source provenance, temporal snapshots, and
  calculation versions so intelligence can be recalculated and audited.
- Provide a closed, versioned metric catalog covering Activity, Community, Maintenance,
  Concentration, Stability, Security, and Adoption with explicit formulas and time windows.
- Present health as independently inspectable dimensions; allow only a secondary transparent overall
  score and never reduce a Project to an unexplained healthy/unhealthy label.
- Compare two to five Projects over exactly the same window without converting missing,
  non-applicable, or incomparable evidence into zero.
- Turn issues and discussions into correctable topic intelligence, releases into evidence-backed
  change intelligence, and explicitly linked documentation into searchable project knowledge.
- Separate reproducible observed trends from versioned predictive early warnings and disclose
  evidence, uncertainty, history requirements, and data cutoff for both.
- Apply deterministic, versioned workspace policies to produce `recommended`, `conditional`,
  `not_recommended`, or `insufficient_data` adoption guidance.
- Derive a technology radar from policy results while preserving attributed human overrides and
  review dates.
- Allow natural-language analysis and a bounded HITL assistant without permitting free-form access
  to credentials, administration, governance, or destructive actions.
- Preserve every AI analysis as an immutable evidence-bearing run and degrade AI features without
  degrading ingestion or deterministic intelligence.
- Notify the workspace through deduplicated in-app alerts and support auditable CSV and JSON
  exports.
- Make access, governance changes, corrections, operator-sensitive actions, and HITL approvals
  attributable without storing secrets in product output.

## User Stories

- `US-001`–`US-004` — public discovery, shared-identity sign-in, local access approval, roles,
  preferences, and account removal.
- `US-005`–`US-009` — portfolio overview, Project registration, multi-repository curation,
  cross-source identity resolution, and lifecycle.
- `US-010`–`US-012` — scheduled and manual collection, history/freshness, and public-data source
  operation.
- `US-013`–`US-017` — metrics, health, contributor sustainability, adoption/security evidence,
  comparison, trends, and forecasts.
- `US-018`–`US-020` — adoption recommendations, policy governance, and technology radar.
- `US-021`–`US-026` — issue/discussion topics, release intelligence, documentation knowledge,
  natural-language analytics, HITL actions, and AI run governance.
- `US-027`–`US-031` — in-app alerts, exports, audit, model-provider degradation, localization,
  accessibility, and mobile operation.
- `US-032` — approved Keycloak service identities using the stable HTTP API under local role, scope,
  quota, and audit enforcement.

[Full user stories](_user_stories.md)

## Core Features

### Public discovery and controlled membership

Anonymous Visitors can search a deliberately limited catalog containing Project names, public
descriptions, and public source links. Full metrics, analyses, comparisons, radar, alert history,
exports, and user information never appear on anonymous surfaces.

Registration and authentication come from the Keycloak deployment shared across the three workspace
products. Authentication establishes external identity but not product access. A newly authenticated
person becomes an Applicant until a local Admin approves membership and assigns one fixed role. This
product owns membership, permissions, suspension, and audit; it does not own passwords, identity
verification, recovery, Keycloak deployment, or global realm governance.

Approved Keycloak service identities may use the HTTP API through local Viewer or Analyst roles and
least-privilege scopes. They never receive Admin, never impersonate a person, and remain subject to
local suspension, quota, and audit even while an external token remains valid.

### Portfolio overview

Approved members land on a portfolio overview that identifies Projects requiring attention. It
summarizes health dimensions, policy recommendations, current alerts, observed trends, early
warnings, important releases, data freshness, source coverage, and synchronization failures. Every
summary leads to its evidence-bearing detail. Partial panel failures never erase deterministic
information that remains valid.

### Project catalog and lifecycle

A Project represents a technology and owns one or more typed Repositories. Registration from a
supported public repository URL creates a Project plus its primary Repository; the inferred name and
metadata remain reviewable. Additional repositories identify core, documentation, examples, SDK, or
other product-defined roles. Project-level results disclose which repositories and sources they
aggregate.

Projects move through `active`, `paused`, and `archived` states before optional permanent deletion.
Active Projects synchronize automatically. Paused Projects retain readable intelligence without
scheduled collection. Archived Projects are read-only and excluded from active views. Permanent
deletion is Admin-only, cancels work, purges Project-owned raw and derived data, and leaves only a
payload-free audit tombstone.

### Cross-source identity and coverage

The product automatically associates public repositories, packages, documentation, websites,
changelogs, feeds, discussions, and advisories with canonical Projects using explainable evidence
and a versioned decision. Analysts can split or reassign associations. Corrections persist as
constraints so rejected links do not silently reappear, and affected downstream intelligence becomes
stale until recalculated.

Contributor identities combine only through a publicly verifiable link or Analyst confirmation; name
similarity and private email data are insufficient. Resolution coverage is disclosed alongside
contributor and concentration metrics.

### Public-source collection

Built-in product sources cover GitHub, GitLab, Gitea, public or locally reachable public Git
repositories, npm, NuGet, PyPI, public project documentation and websites, changelogs, RSS, and
provider-supported public discussions and advisories. Other package registries may participate
through the product's source extension boundary but are not mandatory built-in adapters.

Collection is scheduled and can be requested manually by an Analyst. Compatible duplicate requests
coalesce. Every source shows progress, last attempt, last success, next planned run, actual
historical coverage, stale state, quota state, and failure reason. Interrupted work resumes from
durable progress without duplicating canonical facts.

The default initial history target is 180 days. Admin/operator limits may allow Analysts to request
longer history. Older open issues and pull requests remain eligible even when they precede the
target. Normalized data, daily snapshots, selected raw evidence, analyses, and provenance remain
until their Project is permanently deleted. Every intelligence result reports actual coverage rather
than only the requested range.

Only public content is eligible. VPS Operators may supply least-privilege server-side credentials to
authenticate official requests and obtain usable quotas, but a credential never authorizes private
content ingestion. Users never provide or view source tokens.

### Deterministic metrics and health

The product owns a closed and versioned metric catalog. It includes the seven initial metrics from
the proposal and the additional metrics required by Activity, Community, Maintenance, Concentration,
Stability, Security, and Adoption. Every result discloses definition version, formula, unit,
observation window, cutoff, source coverage, aggregation boundary, applicability, and missing-data
treatment. Users configure adoption policies, not arbitrary metric formulas.

The seven required initial metrics are:

1. releases in the previous 90 days;
2. active contributors in the previous 30 days;
3. issues opened and closed in the previous 30 days;
4. median time to first issue response;
5. median pull-request merge time;
6. backlog change in the previous 30 days; and
7. the top three commit authors' share in the previous 90 days.

Additional required catalog coverage includes commit velocity, time since last release, active
branches, new-contributor rate and retention, discussion activity, issue resolution time, backlog
growth, top-contributor share, maintainer count and new-maintainer rate, breaking-change and major
release frequency, regression-related issues, public security evidence, and registry adoption
signals. Exact formulas and evidence eligibility belong to Part II, but no listed dimension may be
represented by an undocumented proxy.

Health presents all seven dimensions independently. When the active overall-score definition has
sufficient evidence, its result is always visible as a secondary summary. It discloses version,
weights, factors, window, evidence, and missing-data handling, and it does not publish when declared
evidence requirements are unmet. Stars or popularity never substitute for health.

### Historical comparison, trends, and forecasts

Users select preset windows of 30, 90, 180, or 365 days or a custom interval bounded by available
coverage. Comparisons include two to five Projects and use one identical window and cutoff. Only
source and unit-compatible indicators are normalized; `unknown`, `not_applicable`, and
`insufficient_data` remain distinct from numeric zero.

Observed trends use reproducible rules or statistical calculations over explicit observation and
baseline windows. Predictive early warnings are a separate product class and expose forecast
horizon, confidence, calibration or known error, evidence, minimum history, coverage, and model
version. The product never labels a forecast as observed fact. An LLM may explain either result but
cannot create, suppress, or alter the signal.

### Contributor, adoption, and security intelligence

Contributor intelligence covers activity, new participation, retention, maintainers, and top-one and
top-three concentration. It preserves provider identity evidence and makes unresolved identity
coverage visible.

Adoption intelligence shows registry-specific raw indicators and changes over explicit windows.
Normalization occurs only within a demonstrably comparable population; there is no universal
cross-registry adoption score. A policy or health dimension using adoption evidence must disclose
the exact transformation and missing sources.

Security intelligence uses public advisories, security releases, changelogs, issues, and metadata
published by configured code hosts and package registries. It does not scan source code, generate
SBOMs, inspect an organization's dependencies, or claim that absence of public advisories means
absence of vulnerabilities.

### Issue and discussion intelligence

The product combines a versioned known taxonomy with emerging topics discovered in issues and
provider-supported discussions over explicit windows. Topics expose prevalence, temporal change,
representative evidence, confidence, model and prompt version, and source coverage. Analysts may
rename, merge, split, or reassign topics. Corrections remain attributed inputs to later reprocessing
and evaluation rather than silent edits of generated history.

### Release intelligence

Release intelligence classifies evidence-backed changes into known categories including feature,
breaking change, deprecation, security fix, performance improvement, and developer experience.
Claims link to collected changelog sections, pull requests, diff metadata, issues, or other public
source material. Missing changelogs or sparse evidence reduce the analysis rather than invite
invention.

### Documentation knowledge

Documentation ingestion follows only explicit Project URLs and allowed domains. It respects visible
scope controls for depth, size, frequency, content type, and robots behavior and never performs open
web discovery. Snapshots preserve original URL, collection time, content identity, and provenance.
Search and RAG answers cite the exact snapshot and retain original-language evidence.

### Natural-language analytics and HITL actions

Natural-language queries operate over the requesting user's authorized Projects and explicit or
clarified time windows. Responses show scope, cutoff, structured findings, citations, coverage, and
uncertainty. Ambiguous material scope triggers clarification. Missing evidence yields an explicit
insufficient-data response.

The assistant can translate a request into one typed, non-destructive Analyst action at a time:
register or edit Projects and Repositories, request synchronization or reanalysis, manage alert
rules, or add radar annotations. It displays exact inputs, affected resources, expected effect,
quota implications, and expiration before asking for approval. Approval is atomic, action-specific,
non-reusable, permission-checked again at execution, and audited. The assistant cannot manage users,
roles, credentials, policy definitions, archives, deletion, or any Admin-only action.

### Immutable AI analysis

Every probabilistic analysis is an immutable run containing structured output, source evidence,
provider/model identity, prompt version, execution time, language, and terminal status. A rerun
creates a new version. Analysts can flag an output, attach feedback, request a rerun, and select the
currently presented successful version; they cannot edit generated content in place. Stale runs
remain visible as stale and never masquerade as current.

The VPS Operator configures one or more shared external or local model providers. Admins can inspect
redacted capability, health, aggregate usage, and cost status. Provider absence or failure affects
only AI-dependent analysis; collection, deterministic metrics, health, policies, radar, comparison,
and observed trends remain available.

### Adoption policies and technology radar

Every completed policy evaluation returns exactly one of `recommended`, `conditional`,
`not_recommended`, or `insufficient_data`. It discloses policy owner and version, window, inputs,
weights and thresholds, decisive factors, evidence, freshness, and missing data. A transparent
default policy ships with the product. Admins clone, validate, publish, activate, and retire
immutable policy versions. Analysts select an existing version but cannot author one.

The technology radar derives a suggested ring from an explicit mapping in the selected policy
version. Analysts choose displayed Projects, add context, owner, and review date, and may override a
ring with required justification. Recommendation, suggested ring, and human override remain distinct
and attributed. Policy changes or stale evidence never silently move an override.

### In-app alerts

Alerts are in-app only. Analysts and Admins create shared rules for releases, breaking changes,
public security evidence, health dimensions, observed trends, early warnings, and issue/discussion
topics. Each occurrence exposes severity, rule version, Project, window, evidence, detection time,
deduplication, and cooldown behavior.

Occurrences are shared; read state is per user. Acknowledgement, resolution, dismissal, and
reopening are shared, attributed transitions, with justification where the action changes team
interpretation. Repeated qualifying evidence inside cooldown updates or suppresses the existing
occurrence rather than generating alert noise.

### Evidence export and audit

Viewers can export tabular metrics, snapshots, and comparisons as CSV with explicit units, windows,
cutoffs, and missing-data representation. They can also export JSON evidence packages containing
scope, coverage, provenance, formulas, versions, policy context, and references needed to interpret
the result. Exports never exceed the requesting user's access and obey resource quotas.

The Admin audit surface covers membership, roles, integrations, Project and source changes,
automatic associations and corrections, policies, radar, alerts, manual synchronization, analysis
runs, exports, and HITL proposals, approvals, and outcomes. Events identify actor, time, resource,
action, safe prior/new state, and result without secrets or sensitive payloads. Deleted users become
opaque actor identities; deleted Projects retain only their minimal tombstones.

### Bilingual, mobile-first experience

Every Visitor, Applicant, Viewer, Analyst, and Admin journey must remain complete on mobile rather
than offering a read-only or desktop fallback. Desktop and tablet layouts may use denser
visualizations, but narrow layouts cannot omit required information or authorized actions.

English and Brazilian Portuguese are first-class product languages. Fixed UI, validation, errors,
status, and generated analysis follow the user's preference. Original source evidence retains its
language; optional translations are labeled and never replace the original. Stored and exported
instants use UTC; the UI renders the user's selected timezone and allows the UTC instant to be
inspected.

The web product must support keyboard operation, screen readers, visible focus, 200% zoom, reduced
motion, non-color state cues, and textual or tabular equivalents for charts. The complete
mobile-first surface targets WCAG 2.2 AA behavior.

<!-- markdownlint-disable MD029 -->

## Business Rules

### Workspace, identity, and permissions

1. The product contains exactly one shared workspace; it does not partition Projects or intelligence
   into tenant workspaces.
2. Keycloak authentication and local product membership are independent. A valid external subject
   without approved membership is always `pending` and has no protected product permissions.
3. Anonymous users can read only the limited public Project representation. Pending Applicants can
   read only their access status and the public representation.
4. Every approved member has exactly one fixed local role: `Admin`, `Analyst`, or `Viewer`.
5. `Admin` controls membership, roles, policy definitions and versions, and permanent deletion.
6. `Analyst` curates Projects and sources, requests synchronization and analysis, manages alerts,
   selects published policies, corrects probabilistic and identity results, and governs radar
   entries.
7. `Viewer` reads dashboards, evidence, comparisons, recommendations, radar, alerts, and permitted
   exports without mutating shared product state.
8. The final active Admin cannot remove or suspend their own Admin capability.
9. Product roles are local. No Keycloak claim grants an equivalent role in this or another workspace
   product without the local approval mapping.
10. Account deletion removes personal profile, preferences, and sessions, retains shared resources,
    and pseudonymizes historical audit authorship with a non-reusable opaque identity.

### Project identity and lifecycle

11. A Project is not a Repository. Every Project owns at least one Repository and exactly one
    primary Repository while it exists outside deletion.
12. Every Repository has exactly one product-defined role at a time. A primary replacement validates
    before the former primary loses its role.
13. One canonical source resource can belong to only one active or archived Project association at a
    time. Re-registering an exact resource resolves to its existing Project.
14. Automatic source associations retain source evidence, method, confidence, and decision version.
    Analysts can correct them, and retained corrections constrain later resolution.
15. There is no hard product limit of five Projects. The original three-to-five count is a
    validation cohort; operational quotas may bound work and storage without changing the catalog
    model.
16. Valid Project states are `active`, `paused`, and `archived`; permanent deletion is terminal.
17. Only active Projects schedule collection. Paused Projects retain readable data. Archived
    Projects are read-only and excluded from default active views.
18. Permanent deletion is Admin-only, requires explicit Project-specific confirmation, cancels work,
    purges Project-owned data and exports, and retains only a payload-free audit tombstone.

### Sources, collection, and history

19. Every collected resource must be public at collection time. Operator credentials can
    authenticate requests but cannot broaden eligible visibility.
20. Users cannot submit, choose, or retrieve source credentials. Admins see only redacted
    operational capability; the VPS Operator owns secret lifecycle.
21. Collection must be incremental, idempotent, bounded, retryable, quota-aware, checkpointed, and
    independently observable by Project and source.
22. Duplicate compatible collection requests coalesce. A retry cannot duplicate canonical records,
    snapshots, or published evidence.
23. The default initial backfill target is 180 days. An operator-approved range may be longer. Older
    open issues and pull requests remain eligible regardless of that target.
24. Requested range and actual coverage are separate values. Every metric, analysis, comparison,
    trend, policy result, and export reports the actual evidence cutoff and coverage.
25. Selected raw evidence, canonical data, daily metric snapshots, corrections, and analysis
    versions remain until permanent Project deletion. Removal or privatization of a source changes
    current availability but does not silently rewrite historical provenance.
26. Documentation crawling is limited to explicitly associated URLs/domains and declared crawl
    boundaries. It never performs general web discovery.

### Metrics, health, comparison, and signals

27. Numerical metrics, health dimensions, overall health, policy results, observed trends, and
    synchronization decisions never depend on an LLM.
28. Every metric result binds to one definition version, observation window, cutoff, aggregation
    boundary, source coverage, and missing-data rule.
29. Preset windows are exactly 30, 90, 180, and 365 days. Custom intervals cannot exceed actual
    available coverage or use an end before the start.
30. Comparisons include two through five unique Projects and use one identical window, cutoff, and
    compatible metric version set.
31. `unknown`, `not_applicable`, and `insufficient_data` are not numeric zero and cannot be ranked
    as if they were values.
32. Health displays Activity, Community, Maintenance, Concentration, Stability, Security, and
    Adoption independently. A calculable overall score remains secondary and cannot publish without
    its required evidence.
33. Registry adoption indicators normalize only inside a source population with comparable units and
    collection semantics. No universal cross-registry adoption score exists.
34. Contributor accounts combine only through public verification or Analyst confirmation. Private
    emails and name similarity alone cannot establish identity.
35. Observed trends and predictive warnings are different result types. A forecast always discloses
    horizon, uncertainty, evaluation context, and version and never uses factual trend labeling.

### Policies, AI, HITL, radar, and alerts

36. A policy result is exactly `recommended`, `conditional`, `not_recommended`, or
    `insufficient_data`; policy version and evidence remain immutable with the result.
37. Published policy versions are immutable. Retirement prevents new selection but never invalidates
    historical results automatically.
38. AI runs are immutable and evidence-bearing. Reruns create versions; feedback and
    presented-version selection are separate attributed records.
39. AI output may explain deterministic evidence but cannot set a metric, policy outcome, health
    score, observed trend, predictive signal, radar mapping, or collection action.
40. AI failure cannot block or erase deterministic collection and intelligence.
41. A HITL approval applies to one displayed, unexpired, unchanged, non-destructive Analyst action.
    Authorization and state preconditions are rechecked at execution, and replay cannot repeat it.
42. The assistant cannot manage membership, roles, credentials, policies, archive, deletion, or
    Admin-only operations even when the requester is an Admin.
43. Radar suggestions derive from an explicit published policy version. A manual override requires
    justification, author, owner, and review date and never replaces the original recommendation.
44. Alert occurrences are shared workspace facts; read state is per user. Shared acknowledgement,
    resolution, dismissal, and reopening remain attributed.
45. Alerts are in-app only. The product does not send email or outbound webhook notifications.

### Presentation, limits, and evidence

46. All protected responses, exports, assistant evidence, and deep links enforce current local
    membership and role; UI visibility never substitutes for backend authorization.
47. Public registration, protected queries, exports, crawling, collection, and AI work are subject
    to explicit rate and resource limits. Hitting a limit produces a visible retry or
    operator-action state and never silently drops accepted work.
48. English and Brazilian Portuguese are complete product languages. Original-language evidence is
    retained, and any translation is labeled as derived content.
49. UTC is the canonical stored and exported time. User-selected timezones affect presentation only,
    and the UTC value remains inspectable.
50. Every chart has an equivalent textual or tabular representation, and no conclusion depends only
    on color, pointer input, desktop width, animation, or hover.
51. Every export binds to one authorization scope and data cutoff. Project deletion removes
    generated exports owned by that Project.
52. Secrets, authorization headers, private source content, private contributor emails, raw model
    credentials, and sensitive payloads never appear in logs, audit output, API responses, exports,
    prompts, or evidence packages.

<!-- markdownlint-enable MD029 -->

## User Experience

### Visitor and Applicant journey

1. A Visitor lands on the bilingual public catalog and searches public Project names and
   descriptions.
2. Protected calls to action explain that full intelligence requires approved membership.
3. Registration or sign-in transfers identity handling to the shared Keycloak.
4. After successful authentication, an unapproved Applicant sees pending status rather than a blank
   or forbidden product.
5. After local approval, the next valid session opens the portfolio overview with the assigned role.

### Analyst adoption-decision journey

1. The Analyst searches the portfolio or registers a supported public repository URL.
2. The product resolves the Project, primary Repository, related public sources, and actual
   collection coverage while exposing uncertain associations for correction.
3. Collection progress and deterministic metrics appear before AI enrichment and remain usable if AI
   is unavailable.
4. The Analyst inspects independent health dimensions, contributor concentration, adoption and
   security evidence, releases, issue/discussion topics, observed trends, and predictive warnings.
5. The Analyst compares two to five alternatives over one window and cutoff.
6. A selected published policy produces an auditable adoption outcome and suggested radar ring.
7. The Analyst adds organizational context or a justified radar override, configures relevant in-app
   alert rules, and exports evidence when needed.

### Investigation and assistant journey

1. The Analyst asks a question in English or Portuguese and names or selects Projects and a window.
2. The product clarifies material ambiguity, then returns structured findings with cutoff, coverage,
   and source citations.
3. If the request implies a supported mutation, the assistant displays one typed action proposal.
4. The Analyst approves or rejects the exact proposal. Approval executes only if role and state
   remain valid; changed state requires a new preview.
5. The resulting action and audit entry link back to the proposal and user decision.

### Admin governance journey

1. The Admin reviews authenticated Applicants and approves one local fixed role or rejects access.
2. The Admin monitors membership, source/model capability, quotas, failures, and shared audit
   history.
3. The Admin clones and validates the default policy, publishes immutable versions, and controls
   which versions remain selectable.
4. The Admin handles lifecycle operations, using pause and archive for retention and permanent
   deletion only after Project-specific confirmation.
5. Audit search and exports explain who changed identity, evidence, policy, radar, alert, or
   assistant state and whether the operation succeeded.

### Mobile, localization, and accessibility

All journeys above are complete on mobile, tablet, and desktop. Mobile layouts prioritize current
status, decision, evidence, and primary action before secondary detail, but do not remove authorized
functionality. English and Portuguese preferences apply before protected content renders. Charts,
radars, comparisons, trends, progress, and alert severity provide screen-reader and table/text
alternatives. Keyboard, focus, zoom, reduced-motion, and non-color cues are acceptance requirements,
not optional polish.

The screen-by-screen surface contract will be defined in `_uiux.md` during Stage 2 after the product
checkpoint.

## High-Level Technical Constraints

- The product must consume the workspace-shared Keycloak for authentication. Keycloak deployment,
  realm governance, mail, recovery, and any shared identity repository remain external workspace
  responsibilities. This repository owns its client integration, pending membership, local roles,
  authorization enforcement, and product audit.
- The service must remain deployable on a Linux VPS with containers and without a mandatory public
  cloud service. External source and model APIs may be configured, and a local model may satisfy the
  model-provider boundary.
- Browser clients communicate only through versioned HTTP/JSON product contracts. Collection,
  normalization, metrics, policy evaluation, identity resolution, comparison, and scoring remain
  backend responsibilities.
- External source, package registry, model, embedding, reranking, and optional future registry
  implementations must remain replaceable adapters. Provider DTOs and SDK concepts cannot define
  canonical Project intelligence.
- Collection and analysis are asynchronous from the user's perspective. Accepted work exposes a
  durable status, progress or queue state, cancellation/failure semantics, and a reproducible
  result; interactive reads must not wait for full backfill or model execution.
- Deterministic work must be reproducible from versioned definitions and evidence. Concurrent work
  must be bounded, idempotent, cancelable, retryable with backoff, quota-aware, checkpointed, and
  safe across process restart.
- Selected raw source material and provenance required for reprocessing or audit must be preserved
  until Project deletion. Raw content cannot appear in logs merely because it is retained as
  evidence.
- Every AI capability must use structured inputs and outputs, evidence references, versioned
  prompts, model/run metadata, explicit terminal status, and an evaluation dataset appropriate to
  the capability. Model unavailability must degrade only AI-dependent surfaces.
- Source requests, jobs, calculations, policy evaluations, identity decisions, tool calls, and model
  runs must expose correlated operational status without recording secrets or sensitive content.
- Deployment health routes and the redacted Admin operations API must expose configuration validity,
  provider health, quotas, queues, aggregate usage, and repairable failure states. This product does
  not ship a CLI or credential-management endpoint; the VPS Operator uses the deployment's external
  configuration, secret, migration, and process surfaces.
- Built-in adapters are limited to the selected providers. Additional registries may integrate only
  through the source extension boundary and must preserve the same public-data, canonical-model,
  provenance, quota, and observability rules.
- User-visible behavior must meet the complete English/Portuguese, mobile-first, and WCAG 2.2 AA
  requirements in this specification.

## Non-Goals (Out of Scope)

- **Keycloak ownership**: this repository will not deploy, configure globally, migrate, back up, or
  operate the Keycloak shared by the three workspace products, and it will not create the proposed
  shared identity repository.
- **Product-owned passwords or identity recovery**: registration credentials, email verification,
  password reset, MFA, and global sessions belong to the shared identity system.
- **Multi-workspace or multi-tenant portfolios**: all approved users participate in one product
  workspace and one visibility boundary.
- **Custom roles or permission builders**: the product exposes only Admin, Analyst, and Viewer.
- **Private-source intelligence**: private repositories, packages, documentation, websites,
  discussions, advisories, and organization-only data are never collected, even when an operator
  credential could access them.
- **Per-user source or model credentials**: users cannot bring or select API keys.
- **General web search or open-ended crawling**: only explicitly linked Project domains and URLs are
  eligible for documentation ingestion.
- **Source-code, dependency, or environment vulnerability scanning**: the Security dimension uses
  public ecosystem evidence and does not generate SBOMs or inspect adopters' systems.
- **A universal popularity, adoption, or health truth**: cross-registry adoption is not collapsed
  into one universal score, and health is not reduced to a binary label.
- **User-authored metric formulas**: users configure deterministic adoption policies over a
  product-owned metric catalog rather than programming metrics.
- **AI-defined facts or decisions**: LLMs do not set numeric metrics, synchronize data, resolve
  policy outcomes, produce health values, detect deterministic trends, generate forecasts, or place
  radar entries autonomously.
- **Unbounded or destructive assistant operation**: the conversational assistant cannot execute
  Admin, credential, policy-authoring, archive, deletion, or arbitrary provider-tool actions.
- **Email, SMS, push, or outbound webhook alerts**: all alert delivery is in-app for this product.
- **Scheduled PDF reporting, billing, subscriptions, or paid plans**: exports are on-demand CSV and
  JSON; quotas protect the shared VPS rather than define commerce.
- **General-purpose extension marketplace**: the product offers source/provider adapter boundaries,
  not an end-user marketplace or arbitrary runtime code execution surface.

## Open Questions

The product frontier is closed. The following external identity decisions are intentionally owned by
the workspace-level Keycloak initiative and must be supplied as integration constraints during Stage
2 rather than answered or implemented in this repository:

- Whether the shared Keycloak receives a dedicated repository and which workspace team owns its
  deployment, upgrades, backups, availability, and incident response.
- Realm and client topology for the three independent products, including stable subject and claim
  contracts, redirect-origin registration, and environment separation.
- Global registration verification, account recovery, MFA, session policy, mail delivery, and
  identity-side abuse protection.
- The coordination process and compatibility guarantees for Keycloak changes that can affect all
  three repositories.

This specification assumes the shared identity service will expose a stable standards-based client
contract, allow public registration, and return a stable subject identifier. Failure to provide that
contract blocks interactive authentication but does not transfer Keycloak ownership into this
repository.

## Part II — Technical Specification

## Executive Summary

Open Source Project Intelligence will ship as a modular monolith with two Go processes and one React
single-page application. The API owns versioned HTTP contracts, authentication callbacks,
authorization, synchronous reads, and durable command acceptance. The worker owns collection,
normalization, metric materialization, search indexing, analysis, alert evaluation, exports, and
deletion workflows. Both processes share business packages and PostgreSQL, but communicate
asynchronous intent through a transactional outbox and NATS JetStream.

PostgreSQL 18 is the source of truth for identities, projects, normalized intelligence, job state,
checkpoints, metrics, analyses, and audit records. pgvector extends PostgreSQL for evidence
retrieval. An S3-compatible store keeps large raw captures, Git-derived artifacts, document bodies,
diffs, and generated exports. Valkey is non-authoritative: it may accelerate rate limiting, cache
safe projections, and fan out ephemeral job events, but its loss cannot lose accepted work or facts.
NATS JetStream transports durable work notifications; PostgreSQL remains authoritative for Job state
and the outbox.

The public surface is the frozen contract in `_dx.md` and `_uiux.md`. OpenAPI 3.1 drives Go request
validation and the TypeScript client. SQL remains explicit and reviewed, with `sqlc` generating
typed pgx bindings. Every deterministic conclusion is versioned and reproducible. Every AI result is
evidence-backed, schema-constrained, separately versioned, and optional to core deterministic
operation. Google ADK Go v2 is contained inside the conversational analysis adapter and is not a
general application framework.

## MVP Boundary

This specification defines one complete delivery. Its MVP is **all tasks generated from this spec,
from Task 001 through Task N**. The proposal's original MVP, V2, and V3 labels become implementation
increments in the build order below; they are not post-MVP deferrals and do not relax any accepted
story or surface. No capability inside this spec may be silently postponed beyond Task N.

The delivery is complete only when:

- public registration through the shared Keycloak, local approval, and the three fixed product roles
  protect a publicly reachable VPS deployment;
- projects and all accepted public source kinds can be registered, collected, corrected, archived,
  restored, and deleted with the specified evidence and audit semantics;
- the seven metric families, versioned health dimensions, comparisons, trends, forecasts, policies,
  radar, topics, search, documentation intelligence, releases, security evidence, and adoption
  evidence are available through the frozen HTTP and UI surfaces;
- evidence-backed AI analysis and the HITL conversational flow work when model capabilities are
  configured and degrade locally when they are not;
- jobs, checkpoints, outbox delivery, retries, cancellation, exports, alerts, and deletion remain
  correct across process restart and dependency failure;
- English and Brazilian Portuguese, responsive layouts, keyboard use, screen-reader semantics, and
  WCAG 2.2 AA requirements are verified; and
- deployment, observability, migration, backup-facing data ownership, and operational repair paths
  are documented and testable.

The validation cohort starts with three to five projects, but this is not a hard product limit.
Capacity is governed by explicit quotas and bounded workers rather than an artificial row cap.

## Developer Experience

`_dx.md` is the normative developer and operator contract. Browser clients use same-origin `/api/v1`
HTTP/JSON and opaque server-side sessions. Automation uses bearer tokens issued externally by the
shared Keycloak and locally approved service-account identities and scopes. Mutations return durable
resources or Jobs, never process-local handles. JSON Snowflake identifiers are decimal strings,
timestamps are RFC 3339 UTC, windows are half-open `[from, to)`, pagination uses opaque cursors, and
errors use the frozen problem envelope. Concurrent edits use the frozen version and `If-Match`
rules; retriable commands use the frozen idempotency contract.

OpenAPI and generated clients are checked for drift, not an alternative source of product behavior.
The repository keeps a single Go module and existing pnpm workspace. Developers run PostgreSQL,
NATS, Valkey, and S3-compatible storage through the repository Compose project. Keycloak remains an
external shared dependency; local development points at an operator-provided development realm or a
standards-compatible test issuer. No root workspace, cross-repository source import, or shared
application Compose file is introduced.

## System Architecture

```text
Browser / automation
        |
same-origin reverse proxy / TLS
        |---------------- React SPA
        `---------------- Go API (/api/v1, /healthz, /readyz)
                              |        |       |
                         PostgreSQL  Valkey  S3-compatible storage
                              |
                    transactional outbox
                              |
                        NATS JetStream
                              |
                           Go worker
                    /      /    |      \
             provider APIs Git  web  model providers
```

The reverse proxy terminates TLS and exposes one public origin. It serves the built SPA and forwards
the API and health routes. Keycloak redirects return to the same origin. The API commits commands,
their initial Jobs, and outbox records atomically. An outbox relay publishes with the outbox ID as
the JetStream message identity. Workers claim the canonical PostgreSQL Job, renew a lease and
heartbeat, checkpoint progress, and acknowledge the message only after the durable transition
commits.

Large evidence is written to a content-addressed S3 key before its PostgreSQL reference becomes
visible. Deletion uses a resumable purge manifest: database state marks content unavailable first,
objects are deleted idempotently, and final tombstones commit after reconciliation. Valkey may
broadcast progress to SSE connections; the API always hydrates initial and final state from
PostgreSQL and can fall back to bounded database polling.

## Architectural Boundaries

| Boundary                  | Owns                                                                                     | May depend on                                        | Must not depend on                                                       |
| ------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------- | ------------------------------------------------------------------------ |
| `cmd/api`                 | composition, lifecycle, routes, middleware                                               | business constructors, platform adapters             | business rules, provider DTOs                                            |
| `cmd/worker`              | composition, consumers, lifecycle                                                        | business handlers, platform adapters                 | domain decisions, HTTP handlers                                          |
| Business packages         | canonical models, rules, use cases, consumer ports                                       | standard library, narrower business contracts        | platform packages, SDKs, pgx, NATS, Valkey, S3, HTTP DTOs                |
| `internal/platform/*`     | database, broker, cache, object, identity, source, model, Git, crawl, telemetry adapters | vendor SDKs and ports they implement                 | cross-provider canonical policy                                          |
| `internal/analysis/agent` | bounded ADK orchestration and HITL continuation                                          | ADK v2 and analysis/tool ports                       | direct database, broker, provider, or unrestricted infrastructure access |
| HTTP transport            | generated validation, mapping, status/error serialization                                | application services                                 | persistence queries and metric formulas                                  |
| Generated SQL             | typed pgx access to reviewed queries                                                     | migrations and database adapter                      | domain orchestration                                                     |
| `apps/web`                | rendering, navigation, forms, accessible interaction, localization                       | generated TypeScript client through feature adapters | metric, policy, identity, or scoring rules                               |

Existing business packages retain their intended ownership: `project`, `repository`, `collector`,
`issue`, `pullrequest`, `release`, `contributor`, `metric`, `comparison`, and `analysis`. Add
narrowly named packages for `auth`, `membership`, `job`, `source`, `evidence`, `document`, `topic`,
`search`, `policy`, `radar`, `trend`, `alert`, `export`, and `audit` only with their capability.
There is no generic `utils`, `services`, or `daemon` package.

Cross-capability reads use small interfaces declared by the consumer. A comparison service may read
metric and contributor projections through its own ports, but cannot import their repositories or
reach PostgreSQL. Provider DTOs stop in their adapter. ADK types stop in `internal/analysis/agent`.
The two `cmd` roots are the only places where platform implementations are connected to business
ports.

## Product-to-Technical Traceability

The goal numbers below follow the bullet order in Part I `Goals`. A row is satisfied only by the
named backend owner, its frozen HTTP contract, and the named UI surface where one exists.

| Part I goal                                               | Technical owner                                                                              |
| --------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| G1 discovery, identity, approval, bilingual mobile access | `auth`, `membership`, HTTP session/catalog handlers, generated client, localized React shell |
| G2 repository-led multi-source Project registration       | `project`, `repository`, `source`, provider discovery adapters and lifecycle handlers        |
| G3 incremental observable public collection               | `collector`, `job`, checkpoints, outbox, JetStream consumers and operations projections      |
| G4 canonical facts, raw evidence and reproducibility      | canonical capability packages, `evidence.Store`, PostgreSQL versions and S3 references       |
| G5 closed seven-family metric catalog                     | `metric.Engine`, immutable definitions/snapshots/factors and metric routes                   |
| G6 auditable health dimensions and secondary score        | versioned health definitions/snapshots and accessible health presentation                    |
| G7 same-window two-to-five comparison                     | `comparison` with metric/contributor read ports and immutable comparison views               |
| G8 topics, releases and documentation knowledge           | `topic`, `release`, `document`, `search`, crawl/model adapters and evidence references       |
| G9 observed trends and predictive warnings                | `trend`, deterministic methods, immutable results and uncertainty views                      |
| G10 four-state deterministic adoption policy              | `policy`, normalized immutable rules/evaluations and recommendation handlers                 |
| G11 policy-derived radar with overrides                   | `radar`, evaluation read ports, versioned override/history records and radar handlers        |
| G12 natural-language analysis and bounded HITL            | `analysis`, `internal/analysis/agent`, allowlisted tools and proposal/confirmation handlers  |
| G13 immutable evidenced AI and graceful degradation       | prompt/run/evidence/tool records, model capability ports and selected-run lifecycle          |
| G14 deduplicated alerts and auditable exports             | `alert`, `export`, Job/evidence/object ports and member/shared state handlers                |
| G15 attribution without secret exposure                   | `audit`, authorization middleware, redaction policy and operations projections               |

| Story catalog     | Technical components                                                           | Frozen UI/API surfaces                                                       |
| ----------------- | ------------------------------------------------------------------------------ | ---------------------------------------------------------------------------- |
| `US-001`–`US-004` | `auth`, `membership`, `audit`, session store and Keycloak adapter              | S1–S3, S23; public/session/member/admin routes                               |
| `US-005`–`US-009` | `project`, `repository`, `source`, `evidence`, `job`                           | S4–S7; portfolio/project/repository/source/lifecycle routes                  |
| `US-010`–`US-012` | `collector`, `job`, checkpoints, outbox, provider/Git/crawl adapters           | S7–S8, S23; sync/history/job/operations routes                               |
| `US-013`–`US-017` | `metric`, `contributor`, `comparison`, `trend`                                 | S9–S13; metrics/health/contributor/adoption/security/comparison/trend routes |
| `US-018`–`US-020` | `policy`, `radar`, versioned evaluations and overrides                         | S14–S15; recommendation/policy/radar routes                                  |
| `US-021`–`US-026` | `topic`, `release`, `document`, `search`, `analysis`, agent/model ports        | S16–S20; topics/releases/crawl/search/query/analysis/assistant routes        |
| `US-027`–`US-031` | `alert`, `export`, `audit`, operations, localization/accessibility shell       | S1, S21–S23; alerts/exports/audit/operations routes                          |
| `US-032`          | bearer validation, `auth.Authorizer`, service-account scopes, quotas and audit | API-only automation contract plus S23 local binding administration           |

This table covers every Part I story group and every S1–S23 UI surface. `_tests.md` expands it to
one acceptance row for each story, each of its ten edge cases, every HTTP/message contract and every
surface.

## Implementation Design

### Core Interfaces

The signatures below freeze the principal dependency direction. Implementations may add private
types but must preserve these consumer-owned shapes or supersede them through the spec change
process.

```go
package auth

type Principal struct {
    IdentityID string
    Role       Role
    Scopes     []Scope
}

type Authorizer interface {
    Authorize(context.Context, Principal, Action, Resource) error
}
```

`Authorizer` applies local membership, suspension, role, and service-account scope rules. OIDC token
verification is a transport concern; the interface receives no JWT or Keycloak type.

```go
package job

type Command struct {
    Kind           Kind
    ProjectID      string
    IdempotencyKey string
    RequestedBy    string
    Input          any
}

type Dispatcher interface {
    Dispatch(context.Context, Command) (Job, error)
}
```

`Dispatcher` atomically stores the Job, command-specific state, and outbox event. `Input` is a
validated command value selected by kind, never an untyped provider payload.

```go
package collector

type RepositorySource interface {
    Repository(context.Context, SourceRef) (Repository, error)
    Commits(context.Context, SourceRef, Cursor) (CommitPage, error)
    Issues(context.Context, SourceRef, Cursor) (IssuePage, error)
    PullRequests(context.Context, SourceRef, Cursor) (PullRequestPage, error)
    Releases(context.Context, SourceRef, Cursor) (ReleasePage, error)
}
```

Each provider adapter maps responses to canonical ingestion records before returning. Provider
pagination tokens remain opaque inside `Cursor` and never enter canonical entities.

```go
package metric

type Request struct {
    ProjectID  string
    Definition DefinitionRef
    Window     Window
    AsOf       time.Time
}

type Engine interface {
    Compute(context.Context, Request) (Snapshot, error)
    Explain(context.Context, SnapshotID string) (Factors, error)
}
```

The engine accepts half-open UTC windows and immutable version references. It returns coverage and
unavailable dimensions rather than silently changing weights.

```go
package evidence

type Store interface {
    Put(context.Context, Capture) (Reference, error)
    Get(context.Context, Reference) (io.ReadCloser, error)
    Verify(context.Context, Reference) error
    SchedulePurge(context.Context, ProjectID) (Purge, error)
}
```

The business reference contains only an immutable digest, provenance, media type, size, and
storage-neutral ID; large bodies may live in S3.

```go
package analysis

type AgentRunner interface {
    Start(context.Context, AgentRequest) (Run, error)
    Continue(context.Context, RunID, Decision) (Run, error)
    Cancel(context.Context, RunID, PrincipalID) error
}
```

Only conversational and multi-tool HITL flows implement `AgentRunner` through ADK. Release
summaries, labels, explanations, embeddings, and reranking use narrower model capability ports and
ordinary application services.

### Data Models

All public IDs are Snowflake decimal strings and PostgreSQL `bigint` values. Every mutable aggregate
has a monotonic `version`; all timestamps are UTC. Foreign keys and uniqueness constraints enforce
workspace ownership, source identity, and idempotency rather than leaving those invariants solely to
application code.

#### Identity and access

| Entity                   | Binding fields and rationale                                                                                            | Storage decision                                                      |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------- |
| `workspaces`             | `id`, `name`, `created_at`; one seeded row is the authorization boundary                                                | Columns; no tenant JSON                                               |
| `external_identities`    | `id`, `issuer`, `subject`, optional display email/name, created/last-seen times; `(issuer, subject)` is stable identity | Columns; volatile token claims are not persisted wholesale            |
| `memberships`            | workspace, identity, role, status, version, approval and suspension actors/times                                        | Columns because each field participates in authorization or audit     |
| `service_accounts`       | identity tuple, local role/status/version, display label and approval metadata                                          | Columns; credentials remain in Keycloak                               |
| `service_account_scopes` | service account and scope                                                                                               | Side table because scopes are matchable and independently constrained |
| `sessions`               | random ID, identity, hashed verifier, creation, expiry, last-seen and revocation times                                  | Columns; the browser stores only the opaque session cookie            |
| `snowflake_node_leases`  | node ID, holder, lease deadline and version                                                                             | Side table with transactional lease semantics                         |

#### Projects, sources, and collection

| Entity                 | Binding fields and rationale                                                                     | Storage decision                                                    |
| ---------------------- | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------- |
| `projects`             | workspace, slug, name, description, lifecycle/deletion state, version, creator and audit times   | Columns; slug unique among non-deleted projects                     |
| `repositories`         | project, provider, provider ID, owner/name, canonical URL, role, default branch, state/version   | Columns; `(provider, external_id)` prevents duplicate registration  |
| `sources`              | project, kind, canonical URL, provider/external ID, state and configuration version              | Columns; kind selects an explicit adapter                           |
| `source_associations`  | source/project, confidence, method, review state/version and decision audit                      | Side table because analysts accept, reject, or move associations    |
| `identity_corrections` | canonical contributor, source identity, correction kind/version and actor/time                   | Side table so corrections survive re-ingestion                      |
| `sync_checkpoints`     | source, stream, opaque cursor, high-water mark, version and update time                          | Columns plus JSON only for provider-owned opaque cursor tokens      |
| `raw_objects`          | project/source, object key, SHA-256 digest, media type, byte size, capture time, retention state | Columns; bounded provider metadata may be JSON, bodies belong in S3 |
| `source_events`        | source, provider event ID/type, observed/effective times, raw reference and normalization state  | Columns; unique provider identity makes replay idempotent           |

#### Durable work and delivery

| Entity            | Binding fields and rationale                                                                                                                                           | Storage decision                                                              |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| `jobs`            | workspace/project, kind, state, priority, completed/total/unit progress, checkpoint key, idempotency key, version, schedule/start/finish/cancel/error and actor fields | Columns because operations, recovery, filtering and UI read them directly     |
| `job_attempts`    | job, attempt number, worker, start/heartbeat/finish and outcome/error                                                                                                  | Side table so retries never overwrite evidence                                |
| `outbox`          | aggregate type/ID, event type, schema version, payload, create/publish/attempt fields                                                                                  | JSON payload is allowed as a versioned delivery envelope, not canonical state |
| `purge_manifests` | project, phase, object counts, cursor, attempts and terminal state/time                                                                                                | Columns plus an opaque object-store continuation cursor                       |
| `exports`         | job, requester, format, object reference, created/expires/downloaded times and state                                                                                   | Columns; files expire exactly 24 hours after completion                       |

#### Canonical intelligence

| Entity family         | Binding fields and rationale                                                                                              | Storage decision                                                                 |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| Contributors          | canonical contributor, aliases, provider identities, bot/human class, confidence and correction version                   | Contributor, alias, and identity side tables; never a JSON alias array           |
| Commits/contributions | repository, provider ID/SHA, author, authored/committed time, default-branch/merge flags, additions/deletions             | Columns; large message bodies may remain in raw evidence                         |
| Issues/comments       | provider IDs, repository, author, state, labels, public/member/bot flags, creation/update/close/reopen and response times | Main rows plus label, comment, and state-event side tables                       |
| Pull requests         | provider IDs, participants, draft/readiness/merge state, base/head and lifecycle times                                    | Main rows plus review, label, participant, and state-event side tables           |
| Releases/changes      | repository, provider/tag identity, stable/draft/prerelease flags, publish time, previous release and evidence             | Main rows plus change/evidence tables; AI interpretation stays separate          |
| Discussions           | provider identity, author, category, accepted answer/state and times                                                      | Main rows plus comment/participant side tables                                   |
| Packages/adoption     | registry/package/version identity, project association, downloads/dependents/stars and observation time                   | Normalized packages and per-registry time-series snapshots                       |
| Advisories/security   | ecosystem/provider identity, package/range, severity, publication/update/withdrawal and project link                      | Normalized rows plus affected-range and evidence tables                          |
| Documents             | source/canonical URL, version digest, capture time, language, title and object reference                                  | Document, version, and chunk tables; bodies in S3, searchable text in PostgreSQL |

#### Metrics and intelligence products

| Entity                              | Binding fields and rationale                                                                   | Storage decision                                                       |
| ----------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `metric_definitions`                | key, semantic version, formula parameters, valid-from and status                               | Immutable rows; parameters JSON is schema-validated definition data    |
| `metric_snapshots`                  | project, definition, `[from,to)`, `as_of`, value, unit, coverage and status                    | Columns; a unique computation key makes reruns idempotent              |
| `metric_factors`                    | snapshot, factor key/value/unit/evidence                                                       | Side table so explanations remain queryable                            |
| `health_definitions`                | semantic version, seven dimension keys, equal weights, thresholds and valid-from               | Immutable rows; missing dimensions never redistribute weight           |
| `health_snapshots`                  | project, definition, window/as-of, dimension values, coverage and status                       | Main row plus dimension/factor side tables                             |
| `comparisons`                       | requester, project set, metric/health versions, common window/as-of and state                  | Main row plus ordered items; release analysis remains separate         |
| `trends` / `forecasts`              | series, method/version, window/as-of, estimate, slope/seasonality, interval and backtest error | Typed columns with immutable parameter-version references              |
| `topic_clusters`                    | project, algorithm/version, label state, period, centroid and stability                        | Main row plus membership, label evidence, and analyst constraints      |
| `policy_definitions`                | project, name, status/version and owner                                                        | Policy versions and normalized rule tables; no executable formula JSON |
| `policy_evaluations`                | policy version, project, as-of, outcome and matched facts/evidence                             | Main row plus factor table                                             |
| `radar_entries`                     | project/technology, quadrant/ring, rationale, evaluation and state/version                     | Columns; movement history is a side table                              |
| `alert_rules` / `alert_occurrences` | owner, typed condition/version and state; occurrence value/evidence/read state                 | Separate rule and occurrence tables; no outbound delivery payload      |
| `prompt_versions`                   | capability, semantic version, template digest, input/output schemas and status                 | Immutable rows; schemas/templates are versioned artifacts              |
| `analysis_runs`                     | capability/prompt/model versions, requester, state, token/cost/latency, times and error        | Columns; output, evidence, HITL decisions and tool calls are separate  |
| `analysis_evidence`                 | run, evidence reference, quoted-span digest, relevance and order                               | Side table; large generated artifacts live in S3                       |
| `audit_events`                      | actor type/ID, action, resource, outcome, correlation, time and redacted changes               | Append-only columns; bounded redacted changes may be JSON              |

Canonical matchable state is never hidden in JSON. JSON is limited to opaque provider cursors,
versioned schemas or envelopes, bounded raw metadata, and redacted display snapshots. This keeps SQL
constraints, deletion, reprocessing, and audits explicit.

#### HTTP request and response models

OpenAPI defines transport models separately from domain entities. Create and update requests contain
only user-controlled fields and reject unknown properties. Responses use immutable summary, detail,
list-page, Job, metric-series, analysis-run, evidence-reference, and problem models. Generated types
preserve decimal-string IDs and RFC 3339 timestamps. Handlers map generated DTOs to application
commands and views; database rows and provider records are never serialized directly.

### API Endpoints

The route catalog in `_dx.md` is binding. The implementation groups it into the following generated
OpenAPI operations and application handlers:

| Route family                                                                           | Application responsibility                                                                                 | Required failure behavior                                                                                        |
| -------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Public catalog and session                                                             | public project projections, current access state, logout, preferences, account deletion                    | Catalog never leaks protected evidence; unapproved users receive the frozen access state; logout is idempotent   |
| Admin members, service accounts, audit, operations                                     | local approval, fixed roles, scopes, suspension, immutable audit and redacted dependency status            | Admin authorization precedes lookup; duplicate subjects, stale versions and unsafe scopes use frozen problems    |
| Portfolio, projects, repositories, sources, associations                               | aggregate views and lifecycle commands while preserving `Project != Repository` and one primary repository | canonical URL duplicates, private sources, last-primary removal and stale versions fail atomically               |
| Jobs, syncs and history                                                                | quota admission, coalescing, cancellation, durable status and resumable SSE                                | accepted work is durable; non-cancellable purge and terminal Jobs reject cancellation; SSE never becomes truth   |
| Metrics, health, contributors, adoption, security, comparisons, trends, recommendation | deterministic version selection, common windows, coverage and evidence                                     | missing or incomparable data remains explicit; no fallback formula or weight redistribution                      |
| Policies and radar                                                                     | immutable policy versions, deterministic evaluations, attributed overrides                                 | invalid rule trees never activate; override removal exposes the computed placement                               |
| Topics, releases, crawl, search and analysis                                           | corrected topic graph, structured release claims, safe crawl, hybrid retrieval and versioned model runs    | model loss affects only model-dependent output; every claim requires evidence; corrections never rewrite history |
| Assistant proposals                                                                    | bounded read tools, typed allowed proposals, ten-minute single-use HITL confirmation                       | Admin, credential, policy-authoring, archive, deletion and arbitrary actions return `action_not_allowed`         |
| Alerts and exports                                                                     | in-app occurrences/read state and asynchronous CSV/evidence JSON generation                                | deduplication is deterministic; expired exports return `410`; object URLs do not outlive 24-hour metadata        |

The HTTP stack is stdlib `net/http` and `http.ServeMux`. A generated strict-server interface
provides operation input and output types; thin handlers perform authentication, call application
services, and map typed errors. Middleware order is request ID, panic recovery, trusted proxy
normalization, security headers, access logging, tracing, body/timeout limits, authentication, local
authorization, rate limiting, generated validation, handler, and response metrics. Logs contain
route templates and error codes, not query strings, cookies, tokens, prompts, evidence bodies, or
free-form feedback.

Browser authentication uses Authorization Code with PKCE against Keycloak, then creates an opaque
product session. Unsafe cookie-authenticated methods require the exact configured Origin and valid
Fetch Metadata headers; cookies are `Secure`, `HttpOnly`, `SameSite=Lax`, and narrowly scoped.
Bearer authentication validates issuer, audience, signature, time claims, and stable subject through
a cached JWKS whose stale use is bounded. Every principal is then checked against the local
membership or service-account binding; Keycloak claims do not grant product roles directly.

Generated validation rejects unknown JSON properties, oversized bodies, invalid decimal IDs,
unsupported locales/windows, and malformed cursors before application code. Opaque pagination
cursors are canonical serialized positions authenticated with an HMAC key and bound to route,
filters, sort and visibility. `Idempotency-Key` records bind actor, operation, normalized input and
terminal response; reuse with different input returns the frozen conflict. `If-Match` compares the
aggregate version inside the same transaction as mutation.

Every long operation returns a persisted Job. SSE begins with a PostgreSQL snapshot, emits
monotonically ordered Job versions, and terminates after the durable terminal state. `Last-Event-ID`
resumes from a short retained event sequence when available and otherwise emits the latest state.
Valkey loss makes the API poll PostgreSQL at a bounded cadence or close with retry guidance;
`GET /jobs/{id}` remains available. No collection or AI operation holds an HTTP request open until
completion.

### Collection, Metrics, Search, and AI Pipelines

Collection resolves and validates a public source before scheduling work. Provider API adapters
backfill and incrementally synchronize repository objects. A restricted native Git adapter maintains
bare mirrors for history that provider APIs cannot provide efficiently. Registry and crawl adapters
emit canonical capture records. Each page commits normalized rows, raw provenance references,
checkpoint progress, and any downstream outbox events in one transaction. Retries restart at the
last committed checkpoint; upserts compare provider identity and content/version digests.

The scheduler stores intent in PostgreSQL and publishes through the outbox. JetStream subjects are
versioned by capability, consumers use explicit acknowledgements, and messages carry only IDs,
schema version, correlation and trace context. Consumers are at-least-once, so handlers acquire the
Job lease and recheck terminal/idempotency state. Backoff honors provider reset headers and bounded
jitter. Concurrency is separately limited by global worker, provider, host, project and expensive-AI
semaphores. Cancellation is cooperative and checked between pages, model/tool steps and object
writes.

Daily snapshot jobs freeze a cutoff and definition version before calculation. The seven metric
families and health dimensions use the exact cohort rules in ADR 032. Comparisons reference existing
snapshots or materialize all participants under one common cutoff. Trend detection uses Theil-Sen
slope plus Mann-Kendall significance; forecasts use versioned exponential-smoothing or seasonal
baselines selected through rolling backtests and always publish intervals and error. Model output
cannot alter these values.

Document versions are chunked deterministically with headings and source offsets. PostgreSQL full-
text rank and pgvector nearest-neighbor rank are fused through reciprocal-rank fusion; optional
reranking operates on a bounded candidate set and cannot invent evidence. Topic candidates use a
deterministic mutual-k-nearest-neighbor graph over versioned embeddings. LLMs may label candidate
clusters, while analyst merge/split/reassign constraints are canonical inputs to the next run.

Simple AI capabilities call structured-generation, embedding, or rerank ports through ordinary
services. The initial adapter is OpenAI-compatible but no provider type crosses the port. ADK v2 is
used only inside the assistant/NL-analysis runner for a bounded, allowlisted tool plan. The runner
limits steps, duration, output, cost and concurrent tools, persists every tool request/result
digest, and pauses before a typed mutation proposal. Confirmation reauthorizes the actor and
resource version; the token is action-bound, single-use and expires after ten minutes.

## Integration Points

| Integration           | Adapter contract                                                                | Authentication and resilience                                                                               |
| --------------------- | ------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| Shared Keycloak       | OIDC discovery, authorization code/PKCE, JWKS, logout and bearer validation     | client secret from external secret injection; bounded JWKS cache; issuer/audience fail closed; no Admin API |
| GitHub, GitLab, Gitea | source-specific metadata, issues, PRs, releases, discussions and quota adapters | operator tokens only; public-resource check; conditional requests, reset-aware backoff and checkpoints      |
| Native Git remotes    | restricted bare fetch and commit graph reader                                   | HTTPS public URLs only; no credential helpers, hooks, submodules, working tree or local/file/SSH transports |
| Package registries    | typed package identity and observation snapshots                                | public endpoints/operator credentials; source-specific quota and provenance                                 |
| Project websites/docs | allowlisted crawl captures                                                      | DNS/IP revalidation, redirect checks, size/type/depth/time limits and per-host concurrency                  |
| PostgreSQL + pgvector | authoritative repositories, transactions, leases, outbox, FTS and vectors       | TLS in production; context deadlines; readiness probe; transaction-level retries only where safe            |
| NATS JetStream        | versioned work notification subjects and durable consumers                      | external credentials; publish deduplication; explicit ack, redelivery and dead-letter advisory              |
| Valkey                | disposable rate-limit/cache/SSE fan-out                                         | external credentials/TLS; short TTLs; circuit breaker; no accepted work or canonical facts                  |
| S3-compatible storage | immutable evidence objects, purge and expiring exports                          | scoped credentials; checksums, multipart bounds, retry and reconciliation                                   |
| Model providers       | structured generation, embeddings and rerank ports                              | external API key; timeout, quota, concurrency, schema validation and redacted usage telemetry               |
| Google ADK Go v2      | bounded analysis-agent orchestration only                                       | no direct infrastructure tools; persisted run/step/HITL state; application-level authorization              |
| OpenTelemetry         | HTTP, job, collector, SQL, broker, tool and model signals                       | optional exporter; application continues when export fails; sensitive fields excluded                       |

Provider SDKs are permitted only inside their adapter and only when they materially reduce protocol
risk. The adapter maps immediately into canonical records. Integration contracts are exercised
against provider fixtures and controlled HTTP servers; live provider tests are opt-in and never part
of deterministic CI.

## Impact Analysis

| Surface                                     | Change                                                                                           | Risk and migration treatment                                                                                   |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------- |
| `cmd/api`                                   | replace empty `/api/v1` fallback with generated routes, auth, middleware and composition         | High; retain `/healthz`, `/readyz` and graceful shutdown; no compatibility shim for a route that never existed |
| `cmd/worker`                                | replace database-heartbeat ticker with scheduler/outbox/JetStream consumers                      | High; keep graceful shutdown; delete the placeholder loop when the first durable consumer lands                |
| `internal/platform/config`                  | validate identity, broker, cache, object, source, AI and safety configuration                    | High; accumulate redacted startup errors and preserve existing keys                                            |
| `internal/platform/database` and migrations | add PostgreSQL 18 schema, pgvector, transactions, leases and generated sqlc access               | High; forward-only numbered migrations and real-PostgreSQL tests                                               |
| Business packages                           | replace documentation placeholders with canonical models/use cases and add accepted capabilities | High; enforce direction with package tests and import review                                                   |
| Platform adapters                           | add Keycloak, providers, Git, crawl, NATS, Valkey, S3 and model implementations                  | High; each remains replaceable behind consumer ports                                                           |
| OpenAPI/sqlc generation                     | add pinned specs, config, generated Go/TypeScript/SQL code and drift checks                      | Medium; generated files are never hand-edited                                                                  |
| `apps/web`                                  | replace placeholder with the frozen localized accessible SPA                                     | High; generated client and route manifest prevent contract/link drift                                          |
| Compose and deployment                      | add PostgreSQL 18/pgvector, NATS, Valkey and S3 development services                             | Medium; preserve assigned ports; shared Keycloak remains external                                              |
| Repository specs/ADRs                       | publish numbered repository-owned specifications and superseding decisions before code           | Medium; `.compozy` remains workflow source, repository `specs/` becomes implementation contract                |

There is no released business API or persisted business schema to preserve. The empty versioned 404
handler, placeholder web screen, worker heartbeat, and documentation-only package placeholders are
deleted as their real owners land. Health/readiness, current configuration names, graceful shutdown,
telemetry behavior, pgx choice, explicit SQL, and handwritten migration ownership remain. No dual
write, deprecated endpoint, framework compatibility layer, or hidden old scheduler is introduced.

## Extensibility Integration Plan

The product exposes no Compozy skill, MCP server, bridge, plugin marketplace, or runtime
code-loading surface. It needs no extension manifest or user installation lifecycle. Extensibility
is compile-time and adapter-based:

- a source adapter registers a unique kind, implements consumer-owned discovery/collection ports,
  maps into canonical records, and declares capabilities and quota semantics;
- model adapters implement structured-generation, embedding, or reranking ports independently;
- object, broker and cache adapters implement storage-neutral contracts but have one supported MVP
  implementation each; and
- all additions require fixtures, safety policy, provenance mapping, telemetry and a repository ADR
  when they introduce a material dependency or persistence model.

Unknown kinds and capabilities fail explicitly. The API never accepts a package name, executable,
template or endpoint that dynamically loads arbitrary application code.

## Agent Manageability Plan

The product ships no separate CLI. Agents and automation receive the same documented HTTP surface as
other service accounts, with scopes, idempotency, structured problems and durable Jobs. No operation
requires clicking the UI: project/source management, sync, intelligence reads, corrections,
analysis, feedback, alerts and exports all have API routes. Administrative automation uses only
explicitly scoped service accounts and cannot mint credentials through this product.

`GET /healthz` proves process liveness, `/readyz` reports required dependency readiness, and
`GET /api/v1/admin/operations` supplies redacted capability, quota, queue, usage and repair state.
Job GET/SSE/cancel routes provide bounded waiting and recovery. External deployment surfaces own
migrations, secrets, process control, backups and the shared Keycloak. Errors have stable codes,
request IDs, safe remediation details and retry hints so an agent does not need to parse logs or
human copy.

## Configuration Lifecycle

Both processes read environment configuration once at startup into typed immutable structs. The API
and worker validate all applicable fields, accumulate safe errors, and exit before serving or
claiming work when a required invariant is invalid. No endpoint reveals secret values or accepts
credential mutation. Secret material comes from the VPS secret mechanism; checked-in examples
contain names and safe placeholders only.

| Group            | Keys and defaults                                                                                                                                                       | Lifecycle                                                                                  |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| Existing runtime | `ENVIRONMENT=development`, `HTTP_ADDRESS`, `DATABASE_URL`, `SHUTDOWN_TIMEOUT=15s`, `WORKER_CONCURRENCY=4`, existing OTLP keys                                           | Preserve existing meaning; database required by both processes                             |
| Public origin    | `PUBLIC_BASE_URL`; production requires one HTTPS origin                                                                                                                 | Restart API after change; drives redirects, cookie security and Origin checks              |
| OIDC             | `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URL`, `OIDC_AUDIENCE`, `SESSION_TTL=12h`                                                      | Restart API; issuer/audience/redirect mismatch is fatal when authentication is enabled     |
| NATS             | `NATS_URL`, credential/TLS inputs, `NATS_STREAM_PREFIX=opi`, `JOB_MAX_ATTEMPTS=8`, `JOB_HEARTBEAT=15s`                                                                  | Restart API/worker; required for command processing, not liveness                          |
| Valkey           | `VALKEY_URL`, credential/TLS inputs and namespace                                                                                                                       | Restart API/worker; absence is a readiness degradation only where fallback is safe         |
| S3               | endpoint, region `us-east-1`, bucket, access key, secret key, TLS and `S3_FORCE_PATH_STYLE`                                                                             | Restart; bucket and credentials required for evidence-bearing collection                   |
| Sources          | operator provider tokens, `SOURCE_CONCURRENCY=8`, `CRAWL_CONCURRENCY=4`, `CRAWL_MAX_DEPTH=3`, `CRAWL_MAX_BYTES=10485760`, `CRAWL_MAX_REDIRECTS=5`, `SOURCE_TIMEOUT=30s` | Restart worker; missing optional provider token disables that capability visibly           |
| Models           | base URL, API key, structured/embedding/rerank model IDs and per-capability enable flags                                                                                | Restart worker; deterministic surfaces remain ready if all model capabilities are disabled |
| Agent limits     | `ADK_MAX_STEPS=12`, `ADK_TIMEOUT=2m`, bounded output/tokens/cost and tool concurrency                                                                                   | Restart worker; invalid or unbounded values are fatal when agent capability is enabled     |
| Product defaults | `HISTORY_DAYS=180`, supported locales `en,pt-BR`, fallback `en`                                                                                                         | Versioned product defaults; export expiry remains fixed at 24 hours                        |

Conditional requirements are capability-aware. For example, a deployment may start without a model
key and report AI unavailable, but cannot report S3-backed crawl available without its bucket. A
change that can alter historical interpretation belongs in an immutable database definition version,
not an environment variable. Runtime reconfiguration is intentionally absent; operators restart a
process after validated config changes.

## Testing Approach

`_tests.md` is the normative test catalog and coverage matrix. The implementation uses:

- standard-library, table-driven, parallel-safe unit tests for domain rules, time windows, errors,
  identifiers, cursors, authorization, metric cohorts, scoring, trend/forecast methods, topic
  constraints, safety validation and retry decisions;
- fuzz tests for URL/cursor/ID parsing, OpenAPI boundary decoding, metric and rule inputs,
  Git/provider normalization, chunking, assistant proposals and evidence citations;
- Testcontainers with real PostgreSQL/pgvector, NATS JetStream, Valkey and S3-compatible services
  for transaction, constraint, redelivery, lease, cache-loss, object and migration semantics;
- controlled HTTP/OIDC/provider/model servers for authentication, rate-limit, redirect, schema,
  timeout and evidence behavior;
- Vitest, React Testing Library and MSW for web features, routing, localization, accessibility and
  frozen HTTP states;
- Playwright for browser journeys in English and Portuguese, narrow and wide viewports,
  keyboard-only use, session/access transitions, SSE/poll fallback and destructive confirmation; and
- generated OpenAPI/sqlc drift checks, `go test ./...`, `go test -race ./...`, lint, typecheck, web
  tests, production build and Playwright/axe checks in CI.

Tests never weaken rate limits, permissions, evidence requirements, SQL constraints, retry behavior,
or accessibility to pass. Concurrency tests prove cancellation and no goroutine leaks. Integration
fixtures use deterministic clocks/IDs and isolate databases, streams, buckets and cache namespaces.
Live external provider/model tests are opt-in diagnostics, never release gates.

## Development Sequencing

### Build Order

1. Publish numbered repository specifications and superseding ADRs for SQL generation, frontend
   dependencies, data services/messaging, test containers and ADK. Add pinned OpenAPI/sqlc
   generation and drift checks.
2. Introduce PostgreSQL 18/pgvector, Snowflake leases, identity/project/job/outbox base migrations,
   typed configuration and Testcontainers harnesses.
3. Add shared-Keycloak OIDC integration, opaque sessions, local memberships, fixed roles/scopes,
   audit and the public/admin account surfaces.
4. Implement Project, Repository and Source lifecycles, hardened URL validation, quotas, durable
   Jobs, outbox relay, JetStream consumers, Valkey degradation and S3 evidence ownership.
5. Implement GitHub plus restricted native Git ingestion, canonical commits/issues/PRs/releases/
   contributors, checkpoints, correction persistence and initial 180-day backfill.
6. Materialize daily snapshots, the seven deterministic metric families, equal-weight versioned
   health, contribution resolution, comparisons and evidence explanations.
7. Replace the web placeholder with the localized accessible shell, catalog, access states,
   portfolio/project views, job UX and operational admin views.
8. Add GitLab/Gitea, discussions, registries, advisories, safe project-linked crawling, document
   versioning, PostgreSQL FTS/pgvector hybrid retrieval and adoption/security surfaces.
9. Add versioned policy authoring/evaluation, recommendation, radar, alerts, observed trends,
   Theil-Sen/Mann-Kendall early warnings and backtested interval forecasts.
10. Add deterministic topic candidates and analyst constraints, structured release analysis,
    evidence-backed search/analysis, feedback, reruns and selected-run lifecycle.
11. Add the bounded ADK conversational runner, allowlisted read tools, typed proposal preview,
    action-bound HITL confirmation and the frozen forbidden-action policy.
12. Add asynchronous exports, account/project purge reconciliation, scale/race/accessibility/E2E
    suites, recovery drills and final operations/deployment documentation.

Each increment remains deployable and migratable, but Task N is the only completion boundary.

### Technical Dependencies

- Identity, authorization, Snowflake generation, migrations, Jobs and outbox precede protected
  business endpoints.
- Project/Repository/Source canonical models and evidence storage precede collectors.
- Canonical collection and corrections precede metrics, search, topics, policy and AI.
- Versioned metric snapshots precede health, comparison, trends, forecasts, policy, radar and
  alerts.
- Document chunks and evidence references precede hybrid retrieval and any cited model output.
- Model capability ports and deterministic tool services precede ADK orchestration.
- The generated client and localized route manifest precede feature UI implementation.
- Durable purge manifests and storage ownership must exist before account/project deletion is
  enabled.

## Monitoring and Observability

All requests, Jobs, collection pages, SQL transactions, outbox publications, JetStream deliveries,
S3 operations, cache fallbacks, tool calls and model calls carry a request/job correlation ID and
OpenTelemetry trace context. `slog` records structured event names, resource IDs, versions,
duration, outcome, safe error code and retry count. It excludes secrets, auth headers, cookies, raw
source bodies, document chunks, questions, model prompts/outputs, feedback text and unrestricted
URLs.

Required metrics include:

- HTTP rate/latency/error by method, route template and status; session/OIDC outcome; authorization
  denial and rate-limit outcome;
- Job queue age, state counts, lease/heartbeat age, attempts, cancellation latency and terminal
  outcome by kind;
- outbox unpublished age/count, publish latency/failure, JetStream consumer lag/redelivery/ack and
  dead-letter advisory count;
- provider request rate/latency/status, quota remaining/reset, checkpoint age, rows normalized,
  duplicate ratio and collection freshness by adapter;
- database pool saturation/query latency/transaction retries, S3 latency/checksum/purge backlog,
  Valkey fallback/circuit state and Snowflake lease health;
- metric/health computation duration, unavailable/coverage counts, definition version, trend and
  forecast backtest error;
- search candidate counts/latency, FTS/vector/rerank contribution, topic stability, evidence-missing
  rejection and correction backlog;
- model/tool/ADK latency, token and cost buckets, schema rejection, step count, HITL wait/expiry and
  capability availability; and
- web route errors, localization fallback, SSE reconnect/poll fallback and sampled accessibility
  regression telemetry without user content.

Operational objectives start with: health pings under 100 ms locally; API p95 reads under 500 ms and
accepted commands under 1 s at the validation cohort; unpublished outbox oldest age under 30 s;
healthy worker heartbeat under 45 s; no active Job without a heartbeat for two intervals; no
collection checkpoint older than its documented schedule plus one retry window; and zero evidence-
less successful AI claims. These are initial alert thresholds, not product KPIs, and are tuned from
production evidence without changing business definitions.

`/healthz` reports process liveness only. `/readyz` reports required database and role-specific
dependency state while distinguishing optional/degraded capabilities. Admin operations exposes the
same redacted truth plus quotas, lag, freshness, circuit states and remediation codes. The OTel
exporter itself is optional and cannot take the application down.

## Technical Considerations

### Key Decisions

- PostgreSQL owns every accepted fact and Job; NATS delivers, Valkey accelerates, and S3 holds large
  immutable bytes under database ownership.
- An outbox plus idempotent consumers provides restart safety without distributed transactions.
- OpenAPI and sqlc generate boundary code from reviewed contracts while stdlib HTTP, pgx and manual
  migrations remain the architecture.
- Shared Keycloak owns identity; this product owns membership, roles, scopes, sessions and
  authorization.
- Database-leased Snowflake nodes produce decimal-string public IDs without a centralized ID
  service.
- Metrics, health, policies, trends, forecasts and topic candidates remain deterministic and
  versioned; LLMs interpret evidence but do not create facts.
- ADK is restricted to the bounded conversational/HITL adapter; ordinary AI calls remain ordinary
  services behind capability ports.
- The frontend uses generated contracts, React Router Data Mode, TanStack Query, React Hook Form,
  Zod, React Aria, CSS tokens/modules, TanStack Table, ECharts and checked-in i18n dictionaries.

### Known Risks

| Risk                                                               | Mitigation                                                                                                                     |
| ------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------ |
| Shared Keycloak contract or availability blocks interactive login  | Keep OIDC-only integration, external ownership and explicit readiness; test against a standards issuer; document stable claims |
| Public VPS registration and expensive work invite abuse            | pending approval, fixed roles/scopes, per-identity/IP/resource quotas, durable admission checks and bounded concurrency        |
| Provider drift, quotas and incomplete history reduce comparability | source-specific adapters, capability/coverage metadata, checkpoints, conditional requests and explicit unavailable states      |
| NATS/Valkey/S3 expand single-VPS operations                        | explicit ownership, Compose/dev parity, readiness/degradation, reconciliation and recovery tests; no hidden broker truth       |
| Snowflake clock regression or lease collision corrupts identity    | DB-exclusive leases, monotonic timestamp guard, sequence exhaustion wait and fatal clock-regression handling                   |
| Crawl/Git inputs expose SSRF or local execution                    | public-only allowlist, DNS/IP revalidation, transport restrictions, hooks/helpers disabled and resource limits                 |
| Model hallucination or agent overreach harms trust                 | evidence gate, schemas, versioning, allowlisted tools, step/cost bounds, HITL and deterministic action authorization           |
| Full scope creates long sequencing and migration pressure          | vertical increments, one source of truth, explicit dependencies, generated drift gates and Task N completion boundary          |
| pgvector/FTS quality varies across languages and projects          | versioned evaluation datasets, rank-factor telemetry, deterministic fallback and visible coverage                              |
| Project deletion spans database and object storage                 | unavailable-first state, resumable purge manifest, idempotent object delete and reconciliation                                 |

## Safety Invariants

1. No unauthenticated route exposes protected project, evidence, member, Job, analysis or
   operational data; only the frozen public catalog and auth/session-entry behavior are public.
2. Keycloak identity claims never grant a product role; active local membership or service-account
   binding is mandatory on every protected request.
3. Authorization runs before resource-existence disclosure, and every mutation rechecks permission
   and aggregate version in its write transaction.
4. Browser sessions are opaque, server-side, revocable, secure-cookie bound and protected by exact
   Origin and Fetch Metadata checks on unsafe methods.
5. Secrets, tokens, cookies, connection strings, prompts, evidence bodies and sensitive free text
   are never logged, traced, returned by operations status or stored in audit change JSON.
6. Every public ID comes from a currently exclusive Snowflake node lease; backward clock movement
   beyond the tolerated bound stops issuance rather than risking collision.
7. A Project and a Repository are distinct aggregates, and every active Project retains exactly one
   primary Repository.
8. Only public sources are eligible. A credential's ability to access a private source never makes
   that source collectable.
9. Source URL validation rechecks DNS/IP after redirects and rejects loopback, link-local, private,
   metadata, file, SSH and other non-approved targets.
10. Native Git never runs hooks, credential helpers, submodules, working-tree checkout or provider-
    supplied executables, and it enforces byte, object, depth and duration limits.
11. Accepted asynchronous work, idempotency and terminal state are committed in PostgreSQL before a
    successful response; NATS and Valkey cannot be the only record.
12. Outbox publication and every consumer are at-least-once safe; replay cannot duplicate canonical
    facts, audit outcomes, alerts, exports or mutations.
13. Workers use bounded concurrency, leases, heartbeats, cooperative cancellation, checkpoints and
    provider-aware backoff; shutdown stops claims before draining owned work.
14. Job progress is monotonic within a versioned attempt; a terminal Job never returns to an active
    state, and a purge Job is non-cancellable after destructive execution begins.
15. S3 objects become visible only through a committed checksum-bearing database reference; orphan
    reconciliation may delete only objects proven unreferenced.
16. Valkey loss may increase latency or SSE reconnects but cannot lose accepted work, authorization
    state, evidence, metric values or audit history.
17. Canonical matchable state is normalized in typed columns/side tables; opaque JSON cannot decide
    authorization, metrics, lifecycle, deletion or policy outcomes.
18. All metric and comparison windows are UTC half-open intervals `[from,to)` with one frozen
    cutoff, definition version and repository set.
19. Stable release metrics exclude drafts and prereleases; contributor concentration uses human
    non-merge default-branch commits; these cohorts cannot vary per request.
20. Issue first response, PR readiness-to-merge, and backlog reopen semantics follow ADR 032 and
    expose fallback/coverage rather than imputing unavailable facts.
21. Health uses seven equal-weight absolute dimensions; a missing dimension stays unavailable and
    its weight is never redistributed.
22. Trends and forecasts publish method/version, evidence window, coverage, uncertainty and backtest
    quality; they are not deterministic facts about the future.
23. Policy and radar outcomes are derived only from immutable rule versions and canonical facts;
    overrides are attributed, reviewable and never rewrite computed placement.
24. Topic membership originates from the deterministic graph and persisted analyst constraints; LLM
    labeling cannot add or remove evidence members.
25. Every successful AI claim links to accessible immutable evidence and records prompt, model,
    schema, run, tool, usage and terminal status metadata.
26. Model unavailability never changes deterministic metrics, health, policies, trends, forecasts,
    source synchronization or authorization.
27. ADK tools are allowlisted typed application capabilities with no direct SQL, filesystem,
    arbitrary HTTP, credential, broker or provider-SDK access.
28. Assistant mutation proposals are typed, previewed, action-bound, single-use, ten-minute limited,
    reauthorized on confirmation and forbidden for Admin/destructive classes.
29. Analyst corrections and selected analysis versions append history; reprocessing never erases the
    original evidence, run or responsible actor.
30. Project/account deletion first makes data unavailable, then executes a resumable idempotent
    purge and retains only the minimum non-sensitive tombstone/audit record required by the
    contract.
31. Export access is requester-authorized, checksum-bound and expires 24 hours after successful Job
    completion; object-store URLs cannot extend that deadline.
32. English and `pt-BR` share identical permissions, facts and action availability; missing
    translation falls back to English without exposing raw keys or changing route identity.
33. Every interactive feature remains keyboard operable and semantically announced; charts have
    equivalent textual/tabular data and color is never the only status signal.
34. Readiness and Admin operations distinguish required failure from optional capability degradation
    and never claim a dependency healthy from configuration presence alone.
35. Repository-owned specs, ADRs, migrations and generated-contract drift checks must land before
    the implementation that depends on them.

## File References

Paths are repository-relative unless stated otherwise. Line ranges describe the load-bearing content
at specification time; implementations should reread the full file after later edits.

| Path                                                                                  | Read reason                                                                                       |
| ------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `/workspace/docs/opensource_project_intelligence.md:1-949`                            | Primary product proposal, complete capability arc, metric intent and original phase vocabulary    |
| `AGENTS.md:1-77` and `CLAUDE.md:1-77`                                                 | Repository architecture, data, concurrency, testing, specification and safety constraints         |
| `README.md:1-111`                                                                     | Implemented foundation, locked stack, commands, routes and explicit missing business scope        |
| `specs/README.md:1-34`                                                                | Repository specification ownership and numbering contract that Task 001 must populate             |
| `specs/adrs/0001-record-architecture-decisions.md`                                    | Accepted ADR immutability and supersession process                                                |
| `specs/adrs/0002-http-and-application-framework.md:1-38`                              | stdlib HTTP, `cmd` composition roots and modular-monolith constraint                              |
| `specs/adrs/0003-persistence-driver-and-migrations.md:1-42`                           | pgx, explicit SQL, handwritten migrations and query-generator reassessment trigger                |
| `specs/adrs/0004-testing-strategy.md:1-40`                                            | stdlib/table/race testing baseline and real-service boundary expectations                         |
| `specs/adrs/0005-observability-with-opentelemetry.md`                                 | OpenTelemetry and structured logging ownership                                                    |
| `specs/adrs/0006-frontend-toolchain.md:1-35`                                          | React/Vite/TypeScript baseline and dependency-decision requirement                                |
| `cmd/api/main.go:34-120`                                                              | Current API configuration, lifecycle, health/readiness and empty versioned-route replacement seam |
| `cmd/worker/main.go:32-116`                                                           | Current worker lifecycle, concurrency config and placeholder heartbeat replacement seam           |
| `internal/collector/bounded.go:10-74`                                                 | Existing tested bounded-concurrency primitive and cancellation semantics                          |
| `internal/platform/config/config.go:16-136`                                           | Current immutable environment configuration and validation pattern                                |
| `internal/platform/database/database.go:16-55`                                        | Existing pgx pool construction, ping, timeout and close semantics                                 |
| `internal/platform/httpx/httpx.go:1-44`                                               | Existing JSON health response and method behavior                                                 |
| `internal/platform/telemetry/telemetry.go:1-77`                                       | Existing optional OTLP initialization and shutdown behavior                                       |
| `internal/project/doc.go`, `internal/repository/doc.go`                               | Project-versus-Repository ownership invariant and intended capability boundaries                  |
| `internal/issue/doc.go`, `internal/pullrequest/doc.go`, `internal/release/doc.go`     | Canonical collaboration and release capability boundaries                                         |
| `internal/contributor/doc.go`, `internal/metric/doc.go`, `internal/comparison/doc.go` | Identity, deterministic metric and same-window comparison boundaries                              |
| `internal/analysis/doc.go`                                                            | Evidence-backed analysis boundary and LLM interpretation limit                                    |
| `apps/web/src/App.tsx:1-13`                                                           | Placeholder UI to replace with the frozen surface                                                 |
| `apps/web/src/config.ts:1-19`                                                         | Existing backend-only `/api/v1` client boundary                                                   |
| `compose.yaml:1-65`                                                                   | Repository-scoped Compose name, current PostgreSQL service and assigned ports                     |
| `migrations/README.md:1-21`, `scripts/migrate.sh:1-69`                                | Handwritten migration naming, apply/rollback and external lifecycle                               |
| `.compozy/tasks/opensource-project-intelligence/_user_stories.md`                     | Normative 32-story and 320-edge-case product catalog                                              |
| `.compozy/tasks/opensource-project-intelligence/_dx.md`                               | Frozen HTTP, authentication, lifecycle, errors and deployment surface                             |
| `.compozy/tasks/opensource-project-intelligence/_uiux.md`                             | Frozen route, state, responsive, localization and accessibility surface                           |
| `.compozy/tasks/opensource-project-intelligence/_tests.md`                            | Normative verification cases and coverage matrix                                                  |
| `.compozy/tasks/opensource-project-intelligence/adrs/adr-001.md` through `adr-035.md` | Product and technical decision record set for this workflow                                       |

External design references are intentionally cited by concept rather than invented repository paths:
OIDC/OAuth 2.1 and Keycloak client behavior; NATS JetStream delivery/deduplication; Valkey cache
semantics; S3 checksum/object lifecycle; OpenAPI 3.1 and oapi-codegen; sqlc/pgx; PostgreSQL
full-text search and pgvector; Google ADK Go v2; React Router, TanStack Query/Table, React Hook
Form, Zod, React Aria, ECharts, i18next, Playwright and axe. The implementing task must pin exact
versions and record vendored/generated artifacts where applicable before relying on them.

## Assumptions and Defaults

- The product is deployed on one operator-controlled VPS behind TLS and a same-origin reverse proxy.
  Horizontal high availability is not required, but restart recovery and backup/restore are.
- Exactly one product workspace is seeded. The initial three-to-five-project cohort is a validation
  target, not a schema, API or license limit.
- The external workspace initiative supplies a reachable shared Keycloak, stable issuer/audience/
  subject contract, registered redirect origins and production identity operations. This repository
  neither owns nor provisions it.
- Keycloak `sub` plus issuer is the stable external identity. Email and display name are mutable
  presentation attributes, not authorization keys.
- Registration is public at the identity layer, but protected workspace intelligence requires local
  Admin approval. Public catalog descriptions and source links remain intentionally limited.
- Only public Internet source data is in scope. Operator credentials improve quota or API access but
  cannot authorize private collection.
- The default backfill is 180 days while still retaining older currently open issues and pull
  requests needed for correct window/backlog computation.
- PostgreSQL 18 with pgvector, NATS JetStream, Valkey and S3-compatible storage are available in the
  supported deployment. PostgreSQL is the only canonical data authority.
- UTC is the storage/calculation time basis. User timezone affects presentation only. English is the
  translation fallback; English and `pt-BR` dictionaries are checked in together.
- Stable releases exclude drafts and prereleases. Provider data that cannot express a required
  cohort or event produces explicit coverage/unavailable status.
- The OpenAI-compatible adapter is the first model adapter, but no provider is guaranteed. A
  deployment without model capabilities remains valid for deterministic intelligence.
- Generated OpenAPI/sqlc code is committed if required by repository build ergonomics, is never
  hand-edited, and must reproduce cleanly with pinned tools.
- Exports expire 24 hours after successful generation. Opaque sessions default to 12 hours; these
  defaults may be superseded only through the frozen product/config change process.
- The external broker and object store introduced by this complete-scope spec supersede the original
  proposal's MVP infrastructure exclusion because the user explicitly selected the complete product
  vision as one delivery.

## Architecture Decision Records

The following accepted workflow ADRs are normative. Later ADRs supersede only the conflicting clause
they name; notably ADR-025 supersedes ADR-006's optional-score presentation, and ADR-027 requires a
repository ADR before introducing sqlc under ADR 0003's reassessment trigger.

1. [ADR-001: Require Application-Managed Access Control](adrs/adr-001.md)
2. [ADR-002: Treat the Complete Product Vision as One Delivery](adrs/adr-002.md)
3. [ADR-003: Make Adoption Guidance Deterministic and Policy-Driven](adrs/adr-003.md)
4. [ADR-004: Use One Shared Workspace With Fixed Product Roles](adrs/adr-004.md)
5. [ADR-005: Keep Projects Independent From Their Typed Repositories](adrs/adr-005.md)
6. [ADR-006: Present Health as Auditable Dimensions With an Optional Overall Score](adrs/adr-006.md)
7. [ADR-007: Use Four Explicit Adoption Recommendation Outcomes](adrs/adr-007.md)
8. [ADR-008: Resolve Cross-Source Project Identity Automatically and Reversibly](adrs/adr-008.md)
9. [ADR-009: Separate Observed Trends From Predictive Early Warnings](adrs/adr-009.md)
10. [ADR-010: Preserve Versioned, Immutable AI Analysis Runs](adrs/adr-010.md)
11. [ADR-011: Derive the Technology Radar From Versioned Adoption Policies](adrs/adr-011.md)
12. [ADR-012: Limit HITL Assistant Execution to Non-Destructive Analyst Actions](adrs/adr-012.md)
13. [ADR-013: Purge Deleted Project Data and Retain a Minimal Audit Tombstone](adrs/adr-013.md)
14. [ADR-014: Provide a Closed, Versioned Metric Catalog With Comparable Time Windows](adrs/adr-014.md)
15. [ADR-015: Consume a Workspace-Shared Keycloak and Keep Product Authorization Local](adrs/adr-015.md)
16. [ADR-016: Collect Only Public Source Data Using Operator-Managed Read Credentials](adrs/adr-016.md)
17. [ADR-017: Gate Workspace Intelligence Behind Approval While Exposing a Public Catalog Teaser](adrs/adr-017.md)
18. [ADR-018: Use Same-Origin Browser Sessions and Keycloak Bearer Tokens](adrs/adr-018.md)
19. [ADR-019: Expose Snowflake Resource IDs as Decimal Strings](adrs/adr-019.md)
20. [ADR-020: Stream Job Updates With SSE and Retain Polling](adrs/adr-020.md)
21. [ADR-021: Localize Browser Routes and Keep the API Stable in English](adrs/adr-021.md)
22. [ADR-022: Reserve Every Project Lifecycle Transition for Admins](adrs/adr-022.md)
23. [ADR-023: Use Conditional HTTP and Cursor APIs With Numbered Browser Pages](adrs/adr-023.md)
24. [ADR-024: Keep Credential Mutation Outside the Product Surface](adrs/adr-024.md)
25. [ADR-025: Always Show the Overall Health Score When Calculable](adrs/adr-025.md)
26. [ADR-026: Use PostgreSQL, S3 Storage, NATS JetStream, and Valkey With Explicit Ownership](adrs/adr-026.md)
27. [ADR-027: Generate HTTP and SQL Boundaries From Reviewed Contracts](adrs/adr-027.md)
28. [ADR-028: Keep Identity External and Authorization and Sessions Local](adrs/adr-028.md)
29. [ADR-029: Issue Snowflake IDs Through Database-Leased Nodes](adrs/adr-029.md)
30. [ADR-030: Combine Provider APIs With Restricted Git Mirrors and Hardened Crawling](adrs/adr-030.md)
31. [ADR-031: Build an Accessible Localized React SPA From Headless Primitives](adrs/adr-031.md)
32. [ADR-032: Version Metric Cohorts, Health Rubrics, Trends, and Forecasts](adrs/adr-032.md)
33. [ADR-033: Use PostgreSQL Hybrid Retrieval and Deterministic Topic Candidates](adrs/adr-033.md)
34. [ADR-034: Contain Google ADK Go v2 Behind Analysis Ports](adrs/adr-034.md)
35. [ADR-035: Verify With Real Boundaries and Publish Repository Specifications Before Code](adrs/adr-035.md)
