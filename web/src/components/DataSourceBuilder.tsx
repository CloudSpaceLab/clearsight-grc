import { useMemo, useState } from "react";
import type { FormEvent } from "react";
import type { SourceCheckConfig } from "../monitoringApi";
import type { MonitoringDurationUnit, MonitoringMeasurementUnit, SourceOperator } from "../monitoringTypes";
import { createRESTBinding, prepareRESTSource } from "../sourceConfigApi";
import type { PreparedRESTSource, SourceBinding } from "../sourceConfigApi";
import { validateAPICheckSample } from "../apiCheckSample";
import type { APICheckResultType } from "../apiCheckSample";
import { Notice } from "./ui";

type Props = { onSaved: (binding: SourceBinding, config: SourceCheckConfig) => void; onCancel: () => void };
type DisplayUnit = "" | MonitoringMeasurementUnit;

const numericOperators: ReadonlyArray<{ value: SourceOperator; label: string }> = [
  { value: "GREATER_OR_EQUAL", label: "At least" },
  { value: "GREATER_THAN", label: "Above" },
  { value: "LESS_OR_EQUAL", label: "At most" },
  { value: "LESS_THAN", label: "Below" },
  { value: "EQUALS", label: "Equals" },
  { value: "NOT_EQUALS", label: "Does not equal" },
];
const scalarOperators: ReadonlyArray<{ value: SourceOperator; label: string }> = [
  { value: "EQUALS", label: "Equals" },
  { value: "NOT_EQUALS", label: "Does not equal" },
];

