---
status: pending
title: Implement identity, local access governance, audit, and the localized application shell
type: fullstack
complexity: high
---

<!-- markdownlint-disable MD013 MD025 -->

# Task 2: Implement identity, local access governance, audit, and the localized application shell

## Overview

Deliver the complete public, Applicant, approved-member, service-identity, and Admin access model,
including same-origin browser sessions, Keycloak bearer validation, local authorization, membership
governance, preferences, account deletion, audit visibility, operations status, and the bilingual
responsive application shell.

<critical>
- ALWAYS READ `_spec.md` and its catalogs (`_user_stories.md`, `_dx.md`, `_uiux.md` when present, `_tests.md`) before starting
- REFERENCE `_spec.md` Part II for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — implement every test case assigned in ## Tests
</critical>

<requirements>

- Implement US-001 through US-004, US-031, and US-032 in full, including every accepted edge case.
- Consume workspace Keycloak identity while keeping membership, roles, service-account scopes,
  suspension, sessions, and authorization local and authoritative.
- Enforce Visitor, Applicant, Viewer, Analyst, Admin, and approved service-account boundaries before
  any protected disclosure or mutation.
- Implement secure PKCE login/callback, opaque revocable cookies, CSRF/origin protections, bearer
  validation, conditional writes, idempotency, signed cursor behavior, and immutable redacted audit.
- Replace the temporary shell selector with localized route-aware anonymous, pending, member, and
  Admin shells backed only by the generated HTTP client.
- Preserve the imported design system and four UI kits as the visual contract; replace fixtures only
  at the application boundary and do not duplicate shared primitives.

</requirements>

## Visual Contract

| Surface              | Viewport/state contract                                                                                                                                               | Reference                                      | Required evidence                |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- | -------------------------------- |
| S1 application shell | Narrow 320px: initial check, anonymous, pending, each approved role, suspended, offline, expired session, unauthorized/not-found, keyboard order, en/pt-BR, 200% zoom | AppShell and four Kit shells                   | artifacts/task_02/ui/s1-narrow/  |
| S1 application shell | Wide: the same states with the 64px top bar, role-aware navigation, skip link, restored URL state, and reduced motion                                                 | AppShell and four Kit shells                   | artifacts/task_02/ui/s1-wide/    |
| S2 public catalog    | Narrow 320px: populated, empty, search/no-match, cursor page, retained stale page, archived removal, protected deep link, and scale                                   | CatalogScreen and TeaserScreen                 | artifacts/task_02/ui/s2-narrow/  |
| S2 public catalog    | Wide: the same states with accessible tiles, numbered cursor navigation, teaser evidence, and sign-in handoff                                                         | CatalogScreen and TeaserScreen                 | artifacts/task_02/ui/s2-wide/    |
| S3 access/account    | Narrow 320px: redirect, callback failure, pending/rejected/suspended/approved, preferences conflict, exact deletion, deletion Job, and last-Admin guard               | AccessScreen and WorkspaceKit account controls | artifacts/task_02/ui/s3-narrow/  |
| S3 access/account    | Wide: the same states with complete dialogs, focus return, timezone/locale feedback, and no unauthorized controls                                                     | AccessScreen and WorkspaceKit account controls | artifacts/task_02/ui/s3-wide/    |
| S23 administration   | Narrow 320px: applicant/member/service-account states, audit empty/filter/tombstone, redacted operations healthy/degraded/unavailable, and partial failure            | AdministrationKit screens                      | artifacts/task_02/ui/s23-narrow/ |
| S23 administration   | Wide: the same states with paginated tables, stale-action handling, attributable detail, aggregate status, and keyboard operation                                     | AdministrationKit screens                      | artifacts/task_02/ui/s23-wide/   |

## Subtasks

- [ ] Implement local member, Applicant, session, service-account, role/scope, preference, audit, and
      account-deletion domain/application behavior.
- [ ] Implement controlled OIDC/PKCE and bearer adapters without calling the Keycloak Admin API or
      inferring authorization from email/token role claims.
- [ ] Implement the public catalog/session/member/service-account/audit/operations HTTP operations
      assigned to this task.
