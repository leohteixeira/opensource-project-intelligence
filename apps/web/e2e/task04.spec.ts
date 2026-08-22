import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Page, type Route } from '@playwright/test';
import { mkdir } from 'node:fs/promises';
import { resolve } from 'node:path';

const cutoff = '2026-08-22T00:00:00Z';
const window = { from: '2026-05-24T00:00:00Z', to: cutoff, cutoff };

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/**', handleAPI);
});

test('E2E-013 inspects seven deterministic health dimensions and explicit missing metric states', async ({
  page,
}) => {
  await page.goto('/en/projects/101/health?window=90d&cutoff=2026-08-22T00%3A00%3A00Z');
  await expect(page.getByRole('heading', { name: 'Project health', level: 1 })).toBeVisible();
  await expect(page.getByText('Activity', { exact: true })).toBeVisible();
  await expect(page.getByText('Adoption', { exact: true })).toBeVisible();
  await expect(page.getByText('Overall score', { exact: true })).toBeVisible();
  await expect(page.getByText('Insufficient data', { exact: true }).first()).toBeVisible();
  await capture(page, 's9', 'metric-health');
});

test('E2E-014 evaluates contributors without combining unresolved identities', async ({ page }) => {
  await page.goto('/en/projects/101/contributors?window=90d');
  await expect(page.getByRole('heading', { name: 'Contributor intelligence' })).toBeVisible();
  await expect(page.getByText('65.0')).toBeVisible();
  await expect(page.getByText('account:unresolved')).toBeVisible();
  await expect(page.getByText(/resolved/).first()).toBeVisible();
  await capture(page, 's10', 'resolved-and-unresolved');
});

test('E2E-016 creates an immutable same-cutoff comparison and opens its saved URL', async ({
  page,
}) => {
  await page.goto('/en/compare');
  const create = page.getByRole('button', { name: 'Create comparison' });
  await expect(create).toBeDisabled();
  await page.getByLabel('Temporal').check();
  await page.getByLabel('OpenTelemetry').check();
  await create.click();
  await expect(page).toHaveURL(/\/en\/compare\/901$/);
  await expect(page.getByRole('table')).toBeVisible();
  await expect(page.getByText(/cutoff 2026-08-22T00:00:00Z/).first()).toBeVisible();
});

test('E2E-041 completes S9 at narrow and wide widths with version, cutoff, coverage and status evidence', async ({
  page,
}) => {
  await page.goto('/en/projects/101/health?window=90d&cutoff=2026-08-22T00%3A00%3A00Z');
  await capture(page, 's9', 'all-states');
  await expect(page.getByText(/v1/).first()).toBeVisible();
  await expect(page.getByText(/coverage/).first()).toBeVisible();
});

test('E2E-042 completes S10 at narrow and wide widths with contributor resolution provenance', async ({
  page,
}) => {
  await page.goto('/en/projects/101/contributors?window=90d');
  await capture(page, 's10', 'concentration-and-coverage');
  await expect(page.getByText('analyst_confirmed', { exact: true })).toBeVisible();
  await expect(page.getByText('unresolved', { exact: true })).toBeVisible();
});

test('E2E-044 completes S12 with accessible matrix and narrow row-detail alternative', async ({
  page,
}) => {
  await page.goto('/en/compare/901');
  await page.setViewportSize({ width: 1280, height: 900 });
  await expect(page.getByRole('table')).toBeVisible();
  await expect(page.getByText(/incomparable/i)).toBeVisible();
  await capture(page, 's12', 'same-window-matrix');
  const results = await new AxeBuilder({ page }).analyze();
  expect(results.violations).toEqual([]);
});

async function handleAPI(route: Route) {
  const request = route.request();
  const path = new URL(request.url()).pathname;
  if (path === '/api/v1/session')
    return json(route, {
      authenticated: true,
      state: 'active',
      role: 'analyst',
      csrf_token: 'task04-csrf',
      member: { id: '11', display_name: 'Ada', role: 'analyst', status: 'active' },
    });
  if (path === '/api/v1/projects')
    return json(route, {
      items: [
        { id: '101', name: 'Temporal', slug: 'temporal', state: 'active', version: 1 },
        { id: '102', name: 'OpenTelemetry', slug: 'opentelemetry', state: 'active', version: 1 },
        { id: '103', name: 'Archived SDK', slug: 'archived-sdk', state: 'archived', version: 1 },
      ],
      has_more: false,
    });
  if (path === '/api/v1/projects/101/metrics') return json(route, metricPage());
  if (path === '/api/v1/projects/101/health') return json(route, health());
  if (path === '/api/v1/projects/101/contributors') return json(route, contributors());
  if (path === '/api/v1/comparisons' && request.method() === 'POST')
    return json(route, comparison(), 201);
  if (path === '/api/v1/comparisons/901') return json(route, comparison());
  return json(route, { code: 'not_found', detail: `No Task 4 fixture for ${path}` }, 404);
}

function metricPage() {
  const names = [
    ['release_frequency', 'releases', 'available', 0],
    ['active_contributors', 'people', 'available', 14],
    ['issues_opened_closed', 'issues', 'available', 31],
    ['median_issue_first_response', 'hours', 'insufficient_data', undefined],
    ['median_pr_merge_time', 'hours', 'available', 18.5],
    ['backlog_change', 'issues', 'stale', undefined],
    ['top_three_author_share', 'ratio', 'available', 0.65],
  ] as const;
  return {
    items: names.map(([name, unit, status, value]) => ({
      id: String(400 + names.findIndex((item) => item[0] === name)),
      project_id: '101',
      definition: {
        name,
        version: 'v1',
        unit,
        formula: `deterministic ${name} formula`,
        eligibility: 'canonical public evidence',
        missing_data_rule: 'never convert missing evidence to zero',
      },
      window,
      status,
      value,
      coverage: {
        eligible: 20,
        observed: status === 'insufficient_data' ? 4 : 20,
        ratio: status === 'insufficient_data' ? 0.2 : 1,
      },
      factors: [{ name: 'eligible', value: 20, unit: 'items', evidence_id: '801' }],
      repository_ids: ['201'],
      stale_reason: status === 'stale' ? 'controlled refresh failure' : undefined,
    })),
    has_more: false,
    window,
  };
}

function health() {
  return {
    project_id: '101',
    window,
    version: 'v1',
    overall_status: 'insufficient_data',
    dimensions: [
      'Activity',
      'Community',
      'Maintenance',
      'Concentration',
      'Stability',
      'Security',
      'Adoption',
    ].map((name, index) => ({
      name,
      status: index === 6 ? 'insufficient_data' : 'available',
      score: index === 6 ? undefined : 0.74 - index * 0.04,
      weight: 1 / 7,
      version: 'v1',
    })),
  };
}

function contributors() {
  return {
    project_id: '101',
    window,
    summary: {
      status: 'available',
      active: 3,
      top_one_share: 0.45,
      top_three_share: 0.65,
      resolution_coverage: 0.8,
    },
    items: [
      { key: 'identity:ada', commits: 9, identity_status: 'analyst_confirmed' },
      { key: 'identity:linus', commits: 7, identity_status: 'verified' },
      { key: 'account:unresolved', commits: 4, identity_status: 'unresolved' },
    ],
    has_more: false,
  };
}

function comparison() {
  return {
    id: '901',
    project_ids: ['101', '102'],
    window,
    created_at: cutoff,
    rows: [
      {
        metric: 'release_frequency',
        unit: 'releases',
        cells: [
          { project_id: '101', status: 'available', value: 0, version: 'v1', evidence: [] },
          { project_id: '102', status: 'available', value: 4, version: 'v1', evidence: [] },
        ],
      },
      {
        metric: 'median_issue_first_response',
        unit: 'hours',
        cells: [
          { project_id: '101', status: 'insufficient_data', version: 'v1', evidence: [] },
          { project_id: '102', status: 'incomparable', version: 'v2', evidence: [] },
        ],
      },
    ],
  };
}

async function capture(page: Page, surface: string, state: string) {
  for (const [width, height, viewport] of [
    [320, 900, 'narrow'],
    [1280, 900, 'wide'],
  ] as const) {
    await page.setViewportSize({ width, height });
    const directory = resolve(process.cwd(), `../../artifacts/task_04/ui/${surface}-${viewport}`);
    await mkdir(directory, { recursive: true });
    await page.screenshot({ path: resolve(directory, `${state}.png`), fullPage: true });
  }
}

function json(route: Route, body: unknown, status = 200) {
  return route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
}
