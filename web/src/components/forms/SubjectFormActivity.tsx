import { useEffect, useMemo, useState, type ReactNode } from "react";
import {
  loadCompletedResponses,
  loadDistributionPage,
  type CompletedResponseSummary,
  type Distribution,
} from "../../formsDistributionApi";
import { ActionLink, Button, EmptyState, Notice, StatusBadge } from "../ui";
import { concernText, concernTone } from "./responseScorePresentation";

type SubjectType = "MATTER" | "PROGRAM" | "VENDOR_RELATIONSHIP";
type LoadState = "loading" | "live" | "unavailable";

type Props = {
  subjectType: SubjectType;
  subjectID: string;
  subjectLabel: string;
  limit?: number;
  leading?: ReactNode;
  variant?: "default" | "cards";
  refreshKey?: number;
};

export function SubjectFormActivity({ subjectType, subjectID, subjectLabel, limit = 6, leading, variant = "default", refreshKey = 0 }: Props) {
  const [requestState, setRequestState] = useState<LoadState>("loading");
  const [responseState, setResponseState] = useState<LoadState>("loading");
  const [requests, setRequests] = useState<Distribution[]>([]);
  const [responses, setResponses] = useState<CompletedResponseSummary[]>([]);
  const [requestCursor, setRequestCursor] = useState<string>();
  const [responseCursor, setResponseCursor] = useState<string>();
  const [loadingMoreRequests, setLoadingMoreRequests] = useState(false);
  const [loadingMoreResponses, setLoadingMoreResponses] = useState(false);
  const [requestPageError, setRequestPageError] = useState("");
  const [responsePageError, setResponsePageError] = useState("");
  const [requestReload, setRequestReload] = useState(0);
  const [responseReload, setResponseReload] = useState(0);

  useEffect(() => {
    let active = true;
    setRequestState("loading");
    setRequests([]);
    setRequestCursor(undefined);
    setRequestPageError("");
    void loadDistributionPage({ subject_type: subjectType, subject_id: subjectID, limit })
      .then((page) => {
        if (!active) return;
        setRequests(page.items);
        setRequestCursor(page.next_cursor);
        setRequestState("live");
      })
      .catch(() => {
        if (active) setRequestState("unavailable");
      });
    return () => { active = false; };
  }, [limit, refreshKey, requestReload, subjectID, subjectType]);

  useEffect(() => {
    let active = true;
    setResponseState("loading");
    setResponses([]);
    setResponseCursor(undefined);
    setResponsePageError("");
    void loadCompletedResponses({
      subject_type: subjectType,
      subject_id: subjectID,
      current_only: true,
      sort: "COMPLETED_DESC",
      limit,
    }).then((page) => {
      if (!active) return;
      setResponses(page.items);
      setResponseCursor(page.next_cursor);
      setResponseState("live");
    }).catch(() => {
      if (active) setResponseState("unavailable");
    });
    return () => { active = false; };
  }, [limit, refreshKey, responseReload, subjectID, subjectType]);

  async function loadMoreRequests() {
    if (!requestCursor || loadingMoreRequests) return;
    setLoadingMoreRequests(true);
    setRequestPageError("");
    try {
      const page = await loadDistributionPage({
        subject_type: subjectType,
        subject_id: subjectID,
        limit,
        cursor: requestCursor,
      });
      setRequests((current) => appendUniqueByID(current, page.items));
      setRequestCursor(page.next_cursor);
    } catch {
      setRequestPageError("More form requests could not be loaded.");
    } finally {
      setLoadingMoreRequests(false);
    }
  }

  async function loadMoreResponses() {
    if (!responseCursor || loadingMoreResponses) return;
    setLoadingMoreResponses(true);
    setResponsePageError("");
    try {
      const page = await loadCompletedResponses({
        subject_type: subjectType,
        subject_id: subjectID,
        current_only: true,
        sort: "COMPLETED_DESC",
        limit,
        cursor: responseCursor,
      });
      setResponses((current) => appendUniqueByID(current, page.items));
      setResponseCursor(page.next_cursor);
    } catch {
      setResponsePageError("More submitted responses could not be loaded.");
    } finally {
      setLoadingMoreResponses(false);
    }
  }

  const responseDistributionIDs = useMemo(() => new Set(responses.map((response) => response.distribution_id)), [responses]);
  const pending = requests.filter((request) =>
    ["READY", "OPEN", "LOCKED"].includes(request.status) && !responseDistributionIDs.has(request.id),
  ).length;
  const expired = requests.filter((request) => request.status === "EXPIRED").length;
  const needsReview = responses.filter((response) =>
    response.state === "PROVISIONAL" || response.score?.state === "PROVISIONAL" || response.score?.state === "FAILED",
  ).length;
  const attention = [
    pending ? `${pending} response${pending === 1 ? "" : "s"} pending` : "",
    expired ? `${expired} request${expired === 1 ? "" : "s"} expired` : "",
    needsReview ? `${needsReview} submitted response${needsReview === 1 ? " needs" : "s need"} review` : "",
  ].filter(Boolean);
  const attentionIsPartial = Boolean(requestCursor || responseCursor);

  return <section className={`subject-form-activity${variant === "cards" ? " subject-form-activity--cards" : ""}`} aria-label={`Form activity for ${subjectLabel}`}>
    {attention.length > 0 && <Notice tone="warning">{attentionIsPartial ? "Shown records: " : ""}{attention.join(" · ")}.</Notice>}
    {leading}

    <div className={variant === "cards" ? "subject-form-activity__group subject-form-activity__card" : "subject-form-activity__group"}>
      <header><h3>Requests</h3>{requestState === "live" && (variant !== "cards" || requests.length > 0) && <span>{requests.length} shown</span>}</header>
      {requestState === "loading" && <p role="status">Loading form requests…</p>}
      {requestState === "unavailable" && <Notice tone="warning">Form requests are unavailable. Other issue work remains available. <Button variant="secondary" size="compact" onPress={() => setRequestReload((value) => value + 1)}>Retry form requests</Button></Notice>}
      {requestState === "live" && requests.length === 0 && (variant === "cards"
        ? <EmptyState compact population="No requests sent" title="No requests sent" description="Send an approved form to an employee to collect evidence."/>
        : <p>No form requests recorded.</p>)}
      {requests.length > 0 && <ul>{requests.map((request) => <li key={request.id}>
        <div><strong>{request.title}</strong><span>{distributionStatusLabel(request.status)} · Due {formatDate(request.deadline)}</span></div>
        <div className="subject-form-activity__actions">
          <StatusBadge tone={distributionTone(request.status)}>{distributionStatusLabel(request.status)}</StatusBadge>
          <ActionLink href={`#forms?section=sent-forms&distribution=${encodeURIComponent(request.id)}`}>Open sent form</ActionLink>
        </div>
      </li>)}</ul>}
      {requestPageError && <Notice tone="warning">{requestPageError} <Button variant="secondary" size="compact" isLoading={loadingMoreRequests} onPress={() => void loadMoreRequests()}>Retry form requests</Button></Notice>}
      {requestCursor && !requestPageError && <Button variant="secondary" size="compact" isLoading={loadingMoreRequests} onPress={() => void loadMoreRequests()}>Load more form requests</Button>}
    </div>

    <div className={variant === "cards" ? "subject-form-activity__group subject-form-activity__card" : "subject-form-activity__group"}>
      <header><h3>Submitted responses</h3>{responseState === "live" && (variant !== "cards" || responses.length > 0) && <span>{responses.length} shown</span>}</header>
      {responseState === "loading" && <p role="status">Loading submitted responses…</p>}
      {responseState === "unavailable" && <Notice tone="warning">Submitted responses are unavailable. Other issue work remains available. <Button variant="secondary" size="compact" onPress={() => setResponseReload((value) => value + 1)}>Retry submitted responses</Button></Notice>}
      {responseState === "live" && responses.length === 0 && (variant === "cards"
        ? <EmptyState compact population="No responses yet" title="No responses yet" description="Submitted evidence will appear here for review."/>
        : <p>No submitted responses recorded.</p>)}
      {responses.length > 0 && <ul>{responses.map((response) => <li key={response.id}>
        <div>
          <strong>{response.title}</strong>
          <span>{responseStateLabel(response)} · Submitted {formatDate(response.completed_at)}</span>
        </div>
        <div className="subject-form-activity__actions">
          <StatusBadge tone={concernTone(response.score)}>{concernText(response.score)}</StatusBadge>
          <ActionLink href={`#forms?section=responses&response=${encodeURIComponent(response.id)}`}>Review response</ActionLink>
        </div>
      </li>)}</ul>}
      {responsePageError && <Notice tone="warning">{responsePageError} <Button variant="secondary" size="compact" isLoading={loadingMoreResponses} onPress={() => void loadMoreResponses()}>Retry submitted responses</Button></Notice>}
      {responseCursor && !responsePageError && <Button variant="secondary" size="compact" isLoading={loadingMoreResponses} onPress={() => void loadMoreResponses()}>Load more submitted responses</Button>}
    </div>
  </section>;
}

