import {
  getApiV1AdminAudit,
  getApiV1AdminMembers,
  getApiV1AdminOperations,
  getApiV1AdminServiceAccounts,
  getApiV1CatalogProjects,
  getApiV1CatalogProjectsProjectId,
  getApiV1Session,
  patchApiV1AdminMembersMemberId,
  patchApiV1AdminServiceAccountsServiceAccountId,
  patchApiV1MePreferences,
  postApiV1AdminMembersMemberIdApproval,
  postApiV1AdminServiceAccounts,
  postApiV1MeDeletion,
  postApiV1SessionLogout,
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

export async function fetchAudit(): Promise<Page<Document>> {
  return unwrap<Page<Document>>(await getApiV1AdminAudit({ query: { limit: 50 } }));
}

export async function fetchOperations(): Promise<Document> {
  return unwrap<Document>(await getApiV1AdminOperations());
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
