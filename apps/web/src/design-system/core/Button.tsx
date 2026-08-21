import type { CSSProperties, MouseEventHandler, ReactNode } from 'react';

import { Icon } from './Icon';
import type { IconName } from './icons';

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger';
export type ButtonSize = 'lg' | 'md' | 'sm';

const SIZES: Record<ButtonSize, CSSProperties> = {
  lg: { height: 'var(--control-h-lg)', padding: '0 var(--space-2)', fontSize: 'var(--text-base)' },
  md: { height: 'var(--control-h-md)', padding: '0 var(--space-15)', fontSize: 'var(--text-sm)' },
  sm: { height: 'var(--control-h-sm)', padding: '0 var(--space-1)', fontSize: 'var(--text-xs)' },
};

export interface ButtonProps {
  readonly children: ReactNode;
  readonly variant?: ButtonVariant;
  readonly size?: ButtonSize;
  readonly iconStart?: IconName;
  readonly iconEnd?: IconName;
  readonly disabled?: boolean;
  /** A pending action keeps its label, swaps its glyph for a spinner and refuses further clicks. */
  readonly pending?: boolean;
  readonly fullWidth?: boolean;
  readonly type?: 'button' | 'submit' | 'reset';
  readonly href?: string;
  readonly onClick?: MouseEventHandler<HTMLElement>;
  readonly ariaLabel?: string;
  readonly style?: CSSProperties;
}

export function Button({
  children,
  variant = 'secondary',
  size = 'md',
  iconStart,
  iconEnd,
  disabled,
  pending,
  fullWidth,
  type = 'button',
  href,
  onClick,
  ariaLabel,
  style,
}: ButtonProps) {
  const geometry = SIZES[size];
  const glyph = pending ? 'loader-circle' : iconStart;
  const inert = disabled === true || pending === true;
  const shared = {
    className: `opi-btn opi-btn--${variant}`,
    onClick: inert ? undefined : onClick,
    'aria-busy': pending ? true : undefined,
    'aria-label': ariaLabel,
    style: {
      ...geometry,
      width: fullWidth ? '100%' : undefined,
      minHeight: size === 'lg' ? 'var(--control-touch)' : undefined,
      ...style,
    },
  } as const;
  const content = (
    <>
      {glyph ? (
        <Icon
          name={glyph}
          size={size === 'sm' ? 13 : 15}
          className={pending ? 'opi-spin' : undefined}
        />
      ) : null}
      <span>{children}</span>
      {iconEnd && !pending ? <Icon name={iconEnd} size={size === 'sm' ? 13 : 15} /> : null}
    </>
  );

  if (href) {
    return (
      <a {...shared} href={href} aria-disabled={inert ? true : undefined}>
        {content}
      </a>
    );
  }

  return (
    <button {...shared} type={type} disabled={inert}>
      {content}
    </button>
  );
}
