import { useState, type ReactNode } from 'react';

import {
  AppShell,
  Button,
  EmptyState,
  Icon,
  IconButton,
  Menu,
  Panel,
  StatusBadge,
  Tabs,
  type TabItem,
} from '../../design-system';
import { ADMIN_NAV, PRIMARY_NAV } from '../nav';
import { AdoptionSecurityScreen } from './AdoptionSecurityScreen';
import { AiRunsScreen } from './AiRunsScreen';
import { ContributorsScreen } from './ContributorsScreen';
import { ExportsScreen } from './ExportsScreen';
import { KnowledgeScreen } from './KnowledgeScreen';
import { LifecycleScreen } from './LifecycleScreen';
import { OverviewScreen } from './OverviewScreen';
import { ReleasesScreen, type ReleaseLocale } from './ReleasesScreen';
import { SourcesScreen } from './SourcesScreen';
import { CUTOFF, MEMBER, PROJECT } from './fixtures';

const TABS: readonly TabItem[] = [
  { value: 'overview', label: 'Overview' },
  { value: 'health', label: 'Health' },
  { value: 'contributors', label: 'Contributors' },
  { value: 'adoption-security', label: 'Adoption & Security' },
  { value: 'trends', label: 'Trends' },
  { value: 'releases', label: 'Releases' },
  { value: 'topics', label: 'Topics' },
  { value: 'knowledge', label: 'Knowledge' },
  { value: 'sources', label: 'Sources' },
];

const ELSEWHERE: Record<string, string> = {
  health: 'Health',
  trends: 'Trends',
  topics: 'Topics',
};

const PT = {
  title: 'Versões',
  route: '/pt-br/projects/temporal/releases',
  actions: { lifecycle: 'Ciclo de vida', exportAction: 'Exportar', runs: 'Execuções de IA' },
  tabs: {
    overview: 'Visão geral',
    health: 'Saúde',
    contributors: 'Colaboradores',
    'adoption-security': 'Adoção e segurança',
    trends: 'Tendências',
    releases: 'Versões',
    topics: 'Tópicos',
    knowledge: 'Documentação',
    sources: 'Fontes',
  } as Record<string, string>,
};

const WORKSPACE_ROUTES = ['portfolio', 'compare', 'radar', 'alerts'];

/**
 * The nine project surfaces the workspace shell leaves out: overview, contributors, adoption and
 * security, releases, knowledge, sources and associations, exports, AI run governance and lifecycle.
 */
