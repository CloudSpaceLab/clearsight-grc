import { assessmentStateLabels } from "../../vendorFormPresentation";
import { useEffect, useRef, useState } from "react";
import { loadResponseAssessment, recordResponseAssessment, type ResponseAssessmentDetail, type ResponseFieldAssessment } from "../../formAssessmentApi";
import type { ResponseScore } from "../../formsDistributionApi";
import { coverageText, scorePresentation } from "./responseScorePresentation";
import { ApiError } from "../../http";
import { DocumentBrowser } from "../documents/DocumentBrowser";
import { Button, EmptyState, FocusedSheet, Notice, SelectField, StatusBadge, TextArea } from "../ui";
import { fieldAssessmentLabel, needsBankReview } from "./fieldAssessment";
import { assessmentConcernPoints, assessmentConcernThreshold, automaticFieldResults } from "./assessmentFieldResults";
import "./field-assessment.css";

type Props = { responseID: string; onUpdated?: () => void; showResponseContext?: boolean; submissionScore?: ResponseScore };
type Judgement = { outcome_id: string; rationale: string };
type ReviewFilter = "ALL" | "PENDING" | "POOR" | "MISSING" | "REVIEWED";


export function ResponseAssessment(props: Props) { return <AssessmentContent key={props.responseID} {...props}/>; }

