import { useEffect, type CSSProperties, type ReactNode } from 'react';

import { IconButton } from '../core/IconButton';

/**
 * Side sheet for one level of disclosure — the metric detail, an evidence record, a run version.
 * Never a drawer opened from a drawer: the depth contract allows one level only.
 */
export interface DrawerProps {
  readonly open?: boolean;
  readonly title: string;
  readonly eyebrow?: ReactNode;
  readonly children?: ReactNode;
  readonly footer?: ReactNode;
  readonly onClose?: () => void;
  readonly side?: 'right' | 'left';
  readonly width?: string;
  readonly style?: CSSProperties;
}

export function Drawer({
  open = true,
  title,
  eyebrow,
  children,
  footer,
  onClose,
  side = 'right',
  width = 'var(--drawer-w)',
  style,
}: DrawerProps) {
  useEffect(() => {
    if (!open) return;

    const escape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose?.();
    };

    document.addEventListener('keydown', escape);

    return () => document.removeEventListener('keydown', escape);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 'var(--z-drawer)',
        display: 'flex',
        justifyContent: side === 'right' ? 'flex-end' : 'flex-start',
        background: 'var(--scrim)',
      }}
    >
      <aside
        role="dialog"
        aria-modal="true"
        aria-label={title}
        style={{
          width: '100%',
          maxWidth: width,
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          background: 'var(--surface-overlay)',
          borderLeft: side === 'right' ? 'var(--border-default)' : undefined,
          borderRight: side === 'left' ? 'var(--border-default)' : undefined,
          boxShadow: 'var(--shadow-overlay)',
          ...style,
        }}
      >
        <header
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: 'var(--space-1)',
            padding: 'var(--space-2)',
            borderBottom: 'var(--border-hairline)',
          }}
        >
          <div style={{ flex: 1, minWidth: 0, display: 'grid', gap: 2 }}>
            {eyebrow ? (
              <p
                style={{
                  font: 'var(--type-eyebrow)',
                  letterSpacing: 'var(--tracking-eyebrow)',
                  textTransform: 'uppercase',
                  color: 'var(--text-tertiary)',
                }}
              >
                {eyebrow}
              </p>
            ) : null}
            <h2 style={{ font: 'var(--type-section)' }}>{title}</h2>
          </div>
          {onClose ? <IconButton icon="x" label="Close" onClick={onClose} /> : null}
        </header>
        <div
          style={{
            flex: 1,
            overflow: 'auto',
            padding: 'var(--space-2)',
            display: 'grid',
            gap: 'var(--space-2)',
            alignContent: 'start',
          }}
        >
          {children}
        </div>
        {footer ? (
          <footer
            style={{
              display: 'flex',
              justifyContent: 'flex-end',
              gap: 'var(--space-1)',
              padding: 'var(--space-15) var(--space-2)',
              borderTop: 'var(--border-hairline)',
            }}
          >
            {footer}
          </footer>
        ) : null}
      </aside>
    </div>
  );
}
