import { useEffect, useRef, useState } from "react";
import { loadVendorForms, type VendorFormRow, type VendorFormSummary, type VendorFormsPage } from "../vendorFormsApi";
import type { VendorAssessment } from "../vendorAssessmentTypes";
import { Button, FocusedSheet, Notice, StatusBadge } from "./ui";
import { ResponseAssessment } from "./forms/ResponseAssessment";
import "./vendor-compliance-overview.css";

type Props = {
  relationshipID: string; serviceName: string; summary?: VendorFormSummary;
  summaryState: "loading" | "live" | "unavailable";
  assessment: VendorAssessment | null; assessmentState: "loading" | "live" | "unavailable";
  refreshKey: number; onOpenForms: () => void; onOpenDueDiligence: () => void;
  onRequestForm: () => void; onUpdated: () => void; onOpenRequest?: (id: string) => void;
};
const conclusionLabels = { SATISFACTORY: "Satisfactory", SATISFACTORY_WITH_CONDITIONS: "Conditional", UNSATISFACTORY: "Unsatisfactory", INCONCLUSIVE: "Inconclusive" };

export function VendorComplianceOverview(props: Props) {
  return <ComplianceOverview key={props.relationshipID} {...props}/>;
}

function ComplianceOverview({ relationshipID, serviceName, summary, summaryState, assessment, assessmentState, refreshKey, onOpenForms, onOpenDueDiligence, onRequestForm, onUpdated, onOpenRequest }: Props) {
  const [page, setPage] = useState<VendorFormsPage>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [response, setResponse] = useState<VendorFormRow>();
  const sequence = useRef(0);
  async function load(cursor?: string) {
    const current = ++sequence.current;
    setLoading(true); setError(false);
    if (!cursor) setPage(undefined);
    try {
      const result = await loadVendorForms(relationshipID, { limit: 25, ...(cursor ? { cursor } : {}) });
      if (current === sequence.current) setPage((previous) => ({ ...result, items: cursor ? [...(previous?.items ?? []), ...result.items] : result.items }));
    } catch { if (current === sequence.current) setError(true); }
    finally { if (current === sequence.current) setLoading(false); }
  }
  useEffect(() => { void load(); return () => { sequence.current++; }; }, [refreshKey]);
  const available = summaryState === "live" && !!summary && !!page && !error;
  const rows = page?.items ?? [];
  const attention = rows.some((row) => row.current && !!itemsFor(row).length);
  const unchecked = rows.some((row) => row.current && row.response_state === "SUBMITTED" && (row.outdated == null || row.attention_items === undefined));
  const overdueReview = !!assessment?.next_review_recommended_at && new Date(assessment.next_review_recommended_at).getTime() <= Date.now();
  const outdated = summary?.outdated_forms;
  const adverse = summary?.highest_concern && summary.highest_concern !== "LOW";
  const conclusion = assessment?.status === "COMPLETED" && assessment.conclusion ? conclusionLabels[assessment.conclusion] : undefined;
  let status = "Not assessed";
  let tone: "neutral" | "warning" | "error" | "success" | "info" = "neutral";
  if (!available) status = loading || summaryState === "loading" ? "Loading" : "Unavailable";
  else if (conclusion === "Unsatisfactory") { status = conclusion; tone = "error"; }
  else if (attention || adverse || summary.outstanding_forms > 0) { status = "Action required"; tone = "warning"; }
  else if ((outdated ?? 0) > 0 || overdueReview) { status = "Outdated"; tone = "warning"; }
  else if (summary.awaiting_review > 0 || assessment?.status === "SUBMITTED" || assessment?.status === "UNDER_REVIEW") { status = "Awaiting review"; tone = "info"; }
  else if (assessment?.status === "COLLECTING") { status = "Collecting evidence"; tone = "info"; }
  else if (assessment?.status === "SETUP_PENDING" || assessment?.status === "READY_TO_SEND") status = "Due diligence not started";
  else if (summary.unassessed_forms > 0) status = "Not assessed";
  else if (assessmentState !== "live" || outdated === undefined || (summary.freshness_unknown_forms ?? 0) > 0 || unchecked || page.next_cursor) status = "Unknown";
  else if (conclusion) { status = conclusion; tone = conclusion === "Satisfactory" ? "success" : "warning"; }
  else if (summary.assessed_forms > 0 && summary.highest_concern === "LOW") { status = "Low concern"; tone = "neutral"; }
  const empty = available && rows.length === 0 && summary.outstanding_forms === 0 && summary.submitted_forms === 0;
  const nextResponse = rows.find((row) => row.current && row.response_id && (itemsFor(row).length > 0 || row.required_reviews > row.completed_reviews));
  function updated() { void load(); onUpdated(); }
  return <section className="vendor-compliance" aria-label="Vendor compliance">
    <header className="vendor-compliance__heading"><div><h3>Compliance</h3><StatusBadge tone={tone}>{status}</StatusBadge></div><Button variant="primary" onPress={empty ? onRequestForm : nextResponse ? () => setResponse(nextResponse) : assessment ? onOpenDueDiligence : onOpenForms}>{empty ? "Request form" : nextResponse ? "Review response" : assessment ? "Review compliance" : "Manage forms"}</Button></header>
    {available && <dl className="vendor-compliance__counts">
      <div><dt>Incomplete</dt><dd>{summary.outstanding_forms}</dd>{summary.overdue_forms > 0 && <small>{summary.overdue_forms} overdue</small>}</div>
      <div><dt>Awaiting review</dt><dd>{summary.awaiting_review}</dd></div>
      <div><dt>Outdated</dt><dd>{outdated ?? "—"}</dd>{outdated === undefined ? <small>Not checked</small> : (summary.freshness_unknown_forms ?? 0) > 0 && <small>{summary.freshness_unknown_forms} not checked</small>}</div>
      {summary.submitted_forms > 0 && <div><dt>Assessed</dt><dd>{summary.assessed_forms}/{summary.submitted_forms}</dd></div>}
    </dl>}
    {conclusion && <p className="vendor-compliance__review"><strong>Last review: {conclusion}</strong>{assessment?.completed_at && <> · {dateTime(assessment.completed_at)}</>}{overdueReview && <> · Review overdue</>}</p>}
    {(error || summaryState === "unavailable") && <Notice tone="error"><strong>Compliance status unavailable</strong><p>Vendor forms could not be checked.</p><Button onPress={() => { void load(); onUpdated(); }}>Retry</Button></Notice>}
    {loading && !page && <p role="status">Loading vendor forms…</p>}
    {empty && <p>No forms requested</p>}
    {!!rows.length && <>
      <ul className="vendor-compliance__forms">{rows.map((row) => {
        const historical = !row.current || ["SUPERSEDED", "REVOKED", "CANCELLED"].includes(row.response_state);
        const items = historical ? [] : itemsFor(row);
        const score = row.assessed_score ?? row.score;
        const received = row.response_state === "NO_VENDOR_ACTION";
        const submitted = row.response_state === "SUBMITTED";
        return <li key={`${row.request_id}:${row.response_id ?? "pending"}`}>
          <div className="vendor-compliance__form-heading"><strong>{row.title}</strong><div className="vendor-compliance__form-status">{!historical && score?.raw_score != null && <span className="vendor-compliance__score">{scoreValue(score.raw_score)}% {score.mode === "RISK" ? "risk" : "compliance"}</span>}{!historical && score?.band && <StatusBadge tone={concernTone(score.band)}>{concernLabel(score.band)}</StatusBadge>}<StatusBadge tone={historical || submitted || received ? "neutral" : "info"}>{formStateLabel(row)}</StatusBadge></div></div>
          {items.length > 0 && <ul className="vendor-compliance__items">{items.map((item) => <li key={`${item.field_id ?? item.rule_id ?? item.label}:${item.state}`}><span>{item.label}</span><StatusBadge tone={item.state === "GAP" ? "warning" : "error"}>{item.state === "MISSING" ? "Missing" : item.state === "EXPIRED" ? "Expired" : "Not met"}</StatusBadge></li>)}</ul>}
          <div className="vendor-compliance__form-footer"><div>
            {items.some((item) => item.source === "RESPONSE") && <p>Based on submitted answers</p>}
            {items.some((item) => item.source === "REVIEW") && <p>Includes reviewed findings</p>}
            {(row.held_required ?? 0) > 0 && <p>{row.held_required} {row.held_required === 1 ? "document" : "documents"} received</p>}
            {row.response_currency === "PARTIALLY_REPLACED" && <p>Partly replaced · Review required</p>}
            {row.outdated && !items.some((item) => item.state === "EXPIRED") && <p>Outdated response</p>}
            {!historical && row.required_reviews > row.completed_reviews && <p>{row.required_reviews - row.completed_reviews} {row.required_reviews - row.completed_reviews === 1 ? "check" : "checks"} awaiting review</p>}
            {submitted && items.length === 0 && row.attention_items === undefined && <p>Evidence gaps not checked</p>}
            {!historical && !submitted && !received && <p>Due {dateTime(row.deadline)}</p>}
          </div>{historical ? <Button onPress={onOpenForms}>Review requests</Button> : submitted && row.response_id ? <Button aria-label={`Review ${row.title}`} onPress={() => setResponse(row)}>Review</Button> : received ? <Button aria-label={`Review evidence for ${row.title}`} onPress={onOpenDueDiligence}>Review evidence</Button> : <Button aria-label={`Open ${row.title} request`} onPress={() => onOpenRequest ? onOpenRequest(row.request_id) : onOpenForms()}>Open request</Button>}</div>
        </li>;
      })}</ul>
      {page?.next_cursor && <><p className="vendor-compliance__scope">{rows.length} forms shown</p><Button isLoading={loading} onPress={() => void load(page.next_cursor)}>Load more forms</Button></>}
    </>}
    {!conclusion && available && <p className="vendor-compliance__scope">{summary.assessed_forms} of {summary.submitted_forms} submitted forms assessed. Due diligence {assessmentState !== "live" ? "unavailable" : "incomplete"}.</p>}
    {summary && <p className="vendor-compliance__scope">Checked <time dateTime={summary.observed_at}>{dateTime(summary.observed_at)}</time> · {serviceName}</p>}
    {response?.response_id && <FocusedSheet label={`Review ${response.title}`} size="wide" onClose={() => setResponse(undefined)}><ResponseAssessment responseID={response.response_id} submissionScore={response.score} onUpdated={updated}/></FocusedSheet>}
  </section>;
}

