import type { CSSProperties } from 'react';

import { Icon } from '../core/Icon';
import { StatusBadge } from '../core/StatusBadge';

export type RecommendationResult =
  | 'recommended'
  | 'conditional'
  | 'not_recommended'
  | 'insufficient_data';

export interface DecisiveFactor {
  readonly metric: string;
  readonly rule: string;
  readonly value: string;
  readonly pass: boolean;
}

const RESULTS: Record<RecommendationResult, string> = {
  recommended: 'Recommended',
  conditional: 'Conditional',
  not_recommended: 'Not recommended',
  insufficient_data: 'Insufficient data',
};

/**
 * One of exactly four deterministic outcomes, with the policy version, the decisive factors and
 * the missing inputs all visible. The result is never softened and never inferred from a colour.
 */
export interface RecommendationProps {
  readonly result?: RecommendationResult;
  readonly policy?: string;
  readonly version?: string;
  readonly window?: string;
  readonly cutoff?: string;
  readonly conditions?: readonly string[];
  readonly decisive?: readonly DecisiveFactor[];
  readonly missing?: readonly string[];
  readonly stale?: string;
  readonly onEvidence?: () => void;
  readonly style?: CSSProperties;
}

export function Recommendation({
  result = 'insufficient_data',
  policy,
  version,
  window: observationWindow,
  cutoff,
  conditions = [],
  decisive = [],
  missing = [],
  stale,
  onEvidence,
  style,
}: RecommendationProps) {
  return (
    <div style={{ display: 'grid', gap: 'var(--space-15)', minWidth: 0, ...style }}>
      <div
        style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-1)', flexWrap: 'wrap' }}
      >
        <StatusBadge status={result} label={RESULTS[result]} />
        {stale ? <StatusBadge status="stale" detail={stale} size="sm" /> : null}
        <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-secondary)' }}>
          {policy} {version} · {observationWindow} · cutoff {cutoff}
        </span>
      </div>
      {conditions.length ? (
        <ul style={{ display: 'grid', gap: 'var(--space-05)' }}>
          {conditions.map((condition) => (
            <li
              key={condition}
              style={{
                display: 'flex',
                gap: 'var(--space-075)',
                font: 'var(--type-body)',
                fontSize: 'var(--text-sm)',
                color: 'var(--text-body)',
              }}
            >
              <Icon
                name="triangle-alert"
                size={14}
                style={{ color: 'var(--attention-fg)', marginTop: 3 }}
              />
              {condition}
            </li>
          ))}
        </ul>
      ) : null}
      {decisive.length ? (
        <div style={{ display: 'grid', gap: 'var(--space-05)' }}>
          <p style={{ font: 'var(--type-table-head)', color: 'var(--text-secondary)' }}>
            Decisive factors
          </p>
          <ul style={{ display: 'grid', gap: 2 }}>
            {decisive.map((factor) => (
              <li
                key={factor.metric}
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  gap: 'var(--space-15)',
                  font: 'var(--type-mono-xs)',
                  color: 'var(--text-body)',
                  padding: '3px 0',
                  borderBottom: '1px solid var(--border-table)',
                }}
              >
                <span>{factor.metric}</span>
                <span style={{ display: 'flex', gap: 'var(--space-1)', alignItems: 'center' }}>
                  <span style={{ color: 'var(--text-secondary)' }}>{factor.rule}</span>
                  <span style={{ fontWeight: 'var(--weight-semibold)' }}>{factor.value}</span>
                  <Icon
                    name={factor.pass ? 'check' : 'x'}
                    size={12}
                    style={{ color: factor.pass ? 'var(--positive-fg)' : 'var(--critical-fg)' }}
                  />
                </span>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      {missing.length ? (
        <p
          style={{
            display: 'flex',
            gap: 'var(--space-075)',
            alignItems: 'flex-start',
            font: 'var(--type-caption)',
            color: 'var(--text-secondary)',
          }}
        >
          <Icon name="circle-dashed" size={13} style={{ marginTop: 1 }} />
          <span>Missing inputs: {missing.join(', ')}</span>
        </p>
      ) : null}
      {onEvidence ? (
        <button
          type="button"
          onClick={onEvidence}
          style={{
            justifySelf: 'start',
            border: 0,
            background: 'transparent',
            padding: 0,
            font: 'var(--type-ui)',
            color: 'var(--evidence-fg)',
            textDecoration: 'underline',
            cursor: 'pointer',
          }}
        >
          Open policy evaluation evidence
        </button>
      ) : null}
    </div>
  );
}
