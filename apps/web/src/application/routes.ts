import type { Locale } from './i18n';

export type RouteKey =
  | 'catalog'
  | 'project'
  | 'access'
  | 'account'
  | 'members'
  | 'serviceAccounts'
  | 'audit'
  | 'operations';

const segments: Record<Locale, Record<RouteKey, string>> = {
  en: {
    catalog: 'catalog',
    project: 'catalog/:projectId',
    access: 'access',
    account: 'account',
    members: 'admin/members',
    serviceAccounts: 'admin/service-accounts',
    audit: 'admin/audit',
    operations: 'admin/operations',
  },
  'pt-BR': {
    catalog: 'catalogo',
    project: 'catalogo/:projectId',
    access: 'acesso',
    account: 'conta',
    members: 'admin/membros',
    serviceAccounts: 'admin/contas-de-servico',
    audit: 'admin/auditoria',
    operations: 'admin/operacoes',
  },
};

export function localePrefix(locale: Locale): string {
  return locale === 'pt-BR' ? '/pt-br' : '/en';
}

export function routePath(locale: Locale, key: RouteKey, params?: Record<string, string>): string {
  let path = `${localePrefix(locale)}/${segments[locale][key]}`;
  for (const [name, value] of Object.entries(params ?? {})) {
    path = path.replace(`:${name}`, encodeURIComponent(value));
  }
  return path;
}

export function routeSegment(locale: Locale, key: RouteKey): string {
  return segments[locale][key];
}

export function switchLocalePath(pathname: string, from: Locale, to: Locale): string {
  for (const key of Object.keys(segments[from]) as RouteKey[]) {
    const source = segments[from][key];
    const target = segments[to][key];
    if (!source.includes(':') && pathname === `${localePrefix(from)}/${source}`) {
      return `${localePrefix(to)}/${target}`;
    }
    if (key === 'project') {
      const sourcePrefix = `${localePrefix(from)}/${source.split('/:')[0]}/`;
      if (pathname.startsWith(sourcePrefix)) {
        const id = pathname.slice(sourcePrefix.length);
        return routePath(to, key, { projectId: decodeURIComponent(id) });
      }
    }
  }
  return routePath(to, 'catalog');
}
