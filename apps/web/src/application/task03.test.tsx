// @vitest-environment jsdom

import '@testing-library/jest-dom/vitest';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Outlet, Route, Routes } from 'react-router';

import type { ApplicationContext } from './router';
import { routePath, switchLocalePath } from './routes';
import { PortfolioScreen, ProjectsScreen, WorkspaceProjectScreen } from './screens/WorkspaceScreen';

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const project = {
  id: '101',
  name: 'Temporal',
  slug: 'temporal',
  description: 'Durable execution',
  state: 'active',
  version: 4,
};

describe('Task 3 application boundary', () => {
  it('keeps the Portfolio page identity available while data is loading', () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => new Promise(() => undefined)),
    );
    renderScreen(
      <PortfolioScreen />,
      '/en/portfolio',
      '/en/portfolio',
      viewer('en'),
      testClient(0),
    );

    expect(screen.getByRole('heading', { level: 1, name: 'Portfolio' })).toBeVisible();
  });

  it('UT-032 renders intelligence for viewers without Project mutation controls', () => {
    const client = testClient();
    client.setQueryData(['projects', 'active', '', undefined], {
      items: [project],
      has_more: false,
    });
    renderScreen(<ProjectsScreen />, '/en/projects', '/en/projects', viewer('en'), client);

    expect(screen.getByRole('heading', { name: 'Temporal' })).toBeVisible();
    expect(screen.getByRole('searchbox')).toBeVisible();
    expect(screen.queryByLabelText('Public repository URL')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Register project' })).not.toBeInTheDocument();
  });

  it('UT-034 resolves every Task 3 deep link without overview-first state', () => {
    expect(routePath('en', 'workspaceProject', { projectId: '101' })).toBe('/en/projects/101');
    expect(routePath('en', 'projectSources', { projectId: '101' })).toBe(
      '/en/projects/101/sources',
    );
    expect(routePath('en', 'projectJobs', { projectId: '101' })).toBe('/en/projects/101/jobs');
    expect(routePath('en', 'projectLifecycle', { projectId: '101' })).toBe(
      '/en/projects/101/lifecycle',
    );
  });

  it('IT-014 retains deterministic portfolio panels when a background refresh fails', async () => {
    const client = testClient(0);
    client.setQueryData(['portfolio'], {
      projects: [project],
      active_jobs: [],
      attention_count: 0,
      generated_at: '2026-08-22T03:00:00Z',
    });
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('controlled panel failure')));
    renderScreen(<PortfolioScreen />, '/en/portfolio', '/en/portfolio', viewer('en'), client);

    expect(screen.getByRole('heading', { name: 'Temporal' })).toBeVisible();
    await waitFor(() =>
      expect(screen.getByText('The request could not be completed')).toBeVisible(),
    );
    expect(screen.getByRole('heading', { name: 'Temporal' })).toBeVisible();
    expect(screen.getByRole('heading', { name: 'Jobs' })).toBeVisible();
  });

  it('UT-270 IT-145 keeps localized route and action identity without raw keys', () => {
    const client = testClient();
    client.setQueryData(['projects', 'active', '', undefined], {
      items: [project],
      has_more: false,
    });
    renderScreen(<ProjectsScreen />, '/pt-br/projetos', '/pt-br/projetos', viewer('pt-BR'), client);

    expect(screen.getByRole('heading', { name: 'Projetos' })).toBeVisible();
    expect(screen.getByRole('heading', { name: 'Temporal' })).toBeVisible();
    expect(switchLocalePath('/en/projects/101/jobs', 'en', 'pt-BR')).toBe(
      '/pt-br/projetos/101/tarefas',
    );
    expect(document.body.textContent).not.toMatch(/workspaceProject|projectJobs|noProjectsBody/);
  });

  it('IT-146 exposes named lifecycle navigation, confirmation, and disabled destructive state', () => {
    const client = testClient();
    client.setQueryData(['workspace-project', '101'], project);
    client.setQueryData(['workspace-project-resources', '101'], {
      repositories: { items: [], has_more: false },
      sources: { items: [], has_more: false },
      associations: { items: [], has_more: false },
      jobs: { items: [], has_more: false },
    });
    renderScreen(
      <WorkspaceProjectScreen section="lifecycle" />,
      '/en/projects/101/lifecycle',
      '/en/projects/:projectId/lifecycle',
      admin('en'),
      client,
    );

    expect(screen.getByRole('navigation', { name: 'Project sections' })).toBeVisible();
    expect(screen.getByLabelText('Type DELETE temporal')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Permanently delete' })).toBeDisabled();
    expect(screen.getByRole('heading', { name: 'Temporal' })).toBeVisible();
  });
});

function testClient(staleTime = Number.POSITIVE_INFINITY) {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime }, mutations: { retry: false } },
  });
}

function viewer(locale: 'en' | 'pt-BR'): ApplicationContext {
  return {
    locale,
    narrow: false,
    session: { authenticated: true, state: 'active', role: 'viewer' },
  };
}

function admin(locale: 'en' | 'pt-BR'): ApplicationContext {
  return {
    locale,
    narrow: false,
    session: { authenticated: true, state: 'active', role: 'admin', csrf_token: 'test' },
  };
}

function renderScreen(
  element: ReactNode,
  path: string,
  route: string,
  context: ApplicationContext,
  client: QueryClient,
) {
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route element={<Outlet context={context} />}>
            <Route path={route} element={element} />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}
