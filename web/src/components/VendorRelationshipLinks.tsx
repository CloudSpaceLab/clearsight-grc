import { useCallback, useEffect, useId, useRef, useState } from "react";
import { apiErrorKind } from "../http";
import { loadVendorRelationship, loadVendorRelationships } from "../vendorApi";
import { endVendorRelationshipLink, linkVendorRelationship, loadVendorRelationshipLinks } from "../vendorLinkApi";
import type { VendorRelationshipLink, VendorLinkTargetType } from "../vendorLinkTypes";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import { FocusedSheet } from "./FocusedSheet";
import { VendorBrandIcon } from "./VendorBrandIcon";
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
  const [endingLinkID, setEndingLinkID] = useState("");
  const [endReason, setEndReason] = useState("");
  const [ending, setEnding] = useState(false);
  const [endError, setEndError] = useState("");
  const loadSequence = useRef(0);
  const searchSequence = useRef(0);
  const saveSequence = useRef(0);
  const searchTimer = useRef<number | null>(null);
  const activeTarget = useRef("");
  const openButton = useRef<HTMLButtonElement | null>(null);
  const purposeID = useId();
  const customPurposeID = useId();
  const customPurposeHelpID = useId();
  const headingID = useId();
  const targetName = targetType === "PROGRAM" ? "Program" : "issue or change";
  activeTarget.current = `${targetType}:${targetID}`;

  const resetForm = useCallback(() => {
    setQuery("");
    setResults([]);
    setSearching(false);
    setSearchAttempted(false);
    setSearchError(false);
    ++searchSequence.current;
    if (searchTimer.current !== null) window.clearTimeout(searchTimer.current);
    searchTimer.current = null;
    setSelectedID("");
    setPurpose("");
    setCustomPurpose("");
    setSaving(false);
    setSaveError(null);
    setSaveErrorKind(null);
    setNotice(null);
  }, []);

  const closeLinking = useCallback(() => {
    setLinking(false);
    resetForm();
    window.setTimeout(() => openButton.current?.focus(), 0);
  }, [resetForm]);

  useEffect(() => {
    ++saveSequence.current;
    ++searchSequence.current;
    setLinks([]);
    setNextCursor(null);
    setLoadMoreError(false);
    setLinking(false);
    setEndingLinkID("");
    setEndReason("");
    setEndError("");
    resetForm();
    void loadLinks();
  }, [targetType, targetID, resetForm]);

  useEffect(() => {
    const searchValue = query.trim();
    if (!linking || !searchValue) return;
    searchTimer.current = window.setTimeout(() => {
      searchTimer.current = null;
      void runSearch(searchValue);
    }, 300);
    return () => {
      if (searchTimer.current !== null) window.clearTimeout(searchTimer.current);
      searchTimer.current = null;
    };
  }, [linking, query]);

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

  async function search() {
    const search = query.trim();
    if (!search) return;
    if (searchTimer.current !== null) window.clearTimeout(searchTimer.current);
    searchTimer.current = null;
    await runSearch(search);
  }

  async function runSearch(search: string) {
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

  async function endLink(event: React.FormEvent, value: LinkedVendor) {
    event.preventDefault();
    const reason = endReason.trim();
    if (!reason) return;
    const sequence = ++saveSequence.current;
    const targetKey = activeTarget.current;
    setEnding(true);
    setEndError("");
    try {
      const ended = await endVendorRelationshipLink(value.link.relationship_id, value.link.id, { expected_version: value.link.version, reason });
      if (sequence !== saveSequence.current || targetKey !== activeTarget.current) return;
      setLinks((current) => current.map((item) => item.link.id === ended.id ? { ...item, link: ended } : item));
      setEndingLinkID("");
      setEndReason("");
      setNotice("Vendor link ended. Existing history remains available.");
    } catch (error) {
      if (sequence !== saveSequence.current || targetKey !== activeTarget.current) return;
      const kind = apiErrorKind(error);
      setEndError(kind === "conflict" ? "This vendor link changed. Refresh related vendors before trying again." : kind === "forbidden" || kind === "unauthorized" ? "Your current access does not allow this vendor link to be ended." : "The vendor link could not be ended. Your reason remains on this screen.");
    } finally {
      if (sequence === saveSequence.current && targetKey === activeTarget.current) setEnding(false);
    }
  }

  return <section className="vendor-relationship-links" aria-labelledby={headingID} aria-hidden={linking ? "true" : undefined} data-testid={`vendor-links-${targetType}-${targetID}`}>
    <div className="section-heading-row">
      <div><h2 id={headingID}>Related vendors</h2><p>Vendor relationships connected to this {targetName} and the recorded purpose for each link.</p></div>
      {loadState === "ready" && <button ref={openButton} type="button" className="primary-button" onClick={() => { if (!linking) { setNotice(null); setLinking(true); } }}>Link vendor</button>}
    </div>
    {notice && <p role="status">{notice}</p>}
    {loadState === "loading" && <p aria-live="polite" aria-busy="true">Loading vendor relationships linked to this {targetName}…</p>}
    {loadState === "failed" && <div role="alert"><p>Related vendors could not be loaded for this {targetName}. No link has been changed.</p><button type="button" className="primary-button" onClick={() => void loadLinks()}>Try again</button></div>}
    {loadState === "ready" && links.length === 0 && <p>No vendor relationships are linked to this {targetName}.</p>}
    {loadState === "ready" && links.length > 0 && <ul className="vendor-link-list">{links.map((value) => { const { link, relationship } = value; return <li key={link.id} className={link.state === "ENDED" ? "ended" : undefined}>
      <div className="vendor-link-identity">{relationship ? <><strong>{relationship.vendor.legal_name}</strong><span>{relationship.relationship.service_name}</span></> : <><strong>Vendor details unavailable</strong><span>The linked relationship could not be loaded in the current scope.</span></>}</div>
      <div className="vendor-link-purpose"><span>{link.purpose_label}</span>{link.state === "ENDED" ? <small>Link ended</small> : <button type="button" className="text-button" aria-label={`End link for ${relationship?.vendor.legal_name ?? "vendor relationship"}`} onClick={() => { setEndingLinkID(link.id); setEndReason(""); setEndError(""); setNotice(null); }}>End link</button>}</div>
      {endingLinkID === link.id && <form className="vendor-link-end" onSubmit={(event) => void endLink(event, value)}><label>Reason for ending this link<textarea rows={3} maxLength={1000} value={endReason} onChange={(event) => setEndReason(event.target.value)} required/></label>{endError && <p role="alert">{endError}</p>}<div className="form-actions"><button type="button" className="secondary-button" disabled={ending} onClick={() => { setEndingLinkID(""); setEndReason(""); setEndError(""); }}>Keep link</button><button type="submit" className="primary-button" disabled={ending || !endReason.trim()}>{ending ? "Ending…" : "End vendor link"}</button></div></form>}
    </li>; })}</ul>}
    {loadState === "ready" && nextCursor && <button type="button" className="secondary-button vendor-link-load-more" disabled={loadingMore} onClick={() => void loadLinks(nextCursor)}>{loadingMore ? "Loading…" : "Load more related vendors"}</button>}
    {loadMoreError && <div role="alert"><span>More related vendors could not be loaded. The current list remains available.</span><button type="button" className="secondary-button" onClick={() => nextCursor && void loadLinks(nextCursor)}>Try again</button></div>}
    {linking && <FocusedSheet label={`Link vendor to this ${targetName}`} onClose={closeLinking} panelClassName="vendor-link-sheet">
      <form className="vendor-link-form" onSubmit={(event) => void submit(event)}>
      <div className="vendor-link-sheet-heading"><span className="eyebrow">Related vendor</span><h2>Link vendor to this {targetName}</h2><p>Search the vendor register, choose the relevant service relationship, and record why it supports this {targetName}.</p></div>
      <div className="form-grid vendor-link-search"><label>Search vendor relationships<input value={query} onChange={(event) => changeQuery(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") { event.preventDefault(); void search(); } }} autoComplete="off" placeholder="Vendor or service name"/></label><button type="button" className="secondary-button" disabled={searching || !query.trim()} onClick={() => void search()}>{searching ? "Searching…" : "Search vendors"}</button></div>
      {searchError && <div role="alert"><span>Vendor search is unavailable. Your search remains on this screen.</span><button type="button" className="secondary-button" disabled={searching} onClick={() => void search()}>Try again</button></div>}
      {!searchError && searchAttempted && results.length === 0 && <p>No vendor relationships match this search. Check the vendor or service name and search again.</p>}
      {results.length > 0 && <fieldset className="vendor-link-results"><legend>Search results</legend>{results.map((item) => <label key={item.relationship.id} className={selectedID === item.relationship.id ? "selected" : undefined}><input type="radio" name="vendor-relationship" value={item.relationship.id} checked={selectedID === item.relationship.id} onChange={() => setSelectedID(item.relationship.id)}/><VendorBrandIcon vendorID={item.vendor.id} legalName={item.vendor.legal_name} brand={item.brand}/><span className="vendor-link-result-copy"><strong>{item.vendor.legal_name}</strong><span>{item.relationship.service_name}</span><small>{relationshipContext(item)}</small></span></label>)}</fieldset>}
      <div className="form-grid"><div><label htmlFor={purposeID}>Relationship purpose</label><select id={purposeID} value={purpose} onChange={(event) => setPurpose(event.target.value as PurposeSelection)}><option value="">Choose a purpose</option>{Object.entries(relationshipPurposes).map(([code, label]) => <option key={code} value={code}>{label}</option>)}<option value="OTHER">Other</option></select></div>{purpose === "OTHER" && <div><label htmlFor={customPurposeID}>Custom purpose</label><input id={customPurposeID} aria-describedby={customPurposeHelpID} value={customPurpose} maxLength={160} required onChange={(event) => setCustomPurpose(event.target.value)}/><small id={customPurposeHelpID}>Briefly state how this vendor relationship supports the current record.</small></div>}</div>
      {saveError && <div role="alert"><span>{saveError}</span>{saveErrorKind === "conflict" && <button type="button" className="secondary-button" onClick={() => void loadLinks()}>Refresh related vendors</button>}</div>}
      <div className="form-actions"><button type="button" className="secondary-button" disabled={saving} onClick={closeLinking}>Cancel</button><button type="submit" className="primary-button" disabled={saving || !selectedID || !relationshipPurposeInput(purpose, customPurpose)}>{saving ? "Linking…" : "Link vendor"}</button></div>
      </form>
    </FocusedSheet>}
  </section>;
}

function relationshipContext(item: VendorRelationshipAggregate) {
  const criticality = item.relationship.criticality.replaceAll("_", " ").toLowerCase();
  const status = item.relationship.status.replaceAll("_", " ").toLowerCase();
  const titleCase = (value: string) => value ? (value[0]?.toUpperCase() ?? "") + value.slice(1) : value;
  return `${titleCase(criticality)} · ${titleCase(status)}`;
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
