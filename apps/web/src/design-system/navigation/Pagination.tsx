import type { CSSProperties } from 'react';

import { Icon } from '../core/Icon';

/**
 * Cursor pagination behind numbered previous/current/next pages, per the API contract: the browser
 * retains the cursor chain for the current query, and changing filters resets that history.
 */
export interface PaginationProps {
  readonly page?: number;
  readonly pageCount?: number;
  readonly hasMore?: boolean;
  readonly pageSize?: number;
  readonly total?: number;
  readonly onPrev?: () => void;
  readonly onNext?: () => void;
  readonly label?: string;
  readonly previousLabel?: string;
  readonly nextLabel?: string;
  readonly pageLabel?: string;
  readonly ofLabel?: string;
  readonly perPageLabel?: string;
  readonly paginationLabel?: string;
  readonly style?: CSSProperties;
}

export function Pagination({
  page = 1,
  pageCount,
  hasMore,
  pageSize = 50,
  total,
  onPrev,
  onNext,
  label = 'Results',
  previousLabel = 'Previous',
  nextLabel = 'Next',
  pageLabel = 'Page',
  ofLabel = 'of',
  perPageLabel = 'per page',
  paginationLabel,
  style,
}: PaginationProps) {
  const canPrev = page > 1;
  const canNext = hasMore !== undefined ? hasMore : pageCount ? page < pageCount : false;
  const button = (direction: 'prev' | 'next', enabled: boolean, onClick?: () => void) => (
    <button
      type="button"
      className="opi-btn opi-btn--secondary"
      onClick={enabled ? onClick : undefined}
      disabled={!enabled}
      style={{ height: 'var(--control-h-md)', padding: '0 var(--space-1)' }}
    >
      {direction === 'prev' ? <Icon name="chevron-left" size={15} /> : null}
      <span>{direction === 'prev' ? previousLabel : nextLabel}</span>
      {direction === 'next' ? <Icon name="chevron-right" size={15} /> : null}
    </button>
  );

  return (
    <nav
      aria-label={paginationLabel ?? `${label} pagination`}
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 'var(--space-15)',
        flexWrap: 'wrap',
        paddingTop: 'var(--space-15)',
        ...style,
      }}
    >
      <p
        style={{
          font: 'var(--type-caption)',
          color: 'var(--text-secondary)',
          fontVariantNumeric: 'tabular-nums',
        }}
      >
        {pageLabel} {page}
        {pageCount ? ` ${ofLabel} ${pageCount}` : ''}
        {typeof total === 'number'
          ? ` · ${total} ${label.toLowerCase()}`
          : ` · ${pageSize} ${perPageLabel}`}
      </p>
      <div style={{ display: 'flex', gap: 'var(--space-05)' }}>
        {button('prev', canPrev, onPrev)}
        {button('next', canNext, onNext)}
      </div>
    </nav>
  );
}
