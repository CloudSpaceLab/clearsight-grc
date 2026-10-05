import { useEffect, useMemo, useState } from "react";
import { loadNotificationHistory, type NotificationHistoryEvent, type NotificationHistorySubject } from "../notificationHistoryApi";
import { DataTable, EmptyState, Notice, StatusBadge, type DataColumn, type StatusTone } from "./ui";

type LoadState = "loading" | "live" | "unavailable";

type Props = {
  subject: NotificationHistorySubject;
  subjectID: string;
  label: string;
};

function displayKind(kind: string) {
  const value = kind.replace(/^ATTENTION_/, "").replace(/_/g, " ").toLowerCase();
  return value.replace(/^./, (letter) => letter.toUpperCase());
}

function emailLabel(status?: string) {
  switch (status) {
    case "DELIVERED": return "Delivered";
    case "TEMPORARY_FAILURE": return "Retrying";
    case "DELIVERY_OUTCOME_UNKNOWN": return "Outcome unknown";
    case "CONTACT_UNAVAILABLE": return "No email contact";
    case "RECIPIENT_REJECTED": return "Rejected";
    case "PERMANENT_FAILURE": return "Failed";
    default: return "In-app only";
  }
}

function emailTone(status?: string): StatusTone {
  switch (status) {
    case "DELIVERED": return "success";
    case "TEMPORARY_FAILURE": return "info";
    case "DELIVERY_OUTCOME_UNKNOWN": return "warning";
    case "CONTACT_UNAVAILABLE":
    case "RECIPIENT_REJECTED":
    case "PERMANENT_FAILURE": return "error";
    default: return "neutral";
  }
}

export function NotificationHistorySection({ subject, subjectID, label }: Props) {
  const [items, setItems] = useState<NotificationHistoryEvent[]>([]);
  const [state, setState] = useState<LoadState>("loading");

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    void loadNotificationHistory(subject, subjectID, controller.signal)
      .then((value) => {
        if (controller.signal.aborted) return;
        setItems(value.items ?? []);
        setState("live");
      })
      .catch(() => {
        if (controller.signal.aborted) return;
        setItems([]);
        setState("unavailable");
      });
    return () => controller.abort();
  }, [subject, subjectID]);

  const columns = useMemo<readonly DataColumn<NotificationHistoryEvent>[]>(() => [
    {
      id: "change",
      header: "Change",
      mobileLayout: "full-width",
      render: (item) => displayKind(item.kind),
      accessibleText: (item) => displayKind(item.kind),
    },
    {
      id: "in-app",
      header: "In-app",
      kind: "number",
      render: (item) => item.in_app_deliveries,
      accessibleText: (item) => `${item.in_app_deliveries} in-app deliveries`,
    },
    {
      id: "email",
      header: "Email",
      kind: "status",
      render: (item) => <StatusBadge tone={emailTone(item.email_status)}>{emailLabel(item.email_status)}</StatusBadge>,
      accessibleText: (item) => emailLabel(item.email_status),
    },
    {
      id: "time",
      header: "Recorded",
      render: (item) => new Date(item.occurred_at).toLocaleString(),
      accessibleText: (item) => new Date(item.occurred_at).toLocaleString(),
    },
  ], []);

  return <section aria-labelledby={`${subject.toLowerCase()}-notification-history-heading`}>
    <div className="section-header">
      <div>
        <h2 id={`${subject.toLowerCase()}-notification-history-heading`}>Notifications</h2>
        <p>Material notices recorded for this {label}.</p>
      </div>
    </div>
    {state === "loading" && <Notice>Loading notification history…</Notice>}
    {state === "unavailable" && <Notice tone="warning">Notification history is unavailable.</Notice>}
    {state === "live" && items.length === 0 && <EmptyState population={label} title="No material notifications" description="No material notice is recorded for this record."/>}
    {items.length > 0 && <DataTable
      ariaLabel={`${label} notification history`}
      rows={items}
      rowKey={(item) => item.event_id}
      rowName={(item) => `${displayKind(item.kind)}, ${new Date(item.occurred_at).toLocaleString()}`}
      columns={columns}
    />}
  </section>;
}
