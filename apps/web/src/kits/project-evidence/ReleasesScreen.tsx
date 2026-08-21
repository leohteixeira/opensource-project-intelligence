import { useState, type ReactNode } from 'react';

import {
  Banner,
  Button,
  DefinitionList,
  EmptyState,
  Icon,
  Link,
  Pagination,
  Panel,
  RunMetadata,
  StatusBadge,
  Table,
  type RunMetadataLabels,
  type StatusKey,
  type TabItem,
  type TableColumn,
} from '../../design-system';
import { AiLabel, Cols, EvidenceRow, Mono, Provenance, Stack, StateBar } from './kit';
import { CUTOFF, RELEASES, RELEASE_DETAIL, type ClaimGroupData, type ReleaseRow } from './fixtures';

export type ReleaseLocale = 'en' | 'pt-BR';

interface ReleaseCopy {
  readonly states: readonly TabItem[];
  readonly timeline: string;
  readonly release: string;
  readonly published: string;
  readonly claims: string;
  readonly breaking: string;
  readonly security: string;
  readonly analysis: string;
  readonly detail: string;
  readonly noClaims: string;
  readonly rawNotes: string;
  readonly confidence: string;
  readonly openRun: string;
  readonly requestAnalysis: string;
  readonly releasesLabel: string;
  readonly translationNote: string;
  readonly aiLabel: string;
  readonly sourceLang: string;
  readonly exportAction: string;
  readonly prerelease: string;
  readonly duplicateTag: string;
  readonly duplicateDetail: string;
  readonly staleDetail: string;
  readonly failedDetail: string;
  readonly editedDetail: string;
  readonly words: Partial<Record<StatusKey, string>>;
  readonly groups: Record<ClaimGroupData['kind'], string>;
  readonly run: RunMetadataLabels;
}

/**
 * Fixed UI copy follows the locale. Evidence — release notes, changelog lines, tags — stays in its
 * original language and is labelled when a generated translation is shown instead.
 */
const COPY: Record<ReleaseLocale, ReleaseCopy> = {
  en: {
    states: [
      { value: 'analyzed', label: 'Analyzed' },
      { value: 'pending', label: 'Analysis pending' },
      { value: 'no_changelog', label: 'No changelog' },
      { value: 'provider', label: 'Provider unavailable' },
      { value: 'stale', label: 'Stale analysis' },
      { value: 'duplicate', label: 'Prerelease and duplicate tag' },
      { value: 'large', label: 'Large history' },
    ],
    timeline: 'Release timeline',
    release: 'Release',
    published: 'Published',
    claims: 'Claims',
    breaking: 'Breaking',
    security: 'Security',
    analysis: 'Analysis',
    detail: 'Release detail',
    noClaims: 'No claim of this kind was found',
    rawNotes: 'Original release notes',
    confidence: 'Confidence',
    openRun: 'Open AI run',
    requestAnalysis: 'Request analysis',
    releasesLabel: 'releases',
    translationNote: 'Generated translation from English',
    aiLabel: 'AI-extracted',
    sourceLang: 'source language: English',
    exportAction: 'Export',
    prerelease: 'prerelease',
    duplicateTag: 'duplicate tag',
    duplicateDetail: 'two records share this tag',
    staleDetail: 'changelog edited after analysis',
    failedDetail: 'provider returned invalid output',
    editedDetail: 'changelog edited 2026-08-15T09:12Z, after the run',
    words: {},
    groups: {
      breaking: 'Breaking changes',
      feature: 'Features',
      deprecation: 'Deprecations',
      security: 'Security',
      performance: 'Performance',
      dx: 'Developer experience',
    },
    run: {},
  },
  'pt-BR': {
    states: [
      { value: 'analyzed', label: 'Analisado' },
      { value: 'pending', label: 'Análise pendente' },
      { value: 'no_changelog', label: 'Sem changelog' },
      { value: 'provider', label: 'Provedor indisponível' },
      { value: 'stale', label: 'Análise desatualizada' },
      { value: 'duplicate', label: 'Pré-lançamento e tag duplicada' },
      { value: 'large', label: 'Histórico extenso' },
    ],
    timeline: 'Linha do tempo de versões',
    release: 'Versão',
    published: 'Publicado em',
    claims: 'Afirmações',
    breaking: 'Quebra de compatibilidade',
    security: 'Segurança',
    analysis: 'Análise',
    detail: 'Detalhe da versão',
    noClaims: 'Nenhuma afirmação desse tipo foi encontrada',
    rawNotes: 'Notas originais da versão',
    confidence: 'Confiança',
    openRun: 'Abrir execução de IA',
    requestAnalysis: 'Solicitar análise',
    releasesLabel: 'versões',
    translationNote: 'Tradução gerada a partir do inglês',
    aiLabel: 'Extraído por IA',
    sourceLang: 'idioma de origem: inglês',
    exportAction: 'Exportar',
    prerelease: 'pré-lançamento',
    duplicateTag: 'tag duplicada',
    duplicateDetail: 'dois registros compartilham esta tag',
    staleDetail: 'changelog editado após a análise',
    failedDetail: 'o provedor retornou saída inválida',
    editedDetail: 'changelog editado em 2026-08-15T09:12Z, depois da execução',
    words: {
      succeeded: 'Concluída',
      failed: 'Falhou',
      queued: 'Na fila',
      stale: 'Desatualizada',
      unknown: 'Desconhecida',
    },
    groups: {
      breaking: 'Quebras de compatibilidade',
      feature: 'Recursos',
      deprecation: 'Descontinuações',
      security: 'Segurança',
      performance: 'Desempenho',
      dx: 'Experiência de desenvolvimento',
    },
    run: {
      ai: 'Gerado por IA',
      state: 'Concluída',
      stale: 'Desatualizada',
      selected: 'Versão apresentada',
      providerModel: 'Provedor / modelo',
      promptVersion: 'Versão do prompt',
      language: 'Idioma',
      executed: 'Executado em',
      usage: 'Consumo',
    },
  },
};

