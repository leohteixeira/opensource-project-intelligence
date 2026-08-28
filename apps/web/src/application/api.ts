import {
  getApiV1AdminAudit,
  getApiV1AdminMembers,
  getApiV1AdminOperations,
  getApiV1AdminServiceAccounts,
  getApiV1Alerts,
  getApiV1CatalogProjects,
  getApiV1CatalogProjectsProjectId,
  getApiV1JobsJobId,
  getApiV1AnalysisRunsRunId,
  getApiV1ComparisonsComparisonId,
  getApiV1Portfolio,
  getApiV1Policies,
  getApiV1Projects,
  getApiV1ProjectsProjectId,
  getApiV1ProjectsProjectIdAssociations,
  getApiV1ProjectsProjectIdAdoption,
  getApiV1ProjectsProjectIdJobs,
  getApiV1ProjectsProjectIdContributors,
  getApiV1ProjectsProjectIdHealth,
  getApiV1ProjectsProjectIdMetrics,
  getApiV1ProjectsProjectIdMetricsMetricName,
  getApiV1ProjectsProjectIdRecommendation,
  getApiV1ProjectsProjectIdReleases,
  getApiV1ProjectsProjectIdReleasesReleaseId,
  getApiV1ProjectsProjectIdRepositories,
  getApiV1ProjectsProjectIdSources,
  getApiV1ProjectsProjectIdSecurity,
  getApiV1ProjectsProjectIdTopics,
  getApiV1ProjectsProjectIdTrends,
  getApiV1Radar,
  getApiV1Session,
  patchApiV1AdminMembersMemberId,
  patchApiV1AdminServiceAccountsServiceAccountId,
  patchApiV1MePreferences,
  patchApiV1ProjectsProjectId,
  patchApiV1ProjectsProjectIdRepositoriesRepositoryId,
  patchApiV1ProjectsProjectIdSourcesSourceId,
  postApiV1AdminMembersMemberIdApproval,
  postApiV1AdminServiceAccounts,
  postApiV1Comparisons,
  postApiV1AnalysisRunsRunIdFeedback,
  postApiV1AnalysisRunsRunIdReruns,
  postApiV1AnalysisSeriesSeriesIdSelection,
  postApiV1AlertsAlertIdRead,
  postApiV1AlertsAlertIdTransition,
  postApiV1AssistantProposals,
  postApiV1AssistantProposalsProposalIdConfirmation,
  postApiV1Exports,
  getApiV1ExportsExportId,
  postApiV1MeDeletion,
  postApiV1Projects,
  postApiV1ProjectsProjectIdCrawls,
  postApiV1ProjectsProjectIdKnowledgeSearch,
  postApiV1ProjectsProjectIdQueries,
  postApiV1ProjectsProjectIdTopicsTopicIdCorrections,
  postApiV1Policies,
  postApiV1ProjectsProjectIdAssociationsAssociationIdCorrection,
  postApiV1ProjectsProjectIdDeletion,
  postApiV1ProjectsProjectIdHistoryRequests,
  postApiV1ProjectsProjectIdRepositories,
  postApiV1ProjectsProjectIdSources,
  postApiV1ProjectsProjectIdSyncs,
  postApiV1ProjectsProjectIdTransition,
  postApiV1SessionLogout,
  postApiV1RadarProjectIdOverride,
} from '../api/generated';

export type Document = Record<string, unknown>;

export interface SessionDocument extends Document {
  authenticated?: boolean;
  state?: string;
  role?: string;
  actor_kind?: string;
  csrf_token?: string;
  member?: MemberDocument;
}

export interface MemberDocument extends Document {
  id?: string;
  display_name?: string;
  email?: string;
  role?: string;
  status?: string;
  locale?: string;
  timezone?: string;
  version?: number;
}

export interface CatalogProject extends Document {
  id: string;
  name: string;
  slug: string;
  description: string;
  source_links: string[];
}

export interface Page<T> extends Document {
  items: T[];
  has_more: boolean;
  next_cursor?: string;
}

export interface ProjectDocument extends Document {
  id: string;
  name: string;
  slug: string;
  description: string;
  state: string;
  version: number;
  repositories?: Document[];
  sources?: Document[];
}

export interface JobDocument extends Document {
  id: string;
  project_id?: string;
  kind: string;
  state: string;
  version: number;
  progress?: { completed?: number; total?: number; total_status?: string; unit?: string };
  checkpoint?: Document;
  failure?: string;
}

