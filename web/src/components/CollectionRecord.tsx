import type { ReactNode } from "react";
import type { CollectionCurrencyState, CollectionSummary, FormTemplate, MonitoringCheck } from "../monitoringTypes";

type Props = {
  form?: FormTemplate;
  check: MonitoringCheck;
  summary?: CollectionSummary;
  summaryLoading?: boolean;
  summaryUnavailable?: boolean;
  onRetrySummary?: () => void;
  children?: ReactNode;
};

const currencyLabels: Record<CollectionCurrencyState, string> = {
  NO_RESPONSE_SUBMITTED: "No response submitted",
  CURRENT: "Current",
  RENEWAL_DUE: "Renewal due",
  RESPONSE_POTENTIALLY_EXPIRED: "Response potentially expired",
  AWAITING_RESPONSE: "Awaiting response",
  RENEWAL_BLOCKED: "Renewal blocked",
};

function statusLabel(status: MonitoringCheck["status"]) {
  switch (status) {
    case "DRAFT": return "Draft";
    case "PENDING_APPROVAL": return "Awaiting approval";
    case "ACTIVE": return "Active";
    case "PAUSED": return "Paused";
    case "REJECTED": return "Returned";
    case "RETIRED": return "Ended";
  }
}

function formatDateTime(value: string) {
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

export function CollectionRecord({ form, check, summary, summaryLoading, summaryUnavailable, onRetrySummary, children }: Props) {
  const policy = check.collection_policy;
  const respondent = summary?.respondent_label || summary?.recipient_hint || "External respondent";
  return <article className="monitoring-record collection-record">
    <div className="collection-record-heading">
      <span className="record-type">Collection form</span>
      <h4>{check.name}</h4>
      <p>{form ? `${form.fields.length} question${form.fields.length === 1 ? "" : "s"} · ` : ""}{statusLabel(check.status)}{policy ? ` · Valid for ${policy.validity_months} months` : ""}</p>
      {policy ? <p>Renewal starts {policy.renewal_window_days} days before expiry · {policy.reminder_count} reminder{policy.reminder_count === 1 ? "" : "s"}</p> : <p>Response expiry is not configured.</p>}
    </div>
    {summaryLoading ? <div className="collection-activity" aria-live="polite"><span>Loading collection dates…</span></div> : summaryUnavailable ? <div className="collection-summary-unavailable"><strong>Collection dates unavailable</strong><button className="text-button" type="button" onClick={onRetrySummary}>Retry collection dates</button></div> : summary ? <div className="collection-activity">
      <strong className={`collection-currency collection-currency-${summary.currency_state.toLowerCase()}`}>{currencyLabels[summary.currency_state]}</strong>
      {summary.latest_submission_at && <span>Last submitted <time dateTime={summary.latest_submission_at}>{formatDateTime(summary.latest_submission_at)}</time> by {respondent}</span>}
      <span>Expires <time dateTime={summary.expires_at}>{formatDateTime(summary.expires_at)}</time></span>
      {summary.active_request_deadline && <span>Current request due <time dateTime={summary.active_request_deadline}>{formatDateTime(summary.active_request_deadline)}</time> · {summary.reminders_sent} of {summary.reminder_count} reminders sent</span>}
      {summary.last_error_safe && <span className="collection-safe-error">{summary.last_error_safe}</span>}
      <span className="collection-freshness">Collection status updated <time dateTime={summary.projection_generated_at}>{formatDateTime(summary.projection_generated_at)}</time></span>
    </div> : <div className="collection-activity"><strong>No response submitted</strong><span>A collection request has not produced a submitted response.</span></div>}
    {children}
  </article>;
}
