import { useMemo, useState } from "react";
import type {
  IdentityPerson,
  IdentityRole,
  OrganizationPosition,
  OrganizationPositionRevision,
  OrganizationPositionRoleRevision,
  OrganizationPositionRouteSimulation,
  OrganizationScope,
  ProposeOrganizationPositionInput,
  ProposeOrganizationPositionRoleInput,
} from "../../identityAccessApi";
import { Button, FocusedSheet, SelectField, StatusBadge, TextArea, TextField } from "../ui";

type EditorState =
  | { mode: "add" }
  | { mode: "edit" | "retire"; position: OrganizationPosition };

type DecisionState = {
  action: "approve" | "reject";
  revision: OrganizationPositionRevision;
};

type RoleDecisionState = {
  action: "approve" | "reject";
  revision: OrganizationPositionRoleRevision;
};

type Props = {
  positions: OrganizationPosition[];
  allPositions: OrganizationPosition[];
  scopes: OrganizationScope[];
  people: IdentityPerson[];
  revisions: OrganizationPositionRevision[];
  history: OrganizationPositionRevision[];
  roleRevisions: OrganizationPositionRoleRevision[];
  roles: IdentityRole[];
  actorPrincipalID: string;
  canConfigure: boolean;
  isBusy?: boolean;
  scopeByID: Map<string, OrganizationScope>;
  onPropose: (input: ProposeOrganizationPositionInput) => Promise<boolean>;
  onApprove: (revision: OrganizationPositionRevision, rationale: string) => Promise<boolean>;
  onReject: (revision: OrganizationPositionRevision, rationale: string) => Promise<boolean>;
  onSimulate?: (revision: OrganizationPositionRevision) => Promise<OrganizationPositionRouteSimulation | null>;
  onRestore?: (revision: OrganizationPositionRevision) => Promise<boolean>;
  onProposeRole?: (input: ProposeOrganizationPositionRoleInput) => Promise<boolean>;
  onApproveRole?: (revision: OrganizationPositionRoleRevision, rationale: string) => Promise<boolean>;
  onRejectRole?: (revision: OrganizationPositionRoleRevision, rationale: string) => Promise<boolean>;
};

