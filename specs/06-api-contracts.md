# Developer Experience: Open Source Project Intelligence

Public-surface contract for Open Source Project Intelligence. Part II of `_spec.md` must serve this
surface, and `_tests.md` must use these exact routes, values, status codes, and error shapes.

The shipped product has three public surfaces:

1. a bilingual browser application for Visitors, Applicants, Viewers, Analysts, and Admins;
2. a same-origin HTTP/JSON API under `/api/v1`; and
3. deployment environment settings and health routes for the VPS Operator.

There is no product CLI, public SDK, GraphQL endpoint, native agent tool, or user-authored YAML
file. Automation uses the documented HTTP contract with a Keycloak bearer token and an approved
local member or service-account record. Shared Keycloak provisioning remains outside this
repository.

## Golden Path

An Analyst opens `https://opensource.example.com`, selects **Sign in**, and is redirected to the
workspace-shared Keycloak. After authentication and local Admin approval, the Analyst registers a
Project from its primary public repository:

```http
POST /api/v1/projects HTTP/1.1
Host: opensource.example.com
Content-Type: application/json
Accept-Language: en
Idempotency-Key: example-idempotency-key
X-CSRF-Token: iVtEFpVjTtm9hG6gqK1D0w

{
  "repository_url": "https://github.com/temporalio/temporal",
  "history_days": 180
}
```

```http
HTTP/1.1 202 Accepted
Content-Type: application/json
Location: /api/v1/jobs/732684512948649984
ETag: "1"

{
  "project": {
    "id": "732684512931872768",
    "name": "Temporal",
    "slug": "temporal",
    "description": "Durable execution platform",
    "state": "active",
    "primary_repository": {
      "id": "732684512940261376",
      "provider": "github",
      "role": "core",
      "url": "https://github.com/temporalio/temporal"
    }
  },
  "job": {
    "id": "732684512948649984",
    "kind": "initial_sync",
    "state": "queued",
    "progress": {"completed": 0, "total": 8, "unit": "sources"},
    "links": {"self": "/api/v1/jobs/732684512948649984"}
  }
}
```

The browser subscribes to the Job event stream and uses the returned Job URL as its polling
fallback. A restart does not change the job identity or lose completed checkpoints:

```http
GET /api/v1/jobs/732684512948649984 HTTP/1.1
Host: opensource.example.com
```

```json
{
  "id": "732684512948649984",
  "kind": "initial_sync",
  "state": "running",
  "project_id": "732684512931872768",
  "progress": { "completed": 5, "total": 8, "unit": "sources" },
  "started_at": "2026-08-20T14:31:08Z",
  "updated_at": "2026-08-20T14:34:12Z",
  "checkpoint": "github:pull_requests:2026-03-14T00:00:00Z",
  "coalesced_requests": 0,
  "links": { "project": "/api/v1/projects/732684512931872768" }
}
```

The preferred browser transport emits the same representation rather than a partial patch:

```http
GET /api/v1/jobs/732684512948649984/events HTTP/1.1
Host: opensource.example.com
Accept: text/event-stream
Last-Event-ID: 732684513191870464
```

```http
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache
X-Accel-Buffering: no

id: 732684513200259072
event: job.updated
data: {"id":"732684512948649984","kind":"initial_sync","state":"running","project_id":"732684512931872768","progress":{"completed":6,"total":8,"unit":"sources"},"updated_at":"2026-08-20T14:34:18Z"}

```

When deterministic intelligence is ready, the Analyst obtains a policy recommendation with the
evidence and missing data shown rather than hidden:

```http
GET /api/v1/projects/732684512931872768/recommendation?policy=default&window=90d HTTP/1.1
Host: opensource.example.com
```

