import type { CSSProperties } from 'react';

/**
 * Initials avatar. No photographs exist in this product, so a member is identified by a monogram
 * next to their name — never by the monogram alone, which is why it is hidden from assistive
 * technology unless it is given a label.
 */
const BOX = { sm: 'var(--avatar-sm)', md: 'var(--avatar-md)', lg: 'var(--avatar-lg)' } as const;
const FONT = { sm: 10, md: 12, lg: 14 } as const;

export interface AvatarProps {
  readonly name?: string;
  readonly initials?: string;
  readonly size?: keyof typeof BOX;
  readonly tone?: 'neutral' | 'accent';
  readonly label?: string;
  readonly style?: CSSProperties;
}

function monogram(name: string, initials?: string): string {
  if (initials) return initials.toUpperCase();
  const letters = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word.charAt(0))
    .join('');

  return (letters || '?').toUpperCase();
}

export function Avatar({
  name = '',
  initials,
  size = 'md',
  tone = 'neutral',
  label,
  style,
}: AvatarProps) {
  const accent = tone === 'accent';

  return (
    <span
      role={label ? 'img' : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
      style={{
        flex: 'none',
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: BOX[size],
        height: BOX[size],
        borderRadius: 'var(--radius-pill)',
        background: accent ? 'var(--action-accent-soft-bg)' : 'var(--gray-50)',
        color: accent ? 'var(--action-accent-soft-fg)' : 'var(--gray-600)',
        border: `1px solid ${accent ? 'var(--blue-100)' : 'var(--gray-100)'}`,
        font: 'var(--type-ui)',
        fontSize: FONT[size],
        fontWeight: 'var(--weight-semibold)',
        letterSpacing: '.01em',
        userSelect: 'none',
        ...style,
      }}
    >
      {monogram(name, initials)}
    </span>
  );
}