export interface AssistantProposalDocument extends Document {
  id: string;
  status: string;
  action: string;
  inputs?: Document;
  resources?: string[];
  effect?: string;
  quota?: { name?: string; cost?: number; remaining?: number };
  expires_at?: string;
  confirmation_token?: string;
  result?: { repository_id?: string; audit_event_id?: string };
}

export interface ExportDocument extends Document {
  id: string;
  job_id: string;
  state: string;
  row_count?: number;
  media_type?: string;
  sha256?: string;
  size_bytes?: number;
  download_url?: string;
  failure?: string;
  expires_at?: string;
}

export type IntelligenceStatus =
  | 'available'
  | 'unknown'
  | 'not_applicable'
  | 'insufficient_data'
  | 'incomparable'
  | 'stale'
  | 'unavailable';

export interface IntelligenceWindow extends Document {
  from: string;
  to: string;
  cutoff: string;
}

export interface MetricDocument extends Document {
  id?: string;
  project_id: string;
  definition: {
    name: string;
    version: string;
    unit: string;
    formula: string;
    eligibility: string;
    missing_data_rule: string;
  };
  window: IntelligenceWindow;
  status: IntelligenceStatus;
  value?: number;
  coverage: { eligible: number; observed: number; ratio: number; note?: string };
  factors: Array<{ name: string; value?: number; unit: string; evidence_id?: string }>;
  repository_ids: string[];
  stale_reason?: string;
}

export interface HealthDocument extends Document {
  project_id: string;
  window: IntelligenceWindow;
  version: string;
  overall?: number;
  overall_status: IntelligenceStatus;
  dimensions: Array<{
    name: string;
    status: IntelligenceStatus;
    score?: number;
    weight: number;
    version: string;
  }>;
}

export interface ContributorsDocument extends Document {
  project_id: string;
  window: IntelligenceWindow;
  summary: {
    status: IntelligenceStatus;
    active: number;
    top_one_share?: number;
    top_three_share?: number;
    resolution_coverage: number;
  };
  items: Array<{ key: string; commits: number; identity_status: string }>;
  has_more: boolean;
}

export interface ComparisonDocument extends Document {
  id: string;
  project_ids: string[];
  window: IntelligenceWindow;
  created_at: string;
  rows: Array<{
    metric: string;
    unit: string;
    cells: Array<{
      project_id: string;
      status: IntelligenceStatus;
      value?: number;
      version: string;
      evidence: Array<{ name: string; value?: number; unit: string }>;
    }>;
  }>;
}

type Result = { data?: unknown; error?: unknown; response?: Response };

function unwrap<T>(result: Result): T {
  if (result.error !== undefined || result.response?.ok !== true) {
    const problem = result.error as { detail?: string; code?: string } | undefined;
    throw Object.assign(new Error(problem?.detail ?? 'The request could not be completed.'), {
      code: problem?.code,
      status: result.response?.status ?? 0,
    });
  }
  return result.data as T;
}

export async function fetchSession(): Promise<SessionDocument> {
  return unwrap<SessionDocument>(await getApiV1Session());
}

export async function proposeAssistantAction(
  session: SessionDocument,
  message: string,
): Promise<AssistantProposalDocument> {
  return unwrap<AssistantProposalDocument>(
    await postApiV1AssistantProposals({
      body: { message },
      headers: idempotencyHeaders(session),
    }),
  );
}

export async function confirmAssistantAction(
  session: SessionDocument,
  proposalId: string,
  confirmationToken: string,
): Promise<AssistantProposalDocument> {
  return unwrap<AssistantProposalDocument>(
    await postApiV1AssistantProposalsProposalIdConfirmation({
      path: { proposal_id: proposalId },
      body: { confirmation_token: confirmationToken },
      headers: idempotencyHeaders(session),
    }),
  );
}

export async function requestExport(
  session: SessionDocument,
  value: Document,
): Promise<ExportDocument> {
  return unwrap<ExportDocument>(
    await postApiV1Exports({ body: value, headers: idempotencyHeaders(session) }),
  );
}

export async function fetchExport(exportId: string): Promise<ExportDocument> {
  return unwrap<ExportDocument>(await getApiV1ExportsExportId({ path: { export_id: exportId } }));
}

