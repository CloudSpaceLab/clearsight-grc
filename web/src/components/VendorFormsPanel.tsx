import { useEffect, useRef, useState } from "react";
import { loadVendorForms, type VendorFormRow, type VendorFormsFilter, type VendorFormsPage } from "../vendorFormsApi";
import { loadCompletedResponse, loadCompletedResponses, loadDistribution, transitionDistribution, type CompletedResponsePage, type CompletedResponseSummary, type DistributionDetail } from "../formsDistributionApi";
import type { ResponseScore } from "../formsDistributionApi";
import { Button, EmptyState, FocusedSheet, Notice, SelectField, StatusBadge } from "./ui";
import { ResponseAssessment } from "./forms/ResponseAssessment";
import { DocumentBrowser } from "./documents/DocumentBrowser";
import { SentFormDetail } from "./forms/sent/SentFormDetail";
import { DistributionChangePanel } from "./forms/DistributionChangePanel";
import "./vendor-forms.css";

export const vendorWorkFilters = [{ id: "AWAITING_VENDOR", label: "Awaiting vendor" }, { id: "AWAITING_REVIEW", label: "Awaiting bank review" }, { id: "WITH_RISKS", label: "Submitted with risks" }, { id: "HIGH_RISK", label: "High or critical concern" }, { id: "OVERDUE", label: "Overdue" }, { id: "NOT_ASSESSED", label: "Not assessed" }] satisfies Array<{ id: VendorFormsFilter; label: string }>;
type Props = { relationshipID: string; serviceName: string; onRequestForm: () => void; onUpdated?: () => void; onOpenHistory?: () => void; onOpenRequest?: (requestID: string) => void; refreshKey?: number; initialFilter?: VendorFormsFilter };
export function VendorFormsPanel(props: Props) { return <FormsForRelationship key={`${props.relationshipID}:${props.initialFilter ?? "all"}`} {...props}/>; }

