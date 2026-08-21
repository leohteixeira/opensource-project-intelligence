import { useState } from 'react';

import {
  Banner,
  Button,
  DefinitionList,
  Dialog,
  FilterBar,
  Menu,
  Pagination,
  Select,
  StatusBadge,
  Table,
  TextField,
  type TableColumn,
} from '../../design-system';
import { PROJECTS, type WorkspaceProject } from './fixtures';

function RegisterDialog({
  onClose,
  onSubmit,
}: {
  readonly onClose: () => void;
  readonly onSubmit: () => void;
}) {
  const [url, setUrl] = useState('https://github.com/temporalio/sdk-typescript');
  const [preview, setPreview] = useState(false);
  const rejected = url.includes('10.0.0');

  return (
    <Dialog
      title="Register project"
      onClose={onClose}
      size="lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="primary"
            disabled={rejected}
            onClick={() => (preview ? onSubmit() : setPreview(true))}
          >
            {preview ? 'Register and start initial sync' : 'Preview identity'}
          </Button>
        </>
      }
    >
      <TextField
        id="register-url"
        label="Primary repository URL"
        mono
        required
        value={url}
        onChange={(event) => {
          setUrl(event.target.value);
          setPreview(false);
        }}
        hint="Supported hosts: GitHub, GitLab, Gitea, public Git. Public repositories only."
        error={rejected ? 'The source resolves to a private network address.' : undefined}
      />
      <TextField
        id="register-history"
        label="History target"
        suffix="days"
        defaultValue="180"
        size="md"
        hint="Default 180 days. Longer requests consume workspace history quota and remain resumable."
      />
      {preview ? (
        <div style={{ display: 'grid', gap: 'var(--space-1)' }}>
          <div style={{ display: 'flex', gap: 'var(--space-1)', alignItems: 'center' }}>
            <StatusBadge status="available" label="Inferred identity" size="sm" />
            <span style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
              Editable after registration
            </span>
          </div>
          <DefinitionList
            columns={2}
            dense
            items={[
              { label: 'Name', value: 'Temporal TypeScript SDK' },
              { label: 'Slug', value: 'temporal-typescript-sdk', mono: true },
              {
                label: 'Description',
                value: 'TypeScript SDK for the Temporal durable execution platform',
              },
              { label: 'Primary repository role', value: 'sdk', mono: true },
            ]}
          />
          <Banner tone="attention" title="Possible duplicate">
            An active project <strong>Temporal</strong> already links this organisation. Registering
            creates a separate project; use Repositories to attach this SDK to Temporal instead.
          </Banner>
        </div>
      ) : null}
    </Dialog>
  );
}

export function ProjectsScreen({
  onOpenProject,
}: {
  readonly onOpenProject: (project: WorkspaceProject) => void;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const rows = PROJECTS.filter((project) =>
    project.name.toLowerCase().includes(query.toLowerCase()),
  );

  const columns: readonly TableColumn<WorkspaceProject>[] = [
    {
      key: 'name',
      header: 'Project',
      render: (row) => (
        <span style={{ display: 'grid', gap: 1 }}>
          <span style={{ font: 'var(--type-body-strong)', fontSize: 'var(--text-sm)' }}>
            {row.name}
          </span>
          <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>
            {row.slug}
          </span>
        </span>
      ),
    },
    {
      key: 'state',
      header: 'State',
      render: (row) => (
        <StatusBadge status={row.state === 'paused' ? 'paused' : 'active'} size="sm" />
      ),
    },
    { key: 'repos', header: 'Repos', numeric: true },
    { key: 'sources', header: 'Sources', numeric: true },
    {
      key: 'recommendation',
      header: 'Recommendation',
      render: (row) => <StatusBadge status={row.recommendation} size="sm" />,
    },
    {
      key: 'overall',
      header: 'Overall',
      numeric: true,
      render: (row) =>
        row.overall === null ? (
          <StatusBadge status="insufficient_data" size="sm" glyphOnly />
        ) : (
          row.overall
        ),
    },
    {
      key: 'freshness',
      header: 'Freshness',
      render: (row) => (
        <StatusBadge
          status={
            row.freshness === 'stale'
              ? 'stale'
              : row.freshness === 'partial'
                ? 'insufficient_data'
                : 'ready'
          }
          label={row.freshness === 'fresh' ? 'Fresh' : undefined}
          size="sm"
        />
      ),
    },
    {
      key: 'actions',
      header: '',
      render: () => (
        <Menu
          triggerLabel="Project actions"
          items={[
            { label: 'Edit identity', icon: 'pencil' },
            { label: 'Request sync', icon: 'refresh-cw' },
            { separator: true },
            { label: 'Pause collection', icon: 'pause' },
            { label: 'Archive', icon: 'archive' },
            { label: 'Delete permanently', icon: 'trash-2', danger: true },
          ]}
        />
      ),
    },
  ];

  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <FilterBar
        resultLabel={`${rows.length} of ${PROJECTS.length} projects`}
        applied={[{ key: 'state', field: 'state', value: 'active or paused' }]}
      >
        <TextField
          id="project-search"
          type="search"
          iconStart="search"
          size="md"
          placeholder="Search projects"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
        <Select
          id="project-state"
          size="md"
          placeholder="All states"
          options={['active', 'paused', 'archived']}
        />
        <Select
          id="project-recommendation"
          size="md"
          placeholder="All recommendations"
          options={['recommended', 'conditional', 'not_recommended', 'insufficient_data']}
        />
        <Button variant="primary" size="md" iconStart="plus" onClick={() => setOpen(true)}>
          Register project
        </Button>
      </FilterBar>
      <Table
        caption="Projects, 90 day window ending 2026-08-20"
        columns={columns}
        rows={rows}
        onRowClick={onOpenProject}
        footer="Overall score is a secondary summary of seven independently inspectable dimensions."
      />
      <Pagination page={1} hasMore={false} total={rows.length} label="Projects" />
      {open ? (
        <RegisterDialog onClose={() => setOpen(false)} onSubmit={() => setOpen(false)} />
      ) : null}
    </div>
  );
}
