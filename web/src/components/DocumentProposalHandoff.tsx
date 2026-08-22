import { useEffect, useMemo, useState } from "react";
import { loadPrograms } from "../api";
import { authorizeDocumentProposalHandoff, loadDocumentImport, reviewDocumentProposalHandoff } from "../documentApi";
import type { DocumentImport, DocumentProposal, HandoffAuthorizationInput, HandoffReviewInput } from "../documentTypes";
import { apiErrorKind } from "../http";
import type { ProgramAggregate } from "../types";

type Props = {
  documentID: string;
  documentVersion: number;
  legalEntityID?: string;
  proposal: DocumentProposal;
  locked: boolean;
  onDocumentUpdated?: (document: DocumentImport) => void;
};

export function DocumentProposalHandoff({ documentID, documentVersion, legalEntityID, proposal, locked, onDocumentUpdated }: Props) {
  const [currentProposal, setCurrentProposal] = useState(proposal);
  const [currentDocumentVersion, setCurrentDocumentVersion] = useState(documentVersion);
  const [programs, setPrograms] = useState<ProgramAggregate[]>([]);
  const [programsLoaded, setProgramsLoaded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const handoff = currentProposal.handoff;
  const [title, setTitle] = useState(handoff?.draft_title || proposal.title);
  const [statement, setStatement] = useState(handoff?.draft_statement || proposal.statement);
  const [targetType, setTargetType] = useState<"REQUIREMENT" | "CONTROL_OBJECTIVE">(handoff?.target_type || "REQUIREMENT");
  const [programID, setProgramID] = useState(handoff?.target_program_id || "");
  const [note, setNote] = useState("");

  const scopedPrograms = useMemo(() => programs.filter(({ program }) => !program.legal_entity_id || !legalEntityID || program.legal_entity_id === legalEntityID), [programs, legalEntityID]);
  const selectedProgram = scopedPrograms.find(({ program }) => program.id === programID);
  const recordedProgram = programs.find(({ program }) => program.id === handoff?.target_program_id);

  useEffect(() => {
    setCurrentProposal(proposal);
    setCurrentDocumentVersion(documentVersion);
  }, [proposal, documentVersion]);

  useEffect(() => {
    setTitle(handoff?.draft_title || currentProposal.title);
    setStatement(handoff?.draft_statement || currentProposal.statement);
    setTargetType(handoff?.target_type || "REQUIREMENT");
    setProgramID(handoff?.target_program_id || "");
    setNote("");
  }, [handoff?.id, handoff?.version, currentProposal.id, currentProposal.title, currentProposal.statement]);

  useEffect(() => {
    if (programsLoaded || !handoff || (handoff.status !== "AWAITING_REVIEW" && handoff.status !== "AWAITING_AUTHORIZATION")) return;
    let active = true;
    void loadPrograms().then((items) => { if (active) setPrograms(items); }).catch(() => { if (active) setPrograms([]); }).finally(() => { if (active) setProgramsLoaded(true); });
    return () => { active = false; };
  }, [handoff?.status, programsLoaded]);

  if (!handoff) {
    return currentProposal.status === "ACCEPTED"
      ? <div className="proposal-handoff proposal-handoff-unavailable"><strong>Accepted · handoff unavailable</strong><p>This legacy acceptance has no governed handoff receipt. Reload or reconcile this import before relying on it.</p></div>
      : null;
  }

  const expectedHandoffVersion = handoff.version;
  const route = handoff.route;
  const canAct = Boolean(route?.is_current_actor) && !locked;
  const stage = handoffLabel(handoff.status);
  const assignee = routeSummary(route?.status, route?.principal_name);

  return <section className={`proposal-handoff handoff-${handoff.status.toLowerCase().replaceAll("_", "-")}`} aria-label={`Proposal handoff: ${stage}`}>
    <div className="proposal-handoff-head">
      <div><span>Governed handoff</span><strong>{stage}</strong></div>
      {handoff.status === "AWAITING_REVIEW" || handoff.status === "AWAITING_AUTHORIZATION" ? <small>{assignee}</small> : null}
    </div>
    {error && <p className="error-text proposal-handoff-error" role="alert">{error}</p>}

    {handoff.status === "AWAITING_REVIEW" && canAct && <div className="proposal-handoff-form">
      <label><span>Canonical title</span><input value={title} onChange={(event) => setTitle(event.target.value)} disabled={busy}/></label>
      <label><span>Canonical statement</span><textarea rows={3} value={statement} onChange={(event) => setStatement(event.target.value)} disabled={busy}/></label>
      <div className="proposal-handoff-grid">
        <label><span>Create as</span><select value={targetType} onChange={(event) => setTargetType(event.target.value as typeof targetType)} disabled={busy}><option value="REQUIREMENT">Requirement</option><option value="CONTROL_OBJECTIVE">Control objective</option></select></label>
        <label><span>Target Program</span><select value={programID} onChange={(event) => setProgramID(event.target.value)} disabled={busy || !programsLoaded}><option value="">{programsLoaded ? "Choose Program" : "Loading Programs…"}</option>{scopedPrograms.map(({ program }) => <option key={program.id} value={program.id}>{program.code} · {program.name}</option>)}</select></label>
      </div>
      <label><span>Review note</span><textarea rows={2} value={note} onChange={(event) => setNote(event.target.value)} placeholder="Required when returning or rejecting" disabled={busy}/></label>
      <div className="proposal-actions proposal-handoff-actions">
        <button className="text-button" type="button" disabled={busy || !note.trim()} onClick={() => void actReview("RETURN")}>Return</button>
        <button className="secondary-button" type="button" disabled={busy || !note.trim()} onClick={() => void actReview("REJECT")}>Reject</button>
        <button className="primary-button" type="button" disabled={busy || !title.trim() || !statement.trim() || !selectedProgram} onClick={() => void actReview("SUBMIT_FOR_AUTHORIZATION")}>{busy ? "Submitting…" : "Send for authorization"}</button>
      </div>
    </div>}

    {handoff.status === "AWAITING_AUTHORIZATION" && <div className="proposal-handoff-decision">
      <dl><div><dt>Create as</dt><dd>{handoff.target_type === "CONTROL_OBJECTIVE" ? "Control objective" : "Requirement"}</dd></div><div><dt>Program</dt><dd>{recordedProgram ? `${recordedProgram.program.code} · ${recordedProgram.program.name}` : handoff.target_program_id}</dd></div><div><dt>Title</dt><dd>{handoff.draft_title}</dd></div></dl>
      <p>{handoff.draft_statement}</p>
      {canAct && <><label><span>Authorization rationale</span><textarea rows={2} value={note} onChange={(event) => setNote(event.target.value)} placeholder="Required for this decision" disabled={busy}/></label><div className="proposal-actions proposal-handoff-actions"><button className="text-button" type="button" disabled={busy || !note.trim()} onClick={() => void actAuthorize("RETURN")}>Return</button><button className="secondary-button" type="button" disabled={busy || !note.trim()} onClick={() => void actAuthorize("REJECT")}>Reject</button><button className="primary-button" type="button" disabled={busy || !note.trim()} onClick={() => void actAuthorize("APPROVE")}>{busy ? "Authorizing…" : "Authorize conversion"}</button></div></>}
    </div>}

    {(handoff.status === "APPROVED" || handoff.status === "REJECTED" || handoff.status === "RETURNED" || handoff.status === "CONVERSION_FAILED") && <div className="proposal-handoff-receipt">
      {handoff.status === "APPROVED" && <><strong>Canonical object created</strong><p>{human(handoff.result_object_type || handoff.target_type || "Object")} · <code>{handoff.result_object_id}</code></p></>}
      {handoff.status === "REJECTED" && <><strong>Proposal rejected</strong><p>{handoff.authorization_note || handoff.review_note || "No further conversion is pending."}</p></>}
      {handoff.status === "RETURNED" && <><strong>Proposal returned</strong><p>{handoff.authorization_note || handoff.review_note || "The intake requires another governed review."}</p></>}
      {handoff.status === "CONVERSION_FAILED" && <><strong>Conversion needs attention</strong><p>No canonical object should be relied on until this handoff is reconciled.</p></>}
    </div>}
  </section>;

  async function actReview(action: HandoffReviewInput["action"]) {
    const base: HandoffReviewInput = { action, expected_document_version: currentDocumentVersion, expected_handoff_version: expectedHandoffVersion, note: note.trim() || undefined };
    const input = action === "SUBMIT_FOR_AUTHORIZATION" && selectedProgram ? {
      ...base,
      title: title.trim(),
      statement: statement.trim(),
      target_type: targetType,
      target_program_id: selectedProgram.program.id,
      target_program_version: selectedProgram.program.version,
    } : base;
    await run(() => reviewDocumentProposalHandoff(documentID, currentProposal.id, input));
  }

  async function actAuthorize(action: HandoffAuthorizationInput["action"]) {
    await run(() => authorizeDocumentProposalHandoff(documentID, currentProposal.id, {
      action,
      expected_document_version: currentDocumentVersion,
      expected_handoff_version: expectedHandoffVersion,
      note: note.trim() || undefined,
    }));
  }

  async function run(command: () => Promise<DocumentImport>) {
    if (busy) return;
    setBusy(true);
    setError(null);
    try {
      applyUpdated(await command());
    } catch (cause) {
      const conflict = apiErrorKind(cause) === "conflict";
      setError(conflict ? "This proposal changed while you were acting. The latest state has been loaded." : cause instanceof Error ? cause.message : "The handoff action could not be completed.");
      if (conflict) {
        try { applyUpdated(await loadDocumentImport(documentID)); } catch { /* keep the durable state already shown */ }
      }
    } finally {
      setBusy(false);
    }
  }

  function applyUpdated(document: DocumentImport) {
    const updated = document.proposals.find((item) => item.id === currentProposal.id);
    if (updated) setCurrentProposal(updated);
    setCurrentDocumentVersion(document.version);
    onDocumentUpdated?.(document);
  }
}

function handoffLabel(status: NonNullable<DocumentProposal["handoff"]>["status"]) {
  const labels = {
    AWAITING_REVIEW: "Awaiting independent review",
    AWAITING_AUTHORIZATION: "Awaiting authorization",
    RETURNED: "Returned",
    REJECTED: "Rejected",
    APPROVED: "Approved",
    CONVERSION_FAILED: "Conversion failed",
  } as const;
  return labels[status];
}

function routeSummary(status?: string, principalName?: string) {
  if (status === "DIRECT") return principalName ? `Assigned to ${principalName}` : "Assigned";
  if (status === "CANDIDATE_SET") return "Multiple eligible reviewers · routing unresolved";
  if (status === "NO_INDEPENDENT_CANDIDATE") return "No independent eligible reviewer";
  if (status === "NO_ROUTE") return "No authority route configured";
  if (status === "AMBIGUOUS_ROUTE") return "Authority route is ambiguous";
  if (status === "NO_LEGAL_ENTITY") return "Legal-entity scope is missing";
  return "Routing unavailable";
}

function human(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
