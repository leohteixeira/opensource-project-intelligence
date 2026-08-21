import { useState } from 'react';

import {
  Banner,
  Button,
  DefinitionList,
  Drawer,
  EmptyState,
  EvidenceLink,
  JobProgress,
  Link,
  Pagination,
  Panel,
  StatusBadge,
  Table,
  TextField,
  type TabItem,
  type TableColumn,
} from '../../design-system';
import { AiLabel, Cols, EvidenceRow, Mono, Provenance, Stack, StateBar } from './kit';
import {
  CUTOFF,
  KNOWLEDGE,
  type CrawlDomain,
  type CrawlLimit,
  type KnowledgeResult,
  type LexicalMatch,
} from './fixtures';

const STATES: readonly TabItem[] = [
  { value: 'results', label: 'Cited result' },
  { value: 'no_results', label: 'No result' },
  { value: 'unconfigured', label: 'No documentation configured' },
  { value: 'crawl', label: 'Crawl states and limits' },
  { value: 'lexical', label: 'Retrieval unavailable' },
  { value: 'conflicting', label: 'Conflicting snapshots' },
];

interface OpenDocument {
  readonly title: string;
  readonly url: string;
  readonly snapshot?: string;
}

const DOMAIN_COLUMNS: readonly TableColumn<CrawlDomain>[] = [
  { key: 'domain', header: 'Domain', mono: true, wrap: true },
  { key: 'scope', header: 'Scope', mono: true },
  {
    key: 'state',
    header: 'State',
    render: (row) => (
      <StatusBadge
        status={row.state}
        size="sm"
        detail={
          row.state === 'stale'
            ? 'snapshot from 2026-06-02'
            : row.state === 'failed'
              ? 'robots.txt disallows /'
              : undefined
        }
      />
    ),
  },
  {
    key: 'documents',
    header: 'Documents',
    numeric: true,
    render: (row) =>
      row.documents === 0 ? (
        <StatusBadge status="unknown" size="sm" label="None fetched" />
      ) : (
        row.documents.toLocaleString('en-GB')
      ),
  },
  { key: 'depth', header: 'Depth', mono: true },
  { key: 'bytes', header: 'Collected', mono: true, numeric: true },
  { key: 'lastCrawl', header: 'Last crawl', mono: true },
];

const LIMIT_COLUMNS: readonly TableColumn<CrawlLimit>[] = [
  { key: 'rule', header: 'Limit' },
  { key: 'value', header: 'Value', mono: true },
  {
    key: 'hit',
    header: 'Outcome',
    wrap: true,
    render: (row) =>
      row.hit ? (
        <span style={{ color: 'var(--attention-fg)', font: 'var(--type-ui)' }}>{row.detail}</span>
      ) : (
        <span style={{ color: 'var(--text-tertiary)' }}>Not reached</span>
      ),
  },
];

