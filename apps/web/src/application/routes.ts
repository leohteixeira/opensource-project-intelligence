import type { Locale } from './i18n';

export type RouteKey =
  | 'catalog'
  | 'project'
  | 'portfolio'
  | 'projects'
  | 'workspaceProject'
  | 'projectSources'
  | 'projectJobs'
  | 'projectLifecycle'
  | 'projectHealth'
  | 'projectContributors'
  | 'projectAdoptionSecurity'
  | 'projectTopics'
  | 'projectReleases'
  | 'projectKnowledge'
  | 'projectTrends'
  | 'analysisRun'
  | 'compare'
  | 'comparison'
  | 'radar'
  | 'alerts'
  | 'access'
  | 'account'
  | 'members'
  | 'serviceAccounts'
  | 'audit'
  | 'operations'
  | 'policies';

const segments: Record<Locale, Record<RouteKey, string>> = {
  en: {
    catalog: 'catalog',
    project: 'catalog/:projectId',
    portfolio: 'portfolio',
    projects: 'projects',
    workspaceProject: 'projects/:projectId',
    projectSources: 'projects/:projectId/sources',
    projectJobs: 'projects/:projectId/jobs',
    projectLifecycle: 'projects/:projectId/lifecycle',
    projectHealth: 'projects/:projectId/health',
    projectContributors: 'projects/:projectId/contributors',
    projectAdoptionSecurity: 'projects/:projectId/adoption-security',
    projectTopics: 'projects/:projectId/topics',
    projectReleases: 'projects/:projectId/releases',
    projectKnowledge: 'projects/:projectId/knowledge',
    projectTrends: 'projects/:projectId/trends',
    analysisRun: 'analysis-runs/:runId',
    compare: 'compare',
    comparison: 'compare/:comparisonId',
    radar: 'radar',
    alerts: 'alerts',
    access: 'access',
    account: 'account',
    members: 'admin/members',
    serviceAccounts: 'admin/service-accounts',
    audit: 'admin/audit',
    operations: 'admin/operations',
    policies: 'admin/policies',
  },
  'pt-BR': {
    catalog: 'catalogo',
    project: 'catalogo/:projectId',
    portfolio: 'portfolio',
    projects: 'projetos',
    workspaceProject: 'projetos/:projectId',
    projectSources: 'projetos/:projectId/fontes',
    projectJobs: 'projetos/:projectId/tarefas',
    projectLifecycle: 'projetos/:projectId/ciclo-de-vida',
    projectHealth: 'projetos/:projectId/saude',
    projectContributors: 'projetos/:projectId/colaboradores',
    projectAdoptionSecurity: 'projetos/:projectId/adocao-seguranca',
    projectTopics: 'projetos/:projectId/topicos',
    projectReleases: 'projetos/:projectId/versoes',
    projectKnowledge: 'projetos/:projectId/conhecimento',
    projectTrends: 'projetos/:projectId/tendencias',
    analysisRun: 'execucoes-ia/:runId',
    compare: 'comparar',
    comparison: 'comparar/:comparisonId',
    radar: 'radar',
    alerts: 'alertas',
    access: 'acesso',
    account: 'conta',
    members: 'admin/membros',
    serviceAccounts: 'admin/contas-de-servico',
    audit: 'admin/auditoria',
    operations: 'admin/operacoes',
    policies: 'admin/politicas',
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
  const keys = (Object.keys(segments[from]) as RouteKey[]).sort(
    (left, right) => segments[from][right].length - segments[from][left].length,
  );
  for (const key of keys) {
    const source = segments[from][key];
    const target = segments[to][key];
    if (!source.includes(':') && pathname === `${localePrefix(from)}/${source}`) {
      return `${localePrefix(to)}/${target}`;
    }
    const parameter = source.match(/:([A-Za-z][A-Za-z0-9]*)/);
    if (parameter) {
      const token = parameter[0];
      const name = parameter[1];
      if (!name) continue;
      const [beforeID = '', afterID = ''] = source.split(token);
      const sourcePrefix = `${localePrefix(from)}/${beforeID}`;
      if (pathname.startsWith(sourcePrefix) && pathname.endsWith(afterID)) {
        const id = pathname.slice(
          sourcePrefix.length,
          pathname.length - afterID.length || undefined,
        );
        if (!id || id.includes('/')) continue;
        return routePath(to, key, { [name]: decodeURIComponent(id) });
      }
    }
  }
  return routePath(to, 'catalog');
}
