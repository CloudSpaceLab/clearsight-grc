import { useState, type ReactNode } from "react";
import type { JSX } from "react";
import type { ResponseAssessmentDetail, ResponseFieldAssessment } from "../../formAssessmentApi";
import type { FormTemplateField, FormTemplateSection } from "../../monitoringTypes";
import type { CaptureAnswerValue } from "../../types";
import { answerIsPresent, normalizeFieldType } from "../capture/contract";
import { assessmentConcernPoints, assessmentConcernThreshold, automaticFieldResults } from "./assessmentFieldResults";
import { AnswerValueDisplay } from "./AnswerValueDisplay";
import "./response-answer-sheet.css";

export type AnswerSheetField = Pick<FormTemplateField, "id" | "label" | "type" | "required" | "section_id" | "description">;
export type AnswerSheetItem = { field: AnswerSheetField; answer?: CaptureAnswerValue; decision?: { points: number; outcome_id?: string } };
export type AnswerAttention = "MISSING_ANSWER" | "MISSING_EVIDENCE" | "POOR";

const evidenceTypes = new Set(["file", "photo", "signature", "vendor_document"]);

export function answerAttention(item: AnswerSheetItem, context?: ResponseAssessmentDetail): AnswerAttention | undefined {
  const present = answerIsPresent(item.answer);
  if (!present) return evidenceTypes.has(normalizeFieldType(item.field.type) ?? "") ? "MISSING_EVIDENCE" : "MISSING_ANSWER";
  if (!context) return undefined;
  const threshold = assessmentConcernThreshold(context);
  const decisionPoor = item.decision != null && assessmentConcernPoints(item.decision.points, context) >= threshold;
  const automaticPoor = automaticFieldResults(item as ResponseFieldAssessment, context).some((result) => result.concern !== undefined && result.concern >= threshold);
  return decisionPoor || automaticPoor ? "POOR" : undefined;
}

function poorConcernPoints(item: AnswerSheetItem, context: ResponseAssessmentDetail): number | undefined {
  const threshold = assessmentConcernThreshold(context);
  if (item.decision != null) {
    const decisionConcern = assessmentConcernPoints(item.decision.points, context);
    if (decisionConcern >= threshold) return decisionConcern;
  }
  const concerns = automaticFieldResults(item as ResponseFieldAssessment, context)
    .map((result) => result.concern)
    .filter((concern): concern is number => concern !== undefined && concern >= threshold);
  return concerns.length ? Math.max(...concerns) : undefined;
}

export function groupAnswerFields(fields: AnswerSheetItem[], sections?: FormTemplateSection[]): Array<{ id?: string; title: string; help?: string; fields: AnswerSheetItem[] }> {
  if (!sections || sections.length === 0) return [{ title: "All answers", fields: [...fields] }];
  const bySection = new Map<string, AnswerSheetItem[]>();
  const unmatched: AnswerSheetItem[] = [];
  for (const item of fields) {
    const section = item.field.section_id ? sections.find((candidate) => candidate.id === item.field.section_id) : undefined;
    if (section) {
      const bucket = bySection.get(section.id) ?? [];
      bucket.push(item);
      bySection.set(section.id, bucket);
    } else {
      unmatched.push(item);
    }
  }
  const groups: Array<{ id?: string; title: string; help?: string; fields: AnswerSheetItem[] }> = sections
    .filter((section) => bySection.has(section.id))
    .map((section) => ({ id: section.id, title: section.title, help: section.help, fields: bySection.get(section.id) ?? [] }));
  if (unmatched.length) groups.push({ title: "Other answers", fields: unmatched });
  return groups;
}

function answeredCount(fields: AnswerSheetItem[]): number {
  return fields.filter((item) => answerIsPresent(item.answer)).length;
}

function AnswerSheetSummaryContent({ items, getAttention }: { items: AnswerSheetItem[]; getAttention: (item: AnswerSheetItem) => AnswerAttention | undefined }): JSX.Element | null {
  if (items.length === 0) return null;
  let missingAnswer = 0;
  let missingEvidence = 0;
  let poor = 0;
  for (const item of items) {
    const attention = getAttention(item);
    if (attention === "MISSING_ANSWER") missingAnswer += 1;
    else if (attention === "MISSING_EVIDENCE") missingEvidence += 1;
    else if (attention === "POOR") poor += 1;
  }
  const attentionCount = missingAnswer + missingEvidence + poor;
  return <div className="response-answer-sheet__summary">
    <p>{answeredCount(items)} of {items.length} fields answered</p>
    {attentionCount > 0 && <>
      <p>{attentionCount} {attentionCount === 1 ? "field needs" : "fields need"} attention</p>
      <ul className="response-answer-sheet__chips">
        {missingAnswer > 0 && <li className="response-answer-sheet__chip">{missingAnswer} no {missingAnswer === 1 ? "answer" : "answers"}</li>}
        {missingEvidence > 0 && <li className="response-answer-sheet__chip">{missingEvidence} no evidence</li>}
        {poor > 0 && <li className="response-answer-sheet__chip">{poor} poor {poor === 1 ? "result" : "results"}</li>}
      </ul>
    </>}
  </div>;
}

