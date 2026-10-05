import type { NotificationHistoryItem } from "../attentionTypes";
import { DataTable, EmptyState, Notice, StatusBadge, type DataColumn } from "./ui";

type Props = {
  items?: NotificationHistoryItem[];
  complete?: boolean;
  recordLabel: string;
};

function historyKind(value: string) {
  if (value === "ATTENTION_CRITICAL") return "Critical notice";
  if (value === "WORK_ESCALATED") return "Escalated";
  if (value.endsWith("_WORSENED")) return "Worsened";
  if (value.endsWith("_CLEARED")) return "Cleared";
  if (value.endsWith("_OPENED")) return "Opened";
  return "Notification";
}

function channelLabel(value: string) {
  if (value === "IN_APP") return "In app";
  if (value === "ATTENTION_CRITICAL") return "Email";
  return value.replaceAll("_", " ").toLowerCase();
}

function statusTone(value: string): "success" | "warning" | "danger" | "info" | "unknown" {
  if (value === "DELIVERED") return "success";
  if (value === "DELIVERY_STARTED" || value === "TEMPORARY_FAILURE") return "warning";
  if (value === "PERMANENT_FAILURE" || value === "RECIPIENT_REJECTED" || value === "DELIVERY_OUTCOME_UNKNOWN") return "danger";
  if (value === "NOTICE_SUPERSEDED" || value === "CONTACT_UNAVAILABLE") return "info";
  return "unknown";
}

function statusLabel(value: string) {
  return value.replaceAll("_", " ").toLowerCase().replace(/^./, (letter) => letter.toUpperCase());
}

export function NotificationHistory({ items = [], complete = true, recordLabel }: Props) {
  const columns: readonly DataColumn<NotificationHistoryItem>[] = [
    { id: "change", header: "Change", render: (item) => historyKind(item.kind), accessibleText: (item) => historyKind(item.kind) },
    { id: "channel", header: "Channel", render: (item) => channelLabel(item.channel), accessibleText: (item) => channelLabel(item.channel) },
    { id: "status", header: "Delivery", kind: "status", render: (item) => <StatusBadge tone={statusTone(item.status)}>{statusLabel(item.status)}</StatusBadge>, accessibleText: (item) => statusLabel(item.status) },
    { id: "time", header: "Time", render: (item) => new Date(item.occurred_at).toLocaleString(), accessibleText: (item) => new Date(item.occurred_at).toLocaleString() },
  ];

  return <section className="risk-record__history" aria-label="Notification history">
    <div className="section-header"><div><h2>Notifications</h2><p>Material notices sent for this record.</p></div></div>
    {!complete && <Notice tone="warning">Some notification history is unavailable.</Notice>}
    {items.length ? <DataTable
      ariaLabel={recordLabel + " notification history"}
      rows={items}
      rowKey={(item) => [item.channel, item.kind, item.notice_sequence ?? 0, item.occurred_at].join(":")}
      rowName={(item) => [historyKind(item.kind), channelLabel(item.channel), statusLabel(item.status)].join(", ")}
      columns={columns}
    /> : <EmptyState population={recordLabel} title="No notifications" description="No material notice is recorded for this record."/>}
  </section>;
}
