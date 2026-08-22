import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router';

import { Banner, Button, EmptyState, Link, Panel, Skeleton } from '../../design-system';
import { fetchCatalogProject } from '../api';
import { useApplication } from '../router';
import { routePath } from '../routes';

export function ProjectScreen() {
  const { t } = useTranslation();
  const { projectId = '' } = useParams();
  const { locale, session } = useApplication();
  const project = useQuery({
    queryKey: ['catalog-project', projectId],
    queryFn: () => fetchCatalogProject(projectId),
    enabled: Boolean(projectId),
  });

  if (project.isPending) return <Skeleton height={280} />;
  if (project.isError || !project.data) {
    return (
      <EmptyState
        title={t('notFound')}
        action={<Button href={routePath(locale, 'catalog')}>{t('backToCatalog')}</Button>}
      />
    );
  }

  return (
    <div style={{ display: 'grid', gap: 'var(--space-3)' }}>
      <header style={{ display: 'grid', gap: 'var(--space-1)', maxWidth: 820 }}>
        <p style={{ font: 'var(--type-eyebrow)', color: 'var(--text-tertiary)' }}>
          {t('publicIdentity')}
        </p>
        <h1 style={{ font: 'var(--type-page)' }}>{project.data.name}</h1>
        <p style={{ font: 'var(--type-body)', color: 'var(--text-secondary)' }}>
          {project.data.description}
        </p>
      </header>
      <Banner tone="neutral" title={t('publicOnly')}>
        {t('originalEvidence')}
      </Banner>
      <Panel title={t('projectSources')}>
        <ul style={{ display: 'grid', gap: 'var(--space-1)' }}>
          {project.data.source_links.map((source) => (
            <li key={source}>
              <Link href={source} external>
                {source}
              </Link>
            </li>
          ))}
        </ul>
      </Panel>
      <Panel title={t('protectedTeaser')} tone="attention">
        <p style={{ marginBottom: 'var(--space-15)' }}>{t('protectedTeaserBody')}</p>
        {session.state !== 'active' ? (
          <Button
            href={`/auth/login?return_to=${encodeURIComponent(routePath(locale, 'project', { projectId }))}`}
          >
            {t('signIn')}
          </Button>
        ) : null}
      </Panel>
    </div>
  );
}
