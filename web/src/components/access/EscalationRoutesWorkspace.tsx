import { useEffect, useMemo, useState, type FormEvent } from "react";
import {
  approveEscalationSequenceRevision,
  proposeEscalationSequenceRevision,
  restoreEscalationSequenceRevision,
  simulateEscalation,
  type EscalationPolicy,
  type EscalationSequence,
  type EscalationSequenceStepInput,
  type EscalationSimulation,
  type IdentityAccessOverview,
} from "../../identityAccessApi";
import { Button, CheckboxField, Notice, SelectField, StatusBadge, TextField, type StatusTone } from "../ui";

type Props = {
  overview: IdentityAccessOverview;
  isBusy: boolean;
  onBusy: (value: boolean) => void;
  onNotice: (message: string) => void;
  onRefresh: () => Promise<void>;
};

type DraftStep = EscalationSequenceStepInput & { department_levels_up_text: string };

const responsibilityOptions = [
  "PERFORMER",
  "ACCOUNTABLE_OWNER",
  "PROPOSER",
  "REVIEWER",
  "INDEPENDENT_CHALLENGER",
  "AUTHORIZER",
  "SIGNATORY",
  "TRANSMITTER",
  "ACKNOWLEDGEMENT_RECORDER",
  "ESCALATION_OWNER",
].map((id) => ({ id, label: humanize(id) }));

const recoveryOptions = [
  { id: "REVIEW_ROUTE", label: "Review route" },
  { id: "REASSIGN_OWNER", label: "Reassign owner" },
  { id: "RESOLVE_WORK", label: "Resolve source work" },
] as const;

const statusTone: Record<string, StatusTone> = {
  RESOLVED: "success",
  CANDIDATE_SET: "warning",
  NO_ROUTE: "danger",
  AMBIGUOUS_ROUTE: "danger",
  NO_VISIBLE_CANDIDATE: "danger",
  TARGET_CONSTRAINT_NO_MATCH: "danger",
  SOURCE_ROLE_NOT_ALLOWED: "warning",
  DEPARTMENT_SCOPE_UNRESOLVED: "warning",
  DEPARTMENT_ANCESTRY_EXHAUSTED: "warning",
};