export function AnswerSheetSummary(props: { fields: AnswerSheetItem[]; assessmentContext?: ResponseAssessmentDetail; emptyLabel: string; evidenceEmptyLabel: string }): JSX.Element | null {
  return <AnswerSheetSummaryContent items={props.fields} getAttention={(item) => answerAttention(item, props.assessmentContext)} />;
}

export function ResponseAnswerSheet(props: {
  fields: AnswerSheetItem[];
  sections?: FormTemplateSection[];
  assessmentContext?: ResponseAssessmentDetail;
  title?: string;
  note?: string;
  emptyLabel: string;
  evidenceEmptyLabel: string;
  onOpenDocuments?: () => void;
  excludeFromAttention?: (item: AnswerSheetItem) => boolean;
  renderValue?: (item: AnswerSheetItem) => ReactNode;
  headingLevel?: "h3" | "h4";
}): JSX.Element {
  const { fields, sections, assessmentContext, title, note, emptyLabel, evidenceEmptyLabel, onOpenDocuments, excludeFromAttention, renderValue, headingLevel = "h4" } = props;
  const [view, setView] = useState<"ALL" | "NEEDS_ATTENTION">("ALL");
  const GroupHeading: "h3" | "h4" = headingLevel;

  if (fields.length === 0) return <section className="response-assessment" aria-label={title || undefined}>
    {title && <h3>{title}</h3>}
    {note && <p className="response-answer-sheet__note">{note}</p>}
    <p>No answer fields were recorded for this submitted response.</p>
  </section>;

  const attentionOf = (item: AnswerSheetItem): AnswerAttention | undefined => excludeFromAttention?.(item) ? undefined : answerAttention(item, assessmentContext);
  const attentionCount = fields.reduce((count, item) => count + (attentionOf(item) ? 1 : 0), 0);

  function reason(item: AnswerSheetItem): string | undefined {
    const attention = attentionOf(item);
    if (attention === "POOR") return `Poor result · ${poorConcernPoints(item, assessmentContext!)} concern points`;
    if (attention === "MISSING_ANSWER") return "No answer submitted for this field.";
    if (attention === "MISSING_EVIDENCE") return "No evidence submitted for this field.";
    return undefined;
  }

  const visibleFields = view === "NEEDS_ATTENTION" ? fields.filter((item) => attentionOf(item)) : fields;
  const groups = groupAnswerFields(visibleFields, sections);

  return <section className="response-assessment" aria-label={title || undefined}>
    {title && <h3>{title}</h3>}
    {note && <p className="response-answer-sheet__note">{note}</p>}
    <AnswerSheetSummaryContent items={fields} getAttention={attentionOf} />
    <div className="response-answer-sheet__switch" role="group" aria-label="Answer view">
      <button type="button" className="response-answer-sheet__switch-button" aria-pressed={view === "ALL"} onClick={() => setView("ALL")}>All answers</button>
      <button type="button" className="response-answer-sheet__switch-button" aria-pressed={view === "NEEDS_ATTENTION"} disabled={attentionCount === 0} onClick={() => setView("NEEDS_ATTENTION")}>Needs attention</button>
    </div>
    {attentionCount === 0 && <p className="response-answer-sheet__switch-note">No fields in this response need attention.</p>}
    {groups.map((group) => <section className="response-answer-sheet__group" key={group.id ?? group.title}>
      <header className="response-answer-sheet__group-header">
        <GroupHeading>{group.title}</GroupHeading>
        {group.id !== undefined && <span className="response-answer-sheet__group-count">{answeredCount(group.fields)} of {group.fields.length} answered</span>}
      </header>
      {group.help && <p className="response-answer-sheet__group-help">{group.help}</p>}
      <dl className="response-answer-sheet__list">
        {group.fields.map((item) => {
          const rowReason = view === "NEEDS_ATTENTION" ? reason(item) : undefined;
          return <div className="response-answer-sheet__row" key={item.field.id}>
            <dt>{item.field.label}</dt>
            <dd>
              {renderValue ? renderValue(item) : <AnswerValueDisplay type={item.field.type} answer={item.answer} emptyLabel={emptyLabel} evidenceEmptyLabel={evidenceEmptyLabel} onOpenDocuments={onOpenDocuments} />}
              {rowReason && <p className="response-answer-sheet__reason">{rowReason}</p>}
            </dd>
          </div>;
        })}
      </dl>
    </section>)}
  </section>;
}