export function OrganizationPositionManager({
  positions,
  allPositions,
  scopes,
  people,
  revisions,
  history,
  roleRevisions,
  roles,
  actorPrincipalID,
  canConfigure,
  isBusy = false,
  scopeByID,
  onPropose,
  onApprove,
  onReject,
  onSimulate,
  onRestore,
  onProposeRole,
  onApproveRole,
  onRejectRole,
}: Props) {
  const [editor, setEditor] = useState<EditorState>();
  const [decision, setDecision] = useState<DecisionState>();
  const [rolePosition, setRolePosition] = useState<OrganizationPosition>();
  const [roleDecision, setRoleDecision] = useState<RoleDecisionState>();
  const pendingByPosition = useMemo(
    () => new Map(revisions.filter((item) => item.status === "PENDING").map((item) => [item.position_id, item])),
    [revisions],
  );
  const pendingRoleByPosition = useMemo(
    () => new Map(roleRevisions.filter((item) => item.status === "PENDING").map((item) => [item.position_id, item])),
    [roleRevisions],
  );
  const roleByCode = useMemo(() => new Map(roles.map((role) => [role.code, role])), [roles]);
  const peopleByID = useMemo(() => new Map(people.map((person) => [person.id, person.display_name])), [people]);
  const positionByID = useMemo(() => new Map(allPositions.map((position) => [position.id, position])), [allPositions]);
  const visibleHistory = useMemo(() => history.slice(0, 20), [history]);

  return <div className="identity-position-manager">
    <div className="identity-scope-manager__header">
      <span>{allPositions.length} active positions</span>
      {canConfigure && <Button size="compact" onPress={() => setEditor({ mode: "add" })}>Add position</Button>}
    </div>

    {revisions.length > 0 && <div className="identity-scope-changes">
      <strong>Pending position changes</strong>
      <ul>{revisions.map((revision) => {
        const own = revision.maker_id === actorPrincipalID;
        return <li key={revision.id}>
          <div>
            <b>{positionRevisionLabel(revision, positionByID)}</b>
            <span>{positionImpactLabel(revision)}</span>
          </div>
          <StatusBadge tone="warning">Pending</StatusBadge>
          {canConfigure && !own && <div className="identity-scope-change-actions">
            <Button size="compact" variant="secondary" onPress={() => setDecision({ action: "approve", revision })}>Approve</Button>
            <Button size="compact" variant="quiet" onPress={() => setDecision({ action: "reject", revision })}>Reject</Button>
          </div>}
          {own && <small>Awaiting approval</small>}
        </li>;
      })}</ul>
    </div>}

    {roleRevisions.length > 0 && <div className="identity-scope-changes">
      <strong>Pending workspace role changes</strong>
      <ul>{roleRevisions.map((revision) => {
        const own = revision.maker_id === actorPrincipalID;
        const position = positionByID.get(revision.position_id);
        return <li key={revision.id}>
          <div>
            <b>{revision.operation === "ADD" ? "Add" : "Remove"} {revision.role_code}</b>
            <span>{position?.title ?? revision.position_id} · {capabilityLabel(revision.capabilities)}</span>
          </div>
          <StatusBadge tone="warning">Pending</StatusBadge>
          {canConfigure && !own && onApproveRole && onRejectRole && <div className="identity-scope-change-actions">
            <Button size="compact" variant="secondary" onPress={() => setRoleDecision({ action: "approve", revision })}>Approve</Button>
            <Button size="compact" variant="quiet" onPress={() => setRoleDecision({ action: "reject", revision })}>Reject</Button>
          </div>}
          {own && <small>Awaiting approval</small>}
        </li>;
      })}</ul>
    </div>}

    {visibleHistory.length > 0 && <details className="identity-scope-changes">
      <summary><strong>History</strong><span>{visibleHistory.length} recent change{visibleHistory.length === 1 ? "" : "s"}</span></summary>
      <ul>{visibleHistory.map((revision) => {
        const pending = pendingByPosition.get(revision.position_id);
        const canRestore = Boolean(onRestore && revision.status === "APPLIED" && revision.operation !== "RETIRE" && positionByID.has(revision.position_id));
        return <li key={revision.id}>
          <div>
            <b>{positionRevisionLabel(revision, positionByID)}</b>
            <span>{positionRevisionHistoryLabel(revision)}</span>
          </div>
          <StatusBadge tone={positionRevisionStatusTone(revision.status)}>{humanize(revision.status)}</StatusBadge>
          {canRestore && <div className="identity-scope-change-actions">
            <Button
              size="compact"
              variant="quiet"
              isDisabled={Boolean(pending) || Boolean(pendingRoleByPosition.get(revision.position_id)) || isBusy}
              onPress={() => void onRestore?.(revision)}
            >Restore</Button>
          </div>}
        </li>;
      })}</ul>
    </details>}

    {positions.length > 0 && <div className="identity-position-table-wrap">
      <table className="identity-position-table">
        <thead><tr><th>Position</th><th>Current occupant</th><th>Workspace roles</th><th>Reports to</th>{canConfigure && <th>Actions</th>}</tr></thead>
        <tbody>{positions.map((position) => {
          const pending = pendingByPosition.get(position.id);
          const pendingRole = pendingRoleByPosition.get(position.id);
          return <tr key={position.id}>
            <td data-label="Position"><strong>{position.title}</strong><span>{position.code} · {scopeLabel(position, scopeByID)}</span>{pending && <StatusBadge tone="warning">Pending change</StatusBadge>}</td>
            <td data-label="Current occupant">{position.occupant_name
              ? <><strong>{position.occupant_name}</strong><span>{humanize(position.occupant_status || "active")}</span></>
              : <span className="identity-vacancy">Vacant — coverage required</span>}</td>
            <td data-label="Workspace roles"><div className="identity-role-chips">{position.role_codes.length
              ? position.role_codes.map((code) => {
                const role = roleByCode.get(code);
                const locked = (role?.organization_lock_reasons?.length ?? 0) > 0;
                return <span key={code} title={locked ? lockReasonLabel(role?.organization_lock_reasons ?? []) : undefined}>{code}{locked ? " · locked" : ""}</span>;
              })
              : <span className="identity-empty-value">No role assigned</span>}</div>{pendingRole && <small>Role change pending</small>}</td>
            <td data-label="Reports to"><strong>{position.parent_position_title || "Top-level position"}</strong>{position.parent_position_code && <span>{position.parent_position_code}</span>}</td>
            {canConfigure && <td data-label="Actions"><div className="identity-scope-row-actions">
              <Button size="compact" variant="quiet" isDisabled={Boolean(pending) || Boolean(pendingRole) || isBusy} onPress={() => setEditor({ mode: "edit", position })}>Edit</Button>
              {onProposeRole && <Button size="compact" variant="quiet" isDisabled={Boolean(pending) || Boolean(pendingRole) || isBusy} onPress={() => setRolePosition(position)}>Roles</Button>}
              <Button size="compact" variant="quiet" isDisabled={Boolean(pending) || Boolean(pendingRole) || isBusy} onPress={() => setEditor({ mode: "retire", position })}>Retire</Button>
            </div></td>}
          </tr>;
        })}</tbody>
      </table>
    </div>}

    {editor && <OrganizationPositionEditor
      state={editor}
      positions={allPositions}
      scopes={scopes}
      people={people}
      isBusy={isBusy}
      onClose={() => setEditor(undefined)}
      onSubmit={async (input) => {
        const ok = await onPropose(input);
        if (ok) setEditor(undefined);
      }}
    />}

    {rolePosition && onProposeRole && <OrganizationPositionRoleEditor
      position={rolePosition}
      roles={roles}
      isBusy={isBusy}
      onClose={() => setRolePosition(undefined)}
      onSubmit={async (input) => {
        const ok = await onProposeRole(input);
        if (ok) setRolePosition(undefined);
      }}
    />}

    {roleDecision && onApproveRole && onRejectRole && <OrganizationPositionRoleDecision
      state={roleDecision}
      positionByID={positionByID}
      isBusy={isBusy}
      onClose={() => setRoleDecision(undefined)}
      onSubmit={async (rationale) => {
        const ok = roleDecision.action === "approve"
          ? await onApproveRole(roleDecision.revision, rationale)
          : await onRejectRole(roleDecision.revision, rationale);
        if (ok) setRoleDecision(undefined);
      }}
    />}

    {decision && <OrganizationPositionDecision
      state={decision}
      positionByID={positionByID}
      peopleByID={peopleByID}
      isBusy={isBusy}
      onSimulate={onSimulate}
      onClose={() => setDecision(undefined)}
      onSubmit={async (rationale) => {
        const ok = decision.action === "approve"
          ? await onApprove(decision.revision, rationale)
          : await onReject(decision.revision, rationale);
        if (ok) setDecision(undefined);
      }}
    />}
  </div>;
}

