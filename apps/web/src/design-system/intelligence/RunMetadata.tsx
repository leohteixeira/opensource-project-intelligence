import type { CSSProperties, ReactNode } from 'react';

import { StatusBadge } from '../core/StatusBadge';
import type { StatusKey } from '../core/status';
import { DefinitionList, type DefinitionItem } from '../data/DefinitionList';

export interface RunMetadataLabels {
  readonly ai?: string;
  readonly state?: string;
  readonly stale?: string;
  readonly selected?: string;
  readonly providerModel?: string;
  readonly promptVersion?: string;
  readonly language?: string;
  readonly executed?: string;
  readonly usage?: string;
}

const DEFAULT_LABELS: RunMetadataLabels = {
  selected: 'Presented version',
  providerModel: 'Provider / model',
  promptVersion: 'Prompt version',
  language: 'Language',
  executed: 'Executed',
  usage: 'Usage',
};

/**
 * Immutable AI run header. A run is never edited in place: a rerun creates a new version, and a
 * stale run stays visible as stale rather than passing for current. `labels` overrides the fixed
 * copy for a localized surface; the run's own values are never translated.
 */
export interface RunMetadataProps {
  readonly runId: string;
  readonly state?: StatusKey;
  readonly provider: string;
  readonly model: string;
  readonly promptVersion: string;
  readonly language: string;
  readonly executedAt: string;
  readonly versionLabel?: string;
  readonly selected?: boolean;
  readonly stale?: string;
  readonly usage?: string;
  readonly labels?: RunMetadataLabels;
  readonly actions?: ReactNode;
  readonly style?: CSSProperties;
}

export function RunMetadata({
  runId,
  state = 'succeeded',
  provider,
  model,
  promptVersion,
  language,
  executedAt,
  versionLabel,
  selected,
  stale,
  usage,
  labels,
  actions,
  style,
}: RunMetadataProps) {
  const text = { ...DEFAULT_LABELS, ...labels };
  const items: DefinitionItem[] = [
    {
      label: text.providerModel ?? 'Provider / model',
      value: `${provider} · ${model}`,
      mono: true,
    },
    { label: text.promptVersion ?? 'Prompt version', value: promptVersion, mono: true },
    { label: text.language ?? 'Language', value: language, mono: true },
    { label: text.executed ?? 'Executed', value: executedAt, mono: true },
  ];

  if (usage) items.push({ label: text.usage ?? 'Usage', value: usage, mono: true });

  return (
    <div style={{ display: 'grid', gap: 'var(--space-1)', minWidth: 0, ...style }}>
      <div
        style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-1)', flexWrap: 'wrap' }}
      >
        <StatusBadge status="ai" size="sm" label={text.ai} />
        <StatusBadge status={state} size="sm" label={text.state} />
        {stale ? <StatusBadge status="stale" detail={stale} size="sm" label={text.stale} /> : null}
        {selected ? (
          <span
            style={{
              font: 'var(--type-caption)',
              color: 'var(--positive-fg)',
              fontWeight: 'var(--weight-medium)',
            }}
          >
            {text.selected}
          </span>
        ) : null}
        <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>
          {versionLabel} · {runId}
        </span>
        <span style={{ marginLeft: 'auto', display: 'flex', gap: 'var(--space-05)' }}>
          {actions}
        </span>
      </div>
      <DefinitionList dense columns={2} items={items} />
    </div>
  );
}
