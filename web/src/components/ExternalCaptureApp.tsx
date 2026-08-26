import { useEffect, useState, type FormEvent } from "react";
import { loadCaptureSession, redeemCaptureInvitation, submitCaptureSession, uploadCaptureSessionArtifact } from "../captureApi";
import { EXTERNAL_CAPTURE_SESSION_KEY } from "../captureInvitationBrowser";
import { apiErrorKind, ApiError } from "../http";
import type { CaptureRequest } from "../types";
import { CapturePanel } from "./CapturePanel";

export function ExternalCaptureApp({ invitationToken }: { invitationToken: string }) {
  const [audience, setAudience] = useState("");
  const [sessionToken, setSessionToken] = useState("");
  const [request, setRequest] = useState<CaptureRequest | null>(null);
  const [audienceHint, setAudienceHint] = useState("");
  const [state, setState] = useState<"identify" | "loading" | "live" | "error">(invitationToken ? "identify" : "loading");
  const [error, setError] = useState("");

  useEffect(() => {
    if (invitationToken) {
      sessionStorage.removeItem(EXTERNAL_CAPTURE_SESSION_KEY);
      return;
    }
    const saved = sessionStorage.getItem(EXTERNAL_CAPTURE_SESSION_KEY);
    if (!saved) {
      setState("error");
      return;
    }
    setState("loading");
    void openSession(saved).catch(() => {
      sessionStorage.removeItem(EXTERNAL_CAPTURE_SESSION_KEY);
      setSessionToken("");
      setRequest(null);
      setError("This saved request is no longer available. Ask the sender for a new invitation link.");
      setState("error");
    });
  }, [invitationToken]);

  async function openSession(token: string) {
    const payload = await loadCaptureSession(token);
    setSessionToken(token);
    setRequest(payload.request);
    setAudienceHint(payload.session.audience_hint);
    setState("live");
    if (!resumableRequest(payload.request)) sessionStorage.removeItem(EXTERNAL_CAPTURE_SESSION_KEY);
  }

  async function redeem(event: FormEvent) {
    event.preventDefault();
    const identity = audience.trim();
    if (!identity) return;
    setState("loading");
    setError("");
    try {
      const redeemed = await redeemCaptureInvitation(invitationToken, identity);
      sessionStorage.setItem(EXTERNAL_CAPTURE_SESSION_KEY, redeemed.session_token);
      await openSession(redeemed.session_token);
    } catch {
      sessionStorage.removeItem(EXTERNAL_CAPTURE_SESSION_KEY);
      setSessionToken("");
      setRequest(null);
      setState("error");
      setError("This link could not be opened with that email address or phone number. Check what the sender used, or ask them for a new link.");
    }
  }

  async function submit(answers: Record<string, string>) {
    try {
      const receipt = await submitCaptureSession(sessionToken, request!.version, answers);
      sessionStorage.removeItem(EXTERNAL_CAPTURE_SESSION_KEY);
      return receipt;
    } catch (cause) {
      clearEndedSession(cause);
      throw cause;
    }
  }

  async function upload(file: File) {
    try {
      return await uploadCaptureSessionArtifact(sessionToken, file);
    } catch (cause) {
      clearEndedSession(cause);
      throw cause;
    }
  }

  return <main className="external-capture-shell">
    <header className="external-capture-brand"><div className="brand-mark" aria-label="ClearSight">C</div><div><strong>ClearSight</strong><span>Secure evidence submission</span></div></header>
    {state === "identify" || state === "error" ? <section className="external-capture-entry" aria-labelledby="external-capture-title">
      <span className="eyebrow">Evidence request</span>
      <h1 id="external-capture-title">{invitationToken ? "Open your request" : "This request is no longer available"}</h1>
      <p>{invitationToken ? "Enter the email address or phone number this link was sent to. You do not need a ClearSight account." : error || "Ask the sender for a new invitation link."}</p>
      {invitationToken && <form onSubmit={redeem}>
        <label className="field"><span>Email or phone number</span><input value={audience} autoCapitalize="none" autoCorrect="off" onChange={(event) => setAudience(event.target.value)} placeholder="name@example.com or phone number"/></label>
        {error && <p className="error-text" role="alert">{error}</p>}
        <button className="primary-button" type="submit" disabled={!audience.trim()}>Open request</button>
      </form>}
      {invitationToken && <small>Only the request linked to this invitation will be available.</small>}
    </section> : state === "loading" ? <section className="external-capture-entry" aria-live="polite" aria-busy="true"><span className="eyebrow">Evidence request</span><h1>Opening request</h1><p>Checking the invitation…</p></section> : request && sessionToken ? <section className="external-capture-work"><div className="external-session-hint">Opened for {audienceHint || "invited respondent"}</div><CapturePanel request={request} external onSubmit={(_, answers) => submit(answers)} onUploadArtifact={(_, file) => upload(file)}/></section> : null}
  </main>;
}

function resumableRequest(request: CaptureRequest) {
  const deadline = Date.parse(request.deadline);
  return ["READY", "IN_PROGRESS"].includes(request.status) && (!Number.isFinite(deadline) || deadline > Date.now());
}

function clearEndedSession(cause: unknown) {
  const kind = apiErrorKind(cause);
  if (["unauthorized", "forbidden", "not_found"].includes(kind) || (cause instanceof ApiError && cause.code === "request_closed")) {
    sessionStorage.removeItem(EXTERNAL_CAPTURE_SESSION_KEY);
  }
}