export function KnowledgeScreen({ onOpenRun }: { readonly onOpenRun: () => void }) {
  const [state, setState] = useState('results');
  const [query, setQuery] = useState<string>(KNOWLEDGE.query);
  const [openDoc, setOpenDoc] = useState<OpenDocument | null>(null);
  const [page, setPage] = useState(1);

  const lexicalColumns: readonly TableColumn<LexicalMatch>[] = [
    {
      key: 'title',
      header: 'Document',
      wrap: true,
      render: (row) => (
        <Link href="#" size="sm" onClick={() => setOpenDoc(row)}>
          {row.title}
        </Link>
      ),
    },
    { key: 'url', header: 'Source', mono: true, wrap: true },
    { key: 'terms', header: 'Matched terms', mono: true, wrap: true },
    { key: 'score', header: 'Score', numeric: true, mono: true },
  ];

  const searchBox = (
    <div
      style={{ display: 'flex', gap: 'var(--space-1)', alignItems: 'flex-end', flexWrap: 'wrap' }}
    >
      <div style={{ flex: '1 1 320px', minWidth: 240 }}>
        <TextField
          id="knowledge-query"
          label="Question"
          type="search"
          iconStart="search"
          size="lg"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          hint="Answers are drawn only from collected documentation snapshots, never from the model's own knowledge."
        />
      </div>
      <Button variant="primary" size="lg" iconStart="search">
        Search snapshots
      </Button>
    </div>
  );

  if (state === 'unconfigured') {
    return (
      <Stack>
        <StateBar
          items={STATES}
          value={state}
          onChange={setState}
          route="/en/projects/temporal/knowledge"
        />
        <Panel title="Documentation knowledge" meta="no domain configured for this project">
          <EmptyState
            glyph="book-open"
            title="No documentation source is configured"
            action={
              <Button variant="primary" iconStart="globe">
                Add a documentation domain
              </Button>
            }
          >
            Documentation is crawled only from domains a member names explicitly. Nothing is
            discovered automatically and no domain outside the configured list is fetched. Add a
            domain and a path scope to begin collecting snapshots.
          </EmptyState>
        </Panel>
      </Stack>
    );
  }

  if (state === 'crawl') {
    return (
      <Stack>
        <StateBar
          items={STATES}
          value={state}
          onChange={setState}
          route="/en/projects/temporal/knowledge/sources"
        />
        <Panel
          title="Configured domains"
          eyebrow="Explicit list — nothing is discovered"
          meta={`4 domains · cutoff ${CUTOFF}`}
          actions={
            <Button variant="primary" size="sm" iconStart="globe">
              Add domain
            </Button>
          }
          footer={
            <Provenance cutoff={CUTOFF} version="crawler v3" extra="521 documents retained" />
          }
        >
          <Table
            caption={`Configured documentation domains at cutoff ${CUTOFF}`}
            columns={DOMAIN_COLUMNS}
            rows={KNOWLEDGE.domains}
            getRowKey={(row) => row.domain}
            density="compact"
          />
        </Panel>
        <Cols template="minmax(0,1fr) minmax(0,1fr)">
          <Panel title="Crawl in progress" meta="temporal.io/blog/** · depth 2 of 3">
            <JobProgress
              id="732684520044"
              kind="documentation_crawl"
              state="running"
              completed={61}
              unit="documents"
              startedAt="2026-08-20T14:22:01Z"
              updatedAt="2026-08-20T14:34:02Z"
              transport="polling"
              checkpoint="temporal.io:/blog/page/4"
            />
          </Panel>
          <Panel title="Refused and skipped" meta="every limit is stated, never silent">
            <Table
              caption="Crawler limits and what they refused"
              density="compact"
              columns={LIMIT_COLUMNS}
              rows={KNOWLEDGE.limits}
              getRowKey={(row) => row.rule}
            />
          </Panel>
        </Cols>
        <Banner
          tone="critical"
          title="One domain was refused entirely"
          actions={
            <Button variant="secondary" size="md">
              Open crawl log
            </Button>
          }
        >
          learn.temporal.io returned a robots.txt that disallows all paths. No document was fetched
          and none is retained. The refusal is recorded so the gap is visible rather than looking
          like an empty site.
        </Banner>
      </Stack>
    );
  }

  const results: readonly KnowledgeResult[] = KNOWLEDGE.results;

  return (
    <Stack>
      <StateBar
        items={STATES}
        value={state}
        onChange={setState}
        route={`/en/projects/temporal/knowledge?q=${encodeURIComponent(query).slice(0, 40)}${state === 'results' ? `&page=${page}` : ''}`}
      />

      <Panel
        title="Search documentation snapshots"
        eyebrow="521 documents · 4 domains"
        meta="answers cite a snapshot; a claim without a citation is not shown"
        actions={
          <Button variant="secondary" size="sm" onClick={() => setState('crawl')}>
            Manage domains
          </Button>
        }
      >
        {searchBox}
      </Panel>

      {state === 'lexical' ? (
        <Banner
          tone="attention"
          title="Semantic retrieval is unavailable"
          actions={
            <Button variant="secondary" size="md" onClick={onOpenRun}>
              Open provider status
            </Button>
          }
        >
          The embedding provider exhausted its quota at 14:02Z. Lexical search over the same
          snapshots is still running and its results are shown below, with their match terms and
          scores. No generated answer is produced while retrieval is degraded.
        </Banner>
      ) : null}

      {state === 'conflicting' ? (
        <Banner tone="attention" title="Two snapshots of the same page disagree">
          The 2026-08-13 snapshot of the workflow-task-timeout page says the retry is unlimited; the
          2026-06-02 snapshot says it is capped at 10 attempts. Both are retained and shown. The
          newer snapshot is not treated as automatically correct.
        </Banner>
      ) : null}

      {state === 'no_results' ? (
        <Panel
          title="No result"
          meta={`query "${query}" · 521 documents searched · cutoff ${CUTOFF}`}
        >
          <EmptyState
            glyph="circle-help"
            title="No snapshot matched this question"
            action={
              <Button variant="secondary" iconStart="history">
                Request a fresh crawl
              </Button>
            }
          >
            521 documents across four domains were searched and none contained a passage that
            answers this question. Nothing was generated to fill the gap. Widen the wording, or
            crawl a domain that covers this topic.
          </EmptyState>
        </Panel>
      ) : state === 'lexical' ? (
        <Panel
          title="Lexical results"
          eyebrow="Retrieval degraded"
          meta="BM25 over collected snapshots · no generated answer"
          footer={
            <Provenance
              cutoff="2026-08-13T04:00:00Z"
              version="lexical v1"
              extra="semantic retrieval unavailable"
            />
          }
        >
          <Table
            caption="Lexical matches over documentation snapshots"
            density="compact"
            columns={lexicalColumns}
            rows={KNOWLEDGE.lexical}
            getRowKey={(row) => row.title}
          />
        </Panel>
      ) : (
        <Stack>
          {results.map((result, index) => (
            <Panel
              key={result.url}
              title={result.title}
              eyebrow={result.exact ? 'Exact passage' : 'Related passage'}
              meta={`snapshot ${result.snapshot} · ${result.url}`}
              status={
                state === 'conflicting' && index === 0 ? (
                  <StatusBadge status="stale" size="sm" detail="a second snapshot disagrees" />
                ) : null
              }
              actions={
                <Button
                  variant="secondary"
                  size="sm"
                  iconStart="file-text"
                  onClick={() => setOpenDoc(result)}
                >
                  Open document
                </Button>
              }
              footer={
                <Provenance
                  cutoff={result.snapshot}
                  version="retrieval v2"
                  extra={result.exact ? 'verbatim passage' : 'paraphrase of a cited passage'}
                />
              }
            >
              <Stack gap="var(--space-15)">
                <div
                  style={{
                    display: 'flex',
                    gap: 'var(--space-1)',
                    alignItems: 'center',
                    flexWrap: 'wrap',
                  }}
                >
                  <AiLabel>AI-generated answer</AiLabel>
                  <Mono tone="quiet">anthropic · claude-sonnet-4-5 · knowledge-answer v2</Mono>
                </div>
                <p
                  style={{
                    font: 'var(--type-body)',
                    color: 'var(--text-body)',
                    margin: 0,
                    maxWidth: '68ch',
                  }}
                >
                  {result.answer}
                </p>
                {state === 'conflicting' && index === 0 ? (
                  <div
                    style={{
                      display: 'grid',
                      gap: 'var(--space-075)',
                      padding: 'var(--space-15)',
                      background: 'var(--surface-sunken)',
                      borderRadius: 'var(--radius-xs)',
                    }}
                  >
                    <Mono>
                      snapshot {KNOWLEDGE.conflicting.a.snapshot} — {KNOWLEDGE.conflicting.a.text}
                    </Mono>
                    <Mono>
                      snapshot {KNOWLEDGE.conflicting.b.snapshot} — {KNOWLEDGE.conflicting.b.text}
                    </Mono>
                  </div>
                ) : null}
                <EvidenceRow cites={result.cites} />
              </Stack>
            </Panel>
          ))}
          <Pagination
            page={page}
            pageSize={2}
            total={11}
            hasMore={page < 6}
            onPrev={() => setPage(Math.max(1, page - 1))}
            onNext={() => setPage(page + 1)}
            label="passages"
          />
        </Stack>
      )}

      <Drawer
        open={Boolean(openDoc)}
        title={openDoc?.title ?? ''}
        eyebrow="Documentation snapshot"
        width="520px"
        onClose={() => setOpenDoc(null)}
        footer={
          <>
            <Button variant="secondary" onClick={() => setOpenDoc(null)}>
              Close
            </Button>
            <Button variant="primary" iconStart="globe" href="#">
              Open the live page
            </Button>
          </>
        }
      >
        {openDoc ? (
          <Stack gap="var(--space-2)">
            <DefinitionList
              items={[
                { label: 'Source URL', value: openDoc.url, mono: true },
                {
                  label: 'Snapshot taken',
                  value: openDoc.snapshot ?? '2026-08-13T04:00Z',
                  mono: true,
                },
                { label: 'Content type', value: 'text/html', mono: true },
                { label: 'Size', value: '48 kB', mono: true },
                { label: 'Crawl depth', value: '3 of 4', mono: true },
                { label: 'Original language', value: 'English' },
              ]}
            />
            <Panel title="Cited passage" meta="verbatim from the snapshot; not paraphrased">
              <Mono>
                A workflow task timeout schedules a new workflow task rather than failing the
                workflow execution. The retry is unlimited by default and is not governed by the
                activity retry policy.
              </Mono>
            </Panel>
            <EvidenceLink
              kind="snapshot"
              title="Raw snapshot · 48 kB"
              href="#"
              source="crawler v3"
              collectedAt={openDoc.snapshot ?? '2026-08-13T04:00Z'}
            />
            <Banner tone="info" title="The live page may have changed">
              This is the stored snapshot the answer was drawn from. Opening the live page leaves
              OPI&apos;s evidence chain; differences between the two are expected and are not
              corrections.
            </Banner>
          </Stack>
        ) : null}
      </Drawer>
    </Stack>
  );
}
