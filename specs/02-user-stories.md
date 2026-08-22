# User Stories: Open Source Project Intelligence

Canonical behavior catalog for Open Source Project Intelligence. Companion to `_spec.md`; consumed
by `_spec.md` Part II (component mapping), `_uiux.md` (surface states), and `_tests.md` (coverage
matrix).

## Personas

- **Visitor** — an unauthenticated person evaluating the service from its public catalog.
- **Applicant** — a Keycloak-authenticated person waiting for access to the shared product
  workspace.
- **Viewer** — an approved reader who inspects and exports intelligence without changing shared
  product state.
- **Analyst** — the primary persona: a technical lead, staff engineer, architect, or technology
  evaluator who curates projects and turns evidence into adoption decisions.
- **Admin** — a workspace steward who approves membership, governs roles and policies, and performs
  destructive lifecycle actions.
- **VPS Operator** — the person operating the deployment, source credentials, model providers, and
  global resource limits outside normal analytical workflows.
- **Automation Client** — an externally provisioned Keycloak service identity with an approved local
  Viewer or Analyst role and a least-privilege scope subset.

## Story Index

| ID     | Feature Area  | Persona           | Story                                                       |
| ------ | ------------- | ----------------- | ----------------------------------------------------------- |
| US-001 | Discovery     | Visitor           | Browse the limited public project catalog                   |
| US-002 | Access        | Applicant         | Authenticate and await workspace approval                   |
| US-003 | Access        | Admin             | Review and govern workspace membership                      |
| US-004 | Account       | Viewer            | Manage preferences and remove an account                    |
| US-005 | Portfolio     | Viewer            | Understand the portfolio from one overview                  |
| US-006 | Projects      | Analyst           | Register a project from a repository URL                    |
| US-007 | Projects      | Analyst           | Curate a multi-repository project                           |
| US-008 | Identity      | Analyst           | Review and correct automatic source associations            |
| US-009 | Projects      | Admin             | Pause, archive, restore, or delete a project                |
| US-010 | Collection    | Analyst           | Request and monitor synchronization                         |
| US-011 | Collection    | Analyst           | Understand historical coverage and freshness                |
| US-012 | Sources       | VPS Operator      | Operate public-data source integrations safely              |
| US-013 | Metrics       | Viewer            | Inspect deterministic metrics and health dimensions         |
| US-014 | Contributors  | Viewer            | Evaluate contributor sustainability and concentration       |
| US-015 | Adoption      | Viewer            | Interpret registry adoption and security evidence           |
| US-016 | Comparison    | Analyst           | Compare two to five projects in one window                  |
| US-017 | Trends        | Analyst           | Distinguish observed trends from early warnings             |
| US-018 | Policies      | Analyst           | Receive an auditable adoption recommendation                |
| US-019 | Policies      | Admin             | Author and version adoption policies                        |
| US-020 | Radar         | Analyst           | Govern a policy-derived technology radar                    |
| US-021 | Issues        | Analyst           | Explore and correct issue and discussion topics             |
| US-022 | Releases      | Viewer            | Understand what changed in a release                        |
| US-023 | Knowledge     | Analyst           | Search project documentation with evidence                  |
| US-024 | Analytics     | Analyst           | Ask evidence-backed natural-language questions              |
| US-025 | Assistant     | Analyst           | Approve a proposed non-destructive action                   |
| US-026 | AI Governance | Analyst           | Inspect, rerun, and review AI analysis versions             |
| US-027 | Alerts        | Analyst           | Configure and resolve shared in-app alerts                  |
| US-028 | Export        | Viewer            | Export tabular results and evidence packages                |
| US-029 | Audit         | Admin             | Investigate product actions through the audit log           |
| US-030 | Operations    | VPS Operator      | Operate model providers and graceful degradation            |
| US-031 | Experience    | Viewer            | Use the complete product on mobile in English or Portuguese |
| US-032 | Automation    | Automation Client | Use the HTTP API through an approved service identity       |

## Discovery and Access

### US-001: Browse the public catalog

**As a** Visitor, **I want** to see which public projects the service tracks, **so that** I can
decide whether to request access.

Acceptance criteria:

- AC-1: Given no authenticated session, when the Visitor opens the service, then only Project names,
  public descriptions, and public source links are visible.
- AC-2: Given a catalog with many Projects, when the Visitor searches or pages it, then matching
  public entries are returned without exposing derived intelligence.
- AC-3: Given a public Project entry, when the Visitor requests protected metrics or analyses, then
  the product invites authentication and does not disclose the protected response.

Edge cases:

- EC-1 `[Invalid input]`: Malformed search text is treated as text or rejected safely without query
  details.
- EC-2 `[Empty / missing]`: An empty catalog shows an onboarding explanation rather than a broken
  grid.
- EC-3 `[Limits]`: Large catalogs remain paginated and never return an unbounded response.
- EC-4 `[Permissions]`: Anonymous deep links to protected views reveal no protected fields.
- EC-5 `[Concurrency]`: A Project archived during browsing disappears on refresh without corrupting
  the page.
- EC-6 `[Interruption]`: A failed page request preserves the last visible page and offers retry.
- EC-7 `[Repetition]`: Repeating the same search returns the same public representation.
- EC-8 `[Ordering]`: Opening a stale bookmarked page resolves by stable identity or shows not found.
- EC-9 `[State transitions]`: Paused Projects remain public; archived and deleted Projects do not.
- EC-10 `[Scale]`: Search and pagination remain usable at 100 times the expected initial catalog.

### US-002: Authenticate and await approval

**As an** Applicant, **I want** to register through the shared identity service and understand my
access status, **so that** I know when I may use the workspace.

Acceptance criteria:

- AC-1: Given a Visitor chooses registration or sign-in, when shared Keycloak authentication
  succeeds, then the product identifies the external subject without creating product credentials.
- AC-2: Given an authenticated subject without membership, when the Applicant returns, then a
  pending approval page is shown and protected intelligence remains unavailable.
- AC-3: Given an Admin approves the Applicant, when the Applicant starts or refreshes a session,
  then the assigned local role and workspace become available.

Edge cases:

- EC-1 `[Invalid input]`: Invalid or unverifiable identity claims are rejected without creating
  membership.
- EC-2 `[Empty / missing]`: A valid identity with no local profile enters pending state, never an
  implicit role.
- EC-3 `[Limits]`: Registration and callback attempts are rate-limited with a clear retry message.
- EC-4 `[Permissions]`: An authenticated but pending Applicant cannot use protected APIs or exports.
- EC-5 `[Concurrency]`: Simultaneous first logins create one pending membership request.
- EC-6 `[Interruption]`: An interrupted authentication flow can restart without a partial product
  session.
- EC-7 `[Repetition]`: Repeated login preserves one pending request and does not notify Admins
  repeatedly.