- [ ] Add the localized router, session gate, generated client provider, URL-state conventions,
      responsive navigation, account surfaces, and Admin access surfaces.
- [ ] Supply complete en and pt-BR dictionaries, stable localized paths, accessibility behavior,
      error recovery, and role-appropriate control omission.
- [ ] Implement all assigned unit, integration, and Playwright/axe cases.

## Implementation Details

### Relevant Files

- \_spec.md Part II sections for identity, HTTP, security, frontend, and persistence
- \_user_stories.md US-001 through US-004, US-031, and US-032
- \_dx.md authentication, roles, public discovery, membership, administration, and errors
- \_uiux.md S1, S2, S3, and S23
- adrs/adr-001.md, adr-004.md, adr-015.md, adr-017.md, adr-018.md, adr-021.md,
  adr-023.md, adr-024.md, adr-028.md, and adr-031.md
- apps/web/src/design-system/, apps/web/src/kits/, apps/web/src/App.tsx, and apps/web/src/main.tsx
- cmd/api/main.go and internal/platform/httpx/

### Dependent Files

- Task 03 and every later protected capability consume the principal/session contract, generated
  client, authorization middleware, audit append port, localized router, and application shell.

### Related ADRs

- [Workflow ADR 001](adrs/adr-001.md), [ADR 004](adrs/adr-004.md),
  [ADR 015](adrs/adr-015.md), [ADR 017](adrs/adr-017.md),
  [ADR 018](adrs/adr-018.md), [ADR 021](adrs/adr-021.md),
  [ADR 023](adrs/adr-023.md), [ADR 024](adrs/adr-024.md),
  [ADR 028](adrs/adr-028.md), and [ADR 031](adrs/adr-031.md).
- Repository ADRs 0002, 0004, 0005, 0006, and 0007.

## Deliverables

- Secure browser and automation authentication with immediate local revocation.
- Fixed local roles/scopes, applicant approval, membership/service-account governance, and
  last-Admin protection.
- Public catalog, pending access, account/preferences/deletion, audit, and redacted operations
  routes and screens.
- Bilingual mobile-first routed shell preserving query state and accessibility semantics.
- Passing evidence for all 106 assigned tests and every visual-contract row.

## Tests

Implement these normative cases from \_tests.md exactly once:

- Unit: UT-001, UT-002, UT-003, UT-004, UT-005, UT-006, UT-007, UT-008, UT-009, UT-010, UT-011, UT-012, UT-013, UT-014, UT-015, UT-016, UT-017, UT-018, UT-019, UT-020, UT-021, UT-022, UT-023, UT-024, UT-025, UT-026, UT-027, UT-028, UT-211, UT-212, UT-213, UT-214, UT-215, UT-216, UT-217, UT-218, UT-219, UT-220, UT-221, UT-222, UT-223, UT-224, UT-247, UT-248, UT-249, UT-250, UT-272
- Integration: IT-001, IT-002, IT-003, IT-004, IT-005, IT-006, IT-007, IT-008, IT-009, IT-010, IT-011, IT-012, IT-091, IT-092, IT-093, IT-094, IT-095, IT-096, IT-100, IT-101, IT-102, IT-147, IT-148, IT-149, IT-150, IT-151, IT-152, IT-153, IT-154, IT-155, IT-156, IT-157, IT-158, IT-159, IT-160, IT-161, IT-162, IT-163, IT-164, IT-165, IT-166, IT-167, IT-168, IT-169, IT-170, IT-171, IT-172, IT-173, IT-174
- End-to-end: E2E-001, E2E-002, E2E-003, E2E-004, E2E-031, E2E-032, E2E-033, E2E-034, E2E-035, E2E-055

## Success Criteria

- [ ] No protected value or action is disclosed before current local authorization succeeds.
- [ ] Browser sessions, bearer identities, suspension, service scopes, CSRF/origin checks,
      idempotency, ETags, and cursors satisfy the frozen HTTP contract.
- [ ] Every S1, S2, S3, and S23 state works in en and pt-BR at narrow and wide viewports with
      keyboard-only and axe validation.
- [ ] Audit and operations surfaces never expose credentials or sensitive payloads.
- [ ] All 106 assigned tests pass with fresh evidence.