function OrganizationPositionEditor({
  state,
  positions,
  scopes,
  people,
  isBusy,
  onClose,
  onSubmit,
}: {
  state: EditorState;
  positions: OrganizationPosition[];
  scopes: OrganizationScope[];
  people: IdentityPerson[];
  isBusy: boolean;
  onClose: () => void;
  onSubmit: (input: ProposeOrganizationPositionInput) => Promise<void>;
}) {
  const position = "position" in state ? state.position : undefined;
  const [code, setCode] = useState("");
  const [title, setTitle] = useState(position?.title ?? "");
  const [functionName, setFunctionName] = useState(position?.function_name ?? "");
  const [scopeID, setScopeID] = useState(position?.organization_scope_id ?? "");
  const [parentID, setParentID] = useState(position?.parent_position_id ?? "");
  const [occupantID, setOccupantID] = useState(position?.occupant_principal_id ?? "");
  const [effectiveFrom, setEffectiveFrom] = useState("");

  const descendants = position ? positionDescendantIDs(positions, position.id) : new Set<string>();
  const parentOptions = positions
    .filter((item) => item.id !== position?.id && !descendants.has(item.id))
    .map((item) => ({ id: item.id, label: item.title, description: item.code }));
  const scopeOptions = scopes.map((scope) => ({
    id: scope.id,
    label: scope.department_path.join(" / "),
    description: humanize(scope.kind),
  }));
  const occupantOptions = people
    .filter((person) => person.status === "ACTIVE" || person.id === occupantID)
    .map((person) => ({ id: person.id, label: person.display_name, description: humanize(person.status) }));

  const label = state.mode === "add"
    ? "Add position"
    : state.mode === "edit"
      ? "Edit " + position!.title
      : "Retire " + position!.title;

  async function submit() {
    if (state.mode === "add") {
      await onSubmit({
        operation: "CREATE",
        code: code.trim(),
        title: title.trim(),
        function_name: functionName.trim() || undefined,
        organization_scope_id: scopeID || undefined,
        parent_position_id: parentID || undefined,
        occupant_principal_id: occupantID || undefined,
        effective_from: effectiveFromISO(effectiveFrom),
      });
      return;
    }
    if (state.mode === "edit") {
      await onSubmit({
        operation: "UPDATE",
        position_id: position!.id,
        title: title.trim(),
        function_name: functionName.trim() || undefined,
        organization_scope_id: scopeID || undefined,
        parent_position_id: parentID || undefined,
        occupant_principal_id: occupantID || undefined,
        expected_version: position!.version,
        effective_from: effectiveFromISO(effectiveFrom),
      });
      return;
    }
    await onSubmit({
      operation: "RETIRE",
      position_id: position!.id,
      expected_version: position!.version,
      effective_from: effectiveFromISO(effectiveFrom),
    });
  }

  const invalid = state.mode === "add" ? !code.trim() || !title.trim() : state.mode === "edit" ? !title.trim() : false;
  return <FocusedSheet label={label} onClose={onClose} isDismissable={!isBusy}>
    <div className="cs-sheet-heading"><span className="eyebrow">Organization</span><h2>{label}</h2></div>
    <div className="identity-scope-editor">
      {state.mode === "retire" ? <div className="identity-scope-current">
        <span>Position</span><strong>{position!.title}</strong><small>Blocked while active positions report to it or governed responsibilities/authority still reference it. Active role bindings end with the position.</small>
      </div> : <>
        {state.mode === "add"
          ? <TextField label="Code" value={code} onChange={setCode} placeholder="RISK_MANAGER" maxLength={80} isRequired/>
          : <div className="identity-scope-current"><span>Code</span><strong>{position!.code}</strong></div>}
        <TextField label="Title" value={title} onChange={setTitle} maxLength={240} isRequired/>
        <TextField label="Function" value={functionName} onChange={setFunctionName} placeholder="Risk" maxLength={240}/>
        <SelectField label="Organization area" value={scopeID || undefined} placeholder="Legal entity" options={scopeOptions} onChange={(value) => setScopeID(value ?? "")}/>
        <SelectField label="Reports to" value={parentID || undefined} placeholder="Top-level position" options={parentOptions} onChange={(value) => setParentID(value ?? "")}/>
        <SelectField label="Current occupant" value={occupantID || undefined} placeholder="Vacant" options={occupantOptions} onChange={(value) => setOccupantID(value ?? "")}/>
      </>}
      <TextField label="Effective from" type="datetime-local" value={effectiveFrom} onChange={setEffectiveFrom} description="Leave blank to apply immediately after approval."/>
      <div className="identity-scope-editor-actions">
        <Button variant="secondary" onPress={onClose} isDisabled={isBusy}>Cancel</Button>
        <Button variant={state.mode === "retire" ? "destructive" : "primary"} onPress={() => void submit()} isDisabled={invalid} isLoading={isBusy}>
          {state.mode === "retire" ? "Propose retirement" : "Propose change"}
        </Button>
      </div>
    </div>
  </FocusedSheet>;
}

