// @vitest-environment jsdom

import '@testing-library/jest-dom/vitest';

import { QueryClientProvider } from '@tanstack/react-query';
import { act, cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Outlet, Route, Routes } from 'react-router';

import { AppShell, Button } from '../design-system';
import { queryClient } from './query';
import type { ApplicationContext } from './router';
import { AccountScreen } from './screens/AccountScreen';
import i18n, { ptBR } from './i18n';

const session: ApplicationContext['session'] = {
  authenticated: true,
  state: 'active',
  role: 'viewer',
  csrf_token: 'csrf-test',
  member: {
    id: '41',
    display_name: 'Ada',
    locale: 'en',
    timezone: 'UTC',
    version: 3,
  },
};

beforeEach(async () => {
  window.sessionStorage.clear();
  queryClient.clear();
  await i18n.changeLanguage('en');
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe('task 02 localized application integration', () => {
  it('IT-091 changes language during a save without repeating or losing the action', async () => {
    const pending = deferred<Response>();
    const fetchMock = vi.fn(() => pending.promise);
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    renderAccount();

    await user.selectOptions(screen.getByLabelText('Language'), 'pt-BR');
    await user.click(screen.getByRole('button', { name: 'Save preferences' }));
    await act(() => i18n.changeLanguage('pt-BR'));
    pending.resolve(jsonResponse({ ...session.member, locale: 'pt-BR', version: 4 }));

    expect(await screen.findByText(ptBR.saved)).toBeVisible();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('IT-092 preserves a safe account draft across navigation and connection loss', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.reject(new TypeError('offline'))),
    );
    const user = userEvent.setup();
    const first = renderAccount();

    const timezone = screen.getByLabelText('Timezone');
    await user.clear(timezone);
    await user.type(timezone, 'America/Sao_Paulo');
    await user.click(screen.getByRole('button', { name: 'Save preferences' }));
    expect(await screen.findByText('This section could not be loaded.')).toBeVisible();

    first.unmount();
    renderAccount();
    expect(screen.getByLabelText('Timezone')).toHaveValue('America/Sao_Paulo');
  });

  it('IT-093 keeps essential actions named with large localized strings at 200% scale', () => {
    render(
      <div style={{ width: 320, fontSize: '200%' }}>
        <AppShell
          viewport="mobile"
          nav={[{ key: 'catalog', label: ptBR.catalogTitle, icon: 'search' }]}
          activeKey="catalog"
          onNavigate={() => undefined}
          locale="pt-BR"
          skipLabel={ptBR.skip}
          primaryNavigationLabel={ptBR.primaryNavigation}
          utilities={<Button>{ptBR.deleteAction}</Button>}
        >
          <h1>{ptBR.preferences}</h1>
        </AppShell>
      </div>,
    );

    expect(screen.getByRole('button', { name: ptBR.deleteAction })).toBeVisible();
    expect(screen.getByRole('heading', { name: ptBR.preferences })).toBeVisible();
    expect(screen.getByRole('navigation', { name: ptBR.primaryNavigation })).toBeVisible();
  });
});

function renderAccount() {
  const context: ApplicationContext = { locale: 'en', session, narrow: false };
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/en/account']}>
        <Routes>
          <Route element={<ContextOutlet value={context} />}>
            <Route path="/en/account" element={<AccountScreen />} />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function ContextOutlet({ value }: { value: ApplicationContext }) {
  return <Outlet context={value} />;
}

function jsonResponse(value: unknown): Response {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}
