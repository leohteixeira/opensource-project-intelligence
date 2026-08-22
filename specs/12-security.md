# Security

The shared Keycloak deployment authenticates OIDC identities; issuer plus subject is the key. Local
membership, roles, suspension, account versions, and service-account scopes authorize every request.
External role-like claims never grant product access. The last active Admin invariant is atomic.

Secrets are operator-owned environment/secret-store inputs, read once, never returned by product
APIs, never written in plaintext, and redacted from all diagnostics. Public URL collection applies
scheme/host validation, DNS and redirect revalidation, private/link-local denial, response limits,
and safe Git transport settings. SQL is parameterized and generated adapters do not expose query
construction to callers.

State-changing HTTP uses authorization, idempotency, conditional versions where specified, bounded
input, safe problem details, and attributable audit. Evidence/object access follows project and
retention ownership. AI tools are allowlisted and cannot mutate credentials, membership, policies,
archive, or deletion state.
