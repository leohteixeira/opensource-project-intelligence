import type { CSSProperties } from 'react';

import { GLYPHS, type IconName } from './icons';

export interface IconProps {
  readonly name: IconName;
  readonly size?: number;
  readonly strokeWidth?: number;
  /** Giving a label turns a decorative glyph into an image. A status glyph never travels alone. */
  readonly label?: string;
  readonly color?: string;
  readonly style?: CSSProperties;
  readonly className?: string;
}

export function Icon({
  name,
  size = 16,
  strokeWidth = 1.75,
  label,
  color,
  style,
  className,
}: IconProps) {
  const Glyph = GLYPHS[name];

  return (
    <Glyph
      size={size}
      strokeWidth={strokeWidth}
      className={className}
      focusable="false"
      style={{ color, flex: 'none', ...style }}
      role={label ? 'img' : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
    />
  );
}