function AssessmentContent({ responseID, onUpdated, showResponseContext = true, submissionScore }: Props) {
  const [detail, setDetail] = useState<ResponseAssessmentDetail>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const [conflict, setConflict] = useState(false);
  const [receipt, setReceipt] = useState<string>();
  const [drafts, setDrafts] = useState<Record<string, Judgement>>({});
  const [filter, setFilter] = useState<ReviewFilter>("ALL");
  const [documentsOpen, setDocumentsOpen] = useState(false);
  const sequence = useRef(0);

  async function reload() {
    const request = ++sequence.current;
    setLoading(true);
    setError(undefined);
    try {
      const result = await loadResponseAssessment(responseID);
      if (request !== sequence.current) return;
      setDetail(result);
      setConflict(false);
    } catch (cause) {
      if (request !== sequence.current) return;
      setDetail(undefined);
      setError(message(cause, "The assessment could not be loaded. Retry to read this submitted response."));
    } finally { if (request === sequence.current) setLoading(false); }
  }

  useEffect(() => { void reload(); return () => { sequence.current++; }; }, []);

  async function save() {
    if (!detail || saving || conflict) return;
    const request = sequence.current;
    setSaving(true);
    setError(undefined);
    setReceipt(undefined);
    try {
      const result = await recordResponseAssessment(responseID, { expected_version: detail.version, decisions: changes.map(([field_id, value]) => ({ field_id, outcome_id: value.outcome_id, rationale: value.rationale.trim() })) });
      if (request !== sequence.current) return;
      setDetail(result);
      setDrafts({});
      setReceipt("Assessment saved.");
      try { onUpdated?.(); } catch { setReceipt("Assessment saved. Refresh the vendor list to update its summary."); }
    } catch (cause) {
      if (request !== sequence.current) return;
      if (cause instanceof ApiError && cause.kind === "conflict") setConflict(true);
      if (cause instanceof ApiError && (cause.kind === "forbidden" || cause.kind === "unauthorized")) setDetail(undefined);
      setError(message(cause, "The assessment was not saved. Your judgement and rationale are retained for retry."));
    } finally { if (request === sequence.current) setSaving(false); }
  }

  if (!detail) return <section className="response-assessment" aria-label="Assessment">
    {loading ? <p role="status">Loading assessment…</p> : <><Notice tone="error">{error ?? "The assessment is unavailable."}</Notice><Button onPress={() => void reload()}>Reload assessment</Button></>}
    {submissionScore && <><AssessmentScore title="Automatic result" score={submissionScore}/><RuleExplanation score={submissionScore}/></>}
    <Button onPress={() => setDocumentsOpen(true)}>View submitted documents</Button>
    {documentsOpen && <FocusedSheet label="Submitted assessment evidence" size="wide" onClose={() => setDocumentsOpen(false)}><DocumentBrowser responseRevisionID={responseID} scopeLabel="Documents for this submitted response"/></FocusedSheet>}
  </section>;
  const fields = detail.fields ?? [];
  const reviewable = fields.filter((item) => needsBankReview(item.field) && item.may_review === true);
  const editable = detail.current && detail.may_review === true;
  const changes = Object.entries(drafts).filter(([id, value]) => {
    const item = reviewable.find((candidate) => candidate.field.id === id);
    return item && (value.outcome_id !== item.decision?.outcome_id || value.rationale.trim() !== item.decision?.rationale);
  });
  const valid = changes.length > 0 && changes.every(([id, value]) => value.rationale.trim() && reviewable.find((item) => item.field.id === id)?.field.assessment?.rubric?.some((outcome) => outcome.id === value.outcome_id));
  const pending = Math.max(0, detail.required_count - detail.reviewed_required_count);
  const visible = fields.filter((item) => matchesFilter(item, filter, detail));
  const hasEvidence = fields.some((item) => evidenceField(item));

  return <section className="response-assessment" aria-label="Assessment">
    <header><h3>Assessment</h3><StatusBadge tone={pending > 0 ? "warning" : "neutral"}>{assessmentStateLabels[detail.state] ?? "Assessment state unavailable"}</StatusBadge></header>
    {showResponseContext && <p>Form revision {detail.form_template_version} · {detail.current ? "Current response" : "Historical response"}</p>}<p>Assessment version {detail.version}</p>
    {!detail.current && <Notice tone="info">Historical response. Review the current submission to record a new decision; previous decisions remain in history.</Notice>}
    {detail.state !== "NOT_REQUIRED" && detail.may_review !== true && detail.current && <Notice tone="info">Review permission is unavailable for this response under your current responsibility. Reload the assessment after your responsibility changes.</Notice>}
    {pending > 0 && <Notice tone="warning">{`${pending} required ${pending === 1 ? "field" : "fields"} awaiting review`}</Notice>}
    <p>{detail.reviewed_required_count} of {detail.required_count} required fields reviewed</p>
    <div className="response-assessment__scores"><AssessmentScore title="Automatic result" score={detail.automatic_score}/>{detail.state !== "NOT_REQUIRED" && <AssessmentScore title="Reviewed result" score={detail.assessed_score} provisional={pending > 0 || detail.assessed_score?.state === "PROVISIONAL"}/>}</div>
    {receipt && <Notice tone="success">{receipt}</Notice>}
    {error && <Notice tone="error">{error}</Notice>}
    {conflict && <Notice tone="warning">The response or assessment changed. Reload it and compare the saved decisions with your retained judgement before saving again.</Notice>}
    {conflict && <Button onPress={() => void reload()}>Reload assessment</Button>}
    <SelectField label="Assessment fields" value={filter} placeholder="Choose fields to review" allowsEmpty={false} options={[{ id: "ALL", label: "All submitted fields" }, { id: "PENDING", label: "Awaiting review" }, { id: "POOR", label: "Poor results" }, { id: "MISSING", label: "Missing evidence" }, { id: "REVIEWED", label: "Reviewed" }]} onChange={(value) => { if (value) setFilter(value); }}/>
    {filter === "POOR" && <p>Showing automatic or reviewed fields with at least {assessmentConcernThreshold(detail)} concern points{detail.score_profile ? " under the saved concern bands" : " using the stated 50-point filter"}, plus matched critical overrides. Shared conditions can identify more than one question.</p>}
    {visible.length === 0 && <EmptyState population="Fields in this submitted response matching the selected filter" title="No matching fields" description="Choose All submitted fields to review the other answers and evidence."/>}
    <div className="response-assessment__fields">{visible.map((item) => {
      const { field, decision } = item;
      const automatic = automaticFieldResults(item, detail);
      const value = drafts[field.id] ?? { outcome_id: decision?.outcome_id ?? "", rationale: decision?.rationale ?? "" };
      const update = (patch: Partial<Judgement>) => setDrafts((current) => ({ ...current, [field.id]: { ...value, ...patch } }));
      return <article className="response-assessment__field" key={field.id} aria-label={field.label}>
        <header><h4>{field.label}</h4><span>{fieldAssessmentLabel(field)}{field.assessment?.required && needsBankReview(field) ? " · Required review" : ""}</span></header>
        <div className="response-assessment__field-columns">
          <div><h5>Submitted answer and evidence</h5><p className="response-assessment__answer">{answerText(item)}</p>{item.answer?.document && <dl><div><dt>Document type</dt><dd>{item.answer.document.document_type}</dd></div>{item.answer.document.issued_by && <div><dt>Issued by</dt><dd>{item.answer.document.issued_by}</dd></div>}{item.answer.document.expires_on && <div><dt>Expires</dt><dd>{item.answer.document.expires_on}</dd></div>}</dl>}</div>
          <div className="response-assessment__judgement">
            {automatic.length > 0 && <div><h5>Automatic result</h5>{automatic.map((result) => <div key={result.id}><p><strong>{result.label}</strong> · {result.description}</p>{result.shared && <p>This result depends on answers to more than one question.</p>}</div>)}</div>}
            {decision && <div><h5>Saved decision</h5><p>{field.assessment?.rubric?.find((outcome) => outcome.id === decision.outcome_id)?.label ?? "Saved outcome"} · {decision.points} points</p><p>{decision.rationale}</p><p>{decision.reviewer_display_name ? `Reviewed by ${decision.reviewer_display_name}` : "Reviewer name unavailable"} · <time dateTime={decision.assessed_at}>{formatTime(decision.assessed_at)}</time></p><details><summary>Review details</summary><dl><div><dt>Reviewer ID</dt><dd>{decision.reviewer_id}</dd></div></dl></details></div>}
            {needsBankReview(field) && editable && item.may_review === true && <>
              <SelectField label={`Decision for ${field.label}`} value={value.outcome_id || undefined} placeholder="Choose a rubric outcome" options={(field.assessment?.rubric ?? []).map((outcome) => ({ id: outcome.id, label: `${outcome.label} · ${outcome.points} points` }))} isDisabled={saving} onChange={(outcome_id) => update({ outcome_id: outcome_id ?? "" })}/>
              <TextArea label={`Rationale for ${field.label}`} value={value.rationale} onChange={(rationale) => update({ rationale })} maxLength={4000} rows={3} isDisabled={saving} description="Explain the evidence supporting this decision."/>
              {field.assessment?.mode === "AUTOMATIC_REVIEW" && <p>The decision can confirm or increase the automatic concern. It cannot lower it; critical overrides and scoring limits still apply.</p>}
            </>}
            {needsBankReview(field) && !decision && (!editable || item.may_review !== true) && <p>No decision is recorded for this response version. Your current responsibility does not allow review of this field.</p>}
          </div>
        </div>
      </article>;
    })}</div>
    {hasEvidence && <Button onPress={() => setDocumentsOpen(true)}>View submitted documents</Button>}
    <RuleExplanation score={detail.automatic_score}/>
    {editable && reviewable.length > 0 && <div className="response-assessment__actions"><p>{conflict ? "Reload the assessment before saving." : changes.length === 0 ? "Choose a rubric outcome and enter a rationale to save a decision." : !valid ? "Each changed judgement needs an approved outcome and a rationale." : `${changes.length} ${changes.length === 1 ? "decision" : "decisions"} ready to save.`}</p><Button variant="primary" isDisabled={!valid || conflict} isLoading={saving} onPress={() => void save()}>Save assessment</Button></div>}
    {documentsOpen && <FocusedSheet label="Submitted assessment evidence" size="wide" onClose={() => setDocumentsOpen(false)}><DocumentBrowser responseRevisionID={responseID} scopeLabel={`Submitted evidence · form revision ${detail.form_template_version}`}/></FocusedSheet>}
  </section>;
}

