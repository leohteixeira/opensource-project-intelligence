import type { CSSProperties } from 'react';

import { Icon } from '../core/Icon';
import { StatusBadge } from '../core/StatusBadge';

export type RadarRing = 'adopt' | 'trial' | 'assess' | 'hold' | 'unplaced';

export interface RadarOverride {
  readonly owner: string;
  readonly expired?: boolean;
}

export interface RadarEntry {
  readonly project: string;
  readonly policyRing?: RadarRing;
  readonly effectiveRing?: RadarRing;
  readonly override?: RadarOverride;
  readonly reviewDue?: string;
  readonly reviewOverdue?: boolean;
}

/**
 * The radar as a list, which is its primary representation — the spatial plot is optional and
 * supplementary. Policy suggestion, effective placement and an attributed human override stay
 * three separate facts.
 */
const RINGS: readonly { key: RadarRing; label: string; color: string }[] = [
  { key: 'adopt', label: 'Adopt', color: 'var(--ring-adopt)' },
  { key: 'trial', label: 'Trial', color: 'var(--ring-trial)' },
  { key: 'assess', label: 'Assess', color: 'var(--ring-assess)' },
  { key: 'hold', label: 'Hold', color: 'var(--ring-hold)' },
  { key: 'unplaced', label: 'Unplaced', color: 'var(--ring-unplaced)' },
];

export interface RadarListProps {
  readonly entries: readonly RadarEntry[];
  readonly onSelect?: (entry: RadarEntry) => void;
  readonly style?: CSSProperties;
}

export function RadarList({ entries, onSelect, style }: RadarListProps) {
  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)', ...style }}>
      {RINGS.map((ring) => {
        const items = entries.filter((entry) => (entry.effectiveRing ?? 'unplaced') === ring.key);

        if (!items.length) return null;

        return (
          <section key={ring.key} style={{ display: 'grid', gap: 'var(--space-075)' }}>
            <h3
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 'var(--space-075)',
                font: 'var(--type-eyebrow)',
                letterSpacing: 'var(--tracking-eyebrow)',
                textTransform: 'uppercase',
                color: 'var(--text-secondary)',
              }}
            >
              <span
                aria-hidden="true"
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: 'var(--radius-pill)',
                  background: ring.color,
                }}
              />
              {ring.label}
              <span style={{ color: 'var(--text-tertiary)', fontVariantNumeric: 'tabular-nums' }}>
                {items.length}
              </span>
            </h3>
            <ul style={{ display: 'grid', gap: 'var(--space-05)' }}>
              {items.map((entry) => (
                <li key={entry.project}>
                  <button
                    type="button"
                    onClick={() => onSelect?.(entry)}
                    className="opi-item"
                    style={{
                      width: '100%',
                      textAlign: 'left',
                      display: 'flex',
                      alignItems: 'center',
                      gap: 'var(--space-15)',
                      minHeight: 44,
                      padding: 'var(--space-075) var(--space-1)',
                      border: 'var(--border-hairline)',
                      borderLeft: `3px solid ${ring.color}`,
                      borderRadius: 'var(--radius-xs)',
                      background: 'var(--surface-card)',
                      cursor: 'pointer',
                    }}
                  >
                    <span style={{ flex: 1, minWidth: 0, display: 'grid', gap: 1 }}>
                      <span
                        style={{
                          font: 'var(--type-body-strong)',
                          fontSize: 'var(--text-sm)',
                          color: 'var(--text-primary)',
                        }}
                      >
                        {entry.project}
                      </span>
                      <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-secondary)' }}>
                        policy suggests {entry.policyRing ?? 'unplaced'}
                        {entry.override ? ` · overridden by ${entry.override.owner}` : ''}
                      </span>
                    </span>
                    {entry.override ? (
                      <StatusBadge
                        status={entry.override.expired ? 'stale' : 'available'}
                        label={entry.override.expired ? 'Override expired' : 'Override'}
                        size="sm"
                      />
                    ) : null}
                    {entry.reviewDue ? (
                      <span
                        style={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: 3,
                          font: 'var(--type-mono-xs)',
                          color: entry.reviewOverdue
                            ? 'var(--critical-fg)'
                            : 'var(--text-secondary)',
                        }}
                      >
                        <Icon name="calendar-clock" size={11} />
                        {entry.reviewDue}
                      </span>
                    ) : null}
                    <Icon
                      name="chevron-right"
                      size={15}
                      style={{ color: 'var(--text-tertiary)' }}
                    />
                  </button>
                </li>
              ))}
            </ul>
          </section>
        );
      })}
    </div>
  );
}
