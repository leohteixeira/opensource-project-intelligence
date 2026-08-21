import { useState } from 'react';

import {
  Banner,
  Button,
  CoverageDisclosure,
  DefinitionList,
  Dialog,
  EmptyState,
  Icon,
  MetricValue,
  Pagination,
  Panel,
  StatusBadge,
  Table,
  Tooltip,
  type TabItem,
  type TableColumn,
} from '../../design-system';
import { Cols, Mono, Provenance, Stack, StateBar } from './kit';
import { CONTRIBUTORS, CUTOFF, WINDOW, type Cohort, type Contributor } from './fixtures';

const STATES: readonly TabItem[] = [
  { value: 'many', label: 'Many contributors' },
  { value: 'single', label: 'One contributor' },
  { value: 'empty', label: 'No contributors' },
  { value: 'identities', label: 'Unresolved identities' },
  { value: 'conflict', label: 'Conflicting merge' },
  { value: 'retention', label: 'Retention unknown' },
];

const AUTOMATION = {
  bot: { glyph: 'circle-slash', word: 'Automation' },
  service: { glyph: 'circle-slash', word: 'Service account' },
} as const;

function KindChip({ kind }: { readonly kind: Contributor['kind'] }) {
  if (kind === 'person') return null;

  const spec = AUTOMATION[kind];

  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 4,
        padding: '1px 7px',
        borderRadius: 'var(--radius-pill)',
        background: 'var(--unknown-bg)',
        border: '1px solid var(--unknown-border)',
        color: 'var(--unknown-fg)',
        font: 'var(--type-caption)',
        whiteSpace: 'nowrap',
      }}
    >
      <Icon name={spec.glyph} size={11} />
      {spec.word}
    </span>
  );
}

