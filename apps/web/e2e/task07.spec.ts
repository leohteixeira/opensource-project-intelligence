import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Page, type Route } from '@playwright/test';
import { mkdir } from 'node:fs/promises';
import { resolve } from 'node:path';

interface FixtureState {
  proposalExecuted: boolean;
  exportRequests: Array<Record<string, unknown>>;
  auditQuery: string;
}

let fixtureState: FixtureState;

test.beforeEach(async ({ page }) => {
  fixtureState = { proposalExecuted: false, exportRequests: [], auditQuery: '' };
  await page.route('**/api/v1/**', (route) => handleAPI(route, fixtureState));
});

test('E2E-025 previews, confirms, audits, and refuses bounded assistant actions', async ({
  page,
}) => {
  await page.goto('/en/assistant', { waitUntil: 'domcontentloaded' });
  await page.getByLabel('Question or action').fill('Add the public SDK repository');
  await page.getByRole('button', { name: 'Analyze' }).click();
  await expect(page.getByText('repository.add')).toBeVisible();
  await expect(page.getByText(/project_id.*101/)).toBeVisible();
  await expect(page.getByText('project:101')).toBeVisible();
  await expect(page.getByText('One public repository will be attached.')).toBeVisible();
  await expect(page.getByText('1 · 9 remaining')).toBeVisible();
  await expect(page.getByText('2026-08-27T12:10:00Z')).toBeVisible();
  await page.getByRole('button', { name: 'Confirm once' }).click();
  await expect(page.getByRole('heading', { name: 'Execution receipt' })).toBeVisible();
  await expect(page.getByText('audit-9001')).toBeVisible();
  expect(fixtureState.proposalExecuted).toBe(true);

  await page.getByLabel('Question or action').fill('Delete project 101');
  await page.getByRole('button', { name: 'Analyze' }).click();
  await expect(page.getByText('Use the conventional project deletion surface.')).toBeVisible();
});