export async function fetchCatalog(q: string, cursor?: string): Promise<Page<CatalogProject>> {
  return unwrap<Page<CatalogProject>>(
    await getApiV1CatalogProjects({ query: { q: q || undefined, cursor, limit: 24 } }),
  );
}

export async function fetchCatalogProject(projectId: string): Promise<CatalogProject> {
  return unwrap<CatalogProject>(
    await getApiV1CatalogProjectsProjectId({ path: { project_id: projectId } }),
  );
}

export async function savePreferences(
  session: SessionDocument,
  locale: string,
  timezone: string,
): Promise<MemberDocument> {
  return unwrap<MemberDocument>(
    await patchApiV1MePreferences({
      body: { locale, timezone },
      headers: mutationHeaders(session, false, session.member?.version),
    }),
  );
}

export async function deleteAccount(
  session: SessionDocument,
  confirmation: string,
): Promise<Document> {
  return unwrap<Document>(
    await postApiV1MeDeletion({
      body: { confirmation },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function logout(session: SessionDocument): Promise<void> {
  unwrap<unknown>(await postApiV1SessionLogout({ headers: mutationHeaders(session, false) }));
}

export async function fetchMembers(): Promise<Page<Document>> {
  return unwrap<Page<Document>>(await getApiV1AdminMembers({ query: { limit: 50 } }));
}

export async function approveMember(
  session: SessionDocument,
  memberId: string,
  decision: 'approve' | 'reject',
): Promise<Document> {
  return unwrap<Document>(
    await postApiV1AdminMembersMemberIdApproval({
      path: { member_id: memberId },
      body: decision === 'approve' ? { decision, role: 'viewer' } : { decision },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function fetchServiceAccounts(): Promise<Page<Document>> {
  return unwrap<Page<Document>>(await getApiV1AdminServiceAccounts({ query: { limit: 50 } }));
}

export async function createServiceAccount(
  session: SessionDocument,
  account: {
    name: string;
    external_subject: string;
    role: 'viewer' | 'analyst';
    scopes: string[];
  },
): Promise<Document> {
  return unwrap<Document>(
    await postApiV1AdminServiceAccounts({
      body: account,
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function updateMember(
  session: SessionDocument,
  memberId: string,
  version: number,
  role: 'viewer' | 'analyst' | 'admin',
  state: 'active' | 'suspended',
): Promise<Document> {
  return unwrap<Document>(
    await patchApiV1AdminMembersMemberId({
      path: { member_id: memberId },
      body: { role, state },
      headers: mutationHeaders(session, false, version),
    }),
  );
}

export async function updateServiceAccount(
  session: SessionDocument,
  accountId: string,
  version: number,
  role: 'viewer' | 'analyst',
  state: 'active' | 'suspended',
  scopes: string[],
): Promise<Document> {
  return unwrap<Document>(
    await patchApiV1AdminServiceAccountsServiceAccountId({
      path: { service_account_id: accountId },
      body: { role, state, scopes },
      headers: mutationHeaders(session, false, version),
    }),
  );
}

export interface AuditFilters {
  actor?: string;
  action?: string;
  resource?: string;
  outcome?: string;
  from?: string;
  to?: string;
}

export async function fetchAudit(filters: AuditFilters = {}): Promise<Page<Document>> {
  return unwrap<Page<Document>>(
    await getApiV1AdminAudit({
      query: {
        actor: filters.actor || undefined,
        action: filters.action || undefined,
        resource: filters.resource || undefined,
        outcome: filters.outcome || undefined,
        from: filters.from || undefined,
        to: filters.to || undefined,
        limit: 50,
      },
    }),
  );
}

export async function fetchOperations(): Promise<Document> {
  return unwrap<Document>(await getApiV1AdminOperations());
}

export async function fetchPortfolio(): Promise<Document> {
  return unwrap<Document>(await getApiV1Portfolio());
}

export async function fetchProjects(
  state = 'active',
  q = '',
  cursor?: string,
): Promise<Page<ProjectDocument>> {
  return unwrap<Page<ProjectDocument>>(
    await getApiV1Projects({
      query: { state: state || undefined, q: q || undefined, cursor, limit: 24 },
    }),
  );
}

export async function registerProject(
  session: SessionDocument,
  repositoryUrl: string,
  historyDays = 180,
): Promise<{ project: ProjectDocument; job: JobDocument; replay?: boolean }> {
  return unwrap(
    await postApiV1Projects({
      body: { repository_url: repositoryUrl, history_days: historyDays },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function fetchWorkspaceProject(projectId: string): Promise<ProjectDocument> {
  return unwrap<ProjectDocument>(
    await getApiV1ProjectsProjectId({ path: { project_id: projectId } }),
  );
}

export async function updateProject(
  session: SessionDocument,
  value: ProjectDocument,
  name: string,
  description: string,
): Promise<ProjectDocument> {
  return unwrap<ProjectDocument>(
    await patchApiV1ProjectsProjectId({
      path: { project_id: value.id },
      body: { name, description },
      headers: mutationHeaders(session, false, value.version),
    }),
  );
}

export async function transitionProject(
  session: SessionDocument,
  value: ProjectDocument,
  to: 'active' | 'paused' | 'archived',
  reason: string,
): Promise<Document> {
  return unwrap<Document>(
    await postApiV1ProjectsProjectIdTransition({
      path: { project_id: value.id },
      body: { to, reason },
      headers: mutationHeaders(session, true, value.version),
    }),
  );
}

export async function deleteProject(
  session: SessionDocument,
  value: ProjectDocument,
  reason: string,
): Promise<Document> {
  return unwrap<Document>(
    await postApiV1ProjectsProjectIdDeletion({
      path: { project_id: value.id },
      body: { confirmation: `DELETE ${value.slug}`, reason },
      headers: mutationHeaders(session, true, value.version),
    }),
  );
}

export async function fetchProjectResources(projectId: string) {
  const path = { project_id: projectId };
  const [repositories, sources, associations, jobs] = await Promise.all([
    getApiV1ProjectsProjectIdRepositories({ path, query: { limit: 50 } }),
    getApiV1ProjectsProjectIdSources({ path, query: { limit: 50 } }),
    getApiV1ProjectsProjectIdAssociations({ path, query: { limit: 50 } }),
    getApiV1ProjectsProjectIdJobs({ path, query: { limit: 50 } }),
  ]);
  return {
    repositories: unwrap<Page<Document>>(repositories),
    sources: unwrap<Page<Document>>(sources),
    associations: unwrap<Page<Document>>(associations),
    jobs: unwrap<Page<JobDocument>>(jobs),
  };
}

export async function addRepository(
  session: SessionDocument,
  projectId: string,
  url: string,
  role: string,
) {
  return unwrap<Document>(
    await postApiV1ProjectsProjectIdRepositories({
      path: { project_id: projectId },
      body: { url, role },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function changeRepositoryRole(
  session: SessionDocument,
  projectId: string,
  repositoryId: string,
  version: number,
  role: string,
) {
  return unwrap<Document>(
    await patchApiV1ProjectsProjectIdRepositoriesRepositoryId({
      path: { project_id: projectId, repository_id: repositoryId },
      body: { role },
      headers: mutationHeaders(session, false, version),
    }),
  );
}

export async function addSource(
  session: SessionDocument,
  projectId: string,
  kind: string,
  url: string,
) {
  return unwrap<Document>(
    await postApiV1ProjectsProjectIdSources({
      path: { project_id: projectId },
      body: { kind, url },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function updateSourceState(
  session: SessionDocument,
  projectId: string,
  sourceId: string,
  version: number,
  state: string,
) {
  return unwrap<Document>(
    await patchApiV1ProjectsProjectIdSourcesSourceId({
      path: { project_id: projectId, source_id: sourceId },
      body: { state },
      headers: mutationHeaders(session, false, version),
    }),
  );
}

export async function correctAssociation(
  session: SessionDocument,
  projectId: string,
  associationId: string,
  action: string,
  reason: string,
) {
  return unwrap<Document>(
    await postApiV1ProjectsProjectIdAssociationsAssociationIdCorrection({
      path: { project_id: projectId, association_id: associationId },
      body: { action, reason },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function requestSync(session: SessionDocument, projectId: string, scope = 'all') {
  return unwrap<JobDocument>(
    await postApiV1ProjectsProjectIdSyncs({
      path: { project_id: projectId },
      body: { scope },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function requestHistory(
  session: SessionDocument,
  projectId: string,
  from: string,
  reason: string,
) {
  return unwrap<JobDocument>(
    await postApiV1ProjectsProjectIdHistoryRequests({
      path: { project_id: projectId },
      body: { from, reason },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function fetchJob(jobId: string): Promise<JobDocument> {
  return unwrap<JobDocument>(await getApiV1JobsJobId({ path: { job_id: jobId } }));
}

export async function fetchMetrics(
  projectId: string,
  window = '90d',
  cutoff?: string,
): Promise<Page<MetricDocument> & { window: IntelligenceWindow }> {
  return unwrap(
    await getApiV1ProjectsProjectIdMetrics({
      path: { project_id: projectId },
      query: { window, cutoff, limit: 50 },
    }),
  );
}

export async function fetchMetric(
  projectId: string,
  metricName: string,
  window = '90d',
  cutoff?: string,
): Promise<MetricDocument> {
  return unwrap(
    await getApiV1ProjectsProjectIdMetricsMetricName({
      path: { project_id: projectId, metric_name: metricName },
      query: { window, cutoff },
    }),
  );
}

export async function fetchHealth(
  projectId: string,
  window = '90d',
  cutoff?: string,
): Promise<HealthDocument> {
  return unwrap(
    await getApiV1ProjectsProjectIdHealth({
      path: { project_id: projectId },
      query: { window, cutoff },
    }),
  );
}

export async function fetchContributors(
  projectId: string,
  window = '90d',
): Promise<ContributorsDocument> {
  return unwrap(
    await getApiV1ProjectsProjectIdContributors({
      path: { project_id: projectId },
      query: { window, limit: 50 },
    }),
  );
}

export async function createComparison(
  session: SessionDocument,
  projectIds: string[],
  window: string,
  cutoff: string,
): Promise<ComparisonDocument> {
  return unwrap(
    await postApiV1Comparisons({
      body: { project_ids: projectIds, window, cutoff },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function fetchComparison(comparisonId: string): Promise<ComparisonDocument> {
  return unwrap(await getApiV1ComparisonsComparisonId({ path: { comparison_id: comparisonId } }));
}

export async function fetchAdoption(projectId: string, window = '90d'): Promise<Page<Document>> {
  return unwrap(
    await getApiV1ProjectsProjectIdAdoption({
      path: { project_id: projectId },
      query: { window, limit: 50 },
    }),
  );
}

export async function fetchSecurity(projectId: string, window = '365d'): Promise<Document> {
  return unwrap(
    await getApiV1ProjectsProjectIdSecurity({
      path: { project_id: projectId },
      query: { window, limit: 50 },
    }),
  );
}

export async function fetchTopics(projectId: string, window = '90d'): Promise<Page<Document>> {
  return unwrap(
    await getApiV1ProjectsProjectIdTopics({
      path: { project_id: projectId },
      query: { window, limit: 50 },
    }),
  );
}

export async function correctTopic(
  session: SessionDocument,
  projectId: string,
  topicId: string,
  body: Document,
  version = 0,
): Promise<Document> {
  return unwrap(
    await postApiV1ProjectsProjectIdTopicsTopicIdCorrections({
      path: { project_id: projectId, topic_id: topicId },
      body,
      headers: mutationHeaders(session, true, version),
    }),
  );
}

export async function fetchReleases(projectId: string): Promise<Page<Document>> {
  return unwrap(
    await getApiV1ProjectsProjectIdReleases({
      path: { project_id: projectId },
      query: { limit: 50 },
    }),
  );
}

export async function fetchRelease(projectId: string, releaseId: string): Promise<Document> {
  return unwrap(
    await getApiV1ProjectsProjectIdReleasesReleaseId({
      path: { project_id: projectId, release_id: releaseId },
    }),
  );
}

export async function requestCrawl(
  session: SessionDocument,
  projectId: string,
  sourceIds: string[],
  maxDepth: number,
): Promise<JobDocument> {
  return unwrap(
    await postApiV1ProjectsProjectIdCrawls({
      path: { project_id: projectId },
      body: { source_ids: sourceIds, max_depth: maxDepth },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function searchKnowledge(projectId: string, query: string): Promise<Document> {
  return unwrap(
    await postApiV1ProjectsProjectIdKnowledgeSearch({
      path: { project_id: projectId },
      body: { query, language: 'en', limit: 10 },
    }),
  );
}

export async function askProject(
  session: SessionDocument,
  projectId: string,
  question: string,
): Promise<Document> {
  return unwrap(
    await postApiV1ProjectsProjectIdQueries({
      path: { project_id: projectId },
      body: { question, window: '90d', language: 'en' },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function fetchAnalysisRun(runId: string): Promise<Document> {
  return unwrap(await getApiV1AnalysisRunsRunId({ path: { run_id: runId } }));
}

export async function rerunAnalysis(
  session: SessionDocument,
  runId: string,
  reason: string,
): Promise<Document> {
  return unwrap(
    await postApiV1AnalysisRunsRunIdReruns({
      path: { run_id: runId },
      body: { language: 'en', reason },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function submitAnalysisFeedback(
  session: SessionDocument,
  runId: string,
  rating: string,
  comment: string,
): Promise<Document> {
  return unwrap(
    await postApiV1AnalysisRunsRunIdFeedback({
      path: { run_id: runId },
      body: { rating, comment },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function selectAnalysisRun(
  session: SessionDocument,
  seriesId: string,
  runId: string,
  version = 0,
): Promise<Document> {
  return unwrap(
    await postApiV1AnalysisSeriesSeriesIdSelection({
      path: { series_id: seriesId },
      body: { run_id: runId },
      headers: mutationHeaders(session, true, version),
    }),
  );
}

export async function fetchTrends(
  projectId: string,
  kind: 'observed' | 'forecast',
  window = '365d',
): Promise<Page<Document>> {
  return unwrap(
    await getApiV1ProjectsProjectIdTrends({
      path: { project_id: projectId },
      query: { kind, window, limit: 50 },
    }),
  );
}

export async function fetchRecommendation(
  projectId: string,
  policy = 'default',
  window = '90d',
): Promise<Document> {
  return unwrap(
    await getApiV1ProjectsProjectIdRecommendation({
      path: { project_id: projectId },
      query: { policy, window },
    }),
  );
}

export async function fetchPolicies(state?: string): Promise<Page<Document>> {
  return unwrap(await getApiV1Policies({ query: { state, limit: 50 } }));
}

export async function createPolicy(session: SessionDocument, value: Document): Promise<Document> {
  return unwrap(await postApiV1Policies({ body: value, headers: mutationHeaders(session, true) }));
}

export async function fetchRadar(policy = 'default', window = '90d'): Promise<Document> {
  return unwrap(await getApiV1Radar({ query: { policy, window } }));
}

export async function overrideRadar(
  session: SessionDocument,
  projectId: string,
  ring: string,
  reason: string,
  owner: string,
  reviewOn: string,
): Promise<Document> {
  return unwrap(
    await postApiV1RadarProjectIdOverride({
      path: { project_id: projectId },
      body: { ring, reason, owner, review_on: reviewOn },
      headers: mutationHeaders(session, true),
    }),
  );
}

export async function fetchAlerts(state?: string, project?: string): Promise<Page<Document>> {
  return unwrap(await getApiV1Alerts({ query: { state, project, limit: 50 } }));
}

export async function markAlertRead(session: SessionDocument, alertId: string): Promise<void> {
  unwrap(
    await postApiV1AlertsAlertIdRead({
      path: { alert_id: alertId },
      body: {},
      headers: mutationHeaders(session, false),
    }),
  );
}

export async function transitionAlert(
  session: SessionDocument,
  alertId: string,
  version: number,
  to: string,
  reason: string,
): Promise<Document> {
  return unwrap(
    await postApiV1AlertsAlertIdTransition({
      path: { alert_id: alertId },
      body: { to, reason },
      headers: mutationHeaders(session, false, version),
    }),
  );
}

function mutationHeaders(
  session: SessionDocument,
  idempotent: boolean,
  version?: number,
): Record<string, string> {
  const headers: Record<string, string> = {
    'X-CSRF-Token': session.csrf_token ?? '',
    'Sec-Fetch-Site': 'same-origin',
  };
  if (idempotent) headers['Idempotency-Key'] = crypto.randomUUID();
  if (version !== undefined) headers['If-Match'] = `"v${version}"`;
  return headers;
}

function idempotencyHeaders(
  session: SessionDocument,
): Record<string, string> & { 'Idempotency-Key': string } {
  const headers = mutationHeaders(session, true);
  return {
    ...headers,
    'Idempotency-Key': headers['Idempotency-Key'] ?? crypto.randomUUID(),
  };
}
