import { keepPreviousData, useMutation, useQuery } from '@tanstack/react-query';
import { useCallback, useEffect, useState, type FormEvent, type ReactNode } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router';

import {
  Banner,
  Button,
  EmptyState,
  Link,
  Panel,
  Skeleton,
  StatusBadge,
  TextArea,
  TextField,
} from '../../design-system';
import {
  addRepository,
  addSource,
  changeRepositoryRole,
  correctAssociation,
  deleteProject,
  fetchPortfolio,
  fetchProjectResources,
  fetchProjects,
  fetchWorkspaceProject,
  registerProject,
  requestHistory,
  requestSync,
  transitionProject,
  updateProject,
  updateSourceState,
  type Document,
  type JobDocument,
  type ProjectDocument,
} from '../api';
import { queryClient } from '../query';
import { useApplication } from '../router';
import { routePath } from '../routes';

const copy = {
  en: {
    portfolio: 'Portfolio',
    projects: 'Projects',
    register: 'Register project',
    repository: 'Public repository URL',
    noProjects: 'No active projects',
    noProjectsBody: 'Register a public repository to start an evidence-backed project.',
    retry: 'Retry',
    failed: 'The request could not be completed',
    sources: 'Sources',
    jobs: 'Jobs',
    repositories: 'Repositories',
    associations: 'Associations',
    sync: 'Synchronize',
    history: 'Request history',
    identity: 'Project identity',
    save: 'Save identity',
    lifecycle: 'Lifecycle',
    health: 'Health',
    contributors: 'Contributors',
    delete: 'Permanently delete',
  },
  'pt-BR': {
    portfolio: 'Portfólio',
    projects: 'Projetos',
    register: 'Cadastrar projeto',
    repository: 'URL pública do repositório',
    noProjects: 'Nenhum projeto ativo',
    noProjectsBody: 'Cadastre um repositório público para iniciar um projeto com evidências.',
    retry: 'Tentar novamente',
    failed: 'Não foi possível concluir a solicitação',
    sources: 'Fontes',
    jobs: 'Tarefas',
    repositories: 'Repositórios',
    associations: 'Associações',
    sync: 'Sincronizar',
    history: 'Solicitar histórico',
    identity: 'Identidade do projeto',
    save: 'Salvar identidade',
    lifecycle: 'Ciclo de vida',
    health: 'Saúde',
    contributors: 'Colaboradores',
    delete: 'Excluir permanentemente',
  },
} as const;

const grid = { display: 'grid', gap: 'var(--space-2)' } as const;

export function PortfolioScreen() {
  const { locale } = useApplication();
  const labels = copy[locale];
  const portfolio = useQuery({ queryKey: ['portfolio'], queryFn: fetchPortfolio });
  if (portfolio.isPending)
    return (
      <section style={grid}>
        <h1 style={{ font: 'var(--type-page)' }}>{labels.portfolio}</h1>
        <Skeleton height={320} />
      </section>
    );
  if (portfolio.isError && !portfolio.data)
    return (
      <section style={grid}>
        <h1 style={{ font: 'var(--type-page)' }}>{labels.portfolio}</h1>
        <Failure
          retry={() => void portfolio.refetch()}
          label={labels.failed}
          retryLabel={labels.retry}
        />
      </section>
    );
  const projects = documents(portfolio.data.projects);
  const jobs = documents(portfolio.data.active_jobs) as JobDocument[];
  return (
    <section style={grid}>
      <h1 style={{ font: 'var(--type-page)' }}>{labels.portfolio}</h1>
      {portfolio.isError ? (
        <Banner tone="attention" title={labels.failed}>
          <Button variant="secondary" onClick={() => void portfolio.refetch()}>
            {labels.retry}
          </Button>
        </Banner>
      ) : null}
      {projects.length === 0 ? (
        <EmptyState glyph="package" title={labels.noProjects}>
          {labels.noProjectsBody}
        </EmptyState>
      ) : (
        <div style={{ ...grid, gridTemplateColumns: 'repeat(auto-fit,minmax(260px,1fr))' }}>
          {projects.slice(0, 12).map((value) => {
            const project = value as unknown as ProjectDocument;
            return (
              <Panel
                key={project.id}
                title={project.name}
                status={
                  <StatusBadge
                    status={project.state === 'active' ? 'active' : 'unknown'}
                    size="sm"
                  />
                }
              >
                <p>{project.description || project.slug}</p>
                <Link href={routePath(locale, 'workspaceProject', { projectId: project.id })}>
                  {labels.sources}
                </Link>
              </Panel>
            );
          })}
        </div>
      )}
      <Panel title={labels.jobs} meta={`${jobs.length} active`}>
        <ResourceList values={jobs} empty="No active work" />
      </Panel>
    </section>
  );
}

