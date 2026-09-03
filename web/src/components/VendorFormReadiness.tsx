import { useEffect, useMemo, useState } from "react";
import { loadContext, loadProgramSummaries } from "../api";
import { apiErrorKind } from "../http";
import { createFormTemplate, loadFormTemplates, transitionFormTemplate } from "../monitoringApi";
import type { FormTemplate } from "../monitoringTypes";
import type { ProgramSummary } from "../summaryTypes";
import { vendorDueDiligenceStarterForm } from "../vendorDueDiligenceForm";
import { FocusedSheet } from "./FocusedSheet";
import { Notice } from "./ui";
import "./vendor-due-diligence.css";

type Props = {
  onClose: () => void;
  onReady: (form: FormTemplate) => void;
};

type LoadState = "loading" | "live" | "unavailable";

export function VendorFormReadiness({ onClose, onReady }: Props) {
  const [programs, setPrograms] = useState<ProgramSummary[]>([]);
  const [actorID, setActorID] = useState("");
  const [state, setState] = useState<LoadState>("loading");
  const [selectedProgramID, setSelectedProgramID] = useState("");
  const [forms, setForms] = useState<FormTemplate[]>([]);
  const [formState, setFormState] = useState<"idle" | LoadState>("idle");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [conflict, setConflict] = useState(false);
  const current = useMemo(() => forms
    .filter((form) => form.code.trim().toUpperCase() === vendorDueDiligenceStarterForm.code)
    .sort((left, right) => right.version - left.version)[0], [forms]);

  useEffect(() => {
    let mounted = true;
    Promise.all([loadContext(), loadProgramSummaries({ limit: 50 })]).then(([context, page]) => {
      if (!mounted) return;
      setActorID(context.actor.id);
      setPrograms(page.items);
      setState("live");
    }).catch(() => { if (mounted) setState("unavailable"); });
    return () => { mounted = false; };
  }, []);

  async function selectProgram(programID: string) {
    setSelectedProgramID(programID);
    setForms([]);
    setError("");
    setConflict(false);
    if (!programID) { setFormState("idle"); return; }
    await refreshForms(programID);
  }

  async function refreshForms(programID = selectedProgramID) {
    if (!programID) return;
    setFormState("loading");
    setError("");
    setConflict(false);
    try {
      setForms(await loadFormTemplates(programID));
      setFormState("live");
    } catch {
      setFormState("unavailable");
    }
  }

  async function createAndSubmit() {
    if (!selectedProgramID || busy) return;
    setBusy(true); setError(""); setConflict(false);
    try {
      const draft = await createFormTemplate(selectedProgramID, vendorDueDiligenceStarterForm);
      setForms([draft]);
      setFormState("live");
      const pending = await transitionFormTemplate(selectedProgramID, draft.id, draft.version, "PENDING_APPROVAL");
      setForms([pending]);
    } catch (cause) {
      handleMutationError(cause);
    } finally {
      setBusy(false);
    }
  }

  async function transitionCurrent(to: "PENDING_APPROVAL" | "ACTIVE") {
    if (!selectedProgramID || !current || busy) return;
    setBusy(true); setError(""); setConflict(false);
    try {
      const updated = await transitionFormTemplate(selectedProgramID, current.id, current.version, to);
      setForms([updated, ...forms.filter((form) => form.id !== updated.id)]);
      if (updated.status === "ACTIVE" && updated.is_current) onReady(updated);
    } catch (cause) {
      handleMutationError(cause);
    } finally {
      setBusy(false);
    }
  }

  function handleMutationError(cause: unknown) {
    const kind = apiErrorKind(cause);
    if (kind === "forbidden" || kind === "unauthorized") setError("Your current role cannot approve or configure this form. Ask the Program owner or an authorized independent reviewer to continue.");
    else if (kind === "conflict") { setError("The form changed while this action was being completed. Reload its status before continuing."); setConflict(true); }
    else if (kind === "validation") setError("The due-diligence form could not be submitted because its current state is not valid. Reload the form status before trying again.");
    else setError("The due-diligence form could not be updated. Your selected Program remains available; try again.");
  }

  const selectedProgram = programs.find((item) => item.program.id === selectedProgramID)?.program;
  const waitingForReviewer = current?.status === "PENDING_APPROVAL" && current.submitted_by === actorID;

  return <FocusedSheet label="Set up due-diligence form" onClose={onClose} panelClassName="vendor-form-readiness-sheet">
    <div className="vendor-form-readiness">
      <header><span className="eyebrow">Vendor review readiness</span><h2>Set up due-diligence form</h2><p>Attach the bank&apos;s starter questionnaire to a Program, then send it to an independent reviewer before it can be used for vendor reviews.</p></header>
      {state === "loading" && <p aria-live="polite" aria-busy="true">Loading Programs available for this legal entity…</p>}
      {state === "unavailable" && <Notice tone="error"><strong>Programs are unavailable</strong> Programs could not be loaded. Close this panel and try again before configuring the form.</Notice>}
      {state === "live" && programs.length === 0 && <div className="vdd-limitation"><strong>No Programs are available in this legal entity.</strong><p>Create or gain access to a Program before setting up vendor due diligence.</p></div>}
      {state === "live" && programs.length > 0 && <label className="vdd-field"><span>Program</span><select value={selectedProgramID} onChange={(event) => void selectProgram(event.target.value)}><option value="">Select a Program</option>{programs.map(({ program }) => <option key={program.id} value={program.id}>{program.name} · {program.code}</option>)}</select></label>}

      {formState === "loading" && <p aria-live="polite" aria-busy="true">Checking the selected Program&apos;s form status…</p>}
      {formState === "unavailable" && <Notice tone="error"><strong>Form status is unavailable</strong> The forms for {selectedProgram?.name ?? "this Program"} could not be loaded. <button type="button" className="secondary-button" onClick={() => void refreshForms()}>Reload form status</button></Notice>}
      {error && <Notice tone="error">{error}</Notice>}
      {conflict && <button type="button" className="secondary-button" onClick={() => void refreshForms()} disabled={busy}>Reload form status</button>}

      {formState === "live" && !current && <section className="vendor-form-readiness-state"><div><h3>Starter form not created</h3><p>This creates the four-section vendor security and privacy questionnaire as a draft and sends it for independent approval.</p></div><button type="button" className="primary-button" onClick={() => void createAndSubmit()} disabled={busy}>{busy ? "Sending for approval…" : "Create form and send for approval"}</button></section>}
      {formState === "live" && current?.status === "DRAFT" && <section className="vendor-form-readiness-state"><div><h3>Draft form ready</h3><p>{current.name} is attached to {selectedProgram?.name}. Send version {current.version} for independent approval.</p></div><button type="button" className="primary-button" onClick={() => void transitionCurrent("PENDING_APPROVAL")} disabled={busy}>{busy ? "Sending for approval…" : "Send form for approval"}</button></section>}
      {formState === "live" && waitingForReviewer && <section className="vendor-form-readiness-state"><div><h3>Waiting for an independent reviewer</h3><p>Version {current.version} was submitted by your signed-in identity. A different authorized reviewer must activate it before due diligence can start.</p></div></section>}
      {formState === "live" && current?.status === "PENDING_APPROVAL" && !waitingForReviewer && <section className="vendor-form-readiness-state"><div><h3>Independent review required</h3><p>Review {current.name}, version {current.version}. Activation makes this questionnaire available to new vendor reviews in the current legal entity.</p></div><button type="button" className="primary-button" onClick={() => void transitionCurrent("ACTIVE")} disabled={busy}>{busy ? "Activating…" : "Activate due-diligence form"}</button></section>}
      {formState === "live" && current?.status === "ACTIVE" && <section className="vendor-form-readiness-state"><div><h3>Due-diligence form is active</h3><p>{current.name}, version {current.version}, is ready for new vendor reviews.</p></div><button type="button" className="primary-button" onClick={() => onReady(current)}>Use active form</button></section>}
      {formState === "live" && current && !["DRAFT", "PENDING_APPROVAL", "ACTIVE"].includes(current.status) && <section className="vendor-form-readiness-state"><div><h3>Current form cannot be activated</h3><p>The latest vendor form is {current.status.toLowerCase().replaceAll("_", " ")}. Create a replacement draft and send it for approval.</p></div><button type="button" className="primary-button" onClick={() => void createAndSubmit()} disabled={busy}>{busy ? "Sending for approval…" : "Create replacement and send for approval"}</button></section>}
    </div>
  </FocusedSheet>;
}