```json
{
  "project_id": "732684512931872768",
  "result": "conditional",
  "policy": { "id": "732684512965427200", "name": "Default adoption policy", "version": 4 },
  "window": { "from": "2026-05-22", "to": "2026-08-19", "cutoff": "2026-08-20T14:35:00Z" },
  "conditions": ["Review top-three contributor concentration before production adoption."],
  "inputs": [
    { "metric": "release_frequency", "status": "available", "value": 8, "unit": "releases" },
    { "metric": "top_three_author_share", "status": "available", "value": 0.71, "unit": "ratio" },
    { "metric": "nuget_download_change", "status": "not_applicable" }
  ],
  "evidence_links": [
    "/api/v1/projects/732684512931872768/metrics/release_frequency?window=90d",
    "/api/v1/projects/732684512931872768/metrics/top_three_author_share?window=90d"
  ]
}
```

## HTTP Contract

### Transport and representation

- Production uses HTTPS and one public origin. The reverse proxy serves the browser and forwards
  `/api/`, `/auth/`, `/health`, and `/ready` to the API. Browser code never points at `localhost` in
  a remote deployment.
- JSON requests use `Content-Type: application/json`; JSON responses use
  `application/json; charset=utf-8`. Errors use `application/problem+json`.
- Resource identifiers are 64-bit Snowflake IDs serialized as decimal JSON strings, such as
  `"732684512931872768"`, and as decimal URL path segments. String serialization prevents precision
  loss in JavaScript. Clients compare them as opaque strings and do not derive authorization,
  timestamps, ordering, or resource type from their bit layout.
- Timestamps are RFC 3339 UTC. Calendar windows use inclusive ISO dates and an explicit cutoff.
- Monetary values, ratios, scores, and metric values are JSON numbers accompanied by a unit and
  definition version. `unknown`, `not_applicable`, and `insufficient_data` are statuses, never
  numeric zero or `null` stand-ins.
- Collection responses use the envelope `{"items": [...], "page": {...}}`. The page object contains
  `next_cursor` and `has_more`; clients treat the cursor as opaque. Default page size is 50 and the
  maximum is 200. The browser retains the cursor chain for the current query and displays numbered
  previous/current/next pages; changing filters resets that page history.
- `Accept-Language: en` or `pt-BR` localizes fixed human-facing labels and `detail`. Stable codes,
  identifiers, metric names, evidence, and source content remain unchanged. Generated translations
  identify their source language and translation status.
- Every response includes `X-Request-ID`. Rate-limited responses also include `Retry-After`.

### Authentication, browser sessions, and automation

The browser uses a same-origin, Secure, HttpOnly, SameSite=Lax session cookie. Login uses Keycloak
OpenID Connect Authorization Code Flow with PKCE. The product never receives, stores, or renders a
password. State-changing browser requests require the current CSRF token returned by the session
resource. Localized browser routes begin with `/en` or `/pt-br`; the API remains `/api/v1` in both
languages.

```http
GET /auth/login?return_to=%2Fen%2Fprojects HTTP/1.1
Host: opensource.example.com
```

```http
HTTP/1.1 303 See Other
Location: https://auth.example.com/realms/intelligence/protocol/openid-connect/auth?client_id=opensource-project-intelligence&response_type=code&scope=openid%20profile%20email&redirect_uri=https%3A%2F%2Fopensource.example.com%2Fauth%2Fcallback&state=8aSCmPC7Vws&code_challenge=ypXJ5yR7QtU&code_challenge_method=S256
```

After Keycloak redirects to `/auth/callback`, the API validates the flow, creates or refreshes the
local Applicant/member record, sets the session cookie, and returns `303` to the validated local
`return_to` path. External return URLs are rejected.

```http
GET /api/v1/session HTTP/1.1
Host: opensource.example.com
```

```json
{
  "authenticated": true,
  "access": "approved",
  "member": {
    "id": "732684512957038592",
    "display_name": "Ana Silva",
    "role": "analyst",
    "locale": "pt-BR",
    "timezone": "America/Sao_Paulo"
  },
  "csrf_token": "iVtEFpVjTtm9hG6gqK1D0w"
}
```

Pending access returns the same resource with `access: "pending"` and no role. Suspended and
rejected identities return `access: "suspended"` or `"rejected"` and cannot access protected routes.
`POST /api/v1/session/logout` invalidates the product session and returns `204 No Content`.