export function ProjectsScreen() {
  const { locale, session } = useApplication();
  const labels = copy[locale];
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const q = params.get('q') ?? '';
  const state = params.get('state') ?? 'active';
  const cursor = params.get('cursor') ?? undefined;
  const [search, setSearch] = useState(q);
  const [url, setURL] = useState('');
  const projects = useQuery({
    queryKey: ['projects', state, q, cursor],
    queryFn: () => fetchProjects(state, q, cursor),
    placeholderData: keepPreviousData,
  });
  const registration = useMutation({
    mutationFn: () => registerProject(session, url),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ['projects'] });
      navigate(routePath(locale, 'workspaceProject', { projectId: result.project.id }));
    },
  });
  function submitSearch(event: FormEvent) {
    event.preventDefault();
    const next = new URLSearchParams();
    if (search.trim()) next.set('q', search.trim());
    if (state) next.set('state', state);
    setParams(next);
  }
  return (
    <section style={grid}>
      <h1 style={{ font: 'var(--type-page)' }}>{labels.projects}</h1>
      {session.role !== 'viewer' ? (
        <Panel title={labels.register}>
          <form
            onSubmit={(event) => {
              event.preventDefault();
              registration.mutate();
            }}
            style={{ display: 'flex', gap: 'var(--space-1)', flexWrap: 'wrap' }}
          >
            <TextField
              id="repository-url"
              label={labels.repository}
              type="url"
              value={url}
              onChange={(event) => setURL(event.target.value)}
              placeholder="https://github.com/owner/repository"
              style={{ flex: 1, minWidth: 240 }}
            />
            <Button type="submit" disabled={!url.trim() || registration.isPending}>
              {labels.register}
            </Button>
          </form>
          {registration.isError ? (
            <Banner tone="critical" title={labels.failed}>
              {registration.error.message}
            </Banner>
          ) : null}
        </Panel>
      ) : null}
      <form onSubmit={submitSearch} style={{ display: 'flex', gap: 'var(--space-1)' }}>
        <TextField
          id="project-search"
          type="search"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={labels.projects}
          style={{ flex: 1 }}
        />
        <Button type="submit" variant="secondary">
          Search
        </Button>
      </form>
      {projects.isPending ? (
        <Skeleton height={280} />
      ) : projects.isError ? (
        <Failure
          retry={() => void projects.refetch()}
          label={labels.failed}
          retryLabel={labels.retry}
        />
      ) : projects.data.items.length === 0 ? (
        <EmptyState glyph="package" title={labels.noProjects}>
          {labels.noProjectsBody}
        </EmptyState>
      ) : (
        <>
          <div style={{ ...grid, gridTemplateColumns: 'repeat(auto-fit,minmax(260px,1fr))' }}>
            {projects.data.items.map((project) => (
              <Panel
                key={project.id}
                title={project.name}
                eyebrow={project.slug}
                interactive
                onClick={() =>
                  navigate(routePath(locale, 'workspaceProject', { projectId: project.id }))
                }
              >
                <StatusBadge status={project.state === 'active' ? 'active' : 'unknown'} size="sm" />{' '}
                <span>{project.description}</span>
              </Panel>
            ))}
          </div>
          {projects.data.has_more && projects.data.next_cursor ? (
            <Button
              variant="secondary"
              onClick={() => {
                const next = new URLSearchParams(params);
                next.set('cursor', projects.data.next_cursor ?? '');
                setParams(next);
              }}
            >
              Next page
            </Button>
          ) : null}
        </>
      )}
    </section>
  );
}

