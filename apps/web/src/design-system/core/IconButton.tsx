import type { CSSProperties, MouseEventHandler } from 'react';

import { Icon } from './Icon';
import type { IconName } from './icons';

export type IconButtonVariant = 'ghost' | 'secondary' | 'outline';
export type IconButtonSize = 'lg' | 'md' | 'sm';

/**
 * Icon-only control. The visual box may be 28 or 36px, but the hit area is always at least the
 * 44px WCAG 2.2 target, so the outline offset grows as the box shrinks.
 */
const BOX: Record<IconButtonSize, number> = { lg: 44, md: 36, sm: 32 };

export interface IconButtonProps {
  readonly icon: IconName;
  /** Never optional: an icon-only control is unreadable without its accessible name. */
  readonly label: string;
  readonly variant?: IconButtonVariant;
  readonly size?: IconButtonSize;
  readonly shape?: 'square' | 'circle';
  readonly disabled?: boolean;
  readonly selected?: boolean;
  readonly onClick?: MouseEventHandler<HTMLButtonElement>;
  readonly style?: CSSProperties;
  readonly 'aria-expanded'?: boolean;
  readonly 'aria-haspopup'?: 'menu';
}

export function IconButton({
  icon,
  label,
  variant = 'ghost',
  size = 'md',
  shape = 'square',
  disabled,
  selected,
  onClick,
  style,
  ...aria
}: IconButtonProps) {
  const box = BOX[size];
  const modifier = variant === 'outline' ? 'secondary opi-icon-btn--outline' : variant;

  return (
    <button
      type="button"
      className={`opi-btn opi-btn--${modifier}`}
      onClick={disabled ? undefined : onClick}
      disabled={disabled}
      aria-label={label}
      title={label}
      aria-pressed={selected === undefined ? undefined : selected}
      style={{
        width: box,
        height: box,
        padding: 0,
        borderRadius: shape === 'circle' ? 'var(--radius-pill)' : 'var(--radius-sm)',
        background: selected ? 'var(--surface-sunken)' : undefined,
        outlineOffset: Math.max(0, (44 - box) / 2),
        ...style,
      }}
      {...aria}
    >
      <Icon name={icon} size={size === 'sm' ? 14 : 17} />
    </button>
  );
}
