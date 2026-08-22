# Test Contract: Open Source Project Intelligence

This catalog is normative for the complete Task 001–Task N delivery described by `_spec.md`. IDs are
stable: implementations may add subcases, but they must not renumber or silently remove an accepted
case.

## Strategy and Environments

- Unit tests run without network services, use controlled clocks and ID sources, and test one
  observable rule or error path per case.
- Integration tests use isolated Testcontainers for PostgreSQL 18/pgvector, NATS JetStream, Valkey
  and S3-compatible storage plus controlled HTTP/OIDC/provider/model servers.
- Browser tests use a production web build, real API process, isolated real backing services,
  Playwright, axe, en/pt-BR dictionaries, wide/narrow viewports and keyboard-only paths.
- Every suite fixes UTC cutoffs and seeds versioned definitions. No test uses live provider/model
  behavior as a release gate.
- Required repository verification includes generated-contract drift, `go test ./...`,
  `go test -race ./...`, Go lint/vet, web lint/typecheck/unit tests/build, integration tests and
  Playwright/axe.

## Unit Test Cases

### Approved story edge behavior

| ID     | Story / edge | Class             | Scenario and observable result                                                                                                      |
| ------ | ------------ | ----------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| UT-001 | US-001 EC-1  | Invalid input     | Malformed search text is treated as text or rejected safely without query details.                                                  |
| UT-002 | US-001 EC-2  | Empty / missing   | An empty catalog shows an onboarding explanation rather than a broken grid.                                                         |
| UT-003 | US-001 EC-3  | Limits            | Large catalogs remain paginated and never return an unbounded response.                                                             |
| UT-004 | US-001 EC-4  | Permissions       | Anonymous deep links to protected views reveal no protected fields.                                                                 |
| UT-005 | US-001 EC-7  | Repetition        | Repeating the same search returns the same public representation.                                                                   |
| UT-006 | US-001 EC-8  | Ordering          | Opening a stale bookmarked page resolves by stable identity or shows not found.                                                     |
| UT-007 | US-001 EC-9  | State transitions | Paused Projects remain public; archived and deleted Projects do not.                                                                |
| UT-008 | US-002 EC-1  | Invalid input     | Invalid or unverifiable identity claims are rejected without creating membership.                                                   |
| UT-009 | US-002 EC-2  | Empty / missing   | A valid identity with no local profile enters pending state, never an implicit role.                                                |
| UT-010 | US-002 EC-3  | Limits            | Registration and callback attempts are rate-limited with a clear retry message.                                                     |
| UT-011 | US-002 EC-4  | Permissions       | An authenticated but pending Applicant cannot use protected APIs or exports.                                                        |
| UT-012 | US-002 EC-7  | Repetition        | Repeated login preserves one pending request and does not notify Admins repeatedly.                                                 |
| UT-013 | US-002 EC-8  | Ordering          | A callback without a matching login flow is rejected safely.                                                                        |
| UT-014 | US-002 EC-9  | State transitions | A suspended or rejected identity cannot return to pending without Admin action.                                                     |
| UT-015 | US-003 EC-1  | Invalid input     | An unknown role or malformed subject is rejected without changing membership.                                                       |
| UT-016 | US-003 EC-2  | Empty / missing   | With no pending users, the review view shows an empty state.                                                                        |
| UT-017 | US-003 EC-3  | Limits            | Membership lists are searchable and paginated at high user counts.                                                                  |
| UT-018 | US-003 EC-4  | Permissions       | Analysts and Viewers cannot approve, promote, suspend, or inspect applicant details.                                                |
| UT-019 | US-003 EC-7  | Repetition        | Reapproving the same user is idempotent and does not create duplicate membership.                                                   |
| UT-020 | US-003 EC-8  | Ordering          | A role cannot be assigned before the external subject is known.                                                                     |
| UT-021 | US-003 EC-9  | State transitions | The last active Admin cannot remove or suspend their own Admin access.                                                              |
| UT-022 | US-004 EC-1  | Invalid input     | Unsupported locales or timezones are rejected with supported choices.                                                               |
| UT-023 | US-004 EC-2  | Empty / missing   | Missing preferences use English and UTC until the member selects alternatives.                                                      |
| UT-024 | US-004 EC-3  | Limits            | Rapid preference changes are bounded and the latest confirmed value wins.                                                           |
| UT-025 | US-004 EC-4  | Permissions       | A member cannot edit or delete another member's profile.                                                                            |
| UT-026 | US-004 EC-7  | Repetition        | Repeating deletion after success returns a non-sensitive completed outcome.                                                         |
| UT-027 | US-004 EC-8  | Ordering          | Deletion cannot execute before explicit confirmation.                                                                               |
| UT-028 | US-004 EC-9  | State transitions | A suspended member may remove their account but cannot regain workspace access.                                                     |
| UT-029 | US-005 EC-1  | Invalid input     | Invalid filters are rejected or reset with an explanation.                                                                          |
| UT-030 | US-005 EC-2  | Empty / missing   | A portfolio with no active Projects shows role-appropriate next steps.                                                              |
| UT-031 | US-005 EC-3  | Limits            | Summary lists truncate predictably and link to paginated full views.                                                                |
| UT-032 | US-005 EC-4  | Permissions       | Viewers see intelligence but no mutation controls.                                                                                  |
| UT-033 | US-005 EC-7  | Repetition        | Refreshing does not trigger collection or AI work implicitly.                                                                       |
| UT-034 | US-005 EC-8  | Ordering          | Deep links work without visiting the overview first.                                                                                |
| UT-035 | US-005 EC-9  | State transitions | Archived Projects disappear from active summaries but remain available in archives.                                                 |
| UT-036 | US-006 EC-1  | Invalid input     | Malformed, unsupported, private, or hostile URLs are rejected with a safe reason.                                                   |
| UT-037 | US-006 EC-2  | Empty / missing   | A blank URL cannot create a Project.                                                                                                |
| UT-038 | US-006 EC-3  | Limits            | Operational quotas reject excess registration before creating partial state.                                                        |
| UT-039 | US-006 EC-4  | Permissions       | Viewers and pending users cannot register Projects.                                                                                 |
| UT-040 | US-006 EC-7  | Repetition        | Retrying a successful request resolves to the same Project.                                                                         |
| UT-041 | US-006 EC-8  | Ordering          | Additional repositories cannot be attached before their Project exists.                                                             |
| UT-042 | US-006 EC-9  | State transitions | A URL belonging to an archived Project offers restore rather than duplication.                                                      |
| UT-043 | US-007 EC-1  | Invalid input     | Unsupported roles or duplicate canonical URLs are rejected.                                                                         |
| UT-044 | US-007 EC-2  | Empty / missing   | A Project cannot lose its only primary Repository without replacement.                                                              |
| UT-045 | US-007 EC-3  | Limits            | Repository counts obey operational limits with no partial attachment.                                                               |
| UT-046 | US-007 EC-4  | Permissions       | Viewers cannot add, remove, or retype repositories.                                                                                 |
| UT-047 | US-007 EC-7  | Repetition        | Reattaching an existing repository returns its current association.                                                                 |
| UT-048 | US-007 EC-8  | Ordering          | A replacement primary is validated before the old primary loses its role.                                                           |
| UT-049 | US-007 EC-9  | State transitions | Repositories of archived Projects cannot be edited until restoration.                                                               |
| UT-050 | US-008 EC-1  | Invalid input     | A target Project that cannot accept the source is rejected before reassignment.                                                     |
| UT-051 | US-008 EC-2  | Empty / missing   | Associations without sufficient evidence remain visibly unresolved.                                                                 |
| UT-052 | US-008 EC-3  | Limits            | Large review queues are filterable and paginated.                                                                                   |
| UT-053 | US-008 EC-4  | Permissions       | Viewers can inspect provenance but cannot correct associations.                                                                     |
| UT-054 | US-008 EC-7  | Repetition        | Repeating the same correction does not enqueue duplicate recalculations.                                                            |
| UT-055 | US-008 EC-8  | Ordering          | A split completes before downstream recalculation publishes new results.                                                            |
| UT-056 | US-008 EC-9  | State transitions | Deleted Projects cannot receive reassigned sources.                                                                                 |
| UT-057 | US-009 EC-1  | Invalid input     | Unknown lifecycle transitions are rejected with allowed actions.                                                                    |
| UT-058 | US-009 EC-2  | Empty / missing   | Deletion confirmation without the required Project identity does nothing.                                                           |
| UT-059 | US-009 EC-3  | Limits            | Bulk lifecycle operations are not implied by a single-Project confirmation.                                                         |
| UT-060 | US-009 EC-4  | Permissions       | Analysts and Viewers cannot pause, archive, restore, or permanently delete a Project; every lifecycle transition is Admin-only.     |
| UT-061 | US-009 EC-7  | Repetition        | Repeating pause or archive is idempotent; repeating deletion reveals no purged data.                                                |
| UT-062 | US-009 EC-8  | Ordering          | A Project cannot restore after permanent deletion.                                                                                  |
| UT-063 | US-009 EC-9  | State transitions | Archived Projects reject edits, sync requests, and analysis requests.                                                               |
| UT-064 | US-010 EC-1  | Invalid input     | Unsupported refresh scopes are rejected before work is queued.                                                                      |
| UT-065 | US-010 EC-2  | Empty / missing   | A source without a checkpoint begins its configured initial backfill.                                                               |
| UT-066 | US-010 EC-3  | Limits            | Quota or concurrency exhaustion returns queued or delayed status, not silent loss.                                                  |
| UT-067 | US-010 EC-4  | Permissions       | Viewers can inspect status but cannot request refresh.                                                                              |
| UT-068 | US-010 EC-7  | Repetition        | Replaying a completed collection does not duplicate canonical records.                                                              |
| UT-069 | US-010 EC-8  | Ordering          | Analysis does not publish as current before required collection and metrics complete.                                               |
| UT-070 | US-010 EC-9  | State transitions | Paused, archived, or deleting Projects reject new synchronization.                                                                  |
| UT-071 | US-011 EC-1  | Invalid input     | End dates before start dates or future-only ranges are rejected.                                                                    |
| UT-072 | US-011 EC-2  | Empty / missing   | No collected history yields insufficient data, never zero activity.                                                                 |
| UT-073 | US-011 EC-3  | Limits            | Requests beyond provider or operator limits show the maximum allowed range.                                                         |
| UT-074 | US-011 EC-4  | Permissions       | Viewers cannot change backfill targets.                                                                                             |
| UT-075 | US-011 EC-7  | Repetition        | Requesting an already covered range does not recollect it unnecessarily.                                                            |
| UT-076 | US-011 EC-8  | Ordering          | A range extension does not rewrite older snapshots before data is available.                                                        |
| UT-077 | US-011 EC-9  | State transitions | Archived Projects retain coverage metadata without scheduling extension.                                                            |
| UT-078 | US-012 EC-1  | Invalid input     | Over-privileged or malformed credentials fail validation without echoing them.                                                      |
| UT-079 | US-012 EC-2  | Empty / missing   | Missing optional credentials show anonymous capability or unavailable status.                                                       |
| UT-080 | US-012 EC-3  | Limits            | Provider rate limits delay work with reset context rather than causing retry storms.                                                |
| UT-081 | US-012 EC-4  | Permissions       | End users cannot submit, retrieve, or select source credentials.                                                                    |
| UT-082 | US-012 EC-7  | Repetition        | Revalidation is safe and does not trigger collection.                                                                               |
| UT-083 | US-012 EC-8  | Ordering          | A token is validated before it becomes active.                                                                                      |
| UT-084 | US-012 EC-9  | State transitions | A source that becomes private stops collection and is marked unavailable.                                                           |
| UT-085 | US-013 EC-1  | Invalid input     | Unsupported metric windows are rejected with valid choices.                                                                         |
| UT-086 | US-013 EC-2  | Empty / missing   | Missing required evidence yields unavailable or insufficient data, never zero.                                                      |
| UT-087 | US-013 EC-3  | Limits            | Large evidence sets summarize and link to bounded pages.                                                                            |
| UT-088 | US-013 EC-4  | Permissions       | Pending and anonymous users cannot read metric values.                                                                              |
| UT-089 | US-013 EC-7  | Repetition        | Recalculating identical inputs and versions yields identical results.                                                               |
| UT-090 | US-013 EC-8  | Ordering          | An overall score cannot publish before all required dimension results resolve.                                                      |
| UT-091 | US-013 EC-9  | State transitions | Archived Projects retain historical metrics but receive no new snapshots.                                                           |
| UT-092 | US-014 EC-1  | Invalid input     | Invalid contributor filters or windows are rejected safely.                                                                         |
| UT-093 | US-014 EC-2  | Empty / missing   | No contributor evidence yields insufficient data rather than total concentration.                                                   |
| UT-094 | US-014 EC-3  | Limits            | Contributor lists paginate and do not expose private email addresses.                                                               |
| UT-095 | US-014 EC-4  | Permissions       | Only Analysts can confirm or split contributor identities.                                                                          |
| UT-096 | US-014 EC-7  | Repetition        | Reconfirming the same verified link is idempotent.                                                                                  |
| UT-097 | US-014 EC-8  | Ordering          | Identity linkage completes before aggregate concentration republishes.                                                              |
| UT-098 | US-014 EC-9  | State transitions | Deleted source accounts remain historical evidence with source status.                                                              |
| UT-099 | US-015 EC-1  | Invalid input     | Incomparable registry units cannot be forced into one universal rank.                                                               |
| UT-100 | US-015 EC-2  | Empty / missing   | No advisory or registry data is unknown, not evidence of safety or no adoption.                                                     |
| UT-101 | US-015 EC-3  | Limits            | Large advisory and package histories are paginated and windowed.                                                                    |
| UT-102 | US-015 EC-4  | Permissions       | Protected adoption and security intelligence requires approved membership.                                                          |
| UT-103 | US-015 EC-7  | Repetition        | Reingestion of one advisory or package sample does not duplicate it.                                                                |
| UT-104 | US-015 EC-8  | Ordering          | Normalization runs only after source population context is available.                                                               |
| UT-105 | US-015 EC-9  | State transitions | Withdrawn advisories retain provenance and display their withdrawn status.                                                          |
| UT-106 | US-016 EC-1  | Invalid input     | Fewer than two, more than five, duplicates, or invalid windows are rejected.                                                        |
| UT-107 | US-016 EC-2  | Empty / missing   | A Project with no comparable evidence remains in the view as insufficient data.                                                     |
| UT-108 | US-016 EC-3  | Limits            | Evidence details paginate without truncating the comparison conclusion silently.                                                    |
| UT-109 | US-016 EC-4  | Permissions       | Anonymous and pending users cannot run comparisons.                                                                                 |
| UT-110 | US-016 EC-7  | Repetition        | The same inputs and versions yield the same comparison.                                                                             |
| UT-111 | US-016 EC-8  | Ordering          | A comparison cannot run before every Project identity resolves.                                                                     |
| UT-112 | US-016 EC-9  | State transitions | Deleted Projects make saved comparisons unavailable; archived Projects remain historical.                                           |
| UT-113 | US-017 EC-1  | Invalid input     | Invalid baselines or forecast horizons are rejected.                                                                                |
| UT-114 | US-017 EC-2  | Empty / missing   | Sparse history produces insufficient data with the minimum requirement shown.                                                       |
| UT-115 | US-017 EC-3  | Limits            | Signal histories paginate and bounded detectors respect configured horizons.                                                        |
| UT-116 | US-017 EC-4  | Permissions       | Protected trends and warnings are unavailable to anonymous or pending users.                                                        |
| UT-117 | US-017 EC-7  | Repetition        | Deterministic trend reruns reproduce the same result from the same inputs.                                                          |
| UT-118 | US-017 EC-8  | Ordering          | Explanations cannot publish before the underlying signal exists.                                                                    |
| UT-119 | US-017 EC-9  | State transitions | Superseded warnings retain history and outcome-evaluation status.                                                                   |
| UT-120 | US-018 EC-1  | Invalid input     | Inactive or incompatible policy versions cannot be evaluated.                                                                       |
| UT-121 | US-018 EC-2  | Empty / missing   | Missing required evidence yields `insufficient_data`.                                                                               |
| UT-122 | US-018 EC-3  | Limits            | Evidence displays are bounded but retain links to every decisive input.                                                             |
| UT-123 | US-018 EC-4  | Permissions       | Viewers may read results; only Analysts/Admins can select policies for new evaluations.                                             |
| UT-124 | US-018 EC-7  | Repetition        | Identical policy, inputs, and versions reproduce the outcome.                                                                       |
| UT-125 | US-018 EC-8  | Ordering          | Evaluation waits for required metric versions or reports stale prerequisites.                                                       |
| UT-126 | US-018 EC-9  | State transitions | Results retain their original policy version after supersession.                                                                    |
| UT-127 | US-019 EC-1  | Invalid input     | Contradictory outcomes, invalid weights, or unknown metrics block publication.                                                      |
| UT-128 | US-019 EC-2  | Empty / missing   | Required evidence rules cannot be blank when an outcome depends on them.                                                            |
| UT-129 | US-019 EC-3  | Limits            | Policy lists and version histories are paginated.                                                                                   |
| UT-130 | US-019 EC-4  | Permissions       | Analysts may select but cannot create, modify, publish, or retire policies.                                                         |
| UT-131 | US-019 EC-7  | Repetition        | Repeating publication cannot create two versions from one draft state.                                                              |
| UT-132 | US-019 EC-8  | Ordering          | A draft must validate before publication or activation.                                                                             |
| UT-133 | US-019 EC-9  | State transitions | Published versions are immutable; retired versions cannot reactivate silently.                                                      |
| UT-134 | US-020 EC-1  | Invalid input     | Unknown rings, past-invalid review dates, or blank override reasons are rejected.                                                   |
| UT-135 | US-020 EC-2  | Empty / missing   | `insufficient_data` maps only according to an explicit policy mapping.                                                              |
| UT-136 | US-020 EC-3  | Limits            | Large radars filter and group Projects without hiding off-screen counts.                                                            |
| UT-137 | US-020 EC-4  | Permissions       | Viewers read the radar but cannot select, override, or annotate Projects.                                                           |
| UT-138 | US-020 EC-7  | Repetition        | Reapplying the same selection or override is idempotent.                                                                            |
| UT-139 | US-020 EC-8  | Ordering          | A Project needs a policy result before receiving a suggested ring.                                                                  |
| UT-140 | US-020 EC-9  | State transitions | Archived Projects leave the active radar but retain historical placement.                                                           |
| UT-141 | US-021 EC-1  | Invalid input     | Empty names, circular merges, or unsupported assignments are rejected.                                                              |
| UT-142 | US-021 EC-2  | Empty / missing   | No eligible content yields insufficient data, not zero topic prevalence.                                                            |
| UT-143 | US-021 EC-3  | Limits            | Large topic and evidence sets paginate and cap displayed examples transparently.                                                    |
| UT-144 | US-021 EC-4  | Permissions       | Viewers cannot correct classifications.                                                                                             |
| UT-145 | US-021 EC-7  | Repetition        | Repeating a correction does not duplicate feedback.                                                                                 |
| UT-146 | US-021 EC-8  | Ordering          | Trend calculations wait for a complete topic version.                                                                               |
| UT-147 | US-021 EC-9  | State transitions | Retired topics remain in historical results but not new assignments.                                                                |
| UT-148 | US-022 EC-1  | Invalid input     | Malformed provider output cannot publish as a valid structured analysis.                                                            |
| UT-149 | US-022 EC-2  | Empty / missing   | A release without changelog shows limited evidence rather than invented changes.                                                    |
| UT-150 | US-022 EC-3  | Limits            | Large changelogs and evidence sets are bounded with disclosed truncation.                                                           |
| UT-151 | US-022 EC-4  | Permissions       | Anonymous visitors cannot read protected release analysis.                                                                          |
| UT-152 | US-022 EC-7  | Repetition        | Replaying one run request does not overwrite or duplicate the same execution identity.                                              |
| UT-153 | US-022 EC-8  | Ordering          | Analysis does not claim evidence before eligible inputs are collected.                                                              |
| UT-154 | US-022 EC-9  | State transitions | Withdrawn releases retain their source status and historical analysis.                                                              |
| UT-155 | US-023 EC-1  | Invalid input     | Unsupported schemes, unsafe addresses, and out-of-scope domains are rejected.                                                       |
| UT-156 | US-023 EC-2  | Empty / missing   | No indexed documentation yields an explicit no-evidence response.                                                                   |
| UT-157 | US-023 EC-3  | Limits            | Crawl depth, bytes, pages, and request rate stop predictably with visible coverage.                                                 |
| UT-158 | US-023 EC-4  | Permissions       | Only Analysts can configure crawl scope; approved users may search.                                                                 |
| UT-159 | US-023 EC-7  | Repetition        | Unchanged content does not create misleading duplicate versions.                                                                    |
| UT-160 | US-023 EC-8  | Ordering          | Search indexes only validated snapshots.                                                                                            |
| UT-161 | US-023 EC-9  | State transitions | Removed URLs remain historical evidence but leave current search after refresh.                                                     |
| UT-162 | US-024 EC-1  | Invalid input     | Hostile or unsupported requests cannot escape the bounded analytical/action catalog.                                                |
| UT-163 | US-024 EC-2  | Empty / missing   | Blank questions or empty evidence return actionable guidance, not fabricated answers.                                               |
| UT-164 | US-024 EC-3  | Limits            | Token, query, and evidence limits show truncation and refinement guidance.                                                          |
| UT-165 | US-024 EC-4  | Permissions       | The assistant cannot retrieve evidence the requesting user cannot access.                                                           |
| UT-166 | US-024 EC-7  | Repetition        | Repeating a question identifies its current cutoff rather than implying timeless identity.                                          |
| UT-167 | US-024 EC-8  | Ordering          | Clarification must resolve before analysis or action proposal continues.                                                            |
| UT-168 | US-024 EC-9  | State transitions | Questions about deleted Projects return unavailable without leaked history.                                                         |
| UT-169 | US-025 EC-1  | Invalid input     | Untyped or unsupported proposal fields cannot reach execution.                                                                      |
| UT-170 | US-025 EC-2  | Empty / missing   | A proposal without every required value asks for clarification.                                                                     |
| UT-171 | US-025 EC-3  | Limits            | Quota-exceeding proposals show the limit and cannot be approved into a hidden queue.                                                |
| UT-172 | US-025 EC-4  | Permissions       | Approval rechecks the current user's role and resource access.                                                                      |
| UT-173 | US-025 EC-7  | Repetition        | Replaying one approval cannot repeat the mutation.                                                                                  |
| UT-174 | US-025 EC-8  | Ordering          | Approval before preview or after expiration is rejected.                                                                            |
| UT-175 | US-025 EC-9  | State transitions | Actions targeting paused, archived, or deleted resources obey lifecycle rules.                                                      |
| UT-176 | US-026 EC-1  | Invalid input     | Feedback without a target version or reason is rejected.                                                                            |
| UT-177 | US-026 EC-2  | Empty / missing   | No successful run shows unavailable analysis and eligible next actions.                                                             |
| UT-178 | US-026 EC-3  | Limits            | Version and evidence histories paginate; reruns obey quotas.                                                                        |
| UT-179 | US-026 EC-4  | Permissions       | Viewers inspect but cannot rerun, flag, or select versions.                                                                         |
| UT-180 | US-026 EC-7  | Repetition        | Duplicate feedback from one request is idempotent.                                                                                  |
| UT-181 | US-026 EC-8  | Ordering          | A failed or incomplete run cannot be selected as presented output.                                                                  |
| UT-182 | US-026 EC-9  | State transitions | Stale selected runs remain visible with a stale warning until replaced.                                                             |
| UT-183 | US-027 EC-1  | Invalid input     | Rules with unknown signals, invalid thresholds, or negative cooldowns are rejected.                                                 |
| UT-184 | US-027 EC-2  | Empty / missing   | Missing required evidence cannot trigger an alert.                                                                                  |
| UT-185 | US-027 EC-3  | Limits            | Alert lists paginate and rule/occurrence volume respects workspace quotas.                                                          |
| UT-186 | US-027 EC-4  | Permissions       | Viewers manage only personal read state; Analysts/Admins manage shared resolution.                                                  |
| UT-187 | US-027 EC-7  | Repetition        | Replayed signals preserve one deduplicated occurrence.                                                                              |
| UT-188 | US-027 EC-8  | Ordering          | Resolution cannot precede occurrence creation; reopening follows explicit rules.                                                    |
| UT-189 | US-027 EC-9  | State transitions | Archived Projects stop new alert evaluation while history remains readable.                                                         |
| UT-190 | US-028 EC-1  | Invalid input     | Unsupported formats or malformed scopes are rejected before generation.                                                             |
| UT-191 | US-028 EC-2  | Empty / missing   | Empty result sets produce a valid empty export with scope metadata.                                                                 |
| UT-192 | US-028 EC-3  | Limits            | Oversized exports are rejected or generated asynchronously with visible limits and status.                                          |
| UT-193 | US-028 EC-4  | Permissions       | Exports contain only data visible to the requesting approved member.                                                                |
| UT-194 | US-028 EC-7  | Repetition        | Identical requests identify equivalent scope and cutoff without changing data.                                                      |
| UT-195 | US-028 EC-8  | Ordering          | A download is unavailable until generation completes successfully.                                                                  |
| UT-196 | US-028 EC-9  | State transitions | A completed download expires after 24 hours, and Project deletion removes owned generated exports earlier.                          |
| UT-197 | US-029 EC-1  | Invalid input     | Invalid ranges or filters are rejected without arbitrary query execution.                                                           |
| UT-198 | US-029 EC-2  | Empty / missing   | No matches produce a clear empty state.                                                                                             |
| UT-199 | US-029 EC-3  | Limits            | Reads and exports paginate or bound results without dropping event counts silently.                                                 |
| UT-200 | US-029 EC-4  | Permissions       | Only Admins can inspect or export audit history.                                                                                    |
| UT-201 | US-029 EC-7  | Repetition        | Idempotent retries remain attributable without fabricating duplicate success.                                                       |
| UT-202 | US-029 EC-8  | Ordering          | Audit ordering uses event time and stable tie-break identity.                                                                       |
| UT-203 | US-029 EC-9  | State transitions | Audit events remain immutable after subject deletion or role changes.                                                               |
| UT-204 | US-030 EC-1  | Invalid input     | Unsupported model capabilities or malformed configuration fail validation safely.                                                   |
| UT-205 | US-030 EC-2  | Empty / missing   | No provider is a valid degraded state, not an application startup failure.                                                          |
| UT-206 | US-030 EC-3  | Limits            | Model quotas delay or reject AI work with visible status and no retry storm.                                                        |
| UT-207 | US-030 EC-4  | Permissions       | Users cannot submit or retrieve provider secrets; only Admins see redacted status.                                                  |
| UT-208 | US-030 EC-7  | Repetition        | Retried requests create attributable runs without overwriting successful history.                                                   |
| UT-209 | US-030 EC-8  | Ordering          | New configuration validates before becoming active for later runs.                                                                  |
| UT-210 | US-030 EC-9  | State transitions | Disabled providers make dependent features unavailable without deleting prior outputs.                                              |
| UT-211 | US-031 EC-1  | Invalid input     | Unsupported locale input falls back to English and remains changeable.                                                              |
| UT-212 | US-031 EC-2  | Empty / missing   | Missing translation never produces blank controls or hidden errors.                                                                 |
| UT-213 | US-031 EC-3  | Limits            | Dense tables and charts use responsive summaries and bounded detail rather than data loss.                                          |
| UT-214 | US-031 EC-4  | Permissions       | Responsive layouts never reveal controls or data hidden from the role.                                                              |
| UT-215 | US-031 EC-7  | Repetition        | Repeated locale switching does not alter stored evidence or calculations.                                                           |
| UT-216 | US-031 EC-8  | Ordering          | Deep links apply membership, language, and timezone before protected content renders.                                               |
| UT-217 | US-031 EC-9  | State transitions | Disabled actions remain labeled with their lifecycle reason at narrow widths.                                                       |
| UT-218 | US-032 EC-1  | Invalid input     | A malformed, expired, wrong-issuer, wrong-audience, or unverifiable token is rejected without creating a local account.             |
| UT-219 | US-032 EC-2  | Empty / missing   | A valid Keycloak subject without a local binding receives no implicit access.                                                       |
| UT-220 | US-032 EC-3  | Limits            | Service-account requests use the applicable account, workspace, source, and Job quotas.                                             |
| UT-221 | US-032 EC-4  | Permissions       | A service account cannot receive Admin, approve members, change policies, manage credentials, or perform Project lifecycle actions. |
| UT-222 | US-032 EC-7  | Repetition        | Repeated bearer requests remain individually attributable while idempotency preserves one business outcome.                         |
| UT-223 | US-032 EC-8  | Ordering          | Local suspension takes effect before any request authorized against the later account version.                                      |
| UT-224 | US-032 EC-9  | State transitions | Deleting or suspending the local binding prevents access even while the external Keycloak token remains valid.                      |