test('E2E-028 requests localized CSV and evidence JSON from one immutable cutoff', async ({
  page,
}) => {
  await page.goto('/pt-br/exportacoes', { waitUntil: 'domcontentloaded' });
  await page.getByLabel('ID do projeto').fill('101');
  await page.getByLabel('Recurso').selectOption('metrics');
  await page.getByLabel('Formato').selectOption('csv');
  await page.getByRole('button', { name: 'Solicitar exportação' }).click();
  await expect(page.getByText('queued', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Atualizar' }).click();
  await expect(page.getByText('succeeded', { exact: true })).toBeVisible();
  await expect(page.getByText('sha256-task07')).toBeVisible();
  await expect(page.getByRole('link', { name: 'Baixar' })).toHaveAttribute(
    'href',
    '/api/v1/exports/8001/download',
  );
  expect(fixtureState.exportRequests[0]).toMatchObject({
    project_ids: [101],
    resource: 'metrics',
    format: 'csv',
    locale: 'pt-BR',
  });
  expect(fixtureState.exportRequests[0]?.cutoff).toBe(fixtureState.exportRequests[0]?.window_to);

  await page.getByLabel('Formato').selectOption('evidence_json');
  await page.getByRole('button', { name: 'Solicitar exportação' }).click();
  await expect(page.getByText('queued', { exact: true })).toBeVisible();
  expect(fixtureState.exportRequests[1]).toMatchObject({
    project_ids: [101],
    resource: 'metrics',
    format: 'evidence_json',
    locale: 'pt-BR',
  });
});

test('E2E-029 filters immutable redacted audit history in stable order', async ({ page }) => {
  await page.goto('/en/admin/audit', { waitUntil: 'domcontentloaded' });
  await page.getByLabel('Actor').fill('analyst-11');
  await page.getByLabel('Action').fill('assistant.repository.add');
  await page.getByLabel('Resource').fill('project');
  await page.getByLabel('Outcome').selectOption('succeeded');
  await page.getByLabel('From').fill('2026-08-01');
  await page.getByLabel('To', { exact: true }).fill('2026-08-27');
  await page.getByRole('button', { name: 'Apply filters' }).click();
  await expect(page.getByRole('cell', { name: 'analyst-11' })).toBeVisible();
  await expect(page.getByRole('cell', { name: 'assistant.repository.add' })).toBeVisible();
  await expect(page.getByRole('cell', { name: 'succeeded' })).toBeVisible();
  await expect(page.getByText('secret-token')).toHaveCount(0);
  expect(fixtureState.auditQuery).toContain('actor=analyst-11');
  expect(fixtureState.auditQuery).toContain('action=assistant.repository.add');
  expect(fixtureState.auditQuery).toContain('resource=project');
  expect(fixtureState.auditQuery).toContain('outcome=succeeded');
});

test('E2E-030 exposes redacted provider operations while deterministic work remains available', async ({
  page,
}) => {
  await page.goto('/en/admin/operations', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Operations' })).toBeVisible();
  await expect(page.getByText('local-openai')).toBeVisible();
  await expect(page.getByText('qwen-3')).toBeVisible();
  await expect(page.getByText('degraded').first()).toBeVisible();
  await expect(page.getByText(/requests.*14/)).toBeVisible();
  await expect(page.getByText('secret-token')).toHaveCount(0);
  await expect(page.getByText('metrics')).toBeVisible();
  await expect(page.getByText('Enabled · available')).toBeVisible();
});

test('E2E-051 completes S19 in both locales at narrow and wide viewports', async ({ page }) => {
  await verifySurface(page, {
    enPath: '/en/assistant',
    ptPath: '/pt-br/assistente',
    enHeading: 'Analysis assistant',
    ptHeading: 'Assistente de análise',
    surface: 's19',
  });
});

test('E2E-054 completes S22 in both locales at narrow and wide viewports', async ({ page }) => {
  await verifySurface(page, {
    enPath: '/en/exports',
    ptPath: '/pt-br/exportacoes',
    enHeading: 'Exports',
    ptHeading: 'Exportações',
    surface: 's22',
  });
});

interface SurfaceExpectation {
  enPath: string;
  ptPath: string;
  enHeading: string;
  ptHeading: string;
  surface: 's19' | 's22';
}

async function verifySurface(page: Page, expected: SurfaceExpectation) {
  for (const [locale, path, heading] of [
    ['en', expected.enPath, expected.enHeading],
    ['pt-br', expected.ptPath, expected.ptHeading],
  ] as const) {
    await page.goto(path, { waitUntil: 'domcontentloaded' });
    await expect(page.getByRole('heading', { name: heading }).first()).toBeVisible();
    await capture(page, expected.surface, locale);
  }
}

async function handleAPI(route: Route, state: FixtureState) {
  const request = route.request();
  const url = new URL(request.url());
  const path = url.pathname;
  if (path === '/api/v1/session') return json(route, session());
  if (path === '/api/v1/assistant/proposals' && request.method() === 'POST') {
    const body = request.postDataJSON() as { message?: string };
    if (body.message?.toLowerCase().includes('delete')) {
      return json(
        route,
        {
          code: 'action_not_allowed',
          detail: 'Use the conventional project deletion surface.',
        },
        422,
      );
    }
    return json(route, proposal(false), 201);
  }
  if (path === '/api/v1/assistant/proposals/7001/confirmation' && request.method() === 'POST') {
    state.proposalExecuted = true;
    return json(route, proposal(true), 201);
  }
  if (path === '/api/v1/exports' && request.method() === 'POST') {
    state.exportRequests.push(request.postDataJSON() as Record<string, unknown>);
    return json(route, exportDocument('queued'), 202);
  }
  if (path === '/api/v1/exports/8001' && request.method() === 'GET') {
    return json(route, exportDocument('succeeded'));
  }
  if (path === '/api/v1/admin/audit') {
    state.auditQuery = url.search;
    return json(route, auditPage());
  }
  if (path === '/api/v1/admin/operations') return json(route, operations());
  return json(route, { code: 'not_found', detail: `No Task 7 fixture for ${path}` }, 404);
}

function session() {
  return {
    authenticated: true,
    state: 'active',
    role: 'admin',
    csrf_token: 'task07-csrf',
    member: { id: 'analyst-11', display_name: 'Ada', role: 'admin', status: 'active' },
  };
}

function proposal(executed: boolean) {
  return {
    id: '7001',
    status: executed ? 'executed' : 'awaiting_confirmation',
    action: 'repository.add',
    inputs: {
      project_id: 101,
      project_version: 7,
      url: 'https://github.com/example/sdk',
      role: 'sdk',
    },
    resources: ['project:101'],
    effect: 'One public repository will be attached.',
    quota: { name: 'repository_add', cost: 1, remaining: 9 },
    expires_at: '2026-08-27T12:10:00Z',
    confirmation_token: executed ? undefined : 'single-use-task07',
    result: executed ? { repository_id: '3001', audit_event_id: 'audit-9001' } : undefined,
  };
}

function exportDocument(state: 'queued' | 'succeeded') {
  return {
    id: '8001',
    job_id: '8101',
    state,
    row_count: state === 'succeeded' ? 3 : 0,
    media_type: 'text/csv',
    sha256: state === 'succeeded' ? 'sha256-task07' : undefined,
    size_bytes: state === 'succeeded' ? 512 : undefined,
    download_url: state === 'succeeded' ? '/api/v1/exports/8001/download' : undefined,
    expires_at: '2026-08-28T12:00:00Z',
  };
}

function auditPage() {
  return {
    items: [
      {
        id: '9001',
        occurred_at: '2026-08-27T12:01:00Z',
        actor_kind: 'member',
        actor_id: 'analyst-11',
        action: 'assistant.repository.add',
        resource_type: 'project',
        resource_id: '101',
        outcome: 'succeeded',
        metadata: { proposal_id: '7001', fields: ['repository_id'] },
      },
    ],
    has_more: false,
  };
}

function operations() {
  return {
    status: 'degraded',
    health: { database: 'available', deterministic_engine: 'available' },
    capabilities: [
      { name: 'metrics', configured: true, status: 'available' },
      { name: 'assistant', configured: true, status: 'degraded' },
    ],
    model_provider: {
      provider: 'local-openai',
      model: 'qwen-3',
      health: 'degraded',
      enabled: true,
      capabilities: ['analysis', 'assistant'],
      revision: 4,
      usage: { requests: 14, input_tokens: 1200, output_tokens: 320, cost_micros: 0 },
      redacted: true,
    },
  };
}

async function capture(page: Page, surface: string, locale: string) {
  for (const [width, height, viewport] of [
    [320, 1000, 'narrow'],
    [1280, 1000, 'wide'],
  ] as const) {
    await page.setViewportSize({ width, height });
    if (viewport === 'narrow') {
      await page
        .getByRole('navigation', { name: /Primary navigation|Navegação principal/ })
        .evaluate((navigation) => navigation.style.setProperty('position', 'static'));
    }
    const results = await new AxeBuilder({ page }).analyze();
    expect(results.violations).toEqual([]);
    const directory = resolve(process.cwd(), `../../artifacts/task_07/ui/${surface}-${viewport}`);
    await mkdir(directory, { recursive: true });
    await page.screenshot({ path: resolve(directory, `${locale}.png`), fullPage: true });
  }
}

function json(route: Route, body: unknown, status = 200) {
  return route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
}
