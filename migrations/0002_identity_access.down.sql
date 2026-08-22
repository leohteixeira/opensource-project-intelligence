DROP TRIGGER IF EXISTS audit_events_immutable_update ON audit_events;
DROP FUNCTION IF EXISTS reject_audit_event_mutation();

ALTER TABLE audit_events
    DROP COLUMN IF EXISTS changes,
    DROP COLUMN IF EXISTS request_id,
    DROP COLUMN IF EXISTS outcome,
    DROP COLUMN IF EXISTS actor_kind;

DROP TABLE IF EXISTS public_catalog_projects;
DROP TABLE IF EXISTS oidc_login_flows;
DROP TABLE IF EXISTS browser_sessions;
DROP TABLE IF EXISTS service_account_scopes;
DROP TABLE IF EXISTS service_accounts;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS external_identities;
DROP TABLE IF EXISTS workspaces;

