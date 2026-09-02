import { useMemo, useState } from "react";
import type { FormScoreContribution, FormScorePredicateOperator, FormScoreProfile, FormScoreRule, FormScoringMode, FormTemplateField } from "../../../monitoringTypes";
import { previewFormScore, type FormScorePreview, type FormScorePreviewAnswer } from "../../../formPoliciesApi";
import { Button, CheckboxField, Notice, SelectField, StatusBadge, TextField } from "../../ui";

type ScoredMode = Exclude<FormScoringMode, "NONE">;
type Props = {
  mode: ScoredMode;
  profile?: FormScoreProfile;
  fields: FormTemplateField[];
  templateRevision?: { id: string; version: number };
  onChange: (profile: FormScoreProfile) => void;
};

const bands = [
  { band: "LOW", from: 0, through: 24 }, { band: "MODERATE", from: 25, through: 49 },
  { band: "HIGH", from: 50, through: 74 }, { band: "CRITICAL", from: 75, through: 100 },
] as const;
export function AdvancedScoringEditor({ mode, profile, fields, templateRevision, onChange }: Props) {
  const current = profile ?? defaultProfile(mode);
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [preview, setPreview] = useState<FormScorePreview>();
  const [previewError, setPreviewError] = useState("");
  const [previewing, setPreviewing] = useState(false);
  const scoredFields = useMemo(() => fields.filter((field) => !["file", "photo", "signature", "vendor_document", "attestation"].includes(field.type)), [fields]);

  function updateContribution(index: number, change: Partial<FormScoreContribution>) {
    onChange({ ...current, contributions: current.contributions.map((item, itemIndex) => itemIndex === index ? { ...item, ...change } : item) });
  }
  function addContribution() {
    const source = scoredFields[0];
    if (!source || current.contributions.length >= 100) return;
    const number = current.contributions.length + 1;
    onChange({ ...current, contributions: [...current.contributions, { id: `contribution_${number}`, label: `Score contribution ${number}`, weight: 100, predicate: defaultLeaf(source), match_points: 100, non_match_points: 0, missing: "INDETERMINATE" }] });
  }
  function updateRule(index: number, change: Partial<FormScoreRule>) {
    onChange({ ...current, rules: (current.rules ?? []).map((item, itemIndex) => itemIndex === index ? { ...item, ...change } : item) });
  }
  function addRule() {
    const source = scoredFields[0];
    if (!source || (current.rules?.length ?? 0) >= 100) return;
    const number = (current.rules?.length ?? 0) + 1;
    onChange({ ...current, rules: [...(current.rules ?? []), { id: `rule_${number}`, label: `Advanced rule ${number}`, predicate: defaultLeaf(source), effect: { kind: "FLOOR", value: 75 } }] });
  }
  async function runPreview() {
    if (!templateRevision || previewing) return;
    setPreviewing(true); setPreviewError(""); setPreview(undefined);
    const fieldByID = new Map(scoredFields.map((field) => [field.id, field]));
    const payload = Object.fromEntries(Object.entries(answers).filter(([, value]) => value.trim()).map(([id, value]) => [id, previewAnswer(fieldByID.get(id), value)]));
    try { setPreview(await previewFormScore(templateRevision.id, templateRevision.version, payload)); }
    catch (cause) { setPreviewError(cause instanceof Error ? cause.message : "The score preview could not be calculated."); }
    finally { setPreviewing(false); }
  }

  return <div className="forms-advanced-scoring" aria-label="Advanced scoring rules">
    <div className="forms-advanced-scoring__summary">
      <div><strong>Advanced scoring</strong><p>{mode === "RISK" ? "Higher scores mean greater concern." : "Lower scores mean greater concern."} Rules are evaluated from the stored form revision.</p></div>
      <TextField label="Score profile version" value={current.version} onChange={(version) => onChange({ ...current, version })} maxLength={128}/>
    </div>

    <section aria-labelledby="score-contributions-title">
      <div className="forms-policy-section-heading"><div><h4 id="score-contributions-title">Score contributions</h4><p>Set the points awarded when a response matches one bounded condition.</p></div><Button size="compact" onPress={addContribution} isDisabled={!scoredFields.length || current.contributions.length >= 100}>Add score contribution</Button></div>
      {current.contributions.length === 0 && <p className="forms-policy-muted">Add at least one contribution before this score profile can be saved.</p>}
      {current.contributions.map((contribution, index) => <article className="forms-score-rule" key={contribution.id}>
        <div className="forms-score-rule__heading"><strong>{contribution.label || `Contribution ${index + 1}`}</strong><Button variant="quiet" size="compact" onPress={() => onChange({ ...current, contributions: current.contributions.filter((_, itemIndex) => itemIndex !== index) })}>Remove</Button></div>
        <div className="forms-policy-control-grid">
          <TextField label="Contribution name" value={contribution.label} onChange={(label) => updateContribution(index, { label })}/>
          <div className="forms-policy-control-grid__full"><PredicateEditor label="Contribution condition" predicate={contribution.predicate} fields={scoredFields} onChange={(predicate) => updateContribution(index, { predicate })}/></div>
          <NumberField label="Weight" value={contribution.weight} min={1} max={100} onChange={(weight) => updateContribution(index, { weight })}/>
          <NumberField label="Points when matched" value={contribution.match_points} min={0} max={100} onChange={(match_points) => updateContribution(index, { match_points })}/>
          <NumberField label="Points when not matched" value={contribution.non_match_points} min={0} max={100} onChange={(non_match_points) => updateContribution(index, { non_match_points })}/>
          <SelectField label="Missing answer" value={contribution.missing} placeholder="Choose handling" allowsEmpty={false} options={[{ id: "INDETERMINATE", label: "Mark score incomplete" }, { id: "EXCLUDE", label: "Exclude from coverage" }, { id: "ZERO", label: "Use zero points" }]} onChange={(missing) => { if (missing) updateContribution(index, { missing }); }}/>
        </div>
        <CheckboxField label="This contribution is required for a final score" isSelected={Boolean(contribution.required)} onChange={(required) => updateContribution(index, { required })}/>
      </article>)}
    </section>

    <section aria-labelledby="advanced-rules-title">
      <div className="forms-policy-section-heading"><div><h4 id="advanced-rules-title">Cross-field rules</h4><p>Use a floor, cap, extra contribution or disqualification only when the combined answers require it.</p></div><Button size="compact" onPress={addRule} isDisabled={!scoredFields.length || (current.rules?.length ?? 0) >= 100}>Add advanced rule</Button></div>
      {(current.rules ?? []).map((rule, index) => <article className="forms-score-rule" key={rule.id}>
        <div className="forms-score-rule__heading"><strong>{rule.label || `Rule ${index + 1}`}</strong><Button variant="quiet" size="compact" onPress={() => onChange({ ...current, rules: (current.rules ?? []).filter((_, itemIndex) => itemIndex !== index) })}>Remove</Button></div>
        <div className="forms-policy-control-grid">
          <TextField label="Rule name" value={rule.label} onChange={(label) => updateRule(index, { label })}/>
          <div className="forms-policy-control-grid__full"><PredicateEditor label="Rule condition" predicate={rule.predicate} fields={scoredFields} onChange={(predicate) => updateRule(index, { predicate })}/></div>
          <SelectField label="Score effect" value={rule.effect.kind} placeholder="Choose an effect" allowsEmpty={false} options={[{ id: "FLOOR", label: "Set a minimum score" }, { id: "CAP", label: "Set a maximum score" }, { id: "CONTRIBUTION", label: "Add a weighted contribution" }, { id: "DISQUALIFY", label: "Disqualify the response" }]} onChange={(kind) => { if (kind) updateRule(index, { effect: { kind, ...(kind === "DISQUALIFY" ? {} : { value: rule.effect.value ?? 75 }), ...(kind === "CONTRIBUTION" ? { weight: rule.effect.weight ?? 100 } : {}) } }); }}/>
          {rule.effect.kind !== "DISQUALIFY" && (
            <NumberField label="Effect value" value={rule.effect.value ?? 0} min={0} max={100} onChange={(value) => updateRule(index, { effect: { ...rule.effect, value } })}/>
          )}
          {rule.effect.kind === "CONTRIBUTION" && (
            <NumberField label="Effect weight" value={rule.effect.weight ?? 100} min={1} max={100} onChange={(weight) => updateRule(index, { effect: { ...rule.effect, weight } })}/>
          )}
        </div>
      </article>)}
    </section>

    <section className="forms-score-preview" aria-labelledby="score-preview-title">
      <div className="forms-policy-section-heading"><div><h4 id="score-preview-title">Preview stored revision</h4><p>{templateRevision ? `Test revision ${templateRevision.version} without changing a response or Matter.` : "Save this form as a governed revision before requesting a server score preview."}</p></div></div>
      {templateRevision && <div className="forms-policy-control-grid">{scoredFields.map((field) => <PreviewAnswerField key={field.id} field={field} value={answers[field.id] ?? ""} onChange={(value) => setAnswers((currentAnswers) => ({ ...currentAnswers, [field.id]: value }))}/>)}</div>}
      {previewError && <Notice tone="error">{previewError} Check the stored scoring rules and preview answers, then try again.</Notice>}
      {preview && <div className="forms-score-preview__result"><StatusBadge tone={tone(preview.band)}>{title(preview.band)} concern</StatusBadge><strong>{formatScore(preview, mode)}</strong><span>{Math.round(preview.coverage * 100)}% answer coverage{preview.final ? " · final" : " · incomplete"}</span></div>}
      <Button variant="primary" isDisabled={!templateRevision} isLoading={previewing} onPress={() => void runPreview()}>Preview score</Button>
    </section>
  </div>;
}

