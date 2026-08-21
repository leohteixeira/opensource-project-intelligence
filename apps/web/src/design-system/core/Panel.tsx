import type { CSSProperties, MouseEventHandler, ReactNode } from 'react';

/**
 * Section container — the card. White surface, hairline border, one quiet shadow. One level deep,
 * never a panel inside a panel: the depth contract allows a page of sections plus one disclosure.
 * The header carries the panel's own state, so a partial failure stays local and the rest of the
 * page keeps its valid information.
 */
export interface PanelProps {
  readonly title?: ReactNode;
  readonly eyebrow?: ReactNode;
  readonly status?: ReactNode;
  /** The mono metadata line: window · cutoff · definition version. */
  readonly meta?: ReactNode;
  readonly actions?: ReactNode;
  readonly children?: ReactNode;
  readonly footer?: ReactNode;
  readonly padding?: string;
  readonly tone?: 'default' | 'attention' | 'critical';
  readonly interactive?: boolean;
  readonly onClick?: MouseEventHandler<HTMLElement>;
  readonly style?: CSSProperties;
}

export function Panel({
  title,
  eyebrow,
  status,
  meta,
  actions,
  children,
  footer,
  padding = 'var(--space-2)',
  tone = 'default',
  interactive,
  onClick,
  style,
}: PanelProps) {
  const border =
    tone === 'critical'
      ? 'var(--critical-border)'
      : tone === 'attention'
        ? 'var(--attention-border)'
        : 'var(--border-card)';

  return (
    <section
      className={`opi-card${interactive ? ' opi-card--interactive' : ''}`}
      onClick={onClick}
      style={{
        background: 'var(--surface-card)',
        border: `1px solid ${border}`,
        borderRadius: 'var(--radius-md)',
        boxShadow: 'var(--shadow-card)',
        display: 'grid',
        minWidth: 0,
        ...style,
      }}
    >
      {title || actions ? (
        <header
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: 'var(--space-15)',
            padding: children
              ? `var(--space-2) ${padding} var(--space-15)`
              : `var(--space-2) ${padding}`,
            borderBottom: children ? 'var(--border-hairline)' : 'none',
          }}
        >
          <div style={{ flex: 1, minWidth: 0, display: 'grid', gap: 3 }}>
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
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 'var(--space-1)',
                flexWrap: 'wrap',
              }}
            >
              <h2 style={{ font: 'var(--type-subsection)' }}>{title}</h2>
              {status}
            </div>
            {meta ? (
              <p style={{ font: 'var(--type-mono-xs)', color: 'var(--text-secondary)' }}>{meta}</p>
            ) : null}
          </div>
          {actions ? (
            <div style={{ display: 'flex', gap: 'var(--space-05)', alignItems: 'center' }}>
              {actions}
            </div>
          ) : null}
        </header>
      ) : null}
      {children ? <div style={{ padding, minWidth: 0 }}>{children}</div> : null}
      {footer ? (
        <footer
          style={{
            padding: `var(--space-15) ${padding}`,
            borderTop: 'var(--border-hairline)',
            background: 'var(--surface-sunken)',
            borderRadius: '0 0 var(--radius-md) var(--radius-md)',
            font: 'var(--type-caption)',
            color: 'var(--text-secondary)',
            display: 'flex',
            gap: 'var(--space-15)',
            flexWrap: 'wrap',
            alignItems: 'center',
          }}
        >
          {footer}
        </footer>
      ) : null}
    </section>
  );
}
