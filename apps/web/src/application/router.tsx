/* eslint-disable react-refresh/only-export-components */
import { useEffect, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import {
  createBrowserRouter,
  Navigate,
  Outlet,
  redirect,
  useLoaderData,
  useLocation,
  useNavigate,
  useOutletContext,
} from 'react-router';

import { AppShell, Banner, Button, EmptyState, Select } from '../design-system';
import { fetchSession, logout, type SessionDocument } from './api';
import i18n, { type Locale } from './i18n';
import { queryClient } from './query';
import { localePrefix, routePath, routeSegment, switchLocalePath, type RouteKey } from './routes';
import { AccountScreen } from './screens/AccountScreen';
import { AdminScreen } from './screens/AdminScreen';
import { CatalogScreen } from './screens/CatalogScreen';
import { ProjectScreen } from './screens/ProjectScreen';

export interface ApplicationContext {
  locale: Locale;
  session: SessionDocument;
  narrow: boolean;
}

interface SessionLoaderData {
  session: SessionDocument;
  offline: boolean;
}

async function sessionLoader(): Promise<SessionLoaderData> {
  try {
    const session = await queryClient.fetchQuery({
      queryKey: ['session'],
      queryFn: fetchSession,
      staleTime: 0,
    });
    return { session, offline: false };
  } catch {
    return { session: { state: 'anonymous', authenticated: false }, offline: true };
  }
}

function localizedRoutes(locale: Locale) {
  return {
    path: localePrefix(locale),
    loader: sessionLoader,
    element: <LocalizedShell locale={locale} />,
    children: [
      { index: true, loader: () => redirect(routePath(locale, 'catalog')) },
      { path: routeSegment(locale, 'catalog'), element: <CatalogScreen /> },
      { path: routeSegment(locale, 'project'), element: <ProjectScreen /> },
      { path: routeSegment(locale, 'access'), element: <AccessScreen /> },
      {
        path: routeSegment(locale, 'account'),
        element: (
          <Protected required="member">
            <AccountScreen />
          </Protected>
        ),
      },
      {
        path: routeSegment(locale, 'members'),
        element: (
          <Protected required="admin">
            <AdminScreen surface="members" />
          </Protected>
        ),
      },
      {
        path: routeSegment(locale, 'serviceAccounts'),
        element: (
          <Protected required="admin">
            <AdminScreen surface="serviceAccounts" />
          </Protected>
        ),
      },
      {
        path: routeSegment(locale, 'audit'),
        element: (
          <Protected required="admin">
            <AdminScreen surface="audit" />
          </Protected>
        ),
      },
      {
        path: routeSegment(locale, 'operations'),
        element: (
          <Protected required="admin">
            <AdminScreen surface="operations" />
          </Protected>
        ),
      },
      { path: '*', element: <NotFound locale={locale} /> },
    ],
  };
}

export const router = createBrowserRouter([
  localizedRoutes('en'),
  localizedRoutes('pt-BR'),
  { path: '/', element: <Navigate to="/en/catalog" replace /> },
  { path: '*', element: <Navigate to="/en/catalog" replace /> },
]);

function LocalizedShell({ locale }: { locale: Locale }) {
  const { session: initialSession, offline } = useLoaderData() as SessionLoaderData;
  const [session, setSession] = useState(initialSession);
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const narrow = useMedia('(max-width: 720px)');

  useEffect(() => {
    void i18n.changeLanguage(locale);
  }, [locale]);

  // The data router can revalidate this loader without remounting the localized shell.
  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => setSession(initialSession), [initialSession]);

  const activeKey = activeRoute(location.pathname, locale);
  const isAdmin = session.state === 'active' && session.role === 'admin';
  const isMember = session.state === 'active';
  const nav = [
    { key: 'catalog', label: t('catalog'), icon: 'search' as const },
    ...(isMember ? [{ key: 'account', label: t('account'), icon: 'user-cog' as const }] : []),
  ];
  const adminNav = isAdmin
    ? [
        { key: 'members', label: t('members'), icon: 'users' as const },
        { key: 'serviceAccounts', label: t('serviceAccounts'), icon: 'user-cog' as const },
        { key: 'audit', label: t('audit'), icon: 'file-text' as const },
        { key: 'operations', label: t('operations'), icon: 'activity' as const },
      ]
    : [];

  const utilities = (
    <div style={{ display: 'flex', gap: 'var(--space-1)', alignItems: 'center' }}>
      <Select
        id="shell-language"
        placeholder={t('locale')}
        value={locale}
        size="md"
        options={[
          { value: 'en', label: t('localeEnglish') },
          { value: 'pt-BR', label: t('localePortuguese') },
        ]}
        onChange={(event) =>
          navigate(
            switchLocalePath(location.pathname, locale, event.target.value as Locale) +
              location.search,
          )
        }
      />
      {session.authenticated ? (
        <Button
          size="sm"
          variant="ghost"
          onClick={() => {
            void logout(session).finally(() => {
              const anonymous = { state: 'anonymous', authenticated: false };
              queryClient.setQueryData(['session'], anonymous);
              setSession(anonymous);
              navigate(routePath(locale, 'catalog'));
            });
          }}
        >
          {t('signOut')}
        </Button>
      ) : (
        <Button
          size="sm"
          variant="primary"
          href={`/auth/login?return_to=${encodeURIComponent(location.pathname + location.search)}`}
        >
          {t('signIn')}
        </Button>
      )}
    </div>
  );

  return (
    <AppShell
      viewport={narrow ? 'mobile' : 'desktop'}
      nav={nav}
      secondaryNav={adminNav}
      secondaryLabel={t('administration')}
      activeKey={activeKey}
      onNavigate={(key) => navigate(routePath(locale, key as RouteKey))}
      utilities={utilities}
      member={
        isMember
          ? { name: session.member?.display_name ?? t('account'), role: session.role ?? '' }
          : undefined
      }
      locale={locale}
      skipLabel={t('skip')}
      primaryNavigationLabel={t('primaryNavigation')}
    >
      {offline ? (
        <Banner
          tone="attention"
          title={t('offline')}
          actions={<Button onClick={() => window.location.reload()}>{t('retry')}</Button>}
          style={{ marginBottom: 'var(--space-2)' }}
        >
          {t('offlineBody')}
        </Banner>
      ) : null}
      <Outlet context={{ locale, session, narrow } satisfies ApplicationContext} />
    </AppShell>
  );
}