Non-browser clients send `Authorization: Bearer <access-token>`. The API validates the Keycloak
issuer, signature, expiry, and audience, then resolves the token subject to an approved local member
or service account. Bearer requests do not use cookies or CSRF tokens. A human bearer token receives
the same current local role as that person's browser session.

Service accounts are created in shared Keycloak by the VPS Operator and bound locally by an Admin.
They may receive `viewer` or `analyst`, never `admin`, plus a subset of named scopes. Local
suspension revokes product access without mutating Keycloak. Every request is audited against the
service account rather than an impersonated person.

### Concurrency, idempotency, and asynchronous work

- Every state-changing resource representation has an `ETag`. Updates and lifecycle transitions
  require `If-Match`; a stale value returns `412 version_conflict` with the current ETag.
- Create/action requests require `Idempotency-Key`. Repeating the same key and body returns the
  original status and representation. Reusing the key with a different body returns
  `409 idempotency_key_reused`.
- Long-running collection, recalculation, analysis, crawl, export, and purge requests return
  `202 Accepted`, a `Location` header, and a durable Job resource.
- Compatible work for the same Project and parameters coalesces into one active Job. The response
  returns `200 OK`, `X-Job-Coalesced: true`, and the existing Job.
- Jobs have `queued`, `running`, `succeeded`, `failed`, or `cancelled` state. Progress is factual;
  an unknown total uses `total_status: "unknown"` rather than a fabricated percentage.
- Analysts may cancel queued or running sync, reanalysis, crawl, query, and export Jobs. Admin purge
  Jobs cannot be cancelled after deletion confirmation.
- Browsers prefer `GET /api/v1/jobs/{job_id}/events`, a Server-Sent Events stream. Each
  `job.updated` event carries the complete current Job representation and an event ID that can be
  resumed through `Last-Event-ID`. A terminal update closes the stream. Clients fall back to polling
  `GET /api/v1/jobs/{job_id}`; active polling responses return `Retry-After: 2`. No WebSocket
  contract is shipped.

### Roles and lifecycle actions

| Capability                                                    | Visitor | Applicant | Viewer | Analyst | Admin  |
| ------------------------------------------------------------- | ------- | --------- | ------ | ------- | ------ |
| Public catalog                                                | read    | read      | read   | read    | read   |
| Protected intelligence and exports                            | —       | —         | read   | read    | read   |
| Project/repository/source curation                            | —       | —         | —      | change  | change |
| Sync, reanalysis, corrections, radar annotations, alert rules | —       | —         | —      | change  | change |
| Pause, archive, restore, permanent deletion                   | —       | —         | —      | —       | change |
| Membership, roles, policies, audit, operator status           | —       | —         | —      | —       | govern |

An unavailable action is omitted from API discovery links and from the UI. Backend authorization is
authoritative; hiding a control never substitutes for permission checks.

## Route Catalog

The tables below are the complete version-1 route inventory. `page` means the standard cursor
envelope. `Job` means the durable asynchronous representation defined above. Every protected route
may additionally return `401 authentication_required`, `403 access_pending`,
`403 permission_denied`, `429 rate_limited`, and `500 internal_error`.

### Public discovery and account

| Method and path                                  | Request                                             | Success                                                      |
| ------------------------------------------------ | --------------------------------------------------- | ------------------------------------------------------------ |
| `GET /api/v1/catalog/projects?q=&cursor=&limit=` | none                                                | `200` page of `{id,name,slug,description,source_links}`      |
| `GET /api/v1/catalog/projects/{project_id}`      | none                                                | `200` public catalog representation; `404 project_not_found` |
| `GET /api/v1/session`                            | session cookie optional                             | `200` session/access representation                          |
| `POST /api/v1/session/logout`                    | CSRF token                                          | `204`                                                        |
| `PATCH /api/v1/me/preferences`                   | `{"locale":"pt-BR","timezone":"America/Sao_Paulo"}` | `200` member plus new ETag                                   |
| `POST /api/v1/me/deletion`                       | `{"confirmation":"DELETE MY ACCOUNT"}`              | `202` Job                                                    |

### Membership and administration