### Core component and algorithm behavior

| ID     | Component behavior          | Setup / action                                                                | Expected result                                                                          |
| ------ | --------------------------- | ----------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| UT-225 | Snowflake lease exclusivity | Two issuers contend for one node lease.                                       | Only one lease commits and only its holder may issue IDs.                                |
| UT-226 | Snowflake sequence          | Generate more than one ID in the same millisecond.                            | Sequence bits preserve strict uniqueness and stable decimal serialization.               |
| UT-227 | Snowflake clock regression  | The clock moves behind the last issued timestamp beyond tolerance.            | Issuance stops with a typed error and never reuses a timestamp/sequence.                 |
| UT-228 | Half-open window            | An event occurs exactly at from and another exactly at to.                    | The from event is included and the to event is excluded.                                 |
| UT-229 | Stable release cohort       | A window contains stable, draft and prerelease releases.                      | Only non-draft, non-prerelease releases contribute.                                      |
| UT-230 | Contributor cohort          | Default-branch human, bot, merge and non-default commits coexist.             | Only human non-merge default-branch commits contribute to concentration.                 |
| UT-231 | Issue first response        | The opener, bots, outsiders and a member comment in order.                    | The first public non-bot member/collaborator response other than the opener is selected. |
| UT-232 | Unanswered issue coverage   | Some issues have no qualifying response by cutoff.                            | They remain censored and reduce coverage; no response duration is invented.              |
| UT-233 | PR merge cohort             | A PR has ready_for_review and merged events.                                  | Merge time starts at ready_for_review and ends at merged.                                |
| UT-234 | PR readiness fallback       | A merged PR lacks ready_for_review.                                           | Created time is used and the fallback is exposed in coverage.                            |
| UT-235 | Backlog reconstruction      | Issues open, close and reopen around both window boundaries.                  | End-minus-start backlog is reconstructed from state events.                              |
| UT-236 | Health equal weights        | All seven dimensions are available.                                           | Each absolute dimension contributes exactly one seventh.                                 |
| UT-237 | Health missing dimension    | One dimension is unavailable.                                                 | Its state stays unavailable and its weight is not redistributed.                         |
| UT-238 | Common comparison cutoff    | Projects have snapshots at different cutoffs.                                 | The service selects/materializes one common cutoff or marks an item incomparable.        |
| UT-239 | Theil-Sen trend             | The series contains an outlier.                                               | The versioned median pairwise slope is stable and resists the outlier.                   |
| UT-240 | Mann-Kendall trend          | A short/noisy series lacks significance.                                      | The result is insufficient/no trend rather than a directional claim.                     |
| UT-241 | Forecast backtest           | Candidate seasonal and non-seasonal baselines are evaluated.                  | Rolling backtest error selects the method and the result includes an interval.           |
| UT-242 | RRF search                  | FTS and vector rankings disagree.                                             | Reciprocal-rank fusion is deterministic with stable tie breaking.                        |
| UT-243 | Topic mutual-kNN            | Only one of two chunks includes the other among k neighbors.                  | No mutual edge is created.                                                               |
| UT-244 | Topic correction            | An analyst merge/split constraint is applied before reanalysis.               | The constraint is canonical input and original run history remains immutable.            |
| UT-245 | Policy determinism          | The same versioned facts and policy are evaluated twice.                      | Outcome and matched factors are identical.                                               |
| UT-246 | Radar override              | An attributed override expires or is removed.                                 | The computed policy placement becomes effective without history rewrite.                 |
| UT-247 | OIDC identity key           | The same email arrives with different issuer/sub combinations.                | Identity is keyed by issuer and subject, never email.                                    |
| UT-248 | Local role authority        | A valid token contains a role-like claim but no active membership.            | Access is denied and no local role is inferred.                                          |
| UT-249 | Service-account scopes      | An Analyst service account lacks the operation scope.                         | The narrower scope denies the action.                                                    |
| UT-250 | Session verifier            | A stolen raw database row is presented as a cookie.                           | Only the original opaque verifier matches its stored hash.                               |
| UT-251 | Cursor binding              | A signed cursor is reused with different filters or route.                    | HMAC/context validation rejects it.                                                      |
| UT-252 | Idempotency mismatch        | One key is reused by the same actor with different normalized input.          | The second command conflicts and causes no new Job or side effect.                       |
| UT-253 | Conditional update          | If-Match names an older aggregate version.                                    | The mutation fails atomically with the frozen stale-version problem.                     |
| UT-254 | Public URL safety           | DNS resolves or redirects to a private/link-local address.                    | Registration/collection is rejected before any protected request.                        |
| UT-255 | Git transport safety        | A source uses file, SSH, helper, hook or submodule execution.                 | The adapter rejects or disables it and performs no local execution.                      |
| UT-256 | Checkpoint commit           | Normalization fails after a page begins.                                      | The checkpoint advances only with the committed normalized page.                         |
| UT-257 | Job state machine           | A terminal Job receives an active transition.                                 | The transition is rejected and terminal state is unchanged.                              |
| UT-258 | Purge cancellation          | Cancellation arrives after destructive purge begins.                          | The non-cancellable Job continues and reports the frozen conflict.                       |
| UT-259 | Evidence checksum           | Stored bytes do not match the committed digest.                               | The evidence is unavailable/corrupt and cannot support a claim.                          |
| UT-260 | Export expiry               | A download occurs at and after completion plus 24 hours.                      | The former boundary follows the frozen interval and expired access returns 410.          |
| UT-261 | AI evidence gate            | Structured model output contains an unsupported claim.                        | The run cannot succeed until every claim has accessible immutable evidence.              |
| UT-262 | Model schema rejection      | The provider returns malformed structured output.                             | The attempt records a schema error and publishes no partial successful analysis.         |
| UT-263 | ADK tool allowlist          | An agent requests SQL, filesystem or arbitrary HTTP access.                   | The tool request is rejected and recorded without execution.                             |
| UT-264 | HITL confirmation           | A confirmation is reused, expired, action-mismatched or version-stale.        | Execution is denied and no mutation occurs.                                              |
| UT-265 | Assistant forbidden action  | A proposal requests membership, credential, policy, archive or deletion work. | The proposal returns action_not_allowed before HITL.                                     |
| UT-266 | Provider mapping            | Two providers represent the same canonical state differently.                 | Each adapter maps to the same canonical model without provider DTO leakage.              |
| UT-267 | Alert deduplication         | The same versioned condition fires inside its deduplication window.           | One shared occurrence remains, with per-member read state separate.                      |
| UT-268 | Purge manifest              | Object deletion stops midway and restarts.                                    | Reconciliation resumes idempotently and finalizes only after all owned objects are gone. |
| UT-269 | Configuration redaction     | Several conditional settings are missing or invalid.                          | Startup reports all safe field errors together and never reports secret values.          |
| UT-270 | Locale parity               | The same principal performs a flow in en and pt-BR.                           | Authorization, facts and actions are identical; only presentation changes.               |
| UT-271 | Error serialization         | A typed domain error crosses HTTP transport.                                  | It maps to the documented problem code/status/request ID without cause leakage.          |
| UT-272 | Last Admin                  | The last active Admin attempts self-demotion or suspension.                   | The invariant rejects the mutation atomically.                                           |
| UT-273 | Analysis immutability       | Feedback, rerun and selected-run changes occur.                               | Original run/evidence remain immutable and selection history is attributed.              |
| UT-274 | Valkey authority            | The cache contains stale authorization or Job data.                           | PostgreSQL truth wins and stale cache cannot grant access or change terminal state.      |

