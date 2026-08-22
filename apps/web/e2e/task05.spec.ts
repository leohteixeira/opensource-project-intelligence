import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Page, type Route } from '@playwright/test';
import { mkdir } from 'node:fs/promises';
import { resolve } from 'node:path';

const cutoff = '2026-08-22T00:00:00Z';

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/**', handleAPI);
});

test('E2E-015 interprets adoption and qualified public security evidence without a universal score', async ({
  page,
}) => {
  await page.goto('/en/projects/101/adoption-security', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Adoption and security' })).toBeVisible();
  await expect(page.getByText('weekly_downloads')).toBeVisible();
  await expect(page.getByText(/npm_public/)).toBeVisible();
  await expect(page.getByText('ADV-2026-1')).toBeVisible();
  await expect(page.getByText('This is not a vulnerability scan')).toBeVisible();
});

test('E2E-021 explores deterministic topics and appends an attributed Analyst correction', async ({
  page,
}) => {
  await page.goto('/en/projects/101/topics', { waitUntil: 'domcontentloaded' });
  await expect(page.getByText('mutual-knn-v1')).toBeVisible();
  await expect(page.getByText('2 canonical members')).toBeVisible();
  await page.getByLabel('Canonical label').fill('Canonical upgrade maintenance');
  await page.getByRole('button', { name: 'Save correction' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.getByText('Correction recorded')).toBeVisible();
});

test('E2E-022 opens release evidence while model analysis remains unavailable', async ({
  page,
}) => {
  await page.goto('/en/projects/101/releases', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Release intelligence' })).toBeVisible();
  await expect(page.getByText(/v1.2.3/).first()).toBeVisible();
  await expect(page.getByText(/analysis unavailable/)).toBeVisible();
  await expect(page.getByText(/Evidence 5501/)).toBeVisible();
});

test('E2E-023 searches immutable documentation snapshots with source and offset citations', async ({
  page,
}) => {
  await page.goto('/en/projects/101/knowledge', { waitUntil: 'domcontentloaded' });
  await page.getByRole('button', { name: 'Search snapshots' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('heading', { name: 'Upgrade safely' })).toBeVisible();
  await expect(page.getByText(/snapshot 5501/)).toBeVisible();
  await expect(page.getByText(/offsets 0–48/)).toBeVisible();
});

test('E2E-024 asks a bounded natural-language question and receives an immutable queued run', async ({
  page,
}) => {
  await page.goto('/en/projects/101/knowledge', { waitUntil: 'domcontentloaded' });
  await page.getByRole('button', { name: 'Ask with citations' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.getByText('Analysis queued')).toBeVisible();
  await expect(page.getByText(/cutoff 2026-08-22/)).toBeVisible();
  await expect(page.getByText(/rrf-v1/)).toBeVisible();
});

test('E2E-026 reviews immutable AI output and appends rerun feedback and selection history', async ({
  page,
}) => {
  await page.goto('/en/analysis-runs/5901', { waitUntil: 'domcontentloaded' });
  await expect(page.getByText('prompt-release-v1')).toBeVisible();
  await expect(page.getByText('snapshot 5501')).toBeVisible();
  for (const name of ['Create rerun', 'Flag evidence', 'Select this version']) {
    await page.getByRole('button', { name }).focus();
    await page.keyboard.press('Enter');
  }
  await expect(page.getByText('New immutable version queued')).toBeVisible();
  await expect(page.getByText('Feedback recorded')).toBeVisible();
  await expect(page.getByText('Presented version selected')).toBeVisible();
});

test('E2E-043 completes S11 in both locales at narrow and wide viewports', async ({ page }) => {
  await verifySurface(page, {
    enPath: '/en/projects/101/adoption-security',
    ptPath: '/pt-br/projetos/101/adocao-seguranca',
    enHeading: 'Adoption and security',
    ptHeading: 'Adoção e segurança',
    surface: 's11',
    state: 'qualified-public-evidence',
  });
});

test('E2E-048 completes S16 in both locales at narrow and wide viewports', async ({ page }) => {
  await verifySurface(page, {
    enPath: '/en/projects/101/topics',
    ptPath: '/pt-br/projetos/101/topicos',
    enHeading: 'Issue and discussion topics',
    ptHeading: 'Tópicos de issues e discussões',
    surface: 's16',
    state: 'candidate-and-correction',
  });
});

test('E2E-049 completes S17 in both locales at narrow and wide viewports', async ({ page }) => {
  await verifySurface(page, {
    enPath: '/en/projects/101/releases',
    ptPath: '/pt-br/projetos/101/versoes',
    enHeading: 'Release intelligence',
    ptHeading: 'Inteligência de versões',
    surface: 's17',
    state: 'metadata-with-model-degradation',
  });
});

test('E2E-050 completes S18 in both locales at narrow and wide viewports', async ({ page }) => {
  await verifySurface(page, {
    enPath: '/en/projects/101/knowledge',
    ptPath: '/pt-br/projetos/101/conhecimento',
    enHeading: 'Documentation knowledge',
    ptHeading: 'Conhecimento da documentação',
    surface: 's18',
    state: 'authorized-cited-search',
  });
});

test('E2E-052 completes S20 in both locales at narrow and wide viewports', async ({ page }) => {
  await verifySurface(page, {
    enPath: '/en/analysis-runs/5901',
    ptPath: '/pt-br/execucoes-ia/5901',
    enHeading: 'AI run governance',
    ptHeading: 'Governança de execuções de IA',
    surface: 's20',
    state: 'succeeded-immutable-run',
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

async function handleAPI(route: Route) {
  const request = route.request();
  const path = new URL(request.url()).pathname;
  if (path === '/api/v1/session') return json(route, session());
  if (path === '/api/v1/projects') return json(route, projects());
  if (path === '/api/v1/projects/101/adoption') return json(route, adoption());
  if (path === '/api/v1/projects/101/security') return json(route, security());
  if (path === '/api/v1/projects/101/topics' && request.method() === 'GET')
    return json(route, topics());
  if (/^\/api\/v1\/projects\/101\/topics\/5702\/corrections$/.test(path))
    return json(route, { id: '5799', version: 1, action: 'rename' }, 202);
  if (path === '/api/v1/projects/101/releases') return json(route, releases());
  if (path === '/api/v1/projects/101/knowledge/search') return json(route, knowledge());
  if (path === '/api/v1/projects/101/queries') return json(route, queuedRun(), 202);
  if (path === '/api/v1/analysis-runs/5901' && request.method() === 'GET')
    return json(route, analysisRun());
  if (path === '/api/v1/analysis-runs/5901/reruns')
    return json(route, { ...queuedRun(), id: '5902', parent_run_id: '5901' }, 202);
  if (path === '/api/v1/analysis-runs/5901/feedback')
    return json(route, { id: '5991', run_id: '5901', rating: 'not_faithful' }, 201);
  if (path === '/api/v1/analysis-series/5900/selection')
    return json(route, { id: '5992', run_id: '5901', series_id: '5900', version: 1 });
  return json(route, { code: 'not_found', detail: `No Task 5 fixture for ${path}` }, 404);
}

function session() {
  return {
    authenticated: true,
    state: 'active',
    role: 'analyst',
    csrf_token: 'task05-csrf',
    member: { id: '11', display_name: 'Ada', role: 'analyst', status: 'active' },
  };
}

function projects() {
  return {
    items: [{ id: '101', name: 'Temporal', slug: 'temporal', state: 'active', version: 1 }],
    has_more: false,
  };
}

function adoption() {
  return {
    items: [
      {
        id: '5301',
        project_id: '101',
        source_id: '5112',
        registry: 'npm',
        package: 'temporal-sdk',
        unit: 'weekly_downloads',
        population: 'npm_public',
        value: 120000,
        status: 'available',
        window_from: '2026-05-24T00:00:00Z',
        window_to: cutoff,
        observed_at: cutoff,
        evidence_id: '5201',
      },
      {
        id: '5302',
        project_id: '101',
        source_id: '5113',
        registry: 'crates.io',
        package: 'temporal-sdk',
        unit: 'dependents',
        population: 'crates_public',
        status: 'incomparable',
        window_from: '2026-05-24T00:00:00Z',
        window_to: cutoff,
        observed_at: cutoff,
        evidence_id: '5202',
      },
    ],
    has_more: false,
  };
}

function security() {
  return {
    security: {
      status: 'available',
      qualification: 'public_advisory_evidence_only',
      coverage_note: 'public advisories observed; withdrawn records remain visible',
      advisories: [
        {
          id: '5401',
          external_id: 'ADV-2026-1',
          severity: 'high',
          summary: 'Historical dependency advisory',
          state: 'withdrawn',
          published_at: '2026-07-01T00:00:00Z',
          withdrawn_at: '2026-07-10T00:00:00Z',
          evidence_id: '5203',
        },
      ],
    },
    has_more: false,
  };
}

function topics() {
  return {
    items: [
      {
        candidate: {
          id: '5702',
          project_id: '101',
          algorithm_version: 'mutual-knn-v1',
          generated_label: 'Generated maintenance',
          created_at: cutoff,
        },
        label: 'Upgrade maintenance',
        members: ['5601', '5602'],
        history: [],
      },
    ],
    has_more: false,
  };
}

function releases() {
  return {
    items: [
      {
        id: '5801',
        project_id: '101',
        tag: 'v1.2.3',
        title: 'Stable release',
        body: 'Upgrade and maintenance fixes backed by the changelog snapshot.',
        language: 'en',
        state: 'published',
        published_at: '2026-08-20T00:00:00Z',
        evidence_id: '5501',
      },
    ],
    has_more: false,
  };
}

function knowledge() {
  return {
    items: [
      {
        chunk: {
          id: '5601',
          project_id: '101',
          source_id: '5114',
          snapshot_id: '5501',
          heading: 'Upgrade safely',
          text: 'Upgrade safely using the documented migration path.',
          language: 'en',
          start_offset: 0,
          end_offset: 48,
          parser_version: 'parser-v1',
          observed_at: cutoff,
          current: true,
        },
        score: 0.0327,
        modes: ['lexical', 'vector'],
      },
    ],
    retrieval_version: 'rrf-v1',
    modes: ['lexical', 'vector'],
  };
}

function queuedRun() {
  return {
    id: '5903',
    series_id: '5904',
    project_id: '101',
    kind: 'natural_language_query',
    state: 'queued',
    prompt_version: 'query-v1',
    schema_version: 'analysis-v1',
    retrieval_version: 'rrf-v1',
    cutoff,
    created_at: cutoff,
    evidence: [],
    usage: {},
  };
}

function analysisRun() {
  return {
    id: '5901',
    series_id: '5900',
    project_id: '101',
    kind: 'release',
    state: 'succeeded',
    prompt_version: 'prompt-release-v1',
    schema_version: 'analysis-v1',
    retrieval_version: 'rrf-v1',
    provider: 'fixture',
    model: 'fixture-v1',
    cutoff,
    created_at: '2026-08-22T00:01:00Z',
    finished_at: '2026-08-22T00:01:02Z',
    output: { summary: 'Supported release', claims: [{ text: 'Upgrade', citations: [] }] },
    evidence: [{ snapshot_id: '5501', chunk_id: '5601', start_offset: 0, end_offset: 48 }],
    usage: { input_tokens: 20, output_tokens: 10, cost: 0.01, currency: 'USD' },
    selection_version: 0,
  };
}

async function capture(page: Page, surface: string, state: string) {
  for (const [width, height, viewport] of [
    [320, 900, 'narrow'],
    [1280, 900, 'wide'],
  ] as const) {
    await page.setViewportSize({ width, height });
    if (viewport === 'narrow') {
      // A sticky bottom bar is correct in the live viewport, but Playwright's full-page stitching
      // would otherwise paint it over content from a later scroll segment in the review artifact.
      await page
        .getByRole('navigation', { name: 'Primary navigation' })
        .evaluate((navigation) => navigation.style.setProperty('position', 'static'));
    }
    const directory = resolve(process.cwd(), `../../artifacts/task_05/ui/${surface}-${viewport}`);
    await mkdir(directory, { recursive: true });
    await page.screenshot({ path: resolve(directory, `${state}.png`), fullPage: true });
  }
}

function json(route: Route, body: unknown, status = 200) {
  return route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
}
