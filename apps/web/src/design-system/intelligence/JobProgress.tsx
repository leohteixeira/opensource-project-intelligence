import type { CSSProperties, ReactNode } from 'react';

import { Icon } from '../core/Icon';
import { Progress } from '../core/Progress';
import { StatusBadge } from '../core/StatusBadge';
import type { StatusKey } from '../core/status';

/**
 * A durable job. Everything here is factual: the checkpoint it will resume from, the number of
 * coalesced duplicate requests, the retry-after, and the reason it failed.
 */
export interface JobProgressProps {
  readonly id: string;
  readonly kind: string;
  readonly state?: StatusKey;
  readonly completed?: number;
  readonly total?: number;
  readonly unit?: string;
  readonly checkpoint?: string;
  readonly startedAt?: string;
  readonly updatedAt?: string;
  readonly coalesced?: number;
  readonly retryAfter?: string;
  readonly failure?: string;
  readonly transport?: 'stream' | 'polling';
  readonly actions?: ReactNode;
  readonly style?: CSSProperties;
}

function Fact({ term, value }: { readonly term: string; readonly value: ReactNode }) {
  return (
    <span>
      <dt style={{ display: 'inline', color: 'var(--text-tertiary)' }}>{term} </dt>
      <dd style={{ display: 'inline', margin: 0 }}>{value}</dd>
    </span>
  );
}

export function JobProgress({
  id,
  kind,
  state = 'queued',
  completed,
  total,
  unit = 'sources',
  checkpoint,
  startedAt,
  updatedAt,
  coalesced,
  retryAfter,
  failure,
  transport,
  actions,
  style,
}: JobProgressProps) {
  const active = state === 'running' || state === 'queued';

  return (
    <div style={{ display: 'grid', gap: 'var(--space-1)', minWidth: 0, ...style }}>
      <div
        style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-1)', flexWrap: 'wrap' }}
      >
        <StatusBadge status={state} size="sm" />
        <span style={{ font: 'var(--type-ui)', color: 'var(--text-primary)' }}>{kind}</span>
        <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>{id}</span>
        {transport ? (
          <span
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 3,
              font: 'var(--type-mono-xs)',
              color: transport === 'polling' ? 'var(--attention-fg)' : 'var(--text-secondary)',
            }}
          >
            <Icon name={transport === 'polling' ? 'refresh-cw' : 'radio'} size={11} />
            {transport === 'polling' ? 'polling fallback' : 'event stream'}
          </span>
        ) : null}
        <span style={{ marginLeft: 'auto', display: 'flex', gap: 'var(--space-05)' }}>
          {actions}
        </span>
      </div>
      {active ? (
        <Progress
          label={state === 'queued' ? 'Queued' : 'Progress'}
          completed={completed}
          total={total}
          unit={unit}
          indeterminate={!total}
          size="sm"
        />
      ) : null}
      <dl
        style={{
          display: 'flex',
          gap: 'var(--space-2)',
          flexWrap: 'wrap',
          margin: 0,
          font: 'var(--type-mono-xs)',
          color: 'var(--text-secondary)',
        }}
      >
        {startedAt ? <Fact term="started" value={startedAt} /> : null}
        {updatedAt ? <Fact term="updated" value={updatedAt} /> : null}
        {checkpoint ? <Fact term="checkpoint" value={checkpoint} /> : null}
        {typeof coalesced === 'number' && coalesced > 0 ? (
          <Fact term="coalesced" value={`${coalesced} requests`} />
        ) : null}
        {retryAfter ? <Fact term="retry-after" value={retryAfter} /> : null}
      </dl>
      {failure ? (
        <p
          style={{
            display: 'flex',
            gap: 'var(--space-075)',
            alignItems: 'flex-start',
            font: 'var(--type-caption)',
            color: 'var(--critical-fg)',
          }}
        >
          <Icon name="circle-alert" size={13} style={{ marginTop: 1 }} />
          <span>{failure}</span>
        </p>
      ) : null}
    </div>
  );
}
