import { useEffect, useId, useRef, useState } from "react";
import { apiErrorKind } from "../http";
import { loadVendorRelationship, loadVendorRelationships } from "../vendorApi";
import { linkVendorRelationship, loadVendorRelationshipLinks } from "../vendorLinkApi";
import type { VendorRelationshipLink, VendorLinkTargetType } from "../vendorLinkTypes";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import "../vendor-relationship-links.css";

type Props = { targetType: VendorLinkTargetType; targetID: string };
type LinkedVendor = { link: VendorRelationshipLink; relationship: VendorRelationshipAggregate | null };
type LoadState = "loading" | "ready" | "failed";
type PurposeSelection = "" | keyof typeof relationshipPurposes | "OTHER";

const relationshipPurposes = {
  SERVICE_SUPPORT: "Service support",
  EVIDENCE_PROVIDER: "Evidence provider",
  DELIVERY_PARTY: "Delivery party",
  AFFECTED_PARTY: "Affected party",
} as const;

export function VendorRelationshipLinks({ targetType, targetID }: Props) {
  const [links, setLinks] = useState<LinkedVendor[]>([]);
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadMoreError, setLoadMoreError] = useState(false);
  const [linking, setLinking] = useState(false);
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<VendorRelationshipAggregate[]>([]);
  const [searching, setSearching] = useState(false);
  const [searchAttempted, setSearchAttempted] = useState(false);
  const [searchError, setSearchError] = useState(false);
  const [selectedID, setSelectedID] = useState("");
  const [purpose, setPurpose] = useState<PurposeSelection>("");
  const [customPurpose, setCustomPurpose] = useState("");
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveErrorKind, setSaveErrorKind] = useState<ReturnType<typeof apiErrorKind> | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const loadSequence = useRef(0);
  const searchSequence = useRef(0);
  const saveSequence = useRef(0);
  const activeTarget = useRef("");
  const openButton = useRef<HTMLButtonElement | null>(null);
  const searchInput = useRef<HTMLInputElement | null>(null);
  const restoreOpenFocus = useRef(false);
  const purposeID = useId();
  const customPurposeID = useId();
  const customPurposeHelpID = useId();
  const headingID = useId();
  const targetName = targetType === "PROGRAM" ? "Program" : "issue or change";
  activeTarget.current = `${targetType}:${targetID}`;

  useEffect(() => {
    ++saveSequence.current;
    ++searchSequence.current;
    restoreOpenFocus.current = false;
    setLinks([]);
    setNextCursor(null);
    setLoadMoreError(false);
    setLinking(false);
    resetForm();
    void loadLinks();
  }, [targetType, targetID]);

  useEffect(() => {
    if (linking) {
      searchInput.current?.focus();
    } else if (restoreOpenFocus.current) {
      restoreOpenFocus.current = false;
      openButton.current?.focus();
    }
  }, [linking]);

  async function loadLinks(cursor?: string) {
    const sequence = ++loadSequence.current;
    const targetKey = activeTarget.current;
    const append = Boolean(cursor);
    if (append) {
      setLoadingMore(true);
      setLoadMoreError(false);
    } else {
      setLoadState("loading");
    }
    try {
      const page = await loadVendorRelationshipLinks({ target_type: targetType, target_id: targetID, ...(cursor ? { cursor } : {}), limit: 50 });
      const related = await Promise.all(page.items.map(async (link) => ({
        link,
        relationship: await loadVendorRelationship(link.relationship_id).catch(() => null),
      })));
      if (sequence !== loadSequence.current || targetKey !== activeTarget.current) return;
      setLinks((current) => append ? mergeLinks(current, related) : related);
      setNextCursor(page.next_cursor ?? null);
      setLoadState("ready");
    } catch {
      if (sequence !== loadSequence.current || targetKey !== activeTarget.current) return;
      if (append) setLoadMoreError(true);
      else setLoadState("failed");
    } finally {
      if (sequence === loadSequence.current && targetKey === activeTarget.current) setLoadingMore(false);
    }
  }

  function resetForm() {
    setQuery("");
    setResults([]);
    setSearching(false);
    setSearchAttempted(false);
    setSearchError(false);
    ++searchSequence.current;
    setSelectedID("");
    setPurpose("");
    setCustomPurpose("");
    setSaving(false);
    setSaveError(null);
    setSaveErrorKind(null);
    setNotice(null);
  }

  async function search() {
    const search = query.trim();
    if (!search) return;
    const sequence = ++searchSequence.current;
    setSearchAttempted(true);
    setSearchError(false);
    setSearching(true);
    try {
      const page = await loadVendorRelationships({ search, limit: 20 });
      if (sequence !== searchSequence.current) return;
      setResults(page.items);
      if (!page.items.some((item) => item.relationship.id === selectedID)) setSelectedID("");
    } catch {
      if (sequence !== searchSequence.current) return;
      setSearchError(true);
    } finally {
      if (sequence === searchSequence.current) setSearching(false);
    }
  }

  function changeQuery(value: string) {
    ++searchSequence.current;
    setQuery(value);
    setResults([]);
    setSelectedID("");
    setSearchAttempted(false);
    setSearchError(false);
    setSearching(false);
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    const selected = results.find((item) => item.relationship.id === selectedID);
    const purposeInput = relationshipPurposeInput(purpose, customPurpose);
    if (!selected || !purposeInput) return;
    const sequence = ++saveSequence.current;
    const targetKey = activeTarget.current;
    setSaving(true);
    setSaveError(null);
    setSaveErrorKind(null);
    setNotice(null);
    try {
      const link = await linkVendorRelationship(selectedID, {
        target_type: targetType,
        target_id: targetID,
        ...purposeInput,
      });
      if (sequence !== saveSequence.current || targetKey !== activeTarget.current) return;
      setLinks((current) => [...current.filter((item) => item.link.id !== link.id), { link, relationship: selected }]);
      setLinking(false);
      resetForm();
      setNotice(`Vendor linked to this ${targetName}.`);
    } catch (error) {
      if (sequence !== saveSequence.current || targetKey !== activeTarget.current) return;
      const kind = apiErrorKind(error);
      setSaveErrorKind(kind);
      setSaveError(linkErrorMessage(kind));
    } finally {
      if (sequence === saveSequence.current && targetKey === activeTarget.current) setSaving(false);
    }
  }

  return <section className="vendor-relationship-links" aria-labelledby={headingID}>
    <div className="section-heading-row">
      <div><h2 id={headingID}>Related vendors</h2><p>Vendor relationships connected to this {targetName} and the recorded purpose for each link.</p></div>
      {loadState === "ready" && !linking && <button ref={openButton} type="button" className="primary-button" onClick={() => { restoreOpenFocus.current = true; setNotice(null); setLinking(true); }}>Link vendor</button>}
    </div>
    {notice && <p role="status">{notice}</p>}
    {loadState === "loading" && <p aria-live="polite" aria-busy="true">Loading vendor relationships linked to this {targetName}…</p>}
    {loadState === "failed" && <div role="alert"><p>Related vendors could not be loaded for this {targetName}. No link has been changed.</p><button type="button" className="primary-button" onClick={() => void loadLinks()}>Try again</button></div>}
    {loadState === "ready" && links.length === 0 && <p>No vendor relationships are linked to this {targetName}.</p>}
    {loadState === "ready" && links.length > 0 && <ul className="vendor-link-list">{links.map(({ link, relationship }) => <li key={link.id} className={link.state === "ENDED" ? "ended" : undefined}>
      <div className="vendor-link-identity">{relationship ? <><strong>{relationship.vendor.legal_name}</strong><span>{relationship.relationship.service_name}</span></> : <><strong>Vendor details unavailable</strong><span>The linked relationship could not be loaded in the current scope.</span></>}</div>
      <div className="vendor-link-purpose"><span>{link.purpose_label}</span>{link.state === "ENDED" && <small>Link ended</small>}</div>
    </li>)}</ul>}
    {loadState === "ready" && nextCursor && <button type="button" className="secondary-button vendor-link-load-more" disabled={loadingMore} onClick={() => void loadLinks(nextCursor)}>{loadingMore ? "Loading…" : "Load more related vendors"}</button>}
    {loadMoreError && <div role="alert"><span>More related vendors could not be loaded. The current list remains available.</span><button type="button" className="secondary-button" onClick={() => nextCursor && void loadLinks(nextCursor)}>Try again</button></div>}
    {linking && <form className="vendor-link-form" onSubmit={(event) => void submit(event)}>
      <h3>Link a vendor relationship</h3>
      <p>Search the current vendor register, then record why the relationship supports this {targetName}.</p>
      <div className="form-grid"><label>Search vendor relationships<input ref={searchInput} value={query} onChange={(event) => changeQuery(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") { event.preventDefault(); void search(); } }} autoComplete="off"/></label><button type="button" className="secondary-button" disabled={searching || !query.trim()} onClick={() => void search()}>{searching ? "Searching…" : "Search vendors"}</button></div>
      {searchError && <div role="alert"><span>Vendor search is unavailable. Your search remains on this screen.</span><button type="button" className="secondary-button" disabled={searching} onClick={() => void search()}>Try again</button></div>}
      {!searchError && searchAttempted && results.length === 0 && <p>No vendor relationships match this search. Check the vendor or service name and search again.</p>}
      {results.length > 0 && <fieldset><legend>Search results</legend>{results.map((item) => <label key={item.relationship.id}><input type="radio" name="vendor-relationship" value={item.relationship.id} checked={selectedID === item.relationship.id} onChange={() => setSelectedID(item.relationship.id)}/><span><strong>{item.vendor.legal_name}</strong> · {item.relationship.service_name}</span></label>)}</fieldset>}
      <div className="form-grid"><div><label htmlFor={purposeID}>Relationship purpose</label><select id={purposeID} value={purpose} onChange={(event) => setPurpose(event.target.value as PurposeSelection)}><option value="">Choose a purpose</option>{Object.entries(relationshipPurposes).map(([code, label]) => <option key={code} value={code}>{label}</option>)}<option value="OTHER">Other</option></select></div>{purpose === "OTHER" && <div><label htmlFor={customPurposeID}>Custom purpose</label><input id={customPurposeID} aria-describedby={customPurposeHelpID} value={customPurpose} maxLength={160} required onChange={(event) => setCustomPurpose(event.target.value)}/><small id={customPurposeHelpID}>Briefly state how this vendor relationship supports the current record.</small></div>}</div>
      {saveError && <div role="alert"><span>{saveError}</span>{saveErrorKind === "conflict" && <button type="button" className="secondary-button" onClick={() => void loadLinks()}>Refresh related vendors</button>}</div>}
      <div className="form-actions"><button type="button" className="secondary-button" disabled={saving} onClick={() => { setLinking(false); resetForm(); }}>Cancel</button><button type="submit" className="primary-button" disabled={saving || !selectedID || !relationshipPurposeInput(purpose, customPurpose)}>{saving ? "Linking…" : "Link vendor"}</button></div>
    </form>}
  </section>;
}

function relationshipPurposeInput(purpose: PurposeSelection, customPurpose: string) {
  if (purpose === "OTHER") {
    const label = customPurpose.trim();
    return label ? { purpose_code: "OTHER", purpose_label: label } : null;
  }
  if (!purpose) return null;
  return { purpose_code: purpose, purpose_label: relationshipPurposes[purpose] };
}

function mergeLinks(current: LinkedVendor[], next: LinkedVendor[]) {
  const merged = new Map(current.map((item) => [item.link.id, item]));
  for (const item of next) merged.set(item.link.id, item);
  return [...merged.values()];
}

function linkErrorMessage(kind: ReturnType<typeof apiErrorKind>) {
  if (kind === "conflict") return "This vendor link changed. Refresh related vendors before trying again.";
  if (kind === "not_found") return "The selected vendor relationship is no longer available. Search again and choose a current relationship.";
  if (kind === "forbidden" || kind === "unauthorized") return "Your current access does not allow this vendor relationship to be linked. No change was made.";
  if (kind === "validation") return "Check the selected relationship and purpose details, then try again.";
  return "The vendor could not be linked. Your search and purpose details remain on this screen. Try again.";
}
