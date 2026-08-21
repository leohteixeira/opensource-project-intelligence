import { useState } from 'react';

import { Button, EmptyState, Icon, Pagination, TextField } from '../../design-system';
import { CATALOG, type PublicProject } from './fixtures';

export function CatalogScreen({
  onOpen,
  onSignIn,
}: {
  readonly onOpen: (project: PublicProject) => void;
  readonly onSignIn: () => void;
}) {
  const [query, setQuery] = useState('');
  const rows = CATALOG.filter((project) =>
    `${project.name} ${project.description}`.toLowerCase().includes(query.toLowerCase()),
  );

  return (
    <div style={{ display: 'grid', gap: 'var(--space-3)' }}>
      <div style={{ display: 'grid', gap: 'var(--space-15)', maxWidth: '68ch' }}>
        <h1 style={{ font: 'var(--type-display)' }}>
          Public open source technologies tracked by this workspace
        </h1>
        <p style={{ font: 'var(--type-body)', color: 'var(--text-secondary)' }}>
          This catalog lists project names, public descriptions and public source links. Metrics,
          health dimensions, comparisons, radar placement, alerts and analyses are available to
          approved members only.
        </p>
        <div style={{ display: 'flex', gap: 'var(--space-1)', flexWrap: 'wrap' }}>
          <Button variant="primary" size="lg" iconStart="log-in" onClick={onSignIn}>
            Sign in
          </Button>
          <Button variant="secondary" size="lg" iconStart="user-plus" onClick={onSignIn}>
            Request access
          </Button>
        </div>
      </div>
      <div style={{ display: 'grid', gap: 'var(--space-15)' }}>
        <div
          style={{
            display: 'flex',
            gap: 'var(--space-1)',
            alignItems: 'flex-end',
            flexWrap: 'wrap',
          }}
        >
          <TextField
            id="catalog-search"
            label="Search the catalog"
            type="search"
            iconStart="search"
            placeholder="Search by name or description"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            style={{ minWidth: 280 }}
          />
          <span
            style={{
              font: 'var(--type-caption)',
              color: 'var(--text-secondary)',
              paddingBottom: 12,
              fontVariantNumeric: 'tabular-nums',
            }}
          >
            {rows.length} of {CATALOG.length} public projects
          </span>
        </div>
        {rows.length ? (
          <ul
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
              gap: 'var(--space-15)',
            }}
          >
            {rows.map((project) => (
              <li key={project.id}>
                <button
                  type="button"
                  onClick={() => onOpen(project)}
                  className="opi-item"
                  style={{
                    width: '100%',
                    height: '100%',
                    textAlign: 'left',
                    display: 'grid',
                    gap: 'var(--space-1)',
                    alignContent: 'start',
                    padding: 'var(--space-2)',
                    border: '1px solid var(--border-card)',
                    borderRadius: 'var(--radius-md)',
                    boxShadow: 'var(--shadow-card)',
                    background: 'var(--surface-card)',
                    cursor: 'pointer',
                  }}
                >
                  <span style={{ font: 'var(--type-subsection)', color: 'var(--text-primary)' }}>
                    {project.name}
                  </span>
                  <span
                    style={{
                      font: 'var(--type-body)',
                      fontSize: 'var(--text-sm)',
                      color: 'var(--text-secondary)',
                    }}
                  >
                    {project.description}
                  </span>
                  <span style={{ display: 'grid', gap: 2 }}>
                    {project.links.map((link) => (
                      <span
                        key={link}
                        style={{
                          display: 'flex',
                          gap: 5,
                          alignItems: 'center',
                          font: 'var(--type-mono-xs)',
                          color: 'var(--text-tertiary)',
                        }}
                      >
                        <Icon name="link-2" size={11} />
                        {link}
                      </span>
                    ))}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        ) : (
          <EmptyState
            glyph="search-x"
            title={`No public project matches ${JSON.stringify(query)}`}
            action={
              <Button variant="secondary" onClick={() => setQuery('')}>
                Clear search
              </Button>
            }
          >
            Search covers public names and descriptions only. Paused projects remain listed;
            archived and deleted projects do not.
          </EmptyState>
        )}
        <Pagination page={1} hasMore={false} total={rows.length} label="Public projects" />
      </div>
    </div>
  );
}
