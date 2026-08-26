import { useEffect, useState, type FormEvent } from "react";
import { loadCaptureSession, redeemCaptureInvitation, submitCaptureSession, uploadCaptureSessionArtifact } from "../captureApi";
import type { CaptureRequest } from "../types";
import { CapturePanel } from "./CapturePanel";

export const captureActiveSessionStorageKey = "clearsight.capture.active-session";

export function captureSessionStorageKey(sessionID: string) {
  return `clearsight.capture.session:${sessionID}`;
}

function captureStorage(browserWindow: Window): Storage | undefined {
  try {
    return browserWindow.sessionStorage;
  } catch {
    return undefined;
  }
}

function readCaptureStorage(storage: Storage | undefined, key: string) {
  try {
    return storage?.getItem(key) ?? null;
  } catch {
    return null;
  }
}

function writeCaptureStorage(storage: Storage | undefined, key: string, value: string) {
  try {
    storage?.setItem(key, value);
  } catch {
    // The redeemed request remains usable for this page even when resume storage is blocked.
  }
}

function removeCaptureStorage(storage: Storage | undefined, key: string) {
  try {
    storage?.removeItem(key);
  } catch {
    // Storage cleanup is best effort when the browser blocks access.
  }
}

export function bootstrapExternalCapture(browserWindow: Window) {
  const params = new URLSearchParams(browserWindow.location.search);
  const invitationToken = params.get("capture_invite") || undefined;
  const storage = captureStorage(browserWindow);
  const activeSessionID = readCaptureStorage(storage, captureActiveSessionStorageKey)?.trim() || undefined;
  if (invitationToken && activeSessionID) {
    removeCaptureStorage(storage, captureSessionStorageKey(activeSessionID));
    removeCaptureStorage(storage, captureActiveSessionStorageKey);
  }
  if (params.has("capture_invite")) {
    params.delete("capture_invite");
    if (invitationToken) params.set("capture", "1");
    const search = params.toString();
    browserWindow.history.replaceState(
      browserWindow.history.state,
      "",
      `${browserWindow.location.pathname}${search ? `?${search}` : ""}${browserWindow.location.hash}`,
    );
  }
  const resumedSessionID = invitationToken ? undefined : activeSessionID;
  return {
    invitationToken,
    resumedSessionID,
    isExternalCapture: Boolean(invitationToken || params.get("capture") === "1"),
  };
}

export function ExternalCaptureApp({ invitationToken, resumedSessionID }: { invitationToken?: string; resumedSessionID?: string }) {
  const [audience, setAudience] = useState("");
  const [sessionToken, setSessionToken] = useState("");
  const [request, setRequest] = useState<CaptureRequest | null>(null);
  const [audienceHint, setAudienceHint] = useState("");
  const [state, setState] = useState<"identify" | "loading" | "live" | "error" | "unavailable">(resumedSessionID ? "loading" : invitationToken ? "identify" : "unavailable");
  const [error, setError] = useState("");

  useEffect(() => {
    if (!resumedSessionID) return;
    const storage = captureStorage(window);
    const storageKey = captureSessionStorageKey(resumedSessionID);
    const saved = readCaptureStorage(storage, storageKey);
    if (!saved) {
      if (readCaptureStorage(storage, captureActiveSessionStorageKey) === resumedSessionID) removeCaptureStorage(storage, captureActiveSessionStorageKey);
      setState(invitationToken ? "identify" : "unavailable");
      return;
    }
    setState("loading");
    void openSession(saved, resumedSessionID).catch(() => {
      removeCaptureStorage(storage, storageKey);
      if (readCaptureStorage(storage, captureActiveSessionStorageKey) === resumedSessionID) removeCaptureStorage(storage, captureActiveSessionStorageKey);
      setSessionToken("");
      setState(invitationToken ? "identify" : "unavailable");
    });
  }, [invitationToken, resumedSessionID]);

  async function openSession(token: string, expectedSessionID?: string) {
    const payload = await loadCaptureSession(token);
    if (expectedSessionID && payload.session.id !== expectedSessionID) throw new Error("Capture session changed");
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
      if (!invitationToken) throw new Error("Invitation unavailable");
      const redeemed = await redeemCaptureInvitation(invitationToken, identity);
      const storage = captureStorage(window);
      writeCaptureStorage(storage, captureSessionStorageKey(redeemed.session_id), redeemed.session_token);
      writeCaptureStorage(storage, captureActiveSessionStorageKey, redeemed.session_id);
      await openSession(redeemed.session_token, redeemed.session_id);
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
    </section> : state === "loading" ? <section className="external-capture-entry" aria-live="polite" aria-busy="true"><span className="eyebrow">Verification request</span><h1>Opening request</h1><p>Checking the invitation…</p></section> : state === "unavailable" ? <section className="external-capture-entry" aria-labelledby="external-capture-unavailable"><span className="eyebrow">Verification request</span><h1 id="external-capture-unavailable">Request access unavailable</h1><p>This saved session can no longer open the request. Ask the sender for a new secure link.</p></section> : request && sessionToken ? <section className="external-capture-work"><div className="external-session-hint">Opened for {audienceHint || "invited respondent"}</div><CapturePanel request={request} external sessionToken={sessionToken} onSubmit={(_, answers) => submitCaptureSession(sessionToken, request.version, answers)} onUploadArtifact={(_, file) => uploadCaptureSessionArtifact(sessionToken, file)}/></section> : null}
  </main>;
}