function defaultProfile(mode: ScoredMode): FormScoreProfile { return { version: `${mode.toLowerCase()}-v1`, mode, direction: mode === "RISK" ? "HIGH_IS_POOR" : "LOW_IS_POOR", contributions: [], bands: bands.map((band) => ({ ...band })) }; }
function noValue(operator: FormScorePredicateOperator) { return operator === "ANSWERED" || operator === "UNANSWERED"; }
function PredicateEditor({ label, predicate, fields, onChange }: { label: string; predicate: FormScoreContribution["predicate"]; fields: FormTemplateField[]; onChange: (predicate: FormScoreContribution["predicate"]) => void }) {
  const grouped = predicate.operator === "AND" || predicate.operator === "OR";
  const mode = grouped ? predicate.operator === "AND" ? "ALL" : "ANY" : "ONE";
  const children = grouped ? predicate.children ?? [] : [predicate];
  function setMode(next?: string) {
    if (!next) return;
    if (next === "ONE") onChange(children[0] ?? defaultLeaf(fields[0]));
    else {
      const first = children[0] ?? defaultLeaf(fields[0]);
      const second = children[1] ?? defaultLeaf(fields[1] ?? fields[0]);
      onChange({ operator: next === "ALL" ? "AND" : "OR", children: [first, second] });
    }
  }
  return <fieldset className="forms-score-predicate"><legend>{label}</legend>
    <SelectField label={`${label} match`} value={mode} placeholder="Choose how conditions match" allowsEmpty={false} options={[{ id: "ONE", label: "One condition" }, { id: "ALL", label: "All conditions" }, { id: "ANY", label: "Any condition" }]} onChange={setMode}/>
    {children.map((child, index) => <div className="forms-score-predicate__row" key={`${index}:${child.field_id ?? "condition"}`}><LeafPredicateEditor label={grouped ? `Condition ${index + 1}` : label} predicate={child} fields={fields} onChange={(next) => onChange(grouped ? { ...predicate, children: children.map((item, itemIndex) => itemIndex === index ? next : item) } : next)}/>{grouped && children.length > 2 && <Button variant="quiet" size="compact" onPress={() => onChange({ ...predicate, children: children.filter((_, itemIndex) => itemIndex !== index) })}>Remove condition</Button>}</div>)}
    {grouped && <Button size="compact" onPress={() => onChange({ ...predicate, children: [...children, defaultLeaf(fields[Math.min(children.length, fields.length - 1)] ?? fields[0])] })} isDisabled={children.length >= 20}>Add condition</Button>}
  </fieldset>;
}
function LeafPredicateEditor({ label, predicate, fields, onChange }: { label: string; predicate: FormScoreContribution["predicate"]; fields: FormTemplateField[]; onChange: (predicate: FormScoreContribution["predicate"]) => void }) {
  const field = fields.find((candidate) => candidate.id === predicate.field_id) ?? fields[0];
  const operators = predicateOptions(field);
  return <div className="forms-policy-control-grid">
    <SelectField label={`${label} response field`} value={field?.id} placeholder="Choose a response" allowsEmpty={false} options={fields.map((item) => ({ id: item.id, label: item.label || item.id }))} onChange={(field_id) => { const nextField = fields.find((item) => item.id === field_id); if (nextField) onChange(defaultLeaf(nextField)); }}/>
    <SelectField label={`${label} operator`} value={operators.some((item) => item.id === predicate.operator) ? predicate.operator : operators[0]?.id} placeholder="Choose a condition" allowsEmpty={false} options={operators} onChange={(operator) => { if (operator) onChange({ field_id: field?.id, operator, values: noValue(operator) ? undefined : defaultValues(field, operator, predicate.values) }); }}/>
    {!noValue(predicate.operator) && (
      <TextField label={`${label} comparison ${expectsMany(predicate.operator) ? "values" : "value"}`} description={expectsMany(predicate.operator) ? "Separate multiple values with commas." : undefined} value={predicate.values?.join(", ") ?? ""} onChange={(value) => onChange({ ...predicate, values: value.split(",").map((item) => item.trim()).filter(Boolean) })}/>
    )}
  </div>;
}
function defaultLeaf(field?: FormTemplateField): FormScoreContribution["predicate"] { return { field_id: field?.id, operator: "EQUALS", values: [field?.options?.[0] ?? ""] }; }
function defaultValues(field: FormTemplateField | undefined, operator: FormScorePredicateOperator, previous?: string[]) { if (operator === "NUMBER_BETWEEN" || operator === "DATE_BETWEEN") return previous?.length === 2 ? previous : ["", ""]; return previous?.length ? previous : [field?.options?.[0] ?? ""]; }
function expectsMany(operator: FormScorePredicateOperator) { return ["IN", "NOT_IN", "CONTAINS_ANY", "CONTAINS_ALL", "NUMBER_BETWEEN", "DATE_BETWEEN"].includes(operator); }
function predicateOptions(field?: FormTemplateField): Array<{ id: FormScorePredicateOperator; label: string }> {
  const options: Array<{ id: FormScorePredicateOperator; label: string }> = [{ id: "EQUALS", label: "Answer equals" }, { id: "NOT_EQUALS", label: "Answer does not equal" }, { id: "ANSWERED", label: "Has an answer" }, { id: "UNANSWERED", label: "Has no answer" }];
  if (["yes_no", "single_select", "multi_select"].includes(String(field?.type))) options.splice(2, 0, { id: "IN", label: "Answer is one of" }, { id: "NOT_IN", label: "Answer is not one of" });
  if (field?.type === "multi_select") options.splice(2, 0, { id: "CONTAINS", label: "Selections contain" }, { id: "CONTAINS_ANY", label: "Selections contain any" }, { id: "CONTAINS_ALL", label: "Selections contain all" });
  if (["integer", "decimal", "percentage", "currency", "number"].includes(String(field?.type))) options.splice(2, 0, { id: "GREATER_THAN", label: "Number is greater than" }, { id: "GREATER_OR_EQUAL", label: "Number is at least" }, { id: "LESS_THAN", label: "Number is less than" }, { id: "LESS_OR_EQUAL", label: "Number is at most" }, { id: "NUMBER_BETWEEN", label: "Number is between" });
  if (field?.type === "date") options.splice(2, 0, { id: "DATE_BEFORE", label: "Date is before" }, { id: "DATE_ON_OR_AFTER", label: "Date is on or after" }, { id: "DATE_BETWEEN", label: "Date is between" });
  return options;
}
function NumberField({ label, value, min, max, onChange }: { label: string; value: number; min: number; max: number; onChange: (value: number) => void }) { return <TextField label={label} type="number" min={min} max={max} value={String(value)} onChange={(raw) => onChange(Math.min(max, Math.max(min, Number(raw) || min)))}/>; }
function PreviewAnswerField({ field, value, onChange }: { field: FormTemplateField; value: string; onChange: (value: string) => void }) {
  const label = `${field.label || field.id} preview answer`;
  if (field.type === "yes_no" || field.type === "single_select") return <SelectField label={label} value={value || undefined} placeholder="Choose an answer" options={(field.options ?? (field.type === "yes_no" ? ["Yes", "No"] : [])).map((option) => ({ id: option, label: option }))} onChange={(next) => onChange(next ?? "")}/>;
  if (field.type === "multi_select") return <TextField label={label} description="Separate multiple selected answers with commas." value={value} onChange={onChange}/>;
  if (field.type === "date") return <TextField label={label} type="date" value={value} onChange={onChange}/>;
  if (["integer", "decimal", "percentage", "currency", "number"].includes(String(field.type))) return <TextField label={label} type="number" value={value} onChange={onChange}/>;
  return <TextField label={label} value={value} onChange={onChange}/>;
}
function previewAnswer(field: FormTemplateField | undefined, value: string): FormScorePreviewAnswer { return field?.type === "multi_select" ? { values: value.split(",").map((item) => item.trim()).filter(Boolean) } : { text: value.trim() }; }
function title(value: string) { return value.charAt(0) + value.slice(1).toLowerCase(); }
function tone(band: string): "success" | "info" | "warning" | "error" { return band === "LOW" ? "success" : band === "MODERATE" ? "info" : band === "HIGH" ? "warning" : "error"; }
function formatScore(preview: FormScorePreview, mode: ScoredMode) { const score = preview.raw_score; return score === undefined ? "Score incomplete" : `${Math.round(score)} ${mode === "RISK" ? "risk score" : "compliance score"}`; }