function OrganizationPositionRoleEditor({
  position,
  roles,
  isBusy,
  onClose,
  onSubmit,
}: {
  position: OrganizationPosition;
  roles: IdentityRole[];
  isBusy: boolean;
  onClose: () => void;
  onSubmit: (input: ProposeOrganizationPositionRoleInput) => Promise<void>;
}) {
  const assigned = useMemo(() => new Set(position.role_codes), [position.role_codes]);
  return <FocusedSheet label={`Workspace roles · ${position.title}`} onClose={onClose} isDismissable={!isBusy}>
    <div className="cs-sheet-heading"><span className="eyebrow">Organization</span><h2>Workspace roles</h2></div>
    <div className="identity-scope-editor">
      <div className="identity-scope-current">
        <span>Position</span>
        <strong>{position.title}</strong>
        <small>{position.code}</small>
      </div>
      <div className="inline-notice" role="note">
        Only workspace roles can be changed here. Roles used by responsibility, authority, routing, segregation or escalation remain policy-managed.
      </div>
      <div className="identity-scope-changes">
        <ul>{roles.map((role) => {
          const isAssigned = assigned.has(role.code);
          const locked = role.organization_editable !== true;
          return <li key={role.id}>
            <div>
              <b>{role.name}</b>
              <span>{role.code} · {capabilityLabel(role.capabilities)}</span>
              {locked && <small>{lockReasonLabel(role.organization_lock_reasons ?? [])}</small>}
            </div>
            <StatusBadge tone={locked ? "neutral" : isAssigned ? "success" : "neutral"}>
              {locked ? "Policy managed" : isAssigned ? "Assigned" : "Available"}
            </StatusBadge>
            {!locked && <div className="identity-scope-change-actions">
              <Button
                size="compact"
                variant={isAssigned ? "quiet" : "secondary"}
                isDisabled={isBusy}
                onPress={() => void onSubmit({
                  position_id: position.id,
                  role_template_id: role.id,
                  operation: isAssigned ? "REMOVE" : "ADD",
                  expected_position_version: position.version,
                })}
              >{isAssigned ? "Remove" : "Add"}</Button>
            </div>}
          </li>;
        })}</ul>
      </div>
      <div className="identity-scope-editor-actions">
        <Button variant="secondary" onPress={onClose} isDisabled={isBusy}>Close</Button>
      </div>
    </div>
  </FocusedSheet>;
}

