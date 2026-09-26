import { useEffect, useState, type ReactNode } from "react";
import { fetchProcessingActivity, fetchProcessingActivityHistory, type RopaProcessingActivityHistoryParams } from "../ropaApi";
import type { ProcessingActivity, ProcessingActivityHistoryResponse, ProcessingActivityResponse, ProcessingActivityStatus, Recipient, Review } from "../ropaTypes";
import { Button, EmptyState, Notice, StatusBadge, type StatusTone } from "./ui";
import "./ropa.css";

type ActivityPageProps = {
  activityID: string;
  organizationName?: string;
  legalEntityName?: string;
  onBack?: () => void;
  loadActivity?: (id: string, signal?: AbortSignal) => Promise<ProcessingActivityResponse>;
  loadHistory?: (id: string, params?: RopaProcessingActivityHistoryParams, signal?: AbortSignal) => Promise<ProcessingActivityHistoryResponse>;
};

type PageState = "loading" | "live" | "not-found" | "error";
type HistoryState = "loading" | "live" | "error";

const unavailableActivity = async (_id?: string, _signal?: AbortSignal): Promise<ProcessingActivityResponse> => {
  throw new Error("Couldn’t load processing activity.");
};

const emptyHistory = async (_id?: string, _params?: RopaProcessingActivityHistoryParams, _signal?: AbortSignal): Promise<ProcessingActivityHistoryResponse> => ({ events: [], has_more: false });

