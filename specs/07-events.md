# Events and Asynchronous Work

Every durable command/event carries an event ID, schema version, kind, occurred-at timestamp,
correlation/causation IDs, actor/project scope when applicable, Job/outbox identity, and a payload
that contains references rather than credentials or raw evidence.

Business transactions create/update the PostgreSQL Job and append an outbox row atomically. The
publisher uses the outbox ID as JetStream deduplication identity. Durable pull consumers use explicit
acknowledgements, bounded batches, heartbeats, lease/checkpoint persistence, retry backoff, and
idempotent handlers. Delivery is at least once; PostgreSQL Job/checkpoint state decides truth.

SSE is a resumable projection of authoritative events. `Last-Event-ID` resumes within retention;
clients fall back to bounded Job reads after gaps. Valkey fanout loss may delay live presentation but
cannot change a Job or authorization result.