## Integration Test Cases

### Approved cross-boundary edge behavior

| ID     | Story / edge | Class        | Scenario and observable result                                                                                   |
| ------ | ------------ | ------------ | ---------------------------------------------------------------------------------------------------------------- |
| IT-001 | US-001 EC-5  | Concurrency  | A Project archived during browsing disappears on refresh without corrupting the page.                            |
| IT-002 | US-001 EC-6  | Interruption | A failed page request preserves the last visible page and offers retry.                                          |
| IT-003 | US-001 EC-10 | Scale        | Search and pagination remain usable at 100 times the expected initial catalog.                                   |
| IT-004 | US-002 EC-5  | Concurrency  | Simultaneous first logins create one pending membership request.                                                 |
| IT-005 | US-002 EC-6  | Interruption | An interrupted authentication flow can restart without a partial product session.                                |
| IT-006 | US-002 EC-10 | Scale        | A burst of applicants does not expose data or make approved sessions unavailable.                                |
| IT-007 | US-003 EC-5  | Concurrency  | Conflicting approvals use the latest valid state and report the stale action.                                    |
| IT-008 | US-003 EC-6  | Interruption | A failed role change leaves the prior role effective and reports failure.                                        |
| IT-009 | US-003 EC-10 | Scale        | Bulk applicant volume remains operable without bulk implicit approval.                                           |
| IT-010 | US-004 EC-5  | Concurrency  | Concurrent deletion and profile edits resolve to deletion without restoring data.                                |
| IT-011 | US-004 EC-6  | Interruption | An interrupted deletion confirmation does not delete the account.                                                |
| IT-012 | US-004 EC-10 | Scale        | Preference application does not require loading all workspace members.                                           |
| IT-013 | US-005 EC-5  | Concurrency  | New snapshots do not mix incompatible calculation versions in one rendered result.                               |
| IT-014 | US-005 EC-6  | Interruption | One failed panel does not erase deterministic panels that loaded successfully.                                   |
| IT-015 | US-005 EC-10 | Scale        | The overview uses aggregation and pagination rather than rendering every Project at once.                        |
| IT-016 | US-006 EC-5  | Concurrency  | Concurrent registration of the same canonical URL creates one Project.                                           |
| IT-017 | US-006 EC-6  | Interruption | Failure after creation shows recoverable collection state, not a duplicate prompt.                               |
| IT-018 | US-006 EC-10 | Scale        | Registration remains responsive while other Projects are backfilling.                                            |
| IT-019 | US-007 EC-5  | Concurrency  | Concurrent primary changes result in exactly one primary Repository.                                             |
| IT-020 | US-007 EC-6  | Interruption | An interrupted edit preserves the last valid repository set.                                                     |
| IT-021 | US-007 EC-10 | Scale        | Projects with many repositories summarize them and provide pagination or filtering.                              |
| IT-022 | US-008 EC-5  | Concurrency  | Concurrent corrections detect stale source ownership and preserve one valid result.                              |
| IT-023 | US-008 EC-6  | Interruption | A failed correction leaves the prior association and derived status intact.                                      |
| IT-024 | US-008 EC-10 | Scale        | Corrections invalidate only affected evidence, not the entire workspace.                                         |
| IT-025 | US-009 EC-5  | Concurrency  | Collection racing with deletion cannot publish data after the deletion guard takes effect.                       |
| IT-026 | US-009 EC-6  | Interruption | A partial purge remains visibly in progress and resumes safely.                                                  |
| IT-027 | US-009 EC-10 | Scale        | Purging one large Project does not block reading unrelated Projects.                                             |
| IT-028 | US-010 EC-5  | Concurrency  | Simultaneous refresh requests coalesce into one compatible run.                                                  |
| IT-029 | US-010 EC-6  | Interruption | Cancellation or restart preserves the last durable checkpoint.                                                   |
| IT-030 | US-010 EC-10 | Scale        | Workspace-wide backfills remain bounded and expose queue position or delay.                                      |
| IT-031 | US-011 EC-5  | Concurrency  | Overlapping range extensions coalesce into the broadest valid target.                                            |
| IT-032 | US-011 EC-6  | Interruption | Partial backfill publishes its actual boundary and resumable state.                                              |
| IT-033 | US-011 EC-10 | Scale        | Long histories are summarized and paginated without truncating coverage disclosure.                              |
| IT-034 | US-012 EC-5  | Concurrency  | Credential rotation does not mix identities within one source request.                                           |
| IT-035 | US-012 EC-6  | Interruption | Rotation failure preserves the last valid configuration or reports no active credential.                         |
| IT-036 | US-012 EC-10 | Scale        | Quota reporting aggregates safely across many Projects without leaking request details.                          |
| IT-037 | US-013 EC-5  | Concurrency  | Recalculation publishes one internally consistent metric-version snapshot.                                       |
| IT-038 | US-013 EC-6  | Interruption | Failed recalculation leaves the prior snapshot visible and marked stale.                                         |
| IT-039 | US-013 EC-10 | Scale        | Time series remain usable at 100 times the initial history through aggregation.                                  |
| IT-040 | US-014 EC-5  | Concurrency  | Concurrent identity corrections cannot leave one account linked twice.                                           |
| IT-041 | US-014 EC-6  | Interruption | Failed recalculation leaves prior values stale rather than partially updated.                                    |
| IT-042 | US-014 EC-10 | Scale        | High-volume contributor histories use bounded detail without changing aggregates.                                |
| IT-043 | US-015 EC-5  | Concurrency  | New registry data cannot mix cutoffs within one published snapshot.                                              |
| IT-044 | US-015 EC-6  | Interruption | A failed registry or advisory source leaves other evidence visible with stale status.                            |
| IT-045 | US-015 EC-10 | Scale        | Cross-registry portfolios retain source-specific context at high volume.                                         |
| IT-046 | US-016 EC-5  | Concurrency  | Metrics updated during a comparison do not create mixed cutoffs.                                                 |
| IT-047 | US-016 EC-6  | Interruption | Partial failure identifies unavailable Projects and preserves completed deterministic data.                      |
| IT-048 | US-016 EC-10 | Scale        | Many saved comparisons remain searchable and paginated.                                                          |
| IT-049 | US-017 EC-5  | Concurrency  | One signal version publishes against one consistent input snapshot.                                              |
| IT-050 | US-017 EC-6  | Interruption | Failed prediction does not suppress valid observed trends.                                                       |
| IT-051 | US-017 EC-10 | Scale        | Detection remains bounded across the portfolio and exposes delayed status.                                       |
| IT-052 | US-018 EC-5  | Concurrency  | A policy activation during evaluation does not change the selected version.                                      |
| IT-053 | US-018 EC-6  | Interruption | Failed explanation leaves the deterministic result available.                                                    |
| IT-054 | US-018 EC-10 | Scale        | Portfolio evaluation queues remain bounded and expose progress.                                                  |
| IT-055 | US-019 EC-5  | Concurrency  | Concurrent edits detect stale drafts instead of overwriting changes.                                             |
| IT-056 | US-019 EC-6  | Interruption | Failed publication leaves the draft editable and no partial version active.                                      |
| IT-057 | US-019 EC-10 | Scale        | Many historical versions remain searchable without loading all definitions.                                      |
| IT-058 | US-020 EC-5  | Concurrency  | Concurrent movement detects stale radar state.                                                                   |
| IT-059 | US-020 EC-6  | Interruption | A failed override preserves the prior ring and recommendation.                                                   |
| IT-060 | US-020 EC-10 | Scale        | Radar calculation remains bounded when many Projects receive updated recommendations.                            |
| IT-061 | US-021 EC-5  | Concurrency  | Concurrent corrections detect stale topic versions.                                                              |
| IT-062 | US-021 EC-6  | Interruption | Failed reprocessing leaves the prior version visible and stale.                                                  |
| IT-063 | US-021 EC-10 | Scale        | Clustering is bounded and exposes queued or sampled status when needed.                                          |
| IT-064 | US-022 EC-5  | Concurrency  | Concurrent reruns create distinct immutable versions.                                                            |
| IT-065 | US-022 EC-6  | Interruption | Failed analysis retains deterministic metadata and prior successful versions.                                    |
| IT-066 | US-022 EC-10 | Scale        | Release lists and runs paginate across long histories.                                                           |
| IT-067 | US-023 EC-5  | Concurrency  | Duplicate crawl requests coalesce by source and snapshot target.                                                 |
| IT-068 | US-023 EC-6  | Interruption | Partial crawls expose their boundary and resume without duplicating snapshots.                                   |
| IT-069 | US-023 EC-10 | Scale        | Search remains bounded and identifies sampling or truncation at large corpus sizes.                              |
| IT-070 | US-024 EC-5  | Concurrency  | Evidence updates during a response do not mix data cutoffs silently.                                             |
| IT-071 | US-024 EC-6  | Interruption | A canceled query stops generation and preserves no partial action approval.                                      |
| IT-072 | US-024 EC-10 | Scale        | Broad questions are narrowed or bounded rather than scanning the workspace without limit.                        |
| IT-073 | US-025 EC-5  | Concurrency  | Changed resource state invalidates the preview and requires a new proposal.                                      |
| IT-074 | US-025 EC-6  | Interruption | Lost connection or expired proposal executes nothing.                                                            |
| IT-075 | US-025 EC-10 | Scale        | A request containing many actions is split into atomic approvals or rejected as too broad.                       |
| IT-076 | US-026 EC-5  | Concurrency  | Concurrent selection detects stale current-version state.                                                        |
| IT-077 | US-026 EC-6  | Interruption | Interrupted runs remain terminally labeled and never appear successful.                                          |
| IT-078 | US-026 EC-10 | Scale        | Retained versions remain discoverable without loading every output at once.                                      |
| IT-079 | US-027 EC-5  | Concurrency  | Concurrent resolution uses one final state and identifies stale actions.                                         |
| IT-080 | US-027 EC-6  | Interruption | Evaluation failure does not close or duplicate an existing occurrence.                                           |
| IT-081 | US-027 EC-10 | Scale        | High event volume is bounded without losing severity and suppression counts.                                     |
| IT-082 | US-028 EC-5  | Concurrency  | Export uses one explicit data cutoff despite ongoing updates.                                                    |
| IT-083 | US-028 EC-6  | Interruption | Interrupted generation can be retried without duplicate completed artifacts.                                     |
| IT-084 | US-028 EC-10 | Scale        | Large exports do not exhaust interactive product capacity.                                                       |
| IT-085 | US-029 EC-5  | Concurrency  | Simultaneous actions retain distinct event identities and deterministic ordering ties.                           |
| IT-086 | US-029 EC-6  | Interruption | A failed business action records failure without claiming a state change.                                        |
| IT-087 | US-029 EC-10 | Scale        | Long retention remains searchable through bounded pages and time filters.                                        |
| IT-088 | US-030 EC-5  | Concurrency  | Configuration changes do not alter provider identity within an active run.                                       |
| IT-089 | US-030 EC-6  | Interruption | Interrupted runs become failed or canceled versions and never block deterministic work.                          |
| IT-090 | US-030 EC-10 | Scale        | Global model concurrency and usage limits protect interactive deterministic traffic.                             |
| IT-091 | US-031 EC-5  | Concurrency  | Language changes during a save do not repeat or lose the action.                                                 |
| IT-092 | US-031 EC-6  | Interruption | Navigation or connection loss preserves recoverable form state where safe.                                       |
| IT-093 | US-031 EC-10 | Scale        | Large localized strings and 200% zoom remain usable without clipping essential actions.                          |
| IT-094 | US-032 EC-5  | Concurrency  | Concurrent scope changes and requests authorize against one committed local account version.                     |
| IT-095 | US-032 EC-6  | Interruption | An interrupted idempotent action can retry without duplicating the resource or Job.                              |
| IT-096 | US-032 EC-10 | Scale        | Many service-account calls remain isolated by identity and scope without exhausting interactive member capacity. |

### Component boundary and failure behavior

| ID     | Boundary                   | Setup / action                                                                                                  | Expected result                                                                                    |
| ------ | -------------------------- | --------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| IT-097 | Migration round trip       | Apply all migrations to empty PostgreSQL 18/pgvector, validate constraints, roll back supported steps, reapply. | Schema converges and migration history is deterministic.                                           |
| IT-098 | SQL generated access       | Run sqlc queries against migrated PostgreSQL.                                                                   | Generated pgx types scan NULL, bigint and vector fields without hand-written drift.                |
| IT-099 | Snowflake lease failover   | Stop one holder and advance beyond lease expiry.                                                                | A new holder acquires the node without overlapping issuance.                                       |
| IT-100 | OIDC code flow             | Use a controlled issuer for PKCE callback and token validation.                                                 | One opaque session and one pending/active local identity result.                                   |
| IT-101 | OIDC invalid tokens        | Exercise wrong issuer, audience, signature, nonce and expired time.                                             | Each fails closed without membership/session creation.                                             |
| IT-102 | Session revocation         | Suspend a member with active API sessions.                                                                      | Subsequent requests deny access and session state is revoked.                                      |
| IT-103 | Command transaction        | Inject failure between aggregate, Job and outbox writes.                                                        | All three commit or none commit.                                                                   |
| IT-104 | Outbox publish             | Relay a committed outbox row to JetStream.                                                                      | Message identity/schema/correlation are correct and published_at commits afterward.                |
| IT-105 | Outbox retry               | Lose acknowledgement after broker acceptance.                                                                   | Republish is deduplicated and one logical event is consumed.                                       |
| IT-106 | Consumer redelivery        | Crash after business commit but before JetStream acknowledgement.                                               | Redelivery observes idempotent terminal/checkpoint state and duplicates nothing.                   |
| IT-107 | Dead-letter advisory       | Exhaust configured attempts for a poison command.                                                               | Job fails durably, attempt history remains, and an operational advisory is emitted.                |
| IT-108 | Job lease recovery         | Terminate a worker with an active lease.                                                                        | Another worker resumes from the last checkpoint after lease expiry.                                |
| IT-109 | Job cancellation           | Cancel between two collector pages.                                                                             | No later page starts and durable partial coverage/cancel state is consistent.                      |
| IT-110 | SSE resume                 | Disconnect after an event and reconnect with Last-Event-ID.                                                     | No durable Job version is missed; terminal state closes the stream.                                |
| IT-111 | Valkey outage              | Stop Valkey during active SSE and rate-limit cache use.                                                         | PostgreSQL fallback preserves truth and safe degradation is reported.                              |
| IT-112 | S3 atomic visibility       | Fail object upload or reference transaction independently.                                                      | No visible dangling reference; orphan reconciliation is checksum and ownership safe.               |
| IT-113 | S3 purge recovery          | Interrupt a multi-object purge repeatedly.                                                                      | Manifest resumes and final tombstone appears only after reconciliation.                            |
| IT-114 | GitHub incremental sync    | Replay fixture pages with overlaps and updated records.                                                         | Canonical upserts and checkpoints are idempotent.                                                  |
| IT-115 | GitLab/Gitea mapping       | Normalize equivalent fixture data from each provider.                                                           | Canonical invariants and provenance are provider-independent.                                      |
| IT-116 | Git hardening              | Controlled remote advertises unsafe transports/submodules/hooks or exceeds limits.                              | Fetch fails safely without execution or partial canonical commit.                                  |
| IT-117 | Crawler SSRF               | Controlled DNS rebinding and redirect chain target forbidden IPs.                                               | Every hop is revalidated and no forbidden connection occurs.                                       |
| IT-118 | Crawler bounds             | Serve oversized, deep, slow and unsupported content.                                                            | The capture stops at configured bounds and records explicit coverage/error.                        |
| IT-119 | Registry snapshots         | Replay registry observations at the same cutoff.                                                                | One source-contextual snapshot exists per identity/cutoff.                                         |
| IT-120 | Canonical temporal events  | Replay issue/PR reopen/readiness events out of provider page order.                                             | Normalized event order reconstructs the same window metrics.                                       |
| IT-121 | Metric materialization     | Run one snapshot twice under the same computation key.                                                          | One immutable snapshot/factor set exists.                                                          |
| IT-122 | Comparison materialization | Request two-to-five projects with mixed freshness.                                                              | One cutoff/version set is used and missing/incomparable states remain explicit.                    |
| IT-123 | Hybrid retrieval           | Index multilingual chunks in PostgreSQL FTS/pgvector.                                                           | RRF returns stable cited snapshot results with tenant/project filters.                             |
| IT-124 | Topic pipeline             | Run embeddings, mutual-kNN, label adapter and analyst constraints.                                              | Deterministic memberships persist; labels/evidence/version remain separate.                        |
| IT-125 | Model provider degradation | Timeout/disable the model adapter during deterministic requests.                                                | Only AI-dependent Jobs fail/degrade; metrics/policy/sync remain available.                         |
| IT-126 | Structured analysis        | Provider returns valid and invalid cited JSON fixtures.                                                         | Only schema-valid, fully evidenced output reaches succeeded state.                                 |
| IT-127 | ADK HITL recovery          | Restart worker while a run awaits confirmation.                                                                 | Persisted run resumes awaiting the same unexpired action-bound decision.                           |
| IT-128 | ADK budget                 | Agent exceeds step, duration, output or cost bound.                                                             | Run terminates with the specific bounded error and no extra tool executes.                         |
| IT-129 | Audit append               | Perform successful, denied, stale and failed mutations.                                                         | Each required immutable redacted audit outcome is attributable and ordered.                        |
| IT-130 | Alert evaluation           | Recompute the same rule/facts under redelivery.                                                                 | Occurrence deduplication and shared/per-member states remain correct.                              |
| IT-131 | Export generation          | Generate CSV and evidence JSON and verify object/download lifecycle.                                            | Content is authorized, checksummed, locale-correct and inaccessible after 24 hours.                |
| IT-132 | Account deletion           | Delete a member with sessions, preferences, feedback and shared history.                                        | Personal data/sessions purge while shared facts retain an opaque actor.                            |
| IT-133 | Project deletion           | Delete a project with every canonical/object family populated.                                                  | It becomes unavailable first, purges resumably and leaves only minimal tombstone/audit.            |
| IT-134 | Telemetry propagation      | Trace API command through outbox, JetStream, worker, SQL, S3 and model tool.                                    | Correlation survives boundaries and forbidden payloads are absent.                                 |
| IT-135 | Readiness matrix           | Fail each required and optional dependency independently.                                                       | Health remains liveness-only; readiness/operations classify unavailable versus degraded correctly. |
| IT-136 | Generated contract drift   | Modify OpenAPI or SQL without regenerating outputs.                                                             | CI drift check fails; clean regeneration passes.                                                   |
| IT-137 | Race and shutdown          | Run collectors/consumers under race detector while cancelling and shutting down.                                | No race/leak occurs; claims stop before owned work drains.                                         |
| IT-138 | Backup restore             | Restore PostgreSQL and referenced S3 evidence into a clean deployment.                                          | Canonical references, checksums, Jobs and audit history reconcile.                                 |
| IT-139 | Same-origin protection     | Send unsafe cookie requests with absent/wrong Origin or Fetch Metadata.                                         | The API rejects them before application mutation.                                                  |
| IT-140 | Cursor pagination          | Page concurrent inserts using the same signed cursor snapshot rules.                                            | No unauthorized/filter-crossing data appears and ordering is stable.                               |
| IT-141 | Quota admission            | Submit concurrent expensive commands at the quota boundary.                                                     | Admission is atomic; accepted Jobs fit the limit and rejections create no work.                    |
| IT-142 | One-primary constraint     | Concurrent repository role changes target one project.                                                          | The database/application transaction ends with exactly one primary repository.                     |
| IT-143 | Provider quota reset       | Controlled source returns rate-limit reset then succeeds.                                                       | Worker checkpoints, waits with bounded jitter and resumes without duplicate facts.                 |
| IT-144 | Search evidence deletion   | Purge a project while cached/vector results exist.                                                              | Authorization/database filtering prevents any deleted evidence result.                             |
| IT-145 | Localization contract      | Render route/errors/status in both dictionaries with MSW.                                                       | No raw key appears and English fallback preserves route/action identity.                           |
| IT-146 | Accessibility components   | Render dialogs, grids, charts and async states with RTL/axe.                                                    | Keyboard/focus/name/status/table alternatives meet the frozen semantics.                           |

