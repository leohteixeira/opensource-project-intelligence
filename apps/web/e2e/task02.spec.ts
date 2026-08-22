import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Page, type Route } from '@playwright/test';
import { mkdir } from 'node:fs/promises';
import { resolve } from 'node:path';

interface MockState {
  session: Record<string, unknown>;
  sessionFailure?: boolean;
  preferenceConflict?: boolean;
  catalog: Array<Record<string, unknown>>;
  members: Array<Record<string, unknown>>;
  serviceAccounts: Array<Record<string, unknown>>;
  audit: Array<Record<string, unknown>>;
  mutations: string[];
}

test('E2E-001 browses public identity without protected disclosure', async ({ page }) => {
  const state = anonymousState();
  await mockAPI(page, state);
  await page.goto('/en/catalog', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { name: 'Explore open source projects' })).toBeVisible();
  await expect(page.getByText('Temporal')).toBeVisible();
  await expect(page.getByText(/risk score|maintainer analysis/i)).toHaveCount(0);
  await page.getByPlaceholder('Search projects').fill('temporal');
  await page.getByRole('button', { name: 'Search' }).click();
  await expect(page).toHaveURL(/q=temporal/);
  await page.goto('/en/catalog/101', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Temporal' })).toBeVisible();
  await expect(page.getByText('Protected intelligence')).toBeVisible();
  await expect(
    page.getByRole('link', { name: 'https://github.com/temporalio/temporal' }),
  ).toBeVisible();
  await assertAccessible(page);

  await capture(page, 's2-narrow', 'public-catalog', 320, 800);
  await capture(page, 's2-wide', 'public-catalog', 1440, 900);
});