export function DataSourceBuilder({ onSaved, onCancel }: Props) {
  const [prepared, setPrepared] = useState<PreparedRESTSource | null>(null);
  const [sourceName, setSourceName] = useState("");
  const [code, setCode] = useState("");
  const [field, setField] = useState("");
  const [expected, setExpected] = useState("");
  const [claim, setClaim] = useState("");
  const [operator, setOperator] = useState<SourceOperator>("EQUALS");
  const [displayUnit, setDisplayUnit] = useState<DisplayUnit>("");
  const [currency, setCurrency] = useState("NGN");
  const [durationUnit, setDurationUnit] = useState<MonitoringDurationUnit>("MINUTES");
  const [precision, setPrecision] = useState("2");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [sampleType, setSampleType] = useState<APICheckResultType>("boolean");
  const [responseFormat, setResponseFormat] = useState<"standard" | "records">("standard");
  const [sample, setSample] = useState('{\n  "result": true,\n  "message": "Check passed"\n}');
  const [sampleResult, setSampleResult] = useState<string | null>(null);
  const [draftSaved, setDraftSaved] = useState(false);

  const selectedField = useMemo(() => prepared?.view.native_schema.find((item) => item.name === field), [field, prepared]);
  const draftOnly = prepared?.connection.status !== "ACTIVE";
  const numeric = selectedField?.native_type === "json:number";
  const operatorOptions = numeric ? numericOperators : scalarOperators;

  async function testEndpoint(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    setBusy(true); setError("");
    try {
      const value = await prepareRESTSource({
        name: sourceName.trim(), code: code.trim().toUpperCase().replace(/\s+/g, "-"),
        endpoint: String(data.get("endpoint") ?? "").trim(), freshnessMinutes: Number(data.get("freshness") ?? 60),
        ...(responseFormat === "standard" ? { resultType: sampleType === "boolean" ? "BOOLEAN" as const : "NUMBER" as const } : {}),
      });
      if (!value.view.native_schema.length) throw new Error("The endpoint returned no fields.");
      if (responseFormat === "standard") {
        const resultField = value.view.native_schema.find((item) => item.name === "result");
        const expectedType = sampleType === "boolean" ? "json:boolean" : "json:number";
        if (value.resultType !== (sampleType === "boolean" ? "BOOLEAN" : "NUMBER") || !resultField || resultField.native_type !== expectedType || resultField.nullable) throw new Error("The endpoint response does not match the selected result type.");
      }
      setPrepared(value); setField(responseFormat === "standard" ? "result" : value.view.native_schema[0]?.name ?? "");
      setExpected(""); setClaim(`${sourceName.trim()} meets the expected condition.`); setOperator("EQUALS"); setDisplayUnit("");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The endpoint could not be tested.");
    } finally { setBusy(false); }
  }

  function changeField(next: string) {
    setField(next);
    const type = prepared?.view.native_schema.find((item) => item.name === next)?.native_type;
    if (type !== "json:number") { setOperator("EQUALS"); setDisplayUnit(""); }
  }

  async function save() {
    if (!prepared || !field) { setError("Select a source field."); return; }
    if (!Number.isSafeInteger(prepared.source.expected_freshness_minutes) || prepared.source.expected_freshness_minutes < 1 || prepared.source.expected_freshness_minutes > 525600) { setError("Source maximum age is unavailable. Test the endpoint again."); return; }
    if (!draftOnly && (!expected.trim() || !claim.trim())) { setError("Enter the monitoring statement and expected value."); return; }
    if (!draftOnly && prepared.resultType === "NUMBER" && (!/^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?$/.test(expected.trim()) || !Number.isFinite(Number(expected.trim())))) { setError("Enter a valid numeric value."); return; }
    if (!draftOnly && prepared.resultType === "BOOLEAN" && !["true", "false"].includes(expected.trim())) { setError("Choose yes or no."); return; }
    if (displayUnit && !numeric) { setError("Native measurements require a numeric source field."); return; }
    if (displayUnit === "MONEY" && !/^[A-Za-z]{3}$/.test(currency.trim())) { setError("Enter a three-letter currency code."); return; }
    const parsedPrecision = Number(precision);
    if (displayUnit && (!Number.isInteger(parsedPrecision) || parsedPrecision < 0 || parsedPrecision > 6)) { setError("Decimal places must be between 0 and 6."); return; }

    setBusy(true); setError("");
    try {
      const binding = await createRESTBinding(prepared, field);
      if (binding.status === "DRAFT") { setDraftSaved(true); return; }
      if (binding.status !== "ACTIVE" || draftOnly) throw new Error("The source binding is not active. The check was not saved.");
      const measurement = displayUnit ? {
        field, unit: displayUnit,
        ...(displayUnit === "MONEY" ? { currency: currency.trim().toUpperCase() } : {}),
        ...(displayUnit === "DURATION" ? { duration_unit: durationUnit } : {}),
        precision: displayUnit === "COUNT" ? 0 : parsedPrecision,
      } : undefined;
      onSaved(binding, { code: `${code.trim().toUpperCase().replace(/\s+/g, "-")}-CHECK`, name: sourceName.trim(), claim: claim.trim(), field, expected: expected.trim(), operator, freshnessMinutes: prepared.source.expected_freshness_minutes, measurement });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The source could not be saved.");
    } finally { setBusy(false); }
  }

  return <div className="monitoring-builder data-source-builder">
    <div className="monitoring-builder-heading"><div><span className="eyebrow">Connected data</span><h4>New endpoint check</h4></div></div>
    {draftSaved ? <div className="source-field-step"><Notice tone="warning">Source draft saved. The check condition has not been saved. Source approval is unavailable.</Notice><div className="monitoring-form-actions"><button className="primary-button" type="button" onClick={onCancel}>Done</button></div></div> : !prepared ? <form className="monitoring-form-grid" onSubmit={testEndpoint}>
      <label><span>Response format</span><select value={responseFormat} onChange={(event) => { setResponseFormat(event.target.value as "standard" | "records"); setSampleResult(null); }}><option value="standard">Standard result</option><option value="records">Other record fields</option></select></label>
      {responseFormat === "standard" && <>
      <label><span>Sample result type</span><select value={sampleType} onChange={(event) => { const type = event.target.value as APICheckResultType; setSampleType(type); setSample(type === "boolean" ? '{\n  "result": true,\n  "message": "Check passed"\n}' : '{\n  "result": 72,\n  "message": "Latest score"\n}'); setSampleResult(null); }}><option value="boolean">Yes or no</option><option value="number">Numeric score</option></select></label>
      <label className="full"><span>Sample response</span><textarea rows={4} value={sample} onChange={(event) => { setSample(event.target.value); setSampleResult(null); }}/></label>
      <div className="full"><button className="text-button" type="button" onClick={() => setSampleResult(validateAPICheckSample(sample, sampleType) ?? "Sample valid. Endpoint not contacted.")}>Test sample</button>{sampleResult && <p role="status">{sampleResult}</p>}<p>Messages are not recorded.</p></div>
      </>}
      <label><span>Source name</span><input value={sourceName} onChange={(event) => setSourceName(event.target.value)} required placeholder="Mobile channel health"/></label>
      <label><span>Code</span><input value={code} onChange={(event) => setCode(event.target.value)} required placeholder="MOBILE-HEALTH"/></label>
      <label className="full"><span>Status endpoint</span><input name="endpoint" type="url" required placeholder="https://status.example.com/mobile"/></label>
      <label><span>Maximum age (minutes)</span><input name="freshness" type="number" min="1" max="525600" defaultValue="60" required/></label>
      <p className="full">Unauthenticated HTTPS endpoints only. Testing creates a draft source and contacts the endpoint.</p>
      <div className="monitoring-form-actions full"><button className="text-button" type="button" onClick={onCancel}>Cancel</button><button className="primary-button" type="submit" disabled={busy}>{busy ? "Testing…" : "Test endpoint"}</button></div>
    </form> : <div className="source-field-step">
      {draftOnly ? <Notice tone="warning">Endpoint reached. Source approval is unavailable. The check condition cannot be saved or run yet.</Notice> : <Notice tone="success">Endpoint reached.</Notice>}
      <div className="monitoring-form-grid">
        {!prepared.resultType && <label><span>Observed field</span><select aria-label="Observed field" value={field} onChange={(event) => changeField(event.target.value)}>{prepared.view.native_schema.map((item) => <option value={item.name} key={item.name}>{item.name}</option>)}</select></label>}
        {!draftOnly && <>
        <label className="full"><span>Monitoring statement</span><input aria-label="Monitoring statement" value={claim} onChange={(event) => setClaim(event.target.value)} required/></label>
        <label><span>Condition</span><select aria-label="Condition" value={operator} onChange={(event) => setOperator(event.target.value as SourceOperator)}>{operatorOptions.map((item) => <option value={item.value} key={item.value}>{item.label}</option>)}</select></label>
        <label><span>Expected value</span>{prepared.resultType === "BOOLEAN" ? <select aria-label="Expected value" value={expected} onChange={(event) => setExpected(event.target.value)} required><option value="">Select value</option><option value="true">Yes</option><option value="false">No</option></select> : <input aria-label="Expected value" value={expected} onChange={(event) => setExpected(event.target.value)} required placeholder={numeric ? "99.5" : "true"}/>}</label>
        {numeric && <label><span>Display value as</span><select aria-label="Display value as" value={displayUnit} onChange={(event) => setDisplayUnit(event.target.value as DisplayUnit)}><option value="">Condition only</option><option value="COUNT">Count</option><option value="PERCENT">Percent</option><option value="DURATION">Duration</option><option value="MONEY">Money</option></select></label>}
        {displayUnit === "MONEY" && <label><span>Currency</span><input aria-label="Currency" value={currency} maxLength={3} onChange={(event) => setCurrency(event.target.value)} placeholder="NGN"/></label>}
        {displayUnit === "DURATION" && <label><span>Duration unit</span><select aria-label="Duration unit" value={durationUnit} onChange={(event) => setDurationUnit(event.target.value as MonitoringDurationUnit)}><option value="SECONDS">Seconds</option><option value="MINUTES">Minutes</option><option value="HOURS">Hours</option><option value="DAYS">Days</option></select></label>}
        {displayUnit && displayUnit !== "COUNT" && <label><span>Decimal places</span><input aria-label="Decimal places" type="number" min="0" max="6" value={precision} onChange={(event) => setPrecision(event.target.value)}/></label>}
        </>}
      </div>
      <div className="monitoring-form-actions"><button className="text-button" type="button" onClick={() => setPrepared(null)}>Change endpoint</button><button className="primary-button" type="button" disabled={busy} onClick={() => void save()}>{busy ? "Saving…" : "Save draft"}</button></div>
    </div>}
    {error && <Notice tone="error">{error}</Notice>}
  </div>;
}