function OrganizationPositionRoleDecision({
  state,
  positionByID,
  isBusy,
  onClose,
  onSubmit,
}: {
  state: RoleDecisionState;
  positionByID: Map<string, OrganizationPosition>;
  isBusy: boolean;
  onClose: () => void;
  onSubmit: (rationale: string) => Promise<void>;
}) {
  const [rationale, setRationale] = useState("");
  const position = positionByID.get(state.revision.position_id);
  const action = state.revision.operation === "ADD" ? "Add" : "Remove";
  const title = state.action === "approve" ? "Approve workspace role change" : "Reject workspace role change";
  return <FocusedSheet label={title} onClose={onClose} isDismissable={!isBusy}>
    <div className="cs-sheet-heading"><span className="eyebrow">Organization</span><h2>{title}</h2></div>
    <div className="identity-scope-editor">
      <div className="identity-scope-current">
        <span>Change</span>
        <strong>{action} {state.revision.role_code}</strong>
        <small>{position?.title ?? state.revision.position_id} · {capabilityLabel(state.revision.capabilities)}</small>
      </div>
      <div className="inline-notice" role="note">
        Approval rechecks the position version and confirms this role is still outside material responsibility, authority, routing, segregation and escalation policy.
      </div>
      <TextArea label="Rationale" value={rationale} onChange={setRationale} rows={3} maxLength={1000} isRequired/>
      <div className="identity-scope-editor-actions">
        <Button variant="secondary" onPress={onClose} isDisabled={isBusy}>Cancel</Button>
        <Button variant={state.action === "reject" ? "destructive" : "primary"} onPress={() => void onSubmit(rationale.trim())} isDisabled={!rationale.trim()} isLoading={isBusy}>
          {state.action === "approve" ? "Approve" : "Reject"}
        </Button>
      </div>
    </div>
  </FocusedSheet>;
}

