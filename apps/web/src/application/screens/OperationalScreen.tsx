import { useState } from 'react';
import { useOutletContext } from 'react-router';

import {
  Banner,
  Button,
  DefinitionList,
  FormField,
  Panel,
  Select,
  StatusBadge,
  TextArea,
  TextField,
} from '../../design-system';
import {
  confirmAssistantAction,
  fetchExport,
  proposeAssistantAction,
  requestExport,
  type AssistantProposalDocument,
  type ExportDocument,
} from '../api';
import type { ApplicationContext } from '../router';

export function AssistantOperationalScreen() {
  const { locale, session } = useOutletContext<ApplicationContext>();
  const [message, setMessage] = useState('');
  const [proposal, setProposal] = useState<AssistantProposalDocument>();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const portuguese = locale === 'pt-BR';

  async function propose() {
    setBusy(true);
    setError('');
    try {
      setProposal(await proposeAssistantAction(session, message));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy(false);
    }
  }

  async function confirm() {
    if (!proposal?.confirmation_token) return;
    setBusy(true);
    setError('');
    try {
      setProposal(await confirmAssistantAction(session, proposal.id, proposal.confirmation_token));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section aria-labelledby="assistant-title" style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Panel
        title={portuguese ? 'Assistente de análise' : 'Analysis assistant'}
        eyebrow="S19 · bounded HITL"
        meta={
          portuguese
            ? 'Uma ação não destrutiva, tipada e confirmada por vez.'
            : 'One typed, non-destructive, explicitly confirmed action at a time.'
        }
      >
        <h1 id="assistant-title" className="visually-hidden">
          {portuguese ? 'Assistente de análise' : 'Analysis assistant'}
        </h1>
        <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
          <Banner tone="neutral" title={portuguese ? 'Limites de segurança' : 'Safety boundary'}>
            {portuguese
              ? 'O assistente não pode alterar membros, credenciais, políticas, arquivamento ou exclusão. A identidade, a versão do recurso e a cota são verificadas novamente na execução.'
              : 'The assistant cannot change members, credentials, policies, archives, or deletion. Identity, resource version, and quota are checked again at execution.'}
          </Banner>
          <FormField
            id="assistant-message"
            label={portuguese ? 'Pergunta ou ação' : 'Question or action'}
          >
            <TextArea
              id="assistant-message"
              label={portuguese ? 'Pergunta ou ação' : 'Question or action'}
              rows={4}
              value={message}
              onChange={(event) => setMessage(event.target.value)}
              placeholder={
                portuguese
                  ? 'Adicione o repositório público de SDK ao projeto.'
                  : 'Add the public SDK repository to the project.'
              }
            />
          </FormField>
          <div>
            <Button
              variant="primary"
              disabled={busy || message.trim() === ''}
              onClick={() => void propose()}
            >
              {busy
                ? portuguese
                  ? 'Analisando…'
                  : 'Analyzing…'
                : portuguese
                  ? 'Analisar'
                  : 'Analyze'}
            </Button>
          </div>
          <div aria-live="polite">
            {error ? (
              <Banner
                tone="critical"
                title={portuguese ? 'A ação não está disponível' : 'Action unavailable'}
              >
                {error}
              </Banner>
            ) : null}
            {proposal ? (
              <Panel
                title={
                  proposal.status === 'executed'
                    ? portuguese
                      ? 'Recibo da execução'
                      : 'Execution receipt'
                    : portuguese
                      ? 'Confirme a ação exata'
                      : 'Confirm the exact action'
                }
                tone={proposal.status === 'executed' ? 'default' : 'attention'}
                actions={
                  <StatusBadge
                    status={proposal.status === 'executed' ? 'ready' : 'ai'}
                    label={proposal.status}
                  />
                }
              >
                <DefinitionList
                  columns={1}
                  dense
                  items={[
                    { label: portuguese ? 'Ação' : 'Action', value: proposal.action, mono: true },
                    {
                      label: portuguese ? 'Entradas' : 'Inputs',
                      value: JSON.stringify(proposal.inputs ?? {}),
                      mono: true,
                    },
                    {
                      label: portuguese ? 'Recursos' : 'Resources',
                      value: (proposal.resources ?? []).join(', ') || '—',
                    },
                    { label: portuguese ? 'Efeito' : 'Effect', value: proposal.effect ?? '—' },
                    {
                      label: portuguese ? 'Cota' : 'Quota',
                      value: `${proposal.quota?.cost ?? 0} · ${proposal.quota?.remaining ?? 0} remaining`,
                    },
                    {
                      label: portuguese ? 'Expira' : 'Expires',
                      value: proposal.expires_at ?? '—',
                      mono: true,
                    },
                    {
                      label: portuguese ? 'Evento de auditoria' : 'Audit event',
                      value: proposal.result?.audit_event_id ?? '—',
                      mono: true,
                    },
                  ]}
                />
                {proposal.status === 'awaiting_confirmation' ? (
                  <Button variant="primary" disabled={busy} onClick={() => void confirm()}>
                    {portuguese ? 'Confirmar uma vez' : 'Confirm once'}
                  </Button>
                ) : null}
              </Panel>
            ) : null}
          </div>
        </div>
      </Panel>
    </section>
  );
}

export function ExportsOperationalScreen() {
  const { locale, session } = useOutletContext<ApplicationContext>();
  const portuguese = locale === 'pt-BR';
  const [projectID, setProjectID] = useState('');
  const [resource, setResource] = useState('metrics');
  const [format, setFormat] = useState('csv');
  const [value, setValue] = useState<ExportDocument>();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  async function create() {
    setBusy(true);
    setError('');
    const cutoff = new Date();
    const windowTo = new Date(cutoff);
    const windowFrom = new Date(cutoff);
    windowFrom.setUTCDate(windowFrom.getUTCDate() - 90);
    try {
      setValue(
        await requestExport(session, {
          project_ids: [Number(projectID)],
          resource,
          format,
          locale,
          filters: {},
          window_from: windowFrom.toISOString(),
          window_to: windowTo.toISOString(),
          cutoff: cutoff.toISOString(),
        }),
      );
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy(false);
    }
  }

  async function refresh() {
    if (!value) return;
    setBusy(true);
    try {
      setValue(await fetchExport(value.id));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section aria-labelledby="exports-title" style={{ display: 'grid', gap: 'var(--space-2)' }}>
      <Panel
        title={portuguese ? 'Exportações' : 'Exports'}
        eyebrow="S22 · durable evidence"
        meta={
          portuguese
            ? 'Um corte consistente, artefato verificado e validade de 24 horas.'
            : 'One consistent cutoff, checksummed artifact, and 24-hour availability.'
        }
      >
        <h1 id="exports-title" className="visually-hidden">
          {portuguese ? 'Exportações' : 'Exports'}
        </h1>
        <div style={{ display: 'grid', gap: 'var(--space-2)' }}>
          <FormField id="export-project" label={portuguese ? 'ID do projeto' : 'Project ID'}>
            <TextField
              id="export-project"
              inputMode="numeric"
              value={projectID}
              onChange={(event) => setProjectID(event.target.value)}
            />
          </FormField>
          <FormField id="export-resource" label={portuguese ? 'Recurso' : 'Resource'}>
            <Select
              id="export-resource"
              value={resource}
              onChange={(event) => setResource(event.target.value)}
              options={[
                { value: 'metrics', label: portuguese ? 'Métricas' : 'Metrics' },
                { value: 'snapshots', label: portuguese ? 'Snapshots' : 'Snapshots' },
                { value: 'comparisons', label: portuguese ? 'Comparações' : 'Comparisons' },
              ]}
            />
          </FormField>
          <FormField id="export-format" label={portuguese ? 'Formato' : 'Format'}>
            <Select
              id="export-format"
              value={format}
              onChange={(event) => setFormat(event.target.value)}
              options={[
                { value: 'csv', label: 'CSV' },
                { value: 'evidence_json', label: 'Evidence JSON' },
              ]}
            />
          </FormField>
          <Button
            variant="primary"
            disabled={busy || !/^\d+$/.test(projectID)}
            onClick={() => void create()}
          >
            {busy
              ? portuguese
                ? 'Processando…'
                : 'Working…'
              : portuguese
                ? 'Solicitar exportação'
                : 'Request export'}
          </Button>
          <div aria-live="polite">
            {error ? (
              <Banner tone="critical" title={portuguese ? 'Falha na exportação' : 'Export failed'}>
                {error}
              </Banner>
            ) : null}
            {value ? (
              <Panel
                title={portuguese ? 'Estado do artefato' : 'Artifact state'}
                actions={
                  <StatusBadge
                    status={
                      value.state === 'succeeded'
                        ? 'ready'
                        : value.state === 'failed'
                          ? 'failed'
                          : 'running'
                    }
                    label={value.state}
                  />
                }
              >
                <DefinitionList
                  columns={1}
                  dense
                  items={[
                    { label: 'Export ID', value: value.id, mono: true },
                    { label: 'Job ID', value: value.job_id, mono: true },
                    {
                      label: portuguese ? 'Linhas' : 'Rows',
                      value: String(value.row_count ?? 0),
                      mono: true,
                    },
                    { label: 'SHA-256', value: value.sha256 ?? '—', mono: true },
                    {
                      label: portuguese ? 'Tamanho' : 'Size',
                      value: `${value.size_bytes ?? 0} bytes`,
                      mono: true,
                    },
                    {
                      label: portuguese ? 'Expira' : 'Expires',
                      value: value.expires_at ?? '—',
                      mono: true,
                    },
                  ]}
                />
                <div style={{ display: 'flex', gap: 'var(--space-1)', flexWrap: 'wrap' }}>
                  <Button variant="secondary" disabled={busy} onClick={() => void refresh()}>
                    {portuguese ? 'Atualizar' : 'Refresh'}
                  </Button>
                  {value.state === 'succeeded' && value.download_url ? (
                    <Button variant="primary" href={value.download_url}>
                      {portuguese ? 'Baixar' : 'Download'}
                    </Button>
                  ) : null}
                </div>
              </Panel>
            ) : null}
          </div>
        </div>
      </Panel>
    </section>
  );
}
