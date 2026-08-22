import type { CSSProperties } from 'react';

import { Icon } from '../core/Icon';
import { StatusBadge } from '../core/StatusBadge';
import type { StatusKey } from '../core/status';

export interface HealthDimension {
  readonly name: string;
  readonly rubric?: string;
  readonly score?: number;
  readonly status?: StatusKey;
  readonly note?: string;
}

export interface OverallScore {
  readonly calculable: boolean;
  readonly score?: number;
  readonly version?: string;
  readonly reason?: string;
}

/**
 * Seven independently inspectable dimensions, plus the overall score as a secondary summary that
 * is present only when its declared evidence requirements are met. There is no traffic light for
 * the project as a whole, and an unavailable dimension never has its weight redistributed.
 */
const ORDER = [
  'Activity',
  'Community',
  'Maintenance',
  'Concentration',
  'Stability',
  'Security',
  'Adoption',
] as const;

export interface HealthDimensionsProps {
  readonly dimensions: readonly HealthDimension[];
  readonly overall?: OverallScore;
  readonly onOpen?: (dimension: HealthDimension) => void;
  readonly layout?: 'list' | 'grid';
  readonly style?: CSSProperties;
}

export function HealthDimensions({
  dimensions,
  overall,
  onOpen,
  layout = 'list',
  style,
}: HealthDimensionsProps) {
  const ordered = ORDER.map((name) =>
    dimensions.find((dimension) => dimension.name === name),
  ).filter((dimension): dimension is HealthDimension => dimension !== undefined);
  const list = ordered.length ? ordered : dimensions;
  const grid = layout === 'grid';

  return (
    <div style={{ display: 'grid', gap: 'var(--space-15)', ...style }}>
      <ul
        className={
          grid ? 'opi-health-dimensions opi-health-dimensions--grid' : 'opi-health-dimensions'
        }
        style={{
          display: 'grid',
          gap: grid ? 'var(--space-1)' : 0,
          gridTemplateColumns: grid ? 'repeat(auto-fit, minmax(210px, 1fr))' : '1fr',
          minWidth: 0,
          width: '100%',
        }}
      >
        {list.map((dimension, index) => {
          const status = dimension.status ?? 'available';
          const missing = status !== 'available' && status !== 'ready';
          const score = dimension.score ?? 0;

          return (
            <li
              key={dimension.name}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 'var(--space-15)',
                minHeight: 44,
                padding: grid ? 'var(--space-1)' : 'var(--space-075) 0',
                border: grid ? 'var(--border-hairline)' : 'none',
                borderRadius: grid ? 'var(--radius-xs)' : 0,
                borderBottom:
                  !grid && index < list.length - 1 ? '1px solid var(--border-table)' : undefined,
                minWidth: 0,
              }}
            >
              <span style={{ flex: 1, minWidth: 0, display: 'grid', gap: 1 }}>
                <span style={{ font: 'var(--type-ui)', color: 'var(--text-primary)' }}>
                  {dimension.name}
                </span>
                <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>
                  {dimension.rubric}
                </span>
              </span>
              {missing ? (
                <StatusBadge status={status} detail={dimension.note} size="sm" />
              ) : (
                <>
                  <span
                    aria-hidden="true"
                    style={{
                      width: 84,
                      height: 5,
                      background: 'var(--gray-100)',
                      borderRadius: 'var(--radius-pill)',
                      overflow: 'hidden',
                      flex: 'none',
                    }}
                  >
                    <span
                      style={{
                        display: 'block',
                        height: '100%',
                        width: `${Math.max(2, Math.round(score * 100))}%`,
                        background:
                          score >= 0.7
                            ? 'var(--positive-solid)'
                            : score >= 0.4
                              ? 'var(--attention-solid)'
                              : 'var(--critical-solid)',
                      }}
                    />
                  </span>
                  <span
                    style={{
                      width: 44,
                      textAlign: 'right',
                      font: 'var(--type-body-strong)',
                      fontVariantNumeric: 'tabular-nums',
                      color: 'var(--text-primary)',
                    }}
                  >
                    {Math.round(score * 100)}
                  </span>
                </>
              )}
              {onOpen ? (
                <button
                  type="button"
                  onClick={() => onOpen(dimension)}
                  aria-label={`Open ${dimension.name} evidence`}
                  style={{
                    border: 0,
                    background: 'transparent',
                    cursor: 'pointer',
                    color: 'var(--text-secondary)',
                    padding: 6,
                    borderRadius: 'var(--radius-xs)',
                  }}
                >
                  <Icon name="chevron-right" size={15} />
                </button>
              ) : null}
            </li>
          );
        })}
      </ul>
      <div
        style={{
          display: 'flex',
          gap: 'var(--space-15)',
          alignItems: 'center',
          padding: 'var(--space-1) var(--space-15)',
          background: 'var(--surface-sunken)',
          border: 'var(--border-hairline)',
          borderRadius: 'var(--radius-xs)',
          flexWrap: 'wrap',
        }}
      >
        <span
          style={{
            font: 'var(--type-table-head)',
            color: 'var(--text-secondary)',
            fontWeight: 'var(--weight-medium)',
          }}
        >
          Overall score
        </span>
        {overall?.calculable ? (
          <>
            <span
              style={{
                font: 'var(--type-metric)',
                fontSize: 'var(--text-lg)',
                fontVariantNumeric: 'tabular-nums',
                color: 'var(--text-primary)',
              }}
            >
              {overall.score}
            </span>
            <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-secondary)' }}>
              {overall.version} · equal weights · secondary summary
            </span>
          </>
        ) : (
          <StatusBadge
            status="insufficient_data"
            detail={overall?.reason ?? 'evidence requirements not met'}
            size="sm"
          />
        )}
      </div>
    </div>
  );
}
