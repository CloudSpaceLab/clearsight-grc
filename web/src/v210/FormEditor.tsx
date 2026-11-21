import { useEffect, useLayoutEffect, useRef, useState, type FormEvent } from "react";
import { createLibraryFormRevision, loadFormTemplateItem } from "../formsApi";
import { loadContext } from "../api";
import type { FormLibraryItem } from "../formsTypes";
import type { FormTemplateField } from "../monitoringTypes";
import { ApiError } from "../http";
import { focusAuthoringQuestion, QuestionOutline } from "./QuestionOutline";
import "./form-creator.css";

const newQuestionTypes = [
  { value: "short_text", label: "Short text" }, { value: "long_text", label: "Long text" }, { value: "yes_no", label: "Yes / No" },
  { value: "date", label: "Date" }, { value: "decimal", label: "Number" }, { value: "percentage", label: "Percentage" }, { value: "file", label: "Document" },
  { value: "single_select", label: "Choose one" }, { value: "multi_select", label: "Choose multiple" },
] as const;

export function FormEditor({ id }: { id: string }) {
  const [item, setItem] = useState<FormLibraryItem>(); const [state, setState] = useState<"loading" | "ready" | "unavailable" | "saved" | "unknown">("loading");
  const [attempt, setAttempt] = useState(0); const [name, setName] = useState(""); const [purpose, setPurpose] = useState(""); const [fields, setFields] = useState<FormTemplateField[]>([]);
  const [added, setAdded] = useState<Set<string>>(new Set());
  const [optionDrafts, setOptionDrafts] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false); const [error, setError] = useState(""); const lock = useRef(false); const focusNewQuestion = useRef(false); const movedQuestion = useRef<string | undefined>(undefined);
  useLayoutEffect(() => { if (focusNewQuestion.current) { focusAuthoringQuestion(fields.length - 1); focusNewQuestion.current = false; } }, [fields.length]);
  useLayoutEffect(() => { if (movedQuestion.current) { const index = fields.findIndex((field) => field.id === movedQuestion.current); if (index >= 0) focusAuthoringQuestion(index); movedQuestion.current = undefined; } }, [fields]);
  useEffect(() => {
    let active = true; setState("loading");
    Promise.all([loadFormTemplateItem(id), loadContext()]).then(([value, context]) => {
      if (!active) return;
      if (!context.actor.id || context.actor.kind?.toUpperCase() === "SERVICE" || value.template.id !== id || !value.authority_available || !value.operations?.some((operation) => operation.command === "forms.template.revise" && operation.can_act) || !value.template.presentation || !Array.isArray(value.template.sections) || !Array.isArray(value.template.fields)) { setState("unavailable"); return; }
      setItem(value); setName(value.template.name); setPurpose(value.template.purpose); setFields(value.template.fields); setAdded(new Set()); setState("ready");
    }).catch(() => { if (active) setState("unavailable"); });
    return () => { active = false; };
  }, [id, attempt]);
  function addQuestion() {
    const section = item?.template.sections?.[0]; if (!section || fields.length >= 200) return;
    const field: FormTemplateField = { id: `question-${crypto.randomUUID()}`, section_id: section.id, label: "", type: "short_text", required: true };
    focusNewQuestion.current = true; setFields((items) => [...items, field]); setAdded((ids) => new Set(ids).add(field.id));
  }
  function updateField(fieldID: string, patch: Partial<FormTemplateField>) { setFields((items) => items.map((field) => field.id === fieldID ? { ...field, ...patch } : field)); }
  function movedFields(index: number, direction: -1 | 1) {
    const field = fields[index];
    if (!field || !item) return undefined;
    let other = index + direction;
    while (other >= 0 && other < fields.length && fields[other]?.section_id !== field.section_id) other += direction;
    if (other < 0 || other >= fields.length) return undefined;
    const next = [...fields];
    [next[index], next[other]] = [next[other]!, next[index]!];
    const positions = new Map(next.map((question, position) => [question.id, position]));
    const conditions = new Map(item.template.sections.map((section) => [section.id, section.condition]));
    const valid = next.every((question, position) => [question.condition, conditions.get(question.section_id ?? "")].every((condition) => !condition || (positions.has(condition.field_id) && positions.get(condition.field_id)! < position)));
    return valid ? next : undefined;
  }
  function canChangeType(field: FormTemplateField) {
    const simpleKeys = new Set(["id", "section_id", "label", "type", "required", "description"]);
    return (!item?.template.scoring_mode || item.template.scoring_mode === "NONE") && !item?.template.score_profile
      && newQuestionTypes.some((type) => type.value === field.type)
      && Object.entries(field).every(([key, value]) => simpleKeys.has(key) || (key === "constraints" && value && typeof value === "object" && Object.keys(value).length === 0) || (key === "collection_intent" && value === "CAPTURE") || (key === "browser_cache_policy" && value === "ALLOWED") || (key === "options" && field.type === "yes_no" && Array.isArray(value) && value.length === 2 && value[0] === "Yes" && value[1] === "No"))
      && !fields.some((value) => value.condition?.field_id === field.id)
      && !item?.template.sections.some((section) => section.condition?.field_id === field.id);
  }
  function changeType(field: FormTemplateField, type: FormTemplateField["type"]) {
    const next = { ...field, type };
    if (field.type === "yes_no" && type !== "yes_no") delete next.options;
    setFields((items) => items.map((value) => value.id === field.id ? next : value));
  }
  async function submit(event: FormEvent) {
    event.preventDefault(); if (!item || lock.current || !name.trim() || !purpose.trim() || fields.some((field) => !field.label.trim())) return;
    const tooLong = fields.findIndex((field) => new TextEncoder().encode(field.label.trim()).length > 200);
    if (tooLong >= 0) { setError(`Question ${tooLong + 1} is too long. Shorten the question.`); focusAuthoringQuestion(tooLong); return; }
    const invalidOptions = fields.findIndex((field) => {
      if (!added.has(field.id) || (field.type !== "single_select" && field.type !== "multi_select")) return false;
      const options = (optionDrafts[field.id] ?? "").split("\n").map((value) => value.trim()).filter(Boolean);
      return options.length < 2 || options.length > 50 || new Set(options).size !== options.length;
    });
    if (invalidOptions >= 0) { setError(`Question ${invalidOptions + 1} needs 2 to 50 distinct options.`); document.getElementById(`v-authoring-options-${invalidOptions}`)?.focus(); return; }
    const template = item.template; lock.current = true; setBusy(true); setError("");
    try {
      const result = await createLibraryFormRevision(id, template.version, { code: template.code, name: name.trim(), purpose: purpose.trim(), program_id: template.program_id, origin: template.origin, owner_principal_id: template.owner_principal_id, responsible_team: template.responsible_team, approved_uses: template.approved_uses, tags: template.tags, jurisdiction: template.jurisdiction, industry: template.industry, sensitivity: template.sensitivity, scoring_mode: template.scoring_mode, score_profile: template.score_profile, next_review_at: template.next_review_at, presentation: template.presentation, sections: template.sections, fields: fields.map((field) => ({ ...field, label: field.label.trim(), ...((added.has(field.id) && (field.type === "single_select" || field.type === "multi_select")) ? { options: (optionDrafts[field.id] ?? "").split("\n").map((value) => value.trim()).filter(Boolean) } : {}) })) });
      if (result.id !== id || result.version !== template.version + 1 || result.status !== "DRAFT") throw new Error("unconfirmed");
      setState("saved");
    } catch (failure) { if (failure instanceof ApiError && (failure.status === 400 || failure.status === 422)) { setError(failure.message); lock.current = false; } else setState("unknown"); }
    finally { setBusy(false); }
  }
  if (state === "loading") return <p role="status">Loading form…</p>;
  if (state === "unavailable") return <section><h1>Form editing unavailable</h1><p>Check the form and your role.</p><button onClick={() => setAttempt((value) => value + 1)}>Try again</button><p><a href={`#/forms/${encodeURIComponent(id)}`}>View form</a></p></section>;
  if (state === "saved") return <section><h1>Draft saved</h1><a className="v-primary" href={`#/forms/${encodeURIComponent(id)}`}>View draft</a></section>;
  if (state === "unknown") return <section><h1>Draft status unknown</h1><p>Check the form before trying again.</p><div className="v-panel"><h2>Entered details</h2><p>{name}</p><p>{purpose}</p><ul>{fields.map((field) => <li key={field.id}><span>{field.label}</span> · {field.type.replaceAll("_", " ")}{field.required ? " · Required" : ""}{added.has(field.id) && (field.type === "single_select" || field.type === "multi_select") && <p style={{ whiteSpace: "pre-wrap" }}>{optionDrafts[field.id]}</p>}</li>)}</ul></div><p><a className="v-primary" href={`#/forms/${encodeURIComponent(id)}`} target="_blank" rel="noopener noreferrer">Open form in new tab</a></p></section>;
  return <section className="v-form-creator"><a className="v-back" href={`#/forms/${encodeURIComponent(id)}`}>← Form</a><header className="v-page-heading"><div><p className="v-eyebrow">Version {item?.template.version}</p><h1>Edit form</h1></div></header><form onSubmit={(event) => void submit(event)}><QuestionOutline questions={fields}/><section className="v-panel v-create-details"><h2>Form details</h2><label>Form name<input required maxLength={160} value={name} onChange={(event) => setName(event.target.value)}/></label><label>Purpose<textarea required maxLength={1000} value={purpose} onChange={(event) => setPurpose(event.target.value)}/></label></section><section className="v-panel v-create-questions"><h2>Questions</h2><p id="v-order-help">Questions stay within their section and follow their source questions.</p>{fields.length === 0 && <p>No questions recorded.</p>}{fields.map((field, index) => <fieldset key={field.id}><legend>Question {index + 1}</legend><label>Question {index + 1}<input id={`v-authoring-question-${index}`} required maxLength={200} value={field.label} onChange={(event) => updateField(field.id, { label: event.target.value })}/></label>{(added.has(field.id) || canChangeType(field)) && <label>Answer type {index + 1}<select value={field.type} onChange={(event) => changeType(field, event.target.value as FormTemplateField["type"])}>{newQuestionTypes.filter((type) => added.has(field.id) || (type.value !== "single_select" && type.value !== "multi_select")).map((type) => <option key={type.value} value={type.value}>{type.label}</option>)}</select></label>}{added.has(field.id) && <>{(field.type === "single_select" || field.type === "multi_select") && <label>Options {index + 1}<textarea id={`v-authoring-options-${index}`} required aria-label={`Options ${index + 1}`} value={optionDrafts[field.id] ?? ""} onChange={(event) => setOptionDrafts((items) => ({ ...items, [field.id]: event.target.value }))}/><small>Enter one option per line.</small></label>}<label>Section {index + 1}<select value={field.section_id} onChange={(event) => updateField(field.id, { section_id: event.target.value })}>{item?.template.sections.map((section) => <option key={section.id} value={section.id}>{section.title}</option>)}</select></label><label className="v-create-required"><input type="checkbox" checked={field.required} onChange={(event) => updateField(field.id, { required: event.target.checked })}/>Required</label><button type="button" onClick={() => { setFields((items) => items.filter((value) => value.id !== field.id)); setAdded((ids) => { const next = new Set(ids); next.delete(field.id); return next; }); }}>Remove question {index + 1}</button></>}<div className="v-question-order"><button type="button" aria-describedby="v-order-help" aria-label={`Move question ${index + 1}: ${field.label || "Untitled question"} up`} title={movedFields(index, -1) ? undefined : "Cannot move before the first question or its source."} disabled={busy || !movedFields(index, -1)} onClick={() => { const next = movedFields(index, -1); if (next) { movedQuestion.current = field.id; setFields(next); } }}>Move up</button><button type="button" aria-describedby="v-order-help" aria-label={`Move question ${index + 1}: ${field.label || "Untitled question"} down`} title={movedFields(index, 1) ? undefined : "Cannot move after the last question or a dependent question."} disabled={busy || !movedFields(index, 1)} onClick={() => { const next = movedFields(index, 1); if (next) { movedQuestion.current = field.id; setFields(next); } }}>Move down</button></div></fieldset>)}{item?.template.sections.length && fields.length < 200 && (!item.template.scoring_mode || item.template.scoring_mode === "NONE") && !item.template.score_profile ? <button type="button" onClick={addQuestion}>Add question</button> : null}</section>{error && <p role="alert">{error}</p>}<div className="v-create-actions"><a href={`#/forms/${encodeURIComponent(id)}`}>Cancel</a><button className="v-primary" disabled={busy}>{busy ? "Saving…" : "Save draft"}</button></div></form></section>;
}
