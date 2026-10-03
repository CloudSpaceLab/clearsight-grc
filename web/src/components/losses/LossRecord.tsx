import { useEffect, useState } from "react";
import { getLoss, openLossIntervention } from "../../lossApi";
import type { LossAggregate, LossRecovery } from "../../lossTypes";
import { apiErrorKind } from "../../http";
import { Button, DataTable, EmptyState, Notice, StatusBadge, Surface, type DataColumn } from "../ui";
import { formatLossDate, formatLossMoney, lossEventLabel, lossStatusLabel, lossStatusTone, recoveryStatusLabel, recoveryStatusTone } from "./lossPresentation";

type Props = {
  lossID: string;
  onBack: () => void;
  onOpenRisk?: (riskID: string) => void;
  onOpenMatter?: (matterID: string) => void;
  loadLoss?: (id: string, signal?: AbortSignal) => Promise<LossAggregate>;
  openIntervention?: typeof openLossIntervention;
};

type LoadState = "loading" | "live" | "not-found" | "error";

export function LossRecord({
  lossID,
  onBack,
  onOpenRisk,
  onOpenMatter,
  loadLoss = getLoss,
  openIntervention = openLossIntervention,
}: Props) {
  const [value, setValue] = useState<LossAggregate>();
  const [state, setState] = useState<LoadState>("loading");
  const [retry, setRetry] = useState(0);
  const [interventionBusy, setInterventionBusy] = useState(false);
  const [interventionError, setInterventionError] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    setValue(undefined);
    void loadLoss(lossID, controller.signal).then((loaded) => {
      if (controller.signal.aborted) return;
      setValue(loaded);
      setState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setState(apiErrorKind(error) === "not_found" ? "not-found" : "error");
    });
    return () => controller.abort();
  }, [loadLoss, lossID, retry]);

  if (state === "loading" && !value) return <section className="loss-record-page"><p className="loss-load-state" role="status">Loading loss…</p></section>;
  if (state === "not-found") return <section className="loss-record-page"><EmptyState population="Current legal-entity loss register" title="Loss not found" description="Return to the loss register." action={<Button variant="secondary" onPress={onBack}>Back to losses</Button>}/></section>;
  if (state === "error" || !value) return <section className="loss-record-page"><EmptyState population="Selected operational loss" title="Loss unavailable" description="The loss could not be loaded." action={<Button variant="secondary" onPress={() => setRetry((current) => current + 1)}>Try again</Button>} role="alert"/></section>;

  const aggregate = value;
  const loss = aggregate.loss;
  const totals = aggregate.totals;

  const recoveryColumns: readonly DataColumn<LossRecovery>[] = [
    {
      id: "kind",
      header: "Entry",
      kind: "status",
      render: (item) => <StatusBadge tone={item.kind === "RECOVERY" ? "success" : "warning"}>{item.kind === "RECOVERY" ? "Recovery" : "Reversal"}</StatusBadge>,
      accessibleText: (item) => item.kind === "RECOVERY" ? "Recovery" : "Reversal",
    },
    {
      id: "amount",
      header: "Amount",
      kind: "number",
      render: (item) => formatLossMoney(item.amount_minor, item.currency),
      accessibleText: (item) => formatLossMoney(item.amount_minor, item.currency),
    },
    {
      id: "reference",
      header: "Reference",
      mobileLayout: "full-width",
      render: (item) => item.reference || "No reference",
      accessibleText: (item) => item.reference || "No reference",
    },
    {
      id: "date",
      header: "Recovered",
      render: (item) => formatLossDate(item.recovered_at),
      accessibleText: (item) => formatLossDate(item.recovered_at),
    },
  ];

  async function handleIntervention() {
    if (loss.matter_id) {
      onOpenMatter?.(loss.matter_id);
      return;
    }
    if (interventionBusy) return;
    setInterventionBusy(true);
    setInterventionError("");
    try {
      const response = await openIntervention(loss.id, loss.version);
      setValue((current) => current ? { ...current, loss: response.loss } : current);
      onOpenMatter?.(response.matter.id);
    } catch (error) {
      setInterventionError(error instanceof Error ? error.message : "Intervention could not be opened.");
    } finally {
      setInterventionBusy(false);
    }
  }

  return <section className="loss-record-page" aria-labelledby="loss-record-heading">
    <header className="topbar loss-page-header">
      <div>
        <span className="eyebrow">{loss.code} · {lossEventLabel(loss.event_type)}</span>
        <h1 id="loss-record-heading">{loss.title}</h1>
      </div>
      <Button variant="secondary" onPress={onBack}>Back to losses</Button>
    </header>

    <Surface>
      <dl className="loss-record__state" role="group" aria-label="Current loss state">
        <div><dt>Net loss</dt><dd><strong>{formatLossMoney(totals.net_loss_minor, totals.currency)}</strong></dd></div>
        <div><dt>Recovered</dt><dd>{formatLossMoney(totals.recovered_amount_minor, totals.currency)}</dd></div>
        <div><dt>Recovery</dt><dd><StatusBadge tone={recoveryStatusTone(totals.recovery_status)}>{recoveryStatusLabel(totals.recovery_status)}</StatusBadge></dd></div>
        <div><dt>Status</dt><dd><StatusBadge tone={lossStatusTone(loss.status)}>{lossStatusLabel(loss.status)}</StatusBadge></dd></div>
      </dl>
    </Surface>

    <div className="loss-record__overview">
      <Surface>
        <section className="loss-record__section">
          <h2>Event</h2>
          <dl className="loss-record__facts">
            <div><dt>Cause</dt><dd>{loss.cause}</dd></div>
            {loss.description && <div><dt>Description</dt><dd>{loss.description}</dd></div>}
            <div><dt>Occurred</dt><dd>{formatLossDate(loss.occurred_at)}</dd></div>
            <div><dt>Discovered</dt><dd>{formatLossDate(loss.discovered_at)}</dd></div>
            <div><dt>Gross loss</dt><dd>{formatLossMoney(totals.gross_amount_minor, totals.currency)}</dd></div>
          </dl>
        </section>
      </Surface>
      <Surface>
        <section className="loss-record__section">
          <h2>Links</h2>
          <dl className="loss-record__facts">
            <div><dt>Owner</dt><dd>{loss.owner_principal_id ? "Assigned" : "Not assigned"}</dd></div>
            <div><dt>Risk</dt><dd>{loss.risk_id ? "Linked" : "Not linked"}</dd></div>
            <div><dt>Intervention</dt><dd>{loss.matter_id ? "Open record linked" : "Not opened"}</dd></div>
          </dl>
          <div className="loss-record__actions">
            {loss.risk_id && onOpenRisk && <Button variant="secondary" size="compact" onPress={() => onOpenRisk(loss.risk_id!)}>Open linked risk</Button>}
            {onOpenMatter && <Button size="compact" isDisabled={interventionBusy} onPress={() => void handleIntervention()}>{interventionBusy ? "Opening…" : "Open intervention"}</Button>}
          </div>
          {interventionError && <Notice tone="error">{interventionError}</Notice>}
        </section>
      </Surface>
    </div>

    <section className="loss-record__history" aria-labelledby="loss-recoveries-heading">
      <div className="section-header">
        <div>
          <h2 id="loss-recoveries-heading">Recoveries</h2>
          <p>Append-only recoveries and reversals. Net loss is derived from this ledger.</p>
        </div>
      </div>
      {aggregate.recoveries.length ? <DataTable
        ariaLabel="Loss recoveries"
        rows={aggregate.recoveries}
        rowKey={(item) => item.id}
        rowName={(item) => `${item.kind === "RECOVERY" ? "Recovery" : "Reversal"}, ${formatLossMoney(item.amount_minor, item.currency)}, ${formatLossDate(item.recovered_at)}`}
        columns={recoveryColumns}
      /> : <EmptyState population={loss.title} title="No recoveries" description="No recovery has been recorded for this loss."/>}
    </section>
  </section>;
}

function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}