export function WorkspaceProjectScreen({
  section = 'overview',
}: {
  section?: 'overview' | 'sources' | 'jobs' | 'lifecycle';
}) {
  const { projectId = '' } = useParams();
  const { locale, session } = useApplication();
  const labels = copy[locale];
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [repositoryURL, setRepositoryURL] = useState('');
  const [sourceURL, setSourceURL] = useState('');
  const [deletionConfirmation, setDeletionConfirmation] = useState('');
  const [streamStatus, setStreamStatus] = useState<'connected' | 'reconnecting' | 'polling'>(
    'reconnecting',
  );
  const project = useQuery({
    queryKey: ['workspace-project', projectId],
    queryFn: () => fetchWorkspaceProject(projectId),
    enabled: Boolean(projectId),
  });
  const resources = useQuery({
    queryKey: ['workspace-project-resources', projectId],
    queryFn: () => fetchProjectResources(projectId),
    enabled: Boolean(projectId),
    refetchInterval: (query) => {
      const data = query.state.data;
      return data?.jobs.items.some((value) => value.state === 'queued' || value.state === 'running')
        ? 2000
        : false;
    },
  });
  const refresh = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ['workspace-project', projectId] });
    void queryClient.invalidateQueries({ queryKey: ['workspace-project-resources', projectId] });
  }, [projectId]);
  const mutation = useMutation({
    mutationFn: async (action: string) => {
      if (!project.data) throw new Error('Project is unavailable');
      if (action === 'save')
        return updateProject(
          session,
          project.data,
          name || project.data.name,
          description || project.data.description,
        );
      if (action === 'sync') return requestSync(session, projectId);
      if (action === 'history')
        return requestHistory(
          session,
          projectId,
          new Date(Date.now() - 365 * 86400000).toISOString(),
          'Extended review',
        );
      if (action === 'repository') return addRepository(session, projectId, repositoryURL, 'core');
      if (action === 'source') return addSource(session, projectId, 'website', sourceURL);
      if (action === 'delete')
        return deleteProject(session, project.data, 'Requested from project lifecycle');
      return transitionProject(
        session,
        project.data,
        action as 'active' | 'paused' | 'archived',
        'Lifecycle change',
      );
    },
    onSuccess: refresh,
  });
  const resourceMutation = useMutation({
    mutationFn: async (request: {
      kind: 'repository-role' | 'source-state' | 'association';
      value: Document;
    }) => {
      if (request.kind === 'repository-role')
        return changeRepositoryRole(
          session,
          projectId,
          String(request.value.id),
          Number(request.value.version),
          request.value.role === 'primary' ? 'core' : 'primary',
        );
      if (request.kind === 'source-state')
        return updateSourceState(
          session,
          projectId,
          String(request.value.id),
          Number(request.value.version),
          request.value.state === 'paused' ? 'available' : 'paused',
        );
      return correctAssociation(
        session,
        projectId,
        String(request.value.id),
        'confirm',
        'Confirmed from source review',
      );
    },
    onSuccess: refresh,
  });
  const activeJob = resources.data?.jobs.items.find(
    (value) => value.state === 'queued' || value.state === 'running',
  );
  useEffect(() => {
    if (section !== 'jobs' || !activeJob?.id) return;
    const events = new EventSource(`/api/v1/jobs/${activeJob.id}/events`);
    events.onopen = () => setStreamStatus('connected');
    events.addEventListener('job.updated', () => refresh());
    events.onerror = () => {
      setStreamStatus('polling');
      events.close();
    };
    return () => events.close();
  }, [activeJob?.id, refresh, section]);
  if (project.isPending || resources.isPending) return <Skeleton height={360} />;
  if (project.isError || resources.isError || !project.data || !resources.data)
    return (
      <Failure
        retry={() => {
          void project.refetch();
          void resources.refetch();
        }}
        label={labels.failed}
        retryLabel={labels.retry}
      />
    );
  const canWrite = session.role !== 'viewer' && project.data.state !== 'archived';
  const isAdmin = session.role === 'admin';
  return (
    <section style={grid}>
      <header>
        <p style={{ font: 'var(--type-eyebrow)' }}>{project.data.slug}</p>
        <h1 style={{ font: 'var(--type-page)' }}>{project.data.name}</h1>
        <StatusBadge
          status={project.data.state === 'active' ? 'active' : 'unknown'}
          label={project.data.state}
        />
      </header>
      <nav
        style={{ display: 'flex', gap: 'var(--space-1)', flexWrap: 'wrap' }}
        aria-label="Project sections"
      >
        {(['overview', 'sources', 'jobs', 'lifecycle'] as const).map((key) => (
          <Button
            key={key}
            variant={section === key ? 'primary' : 'secondary'}
            href={routePath(
              locale,
              key === 'overview'
                ? 'workspaceProject'
                : key === 'sources'
                  ? 'projectSources'
                  : key === 'jobs'
                    ? 'projectJobs'
                    : 'projectLifecycle',
              { projectId },
            )}
          >
            {key === 'sources'
              ? labels.sources
              : key === 'jobs'
                ? labels.jobs
                : key === 'lifecycle'
                  ? labels.lifecycle
                  : labels.identity}
          </Button>
        ))}
        <Button variant="secondary" href={routePath(locale, 'projectHealth', { projectId })}>
          {labels.health}
        </Button>
        <Button variant="secondary" href={routePath(locale, 'projectContributors', { projectId })}>
          {labels.contributors}
        </Button>
      </nav>
      {mutation.isError ? (
        <Banner tone="critical" title={labels.failed}>
          {mutation.error.message}
        </Banner>
      ) : null}
      {section === 'overview' ? (
        <>
          <Panel title={labels.identity}>
            <div style={grid}>
              <TextField
                id="project-name"
                value={name}
                placeholder={project.data.name}
                onChange={(event) => setName(event.target.value)}
                disabled={!canWrite}
              />
              <TextArea
                id="project-description"
                label="Description"
                value={description}
                placeholder={project.data.description}
                onChange={(event) => setDescription(event.target.value)}
                disabled={!canWrite}
              />
              <Button
                onClick={() => mutation.mutate('save')}
                disabled={!canWrite || mutation.isPending}
              >
                {labels.save}
              </Button>
            </div>
          </Panel>
          <Panel title={labels.repositories}>
            <ResourceList values={resources.data.repositories.items} empty="No repositories" />
          </Panel>
        </>
      ) : null}
      {section === 'sources' ? (
        <>
          <Panel title={labels.repositories}>
            <ResourceList
              values={resources.data.repositories.items}
              empty="No repositories"
              action={
                canWrite
                  ? (value) => (
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => resourceMutation.mutate({ kind: 'repository-role', value })}
                      >
                        {value.role === 'primary' ? 'Make core' : 'Make primary'}
                      </Button>
                    )
                  : undefined
              }
            />
            {canWrite ? (
              <form
                onSubmit={(event) => {
                  event.preventDefault();
                  mutation.mutate('repository');
                }}
                style={{ display: 'flex', gap: 8 }}
              >
                <TextField
                  id="add-repository"
                  value={repositoryURL}
                  onChange={(event) => setRepositoryURL(event.target.value)}
                  placeholder="https://github.com/owner/repository"
                  style={{ flex: 1 }}
                />
                <Button type="submit">Add</Button>
              </form>
            ) : null}
          </Panel>
          <Panel title={labels.sources}>
            <ResourceList
              values={resources.data.sources.items}
              empty="No sources"
              action={
                canWrite
                  ? (value) => (
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => resourceMutation.mutate({ kind: 'source-state', value })}
                      >
                        {value.state === 'paused' ? 'Resume' : 'Pause'}
                      </Button>
                    )
                  : undefined
              }
            />
            {canWrite ? (
              <form
                onSubmit={(event) => {
                  event.preventDefault();
                  mutation.mutate('source');
                }}
                style={{ display: 'flex', gap: 8 }}
              >
                <TextField
                  id="add-source"
                  value={sourceURL}
                  onChange={(event) => setSourceURL(event.target.value)}
                  placeholder="https://example.com/docs"
                  style={{ flex: 1 }}
                />
                <Button type="submit">Add</Button>
              </form>
            ) : null}
          </Panel>
          <Panel title={labels.associations}>
            <ResourceList
              values={resources.data.associations.items}
              empty="No associations"
              action={
                canWrite
                  ? (value) => (
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => resourceMutation.mutate({ kind: 'association', value })}
                      >
                        Confirm association
                      </Button>
                    )
                  : undefined
              }
            />
          </Panel>
        </>
      ) : null}
      {section === 'jobs' ? (
        <>
          <Banner
            tone={streamStatus === 'connected' ? 'positive' : 'attention'}
            title={`Job updates: ${streamStatus}`}
          >
            {streamStatus === 'polling'
              ? 'Live updates are unavailable; polling every two seconds.'
              : 'Durable event replay preserves progress across reconnects.'}
          </Banner>
          <div style={{ display: 'flex', gap: 8 }}>
            <Button onClick={() => mutation.mutate('sync')} disabled={!canWrite}>
              {labels.sync}
            </Button>
            <Button
              variant="secondary"
              onClick={() => mutation.mutate('history')}
              disabled={!canWrite}
            >
              {labels.history}
            </Button>
          </div>
          <Panel title={labels.jobs}>
            <ResourceList values={resources.data.jobs.items} empty="No durable work" />
          </Panel>
        </>
      ) : null}
      {section === 'lifecycle' ? (
        <Panel title={labels.lifecycle}>
          <p>
            Current state: {project.data.state}. Existing evidence remains readable while paused or
            archived.
          </p>
          {isAdmin ? (
            <div style={grid}>
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                <Button
                  onClick={() =>
                    mutation.mutate(project.data.state === 'paused' ? 'active' : 'paused')
                  }
                >
                  {project.data.state === 'paused' ? 'Resume' : 'Pause'}
                </Button>
                <Button
                  variant="secondary"
                  onClick={() =>
                    mutation.mutate(project.data.state === 'archived' ? 'active' : 'archived')
                  }
                >
                  {project.data.state === 'archived' ? 'Restore' : 'Archive'}
                </Button>
              </div>
              <TextField
                id="delete-project-confirmation"
                value={deletionConfirmation}
                onChange={(event) => setDeletionConfirmation(event.target.value)}
                label={`Type DELETE ${project.data.slug}`}
              />
              <Button
                variant="danger"
                disabled={deletionConfirmation !== `DELETE ${project.data.slug}`}
                onClick={() => mutation.mutate('delete')}
              >
                {labels.delete}
              </Button>
            </div>
          ) : (
            <Banner tone="neutral" title="Admin action">
              Lifecycle controls are intentionally absent for this role.
            </Banner>
          )}
        </Panel>
      ) : null}
    </section>
  );
}

