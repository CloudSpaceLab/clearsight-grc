import { useEffect, useMemo, useState } from "react";
import {
  loadCompletedResponses,
  loadDistributionPage,
  type CompletedResponseSummary,
  type Distribution,
} from "../../formsDistributionApi";
import { ActionLink, Notice, StatusBadge } from "../ui";
import { concernText, concernTone } from "./responseScorePresentation";

type SubjectType = "MATTER" | "PROGRAM" | "VENDOR_RELATIONSHIP";

type LoadState = "loading" | "live" | "unavailable";

type Props = {
  subjectType: SubjectType;
  subjectID: string;
  subjectLabel: string;
  limit?: number;
};

export function SubjectFormActivity({ subjectType, subjectID, subjectLabel, limit = 6 }: Props) {
  const [requestState, setRequestState] = useState<LoadState>("loading");
  const [responseState, setResponseState] = useState<LoadState>("loading");
  const [requests, setRequests] = useState<Distribution[]>([]);
  const [responses, setResponses] = useState<CompletedResponseSummary[]>([]);
  const [requestsMore, setRequestsMore] = useState(false);
  const [responsesMore, setResponsesMore] = useState(false);

  useEffect(() => {
    let active = true;
    setRequestState("loading");
    setResponseState("loading");
    setRequests([]);
    setResponses([]);
    setRequestsMore(false);
    setResponsesMore(false);

    void loadDistributionPage({ subject_type: subjectType, subject_id: subjectID, limit })
      .then((page) => {
        if (!active) return;
        setRequests(page.items);
        setRequestsMore(Boolean(page.next_cursor));
        setRequestState("live");
      })
      .catch(() => {
        if (active) setRequestState("unavailable");
      });

    void loadCompletedResponses({
      subject_type: subjectType,
      subject_id: subjectID,
      current_only: true,
      sort: "COMPLETED_DESC",
      limit,
    }).then((page) => {
      if (!active) return;
      setResponses(page.items);
      setResponsesMore(Boolean(page.next_cursor));
      setResponseState("live");
    }).catch(() => {
      if (active) setResponseState("unavailable");
    });

    return () => { active = false; };
  }, [limit, subjectID, subjectType]);

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
    needsReview ? `${needsReview} submitted response${needsReview === 1 ? "" : "s"} need review` : "",
  ].filter(Boolean);

  return <section className="subject-form-activity" aria-label={`Form activity for ${subjectLabel}`}>
    {attention.length > 0 && <Notice tone="warning">{attention.join(" · ")}.</Notice>}

    <div className="subject-form-activity__group">
      <header><h3>Requests</h3>{requestState === "live" && <span>{requests.length} shown{requestsMore ? " · More available" : ""}</span>}</header>
      {requestState === "loading" && <p role="status">Loading form requests…</p>}
      {requestState === "unavailable" && <Notice tone="warning">Form requests are unavailable. Other issue work remains available.</Notice>}
      {requestState === "live" && requests.length === 0 && <p>No form requests recorded.</p>}
      {requests.length > 0 && <ul>{requests.map((request) => <li key={request.id}>
        <div><strong>{request.title}</strong><span>{distributionStatusLabel(request.status)} · Due {formatDate(request.deadline)}</span></div>
        <StatusBadge tone={distributionTone(request.status)}>{distributionStatusLabel(request.status)}</StatusBadge>
      </li>)}</ul>}
    </div>

    <div className="subject-form-activity__group">
      <header><h3>Submitted responses</h3>{responseState === "live" && <span>{responses.length} shown{responsesMore ? " · More available" : ""}</span>}</header>
      {responseState === "loading" && <p role="status">Loading submitted responses…</p>}
      {responseState === "unavailable" && <Notice tone="warning">Submitted responses are unavailable. Other issue work remains available.</Notice>}
      {responseState === "live" && responses.length === 0 && <p>No submitted responses recorded.</p>}
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
    </div>
  </section>;
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
