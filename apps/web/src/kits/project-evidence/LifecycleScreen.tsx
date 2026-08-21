import { useState } from 'react';

import {
  Banner,
  Button,
  Checkbox,
  DefinitionList,
  Dialog,
  EmptyState,
  Icon,
  JobProgress,
  Menu,
  Panel,
  RadioGroup,
  StatusBadge,
  Table,
  TextField,
  type StatusKey,
  type TabItem,
  type TableColumn,
} from '../../design-system';
import { Cols, Mono, Stack, StateBar } from './kit';
import { LIFECYCLE, PROJECT, type LifecycleEvent } from './fixtures';

const STATES: readonly TabItem[] = [
  { value: 'active', label: 'Active' },
  { value: 'pause', label: 'Pause confirmation' },
  { value: 'paused', label: 'Paused' },
  { value: 'archive', label: 'Archive confirmation' },
  { value: 'archived', label: 'Archived' },
  { value: 'restore', label: 'Restore' },
  { value: 'conflict', label: 'Conflicting transition' },
  { value: 'delete', label: 'Typed deletion' },
  { value: 'purging', label: 'Purge running' },
  { value: 'tombstone', label: 'Tombstone' },
  { value: 'forbidden', label: 'Forbidden for Analyst' },
];

interface Effect {
  readonly effect: string;
  readonly outcome: string;
}

const EFFECTS: Record<'pause' | 'archive' | 'delete', readonly Effect[]> = {
  pause: [
    { effect: 'Scheduled collection', outcome: 'Stopped. Three scheduled runs are cancelled.' },
    { effect: 'Collected evidence', outcome: 'Retained and readable.' },
    {
      effect: 'Metric values and health',
      outcome: 'Retained at their existing cutoffs. Not recalculated.',
    },
    { effect: 'Alert delivery', outcome: 'Stopped. Open alerts stay open.' },
    {
      effect: 'Radar placement',
      outcome: 'Retained, marked as evaluated at an older cutoff.',
    },
    { effect: 'Reversible', outcome: 'Yes. Resuming continues from the last checkpoint.' },
  ],
  archive: [
    { effect: 'Scheduled collection', outcome: 'Stopped and removed from the schedule.' },
    { effect: 'Collected evidence', outcome: 'Retained and readable, read-only.' },
    {
      effect: 'Sources and associations',
      outcome: 'Read-only. No source can be added, removed or corrected.',
    },
    {
      effect: 'Comparison and radar',
      outcome: 'The project is excluded from new comparisons; existing saved comparisons keep it.',
    },
    { effect: 'Alert rules', outcome: 'Disabled. Historical alerts are retained.' },
    {
      effect: 'Reversible',
      outcome: 'Yes. Restoring requires choosing whether to backfill the archived gap.',
    },
  ],
  delete: [
    { effect: 'Repositories and sources', outcome: 'Permanently removed.' },
    { effect: 'Collected evidence and snapshots', outcome: 'Permanently removed.' },
    { effect: 'Metric values, health, recommendations', outcome: 'Permanently removed.' },
    { effect: 'AI runs and their outputs', outcome: 'Permanently removed.' },
    {
      effect: 'Alerts and exports',
      outcome: 'Permanently removed. Existing artifact links stop working.',
    },
    {
      effect: 'Audit record',
      outcome: 'Retained: identifier, slug, deletion timestamp and actor.',
    },
    { effect: 'Reversible', outcome: 'No.' },
  ],
};

const EFFECT_COLUMNS: readonly TableColumn<Effect>[] = [
  { key: 'effect', header: 'Affected' },
  { key: 'outcome', header: 'Effect', wrap: true },
];

const HISTORY_COLUMNS: readonly TableColumn<LifecycleEvent>[] = [
  { key: 'at', header: 'When', mono: true },
  {
    key: 'transition',
    header: 'Transition',
    render: (row) => (
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
        <Mono>{row.from ?? '—'}</Mono>
        <Icon name="arrow-right" size={13} />
        <Mono>{row.to}</Mono>
      </span>
    ),
  },
  { key: 'actor', header: 'Actor' },
  { key: 'note', header: 'Recorded reason', wrap: true },
];

