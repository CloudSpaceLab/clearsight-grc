import { useMemo, useState } from "react";
import type {
  IdentityPerson,
  OrganizationPosition,
  OrganizationPositionRevision,
  OrganizationScope,
  ProposeOrganizationPositionInput,
} from "../../identityAccessApi";
import { Button, FocusedSheet, SelectField, StatusBadge, TextArea, TextField } from "../ui";

type EditorState =
  | { mode: "add" }
  | { mode: "edit" | "retire"; position: OrganizationPosition };

type DecisionState = {
  action: "approve" | "reject";
  revision: OrganizationPositionRevision;
};

type Props = {
  positions: OrganizationPosition[];
  allPositions: OrganizationPosition[];
  scopes: OrganizationScope[];
  people: IdentityPerson[];
  revisions: OrganizationPositionRevision[];
  actorPrincipalID: string;
  canConfigure: boolean;
  isBusy?: boolean;
  scopeByID: Map<string, OrganizationScope>;
  onPropose: (input: ProposeOrganizationPositionInput) => Promise<boolean>;
  onApprove: (revision: OrganizationPositionRevision, rationale: string) => Promise<boolean>;
  onReject: (revision: OrganizationPositionRevision, rationale: string) => Promise<boolean>;
};

export function OrganizationPositionManager({
  positions,
  allPositions,
  scopes,
  people,
  revisions,
  actorPrincipalID,
  canConfigure,
  isBusy = false,
  scopeByID,
  onPropose,
  onApprove,
  onReject,
}: Props) {
  const [editor, setEditor] = useState<EditorState>();
  const [decision, setDecision] = useState<DecisionState>();
  const pendingByPosition = useMemo(
    () => new Map(revisions.filter((item) => item.status === "PENDING").map((item) => [item.position_id, item])),
    [revisions],
  );
  const positionByID = useMemo(() => new Map(allPositions.map((position) => [position.id, position])), [allPositions]);

  return <div className="identity-position-manager">
    <div className="identity-scope-manager__header">
      <div><h3>Positions & roles</h3><span>{allPositions.length} active</span></div>
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

    {positions.length > 0 && <div className="identity-position-table-wrap">
      <table className="identity-position-table">
        <thead><tr><th>Position</th><th>Current occupant</th><th>Workspace roles</th><th>Reports to</th>{canConfigure && <th>Actions</th>}</tr></thead>
        <tbody>{positions.map((position) => {
          const pending = pendingByPosition.get(position.id);
          return <tr key={position.id}>
            <td data-label="Position"><strong>{position.title}</strong><span>{position.code} · {scopeLabel(position, scopeByID)}</span>{pending && <StatusBadge tone="warning">Pending change</StatusBadge>}</td>
            <td data-label="Current occupant">{position.occupant_name
              ? <><strong>{position.occupant_name}</strong><span>{humanize(position.occupant_status || "active")}</span></>
              : <span className="identity-vacancy">Vacant — coverage required</span>}</td>
            <td data-label="Workspace roles"><div className="identity-role-chips">{position.role_codes.length
              ? position.role_codes.map((role) => <span key={role}>{role}</span>)
              : <span className="identity-empty-value">No role assigned</span>}</div></td>
            <td data-label="Reports to"><strong>{position.parent_position_title || "Top-level position"}</strong>{position.parent_position_code && <span>{position.parent_position_code}</span>}</td>
            {canConfigure && <td data-label="Actions"><div className="identity-scope-row-actions">
              <Button size="compact" variant="quiet" isDisabled={Boolean(pending) || isBusy} onPress={() => setEditor({ mode: "edit", position })}>Edit</Button>
              <Button size="compact" variant="quiet" isDisabled={Boolean(pending) || isBusy} onPress={() => setEditor({ mode: "retire", position })}>Retire</Button>
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

    {decision && <OrganizationPositionDecision
      state={decision}
      positionByID={positionByID}
      isBusy={isBusy}
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
      });
      return;
    }
    await onSubmit({
      operation: "RETIRE",
      position_id: position!.id,
      expected_version: position!.version,
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
      <div className="identity-scope-editor-actions">
        <Button variant="secondary" onPress={onClose} isDisabled={isBusy}>Cancel</Button>
        <Button variant={state.mode === "retire" ? "destructive" : "primary"} onPress={() => void submit()} isDisabled={invalid} isLoading={isBusy}>
          {state.mode === "retire" ? "Propose retirement" : "Propose change"}
        </Button>
      </div>
    </div>
  </FocusedSheet>;
}

function OrganizationPositionDecision({
  state,
  positionByID,
  isBusy,
  onClose,
  onSubmit,
}: {
  state: DecisionState;
  positionByID: Map<string, OrganizationPosition>;
  isBusy: boolean;
  onClose: () => void;
  onSubmit: (rationale: string) => Promise<void>;
}) {
  const [rationale, setRationale] = useState("");
  const title = state.action === "approve" ? "Approve position change" : "Reject position change";
  return <FocusedSheet label={title} onClose={onClose} isDismissable={!isBusy}>
    <div className="cs-sheet-heading"><span className="eyebrow">Organization</span><h2>{title}</h2></div>
    <div className="identity-scope-editor">
      <div className="identity-scope-current">
        <span>Change</span>
        <strong>{positionRevisionLabel(state.revision, positionByID)}</strong>
        <small>{positionImpactLabel(state.revision)}</small>
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

function positionRevisionLabel(revision: OrganizationPositionRevision, positionByID: Map<string, OrganizationPosition>) {
  const current = positionByID.get(revision.position_id);
  if (revision.operation === "CREATE") return "Add " + (revision.proposed.title || revision.proposed.code);
  if (revision.operation === "RETIRE") return "Retire " + (current?.title ?? revision.base.title ?? revision.position_id);
  return "Edit " + (current?.title ?? revision.base.title ?? revision.position_id);
}

function positionImpactLabel(revision: OrganizationPositionRevision) {
  const impact = revision.impact;
  return impact.child_positions + " child positions · " + impact.responsibility_assignments + " responsibilities · " + impact.authority_grants + " authority grants · " + impact.active_role_bindings + " role bindings";
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
