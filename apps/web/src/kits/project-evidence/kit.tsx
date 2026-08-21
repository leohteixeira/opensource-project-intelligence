import type { CSSProperties, ReactNode } from 'react';

import {
  EvidenceLink,
  Icon,
  Tabs,
  type EvidenceLinkProps,
  type TabItem,
} from '../../design-system';

/**
 * Every screen in this shell exposes the states the UI/UX contract specifies for its surface
 * through one switcher, so the whole state list is reachable without one file per state. The
 * switcher is a kit affordance, not product UI.
 */
export function StateBar({
  items,
  value,
  onChange,
  route,
  note,
}: {
  readonly items: readonly TabItem[];
  readonly value: string;
  readonly onChange: (value: string) => void;
  readonly route?: string;
  readonly note?: string;
}) {
  return (
    <div style={{ display: 'grid', gap: 'var(--space-1)', paddingBottom: 'var(--space-05)' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-15)', minWidth: 0 }}>
        <span
          style={{
            font: 'var(--type-eyebrow)',
            letterSpacing: 'var(--tracking-eyebrow)',
            textTransform: 'uppercase',
            color: 'var(--text-tertiary)',
            whiteSpace: 'nowrap',
          }}
        >
          Specified state
        </span>
        <div style={{ minWidth: 0, flex: '1 1 auto', display: 'flex' }}>
          <Tabs
            variant="pill"
            size="sm"
            value={value}
            onChange={onChange}
            items={items}
            style={{ maxWidth: '100%' }}
          />
        </div>
      </div>
      {route ? (
        <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>{route}</span>
      ) : null}
      {note ? (
        <span style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>{note}</span>
      ) : null}
    </div>
  );
}

export function Stack({
  gap,
  children,
  style,
}: {
  readonly gap?: string;
  readonly children: ReactNode;
  readonly style?: CSSProperties;
}) {
  return <div style={{ display: 'grid', gap: gap ?? 'var(--space-2)', ...style }}>{children}</div>;
}

export function Cols({
  template,
  gap,
  children,
  style,
}: {
  readonly template: string;
  readonly gap?: string;
  readonly children: ReactNode;
  readonly style?: CSSProperties;
}) {
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: template,
        gap: gap ?? 'var(--space-2)',
        alignItems: 'start',
        ...style,
      }}
    >
      {children}
    </div>
  );
}

export function Mono({
  children,
  tone,
}: {
  readonly children: ReactNode;
  readonly tone?: 'quiet';
}) {
  return (
    <span
      style={{
        font: 'var(--type-mono-xs)',
        color: tone === 'quiet' ? 'var(--text-tertiary)' : 'var(--text-secondary)',
      }}
    >
      {children}
    </span>
  );
}

/** The provenance line every panel footer carries: window · cutoff · definition version. */
export function Provenance({
  window: observationWindow,
  cutoff,
  version,
  extra,
}: {
  readonly window?: string;
  readonly cutoff?: string;
  readonly version?: string;
  readonly extra?: string;
}) {
  const parts = [observationWindow, cutoff ? `cutoff ${cutoff}` : undefined, version, extra].filter(
    (part): part is string => Boolean(part),
  );

  return <Mono tone="quiet">{parts.join(' · ')}</Mono>;
}

/** A claim, metric or answer that came out of a model — labelled at the point of reading. */
export function AiLabel({ children }: { readonly children: ReactNode }) {
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 'var(--space-05)',
        padding: '2px 8px',
        borderRadius: 'var(--radius-pill)',
        background: 'var(--ai-bg)',
        border: '1px solid var(--ai-border)',
        color: 'var(--ai-fg)',
        font: 'var(--type-caption)',
        fontWeight: 'var(--weight-medium)',
      }}
    >
      <Icon name="sparkles" size={12} />
      {children}
    </span>
  );
}

export function EvidenceRow({ cites }: { readonly cites: readonly EvidenceLinkProps[] }) {
  return (
    <div style={{ display: 'grid', gap: 'var(--space-075)' }}>
      {cites.map((cite) => (
        <EvidenceLink key={`${cite.kind}-${cite.title}`} {...cite} />
      ))}
    </div>
  );
}
