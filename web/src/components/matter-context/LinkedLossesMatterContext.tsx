import { useEffect, useState } from "react";
import { listLosses } from "../../lossApi";
import type { LossPage } from "../../lossTypes";
import { Button, Notice, Surface } from "../ui";
import { formatLossMoney, recoveryStatusLabel } from "../losses/lossPresentation";

type Props = {
  matterID: string;
  onOpenLoss?: (lossID: string) => void;
  loadPage?: typeof listLosses;
};

type LoadState = "loading" | "live" | "unavailable";

export function LinkedLossesMatterContext({ matterID, onOpenLoss, loadPage = listLosses }: Props) {
  const [state, setState] = useState<LoadState>("loading");
  const [page, setPage] = useState<LossPage>();

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    setPage(undefined);
    void loadPage({ matterID, limit: 10 }, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setPage(value);
      setState("live");
    }).catch(() => {
      if (!controller.signal.aborted) setState("unavailable");
    });
    return () => controller.abort();
  }, [loadPage, matterID]);

  if (state === "loading") return <p className="matter-domain-context-state" role="status">Loading linked losses…</p>;
  if (state === "unavailable") return <Notice tone="warning">Linked loss records are unavailable. Issue work remains available.</Notice>;
  if (!page?.items.length) return null;

  return <Surface>
    <section className="matter-domain-context" aria-label="Linked losses">
      <div className="matter-record-section-heading">
        <div>
          <span className="eyebrow">Loss and recovery</span>
          <h2>{page.items.length} linked loss record{page.items.length === 1 ? "" : "s"}</h2>
        </div>
      </div>
      <div className="matter-linked-loss-list">
        {page.items.map(({ loss, totals }) => <article key={loss.id} className="matter-linked-loss">
          <div>
            <strong>{loss.code} · {loss.title}</strong>
            <span>Net {formatLossMoney(totals.net_loss_minor, totals.currency)} · Recovered {formatLossMoney(totals.recovered_amount_minor, totals.currency)} · {recoveryStatusLabel(totals.recovery_status)}</span>
          </div>
          {onOpenLoss && <Button variant="secondary" size="compact" onPress={() => onOpenLoss(loss.id)}>Open loss {loss.code}</Button>}
        </article>)}
      </div>
    </section>
  </Surface>;
}
