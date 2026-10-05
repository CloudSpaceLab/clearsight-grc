import { useMemo, useState } from "react";
import type { FormEvent } from "react";
import type { SourceCheckConfig } from "../monitoringApi";
import type { MonitoringDurationUnit, MonitoringMeasurementUnit, SourceOperator } from "../monitoringTypes";
import { createRESTBinding, prepareRESTSource } from "../sourceConfigApi";
import type { PreparedRESTSource, SourceBinding } from "../sourceConfigApi";
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

  const selectedField = useMemo(() => prepared?.view.native_schema.find((item) => item.name === field), [field, prepared]);
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
      });
      if (!value.view.native_schema.length) throw new Error("The endpoint returned no fields.");
      setPrepared(value); setField(value.view.native_schema[0]?.name ?? "");
      setClaim(`${sourceName.trim()} meets the expected condition.`); setOperator("EQUALS"); setDisplayUnit("");
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
    if (!prepared || !field || !expected.trim() || !claim.trim()) { setError("Enter the monitoring statement, field and expected value."); return; }
    if (displayUnit && !numeric) { setError("Native measurements require a numeric source field."); return; }
    if (displayUnit === "MONEY" && !/^[A-Za-z]{3}$/.test(currency.trim())) { setError("Enter a three-letter currency code."); return; }
    const parsedPrecision = Number(precision);
    if (displayUnit && (!Number.isInteger(parsedPrecision) || parsedPrecision < 0 || parsedPrecision > 6)) { setError("Decimal places must be between 0 and 6."); return; }

    setBusy(true); setError("");
    try {
      const binding = await createRESTBinding(prepared, field);
      const measurement = displayUnit ? {
        field, unit: displayUnit,
        ...(displayUnit === "MONEY" ? { currency: currency.trim().toUpperCase() } : {}),
        ...(displayUnit === "DURATION" ? { duration_unit: durationUnit } : {}),
        precision: displayUnit === "COUNT" ? 0 : parsedPrecision,
      } : undefined;
      onSaved(binding, { code: `${code.trim().toUpperCase().replace(/\s+/g, "-")}-CHECK`, name: sourceName.trim(), claim: claim.trim(), field, expected: expected.trim(), operator, measurement });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The source could not be saved.");
    } finally { setBusy(false); }
  }

  return <div className="monitoring-builder data-source-builder">
    <div className="monitoring-builder-heading"><div><span className="eyebrow">Connected data</span><h4>New endpoint check</h4><p>Connect an HTTPS endpoint and choose the condition to verify.</p></div></div>
    {!prepared ? <form className="monitoring-form-grid" onSubmit={testEndpoint}>
      <label><span>Source name</span><input value={sourceName} onChange={(event) => setSourceName(event.target.value)} required placeholder="Mobile channel health"/></label>
      <label><span>Code</span><input value={code} onChange={(event) => setCode(event.target.value)} required placeholder="MOBILE-HEALTH"/></label>
      <label className="full"><span>Status endpoint</span><input name="endpoint" type="url" required placeholder="https://status.example.com/mobile"/></label>
      <label><span>Maximum age (minutes)</span><input name="freshness" type="number" min="1" max="525600" defaultValue="60" required/></label>
      <div className="monitoring-form-actions full"><button className="text-button" type="button" onClick={onCancel}>Cancel</button><button className="primary-button" type="submit" disabled={busy}>{busy ? "Testing…" : "Test endpoint"}</button></div>
    </form> : <div className="source-field-step">
      <Notice tone="success">Endpoint reached. {prepared.view.native_schema.length} field{prepared.view.native_schema.length === 1 ? "" : "s"} found.</Notice>
      <div className="monitoring-form-grid">
        <label className="full"><span>Monitoring statement</span><input aria-label="Monitoring statement" value={claim} onChange={(event) => setClaim(event.target.value)} required/></label>
        <label><span>Observed field</span><select aria-label="Observed field" value={field} onChange={(event) => changeField(event.target.value)}>{prepared.view.native_schema.map((item) => <option value={item.name} key={item.name}>{item.name}</option>)}</select></label>
        <label><span>Condition</span><select aria-label="Condition" value={operator} onChange={(event) => setOperator(event.target.value as SourceOperator)}>{operatorOptions.map((item) => <option value={item.value} key={item.value}>{item.label}</option>)}</select></label>
        <label><span>Expected value</span><input aria-label="Expected value" value={expected} onChange={(event) => setExpected(event.target.value)} required placeholder={numeric ? "99.5" : "true"}/></label>
        {numeric && <label><span>Display value as</span><select aria-label="Display value as" value={displayUnit} onChange={(event) => setDisplayUnit(event.target.value as DisplayUnit)}><option value="">Condition only</option><option value="COUNT">Count</option><option value="PERCENT">Percent</option><option value="DURATION">Duration</option><option value="MONEY">Money</option></select></label>}
        {displayUnit === "MONEY" && <label><span>Currency</span><input aria-label="Currency" value={currency} maxLength={3} onChange={(event) => setCurrency(event.target.value)} placeholder="NGN"/></label>}
        {displayUnit === "DURATION" && <label><span>Duration unit</span><select aria-label="Duration unit" value={durationUnit} onChange={(event) => setDurationUnit(event.target.value as MonitoringDurationUnit)}><option value="SECONDS">Seconds</option><option value="MINUTES">Minutes</option><option value="HOURS">Hours</option><option value="DAYS">Days</option></select></label>}
        {displayUnit && displayUnit !== "COUNT" && <label><span>Decimal places</span><input aria-label="Decimal places" type="number" min="0" max="6" value={precision} onChange={(event) => setPrecision(event.target.value)}/></label>}
      </div>
      <div className="monitoring-form-actions"><button className="text-button" type="button" onClick={() => setPrepared(null)}>Change endpoint</button><button className="primary-button" type="button" disabled={busy} onClick={() => void save()}>{busy ? "Saving…" : "Use this source"}</button></div>
    </div>}
    {error && <Notice tone="error">{error}</Notice>}
  </div>;
}