export function EscalationRoutesWorkspace({ overview, isBusy, onBusy, onNotice, onRefresh }: Props) {
  const [policyID, setPolicyID] = useState("");
  const [sequenceID, setSequenceID] = useState("");
  const [draftSequenceID, setDraftSequenceID] = useState("overdue");
  const [steps, setSteps] = useState<DraftStep[]>([]);
  const [recoveryAction, setRecoveryAction] = useState("REVIEW_ROUTE");
  const [simulation, setSimulation] = useState<EscalationSimulation>();
  const [approvalRationale, setApprovalRationale] = useState("");
  const [restoreVersion, setRestoreVersion] = useState("");
  const [editingLevel, setEditingLevel] = useState("0");

  const selectedPolicy = useMemo(
    () => overview.escalation_policies.find((policy) => policy.policy_id === policyID),
    [overview.escalation_policies, policyID],
  );
  const pending = selectedPolicy?.pending_revision;
  const effectiveSequences = pending?.sequences ?? selectedPolicy?.sequences ?? [];
  const selectedSequence = effectiveSequences.find((sequence) => sequence.ID === sequenceID);
  const pendingFromAnotherMaker = Boolean(pending && pending.maker_id !== overview.actor_principal_id);
  const selectedLevel = Math.min(Number(editingLevel) || 0, Math.max(0, steps.length - 1));
  const activeStep = steps[selectedLevel];

  useEffect(() => {
    const first = overview.escalation_policies[0];
    if (!policyID || !overview.escalation_policies.some((policy) => policy.policy_id === policyID)) {
      setPolicyID(first?.policy_id ?? "");
    }
  }, [overview.escalation_policies, policyID]);

  useEffect(() => {
    const first = effectiveSequences[0];
    if (first && !effectiveSequences.some((sequence) => sequence.ID === sequenceID)) {
      setSequenceID(first.ID);
      return;
    }
    if (!first) setSequenceID("");
  }, [effectiveSequences, sequenceID]);

  useEffect(() => {
    const sequence = effectiveSequences.find((value) => value.ID === sequenceID);
    if (sequence) {
      setDraftSequenceID(sequence.ID);
      setSteps(sequence.Steps.map(stepFromSequence));
      setRecoveryAction(sequence.RecoveryAction || "REVIEW_ROUTE");
    } else {
      setDraftSequenceID("overdue");
      setSteps(defaultSteps());
      setRecoveryAction("REVIEW_ROUTE");
    }
    setEditingLevel("0");
    setSimulation(undefined);
  }, [policyID, sequenceID, pending?.version]);

  async function run(action: () => Promise<void>) {
    onBusy(true);
    onNotice("");
    try {
      await action();
    } catch (error) {
      onNotice(error instanceof Error ? error.message : "Escalation configuration could not be updated.");
    } finally {
      onBusy(false);
    }
  }

  async function saveSequence(event: FormEvent) {
    event.preventDefault();
    if (!selectedPolicy || !draftSequenceID.trim() || steps.length === 0) return;
    const normalized = steps.map((step) => ({
      after: step.after.trim(),
      responsibility: step.responsibility,
      department_levels_up: optionalInteger(step.department_levels_up_text),
      source_roles: step.source_roles ?? [],
      target_roles: step.target_roles ?? [],
      target_group_ids: step.target_group_ids ?? [],
      target_position_ids: step.target_position_ids ?? [],
    }));
    await run(async () => {
      const revision = await proposeEscalationSequenceRevision({
        policy_id: selectedPolicy.policy_id,
        sequence_id: draftSequenceID.trim(),
        steps: normalized,
        terminal_handling: "KEEP_OPEN",
        recovery_action: recoveryAction,
        expected_policy_version: selectedPolicy.record_version,
      });
      onNotice(`Revision v${revision.version} proposed. Current routing stays active until independent approval.`);
      await onRefresh();
    });
  }

  async function approve(event: FormEvent) {
    event.preventDefault();
    if (!selectedPolicy || !pending || !approvalRationale.trim()) return;
    await run(async () => {
      await approveEscalationSequenceRevision(selectedPolicy.policy_id, pending.version, {
        expected_policy_version: selectedPolicy.record_version,
        rationale: approvalRationale.trim(),
      });
      setApprovalRationale("");
      onNotice(`Revision v${pending.version} activated.`);
      await onRefresh();
    });
  }

  async function restore() {
    if (!selectedPolicy) return;
    const sourceVersion = Number(restoreVersion);
    if (!Number.isInteger(sourceVersion) || sourceVersion < 1 || sourceVersion >= selectedPolicy.version) return;
    await run(async () => {
      const revision = await restoreEscalationSequenceRevision(selectedPolicy.policy_id, {
        source_version: sourceVersion,
        expected_policy_version: selectedPolicy.record_version,
      });
      onNotice(`Version ${sourceVersion} copied into revision v${revision.version} for independent approval.`);
      setRestoreVersion("");
      await onRefresh();
    });
  }

  async function checkImpact() {
    if (!selectedPolicy || !draftSequenceID.trim()) return;
    await run(async () => {
      const result = await simulateEscalation({
        policy_id: selectedPolicy.policy_id,
        sequence_id: draftSequenceID.trim(),
        revision_version: pending?.version,
        limit: 25,
      });
      setSimulation(result);
      onNotice(result.checked ? `Checked ${result.checked} active work item${result.checked === 1 ? "" : "s"}.` : "No active work currently uses this route.");
    });
  }

  function updateStep(patch: Partial<DraftStep>) {
    setSteps((current) => current.map((step, index) => index === selectedLevel ? { ...step, ...patch } : step));
    setSimulation(undefined);
  }

  function addLevel() {
    if (steps.length >= 8) return;
    const last = steps.at(-1);
    setSteps((current) => [...current, {
      after: nextDelay(last?.after),
      responsibility: "ESCALATION_OWNER",
      department_levels_up_text: "",
      source_roles: [],
      target_roles: [],
      target_group_ids: [],
      target_position_ids: [],
    }]);
    setEditingLevel(String(steps.length));
    setSimulation(undefined);
  }

  function removeLevel() {
    if (steps.length <= 1) return;
    setSteps((current) => current.filter((_, index) => index !== selectedLevel));
    setEditingLevel(String(Math.max(0, selectedLevel - 1)));
    setSimulation(undefined);
  }

  const levelOptions = steps.map((step, index) => ({ id: String(index), label: `Level ${index + 1} · ${humanize(step.responsibility)}` }));
  const sequenceOptions = effectiveSequences.map((sequence) => ({ id: sequence.ID, label: `${sequence.ID} · ${humanize(sequence.Trigger)}` }));
  const restoreOptions = selectedPolicy ? Array.from({ length: Math.max(0, selectedPolicy.version - 1) }, (_, index) => {
    const version = selectedPolicy.version - index - 1;
    return { id: String(version), label: `Version ${version}` };
  }) : [];

  return <div className="identity-access-grid identity-access-grid-single">
    <article className="config-card identity-escalation-card">
      <div className="section-header">
        <div>
          <h3>Escalation routes</h3>
          <p>Configure overdue levels, check current recipients, then submit for independent approval.</p>
        </div>
        {pending && <StatusBadge tone="warning">Revision v{pending.version} pending</StatusBadge>}
      </div>

      <div className="identity-metrics">
        <div><strong>{overview.escalation.escalated_tasks}</strong><span>Escalated work</span></div>
        <div><strong>{overview.escalation.pending_timers}</strong><span>Pending levels</span></div>
        <div><strong>{overview.escalation.unresolved_24h}</strong><span>Unresolved · 24h</span></div>
        <div><strong>{overview.escalation.failed_timers}</strong><span>Failed timers</span></div>
      </div>

      {overview.escalation_policies.length === 0 ? <Notice tone="info">No active routing policy is available in this legal entity.</Notice> : <>
        <div className="identity-inline-form">
          <SelectField
            label="Policy"
            value={policyID || undefined}
            placeholder="Choose policy"
            allowsEmpty={false}
            options={overview.escalation_policies.map(policyOption)}
            onChange={(value) => { setPolicyID(value ?? ""); setSequenceID(""); setSimulation(undefined); }}
          />
          {effectiveSequences.length > 0 && <SelectField
            label="Sequence"
            value={sequenceID || undefined}
            placeholder="Choose sequence"
            allowsEmpty={false}
            options={sequenceOptions}
            onChange={(value) => setSequenceID(value ?? "")}
          />}
          {!selectedSequence && <TextField label="Sequence code" value={draftSequenceID} onChange={setDraftSequenceID} placeholder="overdue"/>}
          {selectedPolicy && <p className="muted-copy">Active v{selectedPolicy.version}{selectedPolicy.effective_from ? ` · since ${formatDateTime(selectedPolicy.effective_from)}` : ""}</p>}
        </div>

        {overview.can_configure_escalation && selectedPolicy && !pendingFromAnotherMaker && <form className="identity-inline-form identity-guard-editor" onSubmit={(event) => void saveSequence(event)}>
          <div className="section-header">
            <div><h4>{pending ? `Edit proposed revision v${pending.version}` : selectedSequence ? "Edit sequence" : "Create sequence"}</h4><p>Levels run from the original due time. Each later level must use a longer delay.</p></div>
            <div className="identity-guard-actions">
              <Button type="button" size="compact" onPress={addLevel} isDisabled={isBusy || steps.length >= 8}>Add level</Button>
              <Button type="button" variant="quiet" size="compact" onPress={removeLevel} isDisabled={isBusy || steps.length <= 1}>Remove</Button>
            </div>
          </div>
          <SelectField label="Level" value={editingLevel} placeholder="Choose level" allowsEmpty={false} options={levelOptions} onChange={(value) => setEditingLevel(value ?? "0")}/>
          {activeStep && <>
            <TextField label="Escalate after" value={activeStep.after} onChange={(after) => updateStep({ after })} placeholder="1h" description="Examples: 30m, 2h, 24h" isRequired/>
            <SelectField label="Responsibility" value={activeStep.responsibility} placeholder="Choose responsibility" allowsEmpty={false} options={responsibilityOptions} onChange={(responsibility) => responsibility && updateStep({ responsibility })}/>
            <TextField label="Department levels up" type="number" min="0" max="8" value={activeStep.department_levels_up_text} onChange={(department_levels_up_text) => updateStep({ department_levels_up_text })} placeholder="Legal entity" description="Leave blank for legal-entity scope."/>
            <GuardChoices
              title="Originating roles"
              values={activeStep.source_roles ?? []}
              options={overview.roles.map((role) => ({ id: role.code, label: role.name }))}
              onChange={(source_roles) => updateStep({ source_roles })}
            />
            <GuardChoices
              title="Allowed recipient roles"
              values={activeStep.target_roles ?? []}
              options={overview.roles.map((role) => ({ id: role.code, label: role.name }))}
              onChange={(target_roles) => updateStep({ target_roles })}
            />
            <GuardChoices
              title="Allowed recipient groups"
              values={activeStep.target_group_ids ?? []}
              options={overview.groups.filter((group) => group.source_state === "ACTIVE").map((group) => ({ id: group.id, label: group.display_name }))}
              onChange={(target_group_ids) => updateStep({ target_group_ids })}
            />
            <GuardChoices
              title="Allowed recipient positions"
              values={activeStep.target_position_ids ?? []}
              options={overview.positions.map((position) => ({ id: position.id, label: position.title }))}
              onChange={(target_position_ids) => updateStep({ target_position_ids })}
            />
          </>}
          <SelectField label="If the final level cannot resolve the work" value={recoveryAction} placeholder="Choose recovery" allowsEmpty={false} options={recoveryOptions} onChange={(value) => value && setRecoveryAction(value)}/>
          <div className="identity-guard-actions">
            <Button type="submit" isLoading={isBusy}>Propose revision</Button>
            <Button type="button" variant="secondary" onPress={() => void checkImpact()} isDisabled={isBusy || !draftSequenceID.trim()}>Check current work</Button>
          </div>
        </form>}

        {pendingFromAnotherMaker && <form className="identity-inline-form identity-guard-approval" onSubmit={(event) => void approve(event)}>
          <h4>Independent approval</h4>
          <p className="muted-copy">Approval rechecks role, group and position references before activation.</p>
          <TextField label="Approval rationale" value={approvalRationale} onChange={setApprovalRationale} placeholder="Reviewed timing, recipients and unresolved routes" isRequired/>
          <Button type="submit" isLoading={isBusy} isDisabled={!approvalRationale.trim()}>Approve & activate v{pending?.version}</Button>
        </form>}
        {pending && !pendingFromAnotherMaker && <Notice tone="info">Revision v{pending.version} is awaiting approval by another authorized administrator.</Notice>}

        {selectedPolicy && selectedPolicy.version > 1 && !pending && overview.can_configure_escalation && <div className="identity-inline-form">
          <h4>Restore an earlier version</h4>
          <SelectField label="Version" value={restoreVersion || undefined} placeholder="Choose version" options={restoreOptions} onChange={(value) => setRestoreVersion(value ?? "")}/>
          <Button type="button" variant="secondary" onPress={() => void restore()} isLoading={isBusy} isDisabled={!restoreVersion}>Propose restore</Button>
        </div>}

        {simulation && <SimulationResults value={simulation}/>}
      </>}
      <p className="identity-footnote">Escalation changes assignment only when the current authority route resolves one eligible recipient. It does not grant approval, review, challenge, authorization or signing authority.</p>
    </article>
  </div>;
}