- EC-8 `[Ordering]`: A callback without a matching login flow is rejected safely.
- EC-9 `[State transitions]`: A suspended or rejected identity cannot return to pending without
  Admin action.
- EC-10 `[Scale]`: A burst of applicants does not expose data or make approved sessions unavailable.

### US-003: Govern workspace membership

**As an** Admin, **I want** to approve applicants and assign fixed roles, **so that** only
authorized people can access or change the shared workspace.

Acceptance criteria:

- AC-1: Given pending applicants, when the Admin approves one, then exactly one Viewer, Analyst, or
  Admin role is assigned.
- AC-2: Given an existing member, when the Admin changes or suspends access, then subsequent
  requests enforce the new state and active access is revoked when required.
- AC-3: Given any membership action, when it completes, then actor, subject, prior state, new state,
  and outcome appear in the audit log.

Edge cases:

- EC-1 `[Invalid input]`: An unknown role or malformed subject is rejected without changing
  membership.
- EC-2 `[Empty / missing]`: With no pending users, the review view shows an empty state.
- EC-3 `[Limits]`: Membership lists are searchable and paginated at high user counts.
- EC-4 `[Permissions]`: Analysts and Viewers cannot approve, promote, suspend, or inspect applicant
  details.
- EC-5 `[Concurrency]`: Conflicting approvals use the latest valid state and report the stale
  action.
- EC-6 `[Interruption]`: A failed role change leaves the prior role effective and reports failure.
- EC-7 `[Repetition]`: Reapproving the same user is idempotent and does not create duplicate
  membership.
- EC-8 `[Ordering]`: A role cannot be assigned before the external subject is known.
- EC-9 `[State transitions]`: The last active Admin cannot remove or suspend their own Admin access.
- EC-10 `[Scale]`: Bulk applicant volume remains operable without bulk implicit approval.

### US-004: Manage preferences and remove an account

**As a** Viewer, **I want** to choose language and timezone and remove my account, **so that** the
product fits my context and respects my decision to leave.

Acceptance criteria:

- AC-1: Given an approved member, when English or Portuguese and a timezone are selected, then fixed
  product content and displayed times use those preferences.
- AC-2: Given the member requests account deletion and confirms it, then sessions and personal data
  are removed while shared resources remain.
- AC-3: Given historical actions by a deleted member, when an Admin audits them, then they reference
  a stable opaque actor without the removed profile data.

Edge cases:

- EC-1 `[Invalid input]`: Unsupported locales or timezones are rejected with supported choices.
- EC-2 `[Empty / missing]`: Missing preferences use English and UTC until the member selects
  alternatives.
- EC-3 `[Limits]`: Rapid preference changes are bounded and the latest confirmed value wins.
- EC-4 `[Permissions]`: A member cannot edit or delete another member's profile.
- EC-5 `[Concurrency]`: Concurrent deletion and profile edits resolve to deletion without restoring
  data.
- EC-6 `[Interruption]`: An interrupted deletion confirmation does not delete the account.
- EC-7 `[Repetition]`: Repeating deletion after success returns a non-sensitive completed outcome.
- EC-8 `[Ordering]`: Deletion cannot execute before explicit confirmation.
- EC-9 `[State transitions]`: A suspended member may remove their account but cannot regain
  workspace access.
- EC-10 `[Scale]`: Preference application does not require loading all workspace members.

## Portfolio and Projects

### US-005: Understand the portfolio overview

**As a** Viewer, **I want** one overview of the shared portfolio, **so that** I can identify
projects that need attention before drilling into details.

Acceptance criteria:

- AC-1: Given active Projects, when the overview loads, then it summarizes health dimensions,
  recommendations, alerts, trends, important releases, freshness, and failures.
- AC-2: Given a summary item, when the Viewer follows it, then the corresponding evidence-bearing
  Project, comparison, alert, trend, or radar view opens.
- AC-3: Given partial or stale data, when the overview renders, then coverage and freshness are
  visible and missing evidence is not displayed as zero.

Edge cases:

- EC-1 `[Invalid input]`: Invalid filters are rejected or reset with an explanation.
- EC-2 `[Empty / missing]`: A portfolio with no active Projects shows role-appropriate next steps.
- EC-3 `[Limits]`: Summary lists truncate predictably and link to paginated full views.
- EC-4 `[Permissions]`: Viewers see intelligence but no mutation controls.
- EC-5 `[Concurrency]`: New snapshots do not mix incompatible calculation versions in one rendered
  result.
- EC-6 `[Interruption]`: One failed panel does not erase deterministic panels that loaded
  successfully.
- EC-7 `[Repetition]`: Refreshing does not trigger collection or AI work implicitly.
- EC-8 `[Ordering]`: Deep links work without visiting the overview first.
- EC-9 `[State transitions]`: Archived Projects disappear from active summaries but remain available
  in archives.
- EC-10 `[Scale]`: The overview uses aggregation and pagination rather than rendering every Project
  at once.

### US-006: Register a project from a repository URL

**As an** Analyst, **I want** to register a Project from a supported repository URL, **so that** I
can begin tracking a technology with minimal setup.

Acceptance criteria:

- AC-1: Given a canonical public repository URL, when registration succeeds, then one Project and
  one typed primary Repository are created and initial collection is queued.
- AC-2: Given metadata is inferred, when the Project is shown, then the Analyst can review and edit
  the generated identity without changing source provenance.
- AC-3: Given the URL already belongs to a Project, when registration is attempted, then the
  existing Project opens instead of creating a duplicate.

Edge cases:

- EC-1 `[Invalid input]`: Malformed, unsupported, private, or hostile URLs are rejected with a safe
  reason.
- EC-2 `[Empty / missing]`: A blank URL cannot create a Project.
- EC-3 `[Limits]`: Operational quotas reject excess registration before creating partial state.
- EC-4 `[Permissions]`: Viewers and pending users cannot register Projects.
- EC-5 `[Concurrency]`: Concurrent registration of the same canonical URL creates one Project.
- EC-6 `[Interruption]`: Failure after creation shows recoverable collection state, not a duplicate
  prompt.
- EC-7 `[Repetition]`: Retrying a successful request resolves to the same Project.
- EC-8 `[Ordering]`: Additional repositories cannot be attached before their Project exists.
- EC-9 `[State transitions]`: A URL belonging to an archived Project offers restore rather than
  duplication.
- EC-10 `[Scale]`: Registration remains responsive while other Projects are backfilling.

### US-007: Curate a multi-repository project

**As an** Analyst, **I want** to attach and type multiple repositories, **so that** project-level
intelligence covers core code, documentation, examples, and SDKs correctly.

Acceptance criteria:

- AC-1: Given a Project, when repositories are added, then each has one explicit supported role and
  exactly one repository remains primary.
- AC-2: Given project-level metrics, when multiple repositories contribute, then the aggregation
  boundary and contributing repositories are visible.
- AC-3: Given a repository role changes, when recalculation is required, then affected intelligence
  becomes visibly stale until refreshed.