| Method and path                                                             | Request                                                                                                                      | Success                                                                           |
| --------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| `GET /api/v1/admin/members?state=&role=&q=&cursor=&limit=`                  | none                                                                                                                         | `200` page of members/applicants                                                  |
| `POST /api/v1/admin/members/{member_id}/approval`                           | `{"decision":"approve","role":"viewer"}` or `{"decision":"reject"}`                                                          | `200` member plus ETag                                                            |
| `PATCH /api/v1/admin/members/{member_id}`                                   | `{"role":"analyst"}` or `{"state":"suspended"}` with If-Match                                                                | `200` member plus ETag                                                            |
| `GET /api/v1/admin/service-accounts?state=&q=&cursor=&limit=`               | none                                                                                                                         | `200` page of locally bound service accounts; no token or secret fields           |
| `POST /api/v1/admin/service-accounts`                                       | `{"external_subject":"opi-exporter","name":"Portfolio exporter","role":"viewer","scopes":["projects:read","exports:write"]}` | `201` local service account plus ETag                                             |
| `PATCH /api/v1/admin/service-accounts/{service_account_id}`                 | role, scope subset, or state with If-Match                                                                                   | `200` service account plus ETag                                                   |
| `GET /api/v1/admin/audit?actor=&action=&resource=&from=&to=&cursor=&limit=` | none                                                                                                                         | `200` page of immutable audit events                                              |
| `GET /api/v1/admin/operations`                                              | none                                                                                                                         | `200` redacted source/model capability, quota, health, and aggregate usage status |

The last active Admin cannot be suspended, demoted, or deleted. Such a request returns
`409 last_admin_required`.

### Portfolio, Projects, repositories, and sources

| Method and path                                                               | Request                                                                          | Success                                                                    |
| ----------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `GET /api/v1/portfolio?window=90d&cutoff=`                                    | none                                                                             | `200` panel summaries with independent status and evidence links           |
| `GET /api/v1/projects?state=active&q=&cursor=&limit=`                         | none                                                                             | `200` page of protected Project summaries                                  |
| `POST /api/v1/projects`                                                       | `{"repository_url":"https://github.com/temporalio/temporal","history_days":180}` | `202` Project and initial-sync Job                                         |
| `GET /api/v1/projects/{project_id}`                                           | none                                                                             | `200` Project, links, capabilities, freshness summary, ETag                |
| `PATCH /api/v1/projects/{project_id}`                                         | editable identity fields with If-Match                                           | `200` Project plus ETag                                                    |
| `POST /api/v1/projects/{project_id}/transition`                               | `{"to":"paused","reason":"Maintenance review"}` with If-Match                    | `202` Project and transition Job                                           |
| `POST /api/v1/projects/{project_id}/deletion`                                 | `{"confirmation":"DELETE temporal","reason":"Duplicate project"}` with If-Match  | `202` non-cancellable purge Job                                            |
| `GET /api/v1/projects/{project_id}/repositories?cursor=&limit=`               | none                                                                             | `200` page of Repository resources                                         |
| `POST /api/v1/projects/{project_id}/repositories`                             | `{"url":"https://github.com/temporalio/sdk-go","role":"sdk"}`                    | `201` Repository plus ETag                                                 |
| `PATCH /api/v1/projects/{project_id}/repositories/{repository_id}`            | `{"role":"primary"}` with If-Match                                               | `200` Repository set with exactly one primary                              |
| `DELETE /api/v1/projects/{project_id}/repositories/{repository_id}`           | If-Match                                                                         | `204`; rejects removal of only primary                                     |
| `GET /api/v1/projects/{project_id}/sources?kind=&state=&cursor=&limit=`       | none                                                                             | `200` page of source coverage/status resources                             |
| `POST /api/v1/projects/{project_id}/sources`                                  | typed public URL/package/feed/document source                                    | `201` Source plus ETag                                                     |
| `PATCH /api/v1/projects/{project_id}/sources/{source_id}`                     | editable scope and collection limits with If-Match                               | `200` Source plus ETag                                                     |
| `DELETE /api/v1/projects/{project_id}/sources/{source_id}`                    | If-Match                                                                         | `202` recalculation Job                                                    |
| `GET /api/v1/projects/{project_id}/associations?status=&cursor=&limit=`       | none                                                                             | `200` page with method, confidence, evidence, decision version, correction |
| `POST /api/v1/projects/{project_id}/associations/{association_id}/correction` | `{"action":"split","reason":"Different product"}`                                | `202` correction and recalculation Job                                     |

