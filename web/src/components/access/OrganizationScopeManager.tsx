import { useEffect, useId, useMemo, useRef, useState, type CSSProperties, type ReactNode } from "react";
import type {
  OrganizationScope,
  OrganizationScopeRevision,
  ProposeOrganizationScopeInput,
} from "../../identityAccessApi";
import { ActionLink, Button, FocusedSheet, IconButton, Notice, SearchField, SelectableRecord, SelectField, StatusBadge, Surface, TextArea, TextField } from "../ui";
import { NavigationIcon } from "../NavigationIcon";
import { enterpriseSetupHash, enterpriseSetupRoute } from "../configure/enterpriseSetupRouting";
import { peopleAccessHash } from "../configure/peopleAccessRouting";
import { filterOrganizationHierarchy, organizationHierarchy, scopeKindLabel } from "./organizationHierarchy";
import "./organization-setup.css";

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
  scopesTruncated?: boolean;
  legalEntityName?: string;
  legalEntityCode?: string;
  selectedScopeID?: string;
  onSelectScope?: (id: string | undefined) => void;
  children?: ReactNode;
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
  scopesTruncated = false,
  legalEntityName = "Current legal entity",
  legalEntityCode,
  selectedScopeID,
  onSelectScope,
  children,
  onPropose,
  onApprove,
  onReject,
}: Props) {
  const [editor, setEditor] = useState<EditorState>();
  const [decision, setDecision] = useState<DecisionState>();
  const [reviewOpen, setReviewOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [collapsed, setCollapsed] = useState<Set<string>>(() => new Set());
  const [localSelection, setLocalSelection] = useState<string>();
  const detailRef = useRef<HTMLElement>(null);
  const treeRef = useRef<HTMLDivElement>(null);
  const identity = useId();
  const selectedID = onSelectScope ? selectedScopeID : localSelection;
  const pendingByScope = useMemo(() => new Map(revisions.filter((item) => item.status === "PENDING").map((item) => [item.scope_id, item])), [revisions]);
  const scopeByID = useMemo(() => new Map(scopes.map((scope) => [scope.id, scope])), [scopes]);
  const hierarchy = useMemo(() => organizationHierarchy(scopes), [scopes]);
  const visible = useMemo(() => filterOrganizationHierarchy(hierarchy, query, collapsed), [hierarchy, query, collapsed]);
  const selected = selectedID ? scopeByID.get(selectedID) : undefined;
  const selectedRow = selectedID ? hierarchy.find((row) => row.scope.id === selectedID) : undefined;
  const pending = selectedID ? pendingByScope.get(selectedID) : undefined;
  const pendingChanges = revisions.filter((revision) => revision.status === "PENDING");
  const imported = selected?.kind === "ORGANIZATION_UNIT";
  const branches = scopes.filter((scope) => scope.kind === "BRANCH").length;
  const departments = scopes.filter((scope) => scope.kind === "DEPARTMENT").length;
  const issueCount = hierarchy.filter((row) => row.issue).length;
  const selections = enterpriseSetupRoute(window.location.hash)?.selections ?? {};
  const groupsHref = enterpriseSetupHash("groups", selectedID, selections);
  const peopleHref = peopleAccessHash("people", selections);
  const directoryHref = peopleAccessHash("organization", selections, "directory");

  useEffect(() => {
    if (!canConfigure) { setEditor(undefined); setDecision(undefined); }
  }, [canConfigure]);
  useEffect(() => { setEditor(undefined); setDecision(undefined); setReviewOpen(false); }, [selectedID]);

  function recordID(id?: string) { return `${identity}-area-${id ?? "entity"}`; }
  function select(id: string | undefined) {
    setLocalSelection(id);
    onSelectScope?.(id);
    if (window.matchMedia("(max-width: 1300px)").matches) requestAnimationFrame(() => {
      const headerBottom = document.querySelector(".context-bar")?.getBoundingClientRect().bottom ?? 0;
      if (detailRef.current) detailRef.current.style.scrollMarginTop = `${Math.max(0, headerBottom) + 16}px`;
      detailRef.current?.focus({ preventScroll: true });
      detailRef.current?.scrollIntoView({ block: "start", behavior: "instant" });
    });
  }
  function returnToHierarchy() {
    const target = document.getElementById(recordID(selectedID)) ?? treeRef.current;
    target?.focus({ preventScroll: true });
    target?.scrollIntoView({ block: "center", behavior: "instant" });
  }
  function toggle(id: string) {
    setCollapsed((current) => { const next = new Set(current); if (next.has(id)) next.delete(id); else next.add(id); return next; });
  }

  return <div className="organization-setup-layout identity-scope-manager">
    <div className="organization-setup-primary">
      <div className="organization-setup-grid">
        <section className="organization-setup-hierarchy" aria-label="Organization structure">
          <Surface>
            <header className="organization-setup-card-heading"><div><h2>Organization structure</h2><p>{scopes.length} loaded areas · Search covers this list</p></div><div className="organization-setup-tools"><SearchField label="Find organization unit" value={query} onChange={setQuery} placeholder="Find a unit"/>{canConfigure && <Button onPress={() => setEditor({ mode: "add" })} isDisabled={isBusy}>Add area</Button>}</div></header>
            <div className="organization-setup-tree" ref={treeRef} tabIndex={-1}>
              <div className="organization-setup-row organization-setup-row--entity">
                <span className="organization-setup-expander" aria-hidden="true"/>
                <SelectableRecord id={recordID()} variant="list" title={legalEntityName} metadata="Legal entity" aria-label={`Inspect ${legalEntityName}`} isSelected={!selectedID} onPress={() => select(undefined)} startContent={<NavigationIcon view="frameworks"/>}/>
              </div>
              <ul aria-label="Loaded organization areas">
                {visible.rows.map((row) => <li key={row.scope.id} className="organization-setup-row" style={{ "--organization-depth": Math.min(row.depth + 1, 3) } as CSSProperties}>
                  {row.childCount > 0 ? <IconButton variant="quiet" aria-label={`${collapsed.has(row.scope.id) && !query.trim() ? "Expand" : "Collapse"} ${row.scope.name}`} aria-expanded={query.trim() ? true : !collapsed.has(row.scope.id)} isDisabled={Boolean(query.trim())} onPress={() => toggle(row.scope.id)}><span aria-hidden="true">{collapsed.has(row.scope.id) && !query.trim() ? "›" : "⌄"}</span></IconButton> : <span className="organization-setup-expander" aria-hidden="true"/>}
                  <SelectableRecord id={recordID(row.scope.id)} variant="list" title={row.scope.name} metadata={scopeKindLabel(row.scope.kind)} aria-label={`Inspect ${row.scope.name}`} isSelected={selectedID === row.scope.id} onPress={() => select(row.scope.id)} startContent={<NavigationIcon view={row.scope.kind === "BRANCH" ? "programs" : row.scope.kind === "DEPARTMENT" ? "people" : "frameworks"}/>}/>
                </li>)}
              </ul>
              {!visible.rows.length && <div className="organization-setup-empty"><h3>{scopes.length ? "No areas match this search" : "No organization areas returned"}</h3><p>{scopes.length ? "Search checks only the loaded names, codes, types and paths." : "No active areas were returned for this legal entity."}</p>{query && <Button variant="quiet" onPress={() => setQuery("")}>Clear search</Button>}</div>}
            </div>
            <footer className="organization-setup-list-footer">{query.trim() ? `${visible.matches} matching areas · ${visible.rows.length} rows including parent context` : `${visible.rows.length} of ${scopes.length} loaded areas shown`}</footer>
            {scopesTruncated && <Notice tone="warning">First 500 areas loaded. Search filters this list; additional areas are not included.</Notice>}
            {issueCount > 0 && <Notice tone="warning">{issueCount} area relationship{issueCount === 1 ? " needs" : "s need"} review. Some parents are missing or form a loop. Review the area relationships.</Notice>}
          </Surface>
        </section>
        <div className="organization-setup-summaries">
          <Surface><section className="organization-setup-summary"><h2>Area changes</h2><p>{pendingChanges.length} pending proposal{pendingChanges.length === 1 ? "" : "s"} loaded</p><ul className="organization-setup-change-preview">{pendingChanges.slice(0, 3).map((revision) => <li key={revision.id}><strong>{revisionLabel(revision, scopeByID)}</strong><span>{revision.maker_id === actorPrincipalID ? "Awaiting approval" : "Independent review"}</span></li>)}</ul>{!pendingChanges.length && <p>No pending area changes were returned.</p>}<Button variant="quiet" onPress={() => setReviewOpen(true)}>Review area changes</Button></section></Surface>
          <Surface><section className="organization-setup-summary"><h2>Structure</h2><p>Areas in this loaded list</p><dl className="organization-setup-structure"><div><dt>Branches</dt><dd>{branches}</dd></div><div><dt>Departments</dt><dd>{departments}</dd></div><div><dt>Other areas</dt><dd>{scopes.length - branches - departments}</dd></div></dl><ActionLink href={groupsHref}>Organization groups</ActionLink></section></Surface>
        </div>
      </div>
      {children}
    </div>
    <aside className="organization-setup-context" ref={detailRef} tabIndex={-1} aria-labelledby={`${identity}-context`}>
      <Surface><div className="organization-setup-context-body">
        <span className="organization-setup-kicker">Current selection</span>
        <h2 id={`${identity}-context`}>{selected?.name || (selectedID ? "Area not in this list" : legalEntityName)}</h2>
        {selected ? <><dl className="organization-setup-facts"><dt>Type</dt><dd>{scopeKindLabel(selected.kind)}</dd><dt>Code</dt><dd>{selected.code}</dd><dt>Parent</dt><dd>{selectedRow?.issue === "cycle" ? "Cyclic relationship — review required" : selected.parent_scope_id ? scopeByID.get(selected.parent_scope_id)?.name || "Parent outside loaded areas" : legalEntityName}</dd><dt>Path</dt><dd>{selected.department_path.join(" / ")}</dd><dt>Source</dt><dd>{selected.origin === "MANAGED" ? "Managed area" : "Imported department path"}</dd><dt>Version</dt><dd>{selected.version}</dd><dt>Effective from</dt><dd><time dateTime={selected.valid_from}>{selected.valid_from.replace("T", " ").replace(/Z$/, " UTC")}</time></dd>{selected.valid_until && <><dt>Effective until</dt><dd><time dateTime={selected.valid_until}>{selected.valid_until.replace("T", " ").replace(/Z$/, " UTC")}</time></dd></>}</dl><StatusBadge tone={pending ? "warning" : "neutral"}>{pending ? "Pending change" : "Active"}</StatusBadge>
          {canConfigure && <div className="organization-setup-context-actions"><Button isDisabled={Boolean(pending) || isBusy} onPress={() => setEditor({ mode: "edit", scope: selected })}>Edit area</Button><Button isDisabled={Boolean(pending) || isBusy || imported} onPress={() => setEditor({ mode: "move", scope: selected })}>Move area</Button><Button variant="quiet" isDisabled={Boolean(pending) || isBusy || imported} onPress={() => setEditor({ mode: "retire", scope: selected })}>Retire area</Button></div>}
          {imported && <p className="organization-setup-note">Choose a managed area type and have the change approved before moving or retiring this area.</p>}
          {pending && <p className="organization-setup-note">{pending.maker_id === actorPrincipalID ? "Your proposal is awaiting independent approval." : "A proposed change awaits independent review."} The current area remains active.</p>}
        </> : selectedID ? <><p>The selected area was not returned for this legal entity. It may be outside the loaded list or no longer available.</p><Button onPress={() => select(undefined)}>View legal entity</Button></> : <><dl className="organization-setup-facts"><dt>Type</dt><dd>Legal entity</dd>{legalEntityCode && <><dt>Code</dt><dd>{legalEntityCode}</dd></>}<dt>Areas</dt><dd>{scopes.length} loaded</dd></dl><p className="organization-setup-note">Select an area to review its current structure and proposed changes.</p></>}
        <Button variant="quiet" onPress={returnToHierarchy}>Back to organization structure</Button>
      </div></Surface>
      <Surface><section className="organization-setup-summary"><h2>Next actions</h2><p>Review pending area changes or maintain existing positions and access.</p><div className="organization-setup-next-actions"><ActionLink href={peopleHref}>People &amp; access</ActionLink><ActionLink href={directoryHref}>Directory groups &amp; access</ActionLink></div></section></Surface>
    </aside>

    {reviewOpen && <FocusedSheet label="Area changes" onClose={() => setReviewOpen(false)}><div className="cs-sheet-heading"><span className="eyebrow">Organization</span><h2>Area changes</h2><p>Up to 100 pending proposals. Current areas change only after independent approval.</p></div><div className="identity-scope-changes">
      <ul>{pendingChanges.map((revision) => {
        const own = revision.maker_id === actorPrincipalID;
        return <li key={revision.id}>
          <div>
            <b>{revisionLabel(revision, scopeByID)}</b>
            <span>{impactLabel(revision)}</span>
          </div>
          <StatusBadge tone="warning">Pending</StatusBadge>
          {canConfigure && !own && <div className="identity-scope-change-actions">
            <Button isDisabled={isBusy} variant="secondary" onPress={() => setDecision({ action: "approve", revision })}>Approve</Button>
            <Button isDisabled={isBusy} variant="quiet" onPress={() => setDecision({ action: "reject", revision })}>Reject</Button>
          </div>}
          {own && <small>Awaiting approval</small>}
        </li>;
      })}</ul>{!pendingChanges.length && <Notice>No pending area changes were returned.</Notice>}
    </div></FocusedSheet>}

    {editor && canConfigure && <OrganizationScopeEditor
      state={editor}
      scopes={scopes}
      isBusy={isBusy}
      onClose={() => setEditor(undefined)}
      onSubmit={async (input) => {
        if (isBusy || !canConfigure) return;
        const ok = await onPropose(input);
        if (ok) setEditor(undefined);
      }}
    />}

    {decision && canConfigure && <OrganizationScopeDecision
      state={decision}
      scopeByID={scopeByID}
      isBusy={isBusy}
      onClose={() => setDecision(undefined)}
      onSubmit={async (rationale) => {
        if (isBusy || !canConfigure || decision.revision.maker_id === actorPrincipalID) return;
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
  const [kind, setKind] = useState<OrganizationScope["kind"] | undefined>(state.mode === "add" ? "DEPARTMENT" : managedKind(scope?.kind) ? scope!.kind : undefined);
  const [parentID, setParentID] = useState(scope?.parent_scope_id ?? "");
  const excluded = scope ? descendantIDs(scopes, scope.id) : new Set<string>();
  const availableParents = scopes
    .filter((item) => item.id !== scope?.id && !excluded.has(item.id))
    .map((item) => ({ id: item.id, label: item.department_path.join(" / "), description: scopeKindLabel(item.kind) }));
  const unavailableParent = Boolean(parentID && !availableParents.some((item) => item.id === parentID));
  const parentOptions = unavailableParent
    ? [{ id: parentID, label: "Current parent unavailable", description: "Choose an available parent or the legal entity." }, ...availableParents]
    : availableParents;

  const label = state.mode === "add"
    ? "Add area"
    : state.mode === "edit"
      ? "Edit " + scope!.name
      : state.mode === "move"
        ? "Move " + scope!.name
        : "Retire " + scope!.name;

  async function submit() {
    if (isBusy || (state.mode === "move" && unavailableParent) || ((state.mode === "add" || state.mode === "edit") && (!kind || !managedKind(kind)))) return;
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
      ? !name.trim() || !kind
      : state.mode === "move" && unavailableParent;

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
        {!managedKind(scope?.kind) && <Notice>Choose a managed area type. The imported area keeps its current type until this change is approved.</Notice>}
        <TextField label="Name" value={name} onChange={setName} maxLength={240} isRequired/>
        <SelectField label="Type" value={kind} placeholder="Choose type" options={kinds} onChange={(value) => value && setKind(value)} allowsEmpty={false}/>
      </>}
      {state.mode === "move" && <>
        <div className="identity-scope-current"><span>Current</span><strong>{scope!.department_path.join(" / ")}</strong></div>
        {unavailableParent && <Notice>The current parent is outside the available destinations. Choose a loaded area or the legal entity before proposing a move.</Notice>}
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
  const revision = state.revision;
  const current = scopeByID.get(revision.scope_id);
  return <FocusedSheet label={title} onClose={onClose} isDismissable={!isBusy}>
    <div className="cs-sheet-heading"><span className="eyebrow">Organization</span><h2>{title}</h2></div>
    <div className="identity-scope-editor">
      <div className="identity-scope-current">
        <span>Change</span>
        <strong>{revisionLabel(state.revision, scopeByID)}</strong>
        <small>{impactLabel(state.revision)}</small>
      </div>
      <dl className="organization-setup-facts">
        <dt>Base version</dt><dd>{revision.base_version}</dd>
        {revision.proposed_code && <><dt>Code</dt><dd>{revision.proposed_code}</dd></>}
        {current && <><dt>Current name</dt><dd>{current.name}</dd><dt>Current type</dt><dd>{scopeKindLabel(current.kind)}</dd></>}
        {revision.proposed_name && <><dt>Proposed name</dt><dd>{revision.proposed_name}</dd></>}
        {revision.proposed_kind && <><dt>Proposed type</dt><dd>{scopeKindLabel(revision.proposed_kind)}</dd></>}
        {(revision.operation === "CREATE" || revision.operation === "MOVE") && <><dt>Proposed parent</dt><dd>{revision.proposed_parent_scope_id ? scopeByID.get(revision.proposed_parent_scope_id)?.name || `${revision.proposed_parent_scope_id} — outside loaded areas` : "Legal entity"}</dd></>}
        <dt>Proposed by</dt><dd>{revision.maker_id}</dd><dt>Proposed at</dt><dd><time dateTime={revision.created_at}>{revision.created_at.replace("T", " ").replace(/Z$/, " UTC")}</time></dd>
      </dl>
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