Edge cases:

- EC-1 `[Invalid input]`: Unsupported roles or duplicate canonical URLs are rejected.
- EC-2 `[Empty / missing]`: A Project cannot lose its only primary Repository without replacement.
- EC-3 `[Limits]`: Repository counts obey operational limits with no partial attachment.
- EC-4 `[Permissions]`: Viewers cannot add, remove, or retype repositories.
- EC-5 `[Concurrency]`: Concurrent primary changes result in exactly one primary Repository.
- EC-6 `[Interruption]`: An interrupted edit preserves the last valid repository set.
- EC-7 `[Repetition]`: Reattaching an existing repository returns its current association.
- EC-8 `[Ordering]`: A replacement primary is validated before the old primary loses its role.
- EC-9 `[State transitions]`: Repositories of archived Projects cannot be edited until restoration.
- EC-10 `[Scale]`: Projects with many repositories summarize them and provide pagination or
  filtering.

### US-008: Correct automatic source associations

**As an** Analyst, **I want** to inspect and correct automatically linked sources, **so that**
project intelligence does not combine unrelated evidence.

Acceptance criteria:

- AC-1: Given an automatic association, when inspected, then its source, resolution method,
  confidence, evidence, and decision version are visible.
- AC-2: Given an incorrect association, when the Analyst splits or reassigns it, then the correction
  is audited and retained against later automatic re-linking.
- AC-3: Given a correction invalidates derived results, when it completes, then affected metrics and
  analyses are marked stale and scheduled for recalculation.

Edge cases:

- EC-1 `[Invalid input]`: A target Project that cannot accept the source is rejected before
  reassignment.
- EC-2 `[Empty / missing]`: Associations without sufficient evidence remain visibly unresolved.
- EC-3 `[Limits]`: Large review queues are filterable and paginated.
- EC-4 `[Permissions]`: Viewers can inspect provenance but cannot correct associations.
- EC-5 `[Concurrency]`: Concurrent corrections detect stale source ownership and preserve one valid
  result.
- EC-6 `[Interruption]`: A failed correction leaves the prior association and derived status intact.
- EC-7 `[Repetition]`: Repeating the same correction does not enqueue duplicate recalculations.
- EC-8 `[Ordering]`: A split completes before downstream recalculation publishes new results.
- EC-9 `[State transitions]`: Deleted Projects cannot receive reassigned sources.
- EC-10 `[Scale]`: Corrections invalidate only affected evidence, not the entire workspace.

### US-009: Manage the project lifecycle

**As an** Admin, **I want** to pause, archive, restore, or permanently delete a Project, **so that**
collection and retention match the workspace's intent.

Acceptance criteria:

- AC-1: Given an active Project, when paused, then scheduled collection stops while existing
  intelligence remains readable.
- AC-2: Given a paused or active Project, when archived, then it becomes read-only and leaves
  default active views; restoration returns it to a valid non-deleted state.
- AC-3: Given explicit permanent-deletion confirmation, when deletion completes, then owned data is
  purged and only the minimal audit tombstone remains.

Edge cases:

- EC-1 `[Invalid input]`: Unknown lifecycle transitions are rejected with allowed actions.
- EC-2 `[Empty / missing]`: Deletion confirmation without the required Project identity does
  nothing.
- EC-3 `[Limits]`: Bulk lifecycle operations are not implied by a single-Project confirmation.
- EC-4 `[Permissions]`: Analysts and Viewers cannot pause, archive, restore, or permanently delete a
  Project; every lifecycle transition is Admin-only.
- EC-5 `[Concurrency]`: Collection racing with deletion cannot publish data after the deletion guard
  takes effect.
- EC-6 `[Interruption]`: A partial purge remains visibly in progress and resumes safely.
- EC-7 `[Repetition]`: Repeating pause or archive is idempotent; repeating deletion reveals no
  purged data.
- EC-8 `[Ordering]`: A Project cannot restore after permanent deletion.
- EC-9 `[State transitions]`: Archived Projects reject edits, sync requests, and analysis requests.
- EC-10 `[Scale]`: Purging one large Project does not block reading unrelated Projects.

## Collection and Sources

### US-010: Request and monitor synchronization

**As an** Analyst, **I want** automatic schedules and manual refresh with visible progress, **so
that** I know when intelligence is current.

Acceptance criteria:

- AC-1: Given an active Project, when its schedule is due, then collection begins without user
  action.
- AC-2: Given an authorized manual request, when accepted, then duplicate work is coalesced and the
  Project shows progress, last attempt, last success, next run, and any failure reason.
- AC-3: Given a transient failure or restart, when service resumes, then collection continues from a
  visible checkpoint without duplicating published data.

Edge cases:

- EC-1 `[Invalid input]`: Unsupported refresh scopes are rejected before work is queued.
- EC-2 `[Empty / missing]`: A source without a checkpoint begins its configured initial backfill.
- EC-3 `[Limits]`: Quota or concurrency exhaustion returns queued or delayed status, not silent
  loss.
- EC-4 `[Permissions]`: Viewers can inspect status but cannot request refresh.
- EC-5 `[Concurrency]`: Simultaneous refresh requests coalesce into one compatible run.
- EC-6 `[Interruption]`: Cancellation or restart preserves the last durable checkpoint.
- EC-7 `[Repetition]`: Replaying a completed collection does not duplicate canonical records.
- EC-8 `[Ordering]`: Analysis does not publish as current before required collection and metrics
  complete.
- EC-9 `[State transitions]`: Paused, archived, or deleting Projects reject new synchronization.
- EC-10 `[Scale]`: Workspace-wide backfills remain bounded and expose queue position or delay.

### US-011: Understand history and freshness

**As an** Analyst, **I want** to see and configure historical coverage, **so that** I do not mistake
partial history for a complete trend.

Acceptance criteria:

- AC-1: Given a new source, when no override exists, then the initial backfill target is 180 days
  and older still-open issues and pull requests are retained.
- AC-2: Given operator limits permit it, when an Analyst requests a longer range, then the requested
  target and eventual actual coverage are shown.
- AC-3: Given every metric or analysis, when viewed, then source coverage, last success, cutoff, and
  stale or incomplete state are visible.

Edge cases:

- EC-1 `[Invalid input]`: End dates before start dates or future-only ranges are rejected.
- EC-2 `[Empty / missing]`: No collected history yields insufficient data, never zero activity.
- EC-3 `[Limits]`: Requests beyond provider or operator limits show the maximum allowed range.
- EC-4 `[Permissions]`: Viewers cannot change backfill targets.
- EC-5 `[Concurrency]`: Overlapping range extensions coalesce into the broadest valid target.
- EC-6 `[Interruption]`: Partial backfill publishes its actual boundary and resumable state.
- EC-7 `[Repetition]`: Requesting an already covered range does not recollect it unnecessarily.
- EC-8 `[Ordering]`: A range extension does not rewrite older snapshots before data is available.
- EC-9 `[State transitions]`: Archived Projects retain coverage metadata without scheduling
  extension.
