import { useState } from 'react';

import {
  Banner,
  Button,
  Checkbox,
  DateRangeField,
  DefinitionList,
  EmptyState,
  FormField,
  JobProgress,
  Panel,
  RadioGroup,
  Select,
  StatusBadge,
  Table,
  TextField,
  type TabItem,
  type TableColumn,
} from '../../design-system';
import { Cols, Mono, Provenance, Stack, StateBar } from './kit';
import { CUTOFF, EXPORTS, WINDOW, type Artifact } from './fixtures';

const STATES: readonly TabItem[] = [
  { value: 'configure', label: 'Configuration' },
  { value: 'invalid', label: 'Validation failure' },
  { value: 'running', label: 'Running' },
  { value: 'succeeded', label: 'Artifact ready' },
  { value: 'zero', label: 'Zero rows' },
  { value: 'expired', label: 'Expired download' },
  { value: 'quota', label: 'Size quota' },
  { value: 'revoked', label: 'Permission revoked' },
];

export function ExportsScreen({ onBack }: { readonly onBack: () => void }) {
  const [state, setState] = useState('configure');
  const [resource, setResource] = useState('metrics');
  const [format, setFormat] = useState('csv');
  const [observationWindow, setObservationWindow] = useState('90d');
  const [includeMissing, setIncludeMissing] = useState(true);

  const invalid = state === 'invalid';
  const jobs = state === 'running' ? EXPORTS.jobs : EXPORTS.jobs.slice(1);
  const [artifact, zeroArtifact] = EXPORTS.artifacts;

  const artifactRows: readonly Artifact[] =
    state === 'zero'
      ? zeroArtifact
        ? [{ ...zeroArtifact, state: 'ready', expiresIn: '23 h 12 min' }]
        : []
      : state === 'expired'
        ? EXPORTS.artifacts.slice().reverse()
        : artifact
          ? [artifact]
          : [];

  const artifactColumns: readonly TableColumn<Artifact>[] = [
    {
      key: 'resource',
      header: 'Resource',
      wrap: true,
      render: (row) => (
        <span style={{ display: 'grid', gap: 2 }}>
          <span style={{ fontWeight: 'var(--weight-medium)' }}>{row.resource}</span>
          <Mono tone="quiet">
            {row.id} · {row.format}
          </Mono>
        </span>
      ),
    },
    {
      key: 'rows',
      header: 'Rows',
      numeric: true,
      mono: true,
      render: (row) =>
        row.rows === '0' ? (
          <span style={{ display: 'grid', gap: 2, justifyItems: 'end' }}>
            <span>0</span>
            <Mono tone="quiet">valid, empty</Mono>
          </span>
        ) : (
          row.rows
        ),
    },
    { key: 'bytes', header: 'Size', numeric: true, mono: true },
    { key: 'cutoff', header: 'Cutoff', mono: true },
    { key: 'version', header: 'Version', mono: true },
    {
      key: 'expiresIn',
      header: 'Expires',
      render: (row) =>
        row.expiresIn ? (
          <StatusBadge status="ready" size="sm" label={`in ${row.expiresIn}`} />
        ) : (
          <StatusBadge status="failed" size="sm" label="Expired" detail={row.expiresAt} />
        ),
    },
    {
      key: 'action',
      header: '',
      render: (row) =>
        row.expiresIn ? (
          <Button variant="secondary" size="sm" iconStart="download">
            Download
          </Button>
        ) : (
          <Button variant="secondary" size="sm" iconStart="history">
            Request again
          </Button>
        ),
    },
  ];

  const form = (
    <Panel
      title="New export"
      eyebrow="Snapshot at request time"
      meta="an export is a Job; the artifact carries the cutoff, catalog version and locale it was built with"
      footer={
        <Provenance
          window={`window ${observationWindow}`}
          cutoff={CUTOFF}
          version="catalog v3"
          extra="locale en"
        />
      }
    >
      <Stack gap="var(--space-2)">
        {state === 'revoked' ? (
          <Banner
            tone="critical"
            title="Your role no longer permits this export"
            actions={
              <Button variant="secondary" size="md">
                Contact an Admin
              </Button>
            }
          >
            Audit events require the Admin role. Your role changed to Analyst at
            2026-08-20T14:10:02Z, so this request was refused before any row was read. Existing
            artifacts you created remain downloadable until they expire.
          </Banner>
        ) : null}

        <FormField
          id="export-resource"
          label="Resource"
          hint="One resource per export. Evidence JSON keeps the source records; CSV keeps one row per value."
        >
          <Select
            id="export-resource"
            options={EXPORTS.resources.map((entry) => ({
              value: entry.value,
              label: entry.label,
              disabled: state === 'revoked' && entry.value === 'audit',
            }))}
            value={resource}
            onChange={(event) => setResource(event.target.value)}
            size="lg"
          />
        </FormField>

        <FormField id="export-window" label="Window">
          <DateRangeField
            value={observationWindow}
            from={WINDOW.from}
            to={WINDOW.to}
            cutoff={CUTOFF}
            coverage={invalid ? '0d of 90d' : undefined}
            onChange={setObservationWindow}
          />
        </FormField>

        <RadioGroup
          name="export-format"
          legend="Format"
          value={format}
          onChange={(event) => setFormat(event.target.value)}
          options={[
            {
              value: 'csv',
              label: 'CSV',
              description:
                'One row per value, with unit, window, cutoff and definition version as columns.',
            },
            {
              value: 'json',
              label: 'Evidence JSON',
              description:
                'Values plus the source records each was derived from. Larger, and traceable.',
            },
          ]}
        />

        <FormField
          id="export-filter"
          label="Filter"
          optional
          hint="The same filter grammar as the URL. Applied at snapshot time."
          error={
            invalid
              ? 'state=deleted is not an accepted value for this resource. Accepted: active, paused, archived.'
              : undefined
          }
        >
          <TextField
            id="export-filter"
            mono
            size="lg"
            defaultValue={invalid ? 'state=deleted' : 'state=active'}
          />
        </FormField>

        <Checkbox
          id="export-include-missing"
          label="Include rows with missing values"
          description="Unknown, Not applicable and Insufficient data are written as those words. They are never written as 0 or as an empty cell."
          checked={includeMissing}
          onChange={(event) => setIncludeMissing(event.target.checked)}
        />

        <div style={{ display: 'flex', gap: 'var(--space-1)', flexWrap: 'wrap' }}>
          <Button
            variant="primary"
            size="lg"
            iconStart="download"
            disabled={invalid || state === 'revoked'}
          >
            Request export
          </Button>
          <Button variant="ghost" size="lg" onClick={onBack}>
            Cancel
          </Button>
        </div>
      </Stack>
    </Panel>
  );

  return (
    <Stack>
      <StateBar
        items={STATES}
        value={state}
        onChange={setState}
        route={`/en/exports?resource=${resource}&window=${observationWindow}&format=${format}`}
      />

      {state === 'quota' ? (
        <Banner
          tone="critical"
          title="The export exceeded its artifact quota"
          actions={
            <>
              <Button variant="primary" size="md" iconStart="history">
                Retry with a 30-day window
              </Button>
              <Button variant="secondary" size="md">
                Split by repository
              </Button>
            </>
          }
        >
          The request reached 200 MB at 4,120 of 61,004 rows and stopped. No partial artifact is
          offered, because a truncated evidence file is indistinguishable from a complete one once
          downloaded. Narrow the window or the resource and request again.
        </Banner>
      ) : null}

      {state === 'succeeded' ? (
        <Banner
          tone="positive"
          title="The artifact is ready and expires in 22 h 29 min"
          actions={
            <Button variant="primary" size="md" iconStart="download">
              Download 18.4 MB
            </Button>
          }
        >
          Rows were read from a single snapshot taken at 2026-08-20T13:02:00Z. Projects updated
          after that instant are not reflected, so the file is internally consistent rather than
          current.
        </Banner>
      ) : null}

      <Cols template="minmax(0,460px) minmax(0,1fr)">
        {form}
        <Stack>
          <Panel
            title="Export jobs"
            meta="durable; a reload or a worker restart resumes from the checkpoint"
            actions={
              <Button variant="secondary" size="sm" iconStart="history">
                Refresh
              </Button>
            }
          >
            {state === 'configure' ? (
              <EmptyState
                compact
                glyph="download"
                title="No export has been requested in this session"
              >
                Requested exports appear here with their progress, checkpoint and outcome. A
                duplicate request for the same resource, window and format is coalesced into the
                running job rather than queued twice.
              </EmptyState>
            ) : (
              <Stack gap="var(--space-15)">
                {jobs.map((job) => (
                  <JobProgress
                    key={job.id}
                    {...job}
                    kind={`export · ${job.resource} · ${job.format}`}
                    actions={
                      job.state === 'running' ? (
                        <Button variant="secondary" size="sm">
                          Cancel
                        </Button>
                      ) : job.state === 'failed' ? (
                        <Button variant="secondary" size="sm" iconStart="history">
                          Retry
                        </Button>
                      ) : null
                    }
                  />
                ))}
              </Stack>
            )}
          </Panel>

          <Panel
            title="Artifacts"
            meta="every artifact expires 24 hours after it is built"
            footer={
              <Mono tone="quiet">
                expiry is enforced server-side; an expired link returns 410 Gone with a Problem
                Details body
              </Mono>
            }
          >
            {state === 'configure' || state === 'invalid' || state === 'revoked' ? (
              <EmptyState compact glyph="file-text" title="No artifact is available">
                An artifact appears here once its job succeeds. Downloads are not e-mailed and are
                not available outside this session&apos;s authenticated requests.
              </EmptyState>
            ) : (
              <Table
                caption={`Export artifacts at ${CUTOFF}`}
                density="compact"
                columns={artifactColumns}
                rows={artifactRows}
                getRowKey={(row) => row.id}
              />
            )}
          </Panel>

          {state === 'zero' ? (
            <Banner
              tone="info"
              title="A zero-row export is a valid result"
              actions={
                <Button variant="secondary" size="md" iconStart="download">
                  Download 412 B
                </Button>
              }
            >
              No public advisory exists for this project in the requested window, so the file
              contains its header row and no data rows. It is not an error and it is not a statement
              that the project has no vulnerabilities.
            </Banner>
          ) : null}

          {state === 'expired' ? (
            <Banner
              tone="attention"
              title="One download link has expired"
              actions={
                <Button variant="primary" size="md" iconStart="history">
                  Request the export again
                </Button>
              }
            >
              The artifact built at 2026-08-19T09:10:12Z passed its 24-hour lifetime and was
              deleted. Requesting it again produces a new snapshot at a new cutoff — it will not
              reproduce the old file byte for byte.
            </Banner>
          ) : null}

          <Panel
            title="What an artifact always carries"
            meta="so a file read months later is still interpretable"
          >
            <DefinitionList
              columns={2}
              items={[
                { label: 'Cutoff', value: '2026-08-20T13:02:00Z', mono: true },
                { label: 'Window', value: '90d · 2026-05-22 to 2026-08-19', mono: true },
                { label: 'Metric catalog version', value: 'catalog v3', mono: true },
                { label: 'Locale of labels', value: 'en', mono: true },
                {
                  label: 'Missing-value words',
                  value: 'Unknown, Not applicable, Insufficient data',
                },
                { label: 'Requested by', value: 'Ana Silva · analyst' },
              ]}
            />
          </Panel>
        </Stack>
      </Cols>
    </Stack>
  );
}
