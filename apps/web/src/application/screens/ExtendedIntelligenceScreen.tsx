import { useMutation, useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { useParams } from 'react-router';

import {
  Banner,
  Button,
  EmptyState,
  Panel,
  Skeleton,
  StatusBadge,
  TextField,
  type StatusKey,
} from '../../design-system';
import {
  askProject,
  correctTopic,
  fetchAdoption,
  fetchAnalysisRun,
  fetchReleases,
  fetchSecurity,
  fetchTopics,
  rerunAnalysis,
  searchKnowledge,
  selectAnalysisRun,
  submitAnalysisFeedback,
  type Document,
} from '../api';
import { useApplication } from '../router';

const layout = { display: 'grid', gap: 'var(--space-2)' } as const;

function title(en: string, pt: string, locale: string) {
  return locale === 'pt-BR' ? pt : en;
}

function text(value: unknown, fallback = '—') {
  return typeof value === 'string' || typeof value === 'number' ? String(value) : fallback;
}

function list(value: unknown): Document[] {
  return Array.isArray(value) ? (value as Document[]) : [];
}

function visibleStatus(value: unknown): StatusKey {
  const status = text(value, 'unknown');
  const supported: StatusKey[] = [
    'available',
    'unknown',
    'stale',
    'queued',
    'running',
    'succeeded',
    'failed',
    'cancelled',
  ];
  if (supported.includes(status as StatusKey)) return status as StatusKey;
  return status === 'incomparable' ? 'insufficient_data' : 'unknown';
}

function LoadingOrError({ pending, error }: { pending: boolean; error: boolean }) {
  if (pending) return <Skeleton height={260} />;
  if (error)
    return (
      <Banner tone="critical" title="Intelligence unavailable">
        The latest immutable response could not be loaded. Retry without changing the evidence
        cutoff.
      </Banner>
    );
  return null;
}

export function AdoptionSecurityIntelligenceScreen() {
  const { projectId = '' } = useParams();
  const { locale } = useApplication();
  const adoption = useQuery({
    queryKey: ['adoption', projectId],
    queryFn: () => fetchAdoption(projectId),
    enabled: Boolean(projectId),
  });
  const security = useQuery({
    queryKey: ['security', projectId],
    queryFn: () => fetchSecurity(projectId),
    enabled: Boolean(projectId),
  });
  const securityValue = security.data?.security as Document | undefined;
  const advisories = list(securityValue?.advisories);

  return (
    <section style={layout} data-testid="adoption-security">
      <header>
        <h1>{title('Adoption and security', 'Adoção e segurança', locale)}</h1>
        <p>
          Source-contextual public evidence. Registry units are never combined into a universal
          score.
        </p>
      </header>
      <LoadingOrError
        pending={adoption.isPending || security.isPending}
        error={adoption.isError || security.isError}
      />
      <Panel title="Adoption signals" meta="one panel per registry · immutable observations">
        {(adoption.data?.items.length ?? 0) === 0 ? (
          <EmptyState glyph="package" title="No registry evidence">
            Adoption is unknown, not zero.
          </EmptyState>
        ) : (
          <div className="opi-intelligence-grid">
            {adoption.data?.items.map((item, index) => (
              <article className="opi-intelligence-card" key={text(item.id, String(index))}>
                <strong>
                  {text(item.registry)} · {text(item.package)}
                </strong>
                <p>
                  {text(item.value, 'Unknown')} {text(item.unit, '')}
                </p>
                <p className="opi-intelligence-meta">
                  population {text(item.population)} · observed {text(item.observed_at)}
                </p>
                <StatusBadge status={visibleStatus(item.status)} size="sm" />
              </article>
            ))}
          </div>
        )}
      </Panel>
      <Panel
        title="Public security evidence"
        meta={text(securityValue?.coverage_note, 'coverage is explicit')}
      >
        <Banner tone="neutral" title="This is not a vulnerability scan">
          No public advisory is never presented as a safety claim.
        </Banner>
        {advisories.length === 0 ? (
          <EmptyState glyph="shield-alert" title="No public advisories found">
            The result remains unknown and qualified by source coverage.
          </EmptyState>
        ) : (
          <div style={layout}>
            {advisories.map((item, index) => (
              <article className="opi-intelligence-card" key={text(item.id, String(index))}>
                <strong>{text(item.external_id)}</strong>
                <p>{text(item.summary)}</p>
                <span>
                  {text(item.published_at)} · {text(item.state)}
                </span>
              </article>
            ))}
          </div>
        )}
      </Panel>
    </section>
  );
}

export function TopicIntelligenceScreen() {
  const { projectId = '' } = useParams();
  const { locale, session } = useApplication();
  const [label, setLabel] = useState('Canonical maintenance');
  const topics = useQuery({
    queryKey: ['topics', projectId],
    queryFn: () => fetchTopics(projectId),
    enabled: Boolean(projectId),
  });
  const correction = useMutation({
    mutationFn: async () => {
      const first = topics.data?.items[0];
      const candidate = first?.candidate as Document | undefined;
      if (!candidate?.id) throw new Error('No topic is available for correction.');
      return correctTopic(
        session,
        projectId,
        text(candidate.id),
        { action: 'rename', label, reason: 'Analyst review' },
        list(first?.history).length,
      );
    },
    onSuccess: () => topics.refetch(),
  });
  return (
    <section style={layout} data-testid="topic-intelligence">
      <header>
        <h1>{title('Issue and discussion topics', 'Tópicos de issues e discussões', locale)}</h1>
        <p>Deterministic mutual-kNN candidates with attributed Analyst corrections.</p>
      </header>
      <LoadingOrError pending={topics.isPending} error={topics.isError} />
      {(topics.data?.items.length ?? 0) > 0 ? (
        <Panel title="Analyst correction" meta="append-only · attributed · optimistic version">
          <TextField
            id="topic-label"
            label="Canonical label"
            value={label}
            onChange={(event) => setLabel(event.target.value)}
          />
          <Button
            onClick={() => correction.mutate()}
            disabled={!label.trim() || correction.isPending}
          >
            Save correction
          </Button>
          {correction.isSuccess ? (
            <Banner tone="info" title="Correction recorded">
              Generated content was not overwritten; the correction will constrain reprocessing.
            </Banner>
          ) : null}
          {correction.isError ? (
            <Banner tone="critical" title="Correction conflict">
              Reload the current topic version before applying another correction.
            </Banner>
          ) : null}
        </Panel>
      ) : null}
      {(topics.data?.items.length ?? 0) === 0 && !topics.isPending ? (
        <EmptyState glyph="message-square" title="Insufficient topic evidence">
          No eligible content yields insufficient data, never zero prevalence.
        </EmptyState>
      ) : (
        <div className="opi-intelligence-grid">
          {topics.data?.items.map((item, index) => {
            const candidate = item.candidate as Document | undefined;
            return (
              <Panel
                key={text(candidate?.id, String(index))}
                title={text(item.label, 'Emerging topic')}
                meta={`${text(candidate?.algorithm_version, 'topic-v1')} · ${list(item.history).length} corrections`}
              >
                <p>{list(item.members).length} canonical members</p>
                <StatusBadge
                  status={candidate?.retired_at ? 'stale' : 'available'}
                  label={candidate?.retired_at ? 'Retired' : 'Current'}
                  size="sm"
                />
                <p className="opi-intelligence-meta">Generated label history remains immutable.</p>
              </Panel>
            );
          })}
        </div>
      )}
    </section>
  );
}

export function ReleaseIntelligenceScreen() {
  const { projectId = '' } = useParams();
  const { locale } = useApplication();
  const releases = useQuery({
    queryKey: ['releases', projectId],
    queryFn: () => fetchReleases(projectId),
    enabled: Boolean(projectId),
  });
  return (
    <section style={layout} data-testid="release-intelligence">
      <header>
        <h1>{title('Release intelligence', 'Inteligência de versões', locale)}</h1>
        <p>Canonical metadata stays usable when model analysis is unavailable.</p>
      </header>
      <LoadingOrError pending={releases.isPending} error={releases.isError} />
      {(releases.data?.items.length ?? 0) === 0 && !releases.isPending ? (
        <EmptyState glyph="tag" title="No releases collected">
          Link a public release source to begin.
        </EmptyState>
      ) : (
        <div style={layout}>
          {releases.data?.items.map((item, index) => (
            <Panel
              key={text(item.id, String(index))}
              title={`${text(item.tag)} · ${text(item.title, 'Untitled release')}`}
              meta={`${text(item.published_at)} · ${text(item.language, 'und')}`}
              status={
                <StatusBadge
                  status={item.state === 'withdrawn' ? 'stale' : 'available'}
                  label={text(item.state)}
                  size="sm"
                />
              }
            >
              <p>{text(item.body, 'No changelog evidence was collected.')}</p>
              <p className="opi-intelligence-meta">
                Evidence {text(item.evidence_id)} · analysis{' '}
                {item.analysis_run_id ? `run ${text(item.analysis_run_id)}` : 'unavailable'}
              </p>
            </Panel>
          ))}
        </div>
      )}
    </section>
  );
}

export function KnowledgeIntelligenceScreen() {
  const { projectId = '' } = useParams();
  const { locale, session } = useApplication();
  const [question, setQuestion] = useState('How are upgrades handled?');
  const search = useMutation({ mutationFn: () => searchKnowledge(projectId, question) });
  const ask = useMutation({ mutationFn: () => askProject(session, projectId, question) });
  const results = list(search.data?.items);
  return (
    <section style={layout} data-testid="knowledge-intelligence">
      <header>
        <h1>{title('Documentation knowledge', 'Conhecimento da documentação', locale)}</h1>
        <p>
          Authorized snapshot search with deterministic lexical fallback and immutable citations.
        </p>
      </header>
      <Panel
        title="Search collected snapshots"
        meta="project/source/cutoff filters apply before ranking"
      >
        <TextField
          id="knowledge-question"
          label="Question"
          value={question}
          onChange={(event) => setQuestion(event.target.value)}
          hint="Answers never use the model's private knowledge."
        />
        <div className="opi-action-row">
          <Button onClick={() => search.mutate()} disabled={!question.trim() || search.isPending}>
            Search snapshots
          </Button>
          <Button
            variant="secondary"
            onClick={() => ask.mutate()}
            disabled={!question.trim() || ask.isPending}
          >
            Ask with citations
          </Button>
        </div>
      </Panel>
      {search.isError || ask.isError ? (
        <Banner tone="critical" title="Request failed">
          Refine the question or inspect source coverage.
        </Banner>
      ) : null}
      {search.isSuccess && results.length === 0 ? (
        <EmptyState glyph="search-x" title="No indexed evidence">
          No answer was fabricated. Add or refresh a public documentation source.
        </EmptyState>
      ) : null}
      {results.map((result, index) => {
        const chunk = result.chunk as Document | undefined;
        return (
          <Panel
            key={text(chunk?.id, String(index))}
            title={text(chunk?.heading, 'Snapshot match')}
            meta={`score ${text(result.score)} · ${list(result.modes).join(' + ')}`}
          >
            <p>{text(chunk?.text)}</p>
            <p className="opi-intelligence-meta">
              snapshot {text(chunk?.snapshot_id)} · observed {text(chunk?.observed_at)} · offsets{' '}
              {text(chunk?.start_offset)}–{text(chunk?.end_offset)}
            </p>
          </Panel>
        );
      })}
      {ask.data ? (
        <Banner tone="info" title="Analysis queued">
          Run {text(ask.data.id)} uses cutoff {text(ask.data.cutoff)} and retrieval{' '}
          {text(ask.data.retrieval_version)}.
        </Banner>
      ) : null}
    </section>
  );
}

export function AnalysisRunIntelligenceScreen() {
  const { runId = '' } = useParams();
  const { locale, session } = useApplication();
  const run = useQuery({
    queryKey: ['analysis-run', runId],
    queryFn: () => fetchAnalysisRun(runId),
    enabled: Boolean(runId),
  });
  const rerun = useMutation({ mutationFn: () => rerunAnalysis(session, runId, 'Analyst review') });
  const feedback = useMutation({
    mutationFn: () =>
      submitAnalysisFeedback(
        session,
        runId,
        'not_faithful',
        'A claim needs additional source evidence.',
      ),
  });
  const selection = useMutation({
    mutationFn: () =>
      selectAnalysisRun(
        session,
        text(run.data?.series_id),
        runId,
        Number(run.data?.selection_version ?? 0),
      ),
  });
  const evidence = list(run.data?.evidence);
  return (
    <section style={layout} data-testid="analysis-run-intelligence">
      <header>
        <h1>{title('AI run governance', 'Governança de execuções de IA', locale)}</h1>
        <p>
          Generated output is immutable; reruns, feedback and selection append attributed history.
        </p>
      </header>
      <LoadingOrError pending={run.isPending} error={run.isError} />
      {run.data ? (
        <>
          <Panel
            title={`Run ${text(run.data.id)}`}
            meta={`${text(run.data.provider, 'provider unavailable')} / ${text(run.data.model, 'no model')} · ${text(run.data.state)}`}
            status={
              <StatusBadge
                status={visibleStatus(run.data.state)}
                label={text(run.data.state)}
                size="sm"
              />
            }
          >
            <dl className="opi-definition-grid">
              <dt>Prompt</dt>
              <dd>{text(run.data.prompt_version)}</dd>
              <dt>Schema</dt>
              <dd>{text(run.data.schema_version)}</dd>
              <dt>Retrieval</dt>
              <dd>{text(run.data.retrieval_version)}</dd>
              <dt>Cutoff</dt>
              <dd>{text(run.data.cutoff)}</dd>
              <dt>Usage</dt>
              <dd>{JSON.stringify(run.data.usage ?? {})}</dd>
            </dl>
          </Panel>
          <Panel title="Immutable output and evidence" meta={`${evidence.length} citations`}>
            <pre className="opi-output-json">
              {JSON.stringify(run.data.output ?? { unavailable: true }, null, 2)}
            </pre>
            {evidence.map((item, index) => (
              <p key={index} className="opi-intelligence-meta">
                snapshot {text(item.snapshot_id)} · chunk {text(item.chunk_id)} ·{' '}
                {text(item.start_offset)}–{text(item.end_offset)}
              </p>
            ))}
          </Panel>
          <Panel
            title="Governance actions"
            meta="append-only history; generated output is immutable"
          >
            <div className="opi-action-row">
              <Button onClick={() => rerun.mutate()} disabled={rerun.isPending}>
                Create rerun
              </Button>
              <Button
                variant="secondary"
                onClick={() => feedback.mutate()}
                disabled={feedback.isPending}
              >
                Flag evidence
              </Button>
              <Button
                variant="secondary"
                onClick={() => selection.mutate()}
                disabled={selection.isPending || run.data.state !== 'succeeded'}
              >
                Select this version
              </Button>
            </div>
            {rerun.data ? (
              <Banner tone="info" title="New immutable version queued">
                Run {text(rerun.data.id)} was created; the current output is unchanged.
              </Banner>
            ) : null}
            {feedback.isSuccess ? (
              <Banner tone="info" title="Feedback recorded">
                The attributed evaluation was appended to this run.
              </Banner>
            ) : null}
            {selection.isSuccess ? (
              <Banner tone="info" title="Presented version selected">
                Selection history was appended without editing generated content.
              </Banner>
            ) : null}
            {rerun.isError || feedback.isError || selection.isError ? (
              <Banner tone="critical" title="Governance action failed">
                Reload the immutable run and retry with its current version.
              </Banner>
            ) : null}
          </Panel>
        </>
      ) : null}
    </section>
  );
}
