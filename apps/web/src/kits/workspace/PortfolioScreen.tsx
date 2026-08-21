import { useState } from 'react';

import {
  Banner,
  Button,
  CoverageDisclosure,
  DateRangeField,
  EvidenceLink,
  Icon,
  Panel,
  RadarList,
  StatusBadge,
  TrendChart,
} from '../../design-system';
import {
  CUTOFF,
  PROJECTS,
  RADAR,
  TREND_FORECAST,
  TREND_SERIES,
  WINDOW,
  type WorkspaceProject,
} from './fixtures';

function AttentionRow({
  project,
  onOpen,
}: {
  readonly project: WorkspaceProject;
  readonly onOpen: (project: WorkspaceProject) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onOpen(project)}
      className="opi-item"
      style={{
        width: '100%',
        textAlign: 'left',
        display: 'flex',
        gap: 'var(--space-15)',
        alignItems: 'center',
        minHeight: 52,
        padding: 'var(--space-1) var(--space-15)',
        border: 0,
        borderBottom: '1px solid var(--border-table)',
        background: 'transparent',
        cursor: 'pointer',
      }}
    >
      <span style={{ flex: 1, minWidth: 0, display: 'grid', gap: 2 }}>
        <span
          style={{
            display: 'flex',
            gap: 'var(--space-075)',
            alignItems: 'center',
            flexWrap: 'wrap',
          }}
        >
          <span
            style={{
              font: 'var(--type-body-strong)',
              fontSize: 'var(--text-sm)',
              color: 'var(--text-primary)',
            }}
          >
            {project.name}
          </span>
          <StatusBadge status={project.recommendation} size="sm" />
          {project.freshness === 'stale' ? (
            <StatusBadge status="stale" detail="last valid 2026-08-14" size="sm" />
          ) : null}
          {project.alerts ? (
            <span style={{ font: 'var(--type-caption)', color: 'var(--critical-fg)' }}>
              {project.alerts} alerts
            </span>
          ) : null}
        </span>
        <span style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
          {project.attention ?? project.description}
        </span>
      </span>
      <span
        style={{
          width: 42,
          textAlign: 'right',
          font: 'var(--type-body-strong)',
          fontVariantNumeric: 'tabular-nums',
          color: 'var(--text-primary)',
        }}
      >
        {project.overall === null ? '—' : project.overall}
      </span>
      <Icon name="chevron-right" size={15} style={{ color: 'var(--text-tertiary)' }} />
    </button>
  );
}