function AssessmentScore({ title, score, provisional }: { title: string; score?: ResponseScore; provisional?: boolean }) {
  const presentation = scorePresentation(score);
  return <section aria-label={title}><h4>{title}</h4><p>{presentation.value}</p><p>{presentation.meaning}</p>{provisional && <p>Provisional</p>}{score?.adverse_score != null && <p>{score.adverse_score} concern points</p>}<p>Coverage: {coverageText(score)}</p>{score?.calculated_at && <p>Calculated <time dateTime={score.calculated_at}>{formatTime(score.calculated_at)}</time></p>}</section>;
}

function RuleExplanation({ score }: { score?: ResponseScore }) {
  if (!score?.contribution_results?.length && !score?.rule_results?.length) return null;
  return <details><summary>Score calculation</summary><p>Submission scoring profile: {score.profile_version ?? "Not recorded"}</p><ul>{score.contribution_results?.map((rule) => <li key={rule.id}>{rule.id}: {rule.points} points · Weight {rule.weight} · {rule.outcome.toLowerCase().replaceAll("_", " ")}</li>)}{score.rule_results?.map((rule) => <li key={rule.id}>{rule.id}: {rule.matched ? "Matched" : "Not matched"} · {rule.effect.toLowerCase().replaceAll("_", " ")}{rule.value !== undefined ? ` ${rule.value}` : ""}</li>)}</ul></details>;
}

function evidenceField(item: ResponseFieldAssessment) { return ["file", "photo", "signature", "vendor_document"].includes(item.field.type); }
function answerText(item: ResponseFieldAssessment) {
  const answer = item.answer;
  if (answer?.text) return answer.text;
  if (answer?.values?.length) return answer.values.join(", ");
  const count = answer?.artifact_ids?.length ?? (answer?.document?.artifact_id ? 1 : 0);
  if (count) return `${count} submitted ${count === 1 ? "document" : "documents"}. Open the submitted evidence to inspect files and availability.`;
  return evidenceField(item) ? "No evidence submitted for this field." : "No answer submitted for this field.";
}
function matchesFilter(item: ResponseFieldAssessment, filter: ReviewFilter, detail: ResponseAssessmentDetail) {
  if (filter === "PENDING") return needsBankReview(item.field) && !item.decision;
  if (filter === "REVIEWED") return Boolean(item.decision);
  if (filter === "MISSING") return evidenceField(item) && !item.answer?.artifact_ids?.length && !item.answer?.document?.artifact_id;
  if (filter === "POOR") return Boolean(item.decision && assessmentConcernPoints(item.decision.points, detail) >= assessmentConcernThreshold(detail)) || automaticFieldResults(item, detail).some((result) => result.concern !== undefined && result.concern >= assessmentConcernThreshold(detail));
  return true;
}
function formatTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Time not recorded" : date.toLocaleString(); }
function message(cause: unknown, fallback: string) { return cause instanceof Error ? cause.message : fallback; }
