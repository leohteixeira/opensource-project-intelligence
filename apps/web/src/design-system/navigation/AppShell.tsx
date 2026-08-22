import type { CSSProperties, ReactNode } from 'react';

import { Avatar } from '../core/Avatar';
import { Icon } from '../core/Icon';
import type { IconName } from '../core/icons';
import { Wordmark } from '../core/Wordmark';
import { Menu } from './Menu';

export interface NavItem {
  readonly key: string;
  readonly label: string;
  readonly icon: IconName;
  readonly badge?: number;
}

export interface ShellMember {
  readonly name: string;
  readonly role: string;
}

/**
 * Role-aware application frame. Desktop carries one horizontal top bar: wordmark, primary
 * destinations, then utilities and the signed-in member. Mobile keeps the five-item bottom bar and
 * moves everything else into the top-right menu. Layout is chosen with the `viewport` prop rather
 * than a media query so a screen can show either state.
 */
export interface AppShellProps {
  readonly viewport?: 'desktop' | 'mobile';
  readonly nav?: readonly NavItem[];
  readonly secondaryNav?: readonly NavItem[];
  readonly secondaryLabel?: string;
  readonly activeKey?: string;
  readonly onNavigate?: (key: string) => void;
  readonly title?: ReactNode;
  readonly titleAdornment?: ReactNode;
  readonly subtitle?: ReactNode;
  readonly onBack?: () => void;
  readonly actions?: ReactNode;
  readonly utilities?: ReactNode;
  readonly member?: ShellMember;
  readonly account?: ReactNode;
  readonly locale?: string;
  readonly skipLabel?: string;
  readonly primaryNavigationLabel?: string;
  readonly backLabel?: string;
  readonly children?: ReactNode;
  readonly sidePanel?: ReactNode;
  readonly style?: CSSProperties;
}

