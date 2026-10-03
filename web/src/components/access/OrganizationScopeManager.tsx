import { useMemo, useState } from "react";
import type {
  OrganizationScope,
  OrganizationScopeRevision,
  ProposeOrganizationScopeInput,
} from "../../identityAccessApi";
import { Button, FocusedSheet, SelectField, StatusBadge, TextArea, TextField } from "../ui";

type EditorState =
  | { mode: "add" }
  | { mode: "edit" | "move" | "retire"; scope: OrganizationScope };

type DecisionState = {
  action: "approve" | "reject";
  revision: OrganizationScopeRevision;
};

type Props = {
  scopes: OrganizationScope[];
  revisions: OrganizationScopeRevision[];
  actorPrincipalID: string;
  canConfigure: boolean;
  isBusy?: boolean;
  onPropose: (input: ProposeOrganizationScopeInput) => Promise<boolean>;
  onApprove: (revision: OrganizationScopeRevision, rationale: string) => Promise<boolean>;
  onReject: (revision: OrganizationScopeRevision, rationale: string) => Promise<boolean>;
};

const kinds: Array<{ id: OrganizationScope["kind"]; label: string }> = [
  { id: "BRANCH", label: "Branch" },
  { id: "DEPARTMENT", label: "Department" },
  { id: "FUNCTION", label: "Function" },
  { id: "BUSINESS_UNIT", label: "Business unit" },
  { id: "CRITICAL_SERVICE", label: "Critical service" },
];