### HTTP operation contracts

Each success case exercises the generated OpenAPI operation through `net/http` and its real
application boundary. Each paired failure case proves authorization/validation/conditional behavior
and absence of side effects.

| Success ID | Failure ID | Operation                                                                                | Valid input                                                                                                                  | Success result                                                                    | Failure result                                                                                                                               |
| ---------- | ---------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| IT-147     | IT-148     | `GET /api/v1/catalog/projects?q=&cursor=&limit=`                                         | none                                                                                                                         | `200` page of `{id,name,slug,description,source_links}`                           | Malformed identifiers/query/cursor are rejected safely without protected data or implementation detail.                                      |
| IT-149     | IT-150     | `GET /api/v1/catalog/projects/{project_id}`                                              | none                                                                                                                         | `200` public catalog representation; `404 project_not_found`                      | Malformed identifiers/query/cursor are rejected safely without protected data or implementation detail.                                      |
| IT-151     | IT-152     | `GET /api/v1/session`                                                                    | session cookie optional                                                                                                      | `200` session/access representation                                               | Malformed identifiers/query/cursor are rejected safely without protected data or implementation detail.                                      |
| IT-153     | IT-154     | `POST /api/v1/session/logout`                                                            | CSRF token                                                                                                                   | `204`                                                                             | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-155     | IT-156     | `PATCH /api/v1/me/preferences`                                                           | `{"locale":"pt-BR","timezone":"America/Sao_Paulo"}`                                                                          | `200` member plus new ETag                                                        | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-157     | IT-158     | `POST /api/v1/me/deletion`                                                               | `{"confirmation":"DELETE MY ACCOUNT"}`                                                                                       | `202` Job                                                                         | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-159     | IT-160     | `GET /api/v1/admin/members?state=&role=&q=&cursor=&limit=`                               | none                                                                                                                         | `200` page of members/applicants                                                  | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-161     | IT-162     | `POST /api/v1/admin/members/{member_id}/approval`                                        | `{"decision":"approve","role":"viewer"}` or `{"decision":"reject"}`                                                          | `200` member plus ETag                                                            | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-163     | IT-164     | `PATCH /api/v1/admin/members/{member_id}`                                                | `{"role":"analyst"}` or `{"state":"suspended"}` with If-Match                                                                | `200` member plus ETag                                                            | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-165     | IT-166     | `GET /api/v1/admin/service-accounts?state=&q=&cursor=&limit=`                            | none                                                                                                                         | `200` page of locally bound service accounts; no token or secret fields           | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-167     | IT-168     | `POST /api/v1/admin/service-accounts`                                                    | `{"external_subject":"opi-exporter","name":"Portfolio exporter","role":"viewer","scopes":["projects:read","exports:write"]}` | `201` local service account plus ETag                                             | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-169     | IT-170     | `PATCH /api/v1/admin/service-accounts/{service_account_id}`                              | role, scope subset, or state with If-Match                                                                                   | `200` service account plus ETag                                                   | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-171     | IT-172     | `GET /api/v1/admin/audit?actor=&action=&resource=&from=&to=&cursor=&limit=`              | none                                                                                                                         | `200` page of immutable audit events                                              | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-173     | IT-174     | `GET /api/v1/admin/operations`                                                           | none                                                                                                                         | `200` redacted source/model capability, quota, health, and aggregate usage status | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-175     | IT-176     | `GET /api/v1/portfolio?window=90d&cutoff=`                                               | none                                                                                                                         | `200` panel summaries with independent status and evidence links                  | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-177     | IT-178     | `GET /api/v1/projects?state=active&q=&cursor=&limit=`                                    | none                                                                                                                         | `200` page of protected Project summaries                                         | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-179     | IT-180     | `POST /api/v1/projects`                                                                  | `{"repository_url":"https://github.com/temporalio/temporal","history_days":180}`                                             | `202` Project and initial-sync Job                                                | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-181     | IT-182     | `GET /api/v1/projects/{project_id}`                                                      | none                                                                                                                         | `200` Project, links, capabilities, freshness summary, ETag                       | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-183     | IT-184     | `PATCH /api/v1/projects/{project_id}`                                                    | editable identity fields with If-Match                                                                                       | `200` Project plus ETag                                                           | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-185     | IT-186     | `POST /api/v1/projects/{project_id}/transition`                                          | `{"to":"paused","reason":"Maintenance review"}` with If-Match                                                                | `202` Project and transition Job                                                  | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-187     | IT-188     | `POST /api/v1/projects/{project_id}/deletion`                                            | `{"confirmation":"DELETE temporal","reason":"Duplicate project"}` with If-Match                                              | `202` non-cancellable purge Job                                                   | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-189     | IT-190     | `GET /api/v1/projects/{project_id}/repositories?cursor=&limit=`                          | none                                                                                                                         | `200` page of Repository resources                                                | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-191     | IT-192     | `POST /api/v1/projects/{project_id}/repositories`                                        | `{"url":"https://github.com/temporalio/sdk-go","role":"sdk"}`                                                                | `201` Repository plus ETag                                                        | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-193     | IT-194     | `PATCH /api/v1/projects/{project_id}/repositories/{repository_id}`                       | `{"role":"primary"}` with If-Match                                                                                           | `200` Repository set with exactly one primary                                     | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-195     | IT-196     | `DELETE /api/v1/projects/{project_id}/repositories/{repository_id}`                      | If-Match                                                                                                                     | `204`; rejects removal of only primary                                            | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-197     | IT-198     | `GET /api/v1/projects/{project_id}/sources?kind=&state=&cursor=&limit=`                  | none                                                                                                                         | `200` page of source coverage/status resources                                    | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-199     | IT-200     | `POST /api/v1/projects/{project_id}/sources`                                             | typed public URL/package/feed/document source                                                                                | `201` Source plus ETag                                                            | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-201     | IT-202     | `PATCH /api/v1/projects/{project_id}/sources/{source_id}`                                | editable scope and collection limits with If-Match                                                                           | `200` Source plus ETag                                                            | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-203     | IT-204     | `DELETE /api/v1/projects/{project_id}/sources/{source_id}`                               | If-Match                                                                                                                     | `202` recalculation Job                                                           | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-205     | IT-206     | `GET /api/v1/projects/{project_id}/associations?status=&cursor=&limit=`                  | none                                                                                                                         | `200` page with method, confidence, evidence, decision version, correction        | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-207     | IT-208     | `POST /api/v1/projects/{project_id}/associations/{association_id}/correction`            | `{"action":"split","reason":"Different product"}`                                                                            | `202` correction and recalculation Job                                            | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-209     | IT-210     | `POST /api/v1/projects/{project_id}/syncs`                                               | `{"scope":"all"}` or selected source IDs                                                                                     | `202` new Job or `200` coalesced Job                                              | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-211     | IT-212     | `POST /api/v1/projects/{project_id}/history-requests`                                    | `{"from":"2025-08-20","reason":"Annual review"}`                                                                             | `202` Job; quota rejection is atomic                                              | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-213     | IT-214     | `GET /api/v1/projects/{project_id}/jobs?kind=&state=&cursor=&limit=`                     | none                                                                                                                         | `200` page of Jobs                                                                | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-215     | IT-216     | `GET /api/v1/jobs/{job_id}`                                                              | none                                                                                                                         | `200` Job; active Jobs include `Retry-After`                                      | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-217     | IT-218     | `GET /api/v1/jobs/{job_id}/events`                                                       | `Accept: text/event-stream`, optional `Last-Event-ID`                                                                        | `200` resumable `job.updated` SSE stream until terminal state                     | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-219     | IT-220     | `POST /api/v1/jobs/{job_id}/cancellation`                                                | `{"reason":"No longer needed"}`                                                                                              | `202` Job moving to cancelled                                                     | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-221     | IT-222     | `GET /api/v1/projects/{project_id}/metrics?dimension=&window=90d&cutoff=&cursor=&limit=` | none                                                                                                                         | `200` page of versioned metric results                                            | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-223     | IT-224     | `GET /api/v1/projects/{project_id}/metrics/{metric_name}?window=90d&cutoff=`             | none                                                                                                                         | `200` formula, value/status, unit, evidence, repositories, coverage, version      | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-225     | IT-226     | `GET /api/v1/projects/{project_id}/health?window=90d&cutoff=`                            | none                                                                                                                         | `200` seven dimensions and a visible secondary overall score whenever calculable  | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-227     | IT-228     | `GET /api/v1/projects/{project_id}/contributors?window=90d&cursor=&limit=`               | none                                                                                                                         | `200` page plus resolution coverage and concentration summary                     | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-229     | IT-230     | `GET /api/v1/projects/{project_id}/adoption?window=90d&cursor=&limit=`                   | none                                                                                                                         | `200` source-contextual indicators; no universal score                            | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-231     | IT-232     | `GET /api/v1/projects/{project_id}/security?window=365d&cursor=&limit=`                  | none                                                                                                                         | `200` public-evidence findings and explicit coverage limitations                  | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-233     | IT-234     | `POST /api/v1/comparisons`                                                               | `{"project_ids":["732684512931872768","732684513124761600"],"window":"90d","cutoff":"2026-08-20T14:35:00Z"}`                 | `201` immutable comparison                                                        | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-235     | IT-236     | `GET /api/v1/comparisons/{comparison_id}`                                                | none                                                                                                                         | `200` same-window matrix with incomparable/missing states preserved               | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-237     | IT-238     | `GET /api/v1/projects/{project_id}/trends?kind=observed&window=365d&cursor=&limit=`      | none                                                                                                                         | `200` page of observed trends or early warnings                                   | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-239     | IT-240     | `GET /api/v1/projects/{project_id}/recommendation?policy=default&window=90d&cutoff=`     | none                                                                                                                         | `200` one four-state deterministic evaluation                                     | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-241     | IT-242     | `GET /api/v1/policies?state=&cursor=&limit=`                                             | none                                                                                                                         | `200` page of policy families and active versions                                 | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-243     | IT-244     | `GET /api/v1/policies/{policy_id}/versions/{version}`                                    | none                                                                                                                         | `200` immutable rule tree and explanation labels                                  | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-245     | IT-246     | `POST /api/v1/policies`                                                                  | `{"name":"Production adoption","description":"...","rules":[...]}`                                                           | `201` draft policy version 1                                                      | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-247     | IT-248     | `POST /api/v1/policies/{policy_id}/versions`                                             | complete next rule tree                                                                                                      | `201` immutable draft version                                                     | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-249     | IT-250     | `POST /api/v1/policies/{policy_id}/versions/{version}/activation`                        | `{"reason":"Quarterly governance update"}`                                                                                   | `200` active version; prior evaluations remain attributed                         | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-251     | IT-252     | `GET /api/v1/radar?policy=default&window=90d`                                            | none                                                                                                                         | `200` Project placements with policy suggestion and effective placement           | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-253     | IT-254     | `POST /api/v1/radar/{project_id}/override`                                               | `{"ring":"assess","reason":"Pilot dependency","review_on":"2026-11-20"}`                                                     | `201` attributed override                                                         | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-255     | IT-256     | `DELETE /api/v1/radar/{project_id}/override`                                             | If-Match                                                                                                                     | `204`; policy placement becomes effective                                         | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-257     | IT-258     | `GET /api/v1/projects/{project_id}/topics?window=90d&cursor=&limit=`                     | none                                                                                                                         | `200` known/emerging topics with evidence, confidence, coverage, version          | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-259     | IT-260     | `POST /api/v1/projects/{project_id}/topics/{topic_id}/corrections`                       | rename, merge, split, or reassign plus reason                                                                                | `202` correction and reanalysis Job                                               | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-261     | IT-262     | `GET /api/v1/projects/{project_id}/releases?cursor=&limit=`                              | none                                                                                                                         | `200` page of releases and analysis status                                        | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-263     | IT-264     | `GET /api/v1/projects/{project_id}/releases/{release_id}`                                | none                                                                                                                         | `200` categorized claims linked to evidence                                       | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-265     | IT-266     | `POST /api/v1/projects/{project_id}/crawls`                                              | `{"source_ids":["732684513351254016"],"max_depth":3}`                                                                        | `202` crawl Job                                                                   | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-267     | IT-268     | `POST /api/v1/projects/{project_id}/knowledge/search`                                    | `{"query":"How are upgrades handled?","language":"en","limit":10}`                                                           | `200` cited snapshot results                                                      | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-269     | IT-270     | `POST /api/v1/projects/{project_id}/queries`                                             | `{"question":"What changed in maintenance risk?","window":"90d","language":"en"}`                                            | `202` analysis Job and Run                                                        | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-271     | IT-272     | `GET /api/v1/analysis-runs/{run_id}`                                                     | none                                                                                                                         | `200` immutable structured output, evidence, versions, usage, status              | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-273     | IT-274     | `POST /api/v1/analysis-runs/{run_id}/reruns`                                             | `{"language":"pt-BR","reason":"Review corrected topic"}`                                                                     | `202` new Run and Job                                                             | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-275     | IT-276     | `POST /api/v1/analysis-runs/{run_id}/feedback`                                           | `{"rating":"incorrect","comment":"The cited issue belongs to the SDK."}`                                                     | `201` immutable feedback                                                          | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-277     | IT-278     | `POST /api/v1/analysis-series/{series_id}/selection`                                     | `{"run_id":"732684513258979328"}`                                                                                            | `200` selected successful version plus ETag                                       | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-279     | IT-280     | `GET /api/v1/alerts?state=&project=&cursor=&limit=`                                      | none                                                                                                                         | `200` page of shared alert events plus requesting member's read state             | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-281     | IT-282     | `POST /api/v1/alert-rules`                                                               | typed condition, scope, severity, and deduplication window                                                                   | `201` rule plus ETag                                                              | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-283     | IT-284     | `PATCH /api/v1/alert-rules/{rule_id}`                                                    | complete editable fields with If-Match                                                                                       | `200` rule plus ETag                                                              | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-285     | IT-286     | `POST /api/v1/alerts/{alert_id}/read`                                                    | none                                                                                                                         | `204`; changes only current member state                                          | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-287     | IT-288     | `POST /api/v1/alerts/{alert_id}/transition`                                              | `{"to":"acknowledged","reason":"Investigating"}`                                                                             | `200` shared alert plus ETag                                                      | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-289     | IT-290     | `POST /api/v1/exports`                                                                   | resource, format `csv` or `evidence_json`, filters, locale                                                                   | `202` export Job                                                                  | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-291     | IT-292     | `GET /api/v1/exports/{export_id}`                                                        | none                                                                                                                         | `200` metadata and download URL that expires 24 hours after the Job succeeds      | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-293     | IT-294     | `GET /api/v1/exports/{export_id}/download`                                               | none                                                                                                                         | `200` file; `410 export_expired` after expiry                                     | A missing or insufficient principal is rejected before resource disclosure with the documented problem.                                      |
| IT-295     | IT-296     | `POST /api/v1/assistant/proposals`                                                       | typed natural-language proposal request and Idempotency-Key                                                                  | 201 awaiting-confirmation proposal or documented 422 action_not_allowed           | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |
| IT-297     | IT-298     | `POST /api/v1/assistant/proposals/{proposal_id}/confirmation`                            | single-use confirmation token and Idempotency-Key                                                                            | 201 executed proposal with result and audit event                                 | Invalid/missing required body, authorization, idempotency or conditional version is rejected with the documented problem and no side effect. |

