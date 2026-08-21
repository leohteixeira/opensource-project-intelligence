import type { CSSProperties } from 'react';

import { Icon } from './Icon';
import { STATUS, TONES, type StatusKey } from './status';

export interface StatusBadgeProps {
  readonly status: StatusKey;
  /** Overrides the frozen word. Use it to localize, never to invent a state. */
  readonly label?: string;
  readonly detail?: string;
  readonly size?: 'md' | 'sm';
  readonly glyphOnly?: boolean;
  readonly style?: CSSProperties;
}

export function StatusBadge({
  status,
  label,
  detail,
  size = 'md',
  glyphOnly,
  style,
}: StatusBadgeProps) {
  const spec = STATUS[status];
  const [fg, bg, border] = TONES[spec.tone];
  const text = label ?? spec.label;

  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 'var(--space-05)',
        maxWidth: '100%',
        color: fg,
        background: spec.outline ? 'transparent' : bg,
        border: `1px solid ${spec.outline ? border : 'transparent'}`,
        borderRadius: 'var(--radius-pill)',
        padding: size === 'sm' ? '2px var(--space-075)' : '3px var(--space-1)',
        font: size === 'sm' ? 'var(--type-caption)' : 'var(--type-ui)',
        fontWeight: 'var(--weight-medium)',
        whiteSpace: 'nowrap',
        ...style,
      }}
      title={detail}
    >
      <Icon
        name={spec.glyph}
        size={size === 'sm' ? 12 : 13}
        className={spec.spin ? 'opi-spin' : undefined}
      />
      {glyphOnly ? (
        <span className="opi-vh">{text}</span>
      ) : (
        <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>{text}</span>
      )}
      {detail && !glyphOnly ? (
        <span style={{ color: 'var(--text-secondary)', fontWeight: 'var(--weight-regular)' }}>
          {detail}
        </span>
      ) : null}
    </span>
  );
}