function formStateLabel(row: VendorFormRow) {
  const labels: Record<string, string> = { SUPERSEDED: "Superseded", REVOKED: "Revoked", CANCELLED: "Cancelled", EXPIRED: "Access expired", NO_VENDOR_ACTION: "Received", SUBMITTED: "Submitted", READY_TO_SUBMIT: "Awaiting submission", REQUEST_READY: "Ready to send", IN_PROGRESS: "In progress", AWAITING_RESPONSE: "Awaiting vendor" };
  if (!row.current && !["SUPERSEDED", "REVOKED", "CANCELLED"].includes(row.response_state)) return "Historical";
  return labels[row.response_state] ?? "Status unknown";
}

function itemsFor(row: VendorFormRow) {
  const items: NonNullable<VendorFormRow["attention_items"]> = [];
  for (const item of row.attention_items ?? []) {
    const existing = items.findIndex((value) => value.field_id === item.field_id && value.rule_id === item.rule_id && value.label === item.label && value.state === item.state);
    if (existing < 0) items.push(item);
    else if (item.source === "REVIEW") items[existing] = item;
  }
  for (const field of row.missing_fields ?? []) if (!items.some((item) => item.field_id === field.id)) items.push({ field_id: field.id, label: field.label, state: "MISSING", source: "RESPONSE" });
  return items;
}
function scoreValue(value: number) { return Number.isInteger(value) ? String(value) : value.toFixed(1); }
function concernLabel(value: string) { return value.charAt(0) + value.slice(1).toLowerCase(); }
function concernTone(value: string): "neutral" | "warning" | "error" | "info" {
  if (value === "CRITICAL") return "error";
  if (value === "HIGH") return "warning";
  if (value === "MODERATE") return "info";
  return "neutral";
}
function dateTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Date unavailable" : date.toLocaleDateString(); }
