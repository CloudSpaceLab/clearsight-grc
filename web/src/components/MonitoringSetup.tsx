import { lazy, Suspense, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { createFormMonitoringCheck, createMonitoringLinkedIssue, createSourceMonitoringCheck, evaluateMonitoringSource, loadFormTemplates, loadMonitoringChecks, loadMonitoringResults, startFormCollection, transitionFormTemplate, transitionMonitoringCheck } from "../monitoringApi";
import type { FormTemplate, LifecycleStatus, MonitoringCheck, MonitoringResult } from "../monitoringTypes";
import type { ProgramAggregate } from "../types";
import { DataSourceBuilder } from "./DataSourceBuilder";
import type { SourceBinding } from "../sourceConfigApi";
import type { ProgramOperation } from "../programOperationsApi";
import { Notice } from "./ui";
import { apiErrorKind } from "../http";

const FormBuilder = lazy(() => import("./FormBuilder").then((module) => ({ default: module.FormBuilder })));

type Props = { aggregate: ProgramAggregate; actorPrincipalID: string; canConfigureSources: boolean; operations: ProgramOperation[]; onOpenMatter?: (matterID: string) => void };
type SetupMode = "closed" | "choose" | "form" | "source";

function latestByID<T extends { id: string; version: number }>(values: T[]) {
  const latest = new Map<string, T>();
  for (const value of values) if (!latest.has(value.id) || value.version > latest.get(value.id)!.version) latest.set(value.id, value);
  return [...latest.values()];
}

function statusLabel(status: LifecycleStatus) {
  switch (status) {
    case "DRAFT": return "Draft";
    case "PENDING_APPROVAL": return "Awaiting approval";
    case "ACTIVE": return "Active";
    case "PAUSED": return "Paused";
    case "REJECTED": return "Returned";
    case "RETIRED": return "Ended";
  }
}

function riskLabel(result: MonitoringResult) {
  return result.evaluation.score == null ? "Not assessed" : `${Math.round(result.evaluation.score)}% risk`;
}

function bandLabel(result: MonitoringResult) {
  return result.evaluation.band.toLowerCase().replaceAll("_", " ").replace(/^./, (value) => value.toUpperCase());
}

function eligibleForLinkedIssue(check: MonitoringCheck, result?: MonitoringResult) {
  if (!result || check.status !== "ACTIVE" || !check.is_current || check.failure_action !== "RECOMMEND_MATTER" || result.monitoring_check_version !== check.version) return false;
  if (result.evaluation.band === "HIGH" || result.evaluation.band === "CRITICAL" || result.evaluation.coverage < check.minimum_coverage || (result.evaluation.critical_failures?.length ?? 0) > 0) return true;
  return (result.evaluation.rule_results ?? []).some((rule) => rule.critical && rule.outcome !== "PASS");
}

function hasScoredQuestions(form: FormTemplate) {
  return form.fields.some((field) => Boolean(field.scoring && Object.keys(field.scoring.answer_scores ?? {}).length));
}

export function MonitoringSetup({ aggregate, actorPrincipalID, canConfigureSources, operations, onOpenMatter = (matterID) => { window.location.hash = `#work/matters/${encodeURIComponent(matterID)}`; } }: Props) {
  const [forms, setForms] = useState<FormTemplate[]>([]);
  const [checks, setChecks] = useState<MonitoringCheck[]>([]);
  const [latestResults, setLatestResults] = useState<Record<string, MonitoringResult>>({});
  const [state, setState] = useState<"loading" | "live" | "unavailable">("loading");
  const [mode, setMode] = useState<SetupMode>("closed");
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [collecting, setCollecting] = useState<FormTemplate | null>(null);
  const [notice, setNotice] = useState("");

  async function reload() {
    setState("loading");
    const [formResult, checkResult] = await Promise.allSettled([loadFormTemplates(aggregate.program.id), loadMonitoringChecks(aggregate.program.id)]);
    if (formResult.status === "fulfilled") setForms(latestByID(formResult.value));
    if (checkResult.status === "fulfilled") {
      const currentChecks = latestByID(checkResult.value);
      setChecks(currentChecks);
      const loaded = await Promise.allSettled(currentChecks.map(async (check) => ({ check, results: await loadMonitoringResults(check.id) })));
      const next: Record<string, MonitoringResult> = {};
      for (const value of loaded) if (value.status === "fulfilled" && value.value.results[0]) next[value.value.check.id] = value.value.results[0];
      setLatestResults(next);
    }
    setState(formResult.status === "fulfilled" || checkResult.status === "fulfilled" ? "live" : "unavailable");
  }

  useEffect(() => { void reload(); }, [aggregate.program.id]);
  const formDefineOperation = operations.find((operation) => operation.command === "program.monitoring.form.define" && !operation.subresource_id);
  const checkDefineOperation = operations.find((operation) => operation.command === "program.monitoring.define" && !operation.subresource_id);
  const canDefineForm = Boolean(formDefineOperation?.can_act);
  const canDefineCheck = Boolean(checkDefineOperation?.can_act);
  const canConfigureMonitoring = canDefineForm || canDefineCheck;
  const checkOperation = (checkID: string, command: string) => operations.find((operation) => operation.command === command && operation.subresource_id === checkID);
  const formTransitionOperation = (formID: string) => operations.find((operation) => operation.command === "program.monitoring.form.transition" && operation.subresource_id === formID);
  const collectionOperation = (formID: string) => operations.find((operation) => operation.command === "program.monitoring.collect" && operation.subresource_id === formID);

  useEffect(() => {
    if (!canConfigureMonitoring) {
      setMode("closed");
      setCollecting(null);
    }
    if (mode === "form" && !canDefineForm) setMode("choose");
    if (mode === "source" && !canDefineCheck) setMode("choose");
  }, [canConfigureMonitoring, canDefineForm, canDefineCheck, mode]);

  const linkedFormVersions = useMemo(() => new Set(checks.filter((check) => check.input_kind === "FORM").map((check) => `${check.form_template_id}:${check.form_template_version}`)), [checks]);
  const activeFormVersions = useMemo(() => new Set(checks.filter((check) => check.input_kind === "FORM" && check.status === "ACTIVE" && check.is_current).map((check) => `${check.form_template_id}:${check.form_template_version}`)), [checks]);
  const formFields = useMemo(() => new Map(forms.flatMap((form) => form.fields.map((field) => [`${form.id}:${field.id}`, field.label] as const))), [forms]);

  async function changeForm(form: FormTemplate, to: LifecycleStatus) {
    const operation = formTransitionOperation(form.id);
    if (!operation?.can_act || !operation.allowed_targets?.includes(to)) return;
    setBusy(form.id); setError("");
    try {
      const updated = await transitionFormTemplate(aggregate.program.id, form.id, form.version, to);
      setForms((current) => latestByID([...current, updated]));
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The form status could not be changed.");
    } finally { setBusy(""); }
  }

  async function addCheck(form: FormTemplate) {
    if (!canDefineCheck) return;
    setBusy(form.id); setError("");
    try {
      const check = await createFormMonitoringCheck(aggregate.program.id, form);
      setChecks((current) => latestByID([...current, check]));
      setNotice("Monitoring check saved as a draft.");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The monitoring check could not be saved.");
    } finally { setBusy(""); }
  }

  async function addSourceCheck(binding: SourceBinding, config: { code: string; name: string; claim: string; field: string; expected: string }) {
    if (!canDefineCheck) return;
    setBusy(binding.binding_id); setError("");
    try {
      const check = await createSourceMonitoringCheck(aggregate.program.id, binding, config);
      setChecks((current) => latestByID([...current, check]));
      setMode("closed"); setNotice("Connected-data check saved as a draft.");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The connected-data check could not be saved.");
    } finally { setBusy(""); }
  }

  async function changeCheck(check: MonitoringCheck, to: LifecycleStatus) {
    const operation = checkOperation(check.id, "program.monitoring.transition");
    if (!operation?.can_act || !operation.allowed_targets?.includes(to)) return;
    setBusy(check.id); setError("");
    try {
      const revisions = await loadMonitoringChecks(aggregate.program.id);
      const latest = latestByID(revisions).find((candidate) => candidate.id === check.id);
      if (!latest) throw new Error("This monitoring check could not be found. Reload the Program and try again.");
      setChecks((current) => latestByID([...current, latest]));
      if (latest.status !== check.status) {
        setError("This monitoring check changed after you opened it. The latest revision has been loaded. Review the current status before taking another action.");
        return;
      }
      const expectedCurrent = checks.find((candidate) => candidate.program_id === latest.program_id && candidate.code === latest.code && candidate.is_current);
      const updated = await transitionMonitoringCheck(latest.id, latest.version, to, expectedCurrent);
      setChecks((current) => latestByID([...current, updated]));
    } catch (caught) {
      if (apiErrorKind(caught) === "conflict") {
        await reload();
        setError("This monitoring check changed after you opened it. The latest revision has been loaded. Review it, then approve again.");
      } else {
        setError(caught instanceof Error ? caught.message : "The monitoring check status could not be changed.");
      }
    } finally { setBusy(""); }
  }

  async function runSourceCheck(check: MonitoringCheck) {
    if (!checkOperation(check.id, "program.monitoring.evaluate")?.can_act) return;
    setBusy(check.id); setError("");
    try {
      const result = await evaluateMonitoringSource(check);
      setLatestResults((current) => ({ ...current, [check.id]: result }));
      const score = result.evaluation.score == null ? "not assessed" : `${Math.round(result.evaluation.score)}% risk`;
      setNotice(`${check.name}: ${score} · ${result.evaluation.band.toLowerCase().replaceAll("_", " ")}.`);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The source could not be evaluated.");
    } finally { setBusy(""); }
  }

  async function createLinkedIssue(check: MonitoringCheck, result: MonitoringResult) {
    const operation = checkOperation(check.id, "program.monitoring.issue.create");
    if (!operation?.can_act) return;
    setBusy(result.id); setError("");
    try {
      const linked = await createMonitoringLinkedIssue(result.id);
      if (!linked.matter?.id) throw new Error("The linked issue was created but its record could not be opened. Reload the Program and try again.");
      setNotice(`Linked issue ${linked.matter.reference} is ready for Control Assurance review.`);
      onOpenMatter(linked.matter.id);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The linked issue could not be created from this monitoring result.");
    } finally { setBusy(""); }
  }

  async function collect(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!collecting || !collectionOperation(collecting.id)?.can_act) return;
    const data = new FormData(event.currentTarget);
    setBusy(collecting.id); setError("");
    try {
      await startFormCollection(collecting, {
        programID: aggregate.program.id,
        periodStart: new Date(String(data.get("period_start"))).toISOString(),
        periodEnd: new Date(String(data.get("period_end"))).toISOString(),
        deadline: new Date(String(data.get("deadline"))).toISOString(),
      });
      setCollecting(null);
      setNotice("Collection request created for the current responder and reviewer.");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The collection request could not be created.");
    } finally { setBusy(""); }
  }

  const today = new Date();
  const periodStart = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 6).toISOString().slice(0, 10);
  const periodEnd = today.toISOString().slice(0, 10);
  const deadline = new Date(today.getTime() + 2 * 86400000).toISOString().slice(0, 16);

  return <section className="monitoring-setup" aria-labelledby={`monitoring-${aggregate.program.id}`}>
    <div className="monitoring-section-heading"><div><h3 id={`monitoring-${aggregate.program.id}`}>Monitoring</h3><p>Collect responses or check connected data and calculate risk.</p></div>{canConfigureMonitoring && <button className="secondary-button" type="button" onClick={() => setMode(mode === "closed" ? "choose" : "closed")}>{mode === "closed" ? "Add monitoring check" : "Close setup"}</button>}</div>
    {!canConfigureMonitoring && <p className="program-operation-reason">{formDefineOperation?.reason ?? checkDefineOperation?.reason ?? "Monitoring changes are disabled until current Program responsibilities are available. Existing checks and results remain available."}</p>}
    {state === "loading" && <p aria-live="polite">Loading monitoring checks…</p>}
    {state === "unavailable" && <div className="inline-error"><p>Monitoring checks could not be loaded.</p><button className="secondary-button" type="button" onClick={() => void reload()}>Try again</button></div>}
    {error && <p className="inline-form-error" role="alert">{error}</p>}
    {notice && <Notice tone="success">{notice}</Notice>}
    {canConfigureMonitoring && mode === "choose" && <div className="monitoring-choice-grid">
      <button type="button" aria-label="Collection form" disabled={!canDefineForm} onClick={() => setMode("form")}><strong>Collection form</strong><span>{canDefineForm ? "Ask assigned staff for structured responses and score the answers." : formDefineOperation?.reason ?? "The current Program owner must create this form."}</span></button>
      <button type="button" aria-label="Connected data" disabled={!canDefineCheck || !canConfigureSources} onClick={() => setMode("source")}><strong>Connected data</strong><span>{!canDefineCheck ? checkDefineOperation?.reason ?? "The current Program owner must add this check." : canConfigureSources ? "Check a status endpoint and calculate risk from the returned value." : "A GRC administrator can connect a new source."}</span></button>
    </div>}
    {canDefineForm && mode === "form" && (
      <Suspense fallback={<p className="monitoring-builder-loading" role="status">Loading form editor…</p>}>
        <FormBuilder programID={aggregate.program.id} onCancel={() => setMode("choose")} onSaved={(form) => { setForms((current) => latestByID([...current, form])); setMode("closed"); setNotice("Form draft saved."); }}/>
      </Suspense>
    )}
    {canDefineCheck && mode === "source" && (
      <DataSourceBuilder onCancel={() => setMode("choose")} onSaved={(binding, config) => void addSourceCheck(binding, config)}/>
    )}
    {state === "live" && <div className="monitoring-records">
      {!forms.length && !checks.length && mode === "closed" && <p className="monitoring-empty">No monitoring checks have been added to this Program.</p>}
      {forms.map((form) => <article className="monitoring-record" key={form.id}>
        <div><span className="record-type">Collection form</span><h4>{form.name}</h4><p>{form.fields.length} question{form.fields.length === 1 ? "" : "s"} · {statusLabel(form.status)}</p></div>
        <div className="record-actions">
          {formTransitionOperation(form.id)?.can_act && formTransitionOperation(form.id)?.allowed_targets?.includes("PENDING_APPROVAL") && form.status === "DRAFT" && <button className="secondary-button" disabled={busy === form.id} onClick={() => void changeForm(form, "PENDING_APPROVAL")}>Send for approval</button>}
          {formTransitionOperation(form.id)?.can_act && formTransitionOperation(form.id)?.allowed_targets?.includes("ACTIVE") && form.status === "PENDING_APPROVAL" && form.submitted_by !== actorPrincipalID && <button className="primary-button" disabled={busy === form.id} onClick={() => void changeForm(form, "ACTIVE")}>Approve form</button>}
          {form.status === "PENDING_APPROVAL" && form.submitted_by === actorPrincipalID && <span className="action-note">Another approver must approve this form.</span>}
          {canDefineCheck && form.status === "ACTIVE" && hasScoredQuestions(form) && !linkedFormVersions.has(`${form.id}:${form.version}`) && <button className="secondary-button" disabled={busy === form.id} onClick={() => void addCheck(form)}>Create monitoring check</button>}
          {collectionOperation(form.id)?.can_act && form.status === "ACTIVE" && activeFormVersions.has(`${form.id}:${form.version}`) && <button className="primary-button" disabled={busy === form.id} onClick={() => setCollecting(form)}>Collect responses</button>}
        </div>
        {canDefineCheck && form.status === "ACTIVE" && !hasScoredQuestions(form) && <p className="program-operation-reason">This active revision has no scored questions. Create and approve a scored revision in Forms before adding a monitoring check.</p>}
        {collecting?.id === form.id && <form className="collection-schedule" onSubmit={collect}>
          <label><span>Period starts</span><input name="period_start" type="date" defaultValue={periodStart} required/></label>
          <label><span>Period ends</span><input name="period_end" type="date" defaultValue={periodEnd} required/></label>
          <label><span>Due</span><input name="deadline" type="datetime-local" defaultValue={deadline} required/></label>
          <p>The request will be assigned to the current responder and kept separate from its reviewer.</p>
          <button className="text-button" type="button" onClick={() => setCollecting(null)}>Cancel</button><button className="primary-button" type="submit" disabled={busy === form.id}>Create request</button>
        </form>}
      </article>)}
      {checks.map((check) => {
        const result = latestResults[check.id];
        const transitionOperation = checkOperation(check.id, "program.monitoring.transition");
        const evaluateOperation = checkOperation(check.id, "program.monitoring.evaluate");
        const issueOperation = checkOperation(check.id, "program.monitoring.issue.create");
        const issueEligible = eligibleForLinkedIssue(check, result);
        return <article className="monitoring-record" key={check.id}>
        <div><span className="record-type">Monitoring check</span><h4>{check.name}</h4><p>{check.input_kind === "FORM" ? "Collection form" : "Connected data"} · {statusLabel(check.status)}</p></div>
        {result && <div className="monitoring-result" aria-label={`Latest result for ${check.name}`}>
          <strong>{riskLabel(result)}</strong>
          <span className={`risk-band risk-band-${result.evaluation.band.toLowerCase()}`}>{bandLabel(result)}</span>
          <span>{Math.round(result.evaluation.coverage * 100)}% coverage</span>
          <time dateTime={result.evaluated_at}>{new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(result.evaluated_at))}</time>
        </div>}
        <div className="record-actions">
          {transitionOperation?.can_act && transitionOperation.allowed_targets?.includes("PENDING_APPROVAL") && <button className="secondary-button" disabled={busy === check.id} onClick={() => void changeCheck(check, "PENDING_APPROVAL")}>Send for approval</button>}
          {transitionOperation?.can_act && transitionOperation.allowed_targets?.includes("ACTIVE") && check.status === "PENDING_APPROVAL" && check.submitted_by !== actorPrincipalID && <button className="primary-button" disabled={busy === check.id} onClick={() => void changeCheck(check, "ACTIVE")}>Approve check</button>}
          {transitionOperation?.can_act && transitionOperation.allowed_targets?.includes("REJECTED") && <button className="secondary-button" disabled={busy === check.id} onClick={() => void changeCheck(check, "REJECTED")}>Return check</button>}
          {transitionOperation?.can_act && transitionOperation.allowed_targets?.includes("PAUSED") && <button className="secondary-button" disabled={busy === check.id} onClick={() => void changeCheck(check, "PAUSED")}>Pause check</button>}
          {transitionOperation?.can_act && transitionOperation.allowed_targets?.includes("ACTIVE") && check.status === "PAUSED" && <button className="primary-button" disabled={busy === check.id} onClick={() => void changeCheck(check, "ACTIVE")}>Resume check</button>}
          {transitionOperation?.can_act && transitionOperation.allowed_targets?.includes("RETIRED") && <button className="secondary-button" disabled={busy === check.id} onClick={() => void changeCheck(check, "RETIRED")}>End check</button>}
          {check.status === "PENDING_APPROVAL" && check.submitted_by === actorPrincipalID && <span className="action-note">Another approver must approve this check.</span>}
          {evaluateOperation?.can_act && check.status === "ACTIVE" && check.input_kind === "SOURCE" && <button className="primary-button" disabled={busy === check.id} onClick={() => void runSourceCheck(check)}>{busy === check.id ? "Checking…" : "Check source now"}</button>}
          {issueEligible && issueOperation && <button className="primary-button" type="button" disabled={!issueOperation.can_act || busy === result!.id} onClick={() => void createLinkedIssue(check, result!)}>{busy === result!.id ? "Creating linked issue…" : "Create linked issue"}</button>}
        </div>
        {!transitionOperation?.can_act && transitionOperation?.reason && <p className="program-operation-reason">{transitionOperation.reason}</p>}
        {!evaluateOperation?.can_act && evaluateOperation?.reason && <p className="program-operation-reason">{evaluateOperation.reason}</p>}
        {issueEligible && issueOperation && !issueOperation.can_act && <p className="program-operation-reason">{issueOperation.reason}</p>}
        {result && <details className="monitoring-result-detail">
          <summary>Review result</summary>
          {result.evaluation.band === "NOT_ASSESSED" ? <p>The available response or source data was not sufficient to calculate risk.</p> : (() => {
            const exceptions = (result.evaluation.rule_results ?? []).filter((rule) => rule.outcome !== "PASS");
            if (!exceptions.length) return <p>All configured checks passed.</p>;
            return <ul>{exceptions.map((rule) => <li key={`${rule.rule_id ?? "form"}-${rule.field_id}`}><strong>{check.form_template_id ? formFields.get(`${check.form_template_id}:${rule.field_id}`) ?? rule.field_id : rule.field_id}</strong><span>{rule.reason}</span></li>)}</ul>;
          })()}
        </details>}
      </article>;})}
    </div>}
  </section>;
}
