import type { CSSProperties, ReactNode } from 'react';

import { Icon } from '../core/Icon';

const PRESETS = [
  { value: '30d', label: '30d' },
  { value: '90d', label: '90d' },
  { value: '180d', label: '180d' },
  { value: '365d', label: '365d' },
  { value: 'custom', label: 'Custom' },
] as const;

/**
 * The window control. Every intelligence surface is bound to one window and one cutoff, so the
 * preset row and the custom interval are one control, and the resolved interval and cutoff are
 * always printed underneath.
 */
export interface DateRangeFieldProps {
  readonly legend?: ReactNode;
  readonly value?: string;
  readonly from?: string;
  readonly to?: string;
  readonly cutoff?: string;
  readonly coverage?: string;
  readonly onChange?: (value: string) => void;
  readonly onCustomChange?: (bound: 'from' | 'to', value: string) => void;
  readonly disabled?: boolean;
  readonly style?: CSSProperties;
}

export function DateRangeField({
  legend = 'Observation window',
  value = '90d',
  from,
  to,
  cutoff,
  coverage,
  onChange,
  onCustomChange,
  disabled,
  style,
}: DateRangeFieldProps) {
  return (
    <fieldset
      style={{
        border: 0,
        margin: 0,
        padding: 0,
        display: 'grid',
        gap: 'var(--space-1)',
        minWidth: 0,
        ...style,
      }}
    >
      <legend
        style={{
          font: 'var(--type-ui)',
          color: 'var(--text-primary)',
          padding: 0,
          marginBottom: 'var(--space-05)',
        }}
      >
        {legend}
      </legend>
      <div role="group" style={{ display: 'flex', gap: 'var(--space-05)', flexWrap: 'wrap' }}>
        {PRESETS.map((preset) => {
          const on = value === preset.value;

          return (
            <button
              key={preset.value}
              type="button"
              onClick={() => onChange?.(preset.value)}
              disabled={disabled}
              aria-pressed={on}
              className="opi-btn"
              style={{
                height: 'var(--control-h-md)',
                padding: '0 var(--space-15)',
                background: on ? 'var(--ink)' : 'var(--surface-card)',
                color: on ? 'var(--action-primary-fg)' : 'var(--action-secondary-fg)',
                borderColor: on ? 'var(--ink)' : 'var(--action-secondary-border)',
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              {preset.label}
            </button>
          );
        })}
      </div>
      {value === 'custom' ? (
        <div
          style={{
            display: 'flex',
            gap: 'var(--space-1)',
            alignItems: 'flex-end',
            flexWrap: 'wrap',
          }}
        >
          <label
            style={{
              display: 'grid',
              gap: 2,
              font: 'var(--type-caption)',
              color: 'var(--text-secondary)',
            }}
          >
            From
            <input
              className="opi-field"
              type="date"
              value={from ?? ''}
              onChange={(event) => onCustomChange?.('from', event.target.value)}
              style={{
                height: 'var(--control-h-md)',
                padding: '0 var(--space-1)',
                font: 'var(--type-mono)',
                width: 156,
              }}
            />
          </label>
          <label
            style={{
              display: 'grid',
              gap: 2,
              font: 'var(--type-caption)',
              color: 'var(--text-secondary)',
            }}
          >
            To
            <input
              className="opi-field"
              type="date"
              value={to ?? ''}
              onChange={(event) => onCustomChange?.('to', event.target.value)}
              style={{
                height: 'var(--control-h-md)',
                padding: '0 var(--space-1)',
                font: 'var(--type-mono)',
                width: 156,
              }}
            />
          </label>
        </div>
      ) : null}
      <p
        style={{
          display: 'flex',
          gap: 'var(--space-075)',
          alignItems: 'center',
          flexWrap: 'wrap',
          font: 'var(--type-mono-xs)',
          color: 'var(--text-secondary)',
        }}
      >
        <Icon name="calendar" size={12} />
        <span>
          {from} to {to}
        </span>
        <span aria-hidden="true">·</span>
        <span>cutoff {cutoff}</span>
        {coverage ? (
          <>
            <span aria-hidden="true">·</span>
            <span>coverage {coverage}</span>
          </>
        ) : null}
      </p>
    </fieldset>
  );
}
