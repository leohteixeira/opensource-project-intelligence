import { useState, type CSSProperties, type ReactNode } from 'react';

/**
 * Tooltip. Supplementary only — never the sole carrier of a definition, a version or a reason,
 * because it is unreachable on touch and in print. Opens on hover and on focus.
 */
export interface TooltipProps {
  readonly children: ReactNode;
  readonly content: ReactNode;
  readonly placement?: 'top' | 'bottom' | 'right';
  readonly style?: CSSProperties;
}

const PLACEMENTS: Record<'top' | 'bottom' | 'right', CSSProperties> = {
  bottom: { top: 'calc(100% + 6px)', left: '50%', transform: 'translateX(-50%)' },
  right: { left: 'calc(100% + 6px)', top: '50%', transform: 'translateY(-50%)' },
  top: { bottom: 'calc(100% + 6px)', left: '50%', transform: 'translateX(-50%)' },
};

export function Tooltip({ children, content, placement = 'top', style }: TooltipProps) {
  const [open, setOpen] = useState(false);

  return (
    <span
      style={{ position: 'relative', display: 'inline-flex', ...style }}
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => setOpen(true)}
      onBlur={() => setOpen(false)}
    >
      {children}
      {open ? (
        <span
          role="tooltip"
          style={{
            position: 'absolute',
            ...PLACEMENTS[placement],
            zIndex: 'var(--z-popover)',
            background: 'var(--surface-inverse)',
            color: 'var(--text-inverse)',
            font: 'var(--type-caption)',
            padding: 'var(--space-05) var(--space-1)',
            borderRadius: 'var(--radius-xs)',
            boxShadow: 'var(--shadow-popover)',
            maxWidth: 260,
            width: 'max-content',
            textAlign: 'left',
            pointerEvents: 'none',
          }}
        >
          {content}
        </span>
      ) : null}
    </span>
  );
}
