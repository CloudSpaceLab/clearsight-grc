import { useEffect, useState } from "react";
import { submitCaptureRequest } from "../api";
import type { CaptureRequest } from "../types";
import { PremiumIllustration } from "./PremiumIllustration";

export function CapturePanel({ request }: { request: CaptureRequest | null }) {
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [reviewing, setReviewing] = useState(false);
  const [receipt, setReceipt] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    setAnswers({});
    setReviewing(false);
    setReceipt(null);
    setError(null);
    setSubmitting(false);
  }, [request?.id]);

  async function submit() {
    if (!request || submitting) return;
    setError(null);
    setSubmitting(true);
    try {
      const result = await submitCaptureRequest(request.id, request.version, answers, request.source);
      setReceipt(`Submitted ${new Date(result.submitted_at).toLocaleString()}`);
      setReviewing(false);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Submission failed");
    } finally {
      setSubmitting(false);
    }
  }

  if (!request) {
    return <div className="panel-content"><span className="eyebrow">Evidence request</span><h2>No request is available</h2><p>Open a live evidence request from Today or Work. Reference requests are available only when stakeholder demo mode is enabled.</p></div>;
  }

  if (receipt) {
    return <div className="panel-content"><span className="eyebrow">Submission receipt</span><h2>Response submitted</h2><PremiumIllustration variant="empty"/><p>{receipt}</p><p>The response has been recorded. It will still be checked against the evidence requirements.</p></div>;
  }

  const requiredMissing = request.fields.some((field) => field.required && !(answers[field.id] ?? "").trim());
  if (reviewing) {
    return <div className="panel-content response-review"><span className="eyebrow">Review response</span><h2>Confirm the assertions you are submitting</h2><p>{request.title}</p><dl className="review-assertions">{request.fields.map((field) => <div key={field.id}><dt>{field.label}</dt><dd>{answers[field.id]?.trim() || "Not provided"}</dd></div>)}</dl><details className="review-context"><summary>Review existing information and purpose</summary><p>{request.purpose}</p><dl className="known-facts">{Object.entries(request.known_facts).map(([key, value]) => <div key={key}><dt>{humanize(key)}</dt><dd>{value}</dd></div>)}</dl><p>Due {new Date(request.deadline).toLocaleString()} · {humanize(request.sensitivity)}</p></details>{error && <p className="error-text" role="alert">{error}</p>}<div className="wizard-actions"><button className="secondary-button" type="button" onClick={() => setReviewing(false)} disabled={submitting}>Edit response</button><button className="primary-button" type="button" onClick={() => void submit()} disabled={submitting}>{submitting ? "Submitting…" : "Submit response"}</button></div></div>;
  }

  return <div className="panel-content"><span className="eyebrow">Evidence request · about {request.estimated_minutes} minutes</span><h2>{request.title}</h2><p>{request.purpose}</p><div className="why-you"><strong>Why you received this</strong><span>{request.why_you}</span></div><h3>Information already available</h3><dl className="known-facts">{Object.entries(request.known_facts).map(([key, value]) => <div key={key}><dt>{humanize(key)}</dt><dd>{value}</dd></div>)}</dl>{request.fields.map((field) => <label className="field" key={field.id}><span>{field.label}{field.required ? " *" : ""}</span>{field.type === "single_select" ? <select value={answers[field.id] ?? ""} onChange={(event) => setAnswers({ ...answers, [field.id]: event.target.value })}><option value="">Select one</option>{field.options?.map((option) => <option key={option}>{option}</option>)}</select> : <textarea value={answers[field.id] ?? ""} onChange={(event) => setAnswers({ ...answers, [field.id]: event.target.value })} placeholder={field.description}/>}</label>)}{error && <p className="error-text" role="alert">{error}</p>}<div className="wizard-actions"><button className="primary-button" type="button" onClick={() => setReviewing(true)} disabled={requiredMissing}>Review response</button></div></div>;
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
