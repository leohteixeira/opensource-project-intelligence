import { useState } from 'react';

import {
  AppShell,
  Button,
  EmptyState,
  Icon,
  IconButton,
  Menu,
  StatusBadge,
} from '../../design-system';
import { ADMIN_NAV, PRIMARY_NAV } from '../nav';
import { AuditScreen } from './AuditScreen';
import { MembersScreen } from './MembersScreen';
import { OperationsScreen } from './OperationsScreen';
import { PoliciesScreen } from './PoliciesScreen';

const ADMIN_NAV_WITH_QUEUE = ADMIN_NAV.map((item) =>
  item.key === 'members' ? { ...item, badge: 2 } : item,
);

const TITLES: Record<string, string> = {
  members: 'Members',
  policies: 'Policies',
  audit: 'Audit',
  operations: 'Operations',
};

/**
 * Admin and operator surfaces. Admin destinations sit behind the top bar's "Administration" menu,
 * not in a separate administration application. Analyst actions never appear here — they are
 * contextual inside the workspace.
 */
export function AdministrationKit({ onOpenKit }: { readonly onOpenKit?: (kit: string) => void }) {
  const [route, setRoute] = useState('members');

  const body =
    route === 'members' ? (
      <MembersScreen />
    ) : route === 'policies' ? (
      <PoliciesScreen />
    ) : route === 'audit' ? (
      <AuditScreen />
    ) : route === 'operations' ? (
      <OperationsScreen />
    ) : (
      <EmptyState
        glyph="layout-dashboard"
        title="Member surfaces live in the workspace"
        action={
          <Button variant="secondary" onClick={() => onOpenKit?.('workspace')}>
            Open the workspace
          </Button>
        }
      >
        Portfolio, projects, comparison, radar and alerts are member surfaces and are built in the
        workspace shell.
      </EmptyState>
    );

  return (
    <AppShell
      nav={PRIMARY_NAV}
      secondaryNav={ADMIN_NAV_WITH_QUEUE}
      activeKey={route}
      onNavigate={setRoute}
      secondaryLabel="Administration"
      title={TITLES[route] ?? 'Workspace'}
      titleAdornment={<StatusBadge status="available" label="Admin" size="sm" />}
      member={{ name: 'Rafael Costa', role: 'Admin · UTC-3' }}
      utilities={
        <>
          <IconButton icon="bell" label="Notifications" variant="outline" shape="circle" />
          <Menu
            align="end"
            triggerLabel="Account"
            trigger={
              <button
                type="button"
                className="opi-btn opi-btn--secondary opi-icon-btn--outline"
                aria-label="Account"
                style={{ width: 36, height: 36, padding: 0, borderRadius: 'var(--radius-pill)' }}
              >
                <Icon name="ellipsis-vertical" size={16} />
              </button>
            }
            items={[
              { label: 'Preferences', icon: 'settings' },
              { label: 'Switch to Portuguese (pt-BR)', icon: 'languages' },
              { separator: true },
              { label: 'Sign out', icon: 'log-out' },
            ]}
          />
        </>
      }
    >
      {body}
    </AppShell>
  );
}