Supported repository roles are `primary`, `core`, `documentation`, `examples`, `sdk`, and `other`.
Supported built-in source kinds are `github`, `gitlab`, `gitea`, `git`, `npm`, `nuget`, `pypi`,
`documentation`, `website`, `changelog`, `rss`, `discussion`, and `advisory`.

### Jobs, history, and synchronization

| Method and path                                                      | Request                                               | Success                                                       |
| -------------------------------------------------------------------- | ----------------------------------------------------- | ------------------------------------------------------------- |
| `POST /api/v1/projects/{project_id}/syncs`                           | `{"scope":"all"}` or selected source IDs              | `202` new Job or `200` coalesced Job                          |
| `POST /api/v1/projects/{project_id}/history-requests`                | `{"from":"2025-08-20","reason":"Annual review"}`      | `202` Job; quota rejection is atomic                          |
| `GET /api/v1/projects/{project_id}/jobs?kind=&state=&cursor=&limit=` | none                                                  | `200` page of Jobs                                            |
| `GET /api/v1/jobs/{job_id}`                                          | none                                                  | `200` Job; active Jobs include `Retry-After`                  |
| `GET /api/v1/jobs/{job_id}/events`                                   | `Accept: text/event-stream`, optional `Last-Event-ID` | `200` resumable `job.updated` SSE stream until terminal state |
| `POST /api/v1/jobs/{job_id}/cancellation`                            | `{"reason":"No longer needed"}`                       | `202` Job moving to cancelled                                 |

Source status reports `last_attempt_at`, `last_success_at`, `next_run_at`, `coverage_from`,
`coverage_to`, `freshness`, `quota`, `failure`, and checkpoint-backed progress. A credential is
never returned, even in an Admin response.

### Intelligence and comparison

| Method and path                                                                          | Request                                                                                                      | Success                                                                          |
| ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------- |
| `GET /api/v1/projects/{project_id}/metrics?dimension=&window=90d&cutoff=&cursor=&limit=` | none                                                                                                         | `200` page of versioned metric results                                           |
| `GET /api/v1/projects/{project_id}/metrics/{metric_name}?window=90d&cutoff=`             | none                                                                                                         | `200` formula, value/status, unit, evidence, repositories, coverage, version     |
| `GET /api/v1/projects/{project_id}/health?window=90d&cutoff=`                            | none                                                                                                         | `200` seven dimensions and a visible secondary overall score whenever calculable |
| `GET /api/v1/projects/{project_id}/contributors?window=90d&cursor=&limit=`               | none                                                                                                         | `200` page plus resolution coverage and concentration summary                    |
| `GET /api/v1/projects/{project_id}/adoption?window=90d&cursor=&limit=`                   | none                                                                                                         | `200` source-contextual indicators; no universal score                           |
| `GET /api/v1/projects/{project_id}/security?window=365d&cursor=&limit=`                  | none                                                                                                         | `200` public-evidence findings and explicit coverage limitations                 |
| `POST /api/v1/comparisons`                                                               | `{"project_ids":["732684512931872768","732684513124761600"],"window":"90d","cutoff":"2026-08-20T14:35:00Z"}` | `201` immutable comparison                                                       |
| `GET /api/v1/comparisons/{comparison_id}`                                                | none                                                                                                         | `200` same-window matrix with incomparable/missing states preserved              |
| `GET /api/v1/projects/{project_id}/trends?kind=observed&window=365d&cursor=&limit=`      | none                                                                                                         | `200` page of observed trends or early warnings                                  |
| `GET /api/v1/projects/{project_id}/recommendation?policy=default&window=90d&cutoff=`     | none                                                                                                         | `200` one four-state deterministic evaluation                                    |

Accepted windows are `30d`, `90d`, `180d`, `365d`, or `from=YYYY-MM-DD&to=YYYY-MM-DD`. A custom
window outside actual coverage returns results with `insufficient_data`; it does not silently
shrink.

