import {
  cloneElement,
  useEffect,
  useRef,
  useState,
  type CSSProperties,
  type ReactElement,
} from 'react';

import { Icon } from '../core/Icon';
import type { IconName } from '../core/icons';
import { IconButton } from '../core/IconButton';

export interface MenuItem {
  readonly key?: string;
  readonly label?: string;
  readonly icon?: IconName;
  readonly onSelect?: (key: string | undefined) => void;
  readonly danger?: boolean;
  readonly disabled?: boolean;
  /** The reason a disabled item is unavailable — a restricted item is not passed in at all. */
  readonly hint?: string;
  readonly separator?: boolean;
}

interface TriggerProps {
  readonly onClick?: () => void;
  readonly 'aria-expanded'?: boolean;
  readonly 'aria-haspopup'?: 'menu';
}

/**
 * Popover action list. Items may be destructive or disabled-with-reason; permission-restricted
 * items are simply not passed in.
 */
export interface MenuProps {
  readonly trigger?: ReactElement<TriggerProps>;
  readonly triggerIcon?: IconName;
  readonly triggerLabel?: string;
  readonly items: readonly MenuItem[];
  readonly align?: 'start' | 'end';
  readonly onSelect?: (key: string | undefined) => void;
  readonly style?: CSSProperties;
}

export function Menu({
  trigger,
  triggerIcon = 'ellipsis-vertical',
  triggerLabel = 'Actions',
  items,
  align = 'end',
  onSelect,
  style,
}: MenuProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    if (!open) return;

    const away = (event: MouseEvent) => {
      if (ref.current && event.target instanceof Node && !ref.current.contains(event.target)) {
        setOpen(false);
      }
    };
    const escape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false);
    };

    document.addEventListener('mousedown', away);
    document.addEventListener('keydown', escape);

    return () => {
      document.removeEventListener('mousedown', away);
      document.removeEventListener('keydown', escape);
    };
  }, [open]);

  return (
    <span ref={ref} style={{ position: 'relative', display: 'inline-flex', ...style }}>
      {trigger ? (
        cloneElement(trigger, {
          onClick: () => setOpen(!open),
          'aria-expanded': open,
          'aria-haspopup': 'menu',
        })
      ) : (
        <IconButton
          icon={triggerIcon}
          label={triggerLabel}
          onClick={() => setOpen(!open)}
          selected={open}
          aria-expanded={open}
          aria-haspopup="menu"
        />
      )}
      {open ? (
        <div
          role="menu"
          style={{
            position: 'absolute',
            top: 'calc(100% + 4px)',
            right: align === 'end' ? 0 : undefined,
            left: align === 'start' ? 0 : undefined,
            zIndex: 'var(--z-popover)',
            minWidth: 216,
            padding: 'var(--space-05)',
            background: 'var(--surface-overlay)',
            border: 'var(--border-default)',
            borderRadius: 'var(--radius-sm)',
            boxShadow: 'var(--shadow-popover)',
          }}
        >
          {items.map((item, index) =>
            item.separator ? (
              <hr
                key={`separator-${index}`}
                style={{
                  border: 0,
                  borderTop: 'var(--border-hairline)',
                  margin: 'var(--space-05) 0',
                }}
              />
            ) : (
              <button
                key={item.key ?? item.label}
                role="menuitem"
                type="button"
                className="opi-item"
                disabled={item.disabled}
                onClick={() => {
                  setOpen(false);
                  (item.onSelect ?? onSelect)?.(item.key);
                }}
                style={{
                  width: '100%',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 'var(--space-1)',
                  minHeight: 'var(--control-h-md)',
                  padding: '0 var(--space-1)',
                  border: 0,
                  borderRadius: 'var(--radius-xs)',
                  background: 'transparent',
                  textAlign: 'left',
                  font: 'var(--type-ui)',
                  color: item.disabled
                    ? 'var(--text-disabled)'
                    : item.danger
                      ? 'var(--critical-fg)'
                      : 'var(--text-body)',
                  cursor: item.disabled ? 'not-allowed' : 'pointer',
                }}
              >
                {item.icon ? <Icon name={item.icon} size={15} /> : null}
                <span style={{ flex: 1 }}>{item.label}</span>
                {item.hint ? (
                  <span style={{ font: 'var(--type-caption)', color: 'var(--text-tertiary)' }}>
                    {item.hint}
                  </span>
                ) : null}
              </button>
            ),
          )}
        </div>
      ) : null}
    </span>
  );
}
