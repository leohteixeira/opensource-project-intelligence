import type { CSSProperties } from 'react';

/**
 * Loading placeholder. Shaped like the content it replaces so layout does not jump, and hidden
 * from assistive technology rather than announced as data.
 */
export interface SkeletonProps {
  readonly width?: number | string;
  readonly height?: number | string;
  readonly radius?: string;
  readonly lines?: number;
  readonly gap?: string;
  readonly style?: CSSProperties;
}

export function Skeleton({
  width = '100%',
  height = 12,
  radius = 'var(--radius-xs)',
  lines = 1,
  gap = 'var(--space-1)',
  style,
}: SkeletonProps) {
  const bar = (barWidth: number | string, key: number) => (
    <span
      key={key}
      className="opi-skeleton"
      style={{
        display: 'block',
        width: barWidth,
        height,
        background: 'var(--gray-100)',
        borderRadius: radius,
      }}
    />
  );

  if (lines <= 1) {
    return (
      <span aria-hidden="true" style={{ display: 'block', ...style }}>
        {bar(width, 0)}
      </span>
    );
  }

  return (
    <span aria-hidden="true" style={{ display: 'grid', gap, ...style }}>
      {Array.from({ length: lines }, (_, index) => bar(index === lines - 1 ? '62%' : width, index))}
    </span>
  );
}