export function ContributorsScreen() {
  const [state, setState] = useState('many');
  const [page, setPage] = useState(1);
  /**
   * The conflict state opens the merge dialog by default; an explicit open or close only holds
   * while the selected state is unchanged, so switching states never leaves a stale dialog open.
   */
  const [mergeChoice, setMergeChoice] = useState<{ state: string; open: boolean } | null>(null);
  const merge = mergeChoice?.state === state ? mergeChoice.open : state === 'conflict';
  const setMerge = (open: boolean) => setMergeChoice({ state, open });

  const people = state === 'single' ? CONTRIBUTORS.people.slice(0, 1) : CONTRIBUTORS.people;

  const peopleColumns: readonly TableColumn<Contributor>[] = [
    {
      key: 'handle',
      header: 'Contributor',
      wrap: true,
      render: (row) => (
        <span style={{ display: 'grid', gap: 2 }}>
          <span
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--space-075)',
              flexWrap: 'wrap',
            }}
          >
            <span style={{ fontWeight: 'var(--weight-medium)' }}>{row.name}</span>
            <KindChip kind={row.kind} />
          </span>
          <Mono tone="quiet">
            {row.handle}
            {row.identities > 1 ? ` · ${row.identities} linked identities` : ''}
          </Mono>
        </span>
      ),
    },
    { key: 'role', header: 'Role' },
    { key: 'commits', header: 'Commits', numeric: true },
    {
      key: 'share',
      header: 'Author share',
      numeric: true,
      render: (row) =>
        row.share === '—' ? (
          <Tooltip content="Automation and service accounts are excluded from concentration by concentration rubric v3.">
            <span style={{ color: 'var(--text-tertiary)' }}>Excluded</span>
          </Tooltip>
        ) : (
          <span
            style={{
              fontWeight: Number(row.share) > 0.15 ? 'var(--weight-medium)' : 'inherit',
            }}
          >
            {row.share}
          </span>
        ),
    },
    { key: 'firstSeen', header: 'First seen', mono: true },
    { key: 'lastSeen', header: 'Last seen', mono: true },
  ];

  const cohortColumns: readonly TableColumn<Cohort>[] = [
    { key: 'cohort', header: 'Cohort', mono: true },
    { key: 'joined', header: 'Joined', numeric: true },
    {
      key: 'active90d',
      header: 'Active in 90d',
      numeric: true,
      render: (row) =>
        row.active90d === null ? (
          <StatusBadge
            status="insufficient_data"
            size="sm"
            detail="cohort younger than the window"
          />
        ) : (
          row.active90d
        ),
    },
    {
      key: 'retained',
      header: 'Retained',
      numeric: true,
      render: (row) =>
        row.retained === null ? (
          <span style={{ color: 'var(--text-tertiary)' }}>Insufficient data</span>
        ) : (
          row.retained
        ),
    },
  ];

  const cohorts: readonly Cohort[] =
    state === 'retention'
      ? CONTRIBUTORS.cohorts.map((cohort) => ({
          ...cohort,
          active90d: null,
          retained: null,
          status: 'unknown',
        }))
      : CONTRIBUTORS.cohorts;

  if (state === 'empty') {
    return (
      <Stack>
        <StateBar
          items={STATES}
          value={state}
          onChange={setState}
          route="/en/projects/temporal/contributors"
        />
        <Panel
          title="Contributors"
          meta={`window ${WINDOW.label} · cutoff ${CUTOFF} · contributors v3`}
        >
          <EmptyState
            glyph="users"
            title="No contributor is attributable in this window"
            action={
              <Button variant="secondary" iconStart="history">
                Request a 365-day history
              </Button>
            }
          >
            The GitHub source collected 0 commits between {WINDOW.from} and {WINDOW.to}.
            Concentration, retention and maintainer counts are Unknown for this window rather than
            zero. A longer history request may return attributable authors.
          </EmptyState>
        </Panel>
      </Stack>
    );
  }

  return (
    <Stack>
      <StateBar
        items={STATES}
        value={state}
        onChange={setState}
        route={`/en/projects/temporal/contributors?window=90d&page=${page}`}
      />

      {state === 'many' ? (
        <Banner
          tone="attention"
          title="Top-three contributor concentration is above the policy threshold"
          actions={
            <Button variant="secondary" size="md" iconStart="scale">
              Open policy rule
            </Button>
          }
        >
          Three authors account for {CONTRIBUTORS.concentration.topThree} of commits in this window;
          adoption policy v4 requires {CONTRIBUTORS.concentration.threshold} or lower. This is one
          failing condition, not a verdict on the project.
        </Banner>
      ) : null}

      {state === 'single' ? (
        <Banner tone="attention" title="One attributable contributor in this window">
          A single author accounts for 1.00 of commits. Concentration is reported as measured;
          retention cohorts need at least two cohorts and report Insufficient data.
        </Banner>
      ) : null}

      <Cols template="minmax(0,1fr) 320px">
        <Stack>
          <Panel
            title="Contributors"
            eyebrow={
              state === 'single' ? '1 author' : `${CONTRIBUTORS.concentration.authors} authors`
            }
            meta={`window ${WINDOW.label} · cutoff ${CUTOFF} · contributors v3`}
            actions={
              <Button variant="secondary" size="sm" iconStart="download">
                Export
              </Button>
            }
            footer={
              <Provenance
                window="coverage 90d of 90d"
                cutoff={CUTOFF}
                version="contributors v3"
                extra="handles and display names are never translated"
              />
            }
          >
            <Table
              caption={`Contributors between ${WINDOW.from} and ${WINDOW.to}`}
              columns={peopleColumns}
              rows={people}
              getRowKey={(row) => row.handle}
              density="compact"
              footer={
                state === 'many' ? (
                  <Pagination
                    page={page}
                    pageSize={6}
                    total={341}
                    hasMore={page < 57}
                    onPrev={() => setPage(Math.max(1, page - 1))}
                    onNext={() => setPage(page + 1)}
                    label="contributors"
                  />
                ) : null
              }
            />
          </Panel>

          <Panel
            title="Retention cohorts"
            meta="cohort membership by first attributable commit · retention v2"
            status={
              state === 'retention' ? (
                <StatusBadge status="unknown" size="sm" detail="identity resolution incomplete" />
              ) : null
            }
            footer={
              state === 'retention' ? (
                <Mono tone="quiet">
                  retention requires resolved identities; 3 of 341 authors are unresolved
                </Mono>
              ) : (
                <Provenance window="90d activity test" cutoff={CUTOFF} version="retention v2" />
              )
            }
          >
            {state === 'retention' ? (
              <Stack gap="var(--space-15)">
                <Banner
                  tone="neutral"
                  glyph="circle-help"
                  title="Retention is Unknown for every cohort"
                >
                  Retention counts a person, not an address. Three unresolved identities could
                  belong to existing contributors, so a retention ratio would be a guess. Resolve
                  them to compute it.
                </Banner>
                <Table
                  caption="Retention cohorts, unresolved"
                  columns={cohortColumns}
                  rows={cohorts}
                  getRowKey={(row) => row.cohort}
                  density="compact"
                />
              </Stack>
            ) : (
              <Table
                caption={`Retention cohorts at cutoff ${CUTOFF}`}
                columns={cohortColumns}
                rows={cohorts}
                getRowKey={(row) => row.cohort}
                density="compact"
              />
            )}
          </Panel>

          {state === 'identities' || state === 'conflict' ? (
            <Panel
              title="Identity association review"
              eyebrow="Analyst correction"
              meta="associations are proposed by evidence and confirmed by a member; every correction is attributed"
              actions={
                <Button variant="secondary" size="sm">
                  Review guidance
                </Button>
              }
            >
              <Stack gap="var(--space-15)">
                {CONTRIBUTORS.unresolved.map((identity) => (
                  <div
                    key={identity.id}
                    style={{
                      border: 'var(--border-default)',
                      borderRadius: 'var(--radius-xs)',
                      padding: 'var(--space-15)',
                      display: 'grid',
                      gap: 'var(--space-1)',
                    }}
                  >
                    <div
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        gap: 'var(--space-2)',
                        flexWrap: 'wrap',
                        alignItems: 'flex-start',
                      }}
                    >
                      <span style={{ display: 'grid', gap: 2 }}>
                        <Mono>{identity.display}</Mono>
                        <span
                          style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}
                        >
                          {identity.basis}
                        </span>
                      </span>
                      {identity.confidence ? (
                        <StatusBadge
                          status={Number(identity.confidence) > 0.8 ? 'available' : 'unknown'}
                          size="sm"
                          label={
                            Number(identity.confidence) > 0.8 ? 'Automatic link' : 'Uncertain link'
                          }
                          detail={`confidence ${identity.confidence}`}
                        />
                      ) : (
                        <StatusBadge
                          status="unknown"
                          size="sm"
                          label="No candidate"
                          detail="confidence unavailable"
                        />
                      )}
                    </div>
                    <div style={{ display: 'flex', gap: 'var(--space-1)', flexWrap: 'wrap' }}>
                      {identity.suggestion ? (
                        <Button variant="secondary" size="sm" iconStart="check">
                          Link to {identity.suggestion}
                        </Button>
                      ) : null}
                      <Button variant="ghost" size="sm">
                        Keep separate
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => setMerge(true)}>
                        Choose another contributor
                      </Button>
                    </div>
                  </div>
                ))}
              </Stack>
            </Panel>
          ) : null}
        </Stack>

        <Stack>
          <Panel
            title="Concentration"
            meta={`window ${WINDOW.label} · concentration ${CONTRIBUTORS.concentration.version}`}
          >
            <Stack gap="var(--space-2)">
              <MetricValue
                name="top_one_author_share_90d"
                label="Top author share"
                value={state === 'single' ? '1.00' : CONTRIBUTORS.concentration.topOne}
                unit="ratio"
                status="available"
                version="v3"
                window="90d"
                size="md"
              />
              <MetricValue
                name="top_three_author_share_90d"
                label="Top three author share"
                value={state === 'single' ? '1.00' : CONTRIBUTORS.concentration.topThree}
                unit="ratio"
                status="available"
                version="v3"
                window="90d"
                size="md"
                delta={state === 'many' ? '+0.09' : undefined}
                deltaDirection="up"
                note={`policy threshold ${CONTRIBUTORS.concentration.threshold}`}
              />
              <MetricValue
                name="maintainer_count"
                label="Maintainers"
                value={state === 'single' ? '1' : '2'}
                unit="people"
                status="available"
                version="v2"
                window="90d"
                size="md"
              />
            </Stack>
          </Panel>
          <Panel title="Identity coverage" meta="resolved people, not addresses">
            <CoverageDisclosure
              requested={`${CONTRIBUTORS.concentration.authors} authors`}
              actual="338 resolved"
              ratio={0.99}
              sources={[
                { name: 'commit e-mail', value: '331' },
                { name: 'linked account', value: '7' },
              ]}
              missing={['3 unresolved addresses']}
              cutoff={CUTOFF}
            />
          </Panel>
          <Panel title="Excluded from concentration" meta="automation is identified, not hidden">
            <DefinitionList
              dense
              items={[
                { label: 'dependabot[bot]', value: 'Automation · 1,204 commits', mono: true },
                { label: 'temporal-ci', value: 'Service account · 806 commits', mono: true },
              ]}
            />
          </Panel>
        </Stack>
      </Cols>

      <Dialog
        open={merge}
        title="Two members corrected this identity"
        tone="danger"
        size="md"
        onClose={() => setMerge(false)}
        footer={
          <>
            <Button variant="secondary" onClick={() => setMerge(false)}>
              Cancel
            </Button>
            <Button variant="primary" onClick={() => setMerge(false)}>
              Keep Ana Silva&apos;s correction
            </Button>
          </>
        }
      >
        <Stack gap="var(--space-15)">
          <p style={{ font: 'var(--type-body)', color: 'var(--text-body)', margin: 0 }}>
            The identity <Mono>{CONTRIBUTORS.conflict.identity}</Mono> was linked to two different
            contributors within two minutes. Neither correction was applied. Choose one; the other
            is recorded as rejected with its actor and timestamp.
          </p>
          <DefinitionList
            items={[
              {
                label: `Ana Silva · ${CONTRIBUTORS.conflict.a.at}`,
                value: <Mono>{CONTRIBUTORS.conflict.a.handle}</Mono>,
              },
              {
                label: `Rafael Costa · ${CONTRIBUTORS.conflict.b.at}`,
                value: <Mono>{CONTRIBUTORS.conflict.b.handle}</Mono>,
              },
            ]}
          />
          <Banner tone="info" title="Both attempts stay in the audit record">
            Identity corrections are attributable and reversible. Retention and concentration are
            recomputed only after one correction is accepted.
          </Banner>
        </Stack>
      </Dialog>
    </Stack>
  );
}
