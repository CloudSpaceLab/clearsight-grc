import { useEffect, useState } from "react";
import { commitRegisterMigration, loadRegisterMigration, saveRegisterMigration, suggestPerson, suggestRelationship } from "../../registerMigrationApi";
import type { RegisterDraft, RegisterGroup, RegisterSelection, RegisterView } from "../../registerMigrationApi";
import { loadVendorRelationship, loadVendorRelationships } from "../../vendorApi";
import type { VendorRelationshipAggregate } from "../../vendorTypes";
import { documentResultLink } from "../../documentResultRouting";
import "./risk-register.css";

export function RiskRegisterMigration({ documentID }: { documentID: string }) {
  const [view, setView] = useState<RegisterView | null>(null);
  const [selection, setSelection] = useState<RegisterSelection | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const [confirmed, setConfirmed] = useState(false);
  const [pending, setPending] = useState<RegisterDraft | null>(null);
  const [reload, setReload] = useState(0);
  const [dirty, setDirty] = useState(false);
  useEffect(() => {
    let active = true;
    setView(null); setSelection(null); setError(""); setPending(null); setConfirmed(false); setDirty(false);
    loadRegisterMigration(documentID).then((result) => {
      if (!active) return;
      const people = result.owners.filter((person) => result.performers.some((p) => p.id === person.id));
      setView(result);
      setSelection({ ...result.draft.selection, owners: result.draft.selection.owners.map((owner) => ({ ...owner, person_id: owner.person_id || (result.draft.version === 0 ? suggestPerson(owner.name, people)?.id ?? "" : "") })) });
    }).catch((cause) => { if (active) setError(message(cause)); });
    return () => { active = false; };
  }, [documentID, reload]);
  useEffect(() => {
    if (!dirty) return;
    const warn = (event: BeforeUnloadEvent) => { event.preventDefault(); };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);
  function change(update: (current: RegisterSelection) => RegisterSelection) {
    if (busy || pending) return;
    setSelection((current) => current && update(current)); setConfirmed(false); setNotice(""); setDirty(true);
  }
  async function submit(importing: boolean) {
    if (!view || !selection || busy) return;
    setBusy(true); setError(""); setNotice("");
    try {
      let draft = pending;
      if (!draft) {
        const saved = await saveRegisterMigration(view.draft, selection);
        setView(saved); setSelection(saved.draft.selection); setDirty(false);
        draft = saved.draft;
      }
      if (importing) {
        setPending(draft);
        const receipt = await commitRegisterMigration(draft);
        setView((current) => current && ({ ...current, draft: receipt })); setPending(null);
      } else setNotice("Migration choices saved.");
    } catch (cause) { setError(message(cause)); }
    finally { setBusy(false); }
  }
  if (!view || !selection) return <section className="risk-register" aria-label="Risk register migration">
    <h3>Import vendor findings</h3>
    {error ? <><p role="alert">{error}</p><button type="button" className="secondary-button" onClick={() => setReload((v) => v + 1)}>Retry register review</button></> : <p role="status">Checking register rows and assignment authority…</p>}
  </section>;
  if (view.draft.status === "IMPORTED") return <section className="risk-register" aria-label="Risk register migration">
    <h3>{view.draft.receipts.length} finding{view.draft.receipts.length === 1 ? "" : "s"} imported</h3>
    <p>Current status and remediation outcomes need review. No vendor invitations were sent.</p>
    <ul>{view.draft.receipts.map((item) => <li key={item.row_id}><a href={documentResultLink("MATTER", item.matter_id)?.href}>{item.title}</a></li>)}</ul>
  </section>;
  const people = view.owners.filter((person) => view.performers.some((p) => p.id === person.id));
  const included = selection.rows.filter((r) => r.include);
  const missingOwners = selection.owners.filter((o) => included.some((r) => r.owner_name === o.name) && !people.some((p) => p.id === o.person_id)).length;
  const missingGroups = selection.groups.filter((g) => !g.relationship_id && included.some((choice) => view.register.rows.some((row) => row.id === choice.row_id && row.group_id === g.group_id))).length;
  const ready = included.length > 0 && !missingOwners && !missingGroups && confirmed && view.can_import;
  return <section className="risk-register" aria-label="Risk register migration" aria-busy={busy}>
    <header><div><span className="eyebrow">Risk register</span><h3>Import vendor findings</h3></div><span>{plural(view.register.rows.length, "finding")} · {plural(view.register.groups.length, "assessment")}</span></header>
    <p>Each selected finding becomes an issue with a remediation action. Source ratings, comments and closure claims require current review.</p>
    {!view.can_import && <p role="alert">{view.reason}</p>}
    <fieldset disabled={busy || !!pending || !view.can_import}>
      <legend>Confirm vendor relationships</legend>
      {view.register.groups.map((group) => <VendorMatch key={group.id} group={group} value={selection.groups.find((g) => g.group_id === group.id)!} suggest={view.draft.version === 0} onChange={(item) => change((current) => ({ ...current, groups: current.groups.map((g) => g.group_id === group.id ? { ...g, relationship_id: item?.relationship.id ?? "", relationship_version: item?.relationship.version ?? 0 } : g) }))} />)}
    </fieldset>
    <fieldset disabled={busy || !!pending || !view.can_import}>
      <legend>Confirm responsibility</legend>
      {selection.owners.map((owner) => <div className="register-owner" key={owner.name}>
        <div><strong>{owner.name}</strong><span>{plural(selection.rows.filter((r) => r.include && r.owner_name === owner.name).length, "selected finding")}</span></div>
        <label>Bank owner for {owner.name}<select value={owner.person_id} onChange={(event) => change((current) => ({ ...current, owners: current.owners.map((o) => o.name === owner.name ? { ...o, person_id: event.target.value } : o) }))}><option value="">Choose an eligible person</option>{people.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></label>
        <label className="register-check"><input type="checkbox" aria-label={`Vendor performs remediation for source name ${owner.name}`} checked={owner.vendor_responsible} onChange={(event) => change((current) => ({ ...current, owners: current.owners.map((o) => o.name === owner.name ? { ...o, vendor_responsible: event.target.checked } : o) }))} />Vendor performs remediation</label>
        {owner.vendor_responsible && <p>The bank owner coordinates the linked vendor’s response and evidence review. No invitation will be sent.</p>}
      </div>)}
      {!people.length && <p>No person is currently eligible for both issue ownership and action follow-up. Ask a GRC administrator to review assignment routes.</p>}
    </fieldset>
    <fieldset disabled={busy || !!pending || !view.can_import}>
      <legend>Review findings and actions</legend>
      {view.register.rows.map((row) => {
        const choice = selection.rows.find((r) => r.row_id === row.id)!;
        const group = view.register.groups.find((g) => g.id === row.group_id)!;
        const edit = (patch: Partial<typeof choice>) => change((current) => ({ ...current, rows: current.rows.map((r) => r.row_id === row.id ? { ...r, ...patch } : r) }));
        return <article className="register-finding" key={row.id}>
          <label className="register-check"><input type="checkbox" checked={choice.include} onChange={(event) => edit({ include: event.target.checked })} />{row.finding}</label>
          <p>{group.vendor} · {group.service} · {row.anchor.sheet}, row {row.anchor.row_start}</p>
          {row.inherited.length > 0 && <p>Carried from assessment: {row.inherited.join(", ")}.</p>}
          <p><strong>Action:</strong> {row.recommendation || "Review finding and agree remediation"}</p>
          <div className="register-row-fields">
            <label>Responsibility for row {row.anchor.row_start}<select disabled={!choice.include} value={choice.owner_name} onChange={(event) => edit({ owner_name: event.target.value })}>{selection.owners.map((o) => <option key={o.name} value={o.name}>{o.name}</option>)}</select></label>
            <label>Due date for row {row.anchor.row_start}<input disabled={!choice.include} type="date" value={choice.due_date} onChange={(event) => edit({ due_date: event.target.value })} /></label>
          </div>
          {!choice.due_date && <p>No due date agreed. The action will need a deadline.</p>}
          <details><summary>Review source row · {row.recorded_status ? `Recorded ${row.recorded_status.toLowerCase()}` : "No recorded status"}</summary><dl>{Object.entries(row.original).map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{value || "—"}</dd></div>)}</dl></details>
        </article>;
      })}
      <label className="register-check"><input type="checkbox" checked={confirmed} onChange={(event) => setConfirmed(event.target.checked)} />I confirm the vendor, owner and finding assignments.</label>
    </fieldset>
    <div aria-live="polite">
      {dirty && !busy && <p>Unsaved migration choices.</p>}
      {!!missingGroups && <p>{missingGroups} vendor relationship{missingGroups === 1 ? " needs" : "s need"} a match.</p>}
      {!!missingOwners && <p>{missingOwners} responsibility name{missingOwners === 1 ? " needs" : "s need"} a bank owner.</p>}
      {!included.length && <p>Select at least one finding to import.</p>}
      {!confirmed && !missingGroups && !missingOwners && <p>Confirm the assignments before importing findings.</p>}
      {notice && <p role="status">{notice}</p>}
    </div>
    {error && <p role="alert">{error}</p>}
    <footer>
      <button type="button" className="primary-button" disabled={busy || (!pending && !ready)} onClick={() => void submit(true)}>{busy ? "Saving migration…" : pending ? "Retry import" : `Import ${plural(included.length, "finding")}`}</button>
      {!pending && <button type="button" className="secondary-button" disabled={busy || !view.can_import} onClick={() => void submit(false)}>Save migration choices</button>}
      {error && <button type="button" className="secondary-button" disabled={busy} onClick={() => setReload((v) => v + 1)}>Reload saved migration</button>}
    </footer>
  </section>;
}

function VendorMatch({ group, value, suggest, onChange }: { group: RegisterGroup; value: RegisterSelection["groups"][number]; suggest: boolean; onChange: (item?: VendorRelationshipAggregate) => void }) {
  const [query, setQuery] = useState(group.vendor);
  const [items, setItems] = useState<VendorRelationshipAggregate[]>([]);
  const [chosen, setChosen] = useState<VendorRelationshipAggregate | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [more, setMore] = useState(false);
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    let active = true;
    const timer = window.setTimeout(() => {
      setLoading(true); setError("");
      loadVendorRelationships({ search: query, limit: 25 }).then((page) => {
        if (!active) return;
        setItems(page.items.filter((item) => item.vendor.status === "ACTIVE" && item.relationship.status !== "TERMINATED")); setMore(!!page.next_cursor);
        if (suggest && !value.relationship_id && query === group.vendor) {
          const match = suggestRelationship(group.vendor, group.service, page);
          if (match) { setChosen(match); onChange(match); }
        }
      }).catch((cause) => { if (active) setError(message(cause)); }).finally(() => { if (active) setLoading(false); });
    }, 250);
    return () => { active = false; window.clearTimeout(timer); };
    // Match only against this completed search; later manual choices remain unchanged.
  }, [query, group.id, retry]);
  useEffect(() => {
    if (!value.relationship_id || chosen?.relationship.id === value.relationship_id) return;
    let active = true;
    loadVendorRelationship(value.relationship_id).then((item) => { if (active) setChosen(item); }).catch(() => { if (active) setError("The saved vendor relationship could not be checked. Search and select it again."); });
    return () => { active = false; };
  }, [value.relationship_id]);
  const options = chosen && !items.some((item) => item.relationship.id === chosen.relationship.id) ? [chosen, ...items] : items;
  return <div className="register-vendor">
    <div><strong>{group.vendor}</strong><p>{group.service}{group.assessment_date && ` · Assessed ${group.assessment_date}`}</p></div>
    <label>Search vendors for {group.service}<input type="search" value={query} onChange={(event) => setQuery(event.target.value)} /></label>
    <label>Vendor relationship for {group.service}<select value={value.relationship_id} onChange={(event) => { const item = options.find((item) => item.relationship.id === event.target.value); setChosen(item ?? null); onChange(item); }}><option value="">Choose a vendor and service</option>{options.map((item) => <option key={item.relationship.id} value={item.relationship.id}>{item.vendor.legal_name} · {item.relationship.service_name}</option>)}</select></label>
    {loading && <p role="status">Checking existing vendors…</p>}
    {!loading && !items.length && !error && <p>No active vendor relationships match “{query}”. Search another vendor name.</p>}
    {more && <p>More matches exist. Refine the vendor name before choosing a relationship.</p>}
    {error && <><p role="alert">{error}</p><button type="button" className="secondary-button" onClick={() => setRetry((v) => v + 1)}>Retry vendor search</button></>}
  </div>;
}
function message(cause: unknown) { return cause instanceof Error ? cause.message : "The migration could not be completed. Reload the saved migration and try again."; }
function plural(count: number, noun: string) { return `${count} ${noun}${count === 1 ? "" : "s"}`; }
