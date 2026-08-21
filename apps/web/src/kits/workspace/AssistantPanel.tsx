import { useState } from 'react';

import {
  Banner,
  Button,
  DefinitionList,
  EvidenceLink,
  IconButton,
  RunMetadata,
  StatusBadge,
  TextArea,
} from '../../design-system';

/**
 * Natural-language analysis plus one bounded human-in-the-loop proposal. The assistant can propose
 * exactly one non-destructive Analyst action, with its inputs, effect, quota cost and expiry shown
 * before approval; it can never touch users, roles, credentials, policies, archives or deletion.
 */
export function AssistantPanel({ onClose }: { readonly onClose: () => void }) {
  const [step, setStep] = useState<'answer' | 'proposal' | 'receipt'>('answer');

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <header
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 'var(--space-1)',
          padding: 'var(--space-15) var(--space-2)',
          borderBottom: '1px solid var(--border-subtle)',
        }}
      >
        <StatusBadge status="ai" size="sm" />
        <h2 style={{ font: 'var(--type-subsection)', flex: 1 }}>Analysis</h2>
        <IconButton icon="x" label="Close assistant" size="sm" onClick={onClose} />
      </header>
      <div
        style={{
          flex: 1,
          overflow: 'auto',
          padding: 'var(--space-2)',
          display: 'grid',
          gap: 'var(--space-2)',
          alignContent: 'start',
        }}
      >
        <div style={{ display: 'grid', gap: 'var(--space-05)' }}>
          <p style={{ font: 'var(--type-table-head)', color: 'var(--text-secondary)' }}>Question</p>
          <p style={{ font: 'var(--type-body)', fontSize: 'var(--text-sm)' }}>
            What changed in maintenance risk for Temporal and Cadence?
          </p>
          <p style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>
            scope 2 projects · window 90d · cutoff 2026-08-20T14:35:00Z · language en
          </p>
        </div>
        <RunMetadata
          runId="732684513904852992"
          versionLabel="v1 of 1"
          selected
          provider="local-ollama"
          model="qwen2.5:32b"
          promptVersion="analysis@12"
          language="en"
          executedAt="2026-08-20T14:36:02Z"
          usage="4.8k tokens"
        />
        <div style={{ display: 'grid', gap: 'var(--space-1)' }}>
          <p style={{ font: 'var(--type-body)', fontSize: 'var(--text-sm)' }}>
            Maintenance risk rose for both projects over the window. Temporal&apos;s median
            pull-request merge time increased from 19.4 to 31.2 hours while its top-three author
            share moved from 0.65 to 0.71. Cadence published no release in the window and its
            maintainer count fell from 3 to 1.
          </p>
          <p style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
            Coverage: Temporal 90d of 90d; Cadence 90d of 90d, advisories unavailable. Discussion
            topics for Temporal are stale since 2026-08-14 and were excluded.
          </p>
          <EvidenceLink
            kind="metric"
            title="median_pr_merge_time · Temporal · v3"
            source="metric evidence · 90d"
            collectedAt="2026-08-20T14:35Z"
            href="#"
            external={false}
          />
          <EvidenceLink
            kind="metric"
            title="maintainer_count · Cadence · v2"
            source="metric evidence · 90d"
            collectedAt="2026-08-20T14:35Z"
            href="#"
            external={false}
          />
        </div>
        {step === 'answer' ? (
          <Button variant="secondary" iconStart="wand-sparkles" onClick={() => setStep('proposal')}>
            Propose a follow-up action
          </Button>
        ) : null}
        {step === 'proposal' ? (
          <div
            style={{
              border: '1px solid var(--ai-border)',
              background: 'var(--ai-bg)',
              borderRadius: 'var(--radius-sm)',
              padding: 'var(--space-15)',
              display: 'grid',
              gap: 'var(--space-1)',
            }}
          >
            <div style={{ display: 'flex', gap: 'var(--space-1)', alignItems: 'center' }}>
              <StatusBadge status="ai" label="Proposed action" size="sm" />
              <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-secondary)' }}>
                expires in 4m 52s
              </span>
            </div>
            <DefinitionList
              dense
              columns={1}
              items={[
                { label: 'Action', value: 'create_alert_rule', mono: true },
                { label: 'Resource', value: 'workspace · shared rule' },
                {
                  label: 'Inputs',
                  value:
                    'metric top_three_author_share, operator >=, threshold 0.60, window 90d, severity attention',
                  mono: true,
                },
                {
                  label: 'Expected effect',
                  value:
                    'One shared rule evaluated after each recalculation. No existing occurrence changes.',
                },
                { label: 'Quota', value: '1 of 50 workspace rules' },
              ]}
            />
            <Banner tone="neutral" title="Approval is single-use">
              Permissions are re-checked at execution. Approving does not authorise any further
              action, and the assistant cannot manage members, roles, credentials, policy
              definitions, archives or deletion.
            </Banner>
            <div style={{ display: 'flex', gap: 'var(--space-1)', flexWrap: 'wrap' }}>
              <Button variant="primary" size="md" onClick={() => setStep('receipt')}>
                Approve this action
              </Button>
              <Button variant="ghost" size="md" onClick={() => setStep('answer')}>
                Discard
              </Button>
            </div>
          </div>
        ) : null}
        {step === 'receipt' ? (
          <Banner
            tone="positive"
            title="Alert rule created"
            actions={
              <Button size="sm" variant="secondary" iconEnd="arrow-right">
                Open rule
              </Button>
            }
          >
            Rule 732684513922121728 · created 2026-08-20T14:38:11Z by Ana Silva through assistant
            proposal 732684513917927424. Recorded in the audit log.
          </Banner>
        ) : null}
      </div>
      <div
        style={{
          padding: 'var(--space-15) var(--space-2)',
          borderTop: '1px solid var(--border-subtle)',
          display: 'grid',
          gap: 'var(--space-1)',
        }}
      >
        <TextArea
          id="assistant-ask"
          label="Ask about your projects"
          rows={2}
          placeholder="Which projects lost maintainers this quarter?"
        />
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            gap: 'var(--space-1)',
            alignItems: 'center',
          }}
        >
          <span style={{ font: 'var(--type-mono-xs)', color: 'var(--text-tertiary)' }}>
            scope: your authorised projects
          </span>
          <Button variant="primary" size="md" iconStart="send">
            Ask
          </Button>
        </div>
      </div>
    </div>
  );
}