export function RopaActivityPage({ activityID, organizationName, legalEntityName, onBack, loadActivity = fetchProcessingActivity, loadHistory = fetchProcessingActivityHistory }: ActivityPageProps) {
  const activityReader = loadActivity ?? unavailableActivity;
  const historyReader = loadHistory ?? emptyHistory;
  const [response, setResponse] = useState<ProcessingActivityResponse>();
  const [history, setHistory] = useState<ProcessingActivityHistoryResponse>();
  const [state, setState] = useState<PageState>("loading");
  const [historyState, setHistoryState] = useState<HistoryState>("loading");
  const [retry, setRetry] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    setResponse(undefined);
    void activityReader(activityID, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setResponse(value);
      setState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setState(isNotFound(error) ? "not-found" : "error");
    });
    return () => controller.abort();
  }, [activityID, activityReader, retry]);

  useEffect(() => {
    const controller = new AbortController();
    setHistoryState("loading");
    setHistory(undefined);
    void historyReader(activityID, { limit: 100 }, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setHistory(value);
      setHistoryState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setHistoryState("error");
    });
    return () => controller.abort();
  }, [activityID, historyReader, retry]);

  function goBack() {
    if (onBack) {
      onBack();
      return;
    }
    if (typeof window !== "undefined") window.location.hash = "#ropa";
  }

  function retryLoads() {
    setRetry((value) => value + 1);
  }

  if (state === "loading") return <section className="ropa-activity-page" aria-busy="true">
    <Button variant="quiet" onPress={goBack}>Back to register</Button>
    <h1>Loading processing activity</h1>
  </section>;

  if (state === "not-found") return <section className="ropa-activity-page" aria-label="Processing activity not found">
    <EmptyState
      population="Processing activities in this legal entity"
      title="Processing activity not found"
      description="Return to the register."
      action={<Button onPress={goBack}>Back to register</Button>}
    />
  </section>;

  if (state === "error" || !response) return <section className="ropa-activity-page" aria-labelledby="ropa-activity-error">
    <h1 id="ropa-activity-error">Processing activity unavailable</h1>
    <Notice tone="error">
      <span>Couldn’t load processing activity.</span> <Button variant="secondary" size="compact" onPress={retryLoads}>Retry</Button>
    </Notice>
    <Button variant="quiet" onPress={goBack}>Back to register</Button>
  </section>;

  const activity = response.activity;
  const blockers = response.closure_blockers ?? [];
  const scope = legalEntityName || "this legal entity";
  const descriptionID = "ropa-closure-blockers-description";

  return <section className="ropa-activity-page" aria-labelledby="ropa-activity-heading">
    <header className="topbar ropa-page-header">
      <div>
        <span className="eyebrow">{organizationName || "Processing activity register"} · {scope}</span>
        <h1 id="ropa-activity-heading">{activity.name || "Unnamed activity"}</h1>
        <p>{activity.code || "No code"} · v{activity.version}</p>
      </div>
      <div className="topbar-actions">
        <StatusBadge tone={activityStatusTone(activity.status)}>{activityStatusLabel(activity.status)}</StatusBadge>
        <Button variant="secondary" onPress={goBack}>Back to register</Button>
      </div>
    </header>

    {activity.description && <p className="ropa-activity-description">{activity.description}</p>}

    {blockers.length > 0 ? <section className="ropa-closure-blockers" aria-labelledby="ropa-closure-blockers-heading">
      <div className="ropa-closure-blockers__heading">
        <StatusBadge tone="warning">Closure blocked</StatusBadge>
        <h2 id="ropa-closure-blockers-heading">Complete these facts before closing</h2>
        <p id={descriptionID}>Complete all required facts.</p>
      </div>
      <ul className="ropa-closure-blockers__list">
        {blockers.map((blocker) => <li key={blocker}><strong>{blockerLabel(blocker)}</strong><span>{blockerInstruction(blocker)}</span></li>)}
      </ul>
      <div className="ropa-closure-blockers__action">
        <Button variant="primary" isDisabled aria-describedby={descriptionID}>Close processing activity</Button>
        <span>Resolve blockers first.</span>
      </div>
    </section> : <Notice tone="success">Closure checks passed.</Notice>}

    <section className="ropa-activity-facts" aria-labelledby="ropa-facts-heading">
      <div className="section-header"><div><h2 id="ropa-facts-heading">Processing activity facts</h2></div></div>
      <dl className="ropa-fact-grid">
        <Fact label="Purpose" value={recordedValue(activity.purpose)}/>
        <Fact label="Lawful basis" value={recordedValue(activity.lawful_basis)}/>
        <Fact label="Controller" value={recordedValue(activity.controller)}/>
        <Fact label="Processor" value={recordedValue(activity.processor)}/>
        <Fact label="Data subject categories" value={recordedValue(activity.data_subject_categories)}/>
        <Fact label="Personal data categories" value={recordedValue(activity.personal_data_categories)}/>
        <Fact label="Retention period" value={recordedValue(activity.retention_period)}/>
        <Fact label="Security measures" value={recordedValue(activity.security_measures)}/>
        <Fact label="Automated decision making" value={<StatusBadge tone={activity.automated_decision_making ? "warning" : "neutral"}>{activity.automated_decision_making ? "Yes" : "No"}</StatusBadge>}/>
        <Fact label="Start date" value={dateOrNotRecorded(activity.start_date, "start date")}/>
        <Fact label="End date" value={dateOrNotRecorded(activity.end_date, "end date")}/>
        <Fact label="Next review" value={<ReviewDate value={activity.next_review_date}/>} />
        <Fact label="Accountable owner" value={principalLabel(activity.owner_display_name, activity.owner_principal_id)}/>
        <Fact label="Required authority" value={principalLabel(activity.required_authority_display_name, activity.required_authority_principal_id)}/>
      </dl>
    </section>

    <div className="ropa-activity-columns">
      <DataCategories activity={activity}/>
      <Recipients activity={activity}/>
      <Systems activity={activity}/>
    </div>

    <section className="ropa-review-history" aria-labelledby="ropa-review-heading">
      <div className="section-header"><div><h2 id="ropa-review-heading">Review history</h2></div></div>
      <Reviews reviews={activity.reviews ?? []} activityName={activity.name}/>
    </section>

    <section className="ropa-change-history" aria-labelledby="ropa-change-heading">
      <div className="section-header"><div><h2 id="ropa-change-heading">Change history</h2></div></div>
      {historyState === "loading" && <p className="ropa-load-state" role="status">Loading history…</p>}
      {historyState === "error" && <Notice tone="error"><span>Couldn’t load history.</span> <Button variant="secondary" size="compact" onPress={retryLoads}>Retry</Button></Notice>}
      {historyState === "live" && history && <History events={history.events} hasMore={history.has_more} activityName={activity.name}/>}
    </section>
  </section>;
}

function Fact({ label, value }: { label: string; value: string | ReactNode }) {
  return <div><dt>{label}</dt><dd>{value}</dd></div>;
}

function DataCategories({ activity }: { activity: ProcessingActivity }) {
  const categories = activity.data_categories ?? [];
  return <section className="ropa-activity-list" aria-labelledby="ropa-data-categories-heading">
    <h2 id="ropa-data-categories-heading">Personal data categories</h2>
    {categories.length > 0 ? <ul>{categories.map((category) => <li key={category.category}><strong>{category.category}</strong><StatusBadge tone={sensitivityTone(category.sensitivity)}>{sensitivityLabel(category.sensitivity)}</StatusBadge></li>)}</ul> : <EmptyState population={`Personal data categories for ${activity.name || "this processing activity"}`} title="No data categories" description="Add data categories."/>}
  </section>;
}