const GROUP_TONE: Record<ClaimGroupData['kind'], string> = {
  breaking: 'critical',
  security: 'critical',
  deprecation: 'attention',
  feature: 'info',
  performance: 'info',
  dx: 'unknown',
};

function ClaimGroup({
  group,
  copy,
  showTranslation,
}: {
  readonly group: ClaimGroupData;
  readonly copy: ReleaseCopy;
  readonly showTranslation: boolean;
}) {
  return (
    <div
      style={{
        borderTop: 'var(--border-hairline)',
        paddingTop: 'var(--space-15)',
        display: 'grid',
        gap: 'var(--space-1)',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-1)' }}>
        <Icon name={group.glyph} size={15} color={`var(--${GROUP_TONE[group.kind]}-fg)`} />
        <span style={{ font: 'var(--type-subsection)', color: 'var(--text-primary)' }}>
          {copy.groups[group.kind]}
        </span>
        <Mono tone="quiet">{group.kind}</Mono>
      </div>
      {group.claims.length === 0 ? (
        <div
          style={{
            padding: 'var(--space-15)',
            background: 'var(--surface-sunken)',
            borderRadius: 'var(--radius-xs)',
          }}
        >
          <span style={{ font: 'var(--type-body)', color: 'var(--text-secondary)' }}>
            {copy.noClaims}. {group.empty}
          </span>
        </div>
      ) : (
        group.claims.map((claim) => (
          <div
            key={claim.text}
            style={{
              display: 'grid',
              gap: 'var(--space-1)',
              padding: 'var(--space-15)',
              border: 'var(--border-default)',
              borderRadius: 'var(--radius-xs)',
            }}
          >
            <div
              style={{
                display: 'flex',
                gap: 'var(--space-1)',
                alignItems: 'flex-start',
                justifyContent: 'space-between',
                flexWrap: 'wrap',
              }}
            >
              <span
                style={{ font: 'var(--type-body)', color: 'var(--text-body)', maxWidth: '68ch' }}
              >
                {claim.text}
              </span>
              <span style={{ display: 'flex', gap: 'var(--space-075)', alignItems: 'center' }}>
                {claim.confidence ? (
                  <StatusBadge
                    status="insufficient_data"
                    size="sm"
                    label={`${copy.confidence}: ${claim.confidence}`}
                  />
                ) : null}
                <AiLabel>{copy.aiLabel}</AiLabel>
              </span>
            </div>
            {claim.note ? (
              <span style={{ font: 'var(--type-caption)', color: 'var(--attention-fg)' }}>
                {claim.note}
              </span>
            ) : null}
            {showTranslation ? (
              <Mono tone="quiet">
                {copy.translationNote} · {copy.sourceLang}
              </Mono>
            ) : null}
            <EvidenceRow cites={claim.cites} />
          </div>
        ))
      )}
    </div>
  );
}

/**
 * The pt-BR run header. `RunMetadata` accepts a `labels` override, but composing it here from
 * `DefinitionList` keeps the localized surface independent of that component's copy defaults.
 */
function LocalizedRunHeader({
  copy,
  actions,
}: {
  readonly copy: ReleaseCopy;
  readonly actions: ReactNode;
}) {
  const run = RELEASE_DETAIL.run;

  return (
    <div style={{ display: 'grid', gap: 'var(--space-1)', minWidth: 0 }}>
      <div
        style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-1)', flexWrap: 'wrap' }}
      >
        <AiLabel>{copy.run.ai}</AiLabel>
        <StatusBadge status="succeeded" size="sm" label={copy.run.state} />
        <span
          style={{
            font: 'var(--type-caption)',
            color: 'var(--positive-fg)',
            fontWeight: 'var(--weight-medium)',
          }}
        >
          {copy.run.selected}
        </span>
        <Mono tone="quiet">
          {run.versionLabel} · {run.runId}
        </Mono>
        <span style={{ marginLeft: 'auto', display: 'flex', gap: 'var(--space-05)' }}>
          {actions}
        </span>
      </div>
      <DefinitionList
        dense
        columns={2}
        items={[
          {
            label: copy.run.providerModel ?? '',
            value: `${run.provider} · ${run.model}`,
            mono: true,
          },
          { label: copy.run.promptVersion ?? '', value: run.promptVersion, mono: true },
          { label: copy.run.language ?? '', value: run.language, mono: true },
          { label: copy.run.executed ?? '', value: run.executedAt, mono: true },
          { label: copy.run.usage ?? '', value: run.usage, mono: true },
        ]}
      />
    </div>
  );
}

