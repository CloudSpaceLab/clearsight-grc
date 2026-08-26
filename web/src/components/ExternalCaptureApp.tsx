import { useEffect, useState, type FormEvent } from "react";
import { loadCaptureSession, redeemCaptureInvitation, submitCaptureSession, uploadCaptureSessionArtifact } from "../captureApi";
import type { CaptureRequest } from "../types";
import { CapturePanel } from "./CapturePanel";

export function ExternalCaptureApp({ invitationToken }: { invitationToken: string }) {
  const [audience, setAudience] = useState("");
  const [sessionToken, setSessionToken] = useState("");
  const [request, setRequest] = useState<CaptureRequest | null>(null);
  const [audienceHint, setAudienceHint] = useState("");
  const [state, setState] = useState<"identify" | "loading" | "live" | "error">("identify");
  const [error, setError] = useState("");

  const storageKey = `clearsight.capture.session:${invitationToken.slice(-12)}`;

  useEffect(() => {
    const saved = sessionStorage.getItem(storageKey);
    if (!saved) return;
    setState("loading");
    void openSession(saved).catch(() => {
      sessionStorage.removeItem(storageKey);
      setSessionToken("");
      setState("identify");
    });
  }, [storageKey]);

  async function openSession(token: string) {
    const payload = await loadCaptureSession(token);
    setSessionToken(token);
    setRequest(payload.request);
    setAudienceHint(payload.session.audience_hint);
    setState("live");
  }

  async function redeem(event: FormEvent) {
    event.preventDefault();
    const identity = audience.trim();
    if (!identity) return;
    setState("loading");
    setError("");
    try {
      const redeemed = await redeemCaptureInvitation(invitationToken, identity);
      sessionStorage.setItem(storageKey, redeemed.session_token);
      await openSession(redeemed.session_token);
    } catch {
      setState("error");
      setError("This link could not be opened with that email address or phone number. Check what the sender used, or ask them for a new link.");
    }
  }

  return <main className="external-capture-shell">
    <header className="external-capture-brand"><div className="brand-mark" aria-label="ClearSight">C</div><div><strong>ClearSight</strong><span>Secure verification</span></div></header>
    {state === "identify" || state === "error" ? <section className="external-capture-entry" aria-labelledby="external-capture-title">
      <span className="eyebrow">Verification request</span>
      <h1 id="external-capture-title">Open your request</h1>
      <p>Enter the email address or phone number this link was sent to. You do not need a ClearSight account.</p>
      <form onSubmit={redeem}>
        <label className="field"><span>Email or phone number</span><input value={audience} autoCapitalize="none" autoCorrect="off" onChange={(event) => setAudience(event.target.value)} placeholder="name@example.com or phone number"/></label>
        {error && <p className="error-text" role="alert">{error}</p>}
        <button className="primary-button" type="submit" disabled={!audience.trim()}>Open request</button>
      </form>
      <small>Only the request linked to this invitation will be available.</small>
    </section> : state === "loading" ? <section className="external-capture-entry" aria-live="polite" aria-busy="true"><span className="eyebrow">Verification request</span><h1>Opening request</h1><p>Checking the invitation…</p></section> : request && sessionToken ? <section className="external-capture-work"><div className="external-session-hint">Opened for {audienceHint || "invited respondent"}</div><CapturePanel request={request} external sessionToken={sessionToken} onSubmit={(_, answers) => submitCaptureSession(sessionToken, request.version, answers)} onUploadArtifact={(_, file) => uploadCaptureSessionArtifact(sessionToken, file)}/></section> : null}
  </main>;
}
