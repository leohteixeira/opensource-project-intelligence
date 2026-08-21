import type { CSSProperties } from 'react';

import { Icon } from '../core/Icon';

export interface CoverageSource {
  readonly name: string;
  readonly value: string;
}

/**
 * Actual coverage, never the requested range. Every intelligence result carries one of these so a
 * thin evidence base is visible instead of implied.
 */
export interface CoverageDisclosureProps {
  readonly requested?: string;
  readonly actual?: string;
  readonly ratio?: number;
  readonly sources?: readonly CoverageSource[];
  readonly missing?: readonly string[];
  readonly cutoff?: string;
  readonly style?: CSSProperties;
}

export function CoverageDisclosure({
  requested,
  actual,
  ratio,
  sources = [],
  missing = [],
  cutoff,
  style,
}: CoverageDisclosureProps) {
  const pct = typeof ratio === 'number' ? Math.round(ratio * 100) : null;
  const tone =
    pct === null
      ? 'var(--coverage-none)'
      : pct >= 95
        ? 'var(--coverage-full)'
        : pct >= 60
          ? 'var(--coverage-partial)'
          : 'var(--critical-solid)';

  return (
    <div style={{ display: 'grid', gap: 'var(--space-075)', ...style }}>
      <div
        style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-1)', flexWrap: 'wrap' }}
      >
        <span
          style={{
            font: 'var(--type-table-head)',
            color: 'var(--text-secondary)',
            fontWeight: 'var(--weight-medium)',
          }}
        >
          Coverage
        </span>
        <span
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 'var(--space-05)',
            font: 'var(--type-mono-xs)',
            color: 'var(--text-body)',
          }}
        >
          <span
            aria-hidden="true"
            style={{
              display: 'inline-block',
              width: 44,
              height: 4,
              borderRadius: 'var(--radius-pill)',
              background: 'var(--gray-100)',
              position: 'relative',
              overflow: 'hidden',
            }}
          >
            <span
              style={{ position: 'absolute', inset: 0, width: `${pct ?? 0}%`, background: tone }}
            />
          </span>
          <span>
            {actual}
            {requested && actual !== requested ? ` of ${requested} requested` : ''}
          </span>
        </span>
      </div>
      {sources.length ? (
        <ul
          style={{
            display: 'flex',
            gap: 'var(--space-1)',
            flexWrap: 'wrap',
            font: 'var(--type-mono-xs)',
            color: 'var(--text-secondary)',
          }}
        >
          {sources.map((source) => (
            <li key={source.name} style={{ display: 'flex', gap: 3, alignItems: 'center' }}>
              <Icon name="check" size={11} style={{ color: 'var(--positive-fg)' }} />
              {source.name} {source.value}
            </li>
          ))}
        </ul>
      ) : null}
      {missing.length ? (
        <ul
          style={{
            display: 'flex',
            gap: 'var(--space-1)',
            flexWrap: 'wrap',
            font: 'var(--type-mono-xs)',
            color: 'var(--attention-fg)',
          }}
        >
          {missing.map((item) => (
            <li key={item} style={{ display: 'flex', gap: 3, alignItems: 'center' }}>
              <Icon name="circle-dashed" size={11} />
              {item}
            </li>
          ))}
        </ul>
      ) : null}
      {cutoff ? (
        <p style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>
          cutoff {cutoff}
        </p>
      ) : null}
    </div>
  );
}
