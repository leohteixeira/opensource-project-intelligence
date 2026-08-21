import { useState } from 'react';

import {
  Banner,
  Button,
  DefinitionList,
  Dialog,
  FilterBar,
  Menu,
  Panel,
  RadioGroup,
  Select,
  StatusBadge,
  Table,
  TextField,
  type TableColumn,
} from '../../design-system';
import {
  APPLICANTS,
  MEMBERS,
  SERVICE_ACCOUNTS,
  type Applicant,
  type Member,
  type ServiceAccount,
} from './fixtures';

const ROLES = [
  {
    value: 'viewer',
    label: 'Viewer',
    description: 'Reads intelligence and exports. Cannot change shared state.',
  },
  {
    value: 'analyst',
    label: 'Analyst',
    description: 'Curates projects, requests collection, corrects associations, governs the radar.',
  },
  {
    value: 'admin',
    label: 'Admin',
    description: 'Governs membership, policies and lifecycle. Only Admins delete projects.',
  },
];

function ApproveDialog({
  applicant,
  onClose,
}: {
  readonly applicant: Applicant;
  readonly onClose: () => void;
}) {
  const [role, setRole] = useState('viewer');

  return (
    <Dialog
      title={`Approve ${applicant.name}`}
      onClose={onClose}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="secondary" onClick={onClose}>
            Reject
          </Button>
          <Button variant="primary" onClick={onClose}>
            Approve as {role}
          </Button>
        </>
      }
    >
      <DefinitionList
        columns={2}
        dense
        items={[
          { label: 'External subject', value: applicant.subject, mono: true },
          { label: 'Email', value: applicant.email },
          { label: 'Requested', value: applicant.requested, mono: true },
          { label: 'Current access', value: 'applicant' },
        ]}
      />
      <RadioGroup
        name="role"
        legend="Local role"
        value={role}
        onChange={(event) => setRole(event.target.value)}
        options={ROLES}
      />
      <Banner tone="neutral" title="One fixed role per member">
        Roles are not additive and cannot be composed. Approval, role change and suspension are
        audited with the acting Admin, the prior state and the new state.
      </Banner>
    </Dialog>
  );
}

export function MembersScreen() {
  const [applicant, setApplicant] = useState<Applicant | null>(null);

  const applicantColumns: readonly TableColumn<Applicant>[] = [
    { key: 'name', header: 'Applicant' },
    { key: 'email', header: 'Email', mono: true },
    { key: 'subject', header: 'External subject', mono: true },
    { key: 'requested', header: 'Requested', mono: true },
    {
      key: 'actions',
      header: '',
      render: (row) => (
        <Button size="sm" variant="primary" onClick={() => setApplicant(row)}>
          Review
        </Button>
      ),
    },
  ];

  const memberColumns: readonly TableColumn<Member>[] = [
    {
      key: 'name',
      header: 'Member',
      render: (row) => (
        <span style={{ display: 'grid', gap: 1 }}>
          <span style={{ font: 'var(--type-body-strong)', fontSize: 'var(--text-sm)' }}>
            {row.name}
          </span>
          <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>
            {row.email}
          </span>
        </span>
      ),
    },
    { key: 'role', header: 'Role', mono: true },
    {
      key: 'state',
      header: 'State',
      render: (row) => (
        <StatusBadge
          status={row.state === 'active' ? 'active' : 'paused'}
          label={row.state === 'active' ? 'Active' : 'Suspended'}
          size="sm"
        />
      ),
    },
    { key: 'lastSeen', header: 'Last seen', mono: true },
    { key: 'id', header: 'Member id', mono: true },
    {
      key: 'actions',
      header: '',
      render: (row) => (
        <Menu
          triggerLabel={`Actions for ${row.name}`}
          items={[
            { label: 'Change role', icon: 'user-cog' },
            {
              label: row.state === 'active' ? 'Suspend' : 'Restore',
              icon: row.state === 'active' ? 'pause' : 'play',
            },
            { separator: true },
            {
              label: 'Remove member',
              icon: 'trash-2',
              danger: true,
              disabled: row.role === 'admin',
              hint: row.role === 'admin' ? 'last admin' : undefined,
            },
          ]}
        />
      ),
    },
  ];

  const serviceColumns: readonly TableColumn<ServiceAccount>[] = [
    { key: 'name', header: 'Service account' },
    { key: 'subject', header: 'External subject', mono: true },
    { key: 'role', header: 'Local role', mono: true },
    { key: 'scopes', header: 'Scopes', mono: true, wrap: true },
    {
      key: 'state',
      header: 'State',
      render: (row) => (
        <StatusBadge
          status={row.state === 'active' ? 'active' : 'paused'}
          label={row.state === 'active' ? 'Active' : 'Suspended'}
          size="sm"
        />
      ),
    },
    { key: 'lastSeen', header: 'Last request', mono: true },
  ];

  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Panel
        title="Applicant queue"
        status={<StatusBadge status="queued" label={`${APPLICANTS.length} waiting`} size="sm" />}
        padding="0"
        meta="authentication establishes identity, not access"
      >
        <Table
          caption="Applicants awaiting approval"
          columns={applicantColumns}
          rows={APPLICANTS}
        />
      </Panel>
      <Panel
        title="Members"
        padding="0"
        actions={
          <FilterBar>
            <TextField
              id="member-search"
              type="search"
              iconStart="search"
              size="md"
              placeholder="Search members"
            />
            <Select
              id="member-role"
              size="md"
              placeholder="All roles"
              options={['viewer', 'analyst', 'admin']}
            />
          </FilterBar>
        }
      >
        <Table caption="Workspace members" columns={memberColumns} rows={MEMBERS} />
      </Panel>
      <Panel
        title="Service accounts"
        meta="externally provisioned Keycloak identities bound to a local role and scope subset"
        padding="0"
        footer={
          <span>
            No token or secret field is ever returned, even to an Admin. Bearer activity is
            attributed to the service identity, never to a person.
          </span>
        }
      >
        <Table caption="Bound service accounts" columns={serviceColumns} rows={SERVICE_ACCOUNTS} />
      </Panel>
      <Banner tone="attention" title="The last active Admin is protected">
        Suspending, demoting or deleting the last Admin is refused with 409 last_admin_required.
      </Banner>
      {applicant ? (
        <ApproveDialog applicant={applicant} onClose={() => setApplicant(null)} />
      ) : null}
    </div>
  );
}