export function ReleasesScreen({
  locale,
  onOpenRun,
}: {
  readonly locale: ReleaseLocale;
  readonly onOpenRun: () => void;
}) {
  const copy = COPY[locale];
  const [state, setState] = useState('analyzed');
  const [page, setPage] = useState(1);
  const portuguese = locale === 'pt-BR';

  let rows: readonly ReleaseRow[] = RELEASES;

  if (state === 'duplicate') {
    rows = [
      ...RELEASES,
      {
        tag: 'v1.24.4',
        date: '2026-07-22',
        state: 'unknown',
        prerelease: false,
        duplicate: true,
        runId: null,
        claims: null,
        breaking: null,
        security: null,
      },
    ];
  }
  if (state === 'large') {
    rows = [
      ...RELEASES,
      ...RELEASES.map((release, index) => ({
        ...release,
        tag: `v1.2${index}.0`,
        date: `2025-0${(index % 9) + 1}-14`,
      })),
    ];
  }

  const columns: readonly TableColumn<ReleaseRow>[] = [
    {
      key: 'tag',
      header: copy.release,
      mono: true,
      render: (row) => (
        <span
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--space-075)',
            flexWrap: 'wrap',
          }}
        >
          <Link href="#" size="sm">
            {row.tag}
          </Link>
          {row.prerelease ? (
            <StatusBadge status="queued" size="sm" label={copy.prerelease} />
          ) : null}
          {row.duplicate ? (
            <StatusBadge
              status="unknown"
              size="sm"
              label={copy.duplicateTag}
              detail={copy.duplicateDetail}
            />
          ) : null}
        </span>
      ),
    },
    { key: 'date', header: copy.published, mono: true },
    {
      key: 'state',
      header: copy.analysis,
      render: (row) => (
        <StatusBadge
          status={row.state}
          size="sm"
          label={copy.words[row.state]}
          detail={
            row.state === 'stale'
              ? copy.staleDetail
              : row.state === 'failed'
                ? copy.failedDetail
                : undefined
          }
        />
      ),
    },
    {
      key: 'claims',
      header: copy.claims,
      numeric: true,
      render: (row) =>
        row.claims === null ? <span style={{ color: 'var(--text-tertiary)' }}>—</span> : row.claims,
    },
    {
      key: 'breaking',
      header: copy.breaking,
      numeric: true,
      render: (row) =>
        row.breaking === null ? (
          <span style={{ color: 'var(--text-tertiary)' }}>—</span>
        ) : (
          row.breaking
        ),
    },
    {
      key: 'security',
      header: copy.security,
      numeric: true,
      render: (row) =>
        row.security === null ? (
          <span style={{ color: 'var(--text-tertiary)' }}>—</span>
        ) : (
          row.security
        ),
    },
  ];

  const detailPanel = () => {
    if (state === 'pending') {
      return (
        <Panel
          title={RELEASE_DETAIL.tag}
          meta={`published ${RELEASE_DETAIL.date} · ${copy.analysis}: queued`}
          status={
            <StatusBadge
              status="queued"
              size="sm"
              label={copy.words.queued}
              detail={portuguese ? 'posição 3 de 7' : 'position 3 of 7'}
            />
          }
        >
          <Stack gap="var(--space-15)">
            <EmptyState
              glyph="clock"
              title={
                portuguese
                  ? 'A análise desta versão está na fila'
                  : 'Analysis of this release is queued'
              }
            >
              {portuguese
                ? 'As notas originais já foram coletadas e podem ser lidas abaixo. Nenhuma afirmação é exibida antes de a execução terminar.'
                : 'The original release notes were already collected and can be read below. No claim is shown before the run completes.'}
            </EmptyState>
            <Panel
              title={copy.rawNotes}
              meta="collected 2026-08-09T18:04Z · original language: English"
            >
              <Mono>
                Added: workflow update per-request timeout. Removed: pendingActivities from the v1
                DescribeWorkflowExecution response. Requires Go 1.24.
              </Mono>
            </Panel>
          </Stack>
        </Panel>
      );
    }

    if (state === 'no_changelog') {
      return (
        <Panel title="v1.24.1" meta="published 2026-05-11 · no changelog file and no release body">
          <EmptyState
            glyph="circle-dashed"
            title={
              portuguese
                ? 'Esta versão não tem evidência textual'
                : 'This release carries no textual evidence'
            }
            action={
              <Button variant="secondary" iconStart="git-branch">
                {portuguese ? 'Ver commits da tag' : 'View commits in the tag'}
              </Button>
            }
          >
            {portuguese
              ? 'A tag existe, mas não há CHANGELOG.md nem corpo de release. Afirmações são Desconhecidas, não zero: 42 commits foram coletados e permanecem disponíveis como evidência.'
              : 'The tag exists but there is no CHANGELOG.md and no release body. Claims are Unknown rather than zero: 42 commits were collected and remain available as evidence.'}
          </EmptyState>
        </Panel>
      );
    }

    if (state === 'provider') {
      return (
        <Panel
          title={RELEASE_DETAIL.tag}
          meta={`published ${RELEASE_DETAIL.date}`}
          status={
            <StatusBadge
              status="failed"
              size="sm"
              label={copy.words.failed}
              detail={
                portuguese
                  ? 'provedor indisponível às 2026-08-09T18:44Z'
                  : 'provider unavailable at 2026-08-09T18:44Z'
              }
            />
          }
          actions={
            <Button variant="secondary" size="sm" iconStart="history">
              {copy.requestAnalysis}
            </Button>
          }
        >
          <Stack gap="var(--space-15)">
            <Banner
              tone="attention"
              title={
                portuguese ? 'O provedor de IA está indisponível' : 'The AI provider is unavailable'
              }
              actions={
                <Button variant="secondary" size="md" onClick={onOpenRun}>
                  {copy.openRun}
                </Button>
              }
            >
              {portuguese
                ? 'Nenhuma afirmação foi extraída. A versão coletada é preservada integralmente e pode ser lida na origem. Métricas determinísticas e saúde não são afetadas.'
                : 'No claim was extracted. The collected release is retained in full and can be read at its source. Deterministic metrics and health are unaffected.'}
            </Banner>
            <Panel
              title={copy.rawNotes}
              meta="collected 2026-08-09T18:04Z · original language: English"
              footer={
                <Provenance cutoff="2026-08-09T18:04Z" version="raw evidence, no model output" />
              }
            >
              <Mono>
                Added: workflow update per-request timeout; Nexus retry state in describe API.
                Removed: pendingActivities. Deprecated: retryPolicy.nonRetryableErrorTypes. Requires
                Go 1.24.
              </Mono>
            </Panel>
          </Stack>
        </Panel>
      );
    }

    return (
      <Panel
        title={RELEASE_DETAIL.tag}
        eyebrow={copy.detail}
        meta={`published ${RELEASE_DETAIL.date} · ${RELEASE_DETAIL.author}`}
        status={
          state === 'stale' ? (
            <StatusBadge
              status="stale"
              size="sm"
              label={copy.words.stale}
              detail={copy.editedDetail}
            />
          ) : (
            <StatusBadge status="succeeded" size="sm" label={copy.words.succeeded} />
          )
        }
        actions={
          <Button variant="secondary" size="sm" iconStart="sparkles" onClick={onOpenRun}>
            {copy.openRun}
          </Button>
        }
        footer={
          <Provenance
            cutoff={RELEASE_DETAIL.run.executedAt}
            version={RELEASE_DETAIL.run.promptVersion}
            extra={RELEASE_DETAIL.run.usage}
          />
        }
      >
        <Stack gap="var(--space-2)">
          {portuguese ? (
            <LocalizedRunHeader
              copy={copy}
              actions={
                <Button variant="ghost" size="sm" iconStart="history" onClick={onOpenRun}>
                  {copy.openRun}
                </Button>
              }
            />
          ) : (
            <RunMetadata
              {...RELEASE_DETAIL.run}
              selected
              labels={copy.run}
              actions={
                <Button variant="ghost" size="sm" iconStart="history" onClick={onOpenRun}>
                  {copy.openRun}
                </Button>
              }
            />
          )}
          {state === 'stale' ? (
            <Banner
              tone="attention"
              title={
                portuguese
                  ? 'A evidência mudou depois desta execução'
                  : 'The evidence changed after this run'
              }
              actions={
                <Button variant="primary" size="md" iconStart="history">
                  {copy.requestAnalysis}
                </Button>
              }
            >
              {portuguese
                ? 'O CHANGELOG.md foi editado em 2026-08-15T09:12Z. As afirmações abaixo continuam válidas para o cutoff da execução e não são atualizadas silenciosamente.'
                : "CHANGELOG.md was edited at 2026-08-15T09:12Z. The claims below remain valid for the run's own cutoff and are not silently updated."}
            </Banner>
          ) : null}
          {portuguese ? (
            <Banner tone="info" glyph="languages" title="Evidência em idioma original">
              A interface está em português. As notas de versão, linhas de changelog e títulos de
              pull request continuam em inglês, como foram publicados. Traduções geradas são
              rotuladas.
            </Banner>
          ) : null}
          {RELEASE_DETAIL.groups.map((group) => (
            <ClaimGroup key={group.kind} group={group} copy={copy} showTranslation={portuguese} />
          ))}
        </Stack>
      </Panel>
    );
  };

  return (
    <Stack>
      <StateBar
        items={copy.states}
        value={state}
        onChange={setState}
        route={`/${portuguese ? 'pt-br' : 'en'}/projects/temporal/releases${state === 'large' ? `?page=${page}` : ''}`}
        note={portuguese ? 'Rota localizada; a API permanece em inglês.' : undefined}
      />

      <Cols template="minmax(0,440px) minmax(0,1fr)">
        <Panel
          title={copy.timeline}
          meta={`cutoff ${CUTOFF} · release-claims v4`}
          actions={
            <Button variant="secondary" size="sm" iconStart="download">
              {copy.exportAction}
            </Button>
          }
          footer={
            state === 'duplicate' ? (
              <Mono tone="quiet">two records share tag v1.24.4; neither is discarded</Mono>
            ) : (
              <Provenance window="365d" cutoff={CUTOFF} version="releases v2" />
            )
          }
        >
          <Table
            caption={`${copy.timeline} — cutoff ${CUTOFF}`}
            columns={columns}
            rows={rows}
            getRowKey={(row) => `${row.tag}${row.date}${row.duplicate ? '-duplicate' : ''}`}
            density="compact"
            footer={
              state === 'large' ? (
                <Pagination
                  page={page}
                  pageSize={12}
                  total={214}
                  hasMore={page < 18}
                  onPrev={() => setPage(Math.max(1, page - 1))}
                  onNext={() => setPage(page + 1)}
                  label={copy.releasesLabel}
                />
              ) : null
            }
          />
        </Panel>
        {detailPanel()}
      </Cols>
    </Stack>
  );
}