- EC-10 `[Scale]`: Long histories are summarized and paginated without truncating coverage
  disclosure.

### US-012: Operate public-data integrations

**As a** VPS Operator, **I want** source credentials and limits to remain outside end-user
workflows, **so that** public collection is reliable without exposing secrets or private content.

Acceptance criteria:

- AC-1: Given an operator-managed read token, when a source request executes, then only public
  resources are accepted even if the credential could see more.
- AC-2: Given an Admin inspects source health, when status loads, then provider, public capability,
  quota, and last validation appear without secret values.
- AC-3: Given a credential is missing or invalid, when collection is attempted, then affected
  sources degrade explicitly while unrelated deterministic intelligence remains available.

Edge cases:

- EC-1 `[Invalid input]`: Over-privileged or malformed credentials fail validation without echoing
  them.
- EC-2 `[Empty / missing]`: Missing optional credentials show anonymous capability or unavailable
  status.
- EC-3 `[Limits]`: Provider rate limits delay work with reset context rather than causing retry
  storms.
- EC-4 `[Permissions]`: End users cannot submit, retrieve, or select source credentials.
- EC-5 `[Concurrency]`: Credential rotation does not mix identities within one source request.
- EC-6 `[Interruption]`: Rotation failure preserves the last valid configuration or reports no
  active credential.
- EC-7 `[Repetition]`: Revalidation is safe and does not trigger collection.
- EC-8 `[Ordering]`: A token is validated before it becomes active.
- EC-9 `[State transitions]`: A source that becomes private stops collection and is marked
  unavailable.
- EC-10 `[Scale]`: Quota reporting aggregates safely across many Projects without leaking request
  details.

## Deterministic Intelligence

### US-013: Inspect metrics and health dimensions

**As a** Viewer, **I want** transparent metrics and independent health dimensions, **so that** I can
judge project health without relying on popularity or an opaque score.

Acceptance criteria:

- AC-1: Given a metric, when opened, then its value, formula version, unit, observation window,
  coverage, contributing sources, and missing-data rules are visible.
- AC-2: Given a Project, when health is shown, then Activity, Community, Maintenance, Concentration,
  Stability, Security, and Adoption remain independently inspectable.
- AC-3: Given an overall score that can be calculated with sufficient coverage, when shown, then it
  is secondary to dimensions and exposes weights, version, factors, window, and evidence without a
  binary healthy/unhealthy label.

Edge cases:

- EC-1 `[Invalid input]`: Unsupported metric windows are rejected with valid choices.
- EC-2 `[Empty / missing]`: Missing required evidence yields unavailable or insufficient data, never
  zero.
- EC-3 `[Limits]`: Large evidence sets summarize and link to bounded pages.
- EC-4 `[Permissions]`: Pending and anonymous users cannot read metric values.
- EC-5 `[Concurrency]`: Recalculation publishes one internally consistent metric-version snapshot.
- EC-6 `[Interruption]`: Failed recalculation leaves the prior snapshot visible and marked stale.
- EC-7 `[Repetition]`: Recalculating identical inputs and versions yields identical results.
- EC-8 `[Ordering]`: An overall score cannot publish before all required dimension results resolve.
- EC-9 `[State transitions]`: Archived Projects retain historical metrics but receive no new
  snapshots.
- EC-10 `[Scale]`: Time series remain usable at 100 times the initial history through aggregation.

### US-014: Evaluate contributor sustainability

**As a** Viewer, **I want** contributor activity and concentration with explainable identity
coverage, **so that** I can recognize community growth and maintainer dependency risk.

Acceptance criteria:

- AC-1: Given a selected window, when contributor intelligence loads, then active and new
  contributors, retention, maintainer count, and top-one/top-three contribution shares are shown.
- AC-2: Given cross-source identities, when contributors are aggregated, then only verified or
  Analyst-confirmed links combine accounts and the resolution coverage is visible.
- AC-3: Given an identity correction, when recalculation completes, then concentration metrics
  update without rewriting source provenance.

Edge cases:

- EC-1 `[Invalid input]`: Invalid contributor filters or windows are rejected safely.
- EC-2 `[Empty / missing]`: No contributor evidence yields insufficient data rather than total
  concentration.
- EC-3 `[Limits]`: Contributor lists paginate and do not expose private email addresses.
- EC-4 `[Permissions]`: Only Analysts can confirm or split contributor identities.
- EC-5 `[Concurrency]`: Concurrent identity corrections cannot leave one account linked twice.
- EC-6 `[Interruption]`: Failed recalculation leaves prior values stale rather than partially
  updated.
- EC-7 `[Repetition]`: Reconfirming the same verified link is idempotent.
- EC-8 `[Ordering]`: Identity linkage completes before aggregate concentration republishes.
- EC-9 `[State transitions]`: Deleted source accounts remain historical evidence with source status.
- EC-10 `[Scale]`: High-volume contributor histories use bounded detail without changing aggregates.

### US-015: Interpret adoption and security evidence

**As a** Viewer, **I want** source-contextual adoption and security signals, **so that** I can
consider usage and published risk without mistaking them for universal truth.

Acceptance criteria:

- AC-1: Given registry evidence, when adoption is shown, then raw values, time-window changes,
  provenance, coverage, and only within-population normalization are visible.
- AC-2: Given security evidence, when the dimension is shown, then public advisories, security
  releases, changelogs, issues, and provider metadata identify their sources and dates.
- AC-3: Given a policy or score uses either dimension, when inspected, then unavailable evidence and
  formula treatment are explicit.

Edge cases:

- EC-1 `[Invalid input]`: Incomparable registry units cannot be forced into one universal rank.
- EC-2 `[Empty / missing]`: No advisory or registry data is unknown, not evidence of safety or no
  adoption.
- EC-3 `[Limits]`: Large advisory and package histories are paginated and windowed.
- EC-4 `[Permissions]`: Protected adoption and security intelligence requires approved membership.
- EC-5 `[Concurrency]`: New registry data cannot mix cutoffs within one published snapshot.
- EC-6 `[Interruption]`: A failed registry or advisory source leaves other evidence visible with
  stale status.
- EC-7 `[Repetition]`: Reingestion of one advisory or package sample does not duplicate it.
- EC-8 `[Ordering]`: Normalization runs only after source population context is available.
- EC-9 `[State transitions]`: Withdrawn advisories retain provenance and display their withdrawn
  status.
- EC-10 `[Scale]`: Cross-registry portfolios retain source-specific context at high volume.

### US-016: Compare projects in one window

**As an** Analyst, **I want** to compare two to five Projects over the same interval, **so that** I
can evaluate alternatives without mismatched timeframes.

Acceptance criteria:

