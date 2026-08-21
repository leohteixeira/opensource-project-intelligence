import { useState } from 'react';

import {
  Banner,
  Button,
  DefinitionList,
  EmptyState,
  EvidenceLink,
  Icon,
  Link,
  MetricValue,
  Pagination,
  Panel,
  StatusBadge,
  Table,
  type TabItem,
  type TableColumn,
} from '../../design-system';
import { Cols, Mono, Provenance, Stack, StateBar } from './kit';
import { ADOPTION, CUTOFF, WINDOW, type Advisory, type Registry } from './fixtures';

const STATES: readonly TabItem[] = [
  { value: 'many', label: 'Many registries' },
  { value: 'single', label: 'One registry' },
  { value: 'none', label: 'No registry linked' },
  { value: 'no_advisories', label: 'No public advisories' },
  { value: 'conflicting', label: 'Conflicting advisory' },
  { value: 'dense', label: 'High-volume timeline' },
];

const SEVERITY_TONE: Record<Advisory['severity'], string> = {
  critical: 'critical',
  high: 'critical',
  moderate: 'attention',
  low: 'unknown',
  unknown: 'unknown',
};

function RegistryPanel({ registry }: { readonly registry: Registry }) {
  return (
    <Panel
      title={registry.registry}
      eyebrow={registry.unit ?? 'no unit'}
      meta={registry.pkg ?? 'no package linked'}
      status={
        registry.status !== 'available' ? <StatusBadge status={registry.status} size="sm" /> : null
      }
      footer={
        <Provenance
          window={registry.window}
          cutoff={registry.collectedAt ?? 'unknown'}
          version={`adoption ${registry.version}`}
        />
      }
    >
      {registry.status === 'available' ? (
        <Stack gap="var(--space-15)">
          <MetricValue
            name={`${registry.registry.toLowerCase().replace(/ /g, '_')}_${registry.unit ?? ''}`}
            label={registry.unit ?? 'Adoption signal'}
            value={registry.value ?? undefined}
            unit={registry.unit ?? undefined}
            status="available"
            version={registry.version}
            window={registry.window}
            delta={registry.change ?? undefined}
            deltaDirection={registry.direction}
            size="md"
          />
          {registry.normalized ? <Mono tone="quiet">normalized: {registry.normalized}</Mono> : null}
          {registry.incomparable ? (
            <Banner tone="neutral" glyph="circle-slash" title="Not comparable across registries">
              {registry.incomparable}
            </Banner>
          ) : null}
        </Stack>
      ) : (
        <Stack gap="var(--space-1)">
          <MetricValue
            label={registry.unit ?? 'Adoption signal'}
            status={registry.status}
            version={registry.version}
            note={registry.note}
            size="md"
          />
          {registry.status === 'unknown' ? (
            <Link href="#" size="sm" evidence>
              Last successful collection · {registry.collectedAt}
            </Link>
          ) : null}
        </Stack>
      )}
    </Panel>
  );
}

