import type { ReactNode } from 'react';

/**
 * Screen-reader-only text. Used for chart alternatives, table captions, glyph-only status labels
 * and the "opens in a new tab" style disclosures.
 */
export interface VisuallyHiddenProps {
  readonly children: ReactNode;
  /** A focusable variant reveals itself on focus, the way the skip link does. */
  readonly focusable?: boolean;
}

export function VisuallyHidden({ children, focusable }: VisuallyHiddenProps) {
  return <span className={focusable ? 'opi-skip' : 'opi-vh'}>{children}</span>;
}