function FormsForRelationship({ relationshipID, serviceName, onRequestForm, onUpdated, onOpenHistory, onOpenRequest, refreshKey, initialFilter }: Props) {
  const [filter, setFilter] = useState<VendorFormsFilter | undefined>(initialFilter);
  const [page, setPage] = useState<VendorFormsPage>();
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string>();
  const [response, setResponse] = useState<VendorFormRow>();
  const [documents, setDocuments] = useState<VendorFormRow>();
  const [distributionID, setDistributionID] = useState<string>();
  const [historyOpen, setHistoryOpen] = useState(false);
  const sequence = useRef(0);
  async function reload() {
    const request = ++sequence.current;
    setLoading(true); setError(undefined); setPage(undefined);
    try { const result = await loadVendorForms(relationshipID, { filter, limit: 25 }); if (request === sequence.current) setPage(result); }
    catch (cause) { if (request === sequence.current) setError(cause instanceof Error ? cause.message : "Forms for this vendor service could not be loaded."); }
    finally { if (request === sequence.current) setLoading(false); }
  }
  useEffect(() => { void reload(); return () => { sequence.current++; }; }, [filter, refreshKey]);
  async function more() {
    if (!page?.next_cursor || loadingMore) return;
    const request = sequence.current;
    setLoadingMore(true); setError(undefined);
    try { const result = await loadVendorForms(relationshipID, { filter, cursor: page.next_cursor, limit: 25 }); if (request === sequence.current) setPage({ ...result, items: [...page.items, ...result.items] }); }
    catch (cause) { if (request === sequence.current) setError(cause instanceof Error ? cause.message : "More vendor forms could not be loaded. The current page remains available."); }
    finally { setLoadingMore(false); }
  }
  function updated() { void reload(); onUpdated?.(); }
  return <section className="vendor-forms-panel" aria-label={`Forms and responses for ${serviceName}`}>
    <header><div><h2>Forms and responses</h2><p>Requests, submissions and bank reviews for {serviceName}.</p></div><div className="vendor-forms-list__actions"><Button onPress={() => onOpenHistory ? onOpenHistory() : setHistoryOpen(true)}>View response history</Button><Button variant="primary" onPress={onRequestForm}>Request form</Button></div></header>
    <SelectField label="Form work to show" value={filter} placeholder="All requests and responses" options={vendorWorkFilters} onChange={setFilter}/>
    {loading && <p role="status">Loading forms and responses for this vendor service…</p>}
    {error && <Notice tone="error">{error}</Notice>}
    {!loading && !page && <Button onPress={() => void reload()}>Reload vendor forms</Button>}
    {page && <><p>{page.items.length} requests on this page · Checked <time dateTime={page.observed_at}>{dateTime(page.observed_at)}</time></p>
      {page.items.length === 0 && <EmptyState population={`Requests for ${serviceName} matching this filter`} title="No matching form requests" description="Choose another filter or request an approved form for this service."/>}
      <ul className="vendor-forms-list">{page.items.map((row) => {
        const submitted = row.response_state === "SUBMITTED";
        const pending = Math.max(0, row.required_reviews - row.completed_reviews);
        return <li key={`${row.request_id}:${row.response_id ?? "pending"}`}>
          <h3>{row.title}</h3><p>Form revision {row.form_template_version} · {row.response_currency === "PARTIALLY_REPLACED" ? "Partly replaced response" : row.current ? "Current request" : "Historical response"}</p>{row.purpose && <p>{row.purpose}</p>}
          <dl><div><dt>Vendor response</dt><dd><StatusBadge tone={submitted ? "neutral" : "info"}>{responseState(row.response_state)}</StatusBadge></dd></div><div><dt>Bank assessment</dt><dd>{row.assessment_state ? assessmentState(row.assessment_state) : submitted ? "Assessment state unavailable" : "Awaiting submission"}{submitted && pending > 0 && <p>{pending} required {pending === 1 ? "field" : "fields"} awaiting review</p>}</dd></div>
            <div><dt>Recipient</dt><dd>{row.recipient_hint || "Recipient not recorded"}</dd></div><div><dt>Deadline</dt><dd><time dateTime={row.deadline}>{dateTime(row.deadline)}</time></dd></div>
            <div><dt>Response coverage</dt><dd>{row.required_count == null || row.answered_required == null ? "Saved response progress unknown" : `${row.answered_required} of ${row.required_count} required answers ${submitted ? "submitted" : "saved"}`}</dd></div>
            {row.submitted_at && <div><dt>Submitted</dt><dd><time dateTime={row.submitted_at}>{dateTime(row.submitted_at)}</time></dd></div>}
            {row.delivery_state && <div><dt>Delivery</dt><dd>{responseState(row.delivery_state)}</dd></div>}
          </dl>
          {!submitted && <p>Saved progress does not include unsaved browser input.</p>}
          {!!row.missing_fields?.length && <details><summary>{row.missing_fields.length} required answers missing</summary><ul>{row.missing_fields.map((field) => <li key={field.id}>{field.label}</li>)}</ul></details>}
          {submitted && <><VendorFormScore title="Automatic submission result" score={row.score}/>{row.assessment_state && row.assessment_state !== "NOT_REQUIRED" && <VendorFormScore title="Bank-assessed result" score={row.assessed_score} provisional={pending > 0}/>}</>}
          {row.response_currency === "PARTIALLY_REPLACED" && <Notice tone="warning">Later submissions replaced some answers. The displayed result covers the earlier full response; the remaining fields require review.</Notice>}
          <div className="vendor-forms-list__actions">{submitted && row.response_id && <Button variant="primary" aria-label={`Review ${row.title} response`} onPress={() => setResponse(row)}>Review response</Button>}{submitted && row.response_id && <Button aria-label={`View ${row.title} documents`} onPress={() => setDocuments(row)}>View documents</Button>}{row.distribution_id && <Button onPress={() => setDistributionID(row.distribution_id)}>Manage request</Button>}{!row.distribution_id && onOpenRequest && <Button onPress={() => onOpenRequest(row.request_id)}>Open request</Button>}</div>
        </li>;
      })}</ul>
      {page.next_cursor && <Button isLoading={loadingMore} onPress={() => void more()}>Load more vendor forms</Button>}
    </>}
    {response?.response_id && <FocusedSheet label={`Review ${response.title} response`} size="wide" onClose={() => setResponse(undefined)}><ResponseAssessment responseID={response.response_id} onUpdated={updated}/></FocusedSheet>}
    {documents?.response_id && <FocusedSheet label={`${documents.title} documents`} size="wide" onClose={() => setDocuments(undefined)}><DocumentBrowser responseRevisionID={documents.response_id} scopeLabel={`${serviceName} · ${documents.title}`}/></FocusedSheet>}
    {distributionID && <FocusedSheet label="Manage vendor form request" size="wide" onClose={() => setDistributionID(undefined)}><VendorSentForm key={distributionID} distributionID={distributionID} onUpdated={updated}/></FocusedSheet>}
    {historyOpen && <FocusedSheet label={`${serviceName} response history`} size="wide" onClose={() => setHistoryOpen(false)}><VendorResponseHistory relationshipID={relationshipID} serviceName={serviceName} onUpdated={updated}/></FocusedSheet>}
  </section>;
}

