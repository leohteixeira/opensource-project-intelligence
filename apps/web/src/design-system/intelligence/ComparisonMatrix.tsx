import type { CSSProperties } from 'react';

import { StatusBadge } from '../core/StatusBadge';
import type { StatusKey } from '../core/status';
import { bestCellIndex } from './ranking';

export interface MatrixCell {
  readonly value?: number;
  readonly display?: string;
  readonly status?: StatusKey;
  readonly label?: string;
}

export interface MatrixRow {
  readonly metric: string;
  readonly unit: string;
  readonly comparable: boolean;
  readonly betterIs?: 'higher' | 'lower';
  readonly cells: readonly MatrixCell[];
}

/**
 * Two to five projects over one identical window and cutoff. A missing, non-applicable or
 * incomparable cell renders as its status and is never ranked; only numerically comparable rows
 * get the best-value marker.
 */
export interface ComparisonMatrixProps {
  readonly projects: readonly string[];
  readonly rows: readonly MatrixRow[];
  readonly window?: string;
  readonly cutoff?: string;
  readonly style?: CSSProperties;
}

export function ComparisonMatrix({
  projects,
  rows,
  window: observationWindow,
  cutoff,
  style,
}: ComparisonMatrixProps) {
  return (
    <div
      style={{
        overflowX: 'auto',
        border: 'var(--border-default)',
        borderRadius: 'var(--radius-sm)',
        background: 'var(--surface-card)',
        ...style,
      }}
    >
      <table>
        <caption className="opi-vh">
          Comparison over {observationWindow} with cutoff {cutoff}
        </caption>
        <thead>
          <tr>
            <th
              scope="col"
              style={{
                textAlign: 'left',
                font: 'var(--type-table-head)',
                color: 'var(--text-secondary)',
                padding: 'var(--space-075) var(--cell-pad-x)',
                borderBottom: 'var(--border-default)',
                position: 'sticky',
                left: 0,
                background: 'var(--surface-table-head)',
                minWidth: 190,
              }}
            >
              Metric
            </th>
            {projects.map((project) => (
              <th
                key={project}
                scope="col"
                style={{
                  textAlign: 'right',
                  font: 'var(--type-table-head)',
                  color: 'var(--text-primary)',
                  padding: 'var(--space-075) var(--cell-pad-x)',
                  borderBottom: 'var(--border-default)',
                  background: 'var(--surface-table-head)',
                  whiteSpace: 'nowrap',
                }}
              >
                {project}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, rowIndex) => {
            const best = bestCellIndex(row, projects.length);
            const lastRow = rowIndex === rows.length - 1;

            return (
              <tr key={row.metric} className="opi-row">
                <th
                  scope="row"
                  style={{
                    textAlign: 'left',
                    padding: 'var(--cell-pad-y) var(--cell-pad-x)',
                    borderBottom: lastRow ? 'none' : '1px solid var(--border-table)',
                    position: 'sticky',
                    left: 0,
                    background: 'var(--surface-card)',
                    font: 'var(--type-mono-xs)',
                    color: 'var(--text-body)',
                    fontWeight: 'var(--weight-regular)',
                  }}
                >
                  <span style={{ display: 'grid', gap: 1 }}>
                    <span>{row.metric}</span>
                    <span style={{ color: 'var(--text-tertiary)' }}>
                      {row.unit}
                      {row.comparable ? '' : ' · not comparable'}
                    </span>
                  </span>
                </th>
                {projects.map((project, columnIndex) => {
                  const cell = row.cells[columnIndex] ?? { status: 'unknown' as StatusKey };
                  const isBest = best === columnIndex;

                  return (
                    <td
                      key={project}
                      style={{
                        textAlign: 'right',
                        height: 'var(--row-h-compact)',
                        padding: 'var(--cell-pad-y) var(--cell-pad-x)',
                        borderBottom: lastRow ? 'none' : '1px solid var(--border-table)',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {cell.status && cell.status !== 'available' ? (
                        <StatusBadge status={cell.status} size="sm" label={cell.label} />
                      ) : (
                        <span
                          style={{
                            display: 'inline-flex',
                            alignItems: 'baseline',
                            gap: 5,
                            justifyContent: 'flex-end',
                            font: 'var(--type-table)',
                            fontWeight: isBest ? 'var(--weight-semibold)' : 'var(--weight-regular)',
                            color: 'var(--text-primary)',
                            fontVariantNumeric: 'tabular-nums',
                          }}
                        >
                          {isBest ? (
                            <span
                              aria-label="best in this comparison"
                              style={{
                                width: 5,
                                height: 5,
                                borderRadius: 'var(--radius-pill)',
                                background: 'var(--positive-solid)',
                              }}
                            />
                          ) : null}
                          {cell.display ?? cell.value}
                        </span>
                      )}
                    </td>
                  );
                })}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
