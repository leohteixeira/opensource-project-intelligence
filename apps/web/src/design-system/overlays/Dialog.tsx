import { useEffect, useRef, type CSSProperties, type ReactNode } from 'react';

import { Icon } from '../core/Icon';
import { IconButton } from '../core/IconButton';

/**
 * Modal. Focus moves into the dialog while it is open and Escape closes it. Destructive variants
 * name the resource, list the irreversible effects, and require the exact typed confirmation
 * before the primary action becomes available.
 */
export interface DialogProps {
  readonly open?: boolean;
  readonly title: ReactNode;
  readonly children?: ReactNode;
  readonly footer?: ReactNode;
  readonly onClose?: () => void;
  readonly size?: 'sm' | 'md' | 'lg';
  readonly tone?: 'default' | 'danger';
  readonly labelledBy?: string;
  readonly style?: CSSProperties;
}

export function Dialog({
  open = true,
  title,
  children,
  footer,
  onClose,
  size = 'md',
  tone = 'default',
  labelledBy,
  style,
}: DialogProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;

    const escape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose?.();
    };

    document.addEventListener('keydown', escape);
    ref.current?.querySelector<HTMLElement>('button,[href],input,select,textarea')?.focus();

    return () => document.removeEventListener('keydown', escape);
  }, [open, onClose]);

  if (!open) return null;

  const width = size === 'sm' ? 400 : size === 'lg' ? 680 : 520;
  const titleId = labelledBy ?? 'opi-dialog-title';

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 'var(--z-dialog)',
        display: 'grid',
        placeItems: 'center',
        padding: 'var(--space-2)',
        background: 'var(--scrim)',
      }}
    >
      <div
        ref={ref}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        style={{
          width: '100%',
          maxWidth: width,
          maxHeight: '88vh',
          overflow: 'auto',
          background: 'var(--surface-overlay)',
          border: tone === 'danger' ? '1px solid var(--critical-border)' : 'var(--border-default)',
          borderRadius: 'var(--radius-md)',
          boxShadow: 'var(--shadow-overlay)',
          ...style,
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: 'var(--space-1)',
            padding: 'var(--space-2) var(--space-2) var(--space-1)',
          }}
        >
          {tone === 'danger' ? (
            <Icon
              name="triangle-alert"
              size={19}
              style={{ color: 'var(--critical-fg)', marginTop: 2 }}
            />
          ) : null}
          <h2 id={titleId} style={{ flex: 1, font: 'var(--type-section)' }}>
            {title}
          </h2>
          {onClose ? <IconButton icon="x" label="Close" size="sm" onClick={onClose} /> : null}
        </div>
        <div
          style={{
            padding: '0 var(--space-2) var(--space-2)',
            display: 'grid',
            gap: 'var(--space-15)',
            font: 'var(--type-body)',
            fontSize: 'var(--text-sm)',
          }}
        >
          {children}
        </div>
        {footer ? (
          <div
            style={{
              display: 'flex',
              justifyContent: 'flex-end',
              gap: 'var(--space-1)',
              padding: 'var(--space-15) var(--space-2)',
              borderTop: 'var(--border-hairline)',
              background: 'var(--surface-sunken)',
              borderRadius: '0 0 var(--radius-md) var(--radius-md)',
              flexWrap: 'wrap',
            }}
          >
            {footer}
          </div>
        ) : null}
      </div>
    </div>
  );
}