export function AppShell({
  viewport = 'desktop',
  nav = [],
  secondaryNav = [],
  secondaryLabel = 'Administration',
  activeKey,
  onNavigate,
  title,
  titleAdornment,
  subtitle,
  onBack,
  actions,
  utilities,
  member,
  account,
  locale = 'en',
  skipLabel = 'Skip to content',
  primaryNavigationLabel = 'Primary',
  backLabel = 'Back',
  children,
  sidePanel,
  style,
}: AppShellProps) {
  const mobile = viewport === 'mobile';

  const navItem = (item: NavItem) => (
    <button
      key={item.key}
      type="button"
      className="opi-nav"
      onClick={() => onNavigate?.(item.key)}
      aria-current={item.key === activeKey ? 'page' : undefined}
    >
      <Icon name={item.icon} size={15} />
      <span>{item.label}</span>
      {typeof item.badge === 'number' && item.badge > 0 ? (
        <span
          style={{
            font: 'var(--type-caption)',
            fontWeight: 'var(--weight-semibold)',
            fontVariantNumeric: 'tabular-nums',
            background: 'var(--critical-bg)',
            color: 'var(--critical-fg)',
            borderRadius: 'var(--radius-pill)',
            padding: '0 6px',
            minWidth: 18,
            textAlign: 'center',
          }}
        >
          {item.badge}
        </span>
      ) : null}
    </button>
  );

  const bottomItem = (item: NavItem) => {
    const on = item.key === activeKey;

    return (
      <button
        key={item.key}
        type="button"
        onClick={() => onNavigate?.(item.key)}
        aria-current={on ? 'page' : undefined}
        style={{
          flex: 1,
          minWidth: 0,
          display: 'grid',
          justifyItems: 'center',
          gap: 2,
          border: 0,
          background: 'transparent',
          padding: 'var(--space-075) 2px',
          color: on ? 'var(--text-link)' : 'var(--text-secondary)',
          cursor: 'pointer',
          minHeight: 'var(--bottombar-h)',
        }}
      >
        <Icon name={item.icon} size={19} strokeWidth={on ? 2.1 : 1.75} />
        <span
          style={{
            font: 'var(--type-caption)',
            fontWeight: on ? 'var(--weight-semibold)' : 'var(--weight-regular)',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            maxWidth: '100%',
          }}
        >
          {item.label}
        </span>
      </button>
    );
  };

  const memberBlock =
    account !== undefined ? (
      account
    ) : member ? (
      <span
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 'var(--space-1)',
          paddingLeft: 'var(--space-1)',
          minWidth: 0,
        }}
      >
        <Avatar name={member.name} tone="accent" />
        <span className="opi-member-text" style={{ maxWidth: 168 }}>
          <span
            style={{
              font: 'var(--type-ui)',
              fontWeight: 'var(--weight-semibold)',
              color: 'var(--text-primary)',
            }}
          >
            {member.name}
          </span>
          <span style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
            {member.role}
          </span>
        </span>
      </span>
    ) : null;

  return (
    <div
      style={{
        minHeight: '100%',
        display: 'flex',
        flexDirection: 'column',
        background: 'var(--surface-page)',
        ...style,
      }}
    >
      <a className="opi-skip" href="#main">
        {skipLabel}
      </a>
      <header
        style={{
          position: 'sticky',
          top: 0,
          zIndex: 'var(--z-sticky)',
          height: 'var(--nav-h)',
          flex: 'none',
          display: 'flex',
          alignItems: 'center',
          gap: 'var(--space-2)',
          padding: mobile ? '0 var(--space-2)' : '0 var(--space-3)',
          background: 'var(--surface-nav)',
          borderBottom: 'var(--border-default)',
        }}
      >
        <Wordmark variant={mobile ? 'mark' : 'inline'} style={{ flex: 'none' }} />
        {!mobile ? (
          <nav aria-label={primaryNavigationLabel} className="opi-topnav">
            <div>
              {nav.map(navItem)}
              {secondaryNav.length ? (
                <Menu
                  align="start"
                  items={secondaryNav.map((item) => ({
                    key: item.key,
                    label: item.label,
                    icon: item.icon,
                    onSelect: () => onNavigate?.(item.key),
                  }))}
                  trigger={
                    <button
                      type="button"
                      className="opi-nav"
                      aria-current={
                        secondaryNav.some((item) => item.key === activeKey) ? 'page' : undefined
                      }
                    >
                      <span>{secondaryLabel}</span>
                      <Icon name="chevron-down" size={14} />
                    </button>
                  }
                />
              ) : null}
            </div>
          </nav>
        ) : (
          <span style={{ flex: 1 }} />
        )}
        <div className="opi-topbar-right">
          {utilities}
          {memberBlock ? (
            <>
              <span
                style={{
                  width: 1,
                  height: 28,
                  background: 'var(--gray-100)',
                  margin: '0 var(--space-05)',
                }}
              />
              {memberBlock}
            </>
          ) : null}
          <span className="opi-vh">
            {locale === 'pt-BR' ? 'Idioma: portugues do Brasil' : 'Language: English'}
          </span>
        </div>
      </header>
      <div style={{ flex: 1, minWidth: 0, display: 'flex' }}>
        <main
          id="main"
          style={{
            flex: 1,
            minWidth: 0,
            width: '100%',
            maxWidth: sidePanel && !mobile ? 'none' : 'var(--content-max)',
            margin: sidePanel && !mobile ? 0 : '0 auto',
            padding: mobile
              ? 'var(--space-2) var(--space-2) calc(var(--bottombar-h) + var(--space-4))'
              : 'var(--space-3) var(--space-3) var(--space-6)',
          }}
        >
          {title ? (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 'var(--space-15)',
                flexWrap: 'wrap',
                marginBottom: 'var(--space-25)',
              }}
            >
              {onBack ? (
                <button
                  type="button"
                  className="opi-btn opi-btn--secondary"
                  onClick={onBack}
                  aria-label={backLabel}
                  style={{ width: 36, height: 36, padding: 0, borderRadius: 'var(--radius-pill)' }}
                >
                  <Icon name="arrow-left" size={16} />
                </button>
              ) : null}
              <div style={{ flex: 1, minWidth: 0, display: 'grid', gap: 2 }}>
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'baseline',
                    gap: 'var(--space-15)',
                    flexWrap: 'wrap',
                  }}
                >
                  <h1 style={{ font: mobile ? 'var(--type-section)' : 'var(--type-page-title)' }}>
                    {title}
                  </h1>
                  {titleAdornment}
                </div>
                {subtitle ? (
                  <p style={{ font: 'var(--type-mono-xs)', color: 'var(--text-secondary)' }}>
                    {subtitle}
                  </p>
                ) : null}
              </div>
              {actions ? (
                <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-1)' }}>
                  {actions}
                </div>
              ) : null}
            </div>
          ) : null}
          {children}
        </main>
        {sidePanel && !mobile ? (
          <aside
            aria-label="Assistant"
            style={{
              width: 'var(--panel-w)',
              flex: 'none',
              borderLeft: 'var(--border-default)',
              background: 'var(--surface-card)',
            }}
          >
            {sidePanel}
          </aside>
        ) : null}
      </div>
      {mobile ? (
        <nav
          aria-label={primaryNavigationLabel}
          style={{
            position: 'sticky',
            bottom: 0,
            display: 'flex',
            background: 'var(--surface-card)',
            borderTop: 'var(--border-default)',
          }}
        >
          {nav.slice(0, 5).map(bottomItem)}
        </nav>
      ) : null}
    </div>
  );
}