function appendUniqueByID<T extends { id: string }>(current: T[], incoming: T[]) {
  if (incoming.length === 0) return current;
  const seen = new Set(current.map((item) => item.id));
  return [...current, ...incoming.filter((item) => !seen.has(item.id))];
}

function distributionStatusLabel(status: Distribution["status"]) {
  if (status === "READY") return "Ready";
  if (status === "OPEN") return "Awaiting response";
  if (status === "LOCKED") return "In progress";
  if (status === "COMPLETED") return "Completed";
  if (status === "EXPIRED") return "Expired";
  if (status === "REVOKED") return "Revoked";
  if (status === "SUPERSEDED") return "Superseded";
  return "Draft";
}

function distributionTone(status: Distribution["status"]) {
  if (status === "COMPLETED") return "success" as const;
  if (status === "EXPIRED") return "error" as const;
  if (status === "REVOKED" || status === "SUPERSEDED") return "unknown" as const;
  if (status === "OPEN" || status === "LOCKED") return "warning" as const;
  return "neutral" as const;
}

function responseStateLabel(response: CompletedResponseSummary) {
  if (response.state === "PROVISIONAL") return "Review pending";
  if (response.score?.state === "FAILED") return "Assessment unavailable";
  return "Final";
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Unknown date";
  return new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric" }).format(date);
}