### Policies and radar

| Method and path                                                   | Request                                                                  | Success                                                                 |
| ----------------------------------------------------------------- | ------------------------------------------------------------------------ | ----------------------------------------------------------------------- |
| `GET /api/v1/policies?state=&cursor=&limit=`                      | none                                                                     | `200` page of policy families and active versions                       |
| `GET /api/v1/policies/{policy_id}/versions/{version}`             | none                                                                     | `200` immutable rule tree and explanation labels                        |
| `POST /api/v1/policies`                                           | `{"name":"Production adoption","description":"...","rules":[...]}`       | `201` draft policy version 1                                            |
| `POST /api/v1/policies/{policy_id}/versions`                      | complete next rule tree                                                  | `201` immutable draft version                                           |
| `POST /api/v1/policies/{policy_id}/versions/{version}/activation` | `{"reason":"Quarterly governance update"}`                               | `200` active version; prior evaluations remain attributed               |
| `GET /api/v1/radar?policy=default&window=90d`                     | none                                                                     | `200` Project placements with policy suggestion and effective placement |
| `POST /api/v1/radar/{project_id}/override`                        | `{"ring":"assess","reason":"Pilot dependency","review_on":"2026-11-20"}` | `201` attributed override                                               |
| `DELETE /api/v1/radar/{project_id}/override`                      | If-Match                                                                 | `204`; policy placement becomes effective                               |

Policy rules may reference only catalogued metrics and explicit operators. A rule using an unknown
metric returns `422 unknown_metric`; arbitrary expressions or code are not accepted.

### Topics, releases, documentation, and AI

| Method and path                                                      | Request                                                                           | Success                                                                  |
| -------------------------------------------------------------------- | --------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| `GET /api/v1/projects/{project_id}/topics?window=90d&cursor=&limit=` | none                                                                              | `200` known/emerging topics with evidence, confidence, coverage, version |
| `POST /api/v1/projects/{project_id}/topics/{topic_id}/corrections`   | rename, merge, split, or reassign plus reason                                     | `202` correction and reanalysis Job                                      |
| `GET /api/v1/projects/{project_id}/releases?cursor=&limit=`          | none                                                                              | `200` page of releases and analysis status                               |
| `GET /api/v1/projects/{project_id}/releases/{release_id}`            | none                                                                              | `200` categorized claims linked to evidence                              |
| `POST /api/v1/projects/{project_id}/crawls`                          | `{"source_ids":["732684513351254016"],"max_depth":3}`                             | `202` crawl Job                                                          |
| `POST /api/v1/projects/{project_id}/knowledge/search`                | `{"query":"How are upgrades handled?","language":"en","limit":10}`                | `200` cited snapshot results                                             |
| `POST /api/v1/projects/{project_id}/queries`                         | `{"question":"What changed in maintenance risk?","window":"90d","language":"en"}` | `202` analysis Job and Run                                               |
| `GET /api/v1/analysis-runs/{run_id}`                                 | none                                                                              | `200` immutable structured output, evidence, versions, usage, status     |
| `POST /api/v1/analysis-runs/{run_id}/reruns`                         | `{"language":"pt-BR","reason":"Review corrected topic"}`                          | `202` new Run and Job                                                    |
| `POST /api/v1/analysis-runs/{run_id}/feedback`                       | `{"rating":"incorrect","comment":"The cited issue belongs to the SDK."}`          | `201` immutable feedback                                                 |
| `POST /api/v1/analysis-series/{series_id}/selection`                 | `{"run_id":"732684513258979328"}`                                                 | `200` selected successful version plus ETag                              |

The original evidence language is retained. A generated translation returns
`{"source_language":"en","language":"pt-BR","translation":"generated"}`. Provider outage returns
`503 ai_provider_unavailable` only for AI-dependent routes; deterministic routes remain available.

### Assistant actions

