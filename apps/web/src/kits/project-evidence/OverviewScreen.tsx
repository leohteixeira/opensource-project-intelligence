import { useState } from 'react';

import {
  Banner,
  Button,
  CoverageDisclosure,
  DateRangeField,
  DefinitionList,
  EmptyState,
  HealthDimensions,
  Link,
  MetricValue,
  Panel,
  Recommendation,
  StatusBadge,
  Table,
  type DefinitionItem,
  type HealthDimension,
  type TabItem,
} from '../../design-system';
import { Cols, Provenance, Stack, StateBar } from './kit';
import { CUTOFF, OVERVIEW, PROJECT, WINDOW } from './fixtures';

const STATES: readonly TabItem[] = [
  { value: 'complete', label: 'Calculable' },
  { value: 'insufficient', label: 'Overall absent' },
  { value: 'stale', label: 'Stale evaluation' },
  { value: 'partial', label: 'Partial coverage' },
  { value: 'window', label: 'Window out of coverage' },
  { value: 'archived', label: 'Archived read-only' },
  { value: 'never', label: 'No collection yet' },
];

export function OverviewScreen({
  onGoTab,
  onOpenLifecycle,
}: {
  readonly onGoTab: (tab: string) => void;
  readonly onOpenLifecycle: () => void;
}) {
  const [state, setState] = useState('complete');
  const [observationWindow, setObservationWindow] = useState('90d');

  const insufficient = state === 'insufficient' || state === 'never';
  const dimensions: readonly HealthDimension[] =
    state === 'insufficient'
      ? OVERVIEW.dimensions.map((dimension, index) =>
          index > 2
            ? {
                ...dimension,
                score: undefined,
                status: 'insufficient_data',
                note: 'needs 90d, has 34d',
              }
            : dimension,
        )
      : OVERVIEW.dimensions;

  const identity: readonly DefinitionItem[] = [
    { label: 'Identifier', value: PROJECT.id, mono: true },
    {
      label: 'Primary repository',
      value: (
        <Link href="#" external evidence size="sm">
          {PROJECT.primaryRepository}
        </Link>
      ),
    },
    { label: 'Repositories', value: `${PROJECT.repositories} · ${PROJECT.sources} sources` },
    { label: 'Declared licence', value: PROJECT.license, mono: true },
    { label: 'Languages', value: PROJECT.languages },
    {
      label: 'Registered',
      value: `${PROJECT.registeredAt} by ${PROJECT.registeredBy}`,
      mono: true,
    },
    {
      label: 'Last successful collection',
      value: state === 'never' ? 'Unknown' : PROJECT.lastCollection,
      mono: true,
    },
  ];

  const banner = () => {
    if (state === 'stale') {
      return (
        <Banner
          tone="attention"
          title="This evaluation is stale"
          actions={
            <>
              <Button variant="primary" size="md" iconStart="history">
                Request recalculation
              </Button>
              <Button variant="secondary" size="md" onClick={() => onGoTab('sources')}>
                Inspect sources
              </Button>
            </>
          }
        >
          Values below were computed at cutoff 2026-08-14T04:00:00Z against metric catalog v2.
          Catalog v3 is now active and mixed-version results are refused, so nothing on this page
          has been recomputed against v3. The displayed numbers remain valid for their own cutoff
          and version.
        </Banner>
      );
    }

    if (state === 'partial') {
      return (
        <Banner
          tone="attention"
          title="Two sources failed in the last collection"
          actions={
            <Button variant="secondary" size="md" onClick={() => onGoTab('sources')}>
              Open sources and jobs
            </Button>
          }
        >
          Discussion and advisory collection failed at 2026-08-20T09:04:10Z. Metrics that depend
          only on GitHub and npm are unaffected and are shown; metrics that depend on the failed
          sources report Insufficient data rather than a lower number.
        </Banner>
      );
    }

    if (state === 'archived') {
      return (
        <Banner
          tone="neutral"
          glyph="archive"
          title="This project is archived"
          actions={
            <Button variant="secondary" size="md" onClick={onOpenLifecycle}>
              Open lifecycle
            </Button>
          }
        >
          Collected evidence and past evaluations are retained and readable. Collection,
          recalculation and alert delivery are stopped. Restore the project to resume them.
        </Banner>
      );
    }

    if (state === 'window') {
      return (
        <Banner
          tone="attention"
          title="The requested window exceeds available coverage"
          actions={
            <Button variant="secondary" size="md" onClick={() => setObservationWindow('90d')}>
              Return to 90d
            </Button>
          }
        >
          365 days were requested and 118 days exist. Results are not extrapolated; every metric
          below reports the coverage it actually had.
        </Banner>
      );
    }

    return null;
  };

  if (state === 'never') {
    return (
      <Stack>
        <StateBar
          items={STATES}
          value={state}
          onChange={setState}
          route="/en/projects/temporal/overview"
        />
        <Panel title="Temporal" meta="registered 2025-11-04T09:12:44Z · no successful collection">
          <EmptyState
            glyph="circle-dashed"
            title="No collection has succeeded for this project yet"
            action={
              <Button variant="primary" iconStart="history">
                Request initial synchronization
              </Button>
            }
          >
            Eight sources are configured and the initial synchronization has not completed. Health,
            metrics and recommendations are Unknown — not zero — until at least one collection
            succeeds.
          </EmptyState>
        </Panel>
        <Panel title="Identity" meta="from registration; not derived from collected evidence">
          <DefinitionList items={identity} columns={2} />
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
        route={`/en/projects/temporal/overview?window=${observationWindow}`}
      />
      {banner()}

      <Cols template="minmax(0,1fr) 340px">
        <Stack>
          <Panel
            title="Health"
            eyebrow="Seven independent dimensions"
            meta={`window ${WINDOW.label} · cutoff ${state === 'stale' ? '2026-08-14T04:00:00Z' : CUTOFF} · rubric v3`}
            status={
              state === 'stale' ? (
                <StatusBadge status="stale" size="sm" detail="cutoff 2026-08-14T04:00Z" />
              ) : null
            }
            actions={
              <Button variant="secondary" size="sm" onClick={() => onGoTab('health')}>
                Metric detail
              </Button>
            }
            footer={
              <Provenance
                window={`coverage ${state === 'partial' ? '34d of 90d' : '90d of 90d'}`}
                cutoff={state === 'stale' ? '2026-08-14T04:00:00Z' : CUTOFF}
                version="rubric v3"
              />
            }
          >
            <HealthDimensions
              dimensions={dimensions}
              layout="grid"
              overall={
                insufficient
                  ? {
                      calculable: false,
                      version: 'rubric v3',
                      reason:
                        'Four of seven dimensions have less coverage than the rubric requires. An overall score is withheld rather than computed from a partial set.',
                    }
                  : {
                      calculable: true,
                      score: OVERVIEW.overall.score,
                      version: OVERVIEW.overall.version,
                    }
              }
            />
          </Panel>

          <Panel
            title="Recommendation"
            meta={`adoption policy v4 · window ${WINDOW.label} · cutoff ${CUTOFF}`}
            actions={
              <Button variant="secondary" size="sm" iconStart="scale">
                Policy version
              </Button>
            }
          >
            <Recommendation
              result={insufficient ? 'insufficient_data' : 'conditional'}
              policy="Adoption policy"
              version="v4"
              window={WINDOW.label}
              cutoff={CUTOFF}
              stale={state === 'stale' ? '2026-08-14T04:00:00Z' : undefined}
              conditions={insufficient ? [] : OVERVIEW.conditions}
              missing={
                insufficient
                  ? [
                      'release_frequency',
                      'top_three_author_share_90d',
                      'regression_issue_share',
                      'nuget_download_change',
                    ]
                  : OVERVIEW.missing
              }
              decisive={
                insufficient
                  ? []
                  : [
                      {
                        metric: 'release_frequency',
                        rule: '≥ 4 releases / 90d',
                        value: '8 releases',
                        pass: true,
                      },
                      {
                        metric: 'median_time_to_first_response',
                        rule: '≤ 24 h',
                        value: '9.4 h',
                        pass: true,
                      },
                      {
                        metric: 'top_three_author_share_90d',
                        rule: '≤ 0.60',
                        value: '0.71',
                        pass: false,
                      },
                      {
                        metric: 'open_advisories',
                        rule: '= 0 verified',
                        value: 'Unknown',
                        pass: false,
                      },
                    ]
              }
            />
          </Panel>

          <Panel
            title="Headline metrics"
            meta={`window ${WINDOW.label} · cutoff ${CUTOFF} · catalog v3`}
            actions={
              <Button
                variant="ghost"
                size="sm"
                iconEnd="arrow-right"
                onClick={() => onGoTab('health')}
              >
                All 12 metrics
              </Button>
            }
          >
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit,minmax(200px,1fr))',
                gap: 'var(--space-2)',
              }}
            >
              {OVERVIEW.headline.map((metric) => {
                const degraded = state === 'partial' && metric.name === 'active_contributors_30d';

                return (
                  <MetricValue
                    key={metric.name}
                    {...metric}
                    window={WINDOW.label}
                    status={degraded ? 'insufficient_data' : metric.status}
                    note={degraded ? 'discussion source failed; needs 90d, has 34d' : metric.note}
                    onOpen={() => onGoTab('health')}
                  />
                );
              })}
              <MetricValue
                {...OVERVIEW.zeroValue}
                window={WINDOW.label}
                note="Zero is a measured value, not missing data."
                onOpen={() => onGoTab('releases')}
              />
            </div>
          </Panel>
        </Stack>

        <Stack>
          <Panel title="Window" meta="URL-owned; a shared link reproduces this view">
            <DateRangeField
              value={observationWindow}
              from={WINDOW.from}
              to={WINDOW.to}
              cutoff={CUTOFF}
              coverage={
                state === 'partial' ? '34d of 90d' : state === 'window' ? '118d of 365d' : undefined
              }
              onChange={(next) => {
                setObservationWindow(next);
                if (next === '365d') setState('window');
              }}
            />
          </Panel>
          <Panel title="Coverage" meta={`cutoff ${CUTOFF}`}>
            <CoverageDisclosure
              {...(state === 'partial' ? OVERVIEW.partialCoverage : OVERVIEW.coverage)}
            />
          </Panel>
          <Panel title="Identity" meta="from registration and collected metadata">
            <DefinitionList items={identity} dense />
          </Panel>
          <Panel title="Next steps">
            <Table
              caption={`Suggested actions at cutoff ${CUTOFF}`}
              density="compact"
              columns={[
                { key: 'what', header: 'Action', wrap: true },
                { key: 'where', header: 'Surface', render: (row) => row.where },
              ]}
              getRowKey={(row) => row.what}
              rows={[
                {
                  what: 'Review contributor concentration',
                  where: (
                    <Link href="#" size="sm" onClick={() => onGoTab('contributors')}>
                      Contributors
                    </Link>
                  ),
                },
                {
                  what: 'Configure an advisory source',
                  where: (
                    <Link href="#" size="sm" onClick={() => onGoTab('sources')}>
                      Sources
                    </Link>
                  ),
                },
                {
                  what: 'Read the breaking changes in v1.25.0',
                  where: (
                    <Link href="#" size="sm" onClick={() => onGoTab('releases')}>
                      Releases
                    </Link>
                  ),
                },
              ]}
            />
          </Panel>
        </Stack>
      </Cols>
    </Stack>
  );
}
