import type { CSSProperties, ReactNode } from 'react';

import { StatusBadge } from '../core/StatusBadge';
import type { StatusKey } from '../core/status';

/**
 * A single catalogued metric result. The whole point of this component is that a value and a
 * missing value look different: statuses never render as 0, blank or a dash.
 */
export interface MetricValueProps {
  /** The catalogued metric name, announced to assistive technology. */
  readonly name?: string;
  readonly label: ReactNode;
  readonly value?: ReactNode;
  readonly unit?: string;
  readonly status?: StatusKey;
  readonly delta?: string;
  readonly deltaDirection?: 'up' | 'down' | 'flat';
  readonly version?: string;
  readonly size?: 'lg' | 'md' | 'sm';
  readonly window?: string;
  readonly note?: string;
  readonly onOpen?: () => void;
  readonly style?: CSSProperties;
}

export function MetricValue({
  name,
  label,
  value,
  unit,
  status = 'available',
  delta,
  deltaDirection,
  version,
  size = 'md',
  window: observationWindow,
  note,
  onOpen,
  style,
}: MetricValueProps) {
  const missing = status !== 'available' && status !== 'ready';
  const numberFont =
    size === 'lg'
      ? 'var(--type-metric-lg)'
      : size === 'sm'
        ? 'var(--type-body-strong)'
        : 'var(--type-metric)';
  const deltaColor =
    deltaDirection === 'up'
      ? 'var(--positive-fg)'
      : deltaDirection === 'down'
        ? 'var(--critical-fg)'
        : 'var(--text-secondary)';

  const body = (
    <>
      <div
        style={{
          display: 'flex',
          alignItems: 'baseline',
          gap: 'var(--space-05)',
          flexWrap: 'wrap',
          minWidth: 0,
        }}
      >
        <span
          style={{
            font: 'var(--type-table-head)',
            color: 'var(--text-secondary)',
            fontWeight: 'var(--weight-medium)',
          }}
        >
          {label}
        </span>
        {version ? (
          <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>
            {version}
          </span>
        ) : null}
      </div>
      {missing ? (
        <StatusBadge status={status} detail={note} size={size === 'sm' ? 'sm' : 'md'} />
      ) : (
        <div
          style={{
            display: 'flex',
            alignItems: 'baseline',
            gap: 'var(--space-075)',
            flexWrap: 'wrap',
          }}
        >
          <span
            style={{
              font: numberFont,
              color: 'var(--text-primary)',
              fontVariantNumeric: 'tabular-nums',
            }}
          >
            {value}
          </span>
          {unit ? (
            <span style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
              {unit}
            </span>
          ) : null}
          {delta ? (
            <span
              style={{
                font: 'var(--type-caption)',
                color: deltaColor,
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              {delta}
            </span>
          ) : null}
        </div>
      )}
      {observationWindow ? (
        <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>
          {observationWindow}
        </span>
      ) : null}
      {name ? <span className="opi-vh">{name}</span> : null}
    </>
  );

  if (!onOpen) return <div style={{ display: 'grid', gap: 3, minWidth: 0, ...style }}>{body}</div>;

  return (
    <button
      type="button"
      onClick={onOpen}
      className="opi-item"
      style={{
        display: 'grid',
        gap: 3,
        minWidth: 0,
        textAlign: 'left',
        border: 0,
        background: 'transparent',
        padding: 'var(--space-075)',
        margin: 'calc(var(--space-075) * -1)',
        borderRadius: 'var(--radius-xs)',
        cursor: 'pointer',
        ...style,
      }}
    >
      {body}
    </button>
  );
}
