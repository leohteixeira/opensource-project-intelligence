import type { CSSProperties } from 'react';

/**
 * No logo or brand mark was supplied with the source specifications, so the product signs itself
 * in type: a blue "OPI" tile beside the product name. Four forms — "inline" for the top bar,
 * "stack" for narrow headers, "line" for public surfaces, "mark" for the tile alone. Do not
 * replace this with a drawn mark.
 */
export interface WordmarkProps {
  readonly variant?: 'inline' | 'stack' | 'line' | 'mark';
  readonly size?: 'sm' | 'md' | 'lg';
  readonly tone?: 'brand' | 'inverse';
  readonly style?: CSSProperties;
}

export function Wordmark({
  variant = 'inline',
  size = 'md',
  tone = 'brand',
  style,
}: WordmarkProps) {
  const scale = size === 'lg' ? 1.35 : size === 'sm' ? 0.82 : 1;
  const inverse = tone === 'inverse';
  const fg = inverse ? 'var(--text-inverse)' : 'var(--text-primary)';
  const accent = inverse ? 'var(--white)' : 'var(--text-brand)';
  const tileStyle: CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    width: 28 * scale,
    height: 28 * scale,
    background: inverse ? 'var(--white)' : 'var(--surface-brand)',
    color: inverse ? 'var(--blue-600)' : 'var(--white)',
    borderRadius: 'var(--radius-sm)',
    font: 'var(--type-eyebrow)',
    fontSize: 11 * scale,
    letterSpacing: '.02em',
  };

  if (variant === 'mark') {
    return (
      <span aria-label="Open Source Project Intelligence" style={{ ...tileStyle, ...style }}>
        OPI
      </span>
    );
  }

  const tile = (
    <span aria-hidden="true" style={tileStyle}>
      OPI
    </span>
  );

  if (variant === 'line') {
    return (
      <span
        style={{ display: 'inline-flex', alignItems: 'center', gap: 'var(--space-1)', ...style }}
      >
        {tile}
        <span style={{ display: 'inline-flex', alignItems: 'baseline', gap: 'var(--space-075)' }}>
          <span
            style={{
              font: 'var(--type-section)',
              fontSize: 17 * scale,
              letterSpacing: 'var(--tracking-display)',
              color: fg,
            }}
          >
            Open Source
          </span>
          <span
            style={{
              font: 'var(--type-section)',
              fontSize: 17 * scale,
              letterSpacing: 'var(--tracking-display)',
              color: accent,
            }}
          >
            Project Intelligence
          </span>
        </span>
      </span>
    );
  }

  if (variant === 'stack') {
    return (
      <span
        style={{ display: 'inline-flex', alignItems: 'center', gap: 'var(--space-1)', ...style }}
      >
        {tile}
        <span style={{ display: 'grid' }}>
          <span
            style={{
              font: 'var(--type-eyebrow)',
              fontSize: 11 * scale,
              letterSpacing: 'var(--tracking-eyebrow)',
              textTransform: 'uppercase',
              color: 'var(--text-secondary)',
            }}
          >
            Open Source
          </span>
          <span
            style={{
              font: 'var(--type-subsection)',
              fontSize: 14 * scale,
              letterSpacing: 'var(--tracking-tight)',
              color: fg,
            }}
          >
            Project Intelligence
          </span>
        </span>
      </span>
    );
  }

  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 'var(--space-1)',
        flex: 'none',
        ...style,
      }}
    >
      {tile}
      <span
        className="opi-wordmark-label"
        style={{
          font: 'var(--type-subsection)',
          fontSize: 15 * scale,
          letterSpacing: 'var(--tracking-tight)',
          color: fg,
          whiteSpace: 'nowrap',
        }}
      >
        Project Intelligence
      </span>
    </span>
  );
}