test('E2E-002 authenticates an applicant and exposes access only after local approval', async ({
  page,
}) => {
  const state = memberState('pending', 'viewer');
  await mockAPI(page, state);
  await page.goto('/en/access', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { name: 'Access request pending' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Account' })).toHaveCount(0);
  await capture(page, 's1-narrow', 'pending-shell', 320, 800);
  await capture(page, 's1-wide', 'pending-shell', 1440, 900);

  state.session = activeSession('viewer');
  await page.reload({ waitUntil: 'domcontentloaded' });
  await expect(page.getByText('Your session expired')).toHaveCount(0);
  await page.goto('/en/account', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Account', exact: true })).toBeVisible();
});

test('E2E-003 governs membership and exposes attributable audit history', async ({ page }) => {
  const state = memberState('active', 'admin');
  await mockAPI(page, state);
  await page.goto('/en/admin/members', { waitUntil: 'domcontentloaded' });

  await page.getByRole('button', { name: 'Approve as Viewer' }).click();
  await expect
    .poll(() => state.mutations.filter((value) => value === 'approve-member').length)
    .toBe(1);
  await expect(page.getByText('The membership decision was recorded.')).toBeVisible();

  await page.goto('/en/admin/audit', { waitUntil: 'domcontentloaded' });
  await expect(page.getByText('membership.approved')).toBeVisible();
  await expect(page.getByText('member', { exact: true })).toBeVisible();
});

test('E2E-004 saves preferences once and queues exact-confirmation account removal', async ({
  page,
}) => {
  const state = memberState('active', 'viewer');
  await mockAPI(page, state);
  await page.goto('/en/account', { waitUntil: 'domcontentloaded' });

  await capture(page, 's3-narrow', 'account', 320, 900);
  await capture(page, 's3-wide', 'account', 1440, 900);
  await page.locator('#account-locale').selectOption('pt-BR');
  await page.getByLabel('Timezone').fill('America/Sao_Paulo');
  await page.getByRole('button', { name: 'Save preferences' }).click();
  await expect(page.getByText('Preferences saved')).toBeVisible();
  await expect
    .poll(() => state.mutations.filter((value) => value === 'preferences').length)
    .toBe(1);

  await page.getByLabel(/Type DELETE MY ACCOUNT/).fill('DELETE MY ACCOUNT');
  await page.getByRole('button', { name: 'Delete my account' }).click();
  await expect(
    page.getByText('Account removal has started. Your sessions are now revoked.'),
  ).toBeVisible();
  await expect.poll(() => state.mutations.filter((value) => value === 'deletion').length).toBe(1);
});

test('E2E-031 keeps the Portuguese mobile journey usable at 200% with reduced motion', async ({
  page,
}) => {
  const state = anonymousState();
  await mockAPI(page, state);
  await page.setViewportSize({ width: 320, height: 800 });
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/pt-br/catalogo', { waitUntil: 'domcontentloaded' });
  await page.evaluate(() => {
    document.documentElement.style.fontSize = '200%';
  });

  await expect(
    page.getByRole('heading', { name: 'Explore projetos de código aberto' }),
  ).toBeVisible();
  await expect(page.getByRole('button', { name: 'Pesquisar' })).toBeVisible();
  await expect(page.getByText('Workflow orchestration engine')).toBeVisible();
  await assertAccessible(page);
});

test('E2E-032 applies current local service-account state and attribution', async ({ page }) => {
  const state = memberState('active', 'admin');
  await mockAPI(page, state);
  await page.goto('/en/admin/service-accounts', { waitUntil: 'domcontentloaded' });

  await expect(page.getByText('opi-exporter')).toBeVisible();
  await page.getByRole('button', { name: 'Suspend' }).click();
  await expect
    .poll(() => state.mutations.filter((value) => value === 'service-update').length)
    .toBe(1);
  await page.goto('/en/admin/audit', { waitUntil: 'domcontentloaded' });
  await expect(page.getByText('service_account')).toBeVisible();
  await expect(page.getByText('export.started')).toBeVisible();
});

test('E2E-033 handles keyboard entry, offline state, unauthorized routes, and not-found routes', async ({
  page,
}) => {
  const state = anonymousState();
  state.sessionFailure = true;
  await mockAPI(page, state);
  await page.goto('/en/not-a-route', { waitUntil: 'domcontentloaded' });

  await expect(page.getByText('The service is temporarily unavailable')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Page not found' })).toBeVisible();
  await tabTo(page, page.getByRole('link', { name: 'Skip to content' }));
  await page.goto('/en/account', { waitUntil: 'domcontentloaded' });
  await expect(
    page.getByRole('heading', { name: 'You do not have access to this page' }),
  ).toBeVisible();
  await expect(page.getByRole('button', { name: 'Save preferences' })).toHaveCount(0);
});

test('E2E-034 represents empty, no-match, paged, and protected catalog states', async ({
  page,
}) => {
  const state = anonymousState();
  state.catalog = [];
  await mockAPI(page, state);
  await page.goto('/en/catalog?q=absent', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'No projects match this search' })).toBeVisible();

  state.catalog = projectRows();
  await page.goto('/en/catalog?page=2&cursor=signed-cursor', {
    waitUntil: 'domcontentloaded',
  });
  await expect(page.getByText('Page 2')).toBeVisible();
  await page.goto('/en/catalog/101', { waitUntil: 'domcontentloaded' });
  await expect(page.getByText(/require an approved local role/i)).toBeVisible();
  await expect(page.getByRole('link', { name: 'Sign in' }).first()).toBeVisible();
  await assertAccessible(page);
});

test('E2E-035 renders suspended access and recoverable preference conflicts without extra controls', async ({
  page,
}) => {
  const state = memberState('suspended', 'viewer');
  await mockAPI(page, state);
  await page.goto('/en/access', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Workspace access suspended' })).toBeVisible();

  state.session = activeSession('viewer');
  state.preferenceConflict = true;
  await page.goto('/en/account', { waitUntil: 'domcontentloaded' });
  await page.getByLabel('Timezone').fill('Europe/Lisbon');
  await page.getByRole('button', { name: 'Save preferences' }).click();
  await expect(
    page.getByText('This account changed elsewhere. Refresh before saving again.'),
  ).toBeVisible();
  await expect(page.getByText('Administration')).toHaveCount(0);
});

test('E2E-055 operates all redacted administration surfaces at narrow and wide widths', async ({
  page,
}) => {
  const state = memberState('active', 'admin');
  await mockAPI(page, state);
  await page.goto('/en/admin/members', { waitUntil: 'domcontentloaded' });
  await capture(page, 's23-narrow', 'members', 320, 900);
  await capture(page, 's23-wide', 'members', 1440, 900);

  await page.goto('/en/admin/service-accounts', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Service accounts' })).toBeVisible();
  await page.getByLabel('Name').fill('Evidence reader');
  await page.getByLabel('External subject').fill('evidence-reader');
  await page.getByRole('button', { name: 'Create service account' }).click();
  await expect
    .poll(() => state.mutations.filter((value) => value === 'service-create').length)
    .toBe(1);

  await page.goto('/en/admin/audit', { waitUntil: 'domcontentloaded' });
  await expect(page.getByText('These security events are append-only')).toBeVisible();
  await page.goto('/en/admin/operations', { waitUntil: 'domcontentloaded' });
  await expect(page.getByText('healthy', { exact: true })).toBeVisible();
  await expect(page.getByText(/Enabled · healthy/)).toBeVisible();
  await expect(page.getByText(/secret-value|https:\/\/keycloak/i)).toHaveCount(0);
  await assertAccessible(page);
});

function anonymousState(): MockState {
  return {
    session: { authenticated: false, state: 'anonymous' },
    catalog: projectRows(),
    members: memberRows(),
    serviceAccounts: serviceRows(),
    audit: auditRows(),
    mutations: [],
  };
}

function memberState(status: string, role: string): MockState {
  const state = anonymousState();
  state.session =
    status === 'active'
      ? activeSession(role)
      : {
          authenticated: true,
          state: status,
          role,
          csrf_token: 'csrf-browser',
          member: { id: '11', display_name: 'Ada Lovelace', status, role, version: 7 },
        };
  return state;
}

function activeSession(role: string): Record<string, unknown> {
  return {
    authenticated: true,
    state: 'active',
    role,
    csrf_token: 'csrf-browser',
    member: {
      id: '11',
      display_name: 'Ada Lovelace',
      email: 'ada@example.test',
      status: 'active',
      role,
      locale: 'en',
      timezone: 'UTC',
      version: 7,
    },
  };
}

function projectRows() {
  return [
    {
      id: '101',
      name: 'Temporal',
      slug: 'temporal',
      description: 'Workflow orchestration engine',
      source_links: ['https://github.com/temporalio/temporal'],
    },
    {
      id: '102',
      name: 'OpenTelemetry',
      slug: 'opentelemetry',
      description: 'Observability APIs and SDKs',
      source_links: ['https://github.com/open-telemetry'],
    },
  ];
}

function memberRows() {
  return [
    {
      id: '21',
      display_name: 'Grace Hopper',
      email: 'grace@example.test',
      status: 'applicant',
      version: 1,
    },
    {
      id: '22',
      display_name: 'Linus Torvalds',
      email: 'linus@example.test',
      role: 'viewer',
      status: 'active',
      version: 2,
    },
  ];
}

function serviceRows() {
  return [
    {
      id: '31',
      name: 'Portfolio exporter',
      external_subject: 'opi-exporter',
      role: 'viewer',
      scopes: ['projects:read'],
      status: 'active',
      version: 1,
    },
  ];
}

function auditRows() {
  return [
    {
      id: '41',
      occurred_at: '2026-08-21T12:00:00Z',
      action: 'membership.approved',
      actor_kind: 'member',
      resource_type: 'membership',
      outcome: 'success',
    },
    {
      id: '42',
      occurred_at: '2026-08-21T12:01:00Z',
      action: 'export.started',
      actor_kind: 'service_account',
      resource_type: 'export',
      outcome: 'success',
    },
  ];
}

async function mockAPI(page: Page, state: MockState) {
  await page.route('**/api/v1/**', async (route) => handleAPI(route, state));
}

async function handleAPI(route: Route, state: MockState) {
  const request = route.request();
  const url = new URL(request.url());
  const path = url.pathname;
  const method = request.method();

  if (path === '/api/v1/session' && method === 'GET') {
    if (state.sessionFailure) return json(route, { code: 'offline' }, 503);
    return json(route, state.session);
  }
  if (path === '/api/v1/session/logout') return json(route, {}, 204);
  if (path === '/api/v1/catalog/projects' && method === 'GET') {
    const q = (url.searchParams.get('q') ?? '').toLowerCase();
    const items = state.catalog.filter((item) => String(item.name).toLowerCase().includes(q));
    return json(route, {
      items,
      has_more: Boolean(url.searchParams.get('cursor')),
      next_cursor: 'next-signed',
    });
  }
  if (path.startsWith('/api/v1/catalog/projects/')) {
    const item = state.catalog.find((row) => String(row.id) === path.split('/').at(-1));
    return item ? json(route, item) : json(route, { code: 'not_found' }, 404);
  }
  if (path === '/api/v1/me/preferences') {
    state.mutations.push('preferences');
    if (state.preferenceConflict) return json(route, { code: 'precondition_failed' }, 412);
    const body = request.postDataJSON() as Record<string, unknown>;
    const member = {
      ...((state.session.member as Record<string, unknown>) ?? {}),
      ...body,
      version: 8,
    };
    state.session = { ...state.session, member };
    return json(route, member);
  }
  if (path === '/api/v1/me/deletion') {
    state.mutations.push('deletion');
    return json(route, { job_id: '81', status: 'queued' }, 202);
  }
  if (path === '/api/v1/admin/members' && method === 'GET') return page(route, state.members);
  if (path.endsWith('/approval')) {
    state.mutations.push('approve-member');
    state.members[0] = { ...state.members[0], status: 'active', role: 'viewer', version: 2 };
    return json(route, state.members[0]);
  }
  if (path.startsWith('/api/v1/admin/members/') && method === 'PATCH') {
    state.mutations.push('member-update');
    return json(route, request.postDataJSON());
  }
  if (path === '/api/v1/admin/service-accounts' && method === 'GET')
    return page(route, state.serviceAccounts);
  if (path === '/api/v1/admin/service-accounts' && method === 'POST') {
    state.mutations.push('service-create');
    const account = { id: '32', ...request.postDataJSON(), status: 'active', version: 1 };
    state.serviceAccounts.push(account);
    return json(route, account, 201);
  }
  if (path.startsWith('/api/v1/admin/service-accounts/') && method === 'PATCH') {
    state.mutations.push('service-update');
    state.serviceAccounts[0] = {
      ...state.serviceAccounts[0],
      ...request.postDataJSON(),
      version: 2,
    };
    return json(route, state.serviceAccounts[0]);
  }
  if (path === '/api/v1/admin/audit') return page(route, state.audit);
  if (path === '/api/v1/admin/operations') {
    return json(route, {
      status: 'healthy',
      capabilities: [{ name: 'external_identity', configured: true, status: 'healthy' }],
      redacted: true,
    });
  }
  return json(route, { code: 'not_found' }, 404);
}

function page(route: Route, items: Array<Record<string, unknown>>) {
  return json(route, { items, has_more: false });
}

function json(route: Route, value: unknown, status = 200) {
  return route.fulfill({
    status,
    contentType: 'application/json',
    body: status === 204 ? '' : JSON.stringify(value),
  });
}

async function assertAccessible(page: Page) {
  const scan = await new AxeBuilder({ page }).analyze();
  expect(scan.violations, scan.violations.map((violation) => violation.id).join(', ')).toEqual([]);
}

async function tabTo(page: Page, target: ReturnType<Page['locator']>) {
  for (let attempt = 0; attempt < 8; attempt += 1) {
    await page.keyboard.press('Tab');
    if (await target.evaluate((element) => element === document.activeElement)) return;
  }
  await expect(target).toBeFocused();
}

async function capture(page: Page, directory: string, name: string, width: number, height: number) {
  await page.setViewportSize({ width, height });
  const target = resolve(process.cwd(), '../../artifacts/task_02/ui', directory);
  await mkdir(target, { recursive: true });
  await page.screenshot({ path: resolve(target, `${name}.png`), fullPage: true });
}