- AC-1: Given two to five Projects and a preset or valid custom window, when comparison runs, then
  all results use the same interval and cutoff.
- AC-2: Given differences in source coverage, when results render, then only comparable signals are
  normalized and unknown or not-applicable remains distinct from zero.
- AC-3: Given a compared value or narrative, when opened, then its metric version, evidence,
  freshness, and Project aggregation boundary are visible.

Edge cases:

- EC-1 `[Invalid input]`: Fewer than two, more than five, duplicates, or invalid windows are
  rejected.
- EC-2 `[Empty / missing]`: A Project with no comparable evidence remains in the view as
  insufficient data.
- EC-3 `[Limits]`: Evidence details paginate without truncating the comparison conclusion silently.
- EC-4 `[Permissions]`: Anonymous and pending users cannot run comparisons.
- EC-5 `[Concurrency]`: Metrics updated during a comparison do not create mixed cutoffs.
- EC-6 `[Interruption]`: Partial failure identifies unavailable Projects and preserves completed
  deterministic data.
- EC-7 `[Repetition]`: The same inputs and versions yield the same comparison.
- EC-8 `[Ordering]`: A comparison cannot run before every Project identity resolves.
- EC-9 `[State transitions]`: Deleted Projects make saved comparisons unavailable; archived Projects
  remain historical.
- EC-10 `[Scale]`: Many saved comparisons remain searchable and paginated.

### US-017: Distinguish trends and early warnings

**As an** Analyst, **I want** observed trends separated from predictive warnings, **so that** I
never mistake a forecast for measured history.

Acceptance criteria:

- AC-1: Given adequate history, when a trend is reported, then observation and baseline windows,
  method version, magnitude, direction, and evidence are visible.
- AC-2: Given a predictive warning, when opened, then it is labeled as forecast and shows horizon,
  confidence, calibration or known error, inputs, coverage, and model version.
- AC-3: Given inadequate evidence, when detection runs, then it returns insufficient data rather
  than a neutral or fabricated signal.

Edge cases:

- EC-1 `[Invalid input]`: Invalid baselines or forecast horizons are rejected.
- EC-2 `[Empty / missing]`: Sparse history produces insufficient data with the minimum requirement
  shown.
- EC-3 `[Limits]`: Signal histories paginate and bounded detectors respect configured horizons.
- EC-4 `[Permissions]`: Protected trends and warnings are unavailable to anonymous or pending users.
- EC-5 `[Concurrency]`: One signal version publishes against one consistent input snapshot.
- EC-6 `[Interruption]`: Failed prediction does not suppress valid observed trends.
- EC-7 `[Repetition]`: Deterministic trend reruns reproduce the same result from the same inputs.
- EC-8 `[Ordering]`: Explanations cannot publish before the underlying signal exists.
- EC-9 `[State transitions]`: Superseded warnings retain history and outcome-evaluation status.
- EC-10 `[Scale]`: Detection remains bounded across the portfolio and exposes delayed status.

## Decisions and Governance

### US-018: Receive an adoption recommendation

**As an** Analyst, **I want** a policy-driven adoption recommendation, **so that** organizational
risk tolerance is applied consistently and transparently.

Acceptance criteria:

- AC-1: Given a Project, policy version, and observation window, when evaluation completes, then the
  result is `recommended`, `conditional`, `not_recommended`, or `insufficient_data`.
- AC-2: Given any result, when inspected, then policy owner/version, inputs, weights, thresholds,
  decisive factors, evidence, freshness, and missing data are visible.
- AC-3: Given a conditional result, when displayed, then its constraints or mitigations are
  explicit; an LLM explanation cannot alter the outcome.

Edge cases:

- EC-1 `[Invalid input]`: Inactive or incompatible policy versions cannot be evaluated.
- EC-2 `[Empty / missing]`: Missing required evidence yields `insufficient_data`.
- EC-3 `[Limits]`: Evidence displays are bounded but retain links to every decisive input.
- EC-4 `[Permissions]`: Viewers may read results; only Analysts/Admins can select policies for new
  evaluations.
- EC-5 `[Concurrency]`: A policy activation during evaluation does not change the selected version.
- EC-6 `[Interruption]`: Failed explanation leaves the deterministic result available.
- EC-7 `[Repetition]`: Identical policy, inputs, and versions reproduce the outcome.
- EC-8 `[Ordering]`: Evaluation waits for required metric versions or reports stale prerequisites.
- EC-9 `[State transitions]`: Results retain their original policy version after supersession.
- EC-10 `[Scale]`: Portfolio evaluation queues remain bounded and expose progress.

### US-019: Author and version adoption policies

**As an** Admin, **I want** to clone, validate, publish, and retire policies, **so that** Analysts
use approved decision rules without rewriting history.

Acceptance criteria:

- AC-1: Given the transparent default policy, when cloned, then the Admin can change explicit
  thresholds, weights, required evidence, missing-data rules, and radar mapping in a draft.
- AC-2: Given a valid draft, when published, then it receives an immutable version and can be
  selected for new evaluations.
- AC-3: Given a published version is retired, when existing results are viewed, then they retain the
  retired version and remain reproducible.

Edge cases:

- EC-1 `[Invalid input]`: Contradictory outcomes, invalid weights, or unknown metrics block
  publication.
- EC-2 `[Empty / missing]`: Required evidence rules cannot be blank when an outcome depends on them.
- EC-3 `[Limits]`: Policy lists and version histories are paginated.
- EC-4 `[Permissions]`: Analysts may select but cannot create, modify, publish, or retire policies.
- EC-5 `[Concurrency]`: Concurrent edits detect stale drafts instead of overwriting changes.
- EC-6 `[Interruption]`: Failed publication leaves the draft editable and no partial version active.
- EC-7 `[Repetition]`: Repeating publication cannot create two versions from one draft state.
- EC-8 `[Ordering]`: A draft must validate before publication or activation.
- EC-9 `[State transitions]`: Published versions are immutable; retired versions cannot reactivate
  silently.
- EC-10 `[Scale]`: Many historical versions remain searchable without loading all definitions.

### US-020: Govern the technology radar

**As an** Analyst, **I want** a policy-derived radar with explicit human overrides, **so that** the
workspace can communicate current technology decisions without hiding disagreement.

Acceptance criteria:

- AC-1: Given a selected Project and recommendation, when added to the radar, then the suggested
  ring follows the visible mapping of the exact policy version.
- AC-2: Given organizational context differs, when an Analyst overrides the ring, then
  justification, author, owner, and review date are required and the original recommendation remains
  visible.
- AC-3: Given policy evidence changes, when the radar is viewed, then stale suggestions and overdue
  overrides are called out without moving a manual override silently.

Edge cases:

- EC-1 `[Invalid input]`: Unknown rings, past-invalid review dates, or blank override reasons are
  rejected.
