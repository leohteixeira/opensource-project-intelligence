import type { CSSProperties, MouseEventHandler, ReactNode } from 'react';

import { Icon } from './Icon';

/**
 * Text link. External and evidence links are marked, because a link that leaves the workspace or
 * opens a source document is a different promise from in-app navigation.
 */
export interface LinkProps {
  readonly children: ReactNode;
  readonly href?: string;
  readonly external?: boolean;
  readonly evidence?: boolean;
  readonly size?: 'inherit' | 'xs' | 'sm' | 'base' | 'md' | 'lg';
  readonly onClick?: MouseEventHandler<HTMLAnchorElement>;
  readonly style?: CSSProperties;
}

export function Link({
  children,
  href,
  external,
  evidence,
  size = 'inherit',
  onClick,
  style,
}: LinkProps) {
  return (
    <a
      href={href ?? '#'}
      onClick={onClick}
      target={external ? '_blank' : undefined}
      rel={external ? 'noreferrer noopener' : undefined}
      style={{
        display: 'inline-flex',
        alignItems: 'baseline',
        gap: 'var(--space-05)',
        color: evidence ? 'var(--evidence-fg)' : undefined,
        textDecorationColor: evidence ? 'var(--evidence-underline)' : undefined,
        fontSize: size === 'inherit' ? undefined : `var(--text-${size})`,
        ...style,
      }}
    >
      <span>{children}</span>
      {external ? <Icon name="external-link" size={12} style={{ alignSelf: 'center' }} /> : null}
    </a>
  );
}