function GuardChoices({ title, values, options, onChange }: { title: string; values: string[]; options: Array<{ id: string; label: string }>; onChange: (values: string[]) => void }) {
  if (!options.length) return <div className="identity-guard-summary open"><strong>{title}</strong><span>No current options.</span></div>;
  return <fieldset className="identity-guard-choices">
    <legend>{title} <span>(optional)</span></legend>
    <div className="identity-guard-choice-list">
      {options.map((option) => <CheckboxField
        key={option.id}
        label={option.label}
        isSelected={values.includes(option.id)}
        onChange={(selected) => onChange(selected ? [...values, option.id] : values.filter((value) => value !== option.id))}
      />)}
    </div>
  </fieldset>;
}

function SimulationResults({ value }: { value: EscalationSimulation }) {
  return <section className="identity-escalation-simulation" aria-label="Current work simulation">
    <div className="section-header"><div><h4>Current work</h4><p>{value.checked} item{value.checked === 1 ? "" : "s"} checked against sequence v{value.sequence_version}.{value.truncated ? " Showing the first 25." : ""}</p></div></div>
    {value.scenarios.length === 0 ? <Notice tone="info">No active work currently uses this routing policy.</Notice> :
      <div className="identity-preview">
        {value.scenarios.map((scenario) => <article key={scenario.task_id} className="identity-escalation-scenario">
          <div className="section-header"><div><strong>{scenario.title}</strong><small>{scenario.current_principal_name || "No current assignee"} · due {formatDateTime(scenario.due_at)}</small></div>{scenario.next_level > 0 && <StatusBadge tone="info">Next level {scenario.next_level}</StatusBadge>}</div>
          {scenario.next_due_at && <p className="muted-copy">Next due {formatDateTime(scenario.next_due_at)} · {humanize(scenario.recovery_action)}</p>}
          <ol>{scenario.steps.map((step) => <li key={step.index}>
            <span>Level {step.index + 1} · {formatDateTime(step.due_at)}</span>
            <strong>{humanize(step.responsibility)}</strong>
            <StatusBadge tone={statusTone[step.status] ?? "neutral"}>{humanize(step.status)}</StatusBadge>
            {step.candidates.length > 0 && <small>{step.candidates.map((candidate) => candidate.display_name || candidate.principal_id).join(", ")}</small>}
          </li>)}</ol>
        </article>)}
      </div>}
  </section>;
}

