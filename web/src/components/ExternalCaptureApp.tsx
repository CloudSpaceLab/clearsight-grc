import { useEffect, useState, type FormEvent } from "react";
import { loadCaptureSession, redeemCaptureInvitation, submitCaptureSession, uploadCaptureSessionArtifact } from "../captureApi";
import { clearCaptureSession, readCaptureSession, saveCaptureSession } from "../captureInvitationBrowser";
import { apiErrorKind, ApiError } from "../http";
import type { CaptureRequest } from "../types";
import { CapturePanel } from "./CapturePanel";

type ExternalCaptureState = "identify" | "loading" | "live" | "recoverable" | "terminal" | "submitted";

export function ExternalCaptureApp({ invitationToken }: { invitationToken: string }) {
  const [audience, setAudience] = useState("");
  const [sessionToken, setSessionToken] = useState("");
  const [request, setRequest] = useState<CaptureRequest | null>(null);
  const [audienceHint, setAudienceHint] = useState("");
  const [state, setState] = useState<ExternalCaptureState>(invitationToken ? "identify" : "loading");
  const [error, setError] = useState("");
  const [terminalTitle, setTerminalTitle] = useState("This request is no longer available");

  useEffect(() => {
    if (invitationToken) {
      clearCaptureSession(sessionStorage);
      clearAuthorityState();
      setState("identify");
      return;
    }
    void resumeSavedSession();
  }, [invitationToken]);

  async function openSession(token: string) {
    const payload = await loadCaptureSession(token);
    if (!resumableRequest(payload.request)) {
      endSession(
        payload.request.status === "EXPIRED" ? "This request has expired" : "This request is no longer available",
        "Ask the sender for a new invitation link.",
      );
      return;
    }
    setSessionToken(token);
    setRequest(payload.request);
    setAudienceHint(payload.session.audience_hint);
    setState("live");
  }

  async function resumeSavedSession() {
    const saved = readCaptureSession(sessionStorage);
    if (!saved) {
      endSession("This request is no longer available", "Ask the sender for a new invitation link.");
      return;
    }
    await resumeSession(saved);
  }

  async function resumeSession(token: string) {
    clearAuthorityState();
    setError("");
    setState("loading");
    try {
      await openSession(token);
    } catch (cause) {
      if (terminalSessionFailure(cause)) {
        endSession("This request is no longer available", "Ask the sender for a new invitation link.");
        return;
      }
      clearAuthorityState();
      setError("Check your connection, then try loading this request again.");
      setState("recoverable");
    }
  }

  async function redeem(event: FormEvent) {
    event.preventDefault();
    const identity = audience.trim();
    if (!identity) return;
    setState("loading");
    setError("");
    let sessionToken: string;
    try {
      const redeemed = await redeemCaptureInvitation(invitationToken, identity);
      sessionToken = redeemed.session_token;
    } catch {
      clearCaptureSession(sessionStorage);
      clearAuthorityState();
      setState("identify");
      setError("This link could not be opened with that email address or phone number. Check what the sender used, or ask them for a new link.");
      return;
    }
    saveCaptureSession(sessionStorage, sessionToken);
    await resumeSession(sessionToken);
  }

  async function submit(answers: Record<string, string>) {
    try {
      const receipt = await submitCaptureSession(sessionToken, request!.version, answers);
      clearCaptureSession(sessionStorage);
      clearAuthorityState();
      setState("submitted");
      return receipt;
    } catch (cause) {
      if (terminalSessionFailure(cause)) {
        endSession("This request is no longer available", "Ask the sender for a new invitation link.");
      }
      throw cause;
    }
  }

  async function upload(file: File) {
    try {
      return await uploadCaptureSessionArtifact(sessionToken, file);
    } catch (cause) {
      if (terminalSessionFailure(cause)) {
        endSession("This request is no longer available", "Ask the sender for a new invitation link.");
      }
      throw cause;
    }
  }

  function clearAuthorityState() {
    setAudience("");
    setSessionToken("");
    setRequest(null);
    setAudienceHint("");
  }

  function endSession(title: string, message: string) {
    clearCaptureSession(sessionStorage);
    clearAuthorityState();
    setTerminalTitle(title);
    setError(message);
    setState("terminal");
  }

  return <main className="external-capture-shell">
    <header className="external-capture-brand"><div className="brand-mark" aria-label="ClearSight">C</div><div><strong>ClearSight</strong><span>Evidence response</span></div></header>
    {state === "identify" ? <section className="external-capture-entry" aria-labelledby="external-capture-title">
      <span className="eyebrow">Evidence request</span>
      <h1 id="external-capture-title">Open your request</h1>
      <p>Enter the email address or phone number this link was sent to. You do not need a ClearSight account.</p>
      <form onSubmit={redeem}>
        <label className="field"><span>Email or phone number</span><input value={audience} autoCapitalize="none" autoCorrect="off" onChange={(event) => setAudience(event.target.value)} placeholder="name@example.com or phone number"/></label>
        {error && <p className="error-text" role="alert">{error}</p>}
        <button className="primary-button" type="submit" disabled={!audience.trim()}>Open request</button>
      </form>
      <small>Only the request linked to this invitation will be available.</small>
    </section> : state === "loading" ? <section className="external-capture-entry" aria-live="polite" aria-busy="true"><span className="eyebrow">Evidence request</span><h1>Opening request</h1><p>Checking the invitation…</p></section>
      : state === "recoverable" ? <section className="external-capture-entry" aria-labelledby="external-capture-title"><span className="eyebrow">Evidence request</span><h1 id="external-capture-title">Request could not be loaded</h1><p>{error}</p><button className="primary-button" type="button" onClick={() => void resumeSavedSession()}>Try again</button></section>
        : state === "terminal" ? <section className="external-capture-entry" aria-labelledby="external-capture-title"><span className="eyebrow">Evidence request</span><h1 id="external-capture-title">{terminalTitle}</h1><p>{error}</p></section>
          : state === "submitted" ? <section className="external-capture-entry" aria-labelledby="external-capture-title"><span className="eyebrow">Evidence request</span><h1 id="external-capture-title">Submitted</h1><p>Your evidence response was submitted for this request.</p></section>
            : request && sessionToken ? <section className="external-capture-work"><div className="external-session-hint">Opened for {audienceHint || "invited respondent"}</div><CapturePanel request={request} external onSubmit={(_, answers) => submit(answers)} onUploadArtifact={(_, file) => upload(file)}/></section> : null}
  </main>;
}

function resumableRequest(request: CaptureRequest) {
  const deadline = Date.parse(request.deadline);
  return ["READY", "IN_PROGRESS"].includes(request.status) && (!Number.isFinite(deadline) || deadline > Date.now());
}

function terminalSessionFailure(cause: unknown) {
  const kind = apiErrorKind(cause);
  return ["unauthorized", "forbidden", "not_found"].includes(kind) || (cause instanceof ApiError && cause.code === "request_closed");
}
