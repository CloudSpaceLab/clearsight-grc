import type { NotificationDeliveryHealth } from "../operationsTypes";
import { EmptyState, Notice, StatusBadge } from "./ui";

type Props = {
  health: NotificationDeliveryHealth | null;
  state: "loading" | "live" | "unavailable";
};

function classLabel(value: string) {
  if (value === "ATTENTION_CRITICAL") return "Critical email";
  if (value === "DAILY_DIGEST") return "Daily digest";
  return value.replaceAll("_", " ").toLowerCase().replace(/^./, (letter) => letter.toUpperCase());
}

export function NotificationDeliveryHealthCard({ health, state }: Props) {
  if (state === "loading" && !health) return <section className="configure-card" aria-labelledby="notification-delivery-heading"><div><h3 id="notification-delivery-heading">Notification delivery</h3><p>Loading delivery status…</p></div></section>;
  if (state === "unavailable" || !health) return <section className="configure-card" aria-labelledby="notification-delivery-heading"><div><h3 id="notification-delivery-heading">Notification delivery</h3><p>Email delivery status is unavailable.</p></div></section>;

  const needsAttention = health.failed + health.outcome_unknown + health.contact_unavailable;
  return <section className="configure-card" aria-labelledby="notification-delivery-heading">
    <div className="configure-card-heading">
      <div>
        <h3 id="notification-delivery-heading">Notification delivery</h3>
        <p>Last 24 hours. Delivery state only; message content and recipients are not shown.</p>
      </div>
      <StatusBadge tone={needsAttention > 0 ? "warning" : health.retrying > 0 ? "info" : "success"}>
        {needsAttention > 0 ? `${needsAttention} need attention` : health.retrying > 0 ? `${health.retrying} retrying` : "Healthy"}
      </StatusBadge>
    </div>

    <div className="configure-record-list">
      {health.classes.map((item) => <article key={item.delivery_class}>
        <div>
          <strong>{classLabel(item.delivery_class)}</strong>
          <span>{item.delivered} delivered · {item.retrying} retrying · {item.failed} failed · {item.outcome_unknown} unknown</span>
        </div>
      </article>)}
    </div>

    {health.classes.length === 0 && <EmptyState population="Notification delivery in the last 24 hours" title="No delivery activity" description="No governed email delivery was attempted in this window."/>}
    {health.failures.length > 0 && <Notice tone="warning">{health.failures.reduce((sum, item) => sum + item.count, 0)} delivery failures or exceptions are recorded in this window.</Notice>}
  </section>;
}