## End-to-End Test Cases

### Persona journeys

| ID      | Story                                              | Journey and required outcome                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| ------- | -------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| E2E-001 | US-001: Browse the public catalog                  | AC-1: Given no authenticated session, when the Visitor opens the service, then only Project names, public descriptions, and public source links are visible. AC-2: Given a catalog with many Projects, when the Visitor searches or pages it, then matching public entries are returned without exposing derived intelligence. AC-3: Given a public Project entry, when the Visitor requests protected metrics or analyses, then the product invites authentication and does not disclose the protected response.                                                                                                       |
| E2E-002 | US-002: Authenticate and await approval            | AC-1: Given a Visitor chooses registration or sign-in, when shared Keycloak authentication succeeds, then the product identifies the external subject without creating product credentials. AC-2: Given an authenticated subject without membership, when the Applicant returns, then a pending approval page is shown and protected intelligence remains unavailable. AC-3: Given an Admin approves the Applicant, when the Applicant starts or refreshes a session, then the assigned local role and workspace become available.                                                                                      |
| E2E-003 | US-003: Govern workspace membership                | AC-1: Given pending applicants, when the Admin approves one, then exactly one Viewer, Analyst, or Admin role is assigned. AC-2: Given an existing member, when the Admin changes or suspends access, then subsequent requests enforce the new state and active access is revoked when required. AC-3: Given any membership action, when it completes, then actor, subject, prior state, new state, and outcome appear in the audit log.                                                                                                                                                                                 |
| E2E-004 | US-004: Manage preferences and remove an account   | AC-1: Given an approved member, when English or Portuguese and a timezone are selected, then fixed product content and displayed times use those preferences. AC-2: Given the member requests account deletion and confirms it, then sessions and personal data are removed while shared resources remain. AC-3: Given historical actions by a deleted member, when an Admin audits them, then they reference a stable opaque actor without the removed profile data.                                                                                                                                                   |
| E2E-005 | US-005: Understand the portfolio overview          | AC-1: Given active Projects, when the overview loads, then it summarizes health dimensions, recommendations, alerts, trends, important releases, freshness, and failures. AC-2: Given a summary item, when the Viewer follows it, then the corresponding evidence-bearing Project, comparison, alert, trend, or radar view opens. AC-3: Given partial or stale data, when the overview renders, then coverage and freshness are visible and missing evidence is not displayed as zero.                                                                                                                                  |
| E2E-006 | US-006: Register a project from a repository URL   | AC-1: Given a canonical public repository URL, when registration succeeds, then one Project and one typed primary Repository are created and initial collection is queued. AC-2: Given metadata is inferred, when the Project is shown, then the Analyst can review and edit the generated identity without changing source provenance. AC-3: Given the URL already belongs to a Project, when registration is attempted, then the existing Project opens instead of creating a duplicate.                                                                                                                              |
| E2E-007 | US-007: Curate a multi-repository project          | AC-1: Given a Project, when repositories are added, then each has one explicit supported role and exactly one repository remains primary. AC-2: Given project-level metrics, when multiple repositories contribute, then the aggregation boundary and contributing repositories are visible. AC-3: Given a repository role changes, when recalculation is required, then affected intelligence becomes visibly stale until refreshed.                                                                                                                                                                                   |
| E2E-008 | US-008: Correct automatic source associations      | AC-1: Given an automatic association, when inspected, then its source, resolution method, confidence, evidence, and decision version are visible. AC-2: Given an incorrect association, when the Analyst splits or reassigns it, then the correction is audited and retained against later automatic re-linking. AC-3: Given a correction invalidates derived results, when it completes, then affected metrics and analyses are marked stale and scheduled for recalculation.                                                                                                                                          |
| E2E-009 | US-009: Manage the project lifecycle               | AC-1: Given an active Project, when paused, then scheduled collection stops while existing intelligence remains readable. AC-2: Given a paused or active Project, when archived, then it becomes read-only and leaves default active views; restoration returns it to a valid non-deleted state. AC-3: Given explicit permanent-deletion confirmation, when deletion completes, then owned data is purged and only the minimal audit tombstone remains.                                                                                                                                                                 |
| E2E-010 | US-010: Request and monitor synchronization        | AC-1: Given an active Project, when its schedule is due, then collection begins without user action. AC-2: Given an authorized manual request, when accepted, then duplicate work is coalesced and the Project shows progress, last attempt, last success, next run, and any failure reason. AC-3: Given a transient failure or restart, when service resumes, then collection continues from a visible checkpoint without duplicating published data.                                                                                                                                                                  |
| E2E-011 | US-011: Understand history and freshness           | AC-1: Given a new source, when no override exists, then the initial backfill target is 180 days and older still-open issues and pull requests are retained. AC-2: Given operator limits permit it, when an Analyst requests a longer range, then the requested target and eventual actual coverage are shown. AC-3: Given every metric or analysis, when viewed, then source coverage, last success, cutoff, and stale or incomplete state are visible.                                                                                                                                                                 |
| E2E-012 | US-012: Operate public-data integrations           | AC-1: Given an operator-managed read token, when a source request executes, then only public resources are accepted even if the credential could see more. AC-2: Given an Admin inspects source health, when status loads, then provider, public capability, quota, and last validation appear without secret values. AC-3: Given a credential is missing or invalid, when collection is attempted, then affected sources degrade explicitly while unrelated deterministic intelligence remains available.                                                                                                              |
| E2E-013 | US-013: Inspect metrics and health dimensions      | AC-1: Given a metric, when opened, then its value, formula version, unit, observation window, coverage, contributing sources, and missing-data rules are visible. AC-2: Given a Project, when health is shown, then Activity, Community, Maintenance, Concentration, Stability, Security, and Adoption remain independently inspectable. AC-3: Given an overall score that can be calculated with sufficient coverage, when shown, then it is secondary to dimensions and exposes weights, version, factors, window, and evidence without a binary healthy/unhealthy label.                                             |
| E2E-014 | US-014: Evaluate contributor sustainability        | AC-1: Given a selected window, when contributor intelligence loads, then active and new contributors, retention, maintainer count, and top-one/top-three contribution shares are shown. AC-2: Given cross-source identities, when contributors are aggregated, then only verified or Analyst-confirmed links combine accounts and the resolution coverage is visible. AC-3: Given an identity correction, when recalculation completes, then concentration metrics update without rewriting source provenance.                                                                                                          |
| E2E-015 | US-015: Interpret adoption and security evidence   | AC-1: Given registry evidence, when adoption is shown, then raw values, time-window changes, provenance, coverage, and only within-population normalization are visible. AC-2: Given security evidence, when the dimension is shown, then public advisories, security releases, changelogs, issues, and provider metadata identify their sources and dates. AC-3: Given a policy or score uses either dimension, when inspected, then unavailable evidence and formula treatment are explicit.                                                                                                                          |
| E2E-016 | US-016: Compare projects in one window             | AC-1: Given two to five Projects and a preset or valid custom window, when comparison runs, then all results use the same interval and cutoff. AC-2: Given differences in source coverage, when results render, then only comparable signals are normalized and unknown or not-applicable remains distinct from zero. AC-3: Given a compared value or narrative, when opened, then its metric version, evidence, freshness, and Project aggregation boundary are visible.                                                                                                                                               |
| E2E-017 | US-017: Distinguish trends and early warnings      | AC-1: Given adequate history, when a trend is reported, then observation and baseline windows, method version, magnitude, direction, and evidence are visible. AC-2: Given a predictive warning, when opened, then it is labeled as forecast and shows horizon, confidence, calibration or known error, inputs, coverage, and model version. AC-3: Given inadequate evidence, when detection runs, then it returns insufficient data rather than a neutral or fabricated signal.                                                                                                                                        |
| E2E-018 | US-018: Receive an adoption recommendation         | AC-1: Given a Project, policy version, and observation window, when evaluation completes, then the result is `recommended`, `conditional`, `not_recommended`, or `insufficient_data`. AC-2: Given any result, when inspected, then policy owner/version, inputs, weights, thresholds, decisive factors, evidence, freshness, and missing data are visible. AC-3: Given a conditional result, when displayed, then its constraints or mitigations are explicit; an LLM explanation cannot alter the outcome.                                                                                                             |
| E2E-019 | US-019: Author and version adoption policies       | AC-1: Given the transparent default policy, when cloned, then the Admin can change explicit thresholds, weights, required evidence, missing-data rules, and radar mapping in a draft. AC-2: Given a valid draft, when published, then it receives an immutable version and can be selected for new evaluations. AC-3: Given a published version is retired, when existing results are viewed, then they retain the retired version and remain reproducible.                                                                                                                                                             |
| E2E-020 | US-020: Govern the technology radar                | AC-1: Given a selected Project and recommendation, when added to the radar, then the suggested ring follows the visible mapping of the exact policy version. AC-2: Given organizational context differs, when an Analyst overrides the ring, then justification, author, owner, and review date are required and the original recommendation remains visible. AC-3: Given policy evidence changes, when the radar is viewed, then stale suggestions and overdue overrides are called out without moving a manual override silently.                                                                                     |
| E2E-021 | US-021: Explore issue and discussion topics        | AC-1: Given a time window, when topic intelligence runs, then known taxonomy categories and emerging topics show prevalence, change, representative evidence, confidence, and analysis version. AC-2: Given a wrong grouping, when the Analyst renames, merges, splits, or reassigns it, then the correction is attributed and available to later reprocessing and evaluation. AC-3: Given source coverage differs, when topics are compared over time or across Projects, then the contributing content and coverage are explicit.                                                                                     |
| E2E-022 | US-022: Understand a release                       | AC-1: Given a collected release, when analysis succeeds, then changes use known categories and each claim links to changelog, referenced pull request, diff metadata, issue, or other source evidence. AC-2: Given multiple analysis runs, when viewing the release, then the presented version and its model, prompt, execution time, language, and status are visible. AC-3: Given no AI provider, when the release opens, then deterministic release metadata remains available and analysis shows unavailable without blocking collection.                                                                          |
| E2E-023 | US-023: Search project documentation               | AC-1: Given linked public URLs, when collected, then only allowed domains, depth, size, frequency, and content types are followed, with robots behavior honored. AC-2: Given a search or RAG answer, when results appear, then every claim links to the original URL and snapshot time; translated text is labeled and the original remains accessible. AC-3: Given a page changes, when recollected, then a new provenance-bearing snapshot supports later analysis without silently rewriting past evidence.                                                                                                          |
| E2E-024 | US-024: Ask natural-language questions             | AC-1: Given a question, when interpreted, then the response identifies scope, Projects, window, data cutoff, structured findings, and citations to product evidence. AC-2: Given ambiguous scope, when the question could produce materially different answers, then the product requests clarification rather than guessing. AC-3: Given missing or stale evidence, when answering, then uncertainty or insufficient data is stated and unsupported claims are omitted. AC-4: Given the request implies a supported mutation, then it becomes a typed HITL proposal governed by US-025 rather than executing directly. |
| E2E-025 | US-025: Approve a non-destructive assistant action | AC-1: Given a supported request, when the assistant proposes an action, then operation, resources, values, expected effect, quota impact, and expiration are displayed before execution. AC-2: Given explicit approval of an unchanged proposal, when execution succeeds, then the result and audit entry identify the requesting Analyst and proposal. AC-3: Given an Admin-only, credential, policy-authoring, archive, deletion, or otherwise destructive request, then the assistant refuses execution and points to the conventional surface.                                                                      |
| E2E-026 | US-026: Review AI analysis versions                | AC-1: Given an analysis run, when opened, then structured output, evidence, provider/model, prompt version, language, execution time, status, and stale state are visible. AC-2: Given a rerun, when it finishes, then it creates a new immutable version without overwriting previous output. AC-3: Given inaccurate output, when an Analyst flags it, then attributed feedback is stored and a different version may be selected without editing generated content.                                                                                                                                                   |
| E2E-027 | US-027: Configure and resolve shared alerts        | AC-1: Given a valid rule, when its evidence condition is met, then one deduplicated occurrence shows severity, rule version, Project, window, evidence, and detected time. AC-2: Given an occurrence, when one user reads it, then only that user's read state changes; when an Analyst acknowledges, resolves, or dismisses it with justification, the shared state changes. AC-3: Given repeated evidence inside cooldown, when evaluated, then it updates or suppresses the existing occurrence according to the visible rule rather than creating noise.                                                            |
| E2E-028 | US-028: Export results and evidence                | AC-1: Given a metrics, snapshot, or comparison view, when CSV is requested, then exported rows use the selected scope, window, cutoff, units, and missing-data representation. AC-2: Given an evidence-package request, when JSON is produced, then it includes scope, coverage, provenance, formulas, versions, policy context, and analysis references needed to interpret it. AC-3: Given English or Portuguese preference, when export is generated, then stable machine fields remain documented and human labels follow the requested language where supported.                                                   |
| E2E-029 | US-029: Investigate the audit log                  | AC-1: Given an auditable action, when recorded, then actor, time, resource, action, prior/new state where safe, and outcome are present without secrets or sensitive payloads. AC-2: Given filters for actor, resource, action, outcome, or time, when applied, then matching events are returned in stable chronological order. AC-3: Given a deleted user or Project, when history is viewed, then only the defined opaque actor or Project tombstone remains.                                                                                                                                                        |
| E2E-030 | US-030: Operate model providers and degradation    | AC-1: Given an operator-configured external or local provider, when available, then approved AI capabilities use the workspace-active model configuration consistently. AC-2: Given an Admin inspects AI status, when the status loads, then active provider/model, supported capabilities, health, and aggregate usage/cost are visible without secrets. AC-3: Given provider failure or no provider, when users open the product, then collection, metrics, health, policies, radar, comparisons, and deterministic trends continue while AI surfaces explain their unavailable or stale state.                       |
| E2E-031 | US-031: Use the bilingual mobile-first product     | AC-1: Given a supported mobile viewport, when any Viewer, Analyst, or Admin journey is opened, then its information and controls remain complete without requiring a desktop-only fallback. AC-2: Given a language preference, when navigating, then fixed UI, validation, status, errors, and product-generated analysis use that language while source evidence remains available in original form. AC-3: Given charts or color-coded states, when used with keyboard, screen reader, zoom, or reduced motion settings, then equivalent text/table information and non-color cues remain available.                   |
| E2E-032 | US-032: Use the API through a service identity     | AC-1: Given a valid Keycloak bearer token, when its subject matches an approved local service account, then the API enforces that account's Viewer or Analyst role and scope subset. AC-2: Given an authorized API request, when it completes, then the audit event attributes the action to the service account and never to a fabricated human actor. AC-3: Given a local service account is suspended or its scope changes, when a later bearer request arrives, then the current local state is enforced without requiring Keycloak credential mutation.                                                            |

### Frozen UI surfaces

| ID      | Surface                                             | Required outcome                                                                                                                                                     |
| ------- | --------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| E2E-033 | S1 Application shell and localized routing          | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-034 | S2 Public catalog and protected teaser              | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-035 | S3 Sign-in, pending access and account              | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-036 | S4 Portfolio overview                               | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-037 | S5 Project catalog, registration and identity       | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-038 | S6 Project lifecycle                                | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-039 | S7 Repositories, sources and associations           | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-040 | S8 Jobs, synchronization and history                | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-041 | S9 Metrics and health                               | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-042 | S10 Contributor intelligence                        | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-043 | S11 Adoption and security                           | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-044 | S12 Comparison workspace                            | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-045 | S13 Trends and early warnings                       | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-046 | S14 Recommendation and policy governance            | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-047 | S15 Technology radar                                | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-048 | S16 Issue and discussion topics                     | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-049 | S17 Release intelligence                            | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-050 | S18 Documentation knowledge                         | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-051 | S19 Natural-language analysis and HITL assistant    | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-052 | S20 AI run governance                               | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-053 | S21 Alerts                                          | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-054 | S22 Exports                                         | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |
| E2E-055 | S23 Members, service accounts, audit and operations | Complete the frozen primary journey in en and pt-BR with keyboard-only use, narrow and wide viewports, loading/empty/error/stale states, and no unauthorized action. |

