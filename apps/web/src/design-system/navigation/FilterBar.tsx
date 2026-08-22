import type { CSSProperties, ReactNode } from 'react';

import { Icon } from '../core/Icon';

export interface AppliedFilter {
  readonly key: string;
  readonly field: string;
  readonly value: string;
}

/**
 * Filter strip above a list or table. Everything in it is URL state, so the applied-filter chips
 * are the readable form of the query string and each one can be removed individually.
 */
export interface FilterBarProps {
  readonly children?: ReactNode;
  readonly applied?: readonly AppliedFilter[];
  readonly onClear?: () => void;
  readonly onRemove?: (key: string) => void;
  readonly resultLabel?: ReactNode;
  readonly clearLabel?: string;
  readonly removeLabel?: (filter: AppliedFilter) => string;
  readonly style?: CSSProperties;
}

export function FilterBar({
  children,
  applied = [],
  onClear,
  onRemove,
  resultLabel,
  clearLabel = 'Clear all',
  removeLabel = (filter) => `Remove filter ${filter.field} ${filter.value}`,
  style,
}: FilterBarProps) {
  return (
    <div style={{ display: 'grid', gap: 'var(--space-1)', ...style }}>
      <div
        style={{ display: 'flex', gap: 'var(--space-1)', flexWrap: 'wrap', alignItems: 'flex-end' }}
      >
        {children}
      </div>
      {applied.length || resultLabel ? (
        <div
          style={{
            display: 'flex',
            gap: 'var(--space-05)',
            flexWrap: 'wrap',
            alignItems: 'center',
          }}
        >
          {resultLabel ? (
            <span
              style={{
                font: 'var(--type-caption)',
                color: 'var(--text-secondary)',
                marginRight: 'var(--space-05)',
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              {resultLabel}
            </span>
          ) : null}
          {applied.map((filter) => (
            <span
              key={filter.key}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 4,
                padding: '2px var(--space-05) 2px var(--space-075)',
                background: 'var(--surface-sunken)',
                border: 'var(--border-default)',
                borderRadius: 'var(--radius-pill)',
                font: 'var(--type-caption)',
                color: 'var(--text-body)',
              }}
            >
              <span style={{ color: 'var(--text-secondary)' }}>{filter.field}</span>
              <span style={{ fontWeight: 'var(--weight-medium)' }}>{filter.value}</span>
              <button
                type="button"
                onClick={() => onRemove?.(filter.key)}
                aria-label={removeLabel(filter)}
                style={{
                  display: 'inline-flex',
                  border: 0,
                  background: 'transparent',
                  padding: 2,
                  cursor: 'pointer',
                  color: 'var(--text-secondary)',
                  borderRadius: 'var(--radius-pill)',
                }}
              >
                <Icon name="x" size={12} />
              </button>
            </span>
          ))}
          {applied.length ? (
            <button
              type="button"
              onClick={onClear}
              style={{
                border: 0,
                background: 'transparent',
                padding: '2px var(--space-05)',
                font: 'var(--type-caption)',
                color: 'var(--text-link)',
                cursor: 'pointer',
                textDecoration: 'underline',
              }}
            >
              {clearLabel}
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
