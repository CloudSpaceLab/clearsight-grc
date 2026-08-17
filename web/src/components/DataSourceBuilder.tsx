import { useState } from "react";
import type { FormEvent } from "react";
import { createRESTBinding, prepareRESTSource } from "../sourceConfigApi";
import type { PreparedRESTSource, SourceBinding } from "../sourceConfigApi";

type RuleConfig = { code: string; name: string; claim: string; field: string; expected: string };
type Props = { onSaved: (binding: SourceBinding, config: RuleConfig) => void; onCancel: () => void };

export function DataSourceBuilder({ onSaved, onCancel }: Props) {
  const [prepared, setPrepared] = useState<PreparedRESTSource | null>(null);
  const [sourceName, setSourceName] = useState("");
  const [code, setCode] = useState("");
  const [field, setField] = useState("");
  const [expected, setExpected] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function testEndpoint(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    setBusy(true); setError("");
    try {
      const value = await prepareRESTSource({
        name: sourceName.trim(), code: code.trim().toUpperCase().replace(/\s+/g, "-"),
        endpoint: String(data.get("endpoint") ?? "").trim(), freshnessMinutes: Number(data.get("freshness") ?? 60),
      });
      if (!value.view.native_schema.length) throw new Error("The endpoint returned no fields.");
      setPrepared(value); setField(value.view.native_schema[0]?.name ?? "");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The endpoint could not be tested.");
    } finally { setBusy(false); }
  }

  async function save() {
    if (!prepared || !field || !expected.trim()) {
      setError("Choose a field and enter the value that should be present.");
      return;
    }
    setBusy(true); setError("");
    try {
      const binding = await createRESTBinding(prepared, field);
      onSaved(binding, { code: `${code.trim().toUpperCase().replace(/\s+/g, "-")}-CHECK`, name: sourceName.trim(), claim: `${sourceName.trim()} is present and operating.`, field, expected: expected.trim() });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The source could not be saved.");
    } finally { setBusy(false); }
  }

  return <div className="monitoring-builder data-source-builder">
    <div className="monitoring-builder-heading"><div><span className="eyebrow">Connected data</span><h4>New endpoint check</h4><p>Connect an HTTPS status endpoint and choose the value ClearSight should verify.</p></div></div>
    {!prepared ? <form className="monitoring-form-grid" onSubmit={testEndpoint}>
      <label><span>Source name</span><input value={sourceName} onChange={(event) => setSourceName(event.target.value)} required placeholder="Live face verification"/></label>
      <label><span>Code</span><input value={code} onChange={(event) => setCode(event.target.value)} required placeholder="FACE-SDK"/></label>
      <label className="full"><span>Status endpoint</span><input name="endpoint" type="url" required placeholder="https://status.example.com/mobile/face-verification"/></label>
      <label><span>Maximum age (minutes)</span><input name="freshness" type="number" min="1" max="525600" defaultValue="60" required/></label>
      <div className="monitoring-form-actions full"><button className="text-button" type="button" onClick={onCancel}>Cancel</button><button className="primary-button" type="submit" disabled={busy}>{busy ? "Testing…" : "Test endpoint"}</button></div>
    </form> : <div className="source-field-step">
      <p className="inline-success" role="status">Endpoint reached. {prepared.view.native_schema.length} field{prepared.view.native_schema.length === 1 ? "" : "s"} found.</p>
      <div className="monitoring-form-grid">
        <label><span>Status field</span><select aria-label="Status field" value={field} onChange={(event) => setField(event.target.value)}>{prepared.view.native_schema.map((item) => <option value={item.name} key={item.name}>{item.name}</option>)}</select></label>
        <label><span>Expected value</span><input aria-label="Expected value" value={expected} onChange={(event) => setExpected(event.target.value)} required placeholder="true"/></label>
      </div>
      <div className="monitoring-form-actions"><button className="text-button" type="button" onClick={() => setPrepared(null)}>Change endpoint</button><button className="primary-button" type="button" disabled={busy} onClick={() => void save()}>{busy ? "Saving…" : "Use this source"}</button></div>
    </div>}
    {error && <p className="inline-form-error" role="alert">{error}</p>}
  </div>;
}