- EC-2 `[Empty / missing]`: `insufficient_data` maps only according to an explicit policy mapping.
- EC-3 `[Limits]`: Large radars filter and group Projects without hiding off-screen counts.
- EC-4 `[Permissions]`: Viewers read the radar but cannot select, override, or annotate Projects.
- EC-5 `[Concurrency]`: Concurrent movement detects stale radar state.
- EC-6 `[Interruption]`: A failed override preserves the prior ring and recommendation.
- EC-7 `[Repetition]`: Reapplying the same selection or override is idempotent.
- EC-8 `[Ordering]`: A Project needs a policy result before receiving a suggested ring.
- EC-9 `[State transitions]`: Archived Projects leave the active radar but retain historical
  placement.
- EC-10 `[Scale]`: Radar calculation remains bounded when many Projects receive updated
  recommendations.

## Qualitative and AI Intelligence

### US-021: Explore issue and discussion topics

**As an** Analyst, **I want** topics across issues and supported discussions with corrections, **so
that** I can see which community problems are growing or changing.

Acceptance criteria:

- AC-1: Given a time window, when topic intelligence runs, then known taxonomy categories and
  emerging topics show prevalence, change, representative evidence, confidence, and analysis
  version.
- AC-2: Given a wrong grouping, when the Analyst renames, merges, splits, or reassigns it, then the
  correction is attributed and available to later reprocessing and evaluation.
- AC-3: Given source coverage differs, when topics are compared over time or across Projects, then
  the contributing content and coverage are explicit.

Edge cases:

- EC-1 `[Invalid input]`: Empty names, circular merges, or unsupported assignments are rejected.
- EC-2 `[Empty / missing]`: No eligible content yields insufficient data, not zero topic prevalence.
- EC-3 `[Limits]`: Large topic and evidence sets paginate and cap displayed examples transparently.
- EC-4 `[Permissions]`: Viewers cannot correct classifications.
- EC-5 `[Concurrency]`: Concurrent corrections detect stale topic versions.
- EC-6 `[Interruption]`: Failed reprocessing leaves the prior version visible and stale.
- EC-7 `[Repetition]`: Repeating a correction does not duplicate feedback.
- EC-8 `[Ordering]`: Trend calculations wait for a complete topic version.
- EC-9 `[State transitions]`: Retired topics remain in historical results but not new assignments.
- EC-10 `[Scale]`: Clustering is bounded and exposes queued or sampled status when needed.

### US-022: Understand a release

**As a** Viewer, **I want** structured release intelligence with source evidence, **so that** I can
understand real features, breaking changes, deprecations, security fixes, performance, and DX
changes.

Acceptance criteria:

- AC-1: Given a collected release, when analysis succeeds, then changes use known categories and
  each claim links to changelog, referenced pull request, diff metadata, issue, or other source
  evidence.
- AC-2: Given multiple analysis runs, when viewing the release, then the presented version and its
  model, prompt, execution time, language, and status are visible.
- AC-3: Given no AI provider, when the release opens, then deterministic release metadata remains
  available and analysis shows unavailable without blocking collection.

Edge cases:

- EC-1 `[Invalid input]`: Malformed provider output cannot publish as a valid structured analysis.
- EC-2 `[Empty / missing]`: A release without changelog shows limited evidence rather than invented
  changes.
- EC-3 `[Limits]`: Large changelogs and evidence sets are bounded with disclosed truncation.
- EC-4 `[Permissions]`: Anonymous visitors cannot read protected release analysis.
- EC-5 `[Concurrency]`: Concurrent reruns create distinct immutable versions.
- EC-6 `[Interruption]`: Failed analysis retains deterministic metadata and prior successful
  versions.
- EC-7 `[Repetition]`: Replaying one run request does not overwrite or duplicate the same execution
  identity.
- EC-8 `[Ordering]`: Analysis does not claim evidence before eligible inputs are collected.
- EC-9 `[State transitions]`: Withdrawn releases retain their source status and historical analysis.
- EC-10 `[Scale]`: Release lists and runs paginate across long histories.

### US-023: Search project documentation

**As an** Analyst, **I want** evidence-backed search across explicitly linked documentation, **so
that** I can connect project claims and changes to maintained source material.

Acceptance criteria:

- AC-1: Given linked public URLs, when collected, then only allowed domains, depth, size, frequency,
  and content types are followed, with robots behavior honored.
- AC-2: Given a search or RAG answer, when results appear, then every claim links to the original
  URL and snapshot time; translated text is labeled and the original remains accessible.
- AC-3: Given a page changes, when recollected, then a new provenance-bearing snapshot supports
  later analysis without silently rewriting past evidence.

Edge cases:

- EC-1 `[Invalid input]`: Unsupported schemes, unsafe addresses, and out-of-scope domains are
  rejected.
- EC-2 `[Empty / missing]`: No indexed documentation yields an explicit no-evidence response.
- EC-3 `[Limits]`: Crawl depth, bytes, pages, and request rate stop predictably with visible
  coverage.
- EC-4 `[Permissions]`: Only Analysts can configure crawl scope; approved users may search.
- EC-5 `[Concurrency]`: Duplicate crawl requests coalesce by source and snapshot target.
- EC-6 `[Interruption]`: Partial crawls expose their boundary and resume without duplicating
  snapshots.
- EC-7 `[Repetition]`: Unchanged content does not create misleading duplicate versions.
- EC-8 `[Ordering]`: Search indexes only validated snapshots.
- EC-9 `[State transitions]`: Removed URLs remain historical evidence but leave current search after
  refresh.
- EC-10 `[Scale]`: Search remains bounded and identifies sampling or truncation at large corpus
  sizes.

### US-024: Ask natural-language questions

**As an** Analyst, **I want** to ask questions about selected Projects and time windows, **so that**
I can investigate intelligence without manually assembling every view.

Acceptance criteria:

- AC-1: Given a question, when interpreted, then the response identifies scope, Projects, window,
  data cutoff, structured findings, and citations to product evidence.
- AC-2: Given ambiguous scope, when the question could produce materially different answers, then
  the product requests clarification rather than guessing.
- AC-3: Given missing or stale evidence, when answering, then uncertainty or insufficient data is
  stated and unsupported claims are omitted.
- AC-4: Given the request implies a supported mutation, then it becomes a typed HITL proposal
  governed by US-025 rather than executing directly.

Edge cases:

- EC-1 `[Invalid input]`: Hostile or unsupported requests cannot escape the bounded
  analytical/action catalog.
- EC-2 `[Empty / missing]`: Blank questions or empty evidence return actionable guidance, not
  fabricated answers.
- EC-3 `[Limits]`: Token, query, and evidence limits show truncation and refinement guidance.
- EC-4 `[Permissions]`: The assistant cannot retrieve evidence the requesting user cannot access.
- EC-5 `[Concurrency]`: Evidence updates during a response do not mix data cutoffs silently.
- EC-6 `[Interruption]`: A canceled query stops generation and preserves no partial action approval.
- EC-7 `[Repetition]`: Repeating a question identifies its current cutoff rather than implying
  timeless identity.