export function AdoptionSecurityScreen() {
  const [state, setState] = useState('many');
  const [page, setPage] = useState(1);

  let registries: readonly Registry[] = ADOPTION.registries;

  if (state === 'single') registries = ADOPTION.registries.slice(0, 1);
  if (state === 'none') registries = [];

  let advisories: readonly Advisory[] = ADOPTION.security;

  if (state === 'no_advisories') advisories = [];
  if (state === 'dense') {
    advisories = [
      ...ADOPTION.security,
      ...ADOPTION.security.map((advisory, index) => ({
        ...advisory,
        id: `${advisory.id}-${index}`,
        published: `2025-1${index + 1}-04`,
      })),
    ];
  }

  const advisoryColumns: readonly TableColumn<Advisory>[] = [
    {
      key: 'id',
      header: 'Identifier',
      mono: true,
      render: (row) => (
        <Link href="#" external evidence size="sm">
          {row.id}
        </Link>
      ),
    },
    {
      key: 'title',
      header: 'Public record',
      wrap: true,
      render: (row) => (
        <span style={{ display: 'grid', gap: 2 }}>
          <span>{row.title}</span>
          {row.note ? (
            <span style={{ font: 'var(--type-caption)', color: 'var(--attention-fg)' }}>
              {row.note}
            </span>
          ) : null}
        </span>
      ),
    },
    {
      key: 'severity',
      header: 'Published severity',
      render: (row) =>
        row.severity === 'unknown' ? (
          <StatusBadge status="unknown" size="sm" label="Unknown" />
        ) : (
          <span
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
              color: `var(--${SEVERITY_TONE[row.severity]}-fg)`,
              font: 'var(--type-ui)',
            }}
          >
            <Icon
              name={
                row.severity === 'high' || row.severity === 'critical'
                  ? 'triangle-alert'
                  : 'circle-alert'
              }
              size={13}
            />
            {row.severity}
          </span>
        ),
    },
    { key: 'affected', header: 'Affected', mono: true },
    {
      key: 'fixedIn',
      header: 'Fixed in',
      mono: true,
      render: (row) =>
        row.fixedIn ?? <span style={{ color: 'var(--text-tertiary)' }}>Unknown</span>,
    },
    { key: 'published', header: 'Published', mono: true },
    {
      key: 'state',
      header: 'Record state',
      render: (row) => (
        <StatusBadge
          status={row.state === 'resolved' ? 'ready' : 'stale'}
          size="sm"
          label={row.state === 'resolved' ? 'Fix published' : 'Conflicting records'}
        />
      ),
    },
  ];

  return (
    <Stack>
      <StateBar
        items={STATES}
        value={state}
        onChange={setState}
        route={`/en/projects/temporal/adoption-security?window=90d${state === 'dense' ? `&page=${page}` : ''}`}
      />

      <Panel
        title="Adoption signals"
        eyebrow="One panel per registry — signals are never summed"
        meta={`window ${WINDOW.label} · cutoff ${CUTOFF} · adoption v2`}
        actions={
          <Button variant="secondary" size="sm" iconStart="download">
            Export
          </Button>
        }
      >
        {registries.length === 0 ? (
          <EmptyState
            glyph="package"
            title="No package registry is linked to this project"
            action={
              <Button variant="primary" iconStart="package">
                Link a package source
              </Button>
            }
          >
            Adoption is Not applicable rather than zero: without a registry there is no download or
            resolution evidence to read. Repository stars are not used as an adoption signal.
          </EmptyState>
        ) : (
          <Stack gap="var(--space-15)">
            <div
              style={{
                display: 'grid',
                gridTemplateColumns:
                  state === 'single' ? '1fr' : 'repeat(auto-fit,minmax(260px,1fr))',
                gap: 'var(--space-2)',
              }}
            >
              {registries.map((registry) => (
                <RegistryPanel key={registry.registry} registry={registry} />
              ))}
            </div>
            <Banner tone="info" title="Each registry reports a different quantity">
              npm and PyPI both report download events and are normalized to a daily rate for
              reading only. Maven Central reports unique-IP resolutions and is not normalized
              against them. The stored values are the source&apos;s own numbers in the source&apos;s
              own unit.
            </Banner>
          </Stack>
        )}
      </Panel>

      <Panel
        title="Public security evidence"
        eyebrow="Published records only"
        meta="source GitHub advisories · cutoff 2026-08-14T04:00:00Z · security v2"
        status={
          <StatusBadge
            status="stale"
            size="sm"
            detail="advisory source last succeeded 2026-08-14T04:00Z"
          />
        }
        actions={
          <Button variant="secondary" size="sm" iconStart="history">
            Request collection
          </Button>
        }
        footer={
          <Provenance
            window="365d"
            cutoff="2026-08-14T04:00:00Z"
            version="security v2"
            extra="4 of 5 advisory pages collected"
          />
        }
      >
        <Stack gap="var(--space-15)">
          <Banner tone="neutral" glyph="shield-alert" title="This is not a vulnerability scan">
            OPI reads advisories, changelogs and release notes that were published about this
            project. It does not scan code or dependencies. An empty result means no public record
            was found in the window, which is not evidence that the project is free of
            vulnerabilities.
          </Banner>

          {state === 'conflicting' ? (
            <Banner
              tone="attention"
              title="Two published records disagree"
              actions={
                <Button variant="secondary" size="md">
                  Open both records
                </Button>
              }
            >
              GHSA-3c8p-r4jd-x21h is marked withdrawn by its publisher, while the v1.23.1 changelog
              still describes a security fix referencing it. Both are retained and neither is
              preferred; the security dimension reports Unknown for the affected range.
            </Banner>
          ) : null}

          {advisories.length === 0 ? (
            <EmptyState
              glyph="circle-help"
              title="No public advisory was found for this project"
              action={
                <Button variant="secondary" iconStart="clock">
                  Widen the window to 365 days
                </Button>
              }
            >
              The advisory source returned 0 records between {WINDOW.from} and {WINDOW.to}. The
              security dimension therefore reports Unknown, not a pass. Confirm the project&apos;s
              advisory namespace is configured correctly before reading this as a positive signal.
            </EmptyState>
          ) : (
            <Table
              caption="Public security records collected at cutoff 2026-08-14T04:00:00Z"
              columns={advisoryColumns}
              rows={advisories}
              getRowKey={(row) => row.id}
              density="compact"
              footer={
                state === 'dense' ? (
                  <Pagination
                    page={page}
                    pageSize={6}
                    total={214}
                    hasMore={page < 36}
                    onPrev={() => setPage(Math.max(1, page - 1))}
                    onNext={() => setPage(page + 1)}
                    label="records"
                  />
                ) : null
              }
            />
          )}

          <Cols template="minmax(0,1fr) minmax(0,1fr)">
            <Panel
              title="Security release referenced in the changelog"
              meta={`${ADOPTION.securityRelease.tag} · ${ADOPTION.securityRelease.date}`}
            >
              <Stack gap="var(--space-1)">
                <span style={{ font: 'var(--type-body)', color: 'var(--text-body)' }}>
                  {ADOPTION.securityRelease.claim}
                </span>
                <EvidenceLink
                  kind="changelog"
                  title="CHANGELOG.md · v1.23.1 Security"
                  href="#"
                  source="github · temporalio/temporal"
                  collectedAt="2026-08-09T18:04Z"
                />
              </Stack>
            </Panel>
            <Panel title="What this panel does not tell you">
              <DefinitionList
                dense
                items={[
                  { label: 'Dependency vulnerabilities', value: 'Not collected' },
                  { label: 'Private or embargoed advisories', value: 'Not collected' },
                  { label: 'Code scanning results', value: 'Not collected' },
                  { label: 'Advisory namespace', value: 'temporalio/*', mono: true },
                ]}
              />
            </Panel>
          </Cols>
        </Stack>
      </Panel>
    </Stack>
  );
}