function documents(value: unknown): Document[] {
  return Array.isArray(value)
    ? value.filter((item): item is Document => Boolean(item) && typeof item === 'object')
    : [];
}

function ResourceList({
  values,
  empty,
  action,
}: {
  values: Document[];
  empty: string;
  action?: (value: Document) => ReactNode;
}) {
  if (values.length === 0) return <p>{empty}</p>;
  return (
    <ul style={{ ...grid, marginBottom: 'var(--space-1)' }}>
      {values.map((value, index) => (
        <li key={String(value.id ?? index)}>
          <strong>{String(value.name ?? value.kind ?? value.url ?? value.id ?? 'Resource')}</strong>
          {value.role ? <> · {String(value.role)}</> : null}
          {value.state ? <> · {String(value.state)}</> : null}
          {value.method ? (
            <>
              {' '}
              · {String(value.method)} ({String(value.confidence ?? 'unknown')})
            </>
          ) : null}
          {value.coverage_from ? (
            <>
              {' '}
              · coverage {String(value.coverage_from)}–{String(value.coverage_to ?? 'in progress')}
            </>
          ) : null}
          {value.last_success_at ? <> · last success {String(value.last_success_at)}</> : null}
          {value.progress && typeof value.progress === 'object' ? (
            <>
              {' '}
              · {String((value.progress as Document).completed ?? 0)}{' '}
              {String((value.progress as Document).unit ?? 'items')}
            </>
          ) : null}
          {value.checkpoint && typeof value.checkpoint === 'object' ? (
            <> · checkpoint {String((value.checkpoint as Document).cursor ?? '')}</>
          ) : null}
          {value.failure ? <> · {String(value.failure)}</> : null}
          {action ? <> · {action(value)}</> : null}
        </li>
      ))}
    </ul>
  );
}

function Failure({
  retry,
  label,
  retryLabel,
}: {
  retry: () => void;
  label: string;
  retryLabel: string;
}) {
  return (
    <Banner tone="critical" title={label} actions={<Button onClick={retry}>{retryLabel}</Button>} />
  );
}