function OrganizationPositionDecision({
  state,
  positionByID,
  peopleByID,
  isBusy,
  onSimulate,
  onClose,
  onSubmit,
}: {
  state: DecisionState;
  positionByID: Map<string, OrganizationPosition>;
  peopleByID: Map<string, string>;
  isBusy: boolean;
  onSimulate?: (revision: OrganizationPositionRevision) => Promise<OrganizationPositionRouteSimulation | null>;
  onClose: () => void;
  onSubmit: (rationale: string) => Promise<void>;
}) {
  const [rationale, setRationale] = useState("");
  const [simulation, setSimulation] = useState<OrganizationPositionRouteSimulation | null>();
  const changedRoutes = simulation?.scenarios.filter((scenario) => scenario.changed) ?? [];
  const title = state.action === "approve" ? "Approve position change" : "Reject position change";
  return <FocusedSheet label={title} onClose={onClose} isDismissable={!isBusy}>
    <div className="cs-sheet-heading"><span className="eyebrow">Organization</span><h2>{title}</h2></div>
    <div className="identity-scope-editor">
      <div className="identity-scope-current">
        <span>Change</span>
        <strong>{positionRevisionLabel(state.revision, positionByID)}</strong>
        <small>{positionImpactLabel(state.revision)}</small>
      </div>
      {positionOccupantChanged(state.revision) && activeWorkCount(state.revision) > 0 && <div className="inline-notice" role="status">
        {activeWorkCount(state.revision)} active work item{activeWorkCount(state.revision) === 1 ? "" : "s"} stay with the current owner or assignee. This position change does not reassign work.
      </div>}
      {state.revision.effective_from && <div className="inline-notice" role="status">Effective {formatDateTime(state.revision.effective_from)} after approval.</div>}
      {state.action === "approve" && onSimulate && <Button variant="secondary" onPress={() => void onSimulate(state.revision).then(setSimulation)} isDisabled={isBusy}>Check route impact</Button>}
      {simulation && <div className="identity-scope-changes" role="status">
        <strong>{simulation.checked} route{simulation.checked === 1 ? "" : "s"} checked</strong>
        <span>{changedRoutes.length} changed{simulation.truncated ? " · result capped at 100" : ""}</span>
        {changedRoutes.length > 0 && <ul>{changedRoutes.map((scenario) => <li key={routeScenarioKey(scenario)}>
          <div>
            <b>{humanize(scenario.responsibility)} · {humanize(scenario.object_type)}</b>
            <span>{routeSnapshotLabel(scenario.current, peopleByID)} → {routeSnapshotLabel(scenario.proposed, peopleByID)}</span>
          </div>
        </li>)}</ul>}
      </div>}
      <TextArea label="Rationale" value={rationale} onChange={setRationale} rows={3} maxLength={1000} isRequired/>
      <div className="identity-scope-editor-actions">
        <Button variant="secondary" onPress={onClose} isDisabled={isBusy}>Cancel</Button>
        <Button variant={state.action === "reject" ? "destructive" : "primary"} onPress={() => void onSubmit(rationale.trim())} isDisabled={!rationale.trim()} isLoading={isBusy}>
          {state.action === "approve" ? "Approve" : "Reject"}
        </Button>
      </div>
    </div>
  </FocusedSheet>;
}

function capabilityLabel(capabilities: string[]) {
  if (!capabilities.length) return "No workspace capabilities";
  return capabilities.map(humanize).join(", ");
}

function lockReasonLabel(reasons: string[]) {
  if (!reasons.length) return "Managed by policy";
  const labels: Record<string, string> = {
    RESPONSIBILITY_ASSIGNMENT: "Used by responsibility routing",
    AUTHORITY_GRANT: "Used by material authority",
    AUTHORITY_ROUTE: "Used by active authority route",
    SEGREGATION_RULE: "Used by segregation control",
    ESCALATION_ROUTE: "Used by escalation route",
  };
  return reasons.map((reason) => labels[reason] ?? humanize(reason)).join(" · ");
}

