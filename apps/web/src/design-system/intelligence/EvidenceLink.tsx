import type { CSSProperties } from 'react';

import { Icon } from '../core/Icon';
import type { IconName } from '../core/icons';

export type EvidenceKind =
  | 'issue'
  | 'pull_request'
  | 'release'
  | 'commit'
  | 'changelog'
  | 'advisory'
  | 'discussion'
  | 'package'
  | 'document'
  | 'snapshot'
  | 'metric'
  | 'run'
  | 'website';

const KIND: Record<EvidenceKind, IconName> = {
  issue: 'circle-dot',
  pull_request: 'git-pull-request',
  release: 'tag',
  commit: 'git-commit-horizontal',
  changelog: 'file-text',
  advisory: 'shield-alert',
  discussion: 'message-square',
  package: 'package',
  document: 'book-open',
  snapshot: 'camera',
  metric: 'chart-line',
  run: 'sparkles',
  website: 'globe',
};

/**
 * A citation. Every claim in this product points at collected evidence, and the citation shows
 * what kind of source it is, when it was collected, and whether it left the workspace.
 */
export interface EvidenceLinkProps {
  readonly kind?: EvidenceKind;
  readonly title: string;
  readonly href: string;
  readonly source?: string;
  readonly collectedAt?: string;
  readonly external?: boolean;
  readonly count?: number;
  readonly style?: CSSProperties;
}

export function EvidenceLink({
  kind = 'issue',
  title,
  href,
  source,
  collectedAt,
  external = true,
  count,
  style,
}: EvidenceLinkProps) {
  return (
    <a
      href={href}
      target={external ? '_blank' : undefined}
      rel={external ? 'noreferrer noopener' : undefined}
      style={{
        display: 'flex',
        gap: 'var(--space-1)',
        alignItems: 'flex-start',
        padding: 'var(--space-075) var(--space-1)',
        border: 'var(--border-hairline)',
        borderRadius: 'var(--radius-xs)',
        background: 'var(--surface-card)',
        textDecoration: 'none',
        color: 'var(--text-body)',
        minWidth: 0,
        ...style,
      }}
    >
      <Icon name={KIND[kind]} size={14} style={{ color: 'var(--evidence-fg)', marginTop: 2 }} />
      <span style={{ display: 'grid', gap: 1, flex: 1, minWidth: 0 }}>
        <span
          style={{
            font: 'var(--type-ui)',
            color: 'var(--evidence-fg)',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {title}
        </span>
        <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-secondary)' }}>
          {source}
          {collectedAt ? ` · collected ${collectedAt}` : ''}
          {count ? ` · ${count} items` : ''}
        </span>
      </span>
      {external ? (
        <Icon
          name="external-link"
          size={12}
          style={{ color: 'var(--text-tertiary)', marginTop: 3 }}
        />
      ) : null}
    </a>
  );
}