| Method and path                                               | Request                                   | Success                                    |
| ------------------------------------------------------------- | ----------------------------------------- | ------------------------------------------ |
| `POST /api/v1/assistant/proposals`                            | message plus `Idempotency-Key`            | `201` typed proposal awaiting confirmation |
| `POST /api/v1/assistant/proposals/{proposal_id}/confirmation` | confirmation token plus `Idempotency-Key` | `201` exact proposal execution receipt     |

```http
POST /api/v1/assistant/proposals HTTP/1.1
Content-Type: application/json
Idempotency-Key: example-idempotency-key

{"message":"Add the Go SDK repository to Temporal and sync it."}
```

```json
{
  "id": "732684512982204416",
  "status": "awaiting_confirmation",
  "action": "repository.add",
  "inputs": {
    "project_id": "732684512931872768",
    "url": "https://github.com/temporalio/sdk-go",
    "role": "sdk"
  },
  "effect": "Adds one public Repository. Synchronization is a separate action.",
  "quota": { "registration_remaining": 42 },
  "expires_at": "2026-08-20T14:50:00Z",
  "confirmation_token": "example-confirmation-token"
}
```

```http
POST /api/v1/assistant/proposals/732684512982204416/confirmation HTTP/1.1
Content-Type: application/json
Idempotency-Key: example-idempotency-key

{"confirmation_token":"example-confirmation-token"}
```

```http
HTTP/1.1 201 Created
Content-Type: application/json

{
  "proposal_id": "732684512982204416",
  "status": "executed",
  "result": {"repository_id":"732684513049313280"},
  "audit_event_id": "732684512998981632"
}
```

`POST /api/v1/assistant/proposals` returns `422 action_not_allowed` for membership, role,
credential, policy-definition, archive, deletion, or other Admin/destructive actions. A confirmation
is single-use, action-specific, expires after ten minutes, and rechecks permission and resource
version.

### Alerts and exports

| Method and path                                     | Request                                                    | Success                                                                      |
| --------------------------------------------------- | ---------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `GET /api/v1/alerts?state=&project=&cursor=&limit=` | none                                                       | `200` page of shared alert events plus requesting member's read state        |
| `POST /api/v1/alert-rules`                          | typed condition, scope, severity, and deduplication window | `201` rule plus ETag                                                         |
| `PATCH /api/v1/alert-rules/{rule_id}`               | complete editable fields with If-Match                     | `200` rule plus ETag                                                         |
| `POST /api/v1/alerts/{alert_id}/read`               | none                                                       | `204`; changes only current member state                                     |
| `POST /api/v1/alerts/{alert_id}/transition`         | `{"to":"acknowledged","reason":"Investigating"}`           | `200` shared alert plus ETag                                                 |
| `POST /api/v1/exports`                              | resource, format `csv` or `evidence_json`, filters, locale | `202` export Job                                                             |
| `GET /api/v1/exports/{export_id}`                   | none                                                       | `200` metadata and download URL that expires 24 hours after the Job succeeds |
| `GET /api/v1/exports/{export_id}/download`          | none                                                       | `200` file; `410 export_expired` after expiry                                |

## Deployment Environment

The VPS Operator supplies non-secret settings directly and secrets through the deployment's secret
injection mechanism. The application fails startup when a required setting is absent or an origin is
insecure in production.

```text
ENVIRONMENT=production
PUBLIC_BASE_URL=https://opensource.example.com
HTTP_ADDRESS=0.0.0.0:8100
DATABASE_URL=postgres://opensource@postgres:5432/opensource?sslmode=require
KEYCLOAK_ISSUER_URL=https://auth.example.com/realms/intelligence
KEYCLOAK_CLIENT_ID=opensource-project-intelligence
KEYCLOAK_CLIENT_SECRET_FILE=/run/secrets/opensource-keycloak-client-secret
SESSION_KEY_FILE=/run/secrets/opensource-session-key
DEFAULT_LOCALE=en
SUPPORTED_LOCALES=en,pt-BR
DEFAULT_TIMEZONE=UTC
WORKER_CONCURRENCY=8
SOURCE_REQUEST_CONCURRENCY=16
DEFAULT_HISTORY_DAYS=180
MAX_HISTORY_DAYS=1825
MAX_PROJECTS=0
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
```