export function PortfolioScreen({
  onOpenProject,
  onGo,
}: {
  readonly onOpenProject: (project: WorkspaceProject) => void;
  readonly onGo: (key: string) => void;
}) {
  const [observationWindow, setObservationWindow] = useState('90d');
  const needsAttention = PROJECTS.filter((project) => project.attention);

  return (
    <div style={{ display: 'grid', gap: 'var(--space-3)' }}>
      <DateRangeField
        value={observationWindow}
        onChange={setObservationWindow}
        from={WINDOW.from}
        to={WINDOW.to}
        cutoff={CUTOFF}
        coverage="90d of 90d"
      />
      <Panel
        title="Requires attention"
        status={
          <StatusBadge
            status="available"
            label={`${needsAttention.length} of 5 projects`}
            size="sm"
          />
        }
        meta={`90d · cutoff ${CUTOFF}`}
        padding="0"
        actions={
          <Button size="sm" variant="ghost" iconEnd="arrow-right" onClick={() => onGo('projects')}>
            All projects
          </Button>
        }
      >
        <div>
          {needsAttention.map((project) => (
            <AttentionRow key={project.id} project={project} onOpen={onOpenProject} />
          ))}
        </div>
      </Panel>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))',
          gap: 'var(--space-2)',
          alignItems: 'start',
        }}
      >
        <Panel title="Policy recommendations" meta="Default adoption policy v4 · 90d">
          <div style={{ display: 'grid', gap: 'var(--space-075)' }}>
            {PROJECTS.map((project) => (
              <div
                key={project.id}
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  gap: 'var(--space-1)',
                  alignItems: 'center',
                  padding: '3px 0',
                  borderBottom: '1px solid var(--border-table)',
                }}
              >
                <span style={{ font: 'var(--type-table)', color: 'var(--text-body)' }}>
                  {project.name}
                </span>
                <StatusBadge status={project.recommendation} size="sm" />
              </div>
            ))}
          </div>
        </Panel>

        <Panel
          title="Observed trends and early warnings"
          meta="observation 90d · baseline 90d · method v2"
          footer={<span>Forecast horizon 120d · confidence 0.62 · model maintainer_count v2</span>}
        >
          <div style={{ display: 'grid', gap: 'var(--space-15)' }}>
            <div
              style={{
                display: 'flex',
                gap: 'var(--space-1)',
                alignItems: 'center',
                flexWrap: 'wrap',
              }}
            >
              <StatusBadge status="observed" label="Observed decrease" size="sm" />
              <span style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
                Temporal · active contributors −38% vs baseline
              </span>
            </div>
            <TrendChart
              label="Active contributors"
              unit="people"
              width={300}
              height={64}
              series={TREND_SERIES}
              forecast={TREND_FORECAST}
            />
            <div
              style={{
                display: 'flex',
                gap: 'var(--space-1)',
                alignItems: 'center',
                flexWrap: 'wrap',
              }}
            >
              <StatusBadge status="forecast" label="Early warning" size="sm" />
              <span style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
                Cadence · maintainer count reaches 0 within 120d
              </span>
            </div>
          </div>
        </Panel>

        <Panel
          title="Data freshness and coverage"
          status={<StatusBadge status="stale" detail="3 of 32 sources" size="sm" />}
          meta="last successful collection 2026-08-20T14:31Z"
        >
          <CoverageDisclosure
            requested="90d"
            actual="90d"
            ratio={0.94}
            sources={[
              { name: 'github', value: '100%' },
              { name: 'npm', value: '100%' },
              { name: 'docs', value: '78%' },
            ]}
            missing={['discussions (Temporal)', 'advisories (Cadence)']}
            cutoff={CUTOFF}
          />
        </Panel>

        <Panel
          title="Synchronization failures"
          tone="critical"
          status={<StatusBadge status="failed" size="sm" />}
          actions={
            <Button size="sm" variant="secondary" iconStart="rotate-ccw">
              Retry all
            </Button>
          }
        >
          <Banner tone="critical" title="npm registry rate limit reached">
            Quota resets at 15:05Z. The history request for Temporal will resume from checkpoint
            npm:downloads:2026-05-01 without duplicating collected facts.
          </Banner>
        </Panel>

        <Panel
          title="Important releases"
          meta="analysed by prompt releases@9"
          footer={
            <span style={{ display: 'inline-flex', gap: 4, alignItems: 'center' }}>
              <Icon name="sparkles" size={12} style={{ color: 'var(--ai-fg)' }} />
              AI-generated claims · local-ollama qwen2.5:32b · each claim cited
            </span>
          }
        >
          <div style={{ display: 'grid', gap: 'var(--space-075)' }}>
            <EvidenceLink
              kind="release"
              title="OpenTelemetry Collector v0.104.0 — 1 breaking change, 2 security fixes"
              source="github · open-telemetry/opentelemetry-collector"
              collectedAt="2026-08-19T17:20Z"
              href="#"
            />
            <EvidenceLink
              kind="release"
              title="Temporal v1.24.0 — 4 features, 1 deprecation"
              source="github · temporalio/temporal"
              collectedAt="2026-08-18T09:02Z"
              href="#"
            />
            <EvidenceLink
              kind="changelog"
              title="Cadence v1.2.9 — security fix claimed, no advisory found"
              source="changelog · cadenceworkflow.io"
              collectedAt="2026-08-20T11:02Z"
              href="#"
            />
          </div>
        </Panel>

        <Panel
          title="Radar placement"
          meta="derived from Default adoption policy v4"
          actions={
            <Button size="sm" variant="ghost" iconEnd="arrow-right" onClick={() => onGo('radar')}>
              Radar
            </Button>
          }
        >
          <RadarList entries={RADAR.slice(0, 3)} />
        </Panel>
      </div>
    </div>
  );
}