function currentState(state: string): string {
  if (state === 'paused') return 'paused';
  if (state === 'pause') return 'active';
  if (state === 'archive') return 'active';
  if (state === 'archived' || state === 'restore') return 'archived';
  if (state === 'tombstone') return 'deleted';

  return 'active';
}

export function LifecycleScreen({ onBack }: { readonly onBack: () => void }) {
  const [state, setState] = useState('active');
  const [typed, setTyped] = useState('');
  const [restoreMode, setRestoreMode] = useState('resume');

  const current = currentState(state);
  const dialog =
    state === 'pause'
      ? 'pause'
      : state === 'archive'
        ? 'archive'
        : state === 'delete'
          ? 'delete'
          : state === 'conflict'
            ? 'conflict'
            : state === 'restore'
              ? 'restore'
              : null;

  if (state === 'forbidden') {
    return (
      <Stack>
        <StateBar
          items={STATES}
          value={state}
          onChange={setState}
          route="/en/projects/temporal/lifecycle"
          note="Rendered as an Analyst would see it: the transition control is absent, not disabled."
        />
        <Panel
          title="Lifecycle"
          meta="current state: active · changed 2026-03-02T08:15:19Z by Rafael Costa"
        >
          <Stack gap="var(--space-15)">
            <Banner
              tone="neutral"
              glyph="circle-slash"
              title="Lifecycle transitions are an Admin action"
            >
              Pausing, archiving, restoring and deleting a project require the Admin role. The
              control is not shown here because it is not available to you, rather than shown and
              refused on click.
            </Banner>
            <Table
              caption="Lifecycle history"
              columns={HISTORY_COLUMNS}
              rows={LIFECYCLE.history}
              getRowKey={(row) => row.at}
              density="compact"
            />
          </Stack>
        </Panel>
      </Stack>
    );
  }

  if (state === 'tombstone') {
    return (
      <Stack>
        <StateBar items={STATES} value={state} onChange={setState} route="/en/projects/temporal" />
        <Banner
          tone="critical"
          title="This project was permanently deleted"
          actions={
            <Button variant="secondary" size="md" onClick={onBack}>
              Return to projects
            </Button>
          }
        >
          Deleted at {LIFECYCLE.tombstone.deletedAt} by {LIFECYCLE.tombstone.actor}. A second
          deletion request for the same identifier returns this same record rather than an error.
        </Banner>
        <Panel
          title="Tombstone"
          eyebrow="Audit record"
          meta="the only data retained for a deleted project"
        >
          <Stack gap="var(--space-15)">
            <DefinitionList
              items={[
                { label: 'Identifier', value: LIFECYCLE.tombstone.id, mono: true },
                { label: 'Slug', value: LIFECYCLE.tombstone.slug, mono: true },
                { label: 'Deleted at', value: LIFECYCLE.tombstone.deletedAt, mono: true },
                { label: 'Deleted by', value: LIFECYCLE.tombstone.actor },
                { label: 'Retained', value: LIFECYCLE.tombstone.retained },
                { label: 'Removed', value: LIFECYCLE.tombstone.removed },
              ]}
            />
            <EmptyState
              compact
              glyph="archive"
              title="No evidence, metric or run remains for this project"
            >
              Every collected record was purged. Export artifacts built before the deletion return
              410 Gone. The slug is not reusable, so a later project cannot silently inherit this
              project&apos;s history.
            </EmptyState>
          </Stack>
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
        route="/en/projects/temporal/lifecycle"
      />

      {state === 'paused' ? (
        <Banner
          tone="attention"
          glyph="pause"
          title="Collection is paused"
          actions={
            <Button variant="primary" size="md" iconStart="history">
              Resume collection
            </Button>
          }
        >
          Paused at 2026-02-18T13:40:02Z by Rafael Costa. Evidence and past evaluations are readable
          and unchanged. Resuming continues from the stored checkpoint rather than starting a new
          full sync.
        </Banner>
      ) : null}

      {state === 'archived' ? (
        <Banner
          tone="neutral"
          glyph="archive"
          title="This project is archived"
          actions={
            <Button variant="primary" size="md" onClick={() => setState('restore')}>
              Restore
            </Button>
          }
        >
          Archived at 2026-08-20T15:02:11Z. All surfaces are readable and read-only. The project is
          excluded from new comparisons and radar evaluation, and its alert rules are disabled.
        </Banner>
      ) : null}

      {state === 'purging' ? (
        <Panel title="Deletion in progress" meta="job 732684519104 · started 2026-08-20T15:02:11Z">
          <Stack gap="var(--space-15)">
            <JobProgress
              id="732684519104"
              kind="purge"
              state="running"
              completed={3}
              total={7}
              unit="record classes"
              startedAt="2026-08-20T15:02:11Z"
              updatedAt="2026-08-20T15:03:02Z"
              transport="stream"
              checkpoint="purge:evidence:documents"
            />
            <Banner tone="critical" title="This cannot be cancelled">
              The purge is irreversible and runs to completion. The project is already unreachable
              from every other surface. A tombstone appears when the last record class is removed.
            </Banner>
          </Stack>
        </Panel>
      ) : null}

      <Cols template="minmax(0,1fr) 340px">
        <Stack>
          <Panel
            title="Lifecycle"
            eyebrow={PROJECT.name}
            meta={`current state: ${current} · every transition is attributed and recorded`}
            status={
              <StatusBadge
                status={(current === 'deleted' ? 'failed' : current) as StatusKey}
                size="sm"
              />
            }
            actions={
              <Menu
                align="end"
                triggerLabel="Change lifecycle state"
                trigger={
                  <Button variant="secondary" size="md" iconEnd="chevron-down">
                    Change state
                  </Button>
                }
                items={[
                  {
                    label: 'Pause collection',
                    icon: 'pause',
                    onSelect: () => setState('pause'),
                    disabled: current !== 'active',
                  },
                  {
                    label: 'Resume collection',
                    icon: 'history',
                    onSelect: () => setState('active'),
                    disabled: current !== 'paused',
                  },
                  {
                    label: 'Archive project',
                    icon: 'archive',
                    onSelect: () => setState('archive'),
                    disabled: current === 'archived',
                  },
                  {
                    label: 'Restore project',
                    icon: 'history',
                    onSelect: () => setState('restore'),
                    disabled: current !== 'archived',
                  },
                  { separator: true },
                  {
                    label: 'Delete permanently',
                    icon: 'octagon-x',
                    danger: true,
                    onSelect: () => setState('delete'),
                  },
                ]}
              />
            }
            footer={
              <Mono tone="quiet">
                transitions are Admin-only; the control is absent for other roles
              </Mono>
            }
          >
            <Table
              caption="Lifecycle history for this project"
              columns={HISTORY_COLUMNS}
              rows={LIFECYCLE.history}
              getRowKey={(row) => row.at}
              density="compact"
            />
          </Panel>

          <Panel
            title="What each transition does"
            meta="the same summary the confirmation dialog shows"
          >
            <Table
              caption="Effects of each lifecycle transition"
              density="compact"
              columns={[
                { key: 'effect', header: 'Affected', wrap: true },
                { key: 'pause', header: 'Pause', wrap: true },
                { key: 'archive', header: 'Archive', wrap: true },
                {
                  key: 'delete',
                  header: 'Delete',
                  wrap: true,
                  render: (row) => (
                    <span style={{ color: 'var(--critical-fg)' }}>{row.delete}</span>
                  ),
                },
              ]}
              rows={[
                {
                  effect: 'Scheduled collection',
                  pause: 'Stopped',
                  archive: 'Removed',
                  delete: 'Removed',
                },
                {
                  effect: 'Collected evidence',
                  pause: 'Retained',
                  archive: 'Read-only',
                  delete: 'Destroyed',
                },
                {
                  effect: 'Metrics and health',
                  pause: 'Frozen at cutoff',
                  archive: 'Read-only',
                  delete: 'Destroyed',
                },
                {
                  effect: 'Alert delivery',
                  pause: 'Stopped',
                  archive: 'Rules disabled',
                  delete: 'Destroyed',
                },
                { effect: 'Reversible', pause: 'Yes', archive: 'Yes', delete: 'No' },
              ]}
              getRowKey={(row) => row.effect}
            />
          </Panel>
        </Stack>

        <Stack>
          <Panel
            title="Scheduled work"
            meta={current === 'active' ? '3 runs scheduled' : 'cancelled by the current state'}
          >
            {current === 'active' ? (
              <Table
                caption="Scheduled jobs for this project"
                density="compact"
                columns={[
                  { key: 'kind', header: 'Job', mono: true },
                  { key: 'nextRun', header: 'Next run', mono: true },
                  { key: 'scope', header: 'Scope' },
                ]}
                rows={LIFECYCLE.scheduled}
                getRowKey={(row) => row.id}
              />
            ) : (
              <EmptyState compact glyph="pause" title="No work is scheduled">
                Three scheduled runs were cancelled when the project left the active state. They are
                rescheduled on resume or restore, not replayed.
              </EmptyState>
            )}
          </Panel>
          <Panel title="Identity" meta="unchanged by any lifecycle transition except deletion">
            <DefinitionList
              dense
              items={[
                { label: 'Identifier', value: PROJECT.id, mono: true },
                { label: 'Slug', value: PROJECT.slug, mono: true },
                { label: 'Registered', value: PROJECT.registeredAt, mono: true },
                { label: 'Primary repository', value: PROJECT.primaryRepository, mono: true },
              ]}
            />
          </Panel>
          <div>
            <Button variant="ghost" iconStart="arrow-left" onClick={onBack}>
              Back to the project
            </Button>
          </div>
        </Stack>
      </Cols>

      <Dialog
        open={dialog === 'pause'}
        title="Pause collection for Temporal"
        size="md"
        onClose={() => setState('active')}
        footer={
          <>
            <Button variant="secondary" onClick={() => setState('active')}>
              Cancel
            </Button>
            <Button variant="primary" onClick={() => setState('paused')}>
              Pause collection
            </Button>
          </>
        }
      >
        <Stack gap="var(--space-15)">
          <Table
            caption="Effects of pausing"
            density="compact"
            columns={EFFECT_COLUMNS}
            rows={EFFECTS.pause}
            getRowKey={(row) => row.effect}
          />
          <Banner tone="attention" title="Three scheduled runs will be cancelled">
            A sync at 20:00Z, a recalculation at 02:00Z and a documentation crawl on 2026-08-27 are
            removed from the schedule. In-flight jobs finish and record their checkpoint.
          </Banner>
        </Stack>
      </Dialog>

      <Dialog
        open={dialog === 'archive'}
        title="Archive Temporal"
        size="md"
        onClose={() => setState('active')}
        footer={
          <>
            <Button variant="secondary" onClick={() => setState('active')}>
              Cancel
            </Button>
            <Button variant="primary" onClick={() => setState('archived')}>
              Archive project
            </Button>
          </>
        }
      >
        <Stack gap="var(--space-15)">
          <Table
            caption="Effects of archiving"
            density="compact"
            columns={EFFECT_COLUMNS}
            rows={EFFECTS.archive}
            getRowKey={(row) => row.effect}
          />
          <Checkbox
            id="archive-alerts"
            label="Also resolve the 2 open alerts for this project"
            description="Left unticked, they stay open and remain visible in the inbox with their original evidence."
          />
        </Stack>
      </Dialog>

      <Dialog
        open={dialog === 'restore'}
        title="Restore Temporal"
        size="md"
        onClose={() => setState('archived')}
        footer={
          <>
            <Button variant="secondary" onClick={() => setState('archived')}>
              Cancel
            </Button>
            <Button variant="primary" onClick={() => setState('active')}>
              Restore project
            </Button>
          </>
        }
      >
        <Stack gap="var(--space-15)">
          <RadioGroup
            name="restore-mode"
            legend="What should happen to the archived gap"
            value={restoreMode}
            onChange={(event) => setRestoreMode(event.target.value)}
            options={[
              {
                value: 'resume',
                label: 'Resume from the last checkpoint',
                description:
                  'Collection continues from 2026-08-20T14:31Z. The archived period stays uncollected and is reported as a coverage gap.',
              },
              {
                value: 'backfill',
                label: 'Backfill the archived period',
                description:
                  'Requests history for the gap. Subject to provider rate limits; the job may take hours and is resumable.',
              },
            ]}
          />
          <Banner tone="info" title="Metrics are not retroactively rewritten">
            Values computed before archiving keep their own cutoffs. A backfill adds evidence for
            the gap and produces new values at a new cutoff alongside them.
          </Banner>
        </Stack>
      </Dialog>

      <Dialog
        open={dialog === 'conflict'}
        title="Another transition is already running"
        tone="danger"
        size="sm"
        onClose={() => setState('active')}
        footer={
          <Button variant="primary" onClick={() => setState('active')}>
            Reload the current state
          </Button>
        }
      >
        <Stack gap="var(--space-15)">
          <p style={{ font: 'var(--type-body)', color: 'var(--text-body)', margin: 0 }}>
            Rafael Costa started an archive transition at 2026-08-20T15:01:40Z. Your request was
            refused and nothing changed. Lifecycle transitions are serialised so two members cannot
            leave a project in an undefined state.
          </p>
          <DefinitionList
            items={[
              { label: 'Running transition', value: 'active → archived', mono: true },
              { label: 'Started by', value: 'Rafael Costa' },
              { label: 'Your request', value: 'active → paused', mono: true },
            ]}
          />
        </Stack>
      </Dialog>

      <Dialog
        open={dialog === 'delete'}
        title="Delete Temporal permanently"
        tone="danger"
        size="lg"
        onClose={() => {
          setTyped('');
          setState('active');
        }}
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => {
                setTyped('');
                setState('active');
              }}
            >
              Cancel
            </Button>
            <Button
              variant="danger"
              disabled={typed !== 'DELETE temporal'}
              onClick={() => {
                setTyped('');
                setState('purging');
              }}
            >
              Delete permanently
            </Button>
          </>
        }
      >
        <Stack gap="var(--space-15)">
          <Table
            caption="Effects of permanent deletion"
            density="compact"
            columns={[
              { key: 'effect', header: 'Affected' },
              {
                key: 'outcome',
                header: 'Effect',
                wrap: true,
                render: (row) => (
                  <span
                    style={{
                      color:
                        row.outcome === 'No.' || row.outcome.startsWith('Permanently')
                          ? 'var(--critical-fg)'
                          : 'inherit',
                    }}
                  >
                    {row.outcome}
                  </span>
                ),
              },
            ]}
            rows={EFFECTS.delete}
            getRowKey={(row) => row.effect}
          />
          <div>
            <span style={{ font: 'var(--type-body)', color: 'var(--text-body)' }}>
              Type <Mono>DELETE temporal</Mono> to confirm. Nothing else is accepted.
            </span>
            <div style={{ marginTop: 'var(--space-1)', maxWidth: 320 }}>
              <TextField
                id="delete-confirm"
                mono
                size="lg"
                value={typed}
                onChange={(event) => setTyped(event.target.value)}
                placeholder="DELETE temporal"
              />
            </div>
          </div>
        </Stack>
      </Dialog>
    </Stack>
  );
}
