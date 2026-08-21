import type { IconName } from './icons';

export type StatusKey =
  | 'ready'
  | 'available'
  | 'succeeded'
  | 'recommended'
  | 'conditional'
  | 'not_recommended'
  | 'insufficient_data'
  | 'unknown'
  | 'not_applicable'
  | 'queued'
  | 'running'
  | 'stale'
  | 'failed'
  | 'cancelled'
  | 'paused'
  | 'archived'
  | 'active'
  | 'forecast'
  | 'observed'
  | 'ai'
  | 'translated';

type Tone =
  | 'positive'
  | 'attention'
  | 'critical'
  | 'info'
  | 'pending'
  | 'unknown'
  | 'forecast'
  | 'neutral';

export interface StatusSpec {
  readonly tone: Tone;
  readonly glyph: IconName;
  readonly label: string;
  readonly spin?: boolean;
  readonly outline?: boolean;
}

/**
 * The frozen signal vocabulary from the UI/UX contract, exported as data so a consumer cannot
 * quietly invent a fifth recommendation state. A status is never colour alone: every badge renders
 * a glyph, a colour and a word. `stale` is the one outline treatment, so a stale value reads as
 * "was valid" rather than "is amber".
 */
export const STATUS: Record<StatusKey, StatusSpec> = {
  ready: { tone: 'positive', glyph: 'check', label: 'Ready' },
  available: { tone: 'neutral', glyph: 'check', label: 'Available' },
  succeeded: { tone: 'positive', glyph: 'check', label: 'Succeeded' },
  recommended: { tone: 'positive', glyph: 'circle-check', label: 'Recommended' },
  conditional: { tone: 'attention', glyph: 'triangle-alert', label: 'Conditional' },
  not_recommended: { tone: 'critical', glyph: 'octagon-x', label: 'Not recommended' },
  insufficient_data: { tone: 'unknown', glyph: 'circle-dashed', label: 'Insufficient data' },
  unknown: { tone: 'unknown', glyph: 'circle-help', label: 'Unknown' },
  not_applicable: { tone: 'unknown', glyph: 'minus', label: 'Not applicable' },
  queued: { tone: 'pending', glyph: 'clock', label: 'Queued' },
  running: { tone: 'info', glyph: 'loader-circle', label: 'Running', spin: true },
  stale: { tone: 'attention', glyph: 'history', label: 'Stale', outline: true },
  failed: { tone: 'critical', glyph: 'circle-alert', label: 'Failed' },
  cancelled: { tone: 'unknown', glyph: 'circle-slash', label: 'Cancelled' },
  paused: { tone: 'neutral', glyph: 'pause', label: 'Paused' },
  archived: { tone: 'neutral', glyph: 'archive', label: 'Archived' },
  active: { tone: 'positive', glyph: 'circle-dot', label: 'Active' },
  forecast: { tone: 'forecast', glyph: 'trending-up', label: 'Forecast' },
  observed: { tone: 'info', glyph: 'chart-line', label: 'Observed' },
  ai: { tone: 'forecast', glyph: 'sparkles', label: 'AI-generated' },
  translated: { tone: 'unknown', glyph: 'languages', label: 'Generated translation' },
};

export const TONES: Record<Tone, readonly [string, string, string]> = {
  positive: ['var(--positive-fg)', 'var(--positive-bg)', 'var(--positive-border)'],
  attention: ['var(--attention-fg)', 'var(--attention-bg)', 'var(--attention-border)'],
  critical: ['var(--critical-fg)', 'var(--critical-bg)', 'var(--critical-border)'],
  info: ['var(--info-fg)', 'var(--info-bg)', 'var(--info-border)'],
  pending: ['var(--pending-fg)', 'var(--pending-bg)', 'var(--pending-border)'],
  unknown: ['var(--unknown-fg)', 'var(--unknown-bg)', 'var(--unknown-border)'],
  forecast: ['var(--forecast-fg)', 'var(--forecast-bg)', 'var(--forecast-border)'],
  neutral: ['var(--gray-700)', 'var(--gray-50)', 'var(--gray-200)'],
};
