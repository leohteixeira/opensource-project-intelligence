import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useSearchParams } from 'react-router';

import {
  Banner,
  Button,
  EmptyState,
  FilterBar,
  Link,
  Pagination,
  Panel,
  Skeleton,
  TextField,
} from '../../design-system';
import { fetchCatalog } from '../api';
import { useApplication } from '../router';
import { routePath } from '../routes';

export function CatalogScreen() {
  const { t } = useTranslation();
  const { locale, narrow } = useApplication();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const query = params.get('q') ?? '';
  const cursor = params.get('cursor') ?? undefined;
  const page = Math.max(1, Number(params.get('page') ?? '1') || 1);
  const [draft, setDraft] = useState(query);
  const cursorHistory = readCursorHistory(params);
  const projects = useQuery({
    queryKey: ['catalog', query, cursor],
    queryFn: () => fetchCatalog(query, cursor),
    placeholderData: keepPreviousData,
  });

  function search(event: FormEvent) {
    event.preventDefault();
    const next = new URLSearchParams();
    if (draft.trim()) next.set('q', draft.trim());
    setParams(next);
  }

  function goNext() {
    if (!projects.data?.next_cursor) return;
    const next = new URLSearchParams(params);
    const history = [...cursorHistory, cursor ?? ''];
    next.set('cursor', projects.data.next_cursor);
    next.set('page', String(page + 1));
    next.set('history', encodeURIComponent(JSON.stringify(history)));
    setParams(next);
  }

  function goPrevious() {
    if (page <= 1) return;
    const history = [...cursorHistory];
    const previous = history.pop();
    const next = new URLSearchParams(params);
    if (previous) next.set('cursor', previous);
    else next.delete('cursor');
    next.set('page', String(page - 1));
    if (history.length) next.set('history', encodeURIComponent(JSON.stringify(history)));
    else next.delete('history');
    setParams(next);
  }

  return (
    <div style={{ display: 'grid', gap: 'var(--space-3)' }}>
      <header style={{ display: 'grid', gap: 'var(--space-1)', maxWidth: 760 }}>
        <h1 style={{ font: 'var(--type-page)' }}>{t('catalogTitle')}</h1>
        <p style={{ font: 'var(--type-body)', color: 'var(--text-secondary)' }}>
          {t('catalogIntro')}
        </p>
      </header>

      <FilterBar
        applied={query ? [{ key: 'q', field: t('search'), value: query }] : []}
        onClear={() => {
          setDraft('');
          setParams({});
        }}
        onRemove={() => {
          setDraft('');
          setParams({});
        }}
      >
        <form onSubmit={search} style={{ display: 'flex', gap: 'var(--space-1)', flex: 1 }}>
          <TextField
            id="catalog-search"
            type="search"
            inputMode="search"
            iconStart="search"
            placeholder={t('search')}
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            style={{ flex: 1, minWidth: narrow ? '100%' : 280 }}
          />
          <Button type="submit">{t('searchAction')}</Button>
        </form>
      </FilterBar>

      {projects.isError ? (
        <Banner
          tone="critical"
          title={t('requestFailed')}
          actions={<Button onClick={() => void projects.refetch()}>{t('retry')}</Button>}
        />
      ) : null}

      {projects.isPending ? (
        <div aria-label={t('loading')} style={{ display: 'grid', gap: 'var(--space-1)' }}>
          <Skeleton height={110} />
          <Skeleton height={110} />
        </div>
      ) : null}

      {projects.data && projects.data.items.length === 0 ? (
        <EmptyState
          glyph={query ? 'search-x' : 'package'}
          title={query ? t('noMatches') : t('noProjects')}
        >
          {t('noProjectsHelp')}
        </EmptyState>
      ) : null}

      {projects.data?.items.length ? (
        <div
          aria-live="polite"
          style={{
            display: 'grid',
            gridTemplateColumns: narrow ? '1fr' : 'repeat(auto-fit, minmax(280px, 1fr))',
            gap: 'var(--space-2)',
            opacity: projects.isPlaceholderData ? 0.65 : 1,
          }}
        >
          {projects.data.items.map((project) => (
            <Panel
              key={project.id}
              title={project.name}
              eyebrow={t('publicIdentity')}
              footer={
                <Link href={routePath(locale, 'project', { projectId: project.id })}>
                  {t('projectSources')}
                </Link>
              }
              interactive
              onClick={() => navigate(routePath(locale, 'project', { projectId: project.id }))}
            >
              <p style={{ font: 'var(--type-body)', color: 'var(--text-secondary)' }}>
                {project.description}
              </p>
            </Panel>
          ))}
        </div>
      ) : null}

      {projects.data && (page > 1 || projects.data.has_more) ? (
        <Pagination
          page={page}
          hasMore={projects.data.has_more}
          pageSize={24}
          label={t('catalog')}
          onPrev={goPrevious}
          onNext={goNext}
        />
      ) : null}
    </div>
  );
}

function readCursorHistory(params: URLSearchParams): string[] {
  const raw = params.get('history');
  if (!raw) return [];
  try {
    const parsed: unknown = JSON.parse(decodeURIComponent(raw));
    return Array.isArray(parsed) && parsed.every((value) => typeof value === 'string')
      ? parsed
      : [];
  } catch {
    return [];
  }
}