## Coverage Matrix

### User stories and edge cases

Every acceptance-criteria set has a persona journey. Every approved edge-case row has its own
directly traceable unit or integration test ID.

| Catalog item | Behavior                                                                                                                            | Test IDs |
| ------------ | ----------------------------------------------------------------------------------------------------------------------------------- | -------- |
| US-001       | Browse the public catalog                                                                                                           | E2E-001  |
| US-001 EC-1  | Malformed search text is treated as text or rejected safely without query details.                                                  | UT-001   |
| US-001 EC-2  | An empty catalog shows an onboarding explanation rather than a broken grid.                                                         | UT-002   |
| US-001 EC-3  | Large catalogs remain paginated and never return an unbounded response.                                                             | UT-003   |
| US-001 EC-4  | Anonymous deep links to protected views reveal no protected fields.                                                                 | UT-004   |
| US-001 EC-5  | A Project archived during browsing disappears on refresh without corrupting the page.                                               | IT-001   |
| US-001 EC-6  | A failed page request preserves the last visible page and offers retry.                                                             | IT-002   |
| US-001 EC-7  | Repeating the same search returns the same public representation.                                                                   | UT-005   |
| US-001 EC-8  | Opening a stale bookmarked page resolves by stable identity or shows not found.                                                     | UT-006   |
| US-001 EC-9  | Paused Projects remain public; archived and deleted Projects do not.                                                                | UT-007   |
| US-001 EC-10 | Search and pagination remain usable at 100 times the expected initial catalog.                                                      | IT-003   |
| US-002       | Authenticate and await approval                                                                                                     | E2E-002  |
| US-002 EC-1  | Invalid or unverifiable identity claims are rejected without creating membership.                                                   | UT-008   |
| US-002 EC-2  | A valid identity with no local profile enters pending state, never an implicit role.                                                | UT-009   |
| US-002 EC-3  | Registration and callback attempts are rate-limited with a clear retry message.                                                     | UT-010   |
| US-002 EC-4  | An authenticated but pending Applicant cannot use protected APIs or exports.                                                        | UT-011   |
| US-002 EC-5  | Simultaneous first logins create one pending membership request.                                                                    | IT-004   |
| US-002 EC-6  | An interrupted authentication flow can restart without a partial product session.                                                   | IT-005   |
| US-002 EC-7  | Repeated login preserves one pending request and does not notify Admins repeatedly.                                                 | UT-012   |
| US-002 EC-8  | A callback without a matching login flow is rejected safely.                                                                        | UT-013   |
| US-002 EC-9  | A suspended or rejected identity cannot return to pending without Admin action.                                                     | UT-014   |
| US-002 EC-10 | A burst of applicants does not expose data or make approved sessions unavailable.                                                   | IT-006   |
| US-003       | Govern workspace membership                                                                                                         | E2E-003  |
| US-003 EC-1  | An unknown role or malformed subject is rejected without changing membership.                                                       | UT-015   |
| US-003 EC-2  | With no pending users, the review view shows an empty state.                                                                        | UT-016   |
| US-003 EC-3  | Membership lists are searchable and paginated at high user counts.                                                                  | UT-017   |
| US-003 EC-4  | Analysts and Viewers cannot approve, promote, suspend, or inspect applicant details.                                                | UT-018   |
| US-003 EC-5  | Conflicting approvals use the latest valid state and report the stale action.                                                       | IT-007   |
| US-003 EC-6  | A failed role change leaves the prior role effective and reports failure.                                                           | IT-008   |
| US-003 EC-7  | Reapproving the same user is idempotent and does not create duplicate membership.                                                   | UT-019   |
| US-003 EC-8  | A role cannot be assigned before the external subject is known.                                                                     | UT-020   |
| US-003 EC-9  | The last active Admin cannot remove or suspend their own Admin access.                                                              | UT-021   |
| US-003 EC-10 | Bulk applicant volume remains operable without bulk implicit approval.                                                              | IT-009   |
| US-004       | Manage preferences and remove an account                                                                                            | E2E-004  |
| US-004 EC-1  | Unsupported locales or timezones are rejected with supported choices.                                                               | UT-022   |
| US-004 EC-2  | Missing preferences use English and UTC until the member selects alternatives.                                                      | UT-023   |
| US-004 EC-3  | Rapid preference changes are bounded and the latest confirmed value wins.                                                           | UT-024   |
| US-004 EC-4  | A member cannot edit or delete another member's profile.                                                                            | UT-025   |
| US-004 EC-5  | Concurrent deletion and profile edits resolve to deletion without restoring data.                                                   | IT-010   |
| US-004 EC-6  | An interrupted deletion confirmation does not delete the account.                                                                   | IT-011   |
| US-004 EC-7  | Repeating deletion after success returns a non-sensitive completed outcome.                                                         | UT-026   |
| US-004 EC-8  | Deletion cannot execute before explicit confirmation.                                                                               | UT-027   |
| US-004 EC-9  | A suspended member may remove their account but cannot regain workspace access.                                                     | UT-028   |
| US-004 EC-10 | Preference application does not require loading all workspace members.                                                              | IT-012   |
| US-005       | Understand the portfolio overview                                                                                                   | E2E-005  |
| US-005 EC-1  | Invalid filters are rejected or reset with an explanation.                                                                          | UT-029   |
| US-005 EC-2  | A portfolio with no active Projects shows role-appropriate next steps.                                                              | UT-030   |
| US-005 EC-3  | Summary lists truncate predictably and link to paginated full views.                                                                | UT-031   |
| US-005 EC-4  | Viewers see intelligence but no mutation controls.                                                                                  | UT-032   |
| US-005 EC-5  | New snapshots do not mix incompatible calculation versions in one rendered result.                                                  | IT-013   |
| US-005 EC-6  | One failed panel does not erase deterministic panels that loaded successfully.                                                      | IT-014   |
| US-005 EC-7  | Refreshing does not trigger collection or AI work implicitly.                                                                       | UT-033   |
| US-005 EC-8  | Deep links work without visiting the overview first.                                                                                | UT-034   |
| US-005 EC-9  | Archived Projects disappear from active summaries but remain available in archives.                                                 | UT-035   |
| US-005 EC-10 | The overview uses aggregation and pagination rather than rendering every Project at once.                                           | IT-015   |
| US-006       | Register a project from a repository URL                                                                                            | E2E-006  |
| US-006 EC-1  | Malformed, unsupported, private, or hostile URLs are rejected with a safe reason.                                                   | UT-036   |
| US-006 EC-2  | A blank URL cannot create a Project.                                                                                                | UT-037   |
| US-006 EC-3  | Operational quotas reject excess registration before creating partial state.                                                        | UT-038   |
| US-006 EC-4  | Viewers and pending users cannot register Projects.                                                                                 | UT-039   |
| US-006 EC-5  | Concurrent registration of the same canonical URL creates one Project.                                                              | IT-016   |
| US-006 EC-6  | Failure after creation shows recoverable collection state, not a duplicate prompt.                                                  | IT-017   |
| US-006 EC-7  | Retrying a successful request resolves to the same Project.                                                                         | UT-040   |
| US-006 EC-8  | Additional repositories cannot be attached before their Project exists.                                                             | UT-041   |
| US-006 EC-9  | A URL belonging to an archived Project offers restore rather than duplication.                                                      | UT-042   |
| US-006 EC-10 | Registration remains responsive while other Projects are backfilling.                                                               | IT-018   |
| US-007       | Curate a multi-repository project                                                                                                   | E2E-007  |
| US-007 EC-1  | Unsupported roles or duplicate canonical URLs are rejected.                                                                         | UT-043   |
| US-007 EC-2  | A Project cannot lose its only primary Repository without replacement.                                                              | UT-044   |
| US-007 EC-3  | Repository counts obey operational limits with no partial attachment.                                                               | UT-045   |
| US-007 EC-4  | Viewers cannot add, remove, or retype repositories.                                                                                 | UT-046   |
| US-007 EC-5  | Concurrent primary changes result in exactly one primary Repository.                                                                | IT-019   |
| US-007 EC-6  | An interrupted edit preserves the last valid repository set.                                                                        | IT-020   |
| US-007 EC-7  | Reattaching an existing repository returns its current association.                                                                 | UT-047   |
| US-007 EC-8  | A replacement primary is validated before the old primary loses its role.                                                           | UT-048   |
| US-007 EC-9  | Repositories of archived Projects cannot be edited until restoration.                                                               | UT-049   |
| US-007 EC-10 | Projects with many repositories summarize them and provide pagination or filtering.                                                 | IT-021   |
| US-008       | Correct automatic source associations                                                                                               | E2E-008  |
| US-008 EC-1  | A target Project that cannot accept the source is rejected before reassignment.                                                     | UT-050   |
| US-008 EC-2  | Associations without sufficient evidence remain visibly unresolved.                                                                 | UT-051   |
| US-008 EC-3  | Large review queues are filterable and paginated.                                                                                   | UT-052   |
| US-008 EC-4  | Viewers can inspect provenance but cannot correct associations.                                                                     | UT-053   |
| US-008 EC-5  | Concurrent corrections detect stale source ownership and preserve one valid result.                                                 | IT-022   |
| US-008 EC-6  | A failed correction leaves the prior association and derived status intact.                                                         | IT-023   |
| US-008 EC-7  | Repeating the same correction does not enqueue duplicate recalculations.                                                            | UT-054   |
| US-008 EC-8  | A split completes before downstream recalculation publishes new results.                                                            | UT-055   |
| US-008 EC-9  | Deleted Projects cannot receive reassigned sources.                                                                                 | UT-056   |
| US-008 EC-10 | Corrections invalidate only affected evidence, not the entire workspace.                                                            | IT-024   |
| US-009       | Manage the project lifecycle                                                                                                        | E2E-009  |
| US-009 EC-1  | Unknown lifecycle transitions are rejected with allowed actions.                                                                    | UT-057   |
| US-009 EC-2  | Deletion confirmation without the required Project identity does nothing.                                                           | UT-058   |
| US-009 EC-3  | Bulk lifecycle operations are not implied by a single-Project confirmation.                                                         | UT-059   |
| US-009 EC-4  | Analysts and Viewers cannot pause, archive, restore, or permanently delete a Project; every lifecycle transition is Admin-only.     | UT-060   |
| US-009 EC-5  | Collection racing with deletion cannot publish data after the deletion guard takes effect.                                          | IT-025   |
| US-009 EC-6  | A partial purge remains visibly in progress and resumes safely.                                                                     | IT-026   |
| US-009 EC-7  | Repeating pause or archive is idempotent; repeating deletion reveals no purged data.                                                | UT-061   |
| US-009 EC-8  | A Project cannot restore after permanent deletion.                                                                                  | UT-062   |
| US-009 EC-9  | Archived Projects reject edits, sync requests, and analysis requests.                                                               | UT-063   |
| US-009 EC-10 | Purging one large Project does not block reading unrelated Projects.                                                                | IT-027   |
| US-010       | Request and monitor synchronization                                                                                                 | E2E-010  |
| US-010 EC-1  | Unsupported refresh scopes are rejected before work is queued.                                                                      | UT-064   |
| US-010 EC-2  | A source without a checkpoint begins its configured initial backfill.                                                               | UT-065   |
| US-010 EC-3  | Quota or concurrency exhaustion returns queued or delayed status, not silent loss.                                                  | UT-066   |
| US-010 EC-4  | Viewers can inspect status but cannot request refresh.                                                                              | UT-067   |
| US-010 EC-5  | Simultaneous refresh requests coalesce into one compatible run.                                                                     | IT-028   |
| US-010 EC-6  | Cancellation or restart preserves the last durable checkpoint.                                                                      | IT-029   |
| US-010 EC-7  | Replaying a completed collection does not duplicate canonical records.                                                              | UT-068   |
| US-010 EC-8  | Analysis does not publish as current before required collection and metrics complete.                                               | UT-069   |
| US-010 EC-9  | Paused, archived, or deleting Projects reject new synchronization.                                                                  | UT-070   |
| US-010 EC-10 | Workspace-wide backfills remain bounded and expose queue position or delay.                                                         | IT-030   |
| US-011       | Understand history and freshness                                                                                                    | E2E-011  |
| US-011 EC-1  | End dates before start dates or future-only ranges are rejected.                                                                    | UT-071   |
| US-011 EC-2  | No collected history yields insufficient data, never zero activity.                                                                 | UT-072   |
| US-011 EC-3  | Requests beyond provider or operator limits show the maximum allowed range.                                                         | UT-073   |
| US-011 EC-4  | Viewers cannot change backfill targets.                                                                                             | UT-074   |
| US-011 EC-5  | Overlapping range extensions coalesce into the broadest valid target.                                                               | IT-031   |
| US-011 EC-6  | Partial backfill publishes its actual boundary and resumable state.                                                                 | IT-032   |
| US-011 EC-7  | Requesting an already covered range does not recollect it unnecessarily.                                                            | UT-075   |
| US-011 EC-8  | A range extension does not rewrite older snapshots before data is available.                                                        | UT-076   |
| US-011 EC-9  | Archived Projects retain coverage metadata without scheduling extension.                                                            | UT-077   |
| US-011 EC-10 | Long histories are summarized and paginated without truncating coverage disclosure.                                                 | IT-033   |
| US-012       | Operate public-data integrations                                                                                                    | E2E-012  |
| US-012 EC-1  | Over-privileged or malformed credentials fail validation without echoing them.                                                      | UT-078   |
| US-012 EC-2  | Missing optional credentials show anonymous capability or unavailable status.                                                       | UT-079   |
| US-012 EC-3  | Provider rate limits delay work with reset context rather than causing retry storms.                                                | UT-080   |
| US-012 EC-4  | End users cannot submit, retrieve, or select source credentials.                                                                    | UT-081   |
| US-012 EC-5  | Credential rotation does not mix identities within one source request.                                                              | IT-034   |
| US-012 EC-6  | Rotation failure preserves the last valid configuration or reports no active credential.                                            | IT-035   |
| US-012 EC-7  | Revalidation is safe and does not trigger collection.                                                                               | UT-082   |
| US-012 EC-8  | A token is validated before it becomes active.                                                                                      | UT-083   |
| US-012 EC-9  | A source that becomes private stops collection and is marked unavailable.                                                           | UT-084   |
| US-012 EC-10 | Quota reporting aggregates safely across many Projects without leaking request details.                                             | IT-036   |
| US-013       | Inspect metrics and health dimensions                                                                                               | E2E-013  |
| US-013 EC-1  | Unsupported metric windows are rejected with valid choices.                                                                         | UT-085   |
| US-013 EC-2  | Missing required evidence yields unavailable or insufficient data, never zero.                                                      | UT-086   |
| US-013 EC-3  | Large evidence sets summarize and link to bounded pages.                                                                            | UT-087   |
| US-013 EC-4  | Pending and anonymous users cannot read metric values.                                                                              | UT-088   |
| US-013 EC-5  | Recalculation publishes one internally consistent metric-version snapshot.                                                          | IT-037   |
| US-013 EC-6  | Failed recalculation leaves the prior snapshot visible and marked stale.                                                            | IT-038   |
| US-013 EC-7  | Recalculating identical inputs and versions yields identical results.                                                               | UT-089   |
| US-013 EC-8  | An overall score cannot publish before all required dimension results resolve.                                                      | UT-090   |
| US-013 EC-9  | Archived Projects retain historical metrics but receive no new snapshots.                                                           | UT-091   |
| US-013 EC-10 | Time series remain usable at 100 times the initial history through aggregation.                                                     | IT-039   |
| US-014       | Evaluate contributor sustainability                                                                                                 | E2E-014  |
| US-014 EC-1  | Invalid contributor filters or windows are rejected safely.                                                                         | UT-092   |
| US-014 EC-2  | No contributor evidence yields insufficient data rather than total concentration.                                                   | UT-093   |
| US-014 EC-3  | Contributor lists paginate and do not expose private email addresses.                                                               | UT-094   |
| US-014 EC-4  | Only Analysts can confirm or split contributor identities.                                                                          | UT-095   |
| US-014 EC-5  | Concurrent identity corrections cannot leave one account linked twice.                                                              | IT-040   |
| US-014 EC-6  | Failed recalculation leaves prior values stale rather than partially updated.                                                       | IT-041   |
| US-014 EC-7  | Reconfirming the same verified link is idempotent.                                                                                  | UT-096   |
| US-014 EC-8  | Identity linkage completes before aggregate concentration republishes.                                                              | UT-097   |
| US-014 EC-9  | Deleted source accounts remain historical evidence with source status.                                                              | UT-098   |
| US-014 EC-10 | High-volume contributor histories use bounded detail without changing aggregates.                                                   | IT-042   |
| US-015       | Interpret adoption and security evidence                                                                                            | E2E-015  |
| US-015 EC-1  | Incomparable registry units cannot be forced into one universal rank.                                                               | UT-099   |
| US-015 EC-2  | No advisory or registry data is unknown, not evidence of safety or no adoption.                                                     | UT-100   |
| US-015 EC-3  | Large advisory and package histories are paginated and windowed.                                                                    | UT-101   |
| US-015 EC-4  | Protected adoption and security intelligence requires approved membership.                                                          | UT-102   |
| US-015 EC-5  | New registry data cannot mix cutoffs within one published snapshot.                                                                 | IT-043   |
| US-015 EC-6  | A failed registry or advisory source leaves other evidence visible with stale status.                                               | IT-044   |
| US-015 EC-7  | Reingestion of one advisory or package sample does not duplicate it.                                                                | UT-103   |
| US-015 EC-8  | Normalization runs only after source population context is available.                                                               | UT-104   |
| US-015 EC-9  | Withdrawn advisories retain provenance and display their withdrawn status.                                                          | UT-105   |
| US-015 EC-10 | Cross-registry portfolios retain source-specific context at high volume.                                                            | IT-045   |
| US-016       | Compare projects in one window                                                                                                      | E2E-016  |
| US-016 EC-1  | Fewer than two, more than five, duplicates, or invalid windows are rejected.                                                        | UT-106   |
| US-016 EC-2  | A Project with no comparable evidence remains in the view as insufficient data.                                                     | UT-107   |
| US-016 EC-3  | Evidence details paginate without truncating the comparison conclusion silently.                                                    | UT-108   |
| US-016 EC-4  | Anonymous and pending users cannot run comparisons.                                                                                 | UT-109   |
| US-016 EC-5  | Metrics updated during a comparison do not create mixed cutoffs.                                                                    | IT-046   |
| US-016 EC-6  | Partial failure identifies unavailable Projects and preserves completed deterministic data.                                         | IT-047   |
| US-016 EC-7  | The same inputs and versions yield the same comparison.                                                                             | UT-110   |
| US-016 EC-8  | A comparison cannot run before every Project identity resolves.                                                                     | UT-111   |
| US-016 EC-9  | Deleted Projects make saved comparisons unavailable; archived Projects remain historical.                                           | UT-112   |
| US-016 EC-10 | Many saved comparisons remain searchable and paginated.                                                                             | IT-048   |
| US-017       | Distinguish trends and early warnings                                                                                               | E2E-017  |
| US-017 EC-1  | Invalid baselines or forecast horizons are rejected.                                                                                | UT-113   |
| US-017 EC-2  | Sparse history produces insufficient data with the minimum requirement shown.                                                       | UT-114   |
| US-017 EC-3  | Signal histories paginate and bounded detectors respect configured horizons.                                                        | UT-115   |
| US-017 EC-4  | Protected trends and warnings are unavailable to anonymous or pending users.                                                        | UT-116   |
| US-017 EC-5  | One signal version publishes against one consistent input snapshot.                                                                 | IT-049   |
| US-017 EC-6  | Failed prediction does not suppress valid observed trends.                                                                          | IT-050   |
| US-017 EC-7  | Deterministic trend reruns reproduce the same result from the same inputs.                                                          | UT-117   |
| US-017 EC-8  | Explanations cannot publish before the underlying signal exists.                                                                    | UT-118   |
| US-017 EC-9  | Superseded warnings retain history and outcome-evaluation status.                                                                   | UT-119   |
| US-017 EC-10 | Detection remains bounded across the portfolio and exposes delayed status.                                                          | IT-051   |
| US-018       | Receive an adoption recommendation                                                                                                  | E2E-018  |
| US-018 EC-1  | Inactive or incompatible policy versions cannot be evaluated.                                                                       | UT-120   |
| US-018 EC-2  | Missing required evidence yields `insufficient_data`.                                                                               | UT-121   |
| US-018 EC-3  | Evidence displays are bounded but retain links to every decisive input.                                                             | UT-122   |
| US-018 EC-4  | Viewers may read results; only Analysts/Admins can select policies for new evaluations.                                             | UT-123   |
| US-018 EC-5  | A policy activation during evaluation does not change the selected version.                                                         | IT-052   |
| US-018 EC-6  | Failed explanation leaves the deterministic result available.                                                                       | IT-053   |
| US-018 EC-7  | Identical policy, inputs, and versions reproduce the outcome.                                                                       | UT-124   |
| US-018 EC-8  | Evaluation waits for required metric versions or reports stale prerequisites.                                                       | UT-125   |
| US-018 EC-9  | Results retain their original policy version after supersession.                                                                    | UT-126   |
| US-018 EC-10 | Portfolio evaluation queues remain bounded and expose progress.                                                                     | IT-054   |
| US-019       | Author and version adoption policies                                                                                                | E2E-019  |
| US-019 EC-1  | Contradictory outcomes, invalid weights, or unknown metrics block publication.                                                      | UT-127   |
| US-019 EC-2  | Required evidence rules cannot be blank when an outcome depends on them.                                                            | UT-128   |
| US-019 EC-3  | Policy lists and version histories are paginated.                                                                                   | UT-129   |
| US-019 EC-4  | Analysts may select but cannot create, modify, publish, or retire policies.                                                         | UT-130   |
| US-019 EC-5  | Concurrent edits detect stale drafts instead of overwriting changes.                                                                | IT-055   |
| US-019 EC-6  | Failed publication leaves the draft editable and no partial version active.                                                         | IT-056   |
| US-019 EC-7  | Repeating publication cannot create two versions from one draft state.                                                              | UT-131   |
| US-019 EC-8  | A draft must validate before publication or activation.                                                                             | UT-132   |
| US-019 EC-9  | Published versions are immutable; retired versions cannot reactivate silently.                                                      | UT-133   |
| US-019 EC-10 | Many historical versions remain searchable without loading all definitions.                                                         | IT-057   |
| US-020       | Govern the technology radar                                                                                                         | E2E-020  |
| US-020 EC-1  | Unknown rings, past-invalid review dates, or blank override reasons are rejected.                                                   | UT-134   |
| US-020 EC-2  | `insufficient_data` maps only according to an explicit policy mapping.                                                              | UT-135   |
| US-020 EC-3  | Large radars filter and group Projects without hiding off-screen counts.                                                            | UT-136   |
| US-020 EC-4  | Viewers read the radar but cannot select, override, or annotate Projects.                                                           | UT-137   |
| US-020 EC-5  | Concurrent movement detects stale radar state.                                                                                      | IT-058   |
| US-020 EC-6  | A failed override preserves the prior ring and recommendation.                                                                      | IT-059   |
| US-020 EC-7  | Reapplying the same selection or override is idempotent.                                                                            | UT-138   |
| US-020 EC-8  | A Project needs a policy result before receiving a suggested ring.                                                                  | UT-139   |
| US-020 EC-9  | Archived Projects leave the active radar but retain historical placement.                                                           | UT-140   |
| US-020 EC-10 | Radar calculation remains bounded when many Projects receive updated recommendations.                                               | IT-060   |
| US-021       | Explore issue and discussion topics                                                                                                 | E2E-021  |
| US-021 EC-1  | Empty names, circular merges, or unsupported assignments are rejected.                                                              | UT-141   |
| US-021 EC-2  | No eligible content yields insufficient data, not zero topic prevalence.                                                            | UT-142   |
| US-021 EC-3  | Large topic and evidence sets paginate and cap displayed examples transparently.                                                    | UT-143   |
| US-021 EC-4  | Viewers cannot correct classifications.                                                                                             | UT-144   |
| US-021 EC-5  | Concurrent corrections detect stale topic versions.                                                                                 | IT-061   |
| US-021 EC-6  | Failed reprocessing leaves the prior version visible and stale.                                                                     | IT-062   |
| US-021 EC-7  | Repeating a correction does not duplicate feedback.                                                                                 | UT-145   |
| US-021 EC-8  | Trend calculations wait for a complete topic version.                                                                               | UT-146   |
| US-021 EC-9  | Retired topics remain in historical results but not new assignments.                                                                | UT-147   |
| US-021 EC-10 | Clustering is bounded and exposes queued or sampled status when needed.                                                             | IT-063   |
| US-022       | Understand a release                                                                                                                | E2E-022  |
| US-022 EC-1  | Malformed provider output cannot publish as a valid structured analysis.                                                            | UT-148   |
| US-022 EC-2  | A release without changelog shows limited evidence rather than invented changes.                                                    | UT-149   |
| US-022 EC-3  | Large changelogs and evidence sets are bounded with disclosed truncation.                                                           | UT-150   |
| US-022 EC-4  | Anonymous visitors cannot read protected release analysis.                                                                          | UT-151   |
| US-022 EC-5  | Concurrent reruns create distinct immutable versions.                                                                               | IT-064   |
| US-022 EC-6  | Failed analysis retains deterministic metadata and prior successful versions.                                                       | IT-065   |
| US-022 EC-7  | Replaying one run request does not overwrite or duplicate the same execution identity.                                              | UT-152   |
| US-022 EC-8  | Analysis does not claim evidence before eligible inputs are collected.                                                              | UT-153   |
| US-022 EC-9  | Withdrawn releases retain their source status and historical analysis.                                                              | UT-154   |
| US-022 EC-10 | Release lists and runs paginate across long histories.                                                                              | IT-066   |
| US-023       | Search project documentation                                                                                                        | E2E-023  |
| US-023 EC-1  | Unsupported schemes, unsafe addresses, and out-of-scope domains are rejected.                                                       | UT-155   |
| US-023 EC-2  | No indexed documentation yields an explicit no-evidence response.                                                                   | UT-156   |
| US-023 EC-3  | Crawl depth, bytes, pages, and request rate stop predictably with visible coverage.                                                 | UT-157   |
| US-023 EC-4  | Only Analysts can configure crawl scope; approved users may search.                                                                 | UT-158   |
| US-023 EC-5  | Duplicate crawl requests coalesce by source and snapshot target.                                                                    | IT-067   |
| US-023 EC-6  | Partial crawls expose their boundary and resume without duplicating snapshots.                                                      | IT-068   |
| US-023 EC-7  | Unchanged content does not create misleading duplicate versions.                                                                    | UT-159   |
| US-023 EC-8  | Search indexes only validated snapshots.                                                                                            | UT-160   |
| US-023 EC-9  | Removed URLs remain historical evidence but leave current search after refresh.                                                     | UT-161   |
| US-023 EC-10 | Search remains bounded and identifies sampling or truncation at large corpus sizes.                                                 | IT-069   |
| US-024       | Ask natural-language questions                                                                                                      | E2E-024  |
| US-024 EC-1  | Hostile or unsupported requests cannot escape the bounded analytical/action catalog.                                                | UT-162   |
| US-024 EC-2  | Blank questions or empty evidence return actionable guidance, not fabricated answers.                                               | UT-163   |
| US-024 EC-3  | Token, query, and evidence limits show truncation and refinement guidance.                                                          | UT-164   |
| US-024 EC-4  | The assistant cannot retrieve evidence the requesting user cannot access.                                                           | UT-165   |
| US-024 EC-5  | Evidence updates during a response do not mix data cutoffs silently.                                                                | IT-070   |
| US-024 EC-6  | A canceled query stops generation and preserves no partial action approval.                                                         | IT-071   |
| US-024 EC-7  | Repeating a question identifies its current cutoff rather than implying timeless identity.                                          | UT-166   |
| US-024 EC-8  | Clarification must resolve before analysis or action proposal continues.                                                            | UT-167   |
| US-024 EC-9  | Questions about deleted Projects return unavailable without leaked history.                                                         | UT-168   |
| US-024 EC-10 | Broad questions are narrowed or bounded rather than scanning the workspace without limit.                                           | IT-072   |
| US-025       | Approve a non-destructive assistant action                                                                                          | E2E-025  |
| US-025 EC-1  | Untyped or unsupported proposal fields cannot reach execution.                                                                      | UT-169   |
| US-025 EC-2  | A proposal without every required value asks for clarification.                                                                     | UT-170   |
| US-025 EC-3  | Quota-exceeding proposals show the limit and cannot be approved into a hidden queue.                                                | UT-171   |
| US-025 EC-4  | Approval rechecks the current user's role and resource access.                                                                      | UT-172   |
| US-025 EC-5  | Changed resource state invalidates the preview and requires a new proposal.                                                         | IT-073   |
| US-025 EC-6  | Lost connection or expired proposal executes nothing.                                                                               | IT-074   |
| US-025 EC-7  | Replaying one approval cannot repeat the mutation.                                                                                  | UT-173   |
| US-025 EC-8  | Approval before preview or after expiration is rejected.                                                                            | UT-174   |
| US-025 EC-9  | Actions targeting paused, archived, or deleted resources obey lifecycle rules.                                                      | UT-175   |
| US-025 EC-10 | A request containing many actions is split into atomic approvals or rejected as too broad.                                          | IT-075   |
| US-026       | Review AI analysis versions                                                                                                         | E2E-026  |
| US-026 EC-1  | Feedback without a target version or reason is rejected.                                                                            | UT-176   |
| US-026 EC-2  | No successful run shows unavailable analysis and eligible next actions.                                                             | UT-177   |
| US-026 EC-3  | Version and evidence histories paginate; reruns obey quotas.                                                                        | UT-178   |
| US-026 EC-4  | Viewers inspect but cannot rerun, flag, or select versions.                                                                         | UT-179   |
| US-026 EC-5  | Concurrent selection detects stale current-version state.                                                                           | IT-076   |
| US-026 EC-6  | Interrupted runs remain terminally labeled and never appear successful.                                                             | IT-077   |
| US-026 EC-7  | Duplicate feedback from one request is idempotent.                                                                                  | UT-180   |
| US-026 EC-8  | A failed or incomplete run cannot be selected as presented output.                                                                  | UT-181   |
| US-026 EC-9  | Stale selected runs remain visible with a stale warning until replaced.                                                             | UT-182   |
| US-026 EC-10 | Retained versions remain discoverable without loading every output at once.                                                         | IT-078   |
| US-027       | Configure and resolve shared alerts                                                                                                 | E2E-027  |
| US-027 EC-1  | Rules with unknown signals, invalid thresholds, or negative cooldowns are rejected.                                                 | UT-183   |
| US-027 EC-2  | Missing required evidence cannot trigger an alert.                                                                                  | UT-184   |
| US-027 EC-3  | Alert lists paginate and rule/occurrence volume respects workspace quotas.                                                          | UT-185   |
| US-027 EC-4  | Viewers manage only personal read state; Analysts/Admins manage shared resolution.                                                  | UT-186   |
| US-027 EC-5  | Concurrent resolution uses one final state and identifies stale actions.                                                            | IT-079   |
| US-027 EC-6  | Evaluation failure does not close or duplicate an existing occurrence.                                                              | IT-080   |
| US-027 EC-7  | Replayed signals preserve one deduplicated occurrence.                                                                              | UT-187   |
| US-027 EC-8  | Resolution cannot precede occurrence creation; reopening follows explicit rules.                                                    | UT-188   |
| US-027 EC-9  | Archived Projects stop new alert evaluation while history remains readable.                                                         | UT-189   |
| US-027 EC-10 | High event volume is bounded without losing severity and suppression counts.                                                        | IT-081   |
| US-028       | Export results and evidence                                                                                                         | E2E-028  |
| US-028 EC-1  | Unsupported formats or malformed scopes are rejected before generation.                                                             | UT-190   |
| US-028 EC-2  | Empty result sets produce a valid empty export with scope metadata.                                                                 | UT-191   |
| US-028 EC-3  | Oversized exports are rejected or generated asynchronously with visible limits and status.                                          | UT-192   |
| US-028 EC-4  | Exports contain only data visible to the requesting approved member.                                                                | UT-193   |
| US-028 EC-5  | Export uses one explicit data cutoff despite ongoing updates.                                                                       | IT-082   |
| US-028 EC-6  | Interrupted generation can be retried without duplicate completed artifacts.                                                        | IT-083   |
| US-028 EC-7  | Identical requests identify equivalent scope and cutoff without changing data.                                                      | UT-194   |
| US-028 EC-8  | A download is unavailable until generation completes successfully.                                                                  | UT-195   |
| US-028 EC-9  | A completed download expires after 24 hours, and Project deletion removes owned generated exports earlier.                          | UT-196   |
| US-028 EC-10 | Large exports do not exhaust interactive product capacity.                                                                          | IT-084   |
| US-029       | Investigate the audit log                                                                                                           | E2E-029  |
| US-029 EC-1  | Invalid ranges or filters are rejected without arbitrary query execution.                                                           | UT-197   |
| US-029 EC-2  | No matches produce a clear empty state.                                                                                             | UT-198   |
| US-029 EC-3  | Reads and exports paginate or bound results without dropping event counts silently.                                                 | UT-199   |
| US-029 EC-4  | Only Admins can inspect or export audit history.                                                                                    | UT-200   |
| US-029 EC-5  | Simultaneous actions retain distinct event identities and deterministic ordering ties.                                              | IT-085   |
| US-029 EC-6  | A failed business action records failure without claiming a state change.                                                           | IT-086   |
| US-029 EC-7  | Idempotent retries remain attributable without fabricating duplicate success.                                                       | UT-201   |
| US-029 EC-8  | Audit ordering uses event time and stable tie-break identity.                                                                       | UT-202   |
| US-029 EC-9  | Audit events remain immutable after subject deletion or role changes.                                                               | UT-203   |
| US-029 EC-10 | Long retention remains searchable through bounded pages and time filters.                                                           | IT-087   |
| US-030       | Operate model providers and degradation                                                                                             | E2E-030  |
| US-030 EC-1  | Unsupported model capabilities or malformed configuration fail validation safely.                                                   | UT-204   |
| US-030 EC-2  | No provider is a valid degraded state, not an application startup failure.                                                          | UT-205   |
| US-030 EC-3  | Model quotas delay or reject AI work with visible status and no retry storm.                                                        | UT-206   |
| US-030 EC-4  | Users cannot submit or retrieve provider secrets; only Admins see redacted status.                                                  | UT-207   |
| US-030 EC-5  | Configuration changes do not alter provider identity within an active run.                                                          | IT-088   |
| US-030 EC-6  | Interrupted runs become failed or canceled versions and never block deterministic work.                                             | IT-089   |
| US-030 EC-7  | Retried requests create attributable runs without overwriting successful history.                                                   | UT-208   |
| US-030 EC-8  | New configuration validates before becoming active for later runs.                                                                  | UT-209   |
| US-030 EC-9  | Disabled providers make dependent features unavailable without deleting prior outputs.                                              | UT-210   |
| US-030 EC-10 | Global model concurrency and usage limits protect interactive deterministic traffic.                                                | IT-090   |
| US-031       | Use the bilingual mobile-first product                                                                                              | E2E-031  |
| US-031 EC-1  | Unsupported locale input falls back to English and remains changeable.                                                              | UT-211   |
| US-031 EC-2  | Missing translation never produces blank controls or hidden errors.                                                                 | UT-212   |
| US-031 EC-3  | Dense tables and charts use responsive summaries and bounded detail rather than data loss.                                          | UT-213   |
| US-031 EC-4  | Responsive layouts never reveal controls or data hidden from the role.                                                              | UT-214   |
| US-031 EC-5  | Language changes during a save do not repeat or lose the action.                                                                    | IT-091   |
| US-031 EC-6  | Navigation or connection loss preserves recoverable form state where safe.                                                          | IT-092   |
| US-031 EC-7  | Repeated locale switching does not alter stored evidence or calculations.                                                           | UT-215   |
| US-031 EC-8  | Deep links apply membership, language, and timezone before protected content renders.                                               | UT-216   |
| US-031 EC-9  | Disabled actions remain labeled with their lifecycle reason at narrow widths.                                                       | UT-217   |
| US-031 EC-10 | Large localized strings and 200% zoom remain usable without clipping essential actions.                                             | IT-093   |
| US-032       | Use the API through a service identity                                                                                              | E2E-032  |
| US-032 EC-1  | A malformed, expired, wrong-issuer, wrong-audience, or unverifiable token is rejected without creating a local account.             | UT-218   |
| US-032 EC-2  | A valid Keycloak subject without a local binding receives no implicit access.                                                       | UT-219   |
| US-032 EC-3  | Service-account requests use the applicable account, workspace, source, and Job quotas.                                             | UT-220   |
| US-032 EC-4  | A service account cannot receive Admin, approve members, change policies, manage credentials, or perform Project lifecycle actions. | UT-221   |
| US-032 EC-5  | Concurrent scope changes and requests authorize against one committed local account version.                                        | IT-094   |
| US-032 EC-6  | An interrupted idempotent action can retry without duplicating the resource or Job.                                                 | IT-095   |
| US-032 EC-7  | Repeated bearer requests remain individually attributable while idempotency preserves one business outcome.                         | UT-222   |
| US-032 EC-8  | Local suspension takes effect before any request authorized against the later account version.                                      | UT-223   |
| US-032 EC-9  | Deleting or suspending the local binding prevents access even while the external Keycloak token remains valid.                      | UT-224   |
| US-032 EC-10 | Many service-account calls remain isolated by identity and scope without exhausting interactive member capacity.                    | IT-096   |

