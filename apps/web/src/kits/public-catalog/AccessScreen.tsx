import { useState } from 'react';

import {
  Banner,
  Button,
  DefinitionList,
  Dialog,
  Panel,
  RadioGroup,
  StatusBadge,
  TextField,
} from '../../design-system';

export type AccessState = 'pending' | 'rejected' | 'suspended' | 'approved';

const STATES = [
  { value: 'pending', label: 'Pending' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'suspended', label: 'Suspended' },
  { value: 'approved', label: 'Approved' },
];

export function AccessScreen({
  state,
  onState,
  onBack,
}: {
  readonly state: AccessState;
  readonly onState: (state: AccessState) => void;
  readonly onBack: () => void;
}) {
  const [deleting, setDeleting] = useState(false);
  const [typed, setTyped] = useState('');

  return (
    <div style={{ display: 'grid', gap: 'var(--space-2)', maxWidth: 720 }}>
      <RadioGroup
        name="access-state"
        legend="Access state (kit control)"
        orientation="horizontal"
        value={state}
        onChange={(event) => onState(event.target.value as AccessState)}
        options={STATES}
      />
      {state === 'pending' ? (
        <Panel
          title="Access pending"
          status={<StatusBadge status="queued" label="Awaiting approval" size="sm" />}
          meta="requested 2026-08-20T14:02:11Z · one request per identity"
        >
          <div style={{ display: 'grid', gap: 'var(--space-15)' }}>
            <p style={{ font: 'var(--type-body)', fontSize: 'var(--text-sm)' }}>
              Your identity is confirmed. A workspace Admin decides membership and assigns one role.
              Repeating the sign-in preserves this single request and does not notify Admins again.
            </p>
            <DefinitionList
              columns={2}
              dense
              items={[
                { label: 'External subject', value: 'kc:9f2a…c41d', mono: true },
                { label: 'Email', value: 'ana.silva@example.com' },
                { label: 'Local role', value: 'none until approved' },
                { label: 'Requested', value: '2026-08-20T14:02:11Z', mono: true },
              ]}
            />
            <Banner tone="info" title="What you can do now">
              Browse the public catalog. Protected APIs, exports and deep links stay unavailable
              while your membership is pending.
            </Banner>
            <Button
              variant="secondary"
              iconStart="chevron-left"
              onClick={onBack}
              style={{ justifySelf: 'start' }}
            >
              Back to the catalog
            </Button>
          </div>
        </Panel>
      ) : null}
      {state === 'rejected' ? (
        <Panel
          title="Access not granted"
          tone="critical"
          status={<StatusBadge status="not_recommended" label="Rejected" size="sm" />}
          meta="decided 2026-08-20T15:10:04Z"
        >
          <p style={{ font: 'var(--type-body)', fontSize: 'var(--text-sm)' }}>
            A workspace Admin declined this request. A rejected identity cannot return to pending on
            its own; contact the workspace Admin if this is unexpected.
          </p>
        </Panel>
      ) : null}
      {state === 'suspended' ? (
        <Panel
          title="Membership suspended"
          tone="attention"
          status={<StatusBadge status="paused" label="Suspended" size="sm" />}
          meta="suspended 2026-08-19T08:00:00Z"
        >
          <p style={{ font: 'var(--type-body)', fontSize: 'var(--text-sm)' }}>
            Your external token may still be valid, but local suspension blocks every protected
            route, export and service-account action until an Admin restores the membership.
          </p>
        </Panel>
      ) : null}
      {state === 'approved' ? (
        <Panel title="Preferences" meta="member 732684512957038592 · analyst">
          <div style={{ display: 'grid', gap: 'var(--space-15)' }}>
            <DefinitionList
              columns={2}
              dense
              items={[
                { label: 'Display name', value: 'Ana Silva' },
                { label: 'Local role', value: 'analyst', mono: true },
                { label: 'Locale', value: 'pt-BR', mono: true },
                { label: 'Timezone', value: 'America/Sao_Paulo', mono: true },
              ]}
            />
            <Banner
              tone="neutral"
              title="Passwords, recovery and identity verification live in Keycloak"
            >
              This product owns membership, permissions, suspension and audit only.
            </Banner>
            <Button
              variant="danger"
              iconStart="trash-2"
              style={{ justifySelf: 'start' }}
              onClick={() => setDeleting(true)}
            >
              Delete my account
            </Button>
          </div>
        </Panel>
      ) : null}
      {deleting ? (
        <Dialog
          tone="danger"
          title="Delete your account"
          onClose={() => setDeleting(false)}
          footer={
            <>
              <Button variant="secondary" onClick={() => setDeleting(false)}>
                Cancel
              </Button>
              <Button variant="danger" disabled={typed !== 'DELETE MY ACCOUNT'}>
                Delete my account
              </Button>
            </>
          }
        >
          <p>
            Your membership, preferences and per-member read state are removed. Actions you
            performed remain in the audit log under an opaque actor identity. Shared workspace data
            is not deleted.
          </p>
          <TextField
            id="delete-confirm"
            mono
            label="Type DELETE MY ACCOUNT to confirm"
            value={typed}
            onChange={(event) => setTyped(event.target.value)}
          />
          <Banner tone="attention" title="The last active Admin cannot be deleted">
            Such a request is refused with 409 last_admin_required.
          </Banner>
        </Dialog>
      ) : null}
    </div>
  );
}
