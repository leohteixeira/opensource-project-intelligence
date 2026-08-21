import { useState } from 'react';

import {
  Banner,
  Button,
  DefinitionList,
  EmptyState,
  Icon,
  JobProgress,
  Panel,
  RadioGroup,
  RunMetadata,
  StatusBadge,
  Table,
  TextArea,
  type TabItem,
  type TableColumn,
} from '../../design-system';
import { Cols, Mono, Provenance, Stack, StateBar } from './kit';
import { RUNS, type ProviderStatus, type RunDiff } from './fixtures';

const STATES: readonly TabItem[] = [
  { value: 'versions', label: 'Multiple versions' },
  { value: 'compare', label: 'Side-by-side' },
  { value: 'older', label: 'Older version selected' },
  { value: 'running', label: 'Queued and running' },
  { value: 'none', label: 'No successful version' },
  { value: 'invalid', label: 'Invalid structured output' },
  { value: 'degraded', label: 'Provider degradation' },
  { value: 'aggregate', label: 'Admin aggregate' },
];

const DIFF_COLUMNS: readonly TableColumn<RunDiff>[] = [
  { key: 'field', header: 'Field', wrap: true },
  {
    key: 'a',
    header: 'v1 · release-claims v3',
    mono: true,
    render: (row) => (
      <span style={{ color: row.changed ? 'var(--text-primary)' : 'var(--text-secondary)' }}>
        {row.a}
      </span>
    ),
  },
  {
    key: 'b',
    header: 'v2 · release-claims v4',
    mono: true,
    render: (row) => (
      <span
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 6,
          color: row.changed ? 'var(--text-primary)' : 'var(--text-secondary)',
          fontWeight: row.changed ? 'var(--weight-medium)' : 'inherit',
        }}
      >
        {row.changed ? <Icon name="circle-dot" size={12} color="var(--forecast-fg)" /> : null}
        {row.b}
      </span>
    ),
  },
];

const PROVIDER_COLUMNS: readonly TableColumn<ProviderStatus>[] = [
  { key: 'provider', header: 'Provider', mono: true },
  { key: 'capability', header: 'Used for', wrap: true },
  {
    key: 'state',
    header: 'State',
    render: (row) => <StatusBadge status={row.state} size="sm" label={row.health} />,
  },
  { key: 'quota', header: 'Quota', mono: true },
  {
    key: 'note',
    header: 'Effect on the product',
    wrap: true,
    render: (row) => row.note ?? <span style={{ color: 'var(--text-tertiary)' }}>None</span>,
  },
];

