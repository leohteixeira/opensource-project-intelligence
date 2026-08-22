import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Page, type Route } from '@playwright/test';
import { mkdir } from 'node:fs/promises';
import { resolve } from 'node:path';

const cutoff = '2026-08-22T00:00:00Z';

interface FixtureState {
  alertRead: boolean;
  alertState: string;
  alertRevision: number;
  overrideSaved: boolean;
  policyCreated: boolean;
}

test.beforeEach(async ({ page }) => {
  const state: FixtureState = {
    alertRead: false,
    alertState: 'open',
    alertRevision: 1,
    overrideSaved: false,
    policyCreated: false,
  };
  await page.route('**/api/v1/**', (route) => handleAPI(route, state));
});

test('E2E-017 distinguishes deterministic observations from calibrated forecast warnings', async ({
  page,
}) => {
  await page.goto('/en/projects/101/trends', { waitUntil: 'domcontentloaded' });
  await expect(
    page.getByRole('heading', { name: 'Trends and adoption recommendation' }),
  ).toBeVisible();
  await expect(page.getByText('theil-sen-mann-kendall-v1')).toBeVisible();
  await expect(page.getByText('2025-08-22T00:00:00Z – 2026-05-24T00:00:00Z')).toBeVisible();
  await expect(page.getByText('6 immutable records')).toBeVisible();

  await page.getByRole('tab', { name: 'Forecast warnings' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.getByText(/rolling-baseline-v1 · seasonal-baseline/)).toBeVisible();
  await expect(page.getByText('30 days')).toBeVisible();
  await expect(page.getByText(/confidence 0.82/)).toBeVisible();
  await expect(page.getByText('0.14')).toBeVisible();
});

test('E2E-018 exposes one immutable four-state recommendation with factors and gaps', async ({
  page,
}) => {
  await page.goto('/en/projects/101/trends', { waitUntil: 'domcontentloaded' });
  await expect(page.getByText('Conditional', { exact: true })).toBeVisible();
  await expect(page.getByText(/Policy 201 · owner Architecture v3/)).toBeVisible();
  await expect(page.getByText('Release cadence')).toBeVisible();
  await expect(page.getByText('Pilot requires an upgrade rehearsal')).toBeVisible();
  await expect(page.getByText(/2 immutable evidence records/)).toBeVisible();
  await expect(page.getByText(/Generated explanation availability cannot change it/)).toBeVisible();
});

test('E2E-019 clones and edits a typed policy draft without rewriting the source version', async ({
  page,
}) => {
  await page.goto('/en/admin/policies', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Adoption policies' })).toBeVisible();
  await expect(
    page.getByText(/Release cadence: release_frequency gte 4 · weight 0.6/),
  ).toBeVisible();
  await expect(page.getByText(/recommended → adopt/)).toBeVisible();
  await page.getByRole('button', { name: 'New policy draft' }).focus();
  await page.keyboard.press('Enter');
  await page.getByLabel('Policy name').fill('Production adoption 2027');
  await page.getByRole('button', { name: 'Create immutable draft v1' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.getByText('Production adoption 2027')).toBeVisible();
  await expect(page.getByText('Production adoption', { exact: true })).toBeVisible();
});

test('E2E-020 applies an attributed radar override while preserving the policy suggestion', async ({
  page,
}) => {
  await page.goto('/en/radar', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Technology radar' })).toBeVisible();
  await expect(page.getByText(/policy suggests trial/)).toBeVisible();
  await page.getByRole('button', { name: /Project 101/ }).focus();
  await page.keyboard.press('Enter');
  await page.getByLabel('Justification').fill('Pilot dependency is approved');
  await page.getByLabel('Owner').fill('Platform architecture');
  await page.getByLabel('Review date').fill('2026-11-20');
  await page.getByRole('button', { name: 'Save attributed override' }).focus();
  await page.keyboard.press('Enter');
  await expect(
    page.getByText(/policy suggests trial · overridden by Platform architecture/),
  ).toBeVisible();
  await expect(page.getByText('2026-11-20')).toBeVisible();
});

test('E2E-027 keeps personal read state separate from deduplicated shared alert transitions', async ({
  page,
}) => {
  await page.goto('/en/alerts', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Shared alerts' })).toBeVisible();
  await expect(page.getByText('Unread')).toBeVisible();
  await expect(page.getByText('critical')).toBeVisible();
  await expect(page.getByText('7', { exact: true })).toBeVisible();
  await expect(page.getByText('2 immutable records')).toBeVisible();
  await page.getByRole('button', { name: 'Mark read' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('cell', { name: 'Read', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Acknowledge' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('cell', { name: 'acknowledged', exact: true })).toBeVisible();
});

test('E2E-045 completes S13 in both locales at narrow and wide viewports', async ({ page }) => {
  await verifySurface(page, {
    enPath: '/en/projects/101/trends',
    ptPath: '/pt-br/projetos/101/tendencias',
    enHeading: 'Trends and adoption recommendation',
    ptHeading: 'Tendências e recomendação de adoção',
    surface: 's13',
    state: 'observed-and-forecast',
  });
});

test('E2E-046 completes S14 in both locales at narrow and wide viewports', async ({ page }) => {
  await verifySurface(page, {
    enPath: '/en/admin/policies',
    ptPath: '/pt-br/admin/politicas',
    enHeading: 'Adoption policies',
    ptHeading: 'Políticas de adoção',
    surface: 's14',
    state: 'immutable-active-policy',
  });
});

test('E2E-047 completes S15 in both locales at narrow and wide viewports', async ({ page }) => {
  await verifySurface(page, {
    enPath: '/en/radar',
    ptPath: '/pt-br/radar',
    enHeading: 'Technology radar',
    ptHeading: 'Radar de tecnologia',
    surface: 's15',
    state: 'suggestion-and-override',
  });
});

test('E2E-053 completes S21 in both locales at narrow and wide viewports', async ({ page }) => {
  await verifySurface(page, {
    enPath: '/en/alerts',
    ptPath: '/pt-br/alertas',
    enHeading: 'Shared alerts',
    ptHeading: 'Alertas compartilhados',
    surface: 's21',
    state: 'deduplicated-unread-alert',
  });
});

interface SurfaceExpectation {
  enPath: string;
  ptPath: string;
  enHeading: string;
  ptHeading: string;
  surface: string;
  state: string;
}

async function verifySurface(page: Page, expected: SurfaceExpectation) {
  await page.goto(expected.enPath, { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: expected.enHeading })).toBeVisible();
  await capture(page, expected.surface, expected.state);
  const results = await new AxeBuilder({ page }).analyze();
  expect(results.violations).toEqual([]);
  await page.goto(expected.ptPath, { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: expected.ptHeading })).toBeVisible();
}

async function handleAPI(route: Route, state: FixtureState) {
  const request = route.request();
  const url = new URL(request.url());
  const path = url.pathname;
  if (path === '/api/v1/session') return json(route, session());
  if (path === '/api/v1/projects/101/trends') {
    return json(route, url.searchParams.get('kind') === 'forecast' ? forecasts() : observations());
  }
  if (path === '/api/v1/projects/101/recommendation') return json(route, recommendation());
  if (path === '/api/v1/policies' && request.method() === 'GET')
    return json(route, policies(state));
  if (path === '/api/v1/policies' && request.method() === 'POST') {
    state.policyCreated = true;
    const body = request.postDataJSON() as Record<string, unknown>;
    return json(route, { id: '299', policy_id: '299', version: 1, state: 'draft', ...body }, 201);
  }
  if (path === '/api/v1/radar' && request.method() === 'GET') return json(route, radar(state));
  if (path === '/api/v1/radar/101/override' && request.method() === 'POST') {
    state.overrideSaved = true;
    return json(
      route,
      {
        id: '401',
        ring: 'assess',
        reason: 'Pilot dependency is approved',
        owner: 'Platform architecture',
        actor_id: '11',
        review_on: '2026-11-20',
        revision: 1,
      },
      201,
    );
  }
  if (path === '/api/v1/alerts' && request.method() === 'GET') return json(route, alerts(state));
  if (path === '/api/v1/alerts/501/read' && request.method() === 'POST') {
    state.alertRead = true;
    return route.fulfill({ status: 204 });
  }
  if (path === '/api/v1/alerts/501/transition' && request.method() === 'POST') {
    state.alertState = 'acknowledged';
    state.alertRevision += 1;
    return json(route, { ...alertItem(state), revision: state.alertRevision });
  }
  return json(route, { code: 'not_found', detail: `No Task 6 fixture for ${path}` }, 404);
}

function session() {
  return {
    authenticated: true,
    state: 'active',
    role: 'admin',
    csrf_token: 'task06-csrf',
    member: { id: '11', display_name: 'Ada', role: 'admin', status: 'active' },
  };
}

function observations() {
  return {
    kind: 'observed',
    items: [
      {
        id: '10101',
        project_id: '101',
        metric_name: 'release_frequency',
        metric_version: 'v1',
        kind: 'observed',
        status: 'increase',
        method_version: 'theil-sen-mann-kendall-v1',
        window_from: '2026-05-24T00:00:00Z',
        window_to: cutoff,
        baseline_from: '2025-08-22T00:00:00Z',
        baseline_to: '2026-05-24T00:00:00Z',
        cutoff,
        magnitude: 0.18,
        coverage: { eligible: 12, observed: 12 },
        evidence_ids: ['6101', '6102', '6103', '6104', '6105', '6106'],
      },
    ],
    has_more: false,
  };
}

function forecasts() {
  return {
    kind: 'forecast',
    items: [
      {
        id: '10102',
        project_id: '101',
        metric_name: 'median_issue_first_response',
        metric_version: 'v1',
        kind: 'forecast',
        status: 'increase',
        method_version: 'rolling-baseline-v1',
        selected_model: 'seasonal-baseline',
        window_from: '2025-08-22T00:00:00Z',
        window_to: cutoff,
        cutoff,
        horizon_days: 30,
        predicted: 19.4,
        interval_low: 16.1,
        interval_high: 23.8,
        confidence: 0.82,
        backtest_error: 0.14,
        outcome_status: 'unevaluated',
        coverage: { eligible: 52, observed: 49 },
        evidence_ids: ['6201', '6202'],
      },
    ],
    has_more: false,
  };
}

function recommendation() {
  return {
    id: '301',
    project_id: '101',
    policy_id: '201',
    policy_version: 3,
    policy_owner: 'Architecture',
    outcome: 'conditional',
    window: { from: '2026-05-24T00:00:00Z', to: cutoff, cutoff },
    conditions: ['Pilot requires an upgrade rehearsal'],
    missing_data: [],
    stale_inputs: [],
    decisive: ['Release cadence'],
    factors: [
      {
        metric_name: 'release_frequency',
        threshold: 4,
        weight: 0.6,
        value: 3.2,
        matched: false,
        label: 'Release cadence',
      },
    ],
    evidence_ids: ['6101', '6102'],
  };
}

function policyItem(id: string, name: string, state: string) {
  return {
    id,
    policy_id: id,
    name,
    description: 'Explicit production adoption rules',
    owner: 'Architecture',
    version: 3,
    state,
    revision: 2,
    rules: [
      {
        metric_name: 'release_frequency',
        metric_version: 'v1',
        operator: 'gte',
        threshold: 4,
        weight: 0.6,
        required: true,
        required_evidence: 'release facts',
        on_failure: 'conditional',
        label: 'Release cadence',
      },
      {
        metric_name: 'active_contributors',
        metric_version: 'v1',
        operator: 'gte',
        threshold: 5,
        weight: 0.4,
        required: true,
        required_evidence: 'contributor facts',
        on_failure: 'not_recommended',
        label: 'Contributor continuity',
      },
    ],
    radar_mapping: {
      recommended: 'adopt',
      conditional: 'trial',
      not_recommended: 'hold',
      insufficient_data: 'unplaced',
    },
  };
}

function policies(state: FixtureState) {
  return {
    items: [
      policyItem('201', 'Production adoption', 'active'),
      ...(state.policyCreated ? [policyItem('299', 'Production adoption 2027', 'draft')] : []),
    ],
    has_more: false,
  };
}

function radar(state: FixtureState) {
  return {
    items: [
      {
        project_id: '101',
        project_state: 'active',
        suggested_ring: 'trial',
        effective_ring: state.overrideSaved ? 'assess' : 'trial',
        override: state.overrideSaved
          ? {
              id: '401',
              ring: 'assess',
              reason: 'Pilot dependency is approved',
              owner: 'Platform architecture',
              actor_id: '11',
              review_on: '2026-11-20',
              revision: 1,
            }
          : undefined,
        recommendation: recommendation(),
      },
    ],
    count: 1,
  };
}

function alertItem(state: FixtureState) {
  return {
    id: '501',
    rule_id: '401',
    rule_version: 2,
    project_id: '101',
    signal_version: 'metric-v1',
    severity: 'critical',
    state: state.alertState,
    window_from: '2026-08-21T00:00:00Z',
    window_to: cutoff,
    first_detected_at: '2026-08-22T00:05:00Z',
    last_detected_at: '2026-08-22T00:35:00Z',
    evidence_ids: ['7101', '7102'],
    suppression_count: 7,
    revision: state.alertRevision,
    read_at: state.alertRead ? '2026-08-22T01:00:00Z' : undefined,
  };
}

function alerts(state: FixtureState) {
  return { items: [alertItem(state)], has_more: false };
}

async function capture(page: Page, surface: string, state: string) {
  for (const [width, height, viewport] of [
    [320, 1000, 'narrow'],
    [1280, 1000, 'wide'],
  ] as const) {
    await page.setViewportSize({ width, height });
    if (viewport === 'narrow') {
      await page
        .getByRole('navigation', { name: 'Primary navigation' })
        .evaluate((navigation) => navigation.style.setProperty('position', 'static'));
    }
    const directory = resolve(process.cwd(), `../../artifacts/task_06/ui/${surface}-${viewport}`);
    await mkdir(directory, { recursive: true });
    await page.screenshot({ path: resolve(directory, `${state}.png`), fullPage: true });
  }
}

function json(route: Route, body: unknown, status = 200) {
  return route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
}
