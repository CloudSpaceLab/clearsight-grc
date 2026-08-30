import { useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import {
  redeemFormAccess,
  sendFormAccessOTP,
  startFormAccess,
  verifyFormAccessOTP,
  type FormAccessOTPReceipt,
  type FormAccessStart,
  type MaskedFormRecipient,
  type RedeemedFormAccessSession,
} from "../../captureApi";
import { ApiError } from "../../http";

const RESEND_COOLDOWN_SECONDS = 30;

type Phase = "starting" | "select" | "code" | "opening" | "terminal";

type Props = {
  routeSelector: string;
  onRedeemed: (session: RedeemedFormAccessSession) => Promise<void>;
};

export function ExternalAccessGate({ routeSelector, onRedeemed }: Props) {
  const [phase, setPhase] = useState<Phase>("starting");
  const [start, setStart] = useState<FormAccessStart | null>(null);
  const [selectedRecipient, setSelectedRecipient] = useState("");
  const [challenge, setChallenge] = useState<FormAccessOTPReceipt | null>(null);
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [terminal, setTerminal] = useState<{ title: string; recovery: string; canRequestCode?: boolean }>();
  const [resendRemaining, setResendRemaining] = useState(0);
  const startedSelector = useRef("");

  useEffect(() => {
    if (!routeSelector || startedSelector.current === routeSelector) return;
    startedSelector.current = routeSelector;
    setPhase("starting");
    setError("");
    setTerminal(undefined);
    void beginAccess(routeSelector);
  }, [routeSelector]);

  useEffect(() => {
    if (resendRemaining <= 0) return;
    const timer = window.setTimeout(() => setResendRemaining((remaining) => Math.max(0, remaining - 1)), 1000);
    return () => window.clearTimeout(timer);
  }, [resendRemaining]);

  const recipients = useMemo(() => start?.recipients ?? [], [start]);
  const selected = recipients.find((recipient) => recipient.selector_id === selectedRecipient);

  async function beginAccess(selector: string) {
    try {
      const access = await startFormAccess(selector);
      setStart(access);
      if (access.policy === "DIRECT_MAGIC_LINK") {
        setPhase("opening");
        await onRedeemed(await redeemFormAccess(selector));
        return;
      }
      if (access.policy === "DIRECT_LINK_EMAIL_OTP") {
        const recipient = access.recipients?.[0];
        if (!recipient?.selector_id) throw new Error("Recipient unavailable");
        setSelectedRecipient(recipient.selector_id);
        await issueOTP(selector, recipient);
        return;
      }
      if (access.policy === "SHARED_LINK_EMAIL_OTP" && access.recipients?.length) {
        setPhase("select");
        return;
      }
      throw new Error("Access policy unavailable");
    } catch {
      setError("This invitation could not be opened. Ask the sender for a new link.");
      setPhase("terminal");
    }
  }

  async function issueOTP(selector: string, recipient: MaskedFormRecipient) {
    setBusy(true);
    setError("");
    try {
      const receipt = await sendFormAccessOTP(selector, recipient.selector_id);
      setChallenge(receipt);
      setCode("");
      setResendRemaining(RESEND_COOLDOWN_SECONDS);
      setPhase("code");
    } catch {
      setError("A verification code could not be sent. Try again or ask the sender for a new link.");
      if (start?.policy === "SHARED_LINK_EMAIL_OTP") setPhase("select");
      else setPhase("terminal");
    } finally {
      setBusy(false);
    }
  }

  async function selectRecipient(event: FormEvent) {
    event.preventDefault();
    const recipient = recipients.find((candidate) => candidate.selector_id === selectedRecipient);
    if (!recipient) return;
    await issueOTP(routeSelector, recipient);
  }

  async function verify(event: FormEvent) {
    event.preventDefault();
    const normalizedCode = code.trim();
    if (!challenge || !/^\d{6}$/.test(normalizedCode) || busy) return;
    setBusy(true);
    setError("");
    try {
      const redeemed = await verifyFormAccessOTP(routeSelector, challenge.challenge_id, normalizedCode);
      setPhase("opening");
      await onRedeemed(redeemed);
    } catch (cause) {
      if (cause instanceof ApiError && cause.code === "otp_expired") {
        setTerminal({ title: "Verification code expired", recovery: "Request another code to continue.", canRequestCode: true });
        setPhase("terminal");
        setBusy(false);
        return;
      }
      if (cause instanceof ApiError && cause.code === "otp_attempts_exhausted") {
        setTerminal({ title: "Verification attempts used", recovery: "Ask the sender for a new invitation link." });
        setPhase("terminal");
        setBusy(false);
        return;
      }
      setError("That code could not be verified. Check the code and try again.");
      setBusy(false);
    }
  }

  async function resend() {
    if (!selected || resendRemaining > 0 || busy) return;
    await issueOTP(routeSelector, selected);
  }

  if (phase === "starting" || phase === "opening") {
    return <section className="external-capture-entry" aria-live="polite" aria-busy="true">
      <span className="eyebrow">Evidence request</span>
      <h1>{phase === "opening" ? "Opening request" : "Checking invitation"}</h1>
      <p>{phase === "opening" ? "Loading the response workspace…" : "Confirming how this invitation should be opened…"}</p>
    </section>;
  }

  if (phase === "terminal") {
    return <section className="external-capture-entry" aria-labelledby="external-access-title">
      <span className="eyebrow">Evidence request</span>
      <h1 id="external-access-title">{terminal?.title ?? "This request is no longer available"}</h1>
      <p role="alert">{terminal?.recovery ?? (error || "Ask the sender for a new invitation link.")}</p>
      {terminal?.canRequestCode && selected ? <button className="primary-button" type="button" disabled={busy} onClick={() => void issueOTP(routeSelector, selected)}>{busy ? "Sending code…" : "Request another code"}</button> : null}
    </section>;
  }

  if (phase === "select") {
    return <section className="external-capture-entry" aria-labelledby="external-access-title">
      <span className="eyebrow">Email verification</span>
      <h1 id="external-access-title">Choose your invitation</h1>
      <p>Select the masked address the sender used. ClearSight will send a one-time code there.</p>
      <form onSubmit={selectRecipient}>
        <fieldset className="external-recipient-options">
          <legend className="sr-only">Invitation recipient</legend>
          {recipients.map((recipient) => <label key={recipient.selector_id} className="external-recipient-option">
            <input
              type="radio"
              name="recipient"
              value={recipient.selector_id}
              checked={selectedRecipient === recipient.selector_id}
              onChange={() => setSelectedRecipient(recipient.selector_id)}
            />
            <span><strong>{recipient.contact_label || "Invited respondent"}</strong><small>{recipient.hint}</small></span>
          </label>)}
        </fieldset>
        {error && <p className="error-text" role="alert">{error}</p>}
        <button className="primary-button" type="submit" disabled={!selectedRecipient || busy}>{busy ? "Sending code…" : "Send code"}</button>
      </form>
    </section>;
  }

  const hint = challenge?.hint || selected?.hint || "your invited address";
  return <section className="external-capture-entry" aria-labelledby="external-access-title">
    <span className="eyebrow">Email verification</span>
    <h1 id="external-access-title">Enter the verification code</h1>
    <p>We sent a six-digit code to <strong>{hint}</strong>.</p>
    <form onSubmit={verify}>
      <label className="field">
        <span>Verification code</span>
        <input
          value={code}
          onChange={(event) => setCode(event.target.value.replace(/\D/g, "").slice(0, 6))}
          inputMode="numeric"
          autoComplete="one-time-code"
          pattern="[0-9]*"
          maxLength={6}
          aria-describedby="verification-code-help"
        />
      </label>
      <small id="verification-code-help">Enter the code from the verification email. Codes expire after a short time.</small>
      {error && <p className="error-text" role="alert">{error}</p>}
      <div className="external-access-actions">
        <button className="primary-button" type="submit" disabled={!/^\d{6}$/.test(code) || busy}>{busy ? "Checking code…" : "Verify and open"}</button>
        <button className="secondary-button" type="button" onClick={() => void resend()} disabled={resendRemaining > 0 || busy}>
          {resendRemaining > 0 ? `Resend code in ${resendRemaining}s` : "Resend code"}
        </button>
      </div>
    </form>
  </section>;
}
