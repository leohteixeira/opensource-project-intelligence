import type { CSSProperties, ReactNode } from 'react';

import { Icon } from '../core/Icon';

export interface TableColumn<Row> {
  readonly key: string;
  readonly header: ReactNode;
  readonly render?: (row: Row) => ReactNode;
  readonly mono?: boolean;
  readonly numeric?: boolean;
  readonly wrap?: boolean;
  readonly width?: number | string;
  readonly sort?: 'asc' | 'desc';
}

/**
 * Compact evidence table. Numeric columns are right-aligned and tabular. On narrow screens the
 * same rows render as a row-detail list instead of shrinking type or dropping columns.
 */
export interface TableProps<Row> {
  readonly columns: readonly TableColumn<Row>[];
  readonly rows: readonly Row[];
  readonly caption?: string;
  readonly getRowKey?: (row: Row, index: number) => string;
  readonly onRowClick?: (row: Row) => void;
  readonly selectedKey?: string;
  readonly density?: 'default' | 'compact';
  readonly layout?: 'table' | 'detail';
  readonly empty?: ReactNode;
  readonly footer?: ReactNode;
  readonly style?: CSSProperties;
}

function cellValue<Row>(column: TableColumn<Row>, row: Row): ReactNode {
  if (column.render) return column.render(row);

  const value = (row as Record<string, unknown>)[column.key];

  return typeof value === 'string' || typeof value === 'number' ? value : null;
}

export function Table<Row>({
  columns,
  rows,
  caption,
  getRowKey,
  onRowClick,
  selectedKey,
  density = 'default',
  layout = 'table',
  empty,
  footer,
  style,
}: TableProps<Row>) {
  const rowHeight = density === 'compact' ? 'var(--row-h-compact)' : 'var(--row-h-default)';
  const keyOf = (row: Row, index: number): string => {
    if (getRowKey) return getRowKey(row, index);

    const id = (row as { id?: unknown }).id;

    return typeof id === 'string' ? id : String(index);
  };

  if (!rows.length && empty) return <div style={style}>{empty}</div>;

  if (layout === 'detail') {
    return (
      <ul style={{ display: 'grid', gap: 'var(--space-1)', ...style }}>
        {rows.map((row, index) => (
          <li
            key={keyOf(row, index)}
            className="opi-row"
            onClick={onRowClick ? () => onRowClick(row) : undefined}
            style={{
              border: 'var(--border-default)',
              borderRadius: 'var(--radius-md)',
              boxShadow: 'var(--shadow-card)',
              background: 'var(--surface-card)',
              padding: 'var(--space-2)',
              display: 'grid',
              gap: 'var(--space-075)',
              cursor: onRowClick ? 'pointer' : 'default',
            }}
          >
            {columns.map((column) => (
              <div
                key={column.key}
                style={{
                  display: 'flex',
                  gap: 'var(--space-1)',
                  justifyContent: 'space-between',
                  alignItems: 'baseline',
                }}
              >
                <span style={{ font: 'var(--type-table-head)', color: 'var(--text-secondary)' }}>
                  {column.header}
                </span>
                <span
                  style={{
                    font: 'var(--type-table)',
                    color: 'var(--text-body)',
                    textAlign: 'right',
                    fontVariantNumeric: column.numeric ? 'tabular-nums' : undefined,
                  }}
                >
                  {cellValue(column, row)}
                </span>
              </div>
            ))}
          </li>
        ))}
      </ul>
    );
  }

  return (
    <div
      className="opi-card"
      tabIndex={0}
      aria-label={caption ? `${caption} table region` : 'Scrollable table region'}
      style={{
        overflowX: 'auto',
        border: 'var(--border-default)',
        borderRadius: 'var(--radius-md)',
        boxShadow: 'var(--shadow-card)',
        background: 'var(--surface-card)',
        ...style,
      }}
    >
      <table>
        {caption ? <caption className="opi-vh">{caption}</caption> : null}
        <thead>
          <tr>
            {columns.map((column) => (
              <th
                key={column.key}
                scope="col"
                aria-sort={
                  column.sort ? (column.sort === 'asc' ? 'ascending' : 'descending') : undefined
                }
                style={{
                  position: 'sticky',
                  top: 0,
                  background: 'var(--surface-table-head)',
                  textAlign: column.numeric ? 'right' : 'left',
                  font: 'var(--type-table-head)',
                  color: 'var(--text-secondary)',
                  whiteSpace: 'nowrap',
                  padding: 'var(--space-1) var(--cell-pad-x)',
                  height: 44,
                  borderBottom: 'var(--border-default)',
                  width: column.width,
                }}
              >
                <span
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: 4,
                    justifyContent: column.numeric ? 'flex-end' : 'flex-start',
                  }}
                >
                  {column.header}
                  {column.sort ? (
                    <Icon name={column.sort === 'asc' ? 'arrow-up' : 'arrow-down'} size={12} />
                  ) : null}
                </span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, index) => {
            const key = keyOf(row, index);
            const selected = selectedKey !== undefined && selectedKey === key;

            return (
              <tr
                key={key}
                className="opi-row"
                onClick={onRowClick ? () => onRowClick(row) : undefined}
                aria-selected={selected || undefined}
                style={{
                  background: selected ? 'var(--surface-row-selected)' : undefined,
                  cursor: onRowClick ? 'pointer' : 'default',
                }}
              >
                {columns.map((column) => (
                  <td
                    key={column.key}
                    style={{
                      height: rowHeight,
                      padding: 'var(--cell-pad-y) var(--cell-pad-x)',
                      borderBottom:
                        index === rows.length - 1 ? 'none' : '1px solid var(--border-table)',
                      font: column.mono ? 'var(--type-mono-xs)' : 'var(--type-table)',
                      color: 'var(--text-body)',
                      textAlign: column.numeric ? 'right' : 'left',
                      fontVariantNumeric: column.numeric ? 'tabular-nums' : undefined,
                      whiteSpace: column.wrap ? 'normal' : 'nowrap',
                    }}
                  >
                    {cellValue(column, row)}
                  </td>
                ))}
              </tr>
            );
          })}
        </tbody>
        {footer ? (
          <tfoot>
            <tr>
              <td
                colSpan={columns.length}
                style={{
                  padding: 'var(--space-1) var(--cell-pad-x)',
                  borderTop: 'var(--border-default)',
                  font: 'var(--type-caption)',
                  color: 'var(--text-secondary)',
                }}
              >
                {footer}
              </td>
            </tr>
          </tfoot>
        ) : null}
      </table>
    </div>
  );
}
