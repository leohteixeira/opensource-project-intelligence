import { useState } from 'react';

import {
  Banner,
  Button,
  DefinitionList,
  Dialog,
  FormField,
  Icon,
  Link,
  Menu,
  Pagination,
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
import {
  ASSOCIATIONS,
  CUTOFF,
  REPOSITORIES,
  SOURCE_INVENTORY,
  type Association,
  type Repository,
  type SourceInventoryRow,
} from './fixtures';

const STATES: readonly TabItem[] = [
  { value: 'repositories', label: 'Many repositories' },
  { value: 'single', label: 'One repository' },
  { value: 'add', label: 'Add source and rejections' },
  { value: 'associations', label: 'Associations' },
  { value: 'concurrent', label: 'Concurrent correction' },
  { value: 'archived', label: 'Archived read-only' },
];

interface ReplaceTarget {
  readonly id?: string;
  readonly url: string;
  readonly concurrent?: boolean;
}

interface RefusalRule {
  readonly rule: string;
  readonly outcome: string;
}

const REFUSALS: readonly RefusalRule[] = [
  {
    rule: 'Private, loopback or link-local host',
    outcome: 'Refused before any network request. The address is echoed back so a typo is visible.',
  },
  {
    rule: 'Duplicate URL on this project',
    outcome:
      'Refused with a link to the existing source. github.com/temporalio/temporal is already configured as a github source.',
  },
  {
    rule: 'Duplicate URL on another active project',
    outcome:
      'Accepted, with a warning naming the other project. Two projects may legitimately watch one repository.',
  },
  {
    rule: 'Unsupported kind',
    outcome:
      'Refused. gitlab is not in the source catalogue for this deployment; the accepted kinds are listed in the field.',
  },
  {
    rule: 'Source limit reached',
    outcome: 'Refused at 12 sources per project. Remove a source or raise the deployment limit.',
  },
  {
    rule: 'Unsupported role for the kind',
    outcome: 'Refused. A registry source cannot take the documentation role.',
  },
];

export function SourcesScreen({ onOpenLifecycle }: { readonly onOpenLifecycle: () => void }) {
  const [state, setState] = useState('repositories');
  const [primary, setPrimary] = useState('732684517001');
  const [replace, setReplace] = useState<ReplaceTarget | null>(null);
  const readOnly = state === 'archived';
  const repos = state === 'single' ? REPOSITORIES.slice(0, 1) : REPOSITORIES;

  const repoColumns: readonly TableColumn<Repository>[] = [
    {
      key: 'primary',
      header: 'Primary',
      width: '84px',
      render: (row) => (
        <label
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            minHeight: 32,
            cursor: readOnly ? 'default' : 'pointer',
          }}
        >
          <input
            type="radio"
            name="primary-repo"
            checked={primary === row.id}
            disabled={readOnly}
            onChange={() => {
              if (primary !== row.id) setReplace({ id: row.id, url: row.url });
            }}
            aria-label={`Make ${row.url} the primary repository`}
            style={{ width: 16, height: 16, accentColor: 'var(--blue-500)' }}
          />
          {primary === row.id ? <Mono tone="quiet">primary</Mono> : null}
        </label>
      ),
    },
    {
      key: 'url',
      header: 'Repository',
      mono: true,
      wrap: true,
      render: (row) => (
        <Link href="#" external evidence size="sm">
          {row.url}
        </Link>
      ),
    },
    {
      key: 'roleLabel',
      header: 'Role',
      render: (row) =>
        row.role === 'primary' ? (
          <StatusBadge status="active" size="sm" label="Primary" />
        ) : (
          <span style={{ font: 'var(--type-ui)', color: 'var(--text-secondary)' }}>{row.role}</span>
        ),
    },
    {
      key: 'state',
      header: 'State',
      render: (row) => (
        <StatusBadge
          status={row.state}
          size="sm"
          detail={row.state === 'stale' ? 'derived results stale' : undefined}
        />
      ),
    },
    {
      key: 'derived',
      header: 'Derived results',
      wrap: true,
      render: (row) =>
        row.derived === 'fresh' ? (
          <span style={{ color: 'var(--text-secondary)' }}>Current</span>
        ) : (
          <span style={{ color: 'var(--attention-fg)' }}>{row.derived}</span>
        ),
    },
    { key: 'lastCollection', header: 'Last collection', mono: true },
    {
      key: 'menu',
      header: '',
      width: '44px',
      render: (row) => (
        <Menu
          align="end"
          triggerIcon="ellipsis-vertical"
          triggerLabel={`Actions for ${row.url}`}
          items={
            readOnly
              ? [
                  { label: 'Collection is stopped while archived', disabled: true },
                  { label: 'Open lifecycle', icon: 'archive', onSelect: onOpenLifecycle },
                ]
              : [
                  { label: 'Change role', icon: 'git-branch' },
                  { label: 'Request collection', icon: 'history' },
                  { separator: true },
                  {
                    label: 'Remove repository',
                    icon: 'octagon-x',
                    danger: true,
                    disabled: primary === row.id,
                  },
                ]
          }
        />
      ),
    },
  ];

  const sourceColumns: readonly TableColumn<SourceInventoryRow>[] = [
    { key: 'kind', header: 'Kind', mono: true },
    { key: 'url', header: 'Source', mono: true, wrap: true },
    {
      key: 'scope',
      header: 'Crawl scope',
      wrap: true,
      render: (row) =>
        row.scope === '—' ? (
          <StatusBadge status="not_applicable" size="sm" detail="no package linked" />
        ) : (
          <Mono>{row.scope}</Mono>
        ),
    },
    {
      key: 'state',
      header: 'State',
      render: (row) => <StatusBadge status={row.state} size="sm" />,
    },
    {
      key: 'credential',
      header: 'Credential',
      mono: true,
      render: (row) =>
        row.credential.includes('redacted') ? (
          <span
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 5,
              color: 'var(--text-secondary)',
            }}
          >
            <Icon name="circle-slash" size={12} />
            {row.credential}
          </span>
        ) : (
          row.credential
        ),
    },
    { key: 'limit', header: 'Rate limit', mono: true },
  ];

  const associationColumns: readonly TableColumn<Association>[] = [
    { key: 'source', header: 'Source', mono: true, wrap: true },
    { key: 'target', header: 'Linked repository', mono: true, wrap: true },
    { key: 'confidence', header: 'Confidence', numeric: true, mono: true },
    {
      key: 'state',
      header: 'Link',
      render: (row) => (
        <StatusBadge
          status={
            row.state === 'automatic'
              ? 'available'
              : row.state === 'uncertain'
                ? 'unknown'
                : 'ready'
          }
          size="sm"
          label={
            row.state === 'automatic'
              ? 'Automatic'
              : row.state === 'uncertain'
                ? 'Uncertain'
                : 'Corrected'
          }
          detail={row.state === 'corrected' ? `by ${row.correctedBy}` : undefined}
        />
      ),
    },
    { key: 'basis', header: 'Basis', wrap: true },
    {
      key: 'action',
      header: '',
      render: (row) =>
        readOnly ? null : (
          <Button
            variant="secondary"
            size="sm"
            onClick={() => {
              if (state === 'concurrent') setReplace({ url: row.source, concurrent: true });
            }}
          >
            {row.state === 'uncertain' ? 'Confirm or correct' : 'Correct'}
          </Button>
        ),
    },
  ];

  return (
    <Stack>
      <StateBar
        items={STATES}
        value={state}
        onChange={setState}
        route="/en/projects/temporal/sources"
      />

      {readOnly ? (
        <Banner
          tone="neutral"
          glyph="archive"
          title="This project is archived; sources are read-only"
          actions={
            <Button variant="secondary" size="md" onClick={onOpenLifecycle}>
              Open lifecycle
            </Button>
          }
        >
          Repositories, sources and associations are shown as they were at 2026-08-20T15:02:11Z.
          Adding, removing, correcting and collecting are all unavailable until the project is
          restored.
        </Banner>
      ) : null}

      {state === 'single' ? (
        <Banner tone="info" title="One repository is a complete configuration">
          A project needs exactly one primary repository. Additional repositories are optional and
          only widen the evidence; nothing on this surface is incomplete because there is one.
        </Banner>
      ) : null}

      <Panel
        title="Repositories"
        eyebrow="Exactly one primary"
        meta={`${repos.length} repositories · the primary repository names the project and anchors its identity`}
        actions={
          readOnly ? null : (
            <Button variant="primary" size="sm" iconStart="git-branch">
              Add repository
            </Button>
          )
        }
        footer={
          <Provenance
            cutoff={CUTOFF}
            version="identity v2"
            extra="changing the primary repository does not rename the project"
          />
        }
      >
        <Table
          caption={`Repositories at cutoff ${CUTOFF}`}
          columns={repoColumns}
          rows={repos}
          getRowKey={(row) => row.id}
          density="compact"
        />
      </Panel>

      {state === 'add' ? (
        <Cols template="minmax(0,460px) minmax(0,1fr)">
          <Panel
            title="Add a source"
            meta="a source is accepted only when its kind, host and scope are all valid"
          >
            <Stack gap="var(--space-2)">
              <FormField
                id="source-kind"
                label="Kind"
                hint="Only catalogued kinds are accepted. An unsupported kind is refused rather than stored as unknown."
              >
                <Select
                  id="source-kind"
                  size="lg"
                  defaultValue="gitlab"
                  options={[
                    { value: 'github', label: 'github' },
                    { value: 'npm', label: 'npm' },
                    { value: 'pypi', label: 'pypi' },
                    { value: 'documentation', label: 'documentation' },
                    { value: 'gitlab', label: 'gitlab — not supported in this deployment' },
                  ]}
                />
              </FormField>
              <FormField
                id="source-url"
                label="Source URL"
                required
                error="10.0.4.19 resolves to a private network address. Private and link-local hosts are refused before any request is made."
                hint="Public hosts only. The URL is resolved and checked before it is stored."
              >
                <TextField
                  id="source-url"
                  mono
                  size="lg"
                  type="url"
                  defaultValue="http://10.0.4.19/git/temporal.git"
                />
              </FormField>
              <FormField
                id="source-scope"
                label="Crawl scope"
                optional
                hint="Path prefixes for documentation sources. Ignored for registries."
              >
                <TextField id="source-scope" mono size="lg" defaultValue="/docs/**" />
              </FormField>
              <div style={{ display: 'flex', gap: 'var(--space-1)' }}>
                <Button variant="primary" size="lg" disabled>
                  Add source
                </Button>
                <Button variant="ghost" size="lg">
                  Cancel
                </Button>
              </div>
            </Stack>
          </Panel>
          <Stack>
            <Banner tone="critical" title="Three rejections are shown together for this kit">
              Each of the rules below refuses a request before anything is stored. In the product
              they appear one at a time, on the field that failed.
            </Banner>
            <Panel title="Refusal rules" meta="stated in full, with the recovery action">
              <Table
                caption="Source validation rules and their refusals"
                density="compact"
                columns={[
                  { key: 'rule', header: 'Rule', wrap: true },
                  { key: 'outcome', header: 'What the member sees', wrap: true },
                ]}
                rows={REFUSALS}
                getRowKey={(row) => row.rule}
              />
            </Panel>
          </Stack>
        </Cols>
      ) : null}

      <Panel
        title="Source inventory"
        eyebrow="Crawl scope and capability"
        meta="5 sources · credentials are never displayed, only their presence"
        actions={
          readOnly ? null : (
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setState('add')}
              iconStart="package"
            >
              Add source
            </Button>
          )
        }
        footer={
          <Mono tone="quiet">no token, key or secret is rendered on this surface in any state</Mono>
        }
      >
        <Table
          caption={`Configured sources at cutoff ${CUTOFF}`}
          columns={sourceColumns}
          rows={SOURCE_INVENTORY}
          getRowKey={(row) => `${row.kind}${row.url}`}
          density="compact"
        />
      </Panel>

      <Panel
        title="Associations"
        eyebrow="Evidence and correction"
        meta="a link between a source and a repository is proposed by evidence and confirmed by a member"
        actions={
          readOnly ? null : (
            <Button variant="secondary" size="sm" iconStart="history">
              Re-derive links
            </Button>
          )
        }
        footer={
          <Provenance
            cutoff={CUTOFF}
            version="association v2"
            extra="a discussion source belongs to exactly one repository"
          />
        }
      >
        <Stack gap="var(--space-15)">
          {state === 'associations' || state === 'concurrent' ? (
            <Banner
              tone="info"
              title="Confidence decides how a link is presented, not whether it is stored"
            >
              At or above 0.90 a link is applied automatically and labelled Automatic. Below that it
              is held as Uncertain and waits for a member. Every correction records who made it and
              when, and the one-repository constraint is re-checked before it is accepted.
            </Banner>
          ) : null}
          <Table
            caption={`Source-to-repository associations at cutoff ${CUTOFF}`}
            columns={associationColumns}
            rows={ASSOCIATIONS}
            getRowKey={(row) => row.id}
            density="compact"
            footer={
              <Pagination page={1} pageSize={3} total={3} hasMore={false} label="associations" />
            }
          />
          {ASSOCIATIONS.filter((association) => association.note).map((association) => (
            <div
              key={association.id}
              style={{
                padding: 'var(--space-15)',
                background: 'var(--surface-sunken)',
                borderRadius: 'var(--radius-xs)',
                display: 'grid',
                gap: 4,
              }}
            >
              <Mono>
                {association.source} → {association.target}
              </Mono>
              <span style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
                {association.note} Corrected by {association.correctedBy} at{' '}
                {association.correctedAt}.
              </span>
            </div>
          ))}
        </Stack>
      </Panel>

      <Dialog
        open={Boolean(replace)}
        title={
          replace?.concurrent
            ? 'Another member corrected this link first'
            : 'Replace the primary repository'
        }
        tone={replace?.concurrent ? 'danger' : 'default'}
        size="md"
        onClose={() => setReplace(null)}
        footer={
          replace?.concurrent ? (
            <>
              <Button variant="secondary" onClick={() => setReplace(null)}>
                Discard my correction
              </Button>
              <Button variant="primary" onClick={() => setReplace(null)}>
                Reload and correct again
              </Button>
            </>
          ) : (
            <>
              <Button variant="secondary" onClick={() => setReplace(null)}>
                Cancel
              </Button>
              <Button
                variant="primary"
                onClick={() => {
                  if (replace?.id) setPrimary(replace.id);
                  setReplace(null);
                }}
              >
                Replace primary
              </Button>
            </>
          )
        }
      >
        {replace?.concurrent ? (
          <Stack gap="var(--space-15)">
            <p style={{ font: 'var(--type-body)', color: 'var(--text-body)', margin: 0 }}>
              Rafael Costa reassigned <Mono>{replace.url}</Mono> at 2026-08-20T14:34:51Z, after this
              page loaded. Your correction was not applied and nothing was overwritten.
            </p>
            <DefinitionList
              items={[
                {
                  label: 'Their correction',
                  value: '→ github.com/temporalio/temporal',
                  mono: true,
                },
                { label: 'Your correction', value: '→ github.com/temporalio/sdk-go', mono: true },
                {
                  label: 'Constraint checked',
                  value: 'a discussion source belongs to exactly one repository',
                },
              ]}
            />
          </Stack>
        ) : (
          <Stack gap="var(--space-15)">
            <p style={{ font: 'var(--type-body)', color: 'var(--text-body)', margin: 0 }}>
              <Mono>{replace?.url ?? ''}</Mono> becomes the primary repository. The current primary
              stays configured with the role you choose for it.
            </p>
            <RadioGroup
              name="demoted-role"
              legend="Role for the current primary"
              value="sdk"
              options={[
                { value: 'sdk', label: 'sdk' },
                { value: 'documentation', label: 'documentation' },
                {
                  value: 'remove',
                  label: 'Remove from this project',
                  description: 'Its collected evidence is retained but no longer refreshed.',
                },
              ]}
            />
            <Banner tone="attention" title="Derived results are recalculated, not discarded">
              Metrics computed from the old primary keep their own cutoff and version. New values
              appear after the next recalculation; nothing is rewritten in place.
            </Banner>
          </Stack>
        )}
      </Dialog>
    </Stack>
  );
}