function Recipients({ activity }: { activity: ProcessingActivity }) {
  const recipients = activity.recipients ?? [];
  return <section className="ropa-activity-list" aria-labelledby="ropa-recipients-heading">
    <h2 id="ropa-recipients-heading">Recipients and transfers</h2>
    {recipients.length > 0 ? <ul>{recipients.map((recipient) => <RecipientRow key={recipient.recipient} recipient={recipient}/>)}</ul> : <EmptyState population={`Recipients for ${activity.name || "this processing activity"}`} title="No recipients" description="Add recipients where applicable."/>}
  </section>;
}

function RecipientRow({ recipient }: { recipient: Recipient }) {
  return <li>
    <strong>{recipient.recipient || "Recipient name not recorded"}</strong>
    <span>{recipientKindLabel(recipient.recipient_kind)}</span>
    {recipient.is_cross_border ? <small>Country: {recipient.country_code || "Not recorded"} · Safeguard: {transferBasisLabel(recipient.transfer_basis)}</small> : <small>No cross-border transfer</small>}
  </li>;
}

function Systems({ activity }: { activity: ProcessingActivity }) {
  const systems = activity.systems ?? [];
  return <section className="ropa-activity-list" aria-labelledby="ropa-systems-heading">
    <h2 id="ropa-systems-heading">Systems</h2>
    {systems.length > 0 ? <ul>{systems.map((system) => <li key={system.system_name}><strong>{system.system_name || "System name not recorded"}</strong><span>{systemKindLabel(system.system_kind)}</span></li>)}</ul> : <EmptyState population={`Systems for ${activity.name || "this processing activity"}`} title="No systems" description="Add supporting systems."/>}
  </section>;
}

function Reviews({ reviews, activityName }: { reviews: Review[]; activityName: string }) {
  if (reviews.length === 0) return <EmptyState population={`Review history for ${activityName || "this processing activity"}`} title="No reviews" description="Record a review."/>;
  return <ul className="ropa-review-list">{reviews.map((review) => <li key={review.id}>
    <div><strong>{review.completed_at ? `Review completed ${formatActivityDate(review.completed_at)}` : `Review due ${formatActivityDate(review.due_date)}`}</strong><StatusBadge tone={reviewTone(review)}>{reviewStatusLabel(review)}</StatusBadge></div>
    <span>{review.outcome ? `Outcome: ${outcomeLabel(review.outcome)}` : "Outcome not recorded"}</span>
    <small>{principalLabel(review.reviewer_display_name, review.reviewer_principal_id, "No reviewer")}</small>
  </li>)}</ul>;
}

function History({ events, hasMore, activityName }: { events: ProcessingActivityHistoryResponse["events"]; hasMore: boolean; activityName: string }) {
  if (events.length === 0) return <EmptyState population={`Recorded changes for ${activityName || "this processing activity"}`} title="No history" description="No changes recorded."/>;
  return <>
    <ol className="ropa-history-list">{events.map((event) => <li key={event.id}><strong>{eventTypeLabel(event.type)}</strong><span>{formatActivityDate(event.occurred_at)}</span><small>Record version {event.aggregate_version}</small></li>)}</ol>
    {hasMore && <p className="ropa-load-state">More history available.</p>}
  </>;
}

function ReviewDate({ value }: { value?: string }) {
  if (!value) return <span>Not recorded</span>;
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return <span>Review date unavailable</span>;
  const overdue = new Date(value).getTime() < startOfToday().getTime();
  return <StatusBadge tone={overdue ? "error" : "info"}>{overdue ? `Overdue · ${formatActivityDate(value)}` : `Due · ${formatActivityDate(value)}`}</StatusBadge>;
}

function principalLabel(displayName: string | undefined, principalID: string | undefined, empty = "Not assigned"): string {
  const name = displayName?.trim();
  if (name) return name;
  return principalID ? "Assigned" : empty;
}

function recordedValue(value: string | undefined): string {
  return value && value.trim() ? value : "Not recorded";
}

function dateOrNotRecorded(value: string | undefined, _label: string): string {
  return value ? formatActivityDate(value) : "Not recorded";
}

function blockerLabel(blocker: string): string {
  const normalized = blocker.toLowerCase();
  if (normalized.includes("lawful")) return "Lawful basis";
  if (normalized.includes("owner")) return "Named owner";
  if (normalized.includes("subject")) return "Data subject category";
  if (normalized.includes("review")) return "Completed review";
  return "Required closure fact";
}