function Protected({ required, children }: { required: 'member' | 'admin'; children: ReactNode }) {
  const { session, locale } = useApplication();
  const { t } = useTranslation();
  if (['pending', 'rejected', 'suspended'].includes(session.state ?? '')) {
    return <AccessState state={session.state ?? 'anonymous'} />;
  }
  if (session.state !== 'active' || (required === 'admin' && session.role !== 'admin')) {
    return (
      <EmptyState
        glyph="lock"
        title={t('unauthorized')}
        action={<Button href={routePath(locale, 'catalog')}>{t('backToCatalog')}</Button>}
      />
    );
  }
  return children;
}

function AccessScreen() {
  const { session, locale } = useApplication();
  if (session.state === 'active') {
    return <Navigate to={routePath(locale, 'account')} replace />;
  }
  return <AccessState state={session.state ?? 'anonymous'} />;
}

function AccessState({ state }: { state: string }) {
  const { t } = useTranslation();
  const { locale } = useApplication();
  const content: Record<string, [string, string]> = {
    pending: [t('accessPending'), t('accessPendingBody')],
    rejected: [t('accessRejected'), t('accessPendingBody')],
    suspended: [t('accessSuspended'), t('accessSuspendedBody')],
  };
  const [title, body] = content[state] ?? [t('sessionExpired'), t('protectedTeaserBody')];
  return (
    <EmptyState
      glyph={state === 'pending' ? 'clock' : 'lock'}
      title={title}
      action={<Button href={routePath(locale, 'catalog')}>{t('backToCatalog')}</Button>}
    >
      {body}
    </EmptyState>
  );
}

function NotFound({ locale }: { locale: Locale }) {
  const { t } = useTranslation();
  return (
    <EmptyState
      title={t('notFound')}
      action={<Button href={routePath(locale, 'catalog')}>{t('backToCatalog')}</Button>}
    />
  );
}

export function useApplication(): ApplicationContext {
  return useOutletContext<ApplicationContext>();
}

function activeRoute(pathname: string, locale: Locale): string {
  for (const key of [
    'members',
    'serviceAccounts',
    'audit',
    'operations',
    'account',
    'catalog',
  ] as RouteKey[]) {
    if (pathname.startsWith(routePath(locale, key))) return key;
  }
  return 'catalog';
}

function useMedia(query: string): boolean {
  const [matches, setMatches] = useState(() => window.matchMedia(query).matches);
  useEffect(() => {
    const media = window.matchMedia(query);
    const update = () => setMatches(media.matches);
    media.addEventListener('change', update);
    return () => media.removeEventListener('change', update);
  }, [query]);
  return matches;
}
