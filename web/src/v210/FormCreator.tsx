import { useEffect, useLayoutEffect, useRef, useState, type FormEvent } from "react";
import { resolveAuthority, type RuntimeContext } from "../api";
import { createLibraryFormDraft } from "../formsApi";
import { ApiError } from "../http";
import type { FormFieldType } from "../monitoringTypes";
import { focusAuthoringQuestion, QuestionOutline } from "./QuestionOutline";
import "./form-creator.css";

type Question = { id: string; label: string; type: FormFieldType; required: boolean; optionsText?: string };
const questionTypes: { value: FormFieldType; label: string }[] = [
  { value: "short_text", label: "Short text" }, { value: "long_text", label: "Long text" },
  { value: "yes_no", label: "Yes / No" }, { value: "date", label: "Date" },
  { value: "decimal", label: "Number" }, { value: "percentage", label: "Percentage" },
  { value: "file", label: "Document" },
  { value: "single_select", label: "Choose one" }, { value: "multi_select", label: "Choose multiple" },
];

export function FormCreator({ context }: { context: RuntimeContext }) {
  const [authority, setAuthority] = useState<"loading" | "allowed" | "denied">("loading");
  const [attempt, setAttempt] = useState(0); const [name, setName] = useState(""); const [purpose, setPurpose] = useState("");
  const [questions, setQuestions] = useState<Question[]>([{ id: "question-1", label: "", type: "short_text", required: true }]);
  const [busy, setBusy] = useState(false); const [outcome, setOutcome] = useState<"unknown" | "created">(); const [error, setError] = useState("");
  const [createdId, setCreatedId] = useState(""); const serial = useRef(1); const lock = useRef(false); const focusNewQuestion = useRef(false);
  useLayoutEffect(() => { if (focusNewQuestion.current) { focusAuthoringQuestion(questions.length - 1); focusNewQuestion.current = false; } }, [questions.length]);
  useEffect(() => {
    let active = true; setAuthority("loading");
    if (!context.actor.id || context.actor.kind?.toUpperCase() === "SERVICE" || !context.legal_entity.id || context.legal_entity.id === "*") { setAuthority("denied"); return; }
    resolveAuthority({ object_type: "LEGAL_ENTITY", object_id: context.legal_entity.id, responsibility: "ACCOUNTABLE_OWNER", decision_type: "forms.template.create", materiality: 2 }).then((result) => {
      if (active) setAuthority(result.principal?.id === context.actor.id || result.candidate_principals?.some((item) => item.id === context.actor.id) ? "allowed" : "denied");
    }).catch(() => { if (active) setAuthority("denied"); });
    return () => { active = false; };
  }, [context.actor.id, context.actor.kind, context.legal_entity.id, attempt]);
  function update(id: string, values: Partial<Question>) { setQuestions((items) => items.map((item) => item.id === id ? { ...item, ...values } : item)); }
  async function submit(event: FormEvent) {
    event.preventDefault(); if (lock.current || authority !== "allowed" || !name.trim() || !purpose.trim() || !questions.length || questions.some((item) => !item.label.trim())) return;
    const tooLong = questions.findIndex((item) => new TextEncoder().encode(item.label.trim()).length > 200);
    if (tooLong >= 0) { setError(`Question ${tooLong + 1} is too long. Shorten the question.`); focusAuthoringQuestion(tooLong); return; }
    const invalidOptions = questions.findIndex((item) => {
      if (item.type !== "single_select" && item.type !== "multi_select") return false;
      const options = (item.optionsText ?? "").split("\n").map((option) => option.trim()).filter(Boolean);
      return options.length < 2 || options.length > 50 || new Set(options).size !== options.length;
    });
    if (invalidOptions >= 0) { setError(`Question ${invalidOptions + 1} needs 2 to 50 distinct options.`); document.getElementById(`v-authoring-options-${invalidOptions}`)?.focus(); return; }
    lock.current = true; setBusy(true); setError("");
    try {
      const result = await createLibraryFormDraft({ code: `FORM-${crypto.randomUUID()}`, name: name.trim(), purpose: purpose.trim(), presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false }, sections: [{ id: "general", title: "Questions" }], fields: questions.map(({ optionsText, ...item }) => ({ ...item, label: item.label.trim(), section_id: "general", ...((item.type === "single_select" || item.type === "multi_select") ? { options: (optionsText ?? "").split("\n").map((option) => option.trim()).filter(Boolean) } : {}) })) });
      if (!result.id || result.status !== "DRAFT" || result.version !== 1 || result.name !== name.trim()) throw new Error("unconfirmed");
      setCreatedId(result.id); setOutcome("created");
    } catch (failure) { if (failure instanceof ApiError && (failure.status === 400 || failure.status === 422)) { setError(failure.message); lock.current = false; } else setOutcome("unknown"); }
    finally { setBusy(false); }
  }
  if (outcome === "created") return <section><h1>Draft created</h1><p>{name}</p><a className="v-primary" href={`#/forms/${encodeURIComponent(createdId)}`}>Open draft</a></section>;
  if (outcome === "unknown") return <section><h1>Draft status unknown</h1><p>Check forms before trying again.</p><div className="v-panel"><h2>Entered details</h2><p>{name}</p><p>{purpose}</p><ul>{questions.map((question) => <li key={question.id}><span>{question.label}</span> · {questionTypes.find((type) => type.value === question.type)?.label}{question.required ? " · Required" : ""}{(question.type === "single_select" || question.type === "multi_select") && <p style={{ whiteSpace: "pre-wrap" }}>{question.optionsText}</p>}</li>)}</ul></div><p><a className="v-primary" href="#/forms" target="_blank" rel="noopener noreferrer">Open forms in new tab</a></p></section>;
  return <section className="v-form-creator"><a className="v-back" href="#/forms">← Forms</a><header className="v-page-heading"><div><p className="v-eyebrow">Forms</p><h1>Create form</h1></div></header>
    {authority === "loading" && <p role="status">Checking access…</p>}
    {authority === "denied" && <div className="v-panel"><p role="alert">Form creation unavailable. Check your role or try again.</p><button onClick={() => setAttempt((value) => value + 1)}>Try again</button></div>}
    {authority === "allowed" && <form onSubmit={(event) => void submit(event)}><QuestionOutline questions={questions}/><section className="v-panel v-create-details"><h2>Form details</h2><label>Form name<input required maxLength={160} value={name} onChange={(event) => setName(event.target.value)}/></label><label>Purpose<textarea required maxLength={1000} value={purpose} onChange={(event) => setPurpose(event.target.value)}/></label></section><section className="v-panel v-create-questions"><h2>Questions</h2>{questions.map((question, index) => <fieldset key={question.id}><legend>Question {index + 1}</legend><label>Question {index + 1}<input id={`v-authoring-question-${index}`} required maxLength={200} value={question.label} onChange={(event) => update(question.id, { label: event.target.value })}/></label><label>Answer type {index + 1}<select value={question.type} onChange={(event) => update(question.id, { type: event.target.value as FormFieldType })}>{questionTypes.map((type) => <option key={type.value} value={type.value}>{type.label}</option>)}</select></label>{(question.type === "single_select" || question.type === "multi_select") && <label>Options {index + 1}<textarea id={`v-authoring-options-${index}`} required aria-label={`Options ${index + 1}`} value={question.optionsText ?? ""} onChange={(event) => update(question.id, { optionsText: event.target.value })}/><small>Enter one option per line.</small></label>}<label className="v-create-required"><input type="checkbox" checked={question.required} onChange={(event) => update(question.id, { required: event.target.checked })}/>Required</label>{questions.length > 1 && <button type="button" onClick={() => setQuestions((items) => items.filter((item) => item.id !== question.id))}>Remove question {index + 1}</button>}</fieldset>)}{questions.length < 200 && <button type="button" onClick={() => { focusNewQuestion.current = true; setQuestions((items) => [...items, { id: `question-${++serial.current}`, label: "", type: "short_text", required: true }]); }}>Add question</button>}</section>{error && <p role="alert">{error}</p>}<div className="v-create-actions"><a href="#/forms">Cancel</a><button className="v-primary" disabled={busy}>{busy ? "Saving…" : "Save draft"}</button></div></form>}
  </section>;
}
