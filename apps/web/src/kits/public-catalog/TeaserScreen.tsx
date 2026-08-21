import { Banner, Button, EmptyState, Icon, Link, Panel } from '../../design-system';
import type { PublicProject } from './fixtures';

export function TeaserScreen({
  project,
  onBack,
  onSignIn,
}: {
  readonly project: PublicProject;
  readonly onBack: () => void;
  readonly onSignIn: () => void;
}) {
  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)', maxWidth: 860 }}>
      <button
        type="button"
        onClick={onBack}
        style={{
          justifySelf: 'start',
          display: 'inline-flex',
          gap: 5,
          alignItems: 'center',
          border: 0,
          background: 'transparent',
          padding: 0,
          font: 'var(--type-ui)',
          color: 'var(--text-link)',
          cursor: 'pointer',
        }}
      >
        <Icon name="chevron-left" size={15} />
        Public catalog
      </button>
      <div style={{ display: 'grid', gap: 'var(--space-05)' }}>
        <h1 style={{ font: 'var(--type-page-title)' }}>{project.name}</h1>
        <p style={{ font: 'var(--type-body)', color: 'var(--text-secondary)' }}>
          {project.description}
        </p>
        <p style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>
          /en/catalog/{project.id}
        </p>
      </div>
      <Panel title="Public sources">
        <div style={{ display: 'grid', gap: 'var(--space-075)' }}>
          {project.links.map((link) => (
            <span
              key={link}
              style={{ display: 'flex', gap: 'var(--space-075)', alignItems: 'center' }}
            >
              <Icon name="link-2" size={13} style={{ color: 'var(--text-tertiary)' }} />
              <Link href={`https://${link}`} external size="sm">
                {link}
              </Link>
            </span>
          ))}
        </div>
      </Panel>
      <Panel title="Intelligence">
        <EmptyState
          glyph="lock"
          title="Metrics and analyses require an approved account"
          action={
            <div style={{ display: 'flex', gap: 'var(--space-1)', flexWrap: 'wrap' }}>
              <Button variant="primary" iconStart="log-in" onClick={onSignIn}>
                Sign in
              </Button>
              <Button variant="secondary" onClick={onSignIn}>
                Request access
              </Button>
            </div>
          }
        >
          Health dimensions, contributor sustainability, adoption and security evidence, comparison,
          radar placement, alert history and exports are never shown on anonymous surfaces —
          including through a deep link to a protected view.
        </EmptyState>
      </Panel>
      <Banner
        tone="neutral"
        title="Identity is provided by the workspace-shared Keycloak deployment"
      >
        Signing in establishes who you are. It does not grant product access: a local Admin approves
        membership and assigns one fixed role.
      </Banner>
    </div>
  );
}
