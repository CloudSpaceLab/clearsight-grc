import { useEffect, useRef, useState } from "react";
import { createDocumentFormProposal } from "../../documentApi";
import type { FormTemplateProposal } from "../../formsTypes";
import { Button, Notice, SelectField } from "../ui";

export function FindingFollowUpPicker({ proposal, onProposalChange }: { proposal: FormTemplateProposal; onProposalChange: (value: FormTemplateProposal) => void }) {
  const [selected, setSelected] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const pending = useRef(false);
  const active = useRef(true);
  useEffect(() => { active.current = true; return () => { active.current = false; }; }, []);
  const assessments = proposal.provenance.finding_assessments ?? [];
  const selectedAssessment = assessments.find((assessment) => assessment.id === selected);
  if (!proposal.source_document_id || !proposal.source_document_version || proposal.base_template_id || (!assessments.length && !proposal.finding_assessment_id)) return null;

  async function prepare(assessmentID?: string) {
    if (pending.current || !proposal.source_document_id || !proposal.source_document_version) return;
    pending.current = true;
    setBusy(true);
    setError(undefined);
    try {
      const result = await createDocumentFormProposal(proposal.source_document_id, proposal.source_document_version, undefined, undefined, assessmentID);
      if (active.current) onProposalChange(result);
    } catch (cause) {
      if (active.current) setError(cause instanceof Error ? cause.message : "The follow-up questions could not be prepared. Try again or review the source rows.");
    } finally {
      pending.current = false;
      if (active.current) setBusy(false);
    }
  }

  return <div className="finding-followup-picker">
    {proposal.finding_assessment_id ? <Button variant="secondary" isDisabled={busy} onPress={() => void prepare()}>Review source-row proposal</Button> : <details>
      <summary>Prepare finding follow-up</summary>
      <div className="finding-followup-fields">
        <p>Choose one historical assessment to prepare response, action, owner, date and evidence questions. Review the vendor and service before approving or sending the form.</p>
        <SelectField label="Source assessment" placeholder="Choose an assessment" value={selected} onChange={setSelected} isDisabled={busy} options={assessments.map((assessment) => ({ id: assessment.id, label: assessment.label, description: `${assessment.finding_count} findings · ${assessment.sheet}, rows ${assessment.row_start}–${assessment.row_end}` }))}/>
        {selectedAssessment && <p>{selectedAssessment.finding_count} findings · {selectedAssessment.sheet}, rows {selectedAssessment.row_start}–{selectedAssessment.row_end}</p>}
        <Button variant="secondary" isDisabled={!selectedAssessment || busy} onPress={() => void prepare(selectedAssessment?.id)}>{busy ? "Preparing questions…" : "Prepare follow-up questions"}</Button>
      </div>
    </details>}
    {error && <Notice tone="error">{error}</Notice>}
  </div>;
}