- `PUBLIC_BASE_URL` controls same-origin links and the exact Keycloak redirect origin.
- Keycloak settings identify the externally managed OIDC client; this product does not create a
  realm, client, user, mail flow, MFA policy, or recovery policy.
- `MAX_PROJECTS=0` means no configured Project-count ceiling. Request, crawl, storage, and provider
  quotas still apply and are visible through the Admin operations view.
- Provider read credentials and model-provider credentials use provider-specific `*_FILE` settings;
  their values never appear in application responses or telemetry.
- No product CLI or credential-management endpoint is shipped. Operators use the deployment's
  secret/configuration surface; approved Admins use `/api/v1/admin/operations` for redacted runtime
  truth.

The liveness and readiness routes stay outside `/api/v1`:

```http
GET /health HTTP/1.1
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok"}
```

```http
GET /ready HTTP/1.1
```

```http
HTTP/1.1 503 Service Unavailable
Content-Type: application/json

{"status":"not_ready","dependencies":{"postgresql":"unavailable"}}
```

Readiness covers only dependencies required for deterministic operation. Optional source and model
providers appear in `/api/v1/admin/operations` and do not make the whole service unready.

## Errors

All API failures use one deterministic Problem Details extension:

```http
HTTP/1.1 422 Unprocessable Entity
Content-Type: application/problem+json
X-Request-ID: req_TkRhQwW7KjvPGf6B0a2J9Q

{
  "type": "https://opensource.example.com/problems/unsafe-source-url",
  "title": "Source URL is not allowed",
  "status": 422,
  "code": "unsafe_source_url",
  "detail": "The source resolves to a private network address.",
  "instance": "/api/v1/projects",
  "request_id": "req_TkRhQwW7KjvPGf6B0a2J9Q",
  "errors": [{"field":"repository_url","code":"private_network_target"}]
}
```

| Status | Stable code               | Condition                                                   | Client action                                  |
| ------ | ------------------------- | ----------------------------------------------------------- | ---------------------------------------------- |
| `400`  | `invalid_request`         | Malformed JSON, cursor, filter, or confirmation             | Correct the named input                        |
| `401`  | `authentication_required` | No valid product session                                    | Start `/auth/login`                            |
| `403`  | `access_pending`          | Authenticated Applicant lacks approved membership           | Show approval status                           |
| `403`  | `permission_denied`       | Role cannot perform action                                  | Remove the unavailable action                  |
| `404`  | `resource_not_found`      | Resource does not exist or is not visible                   | Stop polling or return to list                 |
| `409`  | `duplicate_resource`      | Canonical source already belongs to a Project               | Follow `existing_resource` link                |
| `409`  | `idempotency_key_reused`  | Same key used for a different request                       | Generate a new key                             |
| `409`  | `last_admin_required`     | Action would leave no active Admin                          | Approve/promote another Admin first            |
| `410`  | `resource_expired`        | Export or assistant confirmation expired                    | Request a new resource                         |
| `412`  | `version_conflict`        | `If-Match` is stale                                         | Fetch latest state and review again            |
| `422`  | `unsafe_source_url`       | URL violates scheme, DNS/IP, redirect, or local-path policy | Supply an eligible public source               |
| `422`  | `insufficient_data`       | Requested calculation cannot produce a result               | Inspect coverage and request history           |
| `422`  | `action_not_allowed`      | Assistant proposed a prohibited action                      | Perform an eligible action manually            |
| `429`  | `rate_limited`            | User, source, or workspace quota reached                    | Honor `Retry-After` and inspect quota          |
| `503`  | `ai_provider_unavailable` | Only the requested AI feature lacks a provider              | Retry later; deterministic data remains usable |
| `503`  | `dependency_unavailable`  | Required persistence unavailable                            | Retry after readiness recovers                 |
| `500`  | `internal_error`          | Unexpected failure                                          | Report `request_id`; no internal detail leaks  |

Validation errors list stable field codes. Localized `title` and `detail` never replace `code` as
the automation contract. Responses never include stack traces, SQL, credentials, authorization
headers, provider payloads containing private fields, or model prompts containing sensitive session
data.
