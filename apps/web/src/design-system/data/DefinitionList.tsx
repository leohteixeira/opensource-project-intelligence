import type { CSSProperties, ReactNode } from 'react';

export interface DefinitionItem {
  readonly label: string;
  readonly value: ReactNode;
  readonly mono?: boolean;
}

/**
 * Label/value pairs. This is the product's provenance workhorse: formula, unit, window, cutoff,
 * version, coverage, contributing sources, model and prompt version all render here, because a
 * decision must never depend on a tooltip.
 */
export interface DefinitionListProps {
  readonly items: readonly DefinitionItem[];
  readonly columns?: number;
  readonly dense?: boolean;
  readonly mono?: boolean;
  readonly style?: CSSProperties;
}

export function DefinitionList({ items, columns = 1, dense, mono, style }: DefinitionListProps) {
  return (
    <dl
      style={{
        display: 'grid',
        gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
        gap: dense ? 'var(--space-1) var(--space-3)' : 'var(--space-15) var(--space-3)',
        margin: 0,
        ...style,
      }}
    >
      {items.map((item) => (
        <div key={item.label} style={{ display: 'grid', gap: 2, minWidth: 0 }}>
          <dt
            style={{
              font: 'var(--type-table-head)',
              color: 'var(--text-secondary)',
              fontWeight: 'var(--weight-medium)',
            }}
          >
            {item.label}
          </dt>
          <dd
            style={{
              margin: 0,
              font: item.mono || mono ? 'var(--type-mono)' : 'var(--type-body)',
              fontSize: 'var(--text-sm)',
              color: 'var(--text-body)',
              overflowWrap: 'anywhere',
            }}
          >
            {item.value}
          </dd>
        </div>
      ))}
    </dl>
  );
}
