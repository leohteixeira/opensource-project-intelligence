import { useState } from 'react';

import { Button, Menu, Wordmark } from '../../design-system';
import { AccessScreen, type AccessState } from './AccessScreen';
import { CatalogScreen } from './CatalogScreen';
import { TeaserScreen } from './TeaserScreen';
import type { PublicProject } from './fixtures';

/**
 * The anonymous and Applicant shell. It has no primary navigation: an anonymous visitor has one
 * destination. The language control shows that `/en` and `/pt-br` are real routes rather than a
 * client-side toggle.
 */
export function PublicCatalogKit() {
  const [route, setRoute] = useState<'catalog' | 'teaser' | 'access'>('catalog');
  const [project, setProject] = useState<PublicProject | null>(null);
  const [access, setAccess] = useState<AccessState>('pending');

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      <a className="opi-skip" href="#main">
        Skip to content
      </a>
      <header
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 'var(--space-2)',
          height: 'var(--nav-h)',
          padding: '0 var(--space-3)',
          background: 'var(--surface-nav)',
          borderBottom: 'var(--border-default)',
          position: 'sticky',
          top: 0,
          zIndex: 'var(--z-sticky)',
        }}
      >
        <button
          type="button"
          onClick={() => setRoute('catalog')}
          style={{ border: 0, background: 'transparent', padding: 0, cursor: 'pointer' }}
        >
          <Wordmark variant="inline" />
        </button>
        <nav
          aria-label="Public"
          style={{
            marginLeft: 'auto',
            display: 'flex',
            gap: 'var(--space-1)',
            alignItems: 'center',
          }}
        >
          <Menu
            triggerIcon="languages"
            triggerLabel="Language"
            items={[
              { label: 'English — /en', icon: 'check' },
              { label: 'Portugues (Brasil) — /pt-br', icon: 'languages' },
            ]}
          />
          <Button variant="secondary" size="md" onClick={() => setRoute('access')}>
            Access status
          </Button>
          <Button variant="primary" size="md" iconStart="log-in" onClick={() => setRoute('access')}>
            Sign in
          </Button>
        </nav>
      </header>
      <main
        id="main"
        style={{
          flex: 1,
          padding: 'var(--space-4) var(--space-3) var(--space-6)',
          maxWidth: 'var(--content-max)',
          width: '100%',
          margin: '0 auto',
        }}
      >
        {route === 'catalog' ? (
          <CatalogScreen
            onOpen={(next) => {
              setProject(next);
              setRoute('teaser');
            }}
            onSignIn={() => setRoute('access')}
          />
        ) : null}
        {route === 'teaser' && project ? (
          <TeaserScreen
            project={project}
            onBack={() => setRoute('catalog')}
            onSignIn={() => setRoute('access')}
          />
        ) : null}
        {route === 'access' ? (
          <AccessScreen state={access} onState={setAccess} onBack={() => setRoute('catalog')} />
        ) : null}
      </main>
      <footer
        style={{
          padding: 'var(--space-2) var(--space-3)',
          borderTop: 'var(--border-hairline)',
          background: 'var(--surface-card)',
          font: 'var(--type-caption)',
          color: 'var(--text-secondary)',
          display: 'flex',
          gap: 'var(--space-2)',
          flexWrap: 'wrap',
        }}
      >
        <span>Self-hosted deployment · public data only</span>
        <span style={{ fontFamily: 'var(--font-mono)' }}>/en · /pt-br</span>
      </footer>
    </div>
  );
}
