import { useState } from 'react';

import {
  Banner,
  Button,
  Checkbox,
  ComparisonMatrix,
  DateRangeField,
  Panel,
  StatusBadge,
  type MatrixCell,
  type MatrixRow,
} from '../../design-system';
import { CUTOFF, PROJECTS, WINDOW } from './fixtures';

const cell = (value: number, display: string): MatrixCell => ({ value, display });

/** Cells are indexed by the fixture project order; a selection reads the matching column. */
const ROWS: readonly MatrixRow[] = [
  {
    metric: 'release_frequency',
    unit: 'releases / 90d',
    comparable: true,
    betterIs: 'higher',
    cells: [
      cell(8, '8'),
      cell(0, '0'),
      cell(14, '14'),
      { status: 'insufficient_data' },
      cell(2, '2'),
    ],
  },
  {
    metric: 'active_contributors_30d',
    unit: 'people',
    comparable: true,
    betterIs: 'higher',
    cells: [cell(119, '119'), cell(7, '7'), cell(412, '412'), cell(31, '31'), cell(44, '44')],
  },
  {
    metric: 'median_time_to_first_response',
    unit: 'hours',
    comparable: true,
    betterIs: 'lower',
    cells: [
      cell(9.4, '9.4'),
      cell(212, '212'),
      cell(4.1, '4.1'),
      { status: 'insufficient_data' },
      cell(28, '28'),
    ],
  },
  {
    metric: 'top_three_author_share',
    unit: 'ratio',
    comparable: true,
    betterIs: 'lower',
    cells: [
      cell(0.71, '0.71'),
      cell(0.94, '0.94'),
      cell(0.28, '0.28'),
      cell(0.62, '0.62'),
      cell(0.55, '0.55'),
    ],
  },
  {
    metric: 'npm_download_change',
    unit: 'ratio · npm only',
    comparable: false,
    cells: [
      cell(0.12, '+12%'),
      { status: 'not_applicable', label: 'No npm package' },
      cell(0.31, '+31%'),
      { status: 'unknown', label: 'Maven only' },
      { status: 'not_applicable', label: 'No npm package' },
    ],
  },
  {
    metric: 'public_advisories_365d',
    unit: 'advisories',
    comparable: true,
    betterIs: 'lower',
    cells: [
      cell(1, '1'),
      { status: 'unknown', label: 'No source' },
      cell(3, '3'),
      cell(0, '0'),
      cell(0, '0'),
    ],
  },
];

export function CompareScreen() {
  const [picked, setPicked] = useState<readonly string[]>(['Temporal', 'Cadence', 'Conductor']);
  const [observationWindow, setObservationWindow] = useState('90d');

  const toggle = (name: string) =>
    setPicked((current) =>
      current.includes(name)
        ? current.filter((entry) => entry !== name)
        : current.length < 5
          ? [...current, name]
          : current,
    );

  const order = PROJECTS.map((project) => project.name);
  const columns = picked.map((name) => order.indexOf(name));
  const shown: MatrixRow[] = ROWS.map((row) => ({
    ...row,
    cells: columns.map((index) => row.cells[index] ?? { status: 'unknown' }),
  }));

  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Panel title="Selection" meta="Two to five projects, one identical window and cutoff">
        <div style={{ display: 'grid', gap: 'var(--space-15)' }}>
          <div style={{ display: 'flex', gap: 'var(--space-2)', flexWrap: 'wrap' }}>
            {PROJECTS.map((project) => (
              <Checkbox
                key={project.id}
                id={`compare-${project.id}`}
                label={project.name}
                description={project.state === 'paused' ? 'paused · retained data' : undefined}
                checked={picked.includes(project.name)}
                onChange={() => toggle(project.name)}
              />
            ))}
          </div>
          <DateRangeField
            value={observationWindow}
            onChange={setObservationWindow}
            from={WINDOW.from}
            to={WINDOW.to}
            cutoff={CUTOFF}
            coverage="common coverage 34d (Conductor)"
          />
          <div
            style={{
              display: 'flex',
              gap: 'var(--space-1)',
              alignItems: 'center',
              flexWrap: 'wrap',
            }}
          >
            <StatusBadge
              status={picked.length >= 2 && picked.length <= 5 ? 'ready' : 'failed'}
              label={`${picked.length} selected`}
              detail={
                picked.length < 2
                  ? 'select at least two'
                  : picked.length > 5
                    ? 'maximum five'
                    : undefined
              }
              size="sm"
            />
            <Button variant="primary" size="md" iconStart="table-2" disabled={picked.length < 2}>
              Create comparison
            </Button>
            <Button variant="secondary" size="md" iconStart="link-2">
              Copy shareable URL
            </Button>
          </div>
        </div>
      </Panel>
      {picked.length >= 2 ? (
        <>
          <Banner tone="attention" title="Common coverage is narrower than the requested window">
            Conductor has 34 days of coverage. Rows depending on the full 90 days report
            insufficient data for that column instead of shrinking the window for everyone.
          </Banner>
          <Panel
            title="Comparable indicators"
            meta={`90d · cutoff ${CUTOFF} · immutable comparison 732684513871298560`}
            padding="0"
            actions={
              <Button size="sm" variant="secondary" iconStart="download">
                Export CSV
              </Button>
            }
          >
            <ComparisonMatrix projects={picked} rows={shown} window="90d" cutoff={CUTOFF} />
          </Panel>
        </>
      ) : null}
    </div>
  );
}
