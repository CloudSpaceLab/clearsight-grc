import type { NotificationDeliveryHistoryItem, NotificationDeliveryOutcome } from "../notificationTypes";
import { DataTable, Notice, StatusBadge, type DataColumn, type StatusTone } from "./ui";

function outcomeTone(outcome: NotificationDeliveryOutcome): StatusTone {
  switch (outcome) {
    case "SUCCEEDED": return "success";
    case "FAILED": return "error";
    case "RETRYING":
    case "PENDING": return "warning";
    case "CANCELLED": return "neutral";
  }
}

function outcomeLabel(outcome: NotificationDeliveryOutcome) {
  switch (outcome) {
    case "SUCCEEDED": return "Delivered";
    case "FAILED": return "Failed";
    case "RETRYING": return "Retrying";
    case "PENDING": return "Pending";
    case "CANCELLED": return "Superseded";
  }
}

const columns: readonly DataColumn<NotificationDeliveryHistoryItem>[] = [
  {
    id: "time",
    header: "Time",
    render: (item) => <time dateTime={item.occurred_at}>{new Date(item.occurred_at).toLocaleString()}</time>,
    accessibleText: (item) => new Date(item.occurred_at).toLocaleString(),
  },
  {
    id: "notice",
    header: "Notice",
    mobileLayout: "full-width",
    render: (item) => item.action,
    accessibleText: (item) => item.action,
  },
  {
    id: "outcome",
    header: "Outcome",
    kind: "status",
    render: (item) => <StatusBadge tone={outcomeTone(item.outcome)}>{outcomeLabel(item.outcome)}</StatusBadge>,
    accessibleText: (item) => outcomeLabel(item.outcome),
  },
];

export function NotificationDeliveryHistory({
  items,
  complete = true,
  className,
}: {
  items?: NotificationDeliveryHistoryItem[];
  complete?: boolean;
  className?: string;
}) {
  const history = items ?? [];
  if (complete && history.length === 0) return null;

  return <section className={className} aria-labelledby="notification-delivery-history-heading">
    <div className="section-header">
      <div>
        <h2 id="notification-delivery-history-heading">Delivery history</h2>
        <p>Email delivery status for governed notices on this record.</p>
      </div>
    </div>
    {!complete && <Notice tone="warning">Some delivery history is unavailable.</Notice>}
    {history.length > 0 && <DataTable
      ariaLabel="Notification delivery history"
      rows={history}
      rowKey={(item) => item.event_id}
      rowName={(item) => `${item.action}, ${outcomeLabel(item.outcome)}`}
      columns={columns}
    />}
  </section>;
}