- EC-8 `[Ordering]`: Clarification must resolve before analysis or action proposal continues.
- EC-9 `[State transitions]`: Questions about deleted Projects return unavailable without leaked
  history.
- EC-10 `[Scale]`: Broad questions are narrowed or bounded rather than scanning the workspace
  without limit.

### US-025: Approve a non-destructive assistant action

**As an** Analyst, **I want** to review and approve one exact assistant-proposed action, **so that**
I retain control over conversational mutations.

Acceptance criteria:

- AC-1: Given a supported request, when the assistant proposes an action, then operation, resources,
  values, expected effect, quota impact, and expiration are displayed before execution.
- AC-2: Given explicit approval of an unchanged proposal, when execution succeeds, then the result
  and audit entry identify the requesting Analyst and proposal.
- AC-3: Given an Admin-only, credential, policy-authoring, archive, deletion, or otherwise
  destructive request, then the assistant refuses execution and points to the conventional surface.

Edge cases:

- EC-1 `[Invalid input]`: Untyped or unsupported proposal fields cannot reach execution.
- EC-2 `[Empty / missing]`: A proposal without every required value asks for clarification.
- EC-3 `[Limits]`: Quota-exceeding proposals show the limit and cannot be approved into a hidden
  queue.
- EC-4 `[Permissions]`: Approval rechecks the current user's role and resource access.
- EC-5 `[Concurrency]`: Changed resource state invalidates the preview and requires a new proposal.
- EC-6 `[Interruption]`: Lost connection or expired proposal executes nothing.
- EC-7 `[Repetition]`: Replaying one approval cannot repeat the mutation.
- EC-8 `[Ordering]`: Approval before preview or after expiration is rejected.
- EC-9 `[State transitions]`: Actions targeting paused, archived, or deleted resources obey
  lifecycle rules.
- EC-10 `[Scale]`: A request containing many actions is split into atomic approvals or rejected as
  too broad.

### US-026: Review AI analysis versions

**As an** Analyst, **I want** to inspect, rerun, flag, and select immutable AI outputs, **so that**
probabilistic intelligence remains auditable and correctable.

Acceptance criteria:

- AC-1: Given an analysis run, when opened, then structured output, evidence, provider/model, prompt
  version, language, execution time, status, and stale state are visible.
- AC-2: Given a rerun, when it finishes, then it creates a new immutable version without overwriting
  previous output.
- AC-3: Given inaccurate output, when an Analyst flags it, then attributed feedback is stored and a
  different version may be selected without editing generated content.

Edge cases:

- EC-1 `[Invalid input]`: Feedback without a target version or reason is rejected.
- EC-2 `[Empty / missing]`: No successful run shows unavailable analysis and eligible next actions.
- EC-3 `[Limits]`: Version and evidence histories paginate; reruns obey quotas.
- EC-4 `[Permissions]`: Viewers inspect but cannot rerun, flag, or select versions.
- EC-5 `[Concurrency]`: Concurrent selection detects stale current-version state.
- EC-6 `[Interruption]`: Interrupted runs remain terminally labeled and never appear successful.
- EC-7 `[Repetition]`: Duplicate feedback from one request is idempotent.
- EC-8 `[Ordering]`: A failed or incomplete run cannot be selected as presented output.
- EC-9 `[State transitions]`: Stale selected runs remain visible with a stale warning until
  replaced.
- EC-10 `[Scale]`: Retained versions remain discoverable without loading every output at once.

## Alerts, Export, and Operations

### US-027: Configure and resolve shared alerts

**As an** Analyst, **I want** shared in-app alert rules and individually readable occurrences, **so
that** the team can respond to significant releases, security evidence, health changes, and trends.

Acceptance criteria:

- AC-1: Given a valid rule, when its evidence condition is met, then one deduplicated occurrence
  shows severity, rule version, Project, window, evidence, and detected time.
- AC-2: Given an occurrence, when one user reads it, then only that user's read state changes; when
  an Analyst acknowledges, resolves, or dismisses it with justification, the shared state changes.
- AC-3: Given repeated evidence inside cooldown, when evaluated, then it updates or suppresses the
  existing occurrence according to the visible rule rather than creating noise.

Edge cases:

- EC-1 `[Invalid input]`: Rules with unknown signals, invalid thresholds, or negative cooldowns are
  rejected.
- EC-2 `[Empty / missing]`: Missing required evidence cannot trigger an alert.
- EC-3 `[Limits]`: Alert lists paginate and rule/occurrence volume respects workspace quotas.
- EC-4 `[Permissions]`: Viewers manage only personal read state; Analysts/Admins manage shared
  resolution.
- EC-5 `[Concurrency]`: Concurrent resolution uses one final state and identifies stale actions.
- EC-6 `[Interruption]`: Evaluation failure does not close or duplicate an existing occurrence.
- EC-7 `[Repetition]`: Replayed signals preserve one deduplicated occurrence.
- EC-8 `[Ordering]`: Resolution cannot precede occurrence creation; reopening follows explicit
  rules.
- EC-9 `[State transitions]`: Archived Projects stop new alert evaluation while history remains
  readable.
- EC-10 `[Scale]`: High event volume is bounded without losing severity and suppression counts.

### US-028: Export results and evidence

**As a** Viewer, **I want** CSV data and JSON evidence packages, **so that** I can analyze or audit
a decision outside the product.

Acceptance criteria:

- AC-1: Given a metrics, snapshot, or comparison view, when CSV is requested, then exported rows use
  the selected scope, window, cutoff, units, and missing-data representation.
- AC-2: Given an evidence-package request, when JSON is produced, then it includes scope, coverage,
  provenance, formulas, versions, policy context, and analysis references needed to interpret it.
- AC-3: Given English or Portuguese preference, when export is generated, then stable machine fields
  remain documented and human labels follow the requested language where supported.

Edge cases:

- EC-1 `[Invalid input]`: Unsupported formats or malformed scopes are rejected before generation.
- EC-2 `[Empty / missing]`: Empty result sets produce a valid empty export with scope metadata.
- EC-3 `[Limits]`: Oversized exports are rejected or generated asynchronously with visible limits
  and status.
- EC-4 `[Permissions]`: Exports contain only data visible to the requesting approved member.
- EC-5 `[Concurrency]`: Export uses one explicit data cutoff despite ongoing updates.
- EC-6 `[Interruption]`: Interrupted generation can be retried without duplicate completed
  artifacts.
- EC-7 `[Repetition]`: Identical requests identify equivalent scope and cutoff without changing
  data.
- EC-8 `[Ordering]`: A download is unavailable until generation completes successfully.
- EC-9 `[State transitions]`: A completed download expires after 24 hours, and Project deletion
  removes owned generated exports earlier.
