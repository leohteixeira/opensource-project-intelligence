import { useState } from 'react';

import { AppShell, Button, EmptyState, Icon, IconButton, Menu } from '../../design-system';
import { ADMIN_NAV, PRIMARY_NAV } from '../nav';
import { AlertsScreen } from './AlertsScreen';
import { AssistantPanel } from './AssistantPanel';
import { CompareScreen } from './CompareScreen';
import { PortfolioScreen } from './PortfolioScreen';
import { ProjectDetailScreen } from './ProjectDetailScreen';
import { ProjectsScreen } from './ProjectsScreen';
import { RadarScreen } from './RadarScreen';
import { MEMBER, type WorkspaceProject } from './fixtures';

const TITLES: Record<string, string> = {
  portfolio: 'Portfolio',
  projects: 'Projects',
  compare: 'Compare',
  radar: 'Radar',
  alerts: 'Alerts',
};

const ADMIN_ROUTES = ['members', 'policies', 'audit', 'operations'];

/** The approved-member workspace: portfolio, projects, comparison, radar, alerts and the assistant. */
export function WorkspaceKit({ onOpenKit }: { readonly onOpenKit?: (kit: string) => void }) {
  const [route, setRoute] = useState('portfolio');
  const [project, setProject] = useState<WorkspaceProject | null>(null);
  const [tab, setTab] = useState('health');
  const [assistant, setAssistant] = useState(false);
  const [viewport, setViewport] = useState<'desktop' | 'mobile'>('desktop');

  const openProject = (next: WorkspaceProject) => {
    setProject(next);
    setTab('health');
    setRoute('project');
  };
  const go = (key: string) => {
    setRoute(key);
    setProject(null);
  };

  let body = null;

  if (route === 'portfolio') body = <PortfolioScreen onOpenProject={openProject} onGo={go} />;
  if (route === 'projects') body = <ProjectsScreen onOpenProject={openProject} />;
  if (route === 'project' && project) {
    body = <ProjectDetailScreen project={project} tab={tab} onTab={setTab} onOpenKit={onOpenKit} />;
  }
  if (route === 'compare') body = <CompareScreen />;
  if (route === 'radar') body = <RadarScreen />;
  if (route === 'alerts') body = <AlertsScreen />;
  if (ADMIN_ROUTES.includes(route)) {
    body = (
      <EmptyState
        glyph="shield-alert"
        title="Administration lives in its own shell"
        action={
          <Button variant="secondary" onClick={() => onOpenKit?.('administration')}>
            Open administration
          </Button>
        }
      >
        Members, service accounts, policies, audit and operations are governed in the administration
        shell.
      </EmptyState>
    );
  }

  return (
    <AppShell
      viewport={viewport}
      nav={PRIMARY_NAV}
      secondaryNav={ADMIN_NAV}
      activeKey={route === 'project' ? 'projects' : route}
      onNavigate={go}
      locale={MEMBER.locale}
      member={{ name: MEMBER.displayName, role: `${MEMBER.role} · ${MEMBER.timezone}` }}
      utilities={
        <>
          <IconButton icon="search" label="Search projects" variant="outline" shape="circle" />
          <IconButton
            icon="bell"
            label="Notifications — 2 unread"
            variant="outline"
            shape="circle"
          />
          <Menu
            align="end"
            triggerLabel="Account and language"
            trigger={
              <button
                type="button"
                className="opi-btn opi-btn--secondary opi-icon-btn--outline"
                aria-label="Account and language"
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
              { label: 'Preferences', icon: 'settings' },
              { label: 'Switch to Portuguese (pt-BR)', icon: 'languages' },
              { separator: true },
              { label: 'Sign out', icon: 'log-out' },
            ]}
          />
        </>
      }
      title={route === 'project' && project ? project.name : (TITLES[route] ?? 'Administration')}
      onBack={route === 'project' ? () => go('projects') : undefined}
      subtitle={route === 'project' && project ? `/en/projects/${project.slug}/${tab}` : undefined}
      actions={
        <>
          <Button variant="secondary" size="md" iconStart="download">
            Export
          </Button>
          <Button
            variant="primary"
            size="md"
            iconStart="sparkles"
            onClick={() => setAssistant(!assistant)}
          >
            Ask
          </Button>
        </>
      }
      sidePanel={assistant ? <AssistantPanel onClose={() => setAssistant(false)} /> : null}
    >
      {body}
    </AppShell>
  );
}
