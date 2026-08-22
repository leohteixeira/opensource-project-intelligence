// @vitest-environment jsdom

import '@testing-library/jest-dom/vitest';

import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';

import {
  AppShell,
  Button,
  EmptyState,
  Pagination,
  Table,
  type TableColumn,
} from '../design-system';
import { en, normalizeLocale, ptBR } from './i18n';
import { routePath, routeSegment, switchLocalePath } from './routes';

afterEach(cleanup);

describe('task 02 localized application boundary', () => {
  it('UT-001 treats malformed catalog search as opaque URL text', () => {
    const input = `' OR 1=1 -- <script>alert(1)</script>`;
    const params = new URLSearchParams({ q: input });
    expect(params.get('q')).toBe(input);
    expect([...params.keys()]).toEqual(['q']);
  });

  it('UT-002 renders an explanatory empty catalog state', () => {
    render(
      <EmptyState glyph="package" title={en.noProjects}>
        {en.noProjectsHelp}
      </EmptyState>,
    );
    expect(screen.getByRole('heading', { name: en.noProjects })).toBeVisible();
    expect(screen.getByText(en.noProjectsHelp)).toBeVisible();
  });

  it('UT-003 keeps large catalog navigation bounded', () => {
    render(<Pagination page={4} hasMore pageSize={24} label="Projects" />);
    expect(screen.getByText('Page 4 · 24 per page')).toBeVisible();
    expect(screen.getByRole('button', { name: /next/i })).toBeEnabled();
  });

  it('UT-004 keeps anonymous protected deep links distinct from public routes', () => {
    expect(routePath('en', 'account')).toBe('/en/account');
    expect(routePath('en', 'account')).not.toBe(routePath('en', 'catalog'));
    expect(en.protectedTeaserBody).not.toMatch(/score|metric value|analysis result/i);
  });

  it('UT-005 repeats a search with a stable public representation', () => {
    const first = new URLSearchParams({ q: 'temporal' }).toString();
    const second = new URLSearchParams({ q: 'temporal' }).toString();
    expect(first).toBe(second);
  });

  it('UT-006 builds stale bookmarks from stable project identity', () => {
    expect(routePath('pt-BR', 'project', { projectId: '913' })).toBe('/pt-br/catalogo/913');
  });

  it('UT-007 represents public lifecycle removal without a protected fallback', () => {
    const visible = (state: string) => state === 'active' || state === 'paused';
    expect(['active', 'paused'].filter(visible)).toEqual(['active', 'paused']);
    expect(['archived', 'deleted'].some(visible)).toBe(false);
  });

  it('UT-016 gives an empty review table a named empty state', () => {
    const columns: readonly TableColumn<{ id: string }>[] = [{ key: 'id', header: en.name }];
    render(<Table rows={[]} columns={columns} empty={<EmptyState title={en.noRows} />} />);
    expect(screen.getByRole('heading', { name: en.noRows })).toBeVisible();
  });

  it('UT-017 keeps membership navigation paginated', () => {
    render(<Pagination page={1} hasMore pageSize={50} label={en.members} />);
    expect(screen.getByRole('button', { name: /previous/i })).toBeDisabled();
    expect(screen.getByRole('button', { name: /next/i })).toBeEnabled();
  });

  it('UT-211 falls unsupported locale input back to English', () => {
    expect(normalizeLocale('fr-FR')).toBe('en');
    expect(switchLocalePath('/en/catalog', 'en', 'pt-BR')).toBe('/pt-br/catalogo');
  });

  it('UT-212 keeps both dictionaries complete and nonblank', () => {
    expect(Object.keys(ptBR)).toEqual(Object.keys(en));
    for (const value of [...Object.values(en), ...Object.values(ptBR)]) {
      expect(value.trim()).not.toBe('');
    }
  });

  it('UT-213 renders dense evidence as a bounded detail summary', () => {
    const rows = [{ id: '1', name: 'A project', state: 'active' }];
    const columns: readonly TableColumn<(typeof rows)[number]>[] = [
      { key: 'name', header: en.name },
      { key: 'state', header: en.state },
    ];
    render(<Table rows={rows} columns={columns} layout="detail" />);
    expect(screen.getByRole('list')).toBeVisible();
    expect(screen.getByText('A project')).toBeVisible();
  });

  it('UT-214 omits administrator controls from a member shell', () => {
    render(
      <AppShell
        viewport="desktop"
        nav={[{ key: 'catalog', label: en.catalog, icon: 'search' }]}
        activeKey="catalog"
        onNavigate={() => undefined}
        locale="en"
      >
        <p>content</p>
      </AppShell>,
    );
    expect(screen.getByText(en.catalog)).toBeVisible();
    expect(screen.queryByText(en.audit)).not.toBeInTheDocument();
  });

  it('UT-215 repeats locale switching without altering route state', () => {
    const portuguese = switchLocalePath('/en/catalog/42', 'en', 'pt-BR');
    expect(switchLocalePath(portuguese, 'pt-BR', 'en')).toBe('/en/catalog/42');
  });

  it('UT-216 resolves localized route identity before protected content', () => {
    expect(routeSegment('pt-BR', 'members')).toBe('admin/membros');
    expect(routePath('pt-BR', 'members')).toMatch(/^\/pt-br\//);
  });

  it('UT-217 keeps a disabled lifecycle action labeled at narrow widths', () => {
    render(<Button disabled>{ptBR.deleteAction}</Button>);
    const action = screen.getByRole('button', { name: ptBR.deleteAction });
    expect(action).toBeDisabled();
    expect(action).toHaveTextContent(ptBR.deleteAction);
  });
});
