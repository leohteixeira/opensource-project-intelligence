import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list']],
  use: {
    baseURL: 'http://127.0.0.1:3100',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: [
    {
      command:
        'cd ../.. && mkdir -p .cache/go-build .cache/go-tmp && GOCACHE="$PWD/.cache/go-build" GOTMPDIR="$PWD/.cache/go-tmp" OPI_INTEGRATION_DATABASE_URL="postgres://opensource:opensource@127.0.0.1:5433/opensource_project_intelligence?sslmode=disable" go test -tags=e2e ./internal/platform/projectapi -run TestTask03E2EBackend -count=1',
      url: 'http://127.0.0.1:8100/api/v1/session',
      reuseExistingServer: true,
      timeout: 120_000,
    },
    {
      command: 'pnpm exec vite --host 0.0.0.0 --port 3100',
      url: 'http://127.0.0.1:3100',
      reuseExistingServer: true,
      timeout: 120_000,
    },
  ],
});
