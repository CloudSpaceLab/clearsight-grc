import { useState } from "react";
import {
  loadFormResponseWorkspace,
  normalizeCaptureAnswer,
  saveFormResponseWorkspace,
  submitFormResponseWorkspace,
  uploadCaptureSessionArtifact,
  type FormResponseWorkspace,
  type RedeemedFormAccessSession,
} from "../captureApi";
import { apiErrorKind, ApiError } from "../http";
import type { CaptureAnswerValue, CaptureAnswers, CaptureRequest } from "../types";
import { CapturePanel } from "./CapturePanel";
import { ExternalAccessGate } from "./capture/ExternalAccessGate";

type ExternalCaptureState = "access" | "loading" | "live" | "recoverable" | "terminal" | "submitted";

export function ExternalCaptureApp({ invitationToken }: { invitationToken: string }) {
  const [sessionToken, setSessionToken] = useState("");
  const [request, setRequest] = useState<CaptureRequest | null>(null);
  const [workspace, setWorkspace] = useState<FormResponseWorkspace | null>(null);
  const [audienceHint, setAudienceHint] = useState("");
  const [assurance, setAssurance] = useState("");
  const [state, setState] = useState<ExternalCaptureState>(invitationToken ? "access" : "terminal");
  const [error, setError] = useState(invitationToken ? "" : "Ask the sender for a new invitation link.");
  const [terminalTitle, setTerminalTitle] = useState("This request is no longer available");

  async function openRedeemedSession(redeemed: RedeemedFormAccessSession) {
    setSessionToken(redeemed.session_token);
    setAudienceHint(redeemed.audience_hint);
    setAssurance(redeemed.assurance);
    setState("loading");
    setError("");
    try {
      const payload = await loadFormResponseWorkspace(redeemed.session_token);
      if (!resumableRequest(payload.request) || payload.workspace.workspace.status !== "OPEN") {
        endSession(
          payload.request.status === "EXPIRED" ? "This request has expired" : "This request is no longer available",
          "Ask the sender for a new invitation link.",
        );
        return;
      }
      setRequest(payload.request);
      setWorkspace(payload.workspace);
      setAudienceHint(payload.session.audience_hint || redeemed.audience_hint);
      setAssurance(payload.session.assurance || redeemed.assurance);
      setState("live");
    } catch (cause) {
      if (terminalSessionFailure(cause)) {
        endSession("This request is no longer available", "Ask the sender for a new invitation link.");
        return;
      }
      setError("Check your connection, then try loading this request again.");
      setState("recoverable");
    }
  }

  async function retrySession() {
    if (!sessionToken) {
      endSession("This request is no longer available", "Open the original invitation link again.");
      return;
    }
    setState("loading");
    setError("");
    try {
      const payload = await loadFormResponseWorkspace(sessionToken);
      if (!resumableRequest(payload.request) || payload.workspace.workspace.status !== "OPEN") {
        endSession(
          payload.request.status === "EXPIRED" ? "This request has expired" : "This request is no longer available",
          "Ask the sender for a new invitation link.",
        );
        return;
      }
      setRequest(payload.request);
      setWorkspace(payload.workspace);
      setAudienceHint(payload.session.audience_hint);
      setAssurance(payload.session.assurance);
      setState("live");
    } catch (cause) {
      if (terminalSessionFailure(cause)) {
        endSession("This request is no longer available", "Ask the sender for a new invitation link.");
        return;
      }
      setError("Check your connection, then try loading this request again.");
      setState("recoverable");
    }
  }

  async function submit(answers: CaptureAnswers) {
    if (!workspace || !sessionToken) throw new Error("Response workspace is unavailable");
    try {
      let current = workspace;
      const edits = responseEdits(current, answers);
      if (edits.length > 0) {
        current = await saveFormResponseWorkspace(sessionToken, {
          expected_version: current.workspace.version,
          presentation_mode: current.presentation_mode,
          edits,
        });
        setWorkspace(current);
      }
      const result = await submitFormResponseWorkspace(sessionToken, { expected_version: current.workspace.version });
      clearAuthorityState();
      setState("submitted");
      return result.submission;
    } catch (cause) {
      if (terminalSessionFailure(cause)) {
        endSession("This request is no longer available", "Ask the sender for a new invitation link.");
      }
      throw cause;
    }
  }

  async function upload(file: File, fieldID?: string) {
    try {
      return await uploadCaptureSessionArtifact(sessionToken, file, fieldID);
    } catch (cause) {
      if (terminalSessionFailure(cause)) {
        endSession("This request is no longer available", "Ask the sender for a new invitation link.");
      }
      throw cause;
    }
  }

  function clearAuthorityState() {
    setSessionToken("");
    setRequest(null);
    setWorkspace(null);
    setAudienceHint("");
    setAssurance("");
  }

  function endSession(title: string, message: string) {
    clearAuthorityState();
    setTerminalTitle(title);
    setError(message);
    setState("terminal");
  }

  return <main className="external-capture-shell">
    <header className="external-capture-brand"><div className="brand-mark" aria-label="ClearSight">C</div><div><strong>ClearSight</strong><span>Evidence response</span></div></header>
    {state === "access" ? <ExternalAccessGate routeSelector={invitationToken} onRedeemed={openRedeemedSession}/>
      : state === "loading" ? <section className="external-capture-entry" aria-live="polite" aria-busy="true"><span className="eyebrow">Evidence request</span><h1>Opening request</h1><p>Loading the response workspace…</p></section>
        : state === "recoverable" ? <section className="external-capture-entry" aria-labelledby="external-capture-title"><span className="eyebrow">Evidence request</span><h1 id="external-capture-title">Request could not be loaded</h1><p>{error}</p><button className="primary-button" type="button" onClick={() => void retrySession()}>Try again</button></section>
          : state === "terminal" ? <section className="external-capture-entry" aria-labelledby="external-capture-title"><span className="eyebrow">Evidence request</span><h1 id="external-capture-title">{terminalTitle}</h1><p>{error}</p></section>
            : state === "submitted" ? <section className="external-capture-entry" aria-labelledby="external-capture-title"><span className="eyebrow">Evidence request</span><h1 id="external-capture-title">Submitted</h1><p>Your evidence response was submitted for this request.</p></section>
              : request && workspace && sessionToken ? <section className="external-capture-work">
                <div className="external-session-hint">Opened for {audienceHint || "invited respondent"}{assurance === "EMAIL_VERIFIED" ? " · Email verified" : ""}</div>
                <CapturePanel request={request} external onSubmit={(_, answers) => submit(answers)} onUploadArtifact={(_, file, fieldID) => upload(file, fieldID)}/>
              </section> : null}
  </main>;
}

function responseEdits(workspace: FormResponseWorkspace, answers: CaptureAnswers) {
  return Object.entries(answers).flatMap(([fieldID, value]) => {
    const normalized = normalizeCaptureAnswer(value);
    if (sameAnswer(workspace.answers[fieldID], normalized)) return [];
    return [{
      field_id: fieldID,
      value: normalized,
      base_sequence: workspace.field_sequences[fieldID] ?? 0,
    }];
  });
}

function sameAnswer(left: CaptureAnswerValue | undefined, right: CaptureAnswerValue): boolean {
  return left !== undefined && JSON.stringify(left) === JSON.stringify(right);
}

function resumableRequest(request: CaptureRequest) {
  const deadline = Date.parse(request.deadline);
  return ["READY", "IN_PROGRESS"].includes(request.status) && (!Number.isFinite(deadline) || deadline > Date.now());
}

function terminalSessionFailure(cause: unknown) {
  const kind = apiErrorKind(cause);
  return ["unauthorized", "forbidden", "not_found"].includes(kind)
    || (cause instanceof ApiError && ["request_closed", "workspace_unavailable", "session_unavailable"].includes(cause.code));
}
