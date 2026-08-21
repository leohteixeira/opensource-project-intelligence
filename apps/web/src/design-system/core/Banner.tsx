import type { CSSProperties, ReactNode } from 'react';

import { Icon } from './Icon';
import type { IconName } from './icons';
import { IconButton } from './IconButton';

export type BannerTone = 'info' | 'positive' | 'attention' | 'critical' | 'neutral';

interface ToneSpec {
  readonly fg: string;
  readonly bg: string;
  readonly border: string;
  readonly glyph: IconName;
  readonly role: 'status' | 'alert';
}

const TONES: Record<BannerTone, ToneSpec> = {
  info: {
    fg: 'var(--info-fg)',
    bg: 'var(--info-bg)',
    border: 'var(--info-border)',
    glyph: 'info',
    role: 'status',
  },
  positive: {
    fg: 'var(--positive-fg)',
    bg: 'var(--positive-bg)',
    border: 'var(--positive-border)',
    glyph: 'circle-check',
    role: 'status',
  },
  attention: {
    fg: 'var(--attention-fg)',
    bg: 'var(--attention-bg)',
    border: 'var(--attention-border)',
    glyph: 'triangle-alert',
    role: 'status',
  },
  critical: {
    fg: 'var(--critical-fg)',
    bg: 'var(--critical-bg)',
    border: 'var(--critical-border)',
    glyph: 'circle-alert',
    role: 'alert',
  },
  neutral: {
    fg: 'var(--gray-700)',
    bg: 'var(--gray-50)',
    border: 'var(--gray-200)',
    glyph: 'info',
    role: 'status',
  },
};

/**
 * In-page message. Banners carry outcomes that must survive a reload, which is why this product
 * has no toast primitive: a toast can be the only record of an outcome, and that is forbidden.
 */
export interface BannerProps {
  readonly tone?: BannerTone;
  readonly title?: ReactNode;
  readonly children?: ReactNode;
  readonly actions?: ReactNode;
  readonly onDismiss?: () => void;
  readonly glyph?: IconName;
  readonly style?: CSSProperties;
}

export function Banner({
  tone = 'info',
  title,
  children,
  actions,
  onDismiss,
  glyph,
  style,
}: BannerProps) {
  const spec = TONES[tone];

  return (
    <div
      role={spec.role}
      style={{
        display: 'flex',
        gap: 'var(--space-15)',
        alignItems: 'flex-start',
        padding: 'var(--space-15)',
        background: spec.bg,
        border: `1px solid ${spec.border}`,
        borderRadius: 'var(--radius-sm)',
        ...style,
      }}
    >
      <Icon name={glyph ?? spec.glyph} size={17} style={{ color: spec.fg, marginTop: 1 }} />
      <div style={{ flex: 1, minWidth: 0, display: 'grid', gap: 'var(--space-05)' }}>
        {title ? <p style={{ font: 'var(--type-body-strong)', color: spec.fg }}>{title}</p> : null}
        {children ? (
          <div
            style={{
              font: 'var(--type-body)',
              fontSize: 'var(--text-sm)',
              color: 'var(--text-body)',
            }}
          >
            {children}
          </div>
        ) : null}
        {actions ? (
          <div
            style={{
              display: 'flex',
              flexWrap: 'wrap',
              gap: 'var(--space-1)',
              marginTop: 'var(--space-05)',
            }}
          >
            {actions}
          </div>
        ) : null}
      </div>
      {onDismiss ? <IconButton icon="x" label="Dismiss" size="sm" onClick={onDismiss} /> : null}
    </div>
  );
}