export function VendorResponseHistory({ relationshipID, serviceName, onUpdated, active = true, refreshKey = 0 }: { relationshipID: string; serviceName: string; onUpdated: () => void; active?: boolean; refreshKey?: number }) {
  const [page, setPage] = useState<CompletedResponsePage>();
  const [selected, setSelected] = useState<CompletedResponseSummary>();
  const [documentsOpen, setDocumentsOpen] = useState(false);
  const [currencyChecked, setCurrencyChecked] = useState(true);
  const selectedRef = useRef(selected);
  selectedRef.current = selected;
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>();
  const sequence = useRef(0);
  async function load(cursor?: string) {
    const request = ++sequence.current;
    setLoading(true); setError(undefined);
    if (!cursor) { setPage(undefined); setCurrencyChecked(false); }
    try {
      const selectedID = !cursor ? selectedRef.current?.id : undefined;
      const [result, detail] = await Promise.all([
        loadCompletedResponses({ subject_type: "VENDOR_RELATIONSHIP", subject_id: relationshipID, current_only: false, sort: "COMPLETED_DESC", limit: 25, cursor }),
        selectedID ? loadCompletedResponse(selectedID) : undefined,
      ]);
      if (request === sequence.current) {
        setPage((previous) => ({ ...result, items: cursor ? [...(previous?.items ?? []), ...result.items] : result.items }));
        if (detail) setSelected((current) => current?.id === selectedID ? detail.response : current);
        setCurrencyChecked(true);
      }
    } catch (cause) { if (request === sequence.current) setError(cause instanceof Error ? cause.message : "Response history for this vendor service could not be loaded. Retry to check its submitted revisions."); }
    finally { if (request === sequence.current) setLoading(false); }
  }
  useEffect(() => { if (active) void load(); return () => { sequence.current++; }; }, [active, refreshKey]);
  if (selected) return <div className="vendor-forms-list"><Button onPress={() => setSelected(undefined)}>Back to response history</Button><h2>{selected.title}</h2><p>Response revision {selected.revision} · {!currencyChecked ? "Response currency not checked" : selected.current ? "Current response" : "Historical response"}</p>{error && <Notice tone="error">{error}</Notice>}<Button isDisabled={loading} onPress={() => void load()}>Reload response history</Button><ResponseAssessment responseID={selected.id} current={currencyChecked ? selected.current : null} onUpdated={onUpdated}/><Button onPress={() => setDocumentsOpen(true)}>View revision documents</Button>{documentsOpen && <FocusedSheet label="Response revision documents" size="wide" onClose={() => setDocumentsOpen(false)}><DocumentBrowser responseRevisionID={selected.id} scopeLabel={`${serviceName} · Response revision ${selected.revision}`}/></FocusedSheet>}</div>;
  return <section className="vendor-forms-list" aria-label="Submitted response history"><h2>Response history</h2><p>Current and historical submitted revisions for {serviceName}. Historical results do not describe the latest response.</p>{error && <Notice tone="error">{error}</Notice>}{loading && <p role="status">Loading submitted response revisions…</p>}<Button isDisabled={loading} onPress={() => void load()}>Reload response history</Button>{page && <><p>{page.items.length} submitted revisions loaded</p>{page.items.length === 0 && <EmptyState population={`Submitted response revisions for ${serviceName}`} title="No submitted responses found" description="Return to the vendor requests to check response progress."/>}<ul className="vendor-forms-list">{page.items.map((item) => <li key={item.id}><h3>{item.title}</h3><p>Response revision {item.revision} · Form revision {item.form_template_version} · {item.current ? "Current response" : "Historical response"}</p><p>Submitted <time dateTime={item.completed_at}>{dateTime(item.completed_at)}</time></p><VendorFormScore title="Submission result" score={item.score}/><Button aria-label={`Review ${item.title} revision ${item.revision}`} onPress={() => setSelected(item)}>Review response revision</Button></li>)}</ul>{page.next_cursor && <Button isLoading={loading} onPress={() => void load(page.next_cursor)}>Load earlier responses</Button>}</>}</section>;
}

