import { useState, type FormEvent } from "react";
import { validateAPICheckSample, type APICheckResultType } from "../apiCheckSample";
import "./api-question-sample.css";

/** Sample-only validation. No check, endpoint or assessment is created. */
export function APIQuestionSample() {
  const [type, setType] = useState<APICheckResultType>("boolean");
  const [sample, setSample] = useState('{\n  "result": true\n}');
  const [result, setResult] = useState<{ valid: boolean; message: string }>();
  function test(event: FormEvent) {
    event.preventDefault();
    const error = validateAPICheckSample(sample, type);
    setResult(error ? { valid: false, message: error } : { valid: true, message: "Sample format valid." });
  }
  return <section className="v-api-sample" aria-label="Sample response test">
    <h2>Sample response</h2>
    <p>Sample only · Endpoint not contacted</p>
    <form onSubmit={test}>
      <label>Result type<select value={type} onChange={(event) => { setType(event.target.value as APICheckResultType); setResult(undefined); }}><option value="boolean">True / False</option><option value="number">Number</option></select></label>
      <label>Sample response<textarea value={sample} maxLength={65536} spellCheck={false} onChange={(event) => { setSample(event.target.value); setResult(undefined); }}/></label>
      <button type="submit">Test sample</button>
    </form>
    {result && <p role={result.valid ? "status" : "alert"}>{result.message}</p>}
  </section>;
}