export function AiRunsScreen({ onBack }: { readonly onBack: () => void }) {
  const [state, setState] = useState('versions');
  const [selected, setSelected] = useState('732684515208');
  const [sent, setSent] = useState(false);

  const current = RUNS.versions.find((version) => version.runId === selected);

  if (state === 'none') {
    return (
      <Stack>
        <StateBar
          items={STATES}
          value={state}
          onChange={setState}
          route="/en/projects/temporal/runs/release-claims"
        />
        <Panel
          title={RUNS.subject}
          eyebrow="AI run governance"
          meta="1 attempt · 0 successful versions"
        >
          <Stack gap="var(--space-15)">
            <EmptyState
              glyph="circle-alert"
              title="No version of this analysis has succeeded"
              action={
                <Button variant="primary" iconStart="history">
                  Request a rerun
                </Button>
              }
            >
              The only attempt failed at 2026-05-28T22:41:09Z. Nothing generated is shown for this
              release — not a partial answer, not a lower-confidence answer. The collected release
              notes remain readable as raw evidence.
            </EmptyState>
            <RunMetadata {...RUNS.failedRun} />
          </Stack>
        </Panel>
      </Stack>
    );
  }

  if (state === 'aggregate') {
    return (
      <Stack>
        <StateBar
          items={STATES}
          value={state}
          onChange={setState}
          route="/en/admin/operations/ai-usage"
          note="Admins see aggregates across the deployment. Analysts see the runs attached to their projects and never these totals."
        />
        <Panel
          title="AI usage"
          eyebrow="Administration"
          meta="aggregate only · no prompt or output content is retained here"
          footer={
            <Mono tone="quiet">
              figures are redacted aggregates; per-run cost is not attributed to individual members
            </Mono>
          }
        >
          <Table
            caption="Aggregate AI usage by period"
            density="compact"
            columns={[
              { key: 'period', header: 'Period' },
              { key: 'runs', header: 'Runs', numeric: true, mono: true },
              { key: 'tokens', header: 'Tokens', numeric: true, mono: true },
              { key: 'failures', header: 'Failed', numeric: true, mono: true },
              { key: 'coalesced', header: 'Coalesced', numeric: true, mono: true },
            ]}
            rows={RUNS.aggregate}
            getRowKey={(row) => row.period}
          />
        </Panel>
        <Panel
          title="Provider capability"
          meta="redacted: no key, endpoint or secret is displayed on this surface"
        >
          <Table
            caption="Provider capability and health"
            density="compact"
            columns={PROVIDER_COLUMNS}
            rows={RUNS.providers}
            getRowKey={(row) => row.provider}
          />
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
        route={`/en/projects/temporal/runs/release-claims?selected=${selected}`}
      />

      {state === 'degraded' ? (
        <Banner
          tone="attention"
          title="Two providers are degraded; deterministic results are unaffected"
          actions={
            <Button variant="secondary" size="md" onClick={() => setState('aggregate')}>
              Open provider status
            </Button>
          }
        >
          Translation is disabled by an Admin and documentation retrieval exhausted its quota at
          14:02Z. Metrics, health, recommendations and alerts are computed without a model and
          continue normally. Generated text is withheld rather than degraded.
        </Banner>
      ) : null}

      {state === 'invalid' ? (
        <Banner
          tone="critical"
          title="The provider returned output that failed schema validation"
          actions={
            <Button variant="primary" size="md" iconStart="history">
              Request a rerun
            </Button>
          }
        >
          Run 732684515101 produced a claim object missing its required evidence array. The run is
          recorded as failed and nothing from it is shown. Invalid output is never partially
          rendered.
        </Banner>
      ) : null}

      {state === 'older' ? (
        <Banner
          tone="info"
          title="An older version is the one being presented"
          actions={
            <Button variant="secondary" size="md" onClick={() => setSelected('732684515221')}>
              Select v2
            </Button>
          }
        >
          Version v1 (release-claims v3) is selected for the workspace even though v2 exists.
          Selecting a version never edits or deletes another: both remain readable with their own
          metadata.
        </Banner>
      ) : null}

      <Cols template="minmax(0,1fr) 340px">
        <Stack>
          <Panel
            title={RUNS.subject}
            eyebrow="Immutable runs"
            meta="a run is never edited; a rerun creates a new version"
            actions={
              <Button variant="secondary" size="sm" iconStart="history">
                Request rerun
              </Button>
            }
            footer={
              <Provenance
                cutoff="2026-08-09T18:44:12Z"
                version="release-claims v4"
                extra="reruns for the same subject and prompt version are coalesced"
              />
            }
          >
            <Stack gap="var(--space-15)">
              {state === 'running' ? (
                <JobProgress
                  id="732684515240"
                  kind="analysis · release claims"
                  state="running"
                  completed={2}
                  total={6}
                  unit="releases"
                  startedAt="2026-08-20T14:30:00Z"
                  updatedAt="2026-08-20T14:34:40Z"
                  checkpoint="release:v1.24.2"
                  transport="stream"
                  coalesced={3}
                  actions={
                    <Button variant="secondary" size="sm">
                      Cancel
                    </Button>
                  }
                />
              ) : null}
              {RUNS.versions.map((version) => (
                <RunMetadata
                  key={version.runId}
                  {...version}
                  selected={version.runId === selected}
                  stale={
                    state === 'invalid' && version.runId === '732684515221'
                      ? '2026-08-15T09:12:00Z'
                      : undefined
                  }
                  actions={
                    version.runId === selected ? (
                      <Button variant="ghost" size="sm" iconStart="check" disabled>
                        Selected
                      </Button>
                    ) : (
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() => setSelected(version.runId)}
                      >
                        Select this version
                      </Button>
                    )
                  }
                />
              ))}
              {state === 'invalid' ? <RunMetadata {...RUNS.failedRun} /> : null}
            </Stack>
          </Panel>

          <Panel
            title="Version comparison"
            eyebrow="v1 against v2"
            meta="the same subject, two prompt versions; both outputs are retained in full"
            actions={
              <Button variant="secondary" size="sm" iconStart="table-2">
                Open full outputs
              </Button>
            }
          >
            <Stack gap="var(--space-15)">
              <Table
                caption="Differences between run v1 and run v2"
                density="compact"
                columns={DIFF_COLUMNS}
                rows={RUNS.diff}
                getRowKey={(row) => row.field}
              />
              <Banner tone="info" title="A newer version is not automatically better">
                v2 extracted two more claims and cited all of them, while v1 left one claim uncited.
                That is a reason to prefer v2 — but the choice is a member&apos;s, is attributed,
                and is reversible.
              </Banner>
            </Stack>
          </Panel>

          <Panel
            title="Feedback"
            meta="feedback is attached to one run version and never edits its output"
          >
            {sent ? (
              <Banner
                tone="positive"
                title="Feedback recorded for run 732684515221"
                actions={
                  <Button variant="ghost" size="md" onClick={() => setSent(false)}>
                    Submit different feedback
                  </Button>
                }
              >
                Recorded at 2026-08-20T14:36:02Z by Ana Silva. Submitting again replaces your own
                previous feedback for this version rather than adding a second entry.
              </Banner>
            ) : (
              <Stack gap="var(--space-15)">
                <RadioGroup
                  name="feedback"
                  legend="Was the extracted claim set faithful to the evidence?"
                  orientation="horizontal"
                  options={[
                    { value: 'yes', label: 'Faithful' },
                    { value: 'partial', label: 'Partly' },
                    { value: 'no', label: 'Not faithful' },
                  ]}
                />
                <TextArea
                  id="feedback-note"
                  label="What was wrong"
                  optional
                  rows={3}
                  maxLength={600}
                  counter
                  hint="Name the claim and the evidence it misreads. Free text is stored with your name and the run identifier."
                />
                <div>
                  <Button variant="primary" onClick={() => setSent(true)}>
                    Submit feedback
                  </Button>
                </div>
              </Stack>
            )}
          </Panel>
        </Stack>

        <Stack>
          <Panel title="Selected version" meta="what the workspace currently reads">
            <DefinitionList
              dense
              items={[
                { label: 'Run', value: selected, mono: true },
                { label: 'Prompt version', value: current?.promptVersion ?? '—', mono: true },
                { label: 'Model', value: current?.model ?? '—', mono: true },
                { label: 'Executed', value: current?.executedAt ?? '—', mono: true },
                { label: 'Claims shown', value: current ? String(current.claims) : '—' },
                { label: 'Citations', value: current ? String(current.cites) : '—' },
                { label: 'Usage', value: current?.usage ?? '—', mono: true },
              ]}
            />
          </Panel>
          <Panel title="Provider status" meta="redacted; no key or endpoint is shown">
            <Stack gap="var(--space-1)">
              {RUNS.providers.map((provider) => (
                <div
                  key={provider.provider}
                  style={{
                    display: 'grid',
                    gap: 4,
                    paddingBottom: 'var(--space-1)',
                    borderBottom: 'var(--border-hairline)',
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      gap: 'var(--space-1)',
                      alignItems: 'center',
                    }}
                  >
                    <Mono>{provider.provider}</Mono>
                    <StatusBadge status={provider.state} size="sm" label={provider.health} />
                  </div>
                  <span style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
                    {provider.capability}
                  </span>
                  {provider.note ? (
                    <span style={{ font: 'var(--type-caption)', color: 'var(--attention-fg)' }}>
                      {provider.note}
                    </span>
                  ) : null}
                </div>
              ))}
            </Stack>
          </Panel>
          <Panel title="Rules that hold on this surface">
            <ul
              style={{
                margin: 0,
                paddingLeft: '1.1em',
                display: 'grid',
                gap: 'var(--space-075)',
                font: 'var(--type-caption)',
                color: 'var(--text-secondary)',
                listStyle: 'disc',
              }}
            >
              <li>A run is immutable. Rerunning adds a version; it never overwrites one.</li>
              <li>Generated text always shows provider, model, prompt version and cutoff.</li>
              <li>A claim that cannot cite evidence is not displayed at all.</li>
              <li>
                Failed and invalid runs are visible, so a gap never looks like an absence of
                findings.
              </li>
              <li>Analysts see runs for their projects; only Admins see deployment-wide usage.</li>
            </ul>
          </Panel>
          <div>
            <Button variant="ghost" iconStart="arrow-left" onClick={onBack}>
              Back to the project
            </Button>
          </div>
        </Stack>
      </Cols>
    </Stack>
  );
}
