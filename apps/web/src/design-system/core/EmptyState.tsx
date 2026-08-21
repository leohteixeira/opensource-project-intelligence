import type { CSSProperties, ReactNode } from 'react';

import { Icon } from './Icon';
import type { IconName } from './icons';

/**
 * Empty states carry the next permitted action in the body — never a heading with a helper
 * paragraph under it, and never an empty grid.
 */
export interface EmptyStateProps {
  readonly glyph?: IconName;
  readonly title: ReactNode;
  readonly children?: ReactNode;
  readonly action?: ReactNode;
  readonly compact?: boolean;
  readonly style?: CSSProperties;
}

export function EmptyState({
  glyph = 'circle-dashed',
  title,
  children,
  action,
  compact,
  style,
}: EmptyStateProps) {
  return (
    <div
      style={{
        display: 'grid',
        justifyItems: 'start',
        gap: 'var(--space-1)',
        padding: compact ? 'var(--space-3)' : 'var(--space-5) var(--space-4)',
        border: 'var(--border-dashed)',
        borderRadius: 'var(--radius-sm)',
        background: 'var(--surface-card)',
        ...style,
      }}
    >
      <Icon name={glyph} size={20} style={{ color: 'var(--text-tertiary)' }} />
      <p style={{ font: 'var(--type-subsection)', color: 'var(--text-primary)' }}>{title}</p>
      {children ? (
        <div
          style={{
            font: 'var(--type-body)',
            fontSize: 'var(--text-sm)',
            color: 'var(--text-secondary)',
            maxWidth: '56ch',
          }}
        >
          {children}
        </div>
      ) : null}
      {action ? <div style={{ marginTop: 'var(--space-05)' }}>{action}</div> : null}
    </div>
  );
}
