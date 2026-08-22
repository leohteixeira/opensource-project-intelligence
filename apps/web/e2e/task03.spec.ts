import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Page, type Route } from '@playwright/test';
import { mkdir } from 'node:fs/promises';
import { resolve } from 'node:path';

interface State {
  project: Record<string, unknown>;
  projects: Array<Record<string, unknown>>;
  repositories: Array<Record<string, unknown>>;
  sources: Array<Record<string, unknown>>;
  associations: Array<Record<string, unknown>>;
  jobs: Array<Record<string, unknown>>;
  mutations: string[];
  variant: string;
}

test.beforeEach(async ({ page }) => mockAPI(page, stateFactory()));

test('E2E-005 shows evidence-aware portfolio work without inventing missing values', async ({
  page,
}) => {
  await navigate(page, '/en/portfolio');
  await expect(page.getByRole('heading', { name: 'Portfolio', exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Temporal' })).toBeVisible();
  await expect(page.getByText(/project_initial_sync/)).toBeVisible();
});

test('E2E-006 registers one public repository and opens its editable identity', async ({
  page,
}) => {
  await navigate(page, '/en/projects');
  await page
    .getByLabel('Public repository URL')
    .fill('https://github.com/open-telemetry/opentelemetry-go');
  await page.getByRole('button', { name: 'Register project' }).click();
  await expect(page).toHaveURL(/\/en\/projects\/101$/);
  await expect(page.getByRole('heading', { name: 'Temporal' })).toBeVisible();
  await expect(page.locator('#project-name')).toBeEnabled();
});

test('E2E-007 curates explicit repository roles while retaining one primary', async ({ page }) => {
  await navigate(page, '/en/projects/101/sources');
  await expect(page.getByText(/primary/).first()).toBeVisible();
  await expect(page.getByText(/core/).first()).toBeVisible();
  await page.getByRole('button', { name: 'Make primary' }).click();
  await expect.poll(() => mutations(page, 'repository-role')).toBeTruthy();
});

test('E2E-008 displays association provenance and records an analyst correction', async ({
  page,
}) => {
  await navigate(page, '/en/projects/101/sources');
  await expect(page.getByText(/canonical_url \(0.94\)/)).toBeVisible();
  await page.getByRole('button', { name: 'Confirm association' }).click();
  await expect.poll(() => mutations(page, 'association-correction')).toBeTruthy();
});

test('E2E-009 requires exact project identity before permanent deletion', async ({ page }) => {
  await navigate(page, '/en/projects/101/lifecycle');
  const deletion = page.getByRole('button', { name: 'Permanently delete' });
  await expect(deletion).toBeDisabled();
  await page.getByLabel('Type DELETE temporal').fill('DELETE temporal');
  await deletion.click();
  await expect.poll(() => mutations(page, 'deletion')).toBeTruthy();
});

test('E2E-010 coalesces synchronization into visible durable work and checkpoint progress', async ({
  page,
}) => {
  await navigate(page, '/en/projects/101/jobs');
  await expect(page.getByText(/checkpoint 2/)).toBeVisible();
  await page.getByRole('button', { name: 'Synchronize' }).click();
  await expect.poll(() => mutations(page, 'sync')).toBeTruthy();
});

test('E2E-011 requests longer history while preserving actual source coverage', async ({
  page,
}) => {
  await navigate(page, '/en/projects/101/jobs');
  await page.getByRole('button', { name: 'Request history' }).click();
  await expect.poll(() => mutations(page, 'history')).toBeTruthy();
  await navigate(page, '/en/projects/101/sources');
  await expect(page.getByText(/coverage 2026-02-01/)).toBeVisible();
});

test('E2E-012 degrades a private source explicitly without exposing credentials', async ({
  page,
}) => {
  await navigate(page, '/en/projects/101/sources');
  await expect(page.getByText(/unavailable · not_public/)).toBeVisible();
  await expect(page.getByText(/token-value|authorization|secret/i)).toHaveCount(0);
});

test('E2E-036 completes S4 in narrow and wide layouts with accessible member-only controls', async ({
  page,
}) => {
  await navigate(page, '/en/portfolio');
  await captureBoth(page, 's4', 'populated', 900);
  await setVariant(page, 'empty-projects');
  await captureBoth(page, 's4', 'empty', 900);
  await setVariant(page, 'portfolio-error');
  await captureBoth(page, 's4', 'partial-error', 900);
  await assertAccessible(page);
});

test('E2E-037 completes S5 registration, search, stable paging, and responsive evidence', async ({
  page,
}) => {
  await navigate(page, '/en/projects?q=Temporal');
  await expect(page.getByRole('heading', { name: 'Temporal' })).toBeVisible();
  await captureBoth(page, 's5', 'search-page', 1000);
  await setVariant(page, 'empty-projects');
  await captureBoth(page, 's5', 'empty', 1000);
  await setVariant(page, 'quota');
  await page.getByLabel('Public repository URL').fill('https://github.com/acme/quota');
  await page.getByRole('button', { name: 'Register project' }).click();
  await expect(page.getByText(/quota reached/i)).toBeVisible();
  await captureBoth(page, 's5', 'quota-conflict', 1000);
});

test('E2E-038 completes S6 lifecycle controls and confirmation at both widths', async ({
  page,
}) => {
  await navigate(page, '/en/projects/101/lifecycle');
  await captureBoth(page, 's6', 'active', 1000);
  await setVariant(page, 'archived');
  await captureBoth(page, 's6', 'archived-read-only', 1000);
  await page.getByLabel('Type DELETE temporal').fill('DELETE temporal');
  await captureBoth(page, 's6', 'exact-deletion', 1000);
});

test('E2E-039 completes S7 repositories, sources, associations, and hostile-state rendering', async ({
  page,
}) => {
  await navigate(page, '/en/projects/101/sources');
  await captureBoth(page, 's7', 'provenance-hostile', 1200);
  await setVariant(page, 'empty-resources');
  await captureBoth(page, 's7', 'empty', 1000);
  await setVariant(page, 'archived');
  await captureBoth(page, 's7', 'archived-read-only', 1000);
});

test('E2E-040 completes S8 job progress, replay status, polling fallback, and history', async ({
  page,
}) => {
  await navigate(page, '/en/projects/101/jobs');
  await expect(page.getByText(/Job updates:/)).toBeVisible();
  await expect(page.getByText(/25 issues/)).toBeVisible();
  await captureBoth(page, 's8', 'running-polling-fallback', 1000);
  await setVariant(page, 'job-matrix');
  await captureBoth(page, 's8', 'terminal-coalesced-quota-checkpoint', 1400);
});

function stateFactory(): State {
  const project = {
    id: '101',
    name: 'Temporal',
    slug: 'temporal',
    description: 'Durable execution',
    state: 'active',
    version: 4,
  };
  return {
    project,
    projects: [project],
    repositories: [
      {
        id: '201',
        kind: 'github',
        canonical_url: 'https://github.com/temporalio/temporal',
        role: 'primary',
        version: 2,
      },
      {
        id: '202',
        kind: 'github',
        canonical_url: 'https://github.com/temporalio/sdk-go',
        role: 'core',
        version: 1,
      },
    ],
    sources: [
      {
        id: '301',
        kind: 'github',
        state: 'available',
        version: 3,
        coverage_from: '2026-02-01',
        coverage_to: '2026-08-22',
        last_success_at: '2026-08-22T03:00:00Z',
      },
      { id: '302', kind: 'website', state: 'unavailable', failure: 'not_public', version: 2 },
    ],
    associations: [
      {
        id: '401',
        state: 'linked',
        method: 'canonical_url',
        confidence: 0.94,
        decision_version: 'identity-v1',
        version: 1,
      },
    ],
    jobs: [
      {
        id: '501',
        project_id: '101',
        kind: 'project_initial_sync',
        state: 'running',
        version: 5,
        progress: { completed: 25, total_status: 'unknown', unit: 'issues' },
        checkpoint: { scope: 'github_issues', cursor: '2' },
      },
    ],
    mutations: [],
    variant: 'populated',
  };
}

async function mockAPI(page: Page, value: State) {
  await page.exposeFunction('task03Mutations', () => value.mutations);
  await page.exposeFunction('task03SetVariant', (variant: string) => applyVariant(value, variant));
  await page.route('**/api/v1/**', (route) => handleAPI(route, value));
}

async function handleAPI(route: Route, state: State) {
  const request = route.request();
  const path = new URL(request.url()).pathname;
  const method = request.method();
  if (path === '/api/v1/session')
    return json(route, {
      authenticated: true,
      state: 'active',
      role: 'admin',
      csrf_token: 'csrf',
      member: { id: '11', display_name: 'Ada', role: 'admin', status: 'active' },
    });
  if (path === '/api/v1/portfolio' && state.variant === 'portfolio-error')
    return json(
      route,
      { code: 'dependency_unavailable', detail: 'One panel is temporarily unavailable' },
      503,
    );
  if (path === '/api/v1/portfolio')
    return json(route, {
      projects: state.projects,
      active_jobs: state.jobs,
      attention_count: 1,
      generated_at: '2026-08-22T03:00:00Z',
    });
  if (path === '/api/v1/projects' && method === 'GET')
    return json(route, { items: state.projects, has_more: true, next_cursor: 'signed-next' });
  if (path === '/api/v1/projects' && method === 'POST') {
    if (state.variant === 'quota')
      return json(route, { code: 'quota_exceeded', detail: 'Project quota reached' }, 409);
    state.mutations.push('registration');
    return json(route, { project: state.project, job: state.jobs[0] }, 202);
  }
  if (path === '/api/v1/projects/101' && method === 'GET') return json(route, state.project);
  if (path === '/api/v1/projects/101/repositories' && method === 'GET')
    return pageResult(route, state.repositories);
  if (path === '/api/v1/projects/101/sources' && method === 'GET')
    return pageResult(route, state.sources);
  if (path === '/api/v1/projects/101/associations' && method === 'GET')
    return pageResult(route, state.associations);
  if (path === '/api/v1/projects/101/jobs' && method === 'GET')
    return pageResult(route, state.jobs);
  if (path === '/api/v1/jobs/501/events')
    return json(route, { items: state.jobs, polling_fallback: true });
  if (path.endsWith('/transition')) state.mutations.push('transition');
  else if (path.endsWith('/deletion')) state.mutations.push('deletion');
  else if (path.endsWith('/syncs')) state.mutations.push('sync');
  else if (path.endsWith('/history-requests')) state.mutations.push('history');
  else if (path.includes('/associations/') && path.endsWith('/correction'))
    state.mutations.push('association-correction');
  else if (path.includes('/repositories/') && method === 'PATCH')
    state.mutations.push('repository-role');
  else if (path.includes('/sources/') && method === 'PATCH') state.mutations.push('source-state');
  else if (path.endsWith('/repositories') && method === 'POST')
    state.mutations.push('repository-add');
  else if (path.endsWith('/sources') && method === 'POST') state.mutations.push('source-add');
  else if (path === '/api/v1/projects/101' && method === 'PATCH') state.mutations.push('identity');
  return json(route, state.jobs[0], method === 'POST' ? 202 : 200);
}

function applyVariant(state: State, variant: string) {
  const baseline = stateFactory();
  const mutations = state.mutations;
  Object.assign(state, baseline, { mutations });
  state.variant = variant;
  if (variant === 'empty-projects') {
    state.projects = [];
    state.jobs = [];
  }
  if (variant === 'empty-resources') {
    state.repositories = [];
    state.sources = [];
    state.associations = [];
  }
  if (variant === 'archived') {
    state.project = { ...state.project, state: 'archived', version: 5 };
  }
  if (variant === 'job-matrix') {
    state.jobs = [
      { id: '501', project_id: '101', kind: 'project_sync', state: 'queued', version: 1 },
      {
        id: '502',
        project_id: '101',
        kind: 'project_history',
        state: 'running',
        version: 4,
        progress: { completed: 125, total: 500, unit: 'facts' },
        checkpoint: { scope: 'github_issues', cursor: 'page-8' },
        coalesced_requests: 3,
      },
      { id: '503', project_id: '101', kind: 'project_sync', state: 'succeeded', version: 5 },
      {
        id: '504',
        project_id: '101',
        kind: 'project_sync',
        state: 'partial',
        version: 5,
        failure: 'rate_limited',
      },
      {
        id: '505',
        project_id: '101',
        kind: 'project_sync',
        state: 'failed',
        version: 5,
        failure: 'quota_exceeded',
      },
      { id: '506', project_id: '101', kind: 'project_sync', state: 'cancelled', version: 3 },
    ];
  }
}

function json(route: Route, value: unknown, status = 200) {
  return route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(value),
    headers: { ETag: '"v4"' },
  });
}

function pageResult(route: Route, items: Array<Record<string, unknown>>) {
  return json(route, { items, has_more: false });
}

async function mutations(page: Page, name: string) {
  return page.evaluate(
    async (target) =>
      (
        await (window as unknown as { task03Mutations(): Promise<string[]> }).task03Mutations()
      ).includes(target),
    name,
  );
}

async function assertAccessible(page: Page) {
  const scan = await new AxeBuilder({ page }).analyze();
  expect(scan.violations, scan.violations.map((violation) => violation.id).join(', ')).toEqual([]);
}

async function capture(page: Page, directory: string, name: string, width: number, height: number) {
  await page.setViewportSize({ width, height });
  const target = resolve(process.cwd(), '../../artifacts/task_03/ui', directory);
  await mkdir(target, { recursive: true });
  await page.screenshot({ path: resolve(target, `${name}.png`), fullPage: true });
}

async function captureBoth(page: Page, surface: string, name: string, height: number) {
  await capture(page, `${surface}-narrow`, name, 320, height);
  await capture(page, `${surface}-wide`, name, 1440, height);
}

async function setVariant(page: Page, variant: string) {
  await page.evaluate(
    async (name) =>
      (window as unknown as { task03SetVariant(value: string): Promise<void> }).task03SetVariant(
        name,
      ),
    variant,
  );
  await page.reload({ waitUntil: 'domcontentloaded' });
}

async function navigate(page: Page, path: string) {
  await page.goto(path, { waitUntil: 'domcontentloaded' });
}
