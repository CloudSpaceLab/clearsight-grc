import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { createFormMonitoringCheck, createSourceMonitoringCheck, evaluateMonitoringSource, loadFormTemplates, loadMonitoringChecks, loadMonitoringResults, startFormCollection, transitionFormTemplate, transitionMonitoringCheck } from "../monitoringApi";
import type { FormTemplate, LifecycleStatus, MonitoringCheck, MonitoringResult } from "../monitoringTypes";
import type { ProgramAggregate } from "../types";
import { FormBuilder } from "./FormBuilder";
import { DataSourceBuilder } from "./DataSourceBuilder";
import type { SourceBinding } from "../sourceConfigApi";
import type { ProgramOperation } from "../programOperationsApi";

type Props = { aggregate: ProgramAggregate; actorPrincipalID: string; canConfigureSources: boolean; operations: ProgramOperation[] };
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

export function MonitoringSetup({ aggregate, actorPrincipalID, canConfigureSources, operations }: Props) {
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
    const [formResult, checkResult] = await Promise.allSettled([loadFormTemplates(), loadMonitoringChecks(aggregate.program.id)]);
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
  const defineOperation = operations.find((operation) => operation.command === "program.monitoring.define" && !operation.subresource_id);
  const canDefine = Boolean(defineOperation?.can_act);
  const canReviewForms = operations.some((operation) => operation.command === "program.monitoring.transition" && operation.responsibility === "REVIEWER" && operation.can_act);
  const checkOperation = (checkID: string, command: string) => operations.find((operation) => operation.command === command && operation.subresource_id === checkID);

  useEffect(() => {
    if (!canDefine) {
      setMode("closed");
      setCollecting(null);
    }
  }, [canDefine]);

  const linkedFormVersions = useMemo(() => new Set(checks.filter((check) => check.input_kind === "FORM").map((check) => `${check.form_template_id}:${check.form_template_version}`)), [checks]);
  const activeFormVersions = useMemo(() => new Set(checks.filter((check) => check.input_kind === "FORM" && check.status === "ACTIVE" && check.is_current).map((check) => `${check.form_template_id}:${check.form_template_version}`)), [checks]);
  const formFields = useMemo(() => new Map(forms.flatMap((form) => form.fields.map((field) => [`${form.id}:${field.id}`, field.label] as const))), [forms]);

  async function changeForm(form: FormTemplate, to: LifecycleStatus) {
    if (to === "ACTIVE" ? !canReviewForms : !canDefine) return;
    setBusy(form.id); setError("");
    try {
      const updated = await transitionFormTemplate(form.id, form.version, to);
      setForms((current) => latestByID([...current, updated]));
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The form status could not be changed.");
    } finally { setBusy(""); }
  }

  async function addCheck(form: FormTemplate) {
    if (!canDefine) return;
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
    if (!canDefine) return;
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
      const updated = await transitionMonitoringCheck(check.id, check.version, to);
      setChecks((current) => latestByID([...current, updated]));
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The monitoring check status could not be changed.");
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

  async function collect(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canDefine || !collecting) return;
    const data = new FormData(event.currentTarget);
    setBusy(collecting.id); setError("");
    try {
      await startFormCollection(collecting, {
        programID: aggregate.program.id, respondentPrincipalID: actorPrincipalID, reviewerPrincipalID: actorPrincipalID,
        periodStart: new Date(String(data.get("period_start"))).toISOString(),
        periodEnd: new Date(String(data.get("period_end"))).toISOString(),
        deadline: new Date(String(data.get("deadline"))).toISOString(),
      });
      setCollecting(null);
      setNotice("Collection request created and assigned to you.");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The collection request could not be created.");
    } finally { setBusy(""); }
  }

  const today = new Date();
  const periodStart = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 6).toISOString().slice(0, 10);
  const periodEnd = today.toISOString().slice(0, 10);
  const deadline = new Date(today.getTime() + 2 * 86400000).toISOString().slice(0, 16);

  return <section className="monitoring-setup" aria-labelledby={`monitoring-${aggregate.program.id}`}>
    <div className="monitoring-section-heading"><div><h3 id={`monitoring-${aggregate.program.id}`}>Monitoring</h3><p>Collect responses or check connected data and calculate risk.</p></div>{canDefine && <button className="secondary-button" type="button" onClick={() => setMode(mode === "closed" ? "choose" : "closed")}>{mode === "closed" ? "Add monitoring check" : "Close setup"}</button>}</div>
    {!canDefine && <p className="program-operation-reason">{defineOperation?.reason ?? "Monitoring changes are disabled until current Program responsibilities are available. Existing checks and results remain available."}</p>}
    {state === "loading" && <p aria-live="polite">Loading monitoring checks…</p>}
    {state === "unavailable" && <div className="inline-error"><p>Monitoring checks could not be loaded.</p><button className="secondary-button" type="button" onClick={() => void reload()}>Try again</button></div>}
    {error && <p className="inline-form-error" role="alert">{error}</p>}
    {notice && <p className="inline-success" role="status">{notice}</p>}
    {canDefine && mode === "choose" && <div className="monitoring-choice-grid">
      <button type="button" aria-label="Collection form" onClick={() => setMode("form")}><strong>Collection form</strong><span>Ask assigned staff for structured responses and score the answers.</span></button>
      <button type="button" aria-label="Connected data" disabled={!canConfigureSources} onClick={() => setMode("source")}><strong>Connected data</strong><span>{canConfigureSources ? "Check a status endpoint and calculate risk from the returned value." : "A GRC administrator can connect a new source."}</span></button>
    </div>}
    {canDefine && mode === "form" && (
      <FormBuilder onCancel={() => setMode("choose")} onSaved={(form) => { setForms((current) => latestByID([...current, form])); setMode("closed"); setNotice("Form draft saved."); }}/>
    )}
    {canDefine && mode === "source" && (
      <DataSourceBuilder onCancel={() => setMode("choose")} onSaved={(binding, config) => void addSourceCheck(binding, config)}/>
    )}
    {state === "live" && <div className="monitoring-records">
      {!forms.length && !checks.length && mode === "closed" && <p className="monitoring-empty">No monitoring checks have been added to this Program.</p>}
      {forms.map((form) => <article className="monitoring-record" key={form.id}>
        <div><span className="record-type">Collection form</span><h4>{form.name}</h4><p>{form.fields.length} question{form.fields.length === 1 ? "" : "s"} · {statusLabel(form.status)}</p></div>
        <div className="record-actions">
          {canDefine && form.status === "DRAFT" && <button className="secondary-button" disabled={busy === form.id} onClick={() => void changeForm(form, "PENDING_APPROVAL")}>Send for approval</button>}
          {canReviewForms && form.status === "PENDING_APPROVAL" && form.submitted_by !== actorPrincipalID && <button className="primary-button" disabled={busy === form.id} onClick={() => void changeForm(form, "ACTIVE")}>Approve form</button>}
          {form.status === "PENDING_APPROVAL" && form.submitted_by === actorPrincipalID && <span className="action-note">Another approver must approve this form.</span>}
          {canDefine && form.status === "ACTIVE" && !linkedFormVersions.has(`${form.id}:${form.version}`) && <button className="secondary-button" disabled={busy === form.id} onClick={() => void addCheck(form)}>Create monitoring check</button>}
          {canDefine && form.status === "ACTIVE" && activeFormVersions.has(`${form.id}:${form.version}`) && <button className="primary-button" disabled={busy === form.id} onClick={() => setCollecting(form)}>Collect responses</button>}
        </div>
        {collecting?.id === form.id && <form className="collection-schedule" onSubmit={collect}>
          <label><span>Period starts</span><input name="period_start" type="date" defaultValue={periodStart} required/></label>
          <label><span>Period ends</span><input name="period_end" type="date" defaultValue={periodEnd} required/></label>
          <label><span>Due</span><input name="deadline" type="datetime-local" defaultValue={deadline} required/></label>
          <p>The request will be assigned to you. It can be reassigned from Evidence.</p>
          <button className="text-button" type="button" onClick={() => setCollecting(null)}>Cancel</button><button className="primary-button" type="submit" disabled={busy === form.id}>Create request</button>
        </form>}
      </article>)}
      {checks.map((check) => {
        const result = latestResults[check.id];
        const transitionOperation = checkOperation(check.id, "program.monitoring.transition");
        const evaluateOperation = checkOperation(check.id, "program.monitoring.evaluate");
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
        </div>
        {!transitionOperation?.can_act && transitionOperation?.reason && <p className="program-operation-reason">{transitionOperation.reason}</p>}
        {!evaluateOperation?.can_act && evaluateOperation?.reason && <p className="program-operation-reason">{evaluateOperation.reason}</p>}
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