export function OrganizationScopeManager({
  scopes,
  revisions,
  actorPrincipalID,
  canConfigure,
  isBusy = false,
  onPropose,
  onApprove,
  onReject,
}: Props) {
  const [editor, setEditor] = useState<EditorState>();
  const [decision, setDecision] = useState<DecisionState>();
  const pendingByScope = useMemo(() => new Map(revisions.filter((item) => item.status === "PENDING").map((item) => [item.scope_id, item])), [revisions]);
  const scopeByID = useMemo(() => new Map(scopes.map((scope) => [scope.id, scope])), [scopes]);
  const ordered = useMemo(() => [...scopes].sort((left, right) => left.department_path.join("/").localeCompare(right.department_path.join("/"))), [scopes]);

  return <div className="identity-scope-manager">
    <div className="identity-scope-manager__header">
      <div><h3>Areas</h3><span>{scopes.length} active</span></div>
      {canConfigure && <Button size="compact" onPress={() => setEditor({ mode: "add" })}>Add area</Button>}
    </div>

    {revisions.length > 0 && <div className="identity-scope-changes">
      <strong>Pending changes</strong>
      <ul>{revisions.map((revision) => {
        const own = revision.maker_id === actorPrincipalID;
        return <li key={revision.id}>
          <div>
            <b>{revisionLabel(revision, scopeByID)}</b>
            <span>{impactLabel(revision)}</span>
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

    <div className="identity-scope-table-wrap">
      <table className="identity-scope-table">
        <thead><tr><th>Area</th><th>Type</th><th>State</th>{canConfigure && <th>Actions</th>}</tr></thead>
        <tbody>{ordered.map((scope) => {
          const pending = pendingByScope.get(scope.id);
          return <tr key={scope.id}>
            <td data-label="Area"><strong>{scope.name}</strong><span>{scope.department_path.join(" / ")}</span></td>
            <td data-label="Type">{scopeKindLabel(scope.kind)}</td>
            <td data-label="State">{pending ? <StatusBadge tone="warning">Pending change</StatusBadge> : <StatusBadge tone="neutral">Active</StatusBadge>}</td>
            {canConfigure && <td data-label="Actions">
              <div className="identity-scope-row-actions">
                <Button size="compact" variant="quiet" isDisabled={Boolean(pending) || isBusy} onPress={() => setEditor({ mode: "edit", scope })}>Edit</Button>
                <Button size="compact" variant="quiet" isDisabled={Boolean(pending) || isBusy} onPress={() => setEditor({ mode: "move", scope })}>Move</Button>
                <Button size="compact" variant="quiet" isDisabled={Boolean(pending) || isBusy} onPress={() => setEditor({ mode: "retire", scope })}>Retire</Button>
              </div>
            </td>}
          </tr>;
        })}</tbody>
      </table>
    </div>

    {!ordered.length && <div className="identity-empty-state"><strong>No areas</strong><span>Add a branch, department, function, business unit or critical service.</span></div>}

    {editor && <OrganizationScopeEditor
      state={editor}
      scopes={scopes}
      isBusy={isBusy}
      onClose={() => setEditor(undefined)}
      onSubmit={async (input) => {
        const ok = await onPropose(input);
        if (ok) setEditor(undefined);
      }}
    />}

    {decision && <OrganizationScopeDecision
      state={decision}
      scopeByID={scopeByID}
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

function OrganizationScopeEditor({
  state,
  scopes,
  isBusy,
  onClose,
  onSubmit,
}: {
  state: EditorState;
  scopes: OrganizationScope[];
  isBusy: boolean;
  onClose: () => void;
  onSubmit: (input: ProposeOrganizationScopeInput) => Promise<void>;
}) {
  const scope = "scope" in state ? state.scope : undefined;
  const [name, setName] = useState(scope?.name ?? "");
  const [code, setCode] = useState("");
  const [kind, setKind] = useState<OrganizationScope["kind"]>(managedKind(scope?.kind) ? scope!.kind : "DEPARTMENT");
  const [parentID, setParentID] = useState(scope?.parent_scope_id ?? "");
  const excluded = scope ? descendantIDs(scopes, scope.id) : new Set<string>();
  const parentOptions = scopes
    .filter((item) => item.id !== scope?.id && !excluded.has(item.id))
    .map((item) => ({ id: item.id, label: item.department_path.join(" / "), description: scopeKindLabel(item.kind) }));

  const label = state.mode === "add"
    ? "Add area"
    : state.mode === "edit"
      ? "Edit " + scope!.name
      : state.mode === "move"
        ? "Move " + scope!.name
        : "Retire " + scope!.name;

  async function submit() {
    if (state.mode === "add") {
      await onSubmit({
        operation: "CREATE",
        parent_scope_id: parentID || undefined,
        code: code.trim(),
        name: name.trim(),
        kind,
      });
      return;
    }
    if (state.mode === "edit") {
      await onSubmit({
        operation: "UPDATE",
        scope_id: scope!.id,
        name: name.trim(),
        kind,
        expected_version: scope!.version,
      });
      return;
    }
    if (state.mode === "move") {
      await onSubmit({
        operation: "MOVE",
        scope_id: scope!.id,
        parent_scope_id: parentID || undefined,
        expected_version: scope!.version,
      });
      return;
    }
    await onSubmit({
      operation: "RETIRE",
      scope_id: scope!.id,
      expected_version: scope!.version,
    });
  }

  const invalid = state.mode === "add"
    ? !code.trim() || !name.trim()
    : state.mode === "edit"
      ? !name.trim()
      : false;

  return <FocusedSheet label={label} onClose={onClose} isDismissable={!isBusy}>
    <div className="cs-sheet-heading"><span className="eyebrow">Organization</span><h2>{label}</h2></div>
    <div className="identity-scope-editor">
      {state.mode === "add" && <>
        <TextField label="Code" value={code} onChange={setCode} placeholder="LAGOS_ISLAND" maxLength={80} isRequired/>
        <TextField label="Name" value={name} onChange={setName} placeholder="Lagos Island" maxLength={240} isRequired/>
        <SelectField label="Type" value={kind} placeholder="Choose type" options={kinds} onChange={(value) => value && setKind(value)} allowsEmpty={false}/>
        <SelectField label="Parent" value={parentID || undefined} placeholder="Legal entity" options={parentOptions} onChange={(value) => setParentID(value ?? "")}/>
      </>}
      {state.mode === "edit" && <>
        <div className="identity-scope-current"><span>Area</span><strong>{scope!.department_path.join(" / ")}</strong></div>
        <TextField label="Name" value={name} onChange={setName} maxLength={240} isRequired/>
        <SelectField label="Type" value={kind} placeholder="Choose type" options={kinds} onChange={(value) => value && setKind(value)} allowsEmpty={false}/>
      </>}
      {state.mode === "move" && <>
        <div className="identity-scope-current"><span>Current</span><strong>{scope!.department_path.join(" / ")}</strong></div>
        <SelectField label="New parent" value={parentID || undefined} placeholder="Legal entity" options={parentOptions} onChange={(value) => setParentID(value ?? "")}/>
      </>}
      {state.mode === "retire" && <div className="identity-scope-current">
        <span>Area</span><strong>{scope!.department_path.join(" / ")}</strong>
        <small>Blocked if the area has child areas, positions, access mappings or open matters.</small>
      </div>}
      <div className="identity-scope-editor-actions">
        <Button variant="secondary" onPress={onClose} isDisabled={isBusy}>Cancel</Button>
        <Button variant={state.mode === "retire" ? "destructive" : "primary"} onPress={() => void submit()} isDisabled={invalid} isLoading={isBusy}>
          {state.mode === "retire" ? "Propose retirement" : "Propose change"}
        </Button>
      </div>
    </div>
  </FocusedSheet>;
}

function OrganizationScopeDecision({
  state,
  scopeByID,
  isBusy,
  onClose,
  onSubmit,
}: {
  state: DecisionState;
  scopeByID: Map<string, OrganizationScope>;
  isBusy: boolean;
  onClose: () => void;
  onSubmit: (rationale: string) => Promise<void>;
}) {
  const [rationale, setRationale] = useState("");
  const title = state.action === "approve" ? "Approve change" : "Reject change";
  return <FocusedSheet label={title} onClose={onClose} isDismissable={!isBusy}>
    <div className="cs-sheet-heading"><span className="eyebrow">Organization</span><h2>{title}</h2></div>
    <div className="identity-scope-editor">
      <div className="identity-scope-current">
        <span>Change</span>
        <strong>{revisionLabel(state.revision, scopeByID)}</strong>
        <small>{impactLabel(state.revision)}</small>
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

function revisionLabel(revision: OrganizationScopeRevision, scopeByID: Map<string, OrganizationScope>) {
  const current = scopeByID.get(revision.scope_id);
  switch (revision.operation) {
    case "CREATE": return "Add " + (revision.proposed_name || revision.proposed_code);
    case "UPDATE": return "Edit " + (current?.name ?? revision.scope_id);
    case "MOVE": return "Move " + (current?.name ?? revision.scope_id);
    case "RETIRE": return "Retire " + (current?.name ?? revision.scope_id);
  }
}

function impactLabel(revision: OrganizationScopeRevision) {
  const impact = revision.impact;
  if (!impact) return "No impact count";
  return impact.child_scopes + " child areas · " + impact.positions + " positions · " + impact.access_mappings + " access mappings · " + impact.open_matters + " open matters";
}

function scopeKindLabel(value: OrganizationScope["kind"]) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\\s)\\S/g, (letter) => letter.toUpperCase());
}

function managedKind(value: OrganizationScope["kind"] | undefined): value is OrganizationScope["kind"] {
  return kinds.some((item) => item.id === value);
}

function descendantIDs(scopes: OrganizationScope[], rootID: string) {
  const result = new Set<string>();
  let changed = true;
  while (changed) {
    changed = false;
    for (const scope of scopes) {
      if (scope.parent_scope_id === rootID || (scope.parent_scope_id && result.has(scope.parent_scope_id))) {
        if (!result.has(scope.id)) {
          result.add(scope.id);
          changed = true;
        }
      }
    }
  }
  return result;
}