export function ProjectEvidenceKit({ onOpenKit }: { readonly onOpenKit?: (kit: string) => void }) {
  const [route, setRoute] = useState('project');
  const [tab, setTab] = useState('overview');
  const [locale, setLocale] = useState<ReleaseLocale>('en');
  const [viewport, setViewport] = useState<'desktop' | 'mobile'>('desktop');

  const goTab = (next: string) => {
    setRoute('project');
    setTab(next);
  };

  const crossKit = (label: string) => (
    <Panel title={label} meta="built in the workspace shell">
      <EmptyState
        glyph="arrow-right"
        title={`The ${label} tab is built in the workspace shell`}
        action={
          <Button variant="secondary" onClick={() => onOpenKit?.('workspace')}>
            Open the workspace
          </Button>
        }
      >
        This shell covers the nine surfaces the workspace leaves out. Health, Trends, Topics and
        Sources &amp; Jobs are built there, against the same fixtures and the same components.
      </EmptyState>
    </Panel>
  );

  let body: ReactNode;
  let title: string = PROJECT.name;
  let subtitle = `/en/projects/${PROJECT.slug}/${tab}`;
  let onBack: (() => void) | undefined;
  let actions = null;

  if (route === 'exports') {
    title = 'Exports';
    subtitle = '/en/exports';
    onBack = () => setRoute('project');
    body = <ExportsScreen onBack={() => setRoute('project')} />;
  } else if (route === 'runs') {
    title = 'AI runs';
    subtitle = `/en/projects/${PROJECT.slug}/runs`;
    onBack = () => setRoute('project');
    body = <AiRunsScreen onBack={() => setRoute('project')} />;
  } else if (route === 'lifecycle') {
    title = 'Lifecycle';
    subtitle = `/en/projects/${PROJECT.slug}/lifecycle`;
    onBack = () => setRoute('project');
    body = <LifecycleScreen onBack={() => setRoute('project')} />;
  } else if (WORKSPACE_ROUTES.includes(route)) {
    title = route.charAt(0).toUpperCase() + route.slice(1);
    subtitle = `/en/${route}`;
    body = (
      <Panel title={title} meta="built in the workspace shell">
        <EmptyState
          glyph="layout-dashboard"
          title={`${title} is built in the workspace shell`}
          action={
            <Button variant="secondary" onClick={() => onOpenKit?.('workspace')}>
              Open the workspace
            </Button>
          }
        >
          This shell starts inside one project. The portfolio, comparison, radar and alert
          destinations are built in the workspace against the same components.
        </EmptyState>
      </Panel>
    );
  } else {
    const portugueseReleases = locale === 'pt-BR' && tab === 'releases';

    if (portugueseReleases) {
      title = PT.title;
      subtitle = PT.route;
    }
    onBack = () => onOpenKit?.('workspace');

    const labels = portugueseReleases
      ? PT.actions
      : { lifecycle: 'Lifecycle', exportAction: 'Export', runs: 'AI runs' };

    actions = (
      <>
        <Button
          variant="secondary"
          size="md"
          iconStart="archive"
          onClick={() => setRoute('lifecycle')}
        >
          {labels.lifecycle}
        </Button>
        <Button
          variant="secondary"
          size="md"
          iconStart="download"
          onClick={() => setRoute('exports')}
        >
          {labels.exportAction}
        </Button>
        <Button variant="primary" size="md" iconStart="sparkles" onClick={() => setRoute('runs')}>
          {labels.runs}
        </Button>
      </>
    );

    const inner =
      tab === 'overview' ? (
        <OverviewScreen onGoTab={goTab} onOpenLifecycle={() => setRoute('lifecycle')} />
      ) : tab === 'contributors' ? (
        <ContributorsScreen />
      ) : tab === 'adoption-security' ? (
        <AdoptionSecurityScreen />
      ) : tab === 'releases' ? (
        <ReleasesScreen locale={locale} onOpenRun={() => setRoute('runs')} />
      ) : tab === 'knowledge' ? (
        <KnowledgeScreen onOpenRun={() => setRoute('runs')} />
      ) : tab === 'sources' ? (
        <SourcesScreen onOpenLifecycle={() => setRoute('lifecycle')} />
      ) : (
        crossKit(ELSEWHERE[tab] ?? tab)
      );

    body = (
      <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--space-15)',
            flexWrap: 'wrap',
          }}
        >
          <StatusBadge status="active" size="sm" />
          <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>
            {PROJECT.id} · {PROJECT.repositories} repositories · {PROJECT.sources} sources · cutoff{' '}
            {CUTOFF}
          </span>
        </div>
        <Tabs
          value={tab}
          onChange={setTab}
          items={
            portugueseReleases
              ? TABS.map((entry) => ({ ...entry, label: PT.tabs[entry.value] ?? entry.label }))
              : TABS
          }
        />
        {inner}
      </div>
    );
  }

  const switchLocale = () => {
    setLocale(locale === 'en' ? 'pt-BR' : 'en');
    setRoute('project');
    setTab('releases');
  };

  return (
    <AppShell
      viewport={viewport}
      nav={PRIMARY_NAV}
      secondaryNav={ADMIN_NAV}
      activeKey={WORKSPACE_ROUTES.includes(route) ? route : 'projects'}
      onNavigate={(key) => setRoute(key === 'projects' ? 'project' : key)}
      locale={locale}
      member={{ name: MEMBER.displayName, role: `${MEMBER.role} · ${MEMBER.timezone}` }}
      utilities={
        <>
          <IconButton icon="search" label="Search projects" variant="outline" shape="circle" />
          <IconButton
            icon="languages"
            label={locale === 'en' ? 'Switch to Portuguese (pt-BR)' : 'Switch to English'}
            variant="outline"
            shape="circle"
            selected={locale === 'pt-BR'}
            onClick={switchLocale}
          />
          <Menu
            align="end"
            triggerLabel="Account and preview"
            trigger={
              <button
                type="button"
                className="opi-btn opi-btn--secondary opi-icon-btn--outline"
                aria-label="Account and preview"
                style={{ width: 36, height: 36, padding: 0, borderRadius: 'var(--radius-pill)' }}
              >
                <Icon name="ellipsis-vertical" size={16} />
              </button>
            }
            items={[
              {
                label: viewport === 'desktop' ? 'Preview mobile layout' : 'Preview desktop layout',
                icon: 'smartphone',
                onSelect: () => setViewport(viewport === 'desktop' ? 'mobile' : 'desktop'),
              },
              {
                label: locale === 'en' ? 'Switch to Portuguese (pt-BR)' : 'Switch to English',
                icon: 'languages',
                onSelect: switchLocale,
              },
              { separator: true },
              {
                label: 'Open the workspace',
                icon: 'arrow-right',
                onSelect: () => onOpenKit?.('workspace'),
              },
              { label: 'Sign out', icon: 'log-out' },
            ]}
          />
        </>
      }
      title={title}
      subtitle={subtitle}
      onBack={onBack}
      actions={actions}
    >
      {body}
    </AppShell>
  );
}