### Components and interfaces

| Part II component / interface                                 | Test IDs                               |
| ------------------------------------------------------------- | -------------------------------------- |
| auth.Authorizer and local identity/session services           | UT-247, UT-248, UT-249, IT-100–IT-102  |
| job.Dispatcher, leases, cancellation and state machine        | UT-256–UT-258, IT-103–IT-111           |
| collector.RepositorySource and provider adapters              | UT-255, UT-266, IT-114–IT-120          |
| metric.Engine, health, comparison, trend and forecast         | UT-228–UT-242, IT-121–IT-122           |
| evidence.Store, S3 ownership and purge                        | UT-259, UT-268, IT-112–IT-113          |
| analysis.AgentRunner and model capability ports               | UT-261–UT-265, IT-125–IT-128           |
| PostgreSQL/sqlc/migrations/Snowflake                          | UT-225–UT-227, IT-097–IT-099           |
| Outbox and NATS JetStream                                     | IT-103–IT-107                          |
| Valkey cache/rate limits/SSE fan-out                          | IT-110–IT-111                          |
| Search, documents and topic graph                             | UT-242–UT-244, IT-123–IT-124           |
| Policies, radar and alerts                                    | UT-245–UT-246, UT-267, IT-130          |
| Exports, audit and deletion                                   | UT-260, UT-268, IT-129–IT-133          |
| HTTP generated transport, cursors, idempotency and If-Match   | UT-251–UT-254, IT-136, IT-139–IT-141   |
| Configuration, health, telemetry and graceful shutdown        | UT-269, IT-134–IT-137                  |
| React shell, feature adapters, localization and accessibility | UT-270, IT-145–IT-146, E2E-033–E2E-055 |

