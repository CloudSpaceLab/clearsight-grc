import { useEffect, useState } from "react";
import { getLoss } from "../../lossApi";
import type { LossAggregate } from "../../lossTypes";
import { Button, Notice, Surface } from "../ui";
import { formatLossDate, formatLossMoney, recoveryStatusLabel } from "../losses/lossPresentation";

type Props = {
  matterID: string;
  lossID: string;
  onOpenLoss?: (lossID: string) => void;
  loadLoss?: typeof getLoss;
};

type LoadState = "loading" | "live" | "unavailable";

export function OperationalLossMatterContext({
  matterID,
  lossID,
  onOpenLoss,
  loadLoss = getLoss,
}: Props) {
  const [state, setState] = useState<LoadState>("loading");
  const [aggregate, setAggregate] = useState<LossAggregate>();

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    setAggregate(undefined);
    void loadLoss(lossID, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      if (value.loss.id !== lossID || (value.loss.matter_id && value.loss.matter_id !== matterID)) {
        setState("unavailable");
        return;
      }
      setAggregate(value);
      setState("live");
    }).catch(() => {
      if (controller.signal.aborted) return;
      setState("unavailable");
    });
    return () => controller.abort();
  }, [loadLoss, lossID, matterID]);

  if (state === "loading") return <p className="matter-domain-context-state" role="status">Loading linked loss…</p>;
  if (state === "unavailable" || !aggregate) {
    return <Notice tone="warning">Linked loss record is unavailable. Issue work remains available.</Notice>;
  }

  const { loss, totals } = aggregate;
  return <Surface>
    <section className="matter-domain-context" aria-label="Operational loss context">
      <div className="matter-record-section-heading">
        <div>
          <span className="eyebrow">Operational loss</span>
          <h2>{loss.code} · {loss.title}</h2>
        </div>
        {onOpenLoss && <Button variant="secondary" size="compact" onPress={() => onOpenLoss(loss.id)}>Open loss record</Button>}
      </div>
      <dl className="matter-record-facts">
        <div><dt>Gross loss</dt><dd>{formatLossMoney(totals.gross_amount_minor, totals.currency)}</dd></div>
        <div><dt>Recovered</dt><dd>{formatLossMoney(totals.recovered_amount_minor, totals.currency)}</dd></div>
        <div><dt>Net loss</dt><dd>{formatLossMoney(totals.net_loss_minor, totals.currency)}</dd></div>
        <div><dt>Recovery</dt><dd>{recoveryStatusLabel(totals.recovery_status)}</dd></div>
        <div><dt>Occurred</dt><dd>{formatLossDate(loss.occurred_at)}</dd></div>
      </dl>
    </section>
  </Surface>;
}