function VendorFormScore({ title, score, provisional }: { title: string; score?: ResponseScore; provisional?: boolean }) {
  return <p><strong>{title}: </strong>{score?.state === "FAILED" ? "Unavailable" : score?.raw_score == null ? "Not scored" : `${score.raw_score} ${score.mode === "COMPLIANCE" ? "compliance" : "risk"} score`}{score?.band ? ` · ${score.band.toLowerCase()} concern` : ""}{provisional || score?.state === "PROVISIONAL" ? " · Provisional" : ""}{score?.coverage != null ? ` · ${Math.round(score.coverage * 100)}% score coverage` : " · Score coverage unknown"}</p>;
}
function VendorSentForm({ distributionID, onUpdated }: { distributionID: string; onUpdated: () => void }) {
  const [detail, setDetail] = useState<DistributionDetail>();
  const [error, setError] = useState<string>();
  const [busy, setBusy] = useState<string>();
  const [mode, setMode] = useState<"amend" | "supersede">();
  async function reload() { setError(undefined); try { setDetail(await loadDistribution(distributionID)); } catch (cause) { setError(cause instanceof Error ? cause.message : "The sent request could not be loaded."); } }
  useEffect(() => { void reload(); }, []);
  async function lifecycle(action: "lock" | "reopen" | "revoke") {
    if (!detail || busy) return;
    setBusy(action); setError(undefined);
    try { setDetail(await transitionDistribution(detail.distribution.id, detail.distribution.version, action)); onUpdated(); }
    catch (cause) { setError(cause instanceof Error ? cause.message : "The request could not be changed. Reload it before trying again."); }
    finally { setBusy(undefined); }
  }
  if (!detail) return <>{error ? <><Notice tone="error">{error}</Notice><Button onPress={() => void reload()}>Reload sent request</Button></> : <p role="status">Loading sent request…</p>}</>;
  if (mode) return <DistributionChangePanel detail={detail} mode={mode} onCancel={() => setMode(undefined)} onSaved={(value) => { setDetail(value); setMode(undefined); onUpdated(); }}/>;
  return <><SentFormDetail detail={detail} error={error} busy={busy} onLifecycle={(action) => void lifecycle(action)} onAmend={() => setMode("amend")} onSupersede={() => setMode("supersede")}/>{error && <Button onPress={() => void reload()}>Reload sent request</Button>}</>;
}
function dateTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Time not recorded" : date.toLocaleString(); }
function responseState(value: string) { const labels: Record<string, string> = { SUBMITTED: "Submitted", IN_PROGRESS: "In progress", AWAITING_RESPONSE: "Awaiting vendor", READY_TO_SUBMIT: "Saved answers complete; awaiting submission", REQUEST_READY: "Ready to send", CANCELLED: "Cancelled", EXPIRED: "Access expired" }; return labels[value] ?? value.toLowerCase().replaceAll("_", " "); }
function assessmentState(value: string) { const labels: Record<string, string> = { NOT_REQUIRED: "Bank review not required", AWAITING_REVIEW: "Awaiting bank review", IN_REVIEW: "Bank review in progress", ASSESSED: "Assessed" }; return labels[value] ?? "Assessment state unavailable"; }
