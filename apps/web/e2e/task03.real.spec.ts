import { expect, test } from '@playwright/test';

const bearer = 'task03-browser-admin';

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/**', (route) =>
    route.continue({
      headers: { ...route.request().headers(), authorization: `Bearer ${bearer}` },
    }),
  );
});

test('Task 3 real API journey persists generated-client registration across reload', async ({
  page,
}) => {
  await page.goto('/en/projects');
  await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible();
  await page
    .getByLabel('Public repository URL')
    .fill('https://github.com/open-telemetry/opentelemetry-go');
  await page.getByRole('button', { name: 'Register project' }).click();

  await expect(page).toHaveURL(/\/en\/projects\/\d+$/);
  await expect(page.getByRole('heading', { name: 'opentelemetry-go' })).toBeVisible();
  const persistedURL = page.url();

  await page.reload();
  await expect(page).toHaveURL(persistedURL);
  await expect(page.getByRole('heading', { name: 'opentelemetry-go' })).toBeVisible();

  await page.goto(`${persistedURL}/jobs`);
  await expect(page.getByText(/project_initial_sync/)).toBeVisible();

  await page.goto('/en/portfolio');
  await expect(page.getByRole('heading', { name: 'opentelemetry-go' })).toBeVisible();
});