- EC-10 `[Scale]`: Large exports do not exhaust interactive product capacity.

### US-029: Investigate the audit log

**As an** Admin, **I want** a searchable and exportable audit trail, **so that** I can explain
access, governance, corrections, synchronization, AI, and HITL actions.

Acceptance criteria:

- AC-1: Given an auditable action, when recorded, then actor, time, resource, action, prior/new
  state where safe, and outcome are present without secrets or sensitive payloads.
- AC-2: Given filters for actor, resource, action, outcome, or time, when applied, then matching
  events are returned in stable chronological order.
- AC-3: Given a deleted user or Project, when history is viewed, then only the defined opaque actor
  or Project tombstone remains.

Edge cases:

- EC-1 `[Invalid input]`: Invalid ranges or filters are rejected without arbitrary query execution.
- EC-2 `[Empty / missing]`: No matches produce a clear empty state.
- EC-3 `[Limits]`: Reads and exports paginate or bound results without dropping event counts
  silently.
- EC-4 `[Permissions]`: Only Admins can inspect or export audit history.
- EC-5 `[Concurrency]`: Simultaneous actions retain distinct event identities and deterministic
  ordering ties.
- EC-6 `[Interruption]`: A failed business action records failure without claiming a state change.
- EC-7 `[Repetition]`: Idempotent retries remain attributable without fabricating duplicate success.
- EC-8 `[Ordering]`: Audit ordering uses event time and stable tie-break identity.
- EC-9 `[State transitions]`: Audit events remain immutable after subject deletion or role changes.
- EC-10 `[Scale]`: Long retention remains searchable through bounded pages and time filters.

### US-030: Operate model providers and degradation

**As a** VPS Operator, **I want** shared model-provider configuration and visible degradation, **so
that** AI capabilities remain controllable without destabilizing deterministic intelligence.

Acceptance criteria:

- AC-1: Given an operator-configured external or local provider, when available, then approved AI
  capabilities use the workspace-active model configuration consistently.
- AC-2: Given an Admin inspects AI status, when the status loads, then active provider/model,
  supported capabilities, health, and aggregate usage/cost are visible without secrets.
- AC-3: Given provider failure or no provider, when users open the product, then collection,
  metrics, health, policies, radar, comparisons, and deterministic trends continue while AI surfaces
  explain their unavailable or stale state.

Edge cases:

- EC-1 `[Invalid input]`: Unsupported model capabilities or malformed configuration fail validation
  safely.
- EC-2 `[Empty / missing]`: No provider is a valid degraded state, not an application startup
  failure.
- EC-3 `[Limits]`: Model quotas delay or reject AI work with visible status and no retry storm.
- EC-4 `[Permissions]`: Users cannot submit or retrieve provider secrets; only Admins see redacted
  status.
- EC-5 `[Concurrency]`: Configuration changes do not alter provider identity within an active run.
- EC-6 `[Interruption]`: Interrupted runs become failed or canceled versions and never block
  deterministic work.
- EC-7 `[Repetition]`: Retried requests create attributable runs without overwriting successful
  history.
- EC-8 `[Ordering]`: New configuration validates before becoming active for later runs.
- EC-9 `[State transitions]`: Disabled providers make dependent features unavailable without
  deleting prior outputs.
- EC-10 `[Scale]`: Global model concurrency and usage limits protect interactive deterministic
  traffic.

### US-031: Use the bilingual mobile-first product

**As a** Viewer, **I want** every product journey to work on a phone in English or Portuguese, **so
that** I can inspect and operate the service without a desktop or language barrier.

Acceptance criteria:

- AC-1: Given a supported mobile viewport, when any Viewer, Analyst, or Admin journey is opened,
  then its information and controls remain complete without requiring a desktop-only fallback.
- AC-2: Given a language preference, when navigating, then fixed UI, validation, status, errors, and
  product-generated analysis use that language while source evidence remains available in original
  form.
- AC-3: Given charts or color-coded states, when used with keyboard, screen reader, zoom, or reduced
  motion settings, then equivalent text/table information and non-color cues remain available.

Edge cases:

- EC-1 `[Invalid input]`: Unsupported locale input falls back to English and remains changeable.
- EC-2 `[Empty / missing]`: Missing translation never produces blank controls or hidden errors.
- EC-3 `[Limits]`: Dense tables and charts use responsive summaries and bounded detail rather than
  data loss.
- EC-4 `[Permissions]`: Responsive layouts never reveal controls or data hidden from the role.
- EC-5 `[Concurrency]`: Language changes during a save do not repeat or lose the action.
- EC-6 `[Interruption]`: Navigation or connection loss preserves recoverable form state where safe.
- EC-7 `[Repetition]`: Repeated locale switching does not alter stored evidence or calculations.
- EC-8 `[Ordering]`: Deep links apply membership, language, and timezone before protected content
  renders.
- EC-9 `[State transitions]`: Disabled actions remain labeled with their lifecycle reason at narrow
  widths.
- EC-10 `[Scale]`: Large localized strings and 200% zoom remain usable without clipping essential
  actions.

### US-032: Use the API through a service identity

**As an** Automation Client, **I want** to call permitted HTTP API routes with a Keycloak service
identity, **so that** approved integrations can read intelligence or perform bounded Analyst work
without impersonating a person.

Acceptance criteria:

- AC-1: Given a valid Keycloak bearer token, when its subject matches an approved local service
  account, then the API enforces that account's Viewer or Analyst role and scope subset.
- AC-2: Given an authorized API request, when it completes, then the audit event attributes the
  action to the service account and never to a fabricated human actor.
- AC-3: Given a local service account is suspended or its scope changes, when a later bearer request
  arrives, then the current local state is enforced without requiring Keycloak credential mutation.

Edge cases:

- EC-1 `[Invalid input]`: A malformed, expired, wrong-issuer, wrong-audience, or unverifiable token
  is rejected without creating a local account.
- EC-2 `[Empty / missing]`: A valid Keycloak subject without a local binding receives no implicit
  access.
- EC-3 `[Limits]`: Service-account requests use the applicable account, workspace, source, and Job
  quotas.
- EC-4 `[Permissions]`: A service account cannot receive Admin, approve members, change policies,
  manage credentials, or perform Project lifecycle actions.
- EC-5 `[Concurrency]`: Concurrent scope changes and requests authorize against one committed local
  account version.
- EC-6 `[Interruption]`: An interrupted idempotent action can retry without duplicating the resource
  or Job.
- EC-7 `[Repetition]`: Repeated bearer requests remain individually attributable while idempotency
  preserves one business outcome.
- EC-8 `[Ordering]`: Local suspension takes effect before any request authorized against the later
  account version.
- EC-9 `[State transitions]`: Deleting or suspending the local binding prevents access even while
  the external Keycloak token remains valid.
- EC-10 `[Scale]`: Many service-account calls remain isolated by identity and scope without
  exhausting interactive member capacity.
