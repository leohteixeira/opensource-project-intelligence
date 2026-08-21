import type { CSSProperties, ReactNode } from 'react';

/**
 * Job progress. Two honest shapes only: a known completed/total, or an indeterminate bar labelled
 * "Total unknown". Progress is never invented from elapsed time.
 */
export interface ProgressProps {
  readonly completed?: number;
  readonly total?: number;
  readonly unit?: string;
  readonly label?: ReactNode;
  readonly indeterminate?: boolean;
  readonly tone?: 'info' | 'positive' | 'critical';
  readonly size?: 'md' | 'sm';
  readonly style?: CSSProperties;
}

export function Progress({
  completed,
  total,
  unit = 'items',
  label,
  indeterminate,
  tone = 'info',
  size = 'md',
  style,
}: ProgressProps) {
  const known =
    !indeterminate && typeof total === 'number' && total > 0 && typeof completed === 'number';
  const pct = known ? Math.min(100, Math.round((completed / total) * 100)) : null;
  const fill =
    tone === 'positive'
      ? 'var(--positive-solid)'
      : tone === 'critical'
        ? 'var(--critical-solid)'
        : 'var(--info-solid)';
  const height = size === 'sm' ? 3 : 5;
  const readout = known ? `${completed} / ${total} ${unit}` : 'Total unknown';

  return (
    <div style={{ display: 'grid', gap: 'var(--space-05)', minWidth: 140, ...style }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          gap: 'var(--space-1)',
          font: 'var(--type-caption)',
          color: 'var(--text-secondary)',
        }}
      >
        <span>{label}</span>
        <span style={{ fontVariantNumeric: 'tabular-nums', color: 'var(--text-body)' }}>
          {readout}
        </span>
      </div>
      <div
        role="progressbar"
        aria-valuemin={known ? 0 : undefined}
        aria-valuemax={known ? total : undefined}
        aria-valuenow={known ? completed : undefined}
        aria-valuetext={readout}
        aria-label={typeof label === 'string' ? label : 'Progress'}
        style={{
          position: 'relative',
          height,
          background: 'var(--gray-100)',
          borderRadius: 'var(--radius-pill)',
          overflow: 'hidden',
        }}
      >
        {known ? (
          <div
            style={{
              width: `${pct}%`,
              height: '100%',
              background: fill,
              borderRadius: 'inherit',
              transition: 'width var(--dur-slow) var(--ease-productive)',
            }}
          />
        ) : (
          <div
            className="opi-indeterminate"
            style={{
              position: 'absolute',
              inset: 0,
              width: '33%',
              background: fill,
              borderRadius: 'inherit',
            }}
          />
        )}
      </div>
    </div>
  );
}
