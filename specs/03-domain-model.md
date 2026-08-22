# Domain Model

The canonical model is provider-neutral. A `Project` is the aggregate identity and may own multiple
`Repository`, `Source`, and evidence associations. Provider DTOs end at adapters before entering a
business package.

Core aggregates include members/service accounts, projects/repositories/sources, jobs/checkpoints,
canonical contributions/issues/pull requests/releases, metric snapshots, comparisons, policies and
radar entries, topic/release/knowledge analyses, alerts, exports, evidence objects, and audit events.
Each persisted resource uses a Snowflake identifier and an aggregate version where conditional
mutation applies.

Invariants:

- project lifecycle controls collection and publication; deletion uses an attributed resumable
  purge and cannot be reversed after canonical removal;
- Jobs and checkpoints are PostgreSQL state machines and terminal states never regress;
- normalized facts and analysis runs are immutable/versioned; corrections create attributed new
  versions rather than rewriting provenance;
- unknown, unavailable, insufficient, and not-applicable are distinct states and never numeric zero;
- identity is external issuer plus subject, while roles and service-account scopes are local;
- every probabilistic claim points to immutable accessible evidence and execution metadata.

The complete aggregates, state transitions, ports, and acceptance rules are normative in
`01-product-requirements.md`, `02-user-stories.md`, and `06-api-contracts.md`.
