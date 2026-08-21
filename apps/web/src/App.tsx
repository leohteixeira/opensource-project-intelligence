import { useState } from 'react';

import { Tabs } from './design-system';
import { AdministrationKit } from './kits/administration/AdministrationKit';
import { ProjectEvidenceKit } from './kits/project-evidence/ProjectEvidenceKit';
import { PublicCatalogKit } from './kits/public-catalog/PublicCatalogKit';
import { WorkspaceKit } from './kits/workspace/WorkspaceKit';

/**
 * One browser application with four shells, recreated from the design system's UI kits. The
 * product routes them under localized `/en` and `/pt-br` paths; this phase carries no router
 * (ADR 0006), so the selector below stands in for that navigation. It is scaffolding, not product
 * UI, and is replaced by real routes when the HTTP contract exists.
 */
const SHELLS = [
  { value: 'public-catalog', label: 'Public catalog' },
  { value: 'workspace', label: 'Member workspace' },
  { value: 'project-evidence', label: 'Project evidence' },
  { value: 'administration', label: 'Administration' },
];

export function App() {
  const [shell, setShell] = useState('workspace');

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 'var(--space-15)',
          padding: 'var(--space-1) var(--space-3)',
          background: 'var(--surface-page)',
          borderBottom: 'var(--border-default)',
          flex: 'none',
        }}
      >
        <span
          style={{
            font: 'var(--type-eyebrow)',
            letterSpacing: 'var(--tracking-eyebrow)',
            textTransform: 'uppercase',
            color: 'var(--text-tertiary)',
            whiteSpace: 'nowrap',
          }}
        >
          Shell
        </span>
        <Tabs variant="pill" size="sm" value={shell} onChange={setShell} items={SHELLS} />
      </div>
      <div style={{ flex: 1, minHeight: 0 }}>
        {shell === 'public-catalog' ? <PublicCatalogKit /> : null}
        {shell === 'workspace' ? <WorkspaceKit onOpenKit={setShell} /> : null}
        {shell === 'project-evidence' ? <ProjectEvidenceKit onOpenKit={setShell} /> : null}
        {shell === 'administration' ? <AdministrationKit onOpenKit={setShell} /> : null}
      </div>
    </div>
  );
}
