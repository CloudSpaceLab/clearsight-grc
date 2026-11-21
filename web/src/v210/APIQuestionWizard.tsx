import { useRef, useState } from "react";
import { validateAPICheckSample } from "../apiCheckSample";
import { requestJSON } from "../http";
import "./api-question-wizard.css";

type ResultType = "BOOLEAN" | "NUMBER";
type TestReply = { mode: "SAMPLE" | "LIVE"; outcome: "VALID" | "INVALID" | "UNAVAILABLE"; result_type: ResultType; checked_at: string };
type Feedback = { kind: "status" | "alert"; message: string };
const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

/** A format test is neither an integration nor an assessment result. */
export function APIQuestionWizard({ formID, version, fieldID, label, resultType }: { formID: string; version: number; fieldID: string; label: string; resultType: ResultType }) {
  const [step, setStep] = useState<"closed" | "contract" | "test">("closed");
  const example = resultType === "BOOLEAN" ? '{\n  "result": true,\n  "message": "Optional details"\n}' : '{\n  "result": 0,\n  "message": "Optional details"\n}';
  const contract = [
    "Respond to HTTPS GET with a 2xx status and Content-Type application/json or another +json media type.",
    `Return a UTF-8 JSON object no larger than 65,536 bytes. The required result must be ${resultType === "BOOLEAN" ? "true or false" : "a finite number"}.`,
    "The optional message must be text of at most 1,000 characters and contain no null character. Other fields and duplicate fields are not accepted.",
  ];
  const [sample, setSample] = useState(example); const [endpoint, setEndpoint] = useState("");
  const [feedback, setFeedback] = useState<Feedback>(); const [busy, setBusy] = useState(false); const lock = useRef(false);
  const copyGeneration = useRef(0);

  async function copyInstructions() {
    const generation = ++copyGeneration.current;
    const instructions = `Question: ${label}\n\n${contract.join("\n")}\n\n${example}\n\nTests accept public, unauthenticated endpoints only. Private endpoints and credentials are unavailable.`;
    try {
      await navigator.clipboard.writeText(instructions);
      if (generation === copyGeneration.current) setFeedback({ kind: "status", message: "Instructions copied." });
    } catch {
      if (generation === copyGeneration.current) setFeedback({ kind: "alert", message: "Copy unavailable. Select and copy the instructions above." });
    }
  }

  async function test(mode: "SAMPLE" | "LIVE") {
    if (lock.current) return;
    let payload: { mode: "SAMPLE"; result_type: ResultType; sample: unknown } | { mode: "LIVE"; result_type: ResultType; endpoint: string };
    if (mode === "SAMPLE") {
      const error = validateAPICheckSample(sample, resultType === "BOOLEAN" ? "boolean" : "number");
      if (error) { setFeedback({ kind: "alert", message: error }); return; }
      payload = { mode, result_type: resultType, sample: JSON.parse(sample) as unknown };
    } else {
      try {
        const url = new URL(endpoint);
        if (url.protocol !== "https:" || url.username || url.password || endpoint.includes("?") || endpoint.includes("#")) throw new Error();
      } catch { setFeedback({ kind: "alert", message: "Enter an HTTPS endpoint without credentials, query parameters or a fragment." }); return; }
      payload = { mode, result_type: resultType, endpoint };
    }
    lock.current = true; setBusy(true); setFeedback(undefined);
    try {
      const result = await requestJSON<TestReply>(apiBase, `/api/v1/forms/templates/${encodeURIComponent(formID)}/revisions/${version}/questions/${encodeURIComponent(fieldID)}/test`, { method: "POST", body: JSON.stringify(payload) });
      if (result.mode !== mode || result.result_type !== resultType || !["VALID", "INVALID", "UNAVAILABLE"].includes(result.outcome) || !Number.isFinite(Date.parse(result.checked_at))) throw new Error("Invalid test response");
      if (result.outcome === "VALID") setFeedback({ kind: "status", message: mode === "SAMPLE" ? "Sample format valid. Endpoint not contacted." : `Endpoint response format valid. Tested ${result.checked_at}.` });
      else setFeedback({ kind: "alert", message: result.outcome === "UNAVAILABLE" ? "Endpoint unavailable. Check access and try again." : "Response format invalid. Check the result and message fields." });
    } catch { setFeedback({ kind: "alert", message: "Test unavailable. Check access and try again." }); }
    finally { lock.current = false; setBusy(false); }
  }

  if (step === "closed") return <button className="v-api-wizard-launch" type="button" onClick={() => setStep("contract")}>Prepare API question</button>;
  return <section className="v-api-wizard" aria-label={`API question: ${label}`}>
    <h3>API question</h3><p>{label}</p>
    <p>API setup cannot be saved or activated yet. Answers remain manual.</p>
    {step === "contract" ? <>
      <h4>Instructions for developers</h4>
      <div>{contract.map((instruction) => <p key={instruction}>{instruction}</p>)}</div>
      <pre>{example}</pre>
      <p>Tests accept public, unauthenticated endpoints only. Private endpoints and credentials are unavailable.</p>
      <button type="button" onClick={() => void copyInstructions()}>Copy developer instructions</button>
      {feedback && <p role={feedback.kind}>{feedback.message}</p>}
      <div className="v-api-wizard-actions"><button type="button" onClick={() => { copyGeneration.current++; setFeedback(undefined); setStep("closed"); }}>Close</button><button className="v-primary" type="button" onClick={() => { copyGeneration.current++; setFeedback(undefined); setStep("test"); }}>Continue</button></div>
    </> : <>
      <h4>Test response format</h4>
      <label>Sample response<textarea value={sample} disabled={busy} maxLength={65536} spellCheck={false} onChange={(event) => { setSample(event.target.value); setFeedback(undefined); }}/></label>
      <button type="button" disabled={busy} onClick={() => void test("SAMPLE")}>Test sample</button>
      <label>HTTPS endpoint<input type="url" value={endpoint} disabled={busy} onChange={(event) => { setEndpoint(event.target.value); setFeedback(undefined); }}/></label>
      <button type="button" disabled={busy || !endpoint.trim()} onClick={() => void test("LIVE")}>Test endpoint</button>
      {feedback && <p role={feedback.kind}>{feedback.message}</p>}
      <div className="v-api-wizard-actions"><button type="button" disabled={busy} onClick={() => { setFeedback(undefined); setStep("contract"); }}>Back</button><button type="button" disabled={busy} onClick={() => setStep("closed")}>Close</button></div>
    </>}
  </section>;
}