function blockerInstruction(blocker: string): string {
  const normalized = blocker.toLowerCase();
  if (normalized.includes("lawful")) return "Add lawful basis before closing.";
  if (normalized.includes("owner")) return "Assign an owner before closing.";
  if (normalized.includes("subject")) return "Add a data subject category before closing.";
  if (normalized.includes("review")) return "Complete a review before closing.";
  return "Complete this item before closing.";
}

function sensitivityLabel(value: string): string {
  if (value === "DIRECT_PERSONAL") return "Direct personal data";
  if (value === "INDIRECT_PERSONAL") return "Indirect personal data";
  if (value === "SENSITIVE_BY_NATURE") return "Sensitive by nature";
  if (value === "SENSITIVE_BY_LAW") return "Sensitive by law";
  if (value === "UNCLASSIFIED") return "Not classified";
  return "Sensitivity not recorded";
}

function sensitivityTone(value: string): StatusTone {
  if (value === "SENSITIVE_BY_NATURE" || value === "SENSITIVE_BY_LAW") return "warning";
  if (value === "DIRECT_PERSONAL" || value === "INDIRECT_PERSONAL") return "info";
  return "neutral";
}

function recipientKindLabel(value: string): string {
  if (value === "INTERNAL") return "Internal recipient";
  if (value === "EXTERNAL") return "External recipient";
  if (value === "AUTHORITY") return "Authority recipient";
  return "Recipient type not recorded";
}

function transferBasisLabel(value: string): string {
  if (value === "ADEQUACY") return "Adequacy decision";
  if (value === "APPROVED_INSTRUMENT") return "Approved transfer instrument";
  if (value === "RECOGNISED_LAWFUL_BASIS") return "Recognised lawful basis";
  if (value === "CONSENT") return "Consent";
  if (value === "STANDARD_CONTRACT_CLAUSES") return "Standard contract clauses";
  if (value === "BINDING_CORPORATE_RULES") return "Binding corporate rules";
  if (value === "CERTIFICATION") return "Certification";
  if (value === "NOT_APPLICABLE") return "Not applicable";
  return "Safeguard not recorded";
}

function systemKindLabel(value: string): string {
  if (value === "APPLICATION") return "Application";
  if (value === "DATABASE") return "Database";
  if (value === "FILE") return "File store";
  if (value === "MANUAL") return "Manual process";
  if (value === "THIRD_PARTY") return "Third-party system";
  return "System type not recorded";
}

function outcomeLabel(value: string): string {
  if (value === "CONFIRMED") return "Confirmed";
  if (value === "REVISED") return "Revised";
  if (value === "WITHDRAWN") return "Withdrawn";
  return "Outcome not recorded";
}

function reviewStatusLabel(review: Review): string {
  if (!review.completed_at) return new Date(review.due_date).getTime() < startOfToday().getTime() ? "Review overdue" : "Review pending";
  if (review.outcome === "CONFIRMED") return "Confirmed";
  if (review.outcome === "REVISED") return "Revised";
  if (review.outcome === "WITHDRAWN") return "Withdrawn";
  return "Review completed";
}

function reviewTone(review: Review): StatusTone {
  if (review.completed_at) return review.outcome === "WITHDRAWN" ? "neutral" : "success";
  return new Date(review.due_date).getTime() < startOfToday().getTime() ? "error" : "info";
}

function eventTypeLabel(value: string): string {
  if (value === "processing_activity.created") return "Processing activity recorded";
  if (value === "processing_activity.updated") return "Processing activity updated";
  if (value === "processing_activity.transitioned") return "Processing activity status changed";
  return "Processing activity change";
}

function activityStatusLabel(status: ProcessingActivityStatus): string {
  if (status === "NEW") return "Not started";
  if (status === "OPEN") return "In progress";
  if (status === "CLOSED") return "Complete";
  return "Status unavailable";
}

function activityStatusTone(status: ProcessingActivityStatus): StatusTone {
  if (status === "CLOSED") return "success";
  if (status === "OPEN") return "info";
  return "neutral";
}

function formatActivityDate(value: string): string {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "date unavailable";
  return new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", year: "numeric", timeZone: "UTC" }).format(date);
}

function startOfToday(): Date {
  const now = new Date();
  return new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
}

function isAbortError(error: unknown): boolean {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}

function isNotFound(error: unknown): boolean {
  if (typeof error === "object" && error !== null && "kind" in error && error.kind === "not_found") return true;
  return error instanceof Error && /not found|not available in your legal entity/i.test(error.message);
}