function positionRevisionLabel(revision: OrganizationPositionRevision, positionByID: Map<string, OrganizationPosition>) {
  const current = positionByID.get(revision.position_id);
  if (revision.operation === "CREATE") return "Add " + (revision.proposed.title || revision.proposed.code);
  if (revision.operation === "RETIRE") return "Retire " + (current?.title ?? revision.base.title ?? revision.position_id);
  return "Edit " + (current?.title ?? revision.base.title ?? revision.position_id);
}

function positionImpactLabel(revision: OrganizationPositionRevision) {
  const impact = revision.impact;
  const work = activeWorkCount(revision);
  return impact.child_positions + " reports · " + impact.responsibility_assignments + " responsibilities · " + impact.authority_grants + " authority grants · " + impact.active_role_bindings + " roles · " + work + " active work";
}

function activeWorkCount(revision: OrganizationPositionRevision) {
  return revision.impact.active_programs_owned + revision.impact.open_matters_owned + revision.impact.open_actions_owned;
}

function positionOccupantChanged(revision: OrganizationPositionRevision) {
  return revision.base.occupant_principal_id !== revision.proposed.occupant_principal_id;
}

function effectiveFromISO(value: string) {
  if (!value) return undefined;
  const parsed = new Date(value);
  return Number.isNaN(parsed.valueOf()) ? undefined : parsed.toISOString();
}

function positionRevisionHistoryLabel(revision: OrganizationPositionRevision) {
  if (revision.status === "SCHEDULED" && revision.effective_from) return "Effective " + formatDateTime(revision.effective_from);
  if (revision.status === "FAILED") return revision.activation_error_code ? humanize(revision.activation_error_code) : "Activation failed";
  const value = revision.applied_at ?? revision.decided_at ?? revision.created_at;
  return formatDecisionDate(value) + (revision.restored_from_revision_id ? " · restored" : "");
}

function positionRevisionStatusTone(status: string): "neutral" | "success" | "warning" | "error" {
  if (status === "APPLIED") return "success";
  if (status === "SCHEDULED") return "warning";
  if (status === "FAILED") return "error";
  return "neutral";
}

function routeScenarioKey(scenario: OrganizationPositionRouteSimulation["scenarios"][number]) {
  return [scenario.object_type, scenario.object_id, scenario.responsibility, scenario.decision_type ?? "", scenario.materiality].join(":");
}

function routeSnapshotLabel(snapshot: OrganizationPositionRouteSimulation["scenarios"][number]["current"], peopleByID: Map<string, string>) {
  if (snapshot.status !== "RESOLVED" || snapshot.candidate_ids.length === 0) return humanize(snapshot.status);
  return snapshot.candidate_ids.map((id) => peopleByID.get(id) ?? shortID(id)).join(", ");
}

function formatDateTime(value: string) {
  const parsed = new Date(value);
  return Number.isNaN(parsed.valueOf()) ? value : parsed.toLocaleString();
}

function shortID(value: string) {
  return value.length > 12 ? value.slice(0, 8) + "…" + value.slice(-4) : value;
}

function formatDecisionDate(value: string) {
  const parsed = new Date(value);
  return Number.isNaN(parsed.valueOf()) ? value : parsed.toLocaleDateString();
}

function positionDescendantIDs(positions: OrganizationPosition[], rootID: string) {
  const result = new Set<string>();
  let changed = true;
  while (changed) {
    changed = false;
    for (const position of positions) {
      if (position.parent_position_id === rootID || (position.parent_position_id && result.has(position.parent_position_id))) {
        if (!result.has(position.id)) {
          result.add(position.id);
          changed = true;
        }
      }
    }
  }
  return result;
}

function scopeLabel(position: OrganizationPosition, scopeByID: Map<string, OrganizationScope>) {
  const scope = position.organization_scope_id ? scopeByID.get(position.organization_scope_id) : undefined;
  return scope ? scope.department_path.join(" / ") : departmentLabel(position);
}

function departmentLabel(position: OrganizationPosition) {
  return position.department_path.length ? position.department_path.join(" / ") : position.function_name || "Legal entity wide";
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