function policyOption(policy: EscalationPolicy) {
  return { id: policy.policy_id, label: `${policy.code} · active v${policy.version}${policy.pending_revision ? ` · proposed v${policy.pending_revision.version}` : ""}` };
}

function stepFromSequence(step: EscalationSequence["Steps"][number]): DraftStep {
  return {
    after: formatDuration(step.After),
    responsibility: step.Responsibility,
    department_levels_up: step.DepartmentLevelsUp,
    department_levels_up_text: step.DepartmentLevelsUp == null ? "" : String(step.DepartmentLevelsUp),
    source_roles: step.SourceRoles ?? [],
    target_roles: step.TargetRoles ?? [],
    target_group_ids: step.TargetGroupIDs ?? [],
    target_position_ids: step.TargetPositionIDs ?? [],
  };
}

function defaultSteps(): DraftStep[] {
  return [
    { after: "30m", responsibility: "ACCOUNTABLE_OWNER", department_levels_up_text: "", source_roles: [], target_roles: [], target_group_ids: [], target_position_ids: [] },
    { after: "2h", responsibility: "ESCALATION_OWNER", department_levels_up_text: "1", source_roles: [], target_roles: [], target_group_ids: [], target_position_ids: [] },
    { after: "8h", responsibility: "AUTHORIZER", department_levels_up_text: "", source_roles: [], target_roles: [], target_group_ids: [], target_position_ids: [] },
  ];
}

function optionalInteger(value: string) {
  const trimmed = value.trim();
  if (!trimmed) return undefined;
  const parsed = Number(trimmed);
  return Number.isInteger(parsed) ? parsed : undefined;
}

function nextDelay(value?: string) {
  if (!value) return "1h";
  const match = value.trim().match(/^(\d+)(m|h)$/i);
  if (!match) return "24h";
  const amount = Number(match[1]);
  return match[2].toLowerCase() === "m" ? `${Math.max(60, amount * 2)}m` : `${Math.max(1, amount * 2)}h`;
}

function formatDuration(nanoseconds: number) {
  if (!Number.isFinite(nanoseconds) || nanoseconds < 0) return "0s";
  const second = 1_000_000_000;
  const minute = 60 * second;
  const hour = 60 * minute;
  if (nanoseconds % hour === 0) return `${nanoseconds / hour}h`;
  if (nanoseconds % minute === 0) return `${nanoseconds / minute}m`;
  if (nanoseconds % second === 0) return `${nanoseconds / second}s`;
  return `${nanoseconds}ns`;
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString([], { dateStyle: "medium", timeStyle: "short" });
}
