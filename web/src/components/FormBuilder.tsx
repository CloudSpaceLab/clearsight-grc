import { useState } from "react";
import type { FormEvent } from "react";
import { createFormTemplate } from "../monitoringApi";
import type { FormTemplate, FormTemplateField } from "../monitoringTypes";

type Props = { onSaved: (form: FormTemplate) => void; onCancel: () => void };

type Question = { id: string; label: string; required: boolean; scored: boolean; weight: number; noScore: number; criticalNo: boolean };

const blankQuestion = (index: number): Question => ({ id: `question_${index}`, label: "", required: true, scored: true, weight: 1, noScore: 100, criticalNo: false });

const passwordResetQuestions = [
  "Was the customer’s identity verified before the reset?",
  "Was the one-time code sent only to a registered channel?",
  "Were changes to recovery details separately authenticated?",
  "Were repeated failed reset attempts blocked or rate-limited?",
  "Were reset events logged and reviewed for unusual activity?",
];

export function FormBuilder({ onSaved, onCancel }: Props) {
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [purpose, setPurpose] = useState("");
  const [questions, setQuestions] = useState<Question[]>([blankQuestion(1)]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  function usePasswordResetReview() {
    setCode("PASSWORD-RESET-REVIEW");
    setName("Password reset security review");
    setPurpose("Confirm that password reset safeguards operated during the reporting period.");
    setQuestions(passwordResetQuestions.map((label, index) => ({ ...blankQuestion(index + 1), id: `reset_${index + 1}`, label, criticalNo: index < 3 })));
    setError("");
  }

  function updateQuestion(index: number, change: Partial<Question>) {
    setQuestions((current) => current.map((question, questionIndex) => questionIndex === index ? { ...question, ...change } : question));
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    if (!code.trim() || !name.trim() || !purpose.trim()) {
      setError("Enter a code, name and purpose.");
      return;
    }
    if (questions.some((question) => !question.label.trim())) {
      setError("Enter every question before saving.");
      return;
    }
    const ids = new Set<string>();
    const fields: FormTemplateField[] = questions.map((question, index) => {
      let id = question.id.trim().toLowerCase().replace(/[^a-z0-9]+/g, "_").replace(/^_|_$/g, "") || `question_${index + 1}`;
      while (ids.has(id)) id = `${id}_${index + 1}`;
      ids.add(id);
      return {
        id, label: question.label.trim(), type: "single_select", required: question.required, options: ["Yes", "No"],
        scoring: question.scored ? { weight: question.weight, answer_scores: { Yes: 0, No: question.noScore }, critical_answers: question.criticalNo ? ["No"] : [] } : undefined,
      };
    });
    setSaving(true);
    try {
      const saved = await createFormTemplate({ code: code.trim().toUpperCase().replace(/\s+/g, "-"), name: name.trim(), purpose: purpose.trim(), fields });
      onSaved(saved);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The form could not be saved. Check the fields and try again.");
    } finally {
      setSaving(false);
    }
  }

  return <form className="monitoring-builder" onSubmit={submit}>
    <div className="monitoring-builder-heading">
      <div><span className="eyebrow">Collection form</span><h4>New collection form</h4><p>Define the questions and how each response affects risk.</p></div>
      <button className="secondary-button" type="button" onClick={usePasswordResetReview}>Use password reset review</button>
    </div>
    <div className="monitoring-form-grid">
      <label><span>Form name</span><input value={name} onChange={(event) => setName(event.target.value)} required/></label>
      <label><span>Code</span><input value={code} onChange={(event) => setCode(event.target.value)} required/></label>
      <label className="full"><span>Purpose</span><textarea value={purpose} onChange={(event) => setPurpose(event.target.value)} rows={2} required/></label>
    </div>
    <fieldset className="question-list">
      <legend>Questions</legend>
      {questions.map((question, index) => <article className="question-editor" key={`${question.id}-${index}`}>
        <div className="question-number">{index + 1}</div>
        <label className="question-label"><span>Question</span><input aria-label="Question" value={question.label} onChange={(event) => updateQuestion(index, { label: event.target.value })} required/></label>
        <label className="compact-control"><input type="checkbox" checked={question.required} onChange={(event) => updateQuestion(index, { required: event.target.checked })}/> Required</label>
        <label className="compact-control"><input type="checkbox" checked={question.scored} onChange={(event) => updateQuestion(index, { scored: event.target.checked })}/> Include in risk score</label>
        {question.scored && <div className="question-scoring">
          <label><span>Weight</span><input type="number" min="1" max="100" value={question.weight} onChange={(event) => updateQuestion(index, { weight: Number(event.target.value) })}/></label>
          <label><span>Risk when No</span><input type="number" min="0" max="100" value={question.noScore} onChange={(event) => updateQuestion(index, { noScore: Number(event.target.value) })}/></label>
          <label className="compact-control"><input type="checkbox" checked={question.criticalNo} onChange={(event) => updateQuestion(index, { criticalNo: event.target.checked })}/> A No answer is critical</label>
        </div>}
        {questions.length > 1 && <button className="text-button danger-text" type="button" onClick={() => setQuestions((current) => current.filter((_, questionIndex) => questionIndex !== index))}>Remove</button>}
      </article>)}
      <button className="secondary-button" type="button" onClick={() => setQuestions((current) => [...current, blankQuestion(current.length + 1)])}>Add question</button>
    </fieldset>
    {error && <p className="inline-form-error" role="alert">{error}</p>}
    <div className="monitoring-form-actions"><button className="text-button" type="button" onClick={onCancel}>Cancel</button><button className="primary-button" type="submit" disabled={saving}>{saving ? "Saving…" : "Save draft"}</button></div>
  </form>;
}
