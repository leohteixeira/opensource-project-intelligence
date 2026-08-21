import type { CSSProperties, ReactNode } from 'react';

import { Icon } from '../core/Icon';
import type { IconName } from '../core/icons';

export interface TabItem {
  readonly value: string;
  readonly label: string;
  readonly icon?: IconName;
  readonly count?: number;
}

/**
 * Horizontal tab list, in two shapes. "underline" is the page-level tab set — project detail has
 * nine tabs, so the list scrolls horizontally and never becomes a second nested rail. "pill" is
 * the segmented filter set that sits above a collection and usually carries counts.
 */
export interface TabsProps {
  readonly items: readonly TabItem[];
  readonly value: string;
  readonly onChange?: (value: string) => void;
  readonly overflow?: ReactNode;
  readonly size?: 'md' | 'sm';
  readonly variant?: 'underline' | 'pill';
  readonly style?: CSSProperties;
}

export function Tabs({
  items,
  value,
  onChange,
  overflow,
  size = 'md',
  variant = 'underline',
  style,
}: TabsProps) {
  const pill = variant === 'pill';

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 'var(--space-1)',
        borderBottom: pill ? 'none' : 'var(--border-default)',
        background: pill ? 'var(--surface-sunken)' : 'transparent',
        borderRadius: pill ? 'var(--radius-md)' : 0,
        padding: pill ? 'var(--space-05)' : 0,
        ...style,
      }}
    >
      <div
        role="tablist"
        aria-orientation="horizontal"
        style={{
          display: 'flex',
          gap: 2,
          overflowX: 'auto',
          scrollbarWidth: 'thin',
          flex: 1,
          minWidth: 0,
        }}
      >
        {items.map((tab) => {
          const on = tab.value === value;

          return (
            <button
              key={tab.value}
              role="tab"
              aria-selected={on}
              tabIndex={on ? 0 : -1}
              type="button"
              onClick={() => onChange?.(tab.value)}
              className={pill ? 'opi-seg' : 'opi-tab'}
              style={
                pill
                  ? undefined
                  : {
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: 'var(--space-05)',
                      border: 0,
                      background: 'transparent',
                      padding:
                        size === 'sm'
                          ? 'var(--space-075) var(--space-1)'
                          : 'var(--space-1) var(--space-15)',
                      minHeight: size === 'sm' ? 36 : 42,
                      whiteSpace: 'nowrap',
                      font: 'var(--type-ui)',
                      fontWeight: on ? 'var(--weight-semibold)' : 'var(--weight-medium)',
                      color: on ? 'var(--text-primary)' : 'var(--text-secondary)',
                      boxShadow: on ? 'inset 0 -2px 0 var(--blue-500)' : 'none',
                    }
              }
            >
              {tab.icon ? <Icon name={tab.icon} size={14} /> : null}
              <span>{tab.label}</span>
              {typeof tab.count === 'number' ? (
                <span
                  style={{
                    font: 'var(--type-caption)',
                    color: on ? 'var(--text-secondary)' : 'var(--text-tertiary)',
                    fontVariantNumeric: 'tabular-nums',
                  }}
                >
                  {pill ? `· ${tab.count}` : tab.count}
                </span>
              ) : null}
            </button>
          );
        })}
      </div>
      {overflow ? <div style={{ display: 'flex', alignItems: 'center' }}>{overflow}</div> : null}
    </div>
  );
}
