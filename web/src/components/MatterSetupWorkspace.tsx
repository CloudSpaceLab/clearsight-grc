import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";
import { loadProgramSummaries } from "../api";
import { createMatter } from "../continuityCommands";
import type { ProgramSummary } from "../summaryTypes";
import type { MatterAggregate } from "../types";
import { selectedDateEndOfLocalDay } from "../dueDate";

type Props = { onCreated: (aggregate: MatterAggregate) => void; onClose: () => void; initialProgramID?: string };
type ProgramState = "loading" | "live" | "unavailable";

const WORK_TYPES = [
  ["RISK_SITUATION", "Risk issue"],
  ["CONTROL_GAP", "Control gap"],
  ["REGULATORY_CHANGE", "Regulatory change"],
  ["AUDIT_FINDING", "Audit finding"],
  ["SUPERVISORY_FINDING", "Supervisory finding"],
  ["AUTHORITY_REQUEST", "Authority request"],
  ["EXCEPTION", "Exception"],
  ["INCIDENT", "Incident"],
  ["OPERATIONAL_LOSS", "Operational loss"],
  ["DATA_BREACH", "Data breach"],
  ["VENDOR_DEFICIENCY", "Vendor issue"],
  ["CUSTOMER_CONCERN", "Customer concern"],
] as const;

function nonEmptyLines(value: FormDataEntryValue | null) {
  return String(value ?? "").split(/\r?\n/).map((line) => line.trim()).filter(Boolean);
}

export function MatterSetupWorkspace({ onCreated, onClose, initialProgramID = "" }: Props) {
  const [programs, setPrograms] = useState<ProgramSummary[]>([]);
  const [programState, setProgramState] = useState<ProgramState>("loading");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [programID, setProgramID] = useState(initialProgramID);
  const firstField = useRef<HTMLSelectElement>(null);

  useEffect(() => {
    firstField.current?.focus();
    let active = true;
    void loadProgramSummaries({ limit: 50 }).then((page) => {
      if (!active) return;
      setPrograms(page.items.filter((item) => item.program.status !== "RETIRED"));
      setProgramState("live");
    }).catch(() => {
      if (active) setProgramState("unavailable");
    });
    return () => { active = false; };
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    setSaving(true);
    setError("");
    try {
      const created = await createMatter({
        type: String(data.get("type") ?? "RISK_SITUATION"),
        priority: Number(data.get("priority") ?? 3),
        title: String(data.get("title") ?? "").trim(),
        summary: String(data.get("summary") ?? "").trim(),
        affectedArea: String(data.get("affected_area") ?? "").trim(),
        knownInformation: String(data.get("known_information") ?? "").trim() || undefined,
        missingInformation: nonEmptyLines(data.get("missing_information")),
        dueAt: selectedDateEndOfLocalDay(String(data.get("due_date") ?? "")),
        programID: String(data.get("program_id") ?? "").trim() || undefined,
      });
      onCreated(created);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The issue or change could not be created.");
    } finally {
      setSaving(false);
    }
  }

  return <section className="matter-setup-workspace" aria-labelledby="matter-setup-title">
    <div className="setup-heading">
      <div><span className="eyebrow">Issues and changes</span><h2 id="matter-setup-title">New issue or change</h2><p>Record what needs attention and when it is due.</p></div>
      <button className="text-button" type="button" onClick={onClose}>Close</button>
    </div>
    {error && <p className="inline-form-error" role="alert">{error}</p>}
    <form className="setup-form" onSubmit={submit}>
      <div className="monitoring-form-grid">
        <label><span>Work type</span><select ref={firstField} name="type" defaultValue="RISK_SITUATION">{WORK_TYPES.map(([value, label]) => <option value={value} key={value}>{label}</option>)}</select></label>
        <label><span>Priority</span><select name="priority" defaultValue="3"><option value="1">Low</option><option value="2">Normal</option><option value="3">Medium</option><option value="4">High</option><option value="5">Critical</option></select></label>
        <label className="full"><span>Title</span><input name="title" required maxLength={180} placeholder="Face verification is unavailable"/></label>
        <label className="full"><span>What happened or changed?</span><textarea name="summary" required rows={3} placeholder="Describe the issue, change or request that needs attention."/></label>
        <label><span>Affected area</span><input name="affected_area" required placeholder="Mobile banking"/></label>
        <label><span>Due date</span><input name="due_date" type="date"/></label>
        <label className="full"><span>Program (optional)</span><select name="program_id" disabled={programState === "loading"} value={programID} onChange={(event) => setProgramID(event.target.value)}><option value="">No Program link</option>{initialProgramID && !programs.some((item) => item.program.id === initialProgramID) && <option value={initialProgramID}>Current Program</option>}{programs.map((item) => <option value={item.program.id} key={item.program.id}>{item.program.name} ({item.program.code})</option>)}</select></label>
        {programState === "loading" && <p className="field-note full" role="status">Loading Programs…</p>}
        {programState === "unavailable" && <p className="field-note full">Programs could not be loaded. You can create this item and link it later.</p>}
        <label className="full"><span>What is already known?</span><textarea name="known_information" rows={3} placeholder="Add confirmed information that will help the owner start work."/></label>
        <label className="full"><span>What information is still needed?</span><textarea name="missing_information" rows={3} placeholder="Add one item per line."/></label>
      </div>
      <div className="monitoring-form-actions"><button className="primary-button" disabled={saving} type="submit">{saving ? "Creating…" : "Create issue or change"}</button></div>
    </form>
  </section>;
}