### API and message contracts

| Contract                                                                                 | Success / resilience | Failure / rejection |
| ---------------------------------------------------------------------------------------- | -------------------- | ------------------- |
| `GET /api/v1/catalog/projects?q=&cursor=&limit=`                                         | IT-147               | IT-148              |
| `GET /api/v1/catalog/projects/{project_id}`                                              | IT-149               | IT-150              |
| `GET /api/v1/session`                                                                    | IT-151               | IT-152              |
| `POST /api/v1/session/logout`                                                            | IT-153               | IT-154              |
| `PATCH /api/v1/me/preferences`                                                           | IT-155               | IT-156              |
| `POST /api/v1/me/deletion`                                                               | IT-157               | IT-158              |
| `GET /api/v1/admin/members?state=&role=&q=&cursor=&limit=`                               | IT-159               | IT-160              |
| `POST /api/v1/admin/members/{member_id}/approval`                                        | IT-161               | IT-162              |
| `PATCH /api/v1/admin/members/{member_id}`                                                | IT-163               | IT-164              |
| `GET /api/v1/admin/service-accounts?state=&q=&cursor=&limit=`                            | IT-165               | IT-166              |
| `POST /api/v1/admin/service-accounts`                                                    | IT-167               | IT-168              |
| `PATCH /api/v1/admin/service-accounts/{service_account_id}`                              | IT-169               | IT-170              |
| `GET /api/v1/admin/audit?actor=&action=&resource=&from=&to=&cursor=&limit=`              | IT-171               | IT-172              |
| `GET /api/v1/admin/operations`                                                           | IT-173               | IT-174              |
| `GET /api/v1/portfolio?window=90d&cutoff=`                                               | IT-175               | IT-176              |
| `GET /api/v1/projects?state=active&q=&cursor=&limit=`                                    | IT-177               | IT-178              |
| `POST /api/v1/projects`                                                                  | IT-179               | IT-180              |
| `GET /api/v1/projects/{project_id}`                                                      | IT-181               | IT-182              |
| `PATCH /api/v1/projects/{project_id}`                                                    | IT-183               | IT-184              |
| `POST /api/v1/projects/{project_id}/transition`                                          | IT-185               | IT-186              |
| `POST /api/v1/projects/{project_id}/deletion`                                            | IT-187               | IT-188              |
| `GET /api/v1/projects/{project_id}/repositories?cursor=&limit=`                          | IT-189               | IT-190              |
| `POST /api/v1/projects/{project_id}/repositories`                                        | IT-191               | IT-192              |
| `PATCH /api/v1/projects/{project_id}/repositories/{repository_id}`                       | IT-193               | IT-194              |
| `DELETE /api/v1/projects/{project_id}/repositories/{repository_id}`                      | IT-195               | IT-196              |
| `GET /api/v1/projects/{project_id}/sources?kind=&state=&cursor=&limit=`                  | IT-197               | IT-198              |
| `POST /api/v1/projects/{project_id}/sources`                                             | IT-199               | IT-200              |
| `PATCH /api/v1/projects/{project_id}/sources/{source_id}`                                | IT-201               | IT-202              |
| `DELETE /api/v1/projects/{project_id}/sources/{source_id}`                               | IT-203               | IT-204              |
| `GET /api/v1/projects/{project_id}/associations?status=&cursor=&limit=`                  | IT-205               | IT-206              |
| `POST /api/v1/projects/{project_id}/associations/{association_id}/correction`            | IT-207               | IT-208              |
| `POST /api/v1/projects/{project_id}/syncs`                                               | IT-209               | IT-210              |
| `POST /api/v1/projects/{project_id}/history-requests`                                    | IT-211               | IT-212              |
| `GET /api/v1/projects/{project_id}/jobs?kind=&state=&cursor=&limit=`                     | IT-213               | IT-214              |
| `GET /api/v1/jobs/{job_id}`                                                              | IT-215               | IT-216              |
| `GET /api/v1/jobs/{job_id}/events`                                                       | IT-217               | IT-218              |
| `POST /api/v1/jobs/{job_id}/cancellation`                                                | IT-219               | IT-220              |
| `GET /api/v1/projects/{project_id}/metrics?dimension=&window=90d&cutoff=&cursor=&limit=` | IT-221               | IT-222              |
| `GET /api/v1/projects/{project_id}/metrics/{metric_name}?window=90d&cutoff=`             | IT-223               | IT-224              |
| `GET /api/v1/projects/{project_id}/health?window=90d&cutoff=`                            | IT-225               | IT-226              |
| `GET /api/v1/projects/{project_id}/contributors?window=90d&cursor=&limit=`               | IT-227               | IT-228              |
| `GET /api/v1/projects/{project_id}/adoption?window=90d&cursor=&limit=`                   | IT-229               | IT-230              |
| `GET /api/v1/projects/{project_id}/security?window=365d&cursor=&limit=`                  | IT-231               | IT-232              |
| `POST /api/v1/comparisons`                                                               | IT-233               | IT-234              |
| `GET /api/v1/comparisons/{comparison_id}`                                                | IT-235               | IT-236              |
| `GET /api/v1/projects/{project_id}/trends?kind=observed&window=365d&cursor=&limit=`      | IT-237               | IT-238              |
| `GET /api/v1/projects/{project_id}/recommendation?policy=default&window=90d&cutoff=`     | IT-239               | IT-240              |
| `GET /api/v1/policies?state=&cursor=&limit=`                                             | IT-241               | IT-242              |
| `GET /api/v1/policies/{policy_id}/versions/{version}`                                    | IT-243               | IT-244              |
| `POST /api/v1/policies`                                                                  | IT-245               | IT-246              |
| `POST /api/v1/policies/{policy_id}/versions`                                             | IT-247               | IT-248              |
| `POST /api/v1/policies/{policy_id}/versions/{version}/activation`                        | IT-249               | IT-250              |
| `GET /api/v1/radar?policy=default&window=90d`                                            | IT-251               | IT-252              |
| `POST /api/v1/radar/{project_id}/override`                                               | IT-253               | IT-254              |
| `DELETE /api/v1/radar/{project_id}/override`                                             | IT-255               | IT-256              |
| `GET /api/v1/projects/{project_id}/topics?window=90d&cursor=&limit=`                     | IT-257               | IT-258              |
| `POST /api/v1/projects/{project_id}/topics/{topic_id}/corrections`                       | IT-259               | IT-260              |
| `GET /api/v1/projects/{project_id}/releases?cursor=&limit=`                              | IT-261               | IT-262              |
| `GET /api/v1/projects/{project_id}/releases/{release_id}`                                | IT-263               | IT-264              |
| `POST /api/v1/projects/{project_id}/crawls`                                              | IT-265               | IT-266              |
| `POST /api/v1/projects/{project_id}/knowledge/search`                                    | IT-267               | IT-268              |
| `POST /api/v1/projects/{project_id}/queries`                                             | IT-269               | IT-270              |
| `GET /api/v1/analysis-runs/{run_id}`                                                     | IT-271               | IT-272              |
| `POST /api/v1/analysis-runs/{run_id}/reruns`                                             | IT-273               | IT-274              |
| `POST /api/v1/analysis-runs/{run_id}/feedback`                                           | IT-275               | IT-276              |
| `POST /api/v1/analysis-series/{series_id}/selection`                                     | IT-277               | IT-278              |
| `GET /api/v1/alerts?state=&project=&cursor=&limit=`                                      | IT-279               | IT-280              |
| `POST /api/v1/alert-rules`                                                               | IT-281               | IT-282              |
| `PATCH /api/v1/alert-rules/{rule_id}`                                                    | IT-283               | IT-284              |
| `POST /api/v1/alerts/{alert_id}/read`                                                    | IT-285               | IT-286              |
| `POST /api/v1/alerts/{alert_id}/transition`                                              | IT-287               | IT-288              |
| `POST /api/v1/exports`                                                                   | IT-289               | IT-290              |
| `GET /api/v1/exports/{export_id}`                                                        | IT-291               | IT-292              |
| `GET /api/v1/exports/{export_id}/download`                                               | IT-293               | IT-294              |
| `POST /api/v1/assistant/proposals`                                                       | IT-295               | IT-296              |
| `POST /api/v1/assistant/proposals/{proposal_id}/confirmation`                            | IT-297               | IT-298              |
| PostgreSQL Job + outbox commit                                                           | IT-103               | IT-103              |
| Outbox → NATS publish                                                                    | IT-104               | IT-105              |
| NATS consumer delivery/ack                                                               | IT-106               | IT-107              |
| Job lease/heartbeat/checkpoint                                                           | IT-108               | IT-109              |
| SSE job.updated / Last-Event-ID                                                          | IT-110               | IT-111              |
| S3 evidence reference/purge                                                              | IT-112               | IT-113              |
| ADK run/HITL continuation                                                                | IT-127               | IT-128              |
| OpenTelemetry propagation                                                                | IT-134               | IT-134              |

### UI surfaces

| `_uiux.md` surface                                  | Primary E2E test |
| --------------------------------------------------- | ---------------- |
| S1 Application shell and localized routing          | E2E-033          |
| S2 Public catalog and protected teaser              | E2E-034          |
| S3 Sign-in, pending access and account              | E2E-035          |
| S4 Portfolio overview                               | E2E-036          |
| S5 Project catalog, registration and identity       | E2E-037          |
| S6 Project lifecycle                                | E2E-038          |
| S7 Repositories, sources and associations           | E2E-039          |
| S8 Jobs, synchronization and history                | E2E-040          |
| S9 Metrics and health                               | E2E-041          |
| S10 Contributor intelligence                        | E2E-042          |
| S11 Adoption and security                           | E2E-043          |
| S12 Comparison workspace                            | E2E-044          |
| S13 Trends and early warnings                       | E2E-045          |
| S14 Recommendation and policy governance            | E2E-046          |
| S15 Technology radar                                | E2E-047          |
| S16 Issue and discussion topics                     | E2E-048          |
| S17 Release intelligence                            | E2E-049          |
| S18 Documentation knowledge                         | E2E-050          |
| S19 Natural-language analysis and HITL assistant    | E2E-051          |
| S20 AI run governance                               | E2E-052          |
| S21 Alerts                                          | E2E-053          |
| S22 Exports                                         | E2E-054          |
| S23 Members, service accounts, audit and operations | E2E-055          |

## Coverage Demands

- All 32 user stories, all acceptance-criteria sets and all 320 approved edge cases must pass; the
  matrix contains one row for each.
- Every interface and architectural boundary in `_spec.md` must exercise success, typed failure,
  cancellation/timeout where applicable, and unavailable-dependency behavior.
- Every frozen HTTP operation must pass its paired success and failure test; mutation failures must
  prove no partial side effect. Generated request/response examples must validate against OpenAPI.
- Every durable message contract must prove at-least-once replay, schema version, correlation,
  idempotency, checkpoint and terminal-state behavior.
- Every `_uiux.md` surface must pass wide/narrow, en/pt-BR, keyboard, focus, accessible-name/status
  and loading/empty/error/stale-state checks; charts require equivalent tables/text.
- Database behavior must use real PostgreSQL/pgvector; delivery/cache/object semantics must use real
  NATS/Valkey/S3-compatible services. Mocks cannot substitute for the boundary whose semantics are
  under test.
- Race detection, bounded goroutine shutdown, generated drift, migration convergence, security input
  fuzzing, evidence completeness and secret-redaction checks are release gates.
- A temporarily unavailable external model/provider may skip only explicitly opt-in live
  diagnostics. Controlled contract fixtures and degradation behavior remain mandatory.

## Test Data and Isolation

Seed at least five synthetic projects with multiple repositories, overlapping contributor
identities, reopened issues, draft/stable/prerelease releases, incomplete provider coverage,
registry/advisory/document sources, corrected topics and mixed freshness. Fixtures include valid and
hostile URLs, rate-limit sequences, duplicate/out-of-order provider pages, multilingual
documentation, unsupported model claims and deletion-spanning objects. Synthetic identities cover
Applicant, Viewer, Analyst, Admin, suspended/rejected members and scoped service accounts. No
fixture contains production credentials or copied private-source data.

Each test owns a database schema/database, JetStream stream/consumer namespace, Valkey prefix and S3
bucket/prefix. Tests use fixed clocks, deterministic Snowflake node leases, explicit cutoffs and
seeded immutable definition/prompt versions. Cleanup may delete only resources created by the test
namespace.

## Exit Criteria

A task may mark one of these cases complete only after the named automated test exists and passes in
the repository's documented verification command. The specification is complete only when every
matrix row has passing evidence, generated artifacts are clean, migrations converge from empty
state, race detection passes, and the browser suite verifies the production build. Flaky retries,
disabled assertions, reduced permissions, fake database/broker/object semantics, or unreviewed
snapshot updates do not count as evidence.
