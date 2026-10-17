import { useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent } from "react";
import type { FrameworkEditionTarget } from "../appRouting";
import { apiErrorKind } from "../http";
import { decideProgramFrameworkAdoption, proposeProgramFrameworkAdoption } from "../programOperationsApi";
import type { ProgramOperation } from "../programOperationsApi";
import type { ProgramAggregate } from "../types";
import { useFrameworkEditionCatalog } from "../hooks/useFrameworkEditionCatalog";
import { Button, FocusedSheet, Notice, SearchField, SelectField, StatusBadge, TextArea } from "./ui";
import { FrameworkEditionProvenance, trustedFrameworkSourceURL } from "./FrameworkEditionProvenance";
import "../program-standards.css";

type Props = {
  aggregate: ProgramAggregate;
  operations: ProgramOperation[];
  frameworkEdition?: FrameworkEditionTarget;
  onUpdated: (value: ProgramAggregate) => void;
  onReload: () => void;
};

function adoptionStatus(status: string) {
  if (status === "ACTIVE") return { label: "Active", tone: "success" as const };
  if (status === "PENDING") return { label: "Awaiting approval", tone: "warning" as const };
  return { label: status === "REJECTED" ? "Rejected" : status === "SUPERSEDED" ? "Superseded" : "Retired", tone: "neutral" as const };
}

export function ProgramStandardsPanel({ aggregate, operations, frameworkEdition, onUpdated, onReload }: Props) {
  const program = aggregate.program;
  const adoptions = aggregate.framework_adoptions ?? [];
  const pending = adoptions.filter((item) => item.status === "PENDING");
  const proposeOperation = operations.find((operation) => operation.command === "program.framework.propose");
  const approveOperation = operations.find((operation) => operation.command === "program.framework.approve");
  const rejectOperation = operations.find((operation) => operation.command === "program.framework.reject");
  const canReview = Boolean(approveOperation?.can_act || rejectOperation?.can_act);
  const [addOpen, setAddOpen] = useState(false);
  const [jurisdiction, setJurisdiction] = useState("GLOBAL");
  const [search, setSearch] = useState("");
  const [editionID, setEditionID] = useState("");
  const [proposalRationale, setProposalRationale] = useState("");
  const [reviewingID, setReviewingID] = useState<string | null>(null);
  const [reviewJurisdiction, setReviewJurisdiction] = useState("GLOBAL");
  const [reviewRationale, setReviewRationale] = useState("");
  const [busy, setBusy] = useState(false);
  const inFlight = useRef(false);
  const [error, setError] = useState("");
  const [reloadRequired, setReloadRequired] = useState(false);
  const selectedReview = pending.find((item) => item.id === reviewingID);
  const reviewing = Boolean(selectedReview && canReview);
  const catalogJurisdiction = reviewing ? reviewJurisdiction : jurisdiction;
  const catalog = useFrameworkEditionCatalog(
    (addOpen && Boolean(proposeOperation?.can_act)) || reviewing,
    catalogJurisdiction,
  );

  // A decision is tied to a particular Program version, never another record.
  useEffect(() => {
    setAddOpen(false);
    setReviewingID(null);
    setEditionID("");
    setProposalRationale("");
    setReviewRationale("");
    setError("");
    setReloadRequired(false);
  }, [program.id, program.version]);

  useEffect(() => {
    if (!frameworkEdition || !proposeOperation?.can_act) return;
    // A URL carries only identity; the selected metadata must be read from the
    // current authorized catalog page before a material proposal can proceed.
    setJurisdiction(frameworkEdition.jurisdiction);
    setEditionID(frameworkEdition.editionID);
    setProposalRationale("");
    setReviewingID(null);
    setReviewRationale("");
    setError("");
    setReloadRequired(false);
    setAddOpen(true);
  }, [frameworkEdition?.editionID, frameworkEdition?.jurisdiction, program.id, proposeOperation?.can_act]);

  const loadedMatches = useMemo(() => {
    const query = search.trim().toLocaleLowerCase();
    if (!query) return catalog.items;
    return catalog.items.filter((item) => [
      item.title, item.publisher_code, item.framework_code, item.edition_code, item.jurisdiction,
    ].some((value) => value.toLocaleLowerCase().includes(query)));
  }, [catalog.items, search]);

  const selectedEdition = catalog.items.find((item) => item.id === editionID);
  const alreadyLinked = adoptions.some((item) =>
    item.edition_id === editionID && (item.status === "ACTIVE" || item.status === "PENDING"));
  // A pending review must match the exact published edition identity. A
  // jurisdiction cursor or an edition with reused labels is not evidence.
  const pinnedEdition = selectedReview && catalog.items.find((item) => item.id === selectedReview.edition_id);
  const reviewEdition = pinnedEdition && selectedReview &&
    pinnedEdition.publisher_code === selectedReview.publisher_code &&
    pinnedEdition.framework_code === selectedReview.framework_code &&
    pinnedEdition.edition_code === selectedReview.edition_code ? pinnedEdition : null;
  const reviewSourceReady = Boolean(reviewEdition &&
    catalog.state === "ready" && catalog.loadedJurisdiction === reviewJurisdiction &&
    trustedFrameworkSourceURL(reviewEdition.source_url) &&
    (reviewEdition.license_basis === "CITATIONS_ONLY" || reviewEdition.license_basis === "SUMMARIES_PERMITTED"));
  const localJurisdiction = program.jurisdiction?.trim().toUpperCase();
  const reviewJurisdictions = localJurisdiction && localJurisdiction !== "GLOBAL" && /^[A-Z][A-Z0-9._-]{0,63}$/.test(localJurisdiction)
    ? [{ id: "GLOBAL", label: "Global" }, { id: localJurisdiction, label: localJurisdiction }] : [{ id: "GLOBAL", label: "Global" }];
  const activeCount = adoptions.filter((item) => item.status === "ACTIVE").length;

  function openCatalog() {
    setJurisdiction("GLOBAL");
    setSearch("");
    setEditionID("");
    setProposalRationale("");
    setError("");
    setReloadRequired(false);
    setReviewingID(null);
    setReviewRationale("");
    setAddOpen(true);
  }

  function openReview(id: string) {
    setAddOpen(false);
    setReviewingID(id);
    setReviewJurisdiction("GLOBAL");
    setReviewRationale("");
    setError("");
    setReloadRequired(false);
  }

  function closeReview() {
    if (inFlight.current) return;
    setReviewingID(null);
    setReviewRationale("");
    setError("");
    setReloadRequired(false);
  }

  async function submitProposal(event: FormEvent) {
    event.preventDefault();
    if (inFlight.current || !proposeOperation?.can_act || !selectedEdition || alreadyLinked ||
        !proposalRationale.trim() || reloadRequired || catalog.state !== "ready" ||
        catalog.loadedJurisdiction !== jurisdiction) return;
    inFlight.current = true;
    setBusy(true);
    setError("");
    try {
      const updated = await proposeProgramFrameworkAdoption(
        program.id, program.version, selectedEdition.id, proposalRationale.trim(),
      );
      onUpdated(updated);
      setAddOpen(false);
      setProposalRationale("");
    } catch (cause) {
      setReloadRequired(apiErrorKind(cause) === "conflict");
      setError(cause instanceof Error ? cause.message : "Standard could not be submitted.");
    } finally {
      inFlight.current = false;
      setBusy(false);
    }
  }

  async function decide(id: string, action: "approve" | "reject") {
    const allowed = action === "approve" ? approveOperation?.can_act : rejectOperation?.can_act;
    if (inFlight.current || !allowed || reviewingID !== id || !selectedReview || !reviewRationale.trim() ||
        reloadRequired || (action === "approve" && !reviewSourceReady)) return;
    inFlight.current = true;
    setBusy(true);
    setError("");
    try {
      const updated = await decideProgramFrameworkAdoption(
        program.id, id, program.version, action, reviewRationale.trim(),
      );
      onUpdated(updated);
      setReviewingID(null);
      setReviewRationale("");
    } catch (cause) {
      setReloadRequired(apiErrorKind(cause) === "conflict");
      setError(cause instanceof Error ? cause.message : "Standard review could not be saved.");
    } finally {
      inFlight.current = false;
      setBusy(false);
    }
  }

  return <section aria-label="Standards" className="program-record-subsection cs-program-standards">
    <div className="program-panel-heading">
      <div>
        <h3>Standards</h3>
        <p className="cs-program-standards__summary">
          {adoptions.length ? activeCount + " active · " + pending.length + " awaiting approval" : "No standards linked"}
        </p>
      </div>
      <div className="cs-program-standards__actions">
        <a className="cs-program-standards__source" href="#programs/library">Browse library</a>
        {proposeOperation?.can_act && !addOpen && <Button variant="secondary" onPress={openCatalog}>Add standard</Button>}
      </div>
    </div>

    {frameworkEdition && !proposeOperation?.can_act && <p role="status" className="cs-program-standards__meta">Edition selected. Adoption is unavailable until Program authority and review status are confirmed.</p>}
    {adoptions.length > 0 && <ul className="cs-program-standards__adoptions" aria-label="Program standards">
      {adoptions.map((item) => {
        const status = adoptionStatus(item.status);
        const name = item.publisher_code + " · " + item.framework_code + " " + item.edition_code;
        return <li key={item.id} className="cs-program-standards__adoption">
          <div>
            <strong>{name}</strong>
            <span className="cs-program-standards__meta">{item.status === "PENDING" ? "Proposed " + item.proposed_at.slice(0, 10) : item.reviewed_at ? "Reviewed " + item.reviewed_at.slice(0, 10) : "Edition pinned"}</span>
          </div>
          <StatusBadge tone={status.tone}>{status.label}</StatusBadge>
          {item.status === "PENDING" && canReview &&
            <Button variant="secondary" size="compact" isDisabled={busy} onPress={() => openReview(item.id)}>Review {item.framework_code} {item.edition_code}</Button>}
        </li>;
      })}
    </ul>}

    {selectedReview && canReview && <FocusedSheet
      label={"Review standard adoption: " + selectedReview.framework_code + " " + selectedReview.edition_code}
      closeLabel="Close standard review" size="wide" isDismissable={!busy} onClose={closeReview}>
      <div className="cs-program-standards__review" aria-label="Standard adoption review">
      <div className="cs-program-standards__section-head">
        <h4>{selectedReview.publisher_code} · {selectedReview.framework_code} {selectedReview.edition_code}</h4>
        <StatusBadge tone="warning">Awaiting approval</StatusBadge>
      </div>
      <dl className="cs-program-standards__facts">
        <div><dt>Proposed by</dt><dd>{selectedReview.proposed_by}</dd></div>
        <div><dt>Proposed</dt><dd>{selectedReview.proposed_at.slice(0, 10)}</dd></div>
        <div><dt>Proposal rationale</dt><dd>{selectedReview.rationale}</dd></div>
        <div><dt>Edition</dt><dd>{selectedReview.edition_code}</dd></div>
      </dl>
      <section className="cs-program-standards__review-source" aria-label="Published edition source">
        <h4>Published source</h4>
        {reviewJurisdictions.length > 1 && <SelectField
          label="Catalog jurisdiction" placeholder="Select jurisdiction"
          value={reviewJurisdiction} options={reviewJurisdictions} allowsEmpty={false}
          onChange={(value) => { if (value) setReviewJurisdiction(value); }}/>}
        {catalog.state === "loading" && <p role="status">Loading published edition…</p>}
        {catalog.state === "denied" && <Notice tone="warning">Published source access denied. Approval is unavailable.</Notice>}
        {catalog.state === "error" && <Notice tone="warning">Published source unavailable.
          <Button variant="secondary" size="compact" onPress={catalog.retry}>Retry source</Button>
        </Notice>}
        {catalog.state === "ready" && catalog.loadedJurisdiction === reviewJurisdiction && reviewEdition &&
          <FrameworkEditionProvenance edition={reviewEdition}/>}
        {catalog.state === "ready" && catalog.loadedJurisdiction === reviewJurisdiction && !reviewEdition &&
          <Notice tone={pinnedEdition ? "error" : "info"}>
            {pinnedEdition ? "Published edition identity does not match this adoption."
              : catalog.nextAfterID ? "The pending edition is not in the loaded catalog page."
              : "Pending edition not found in this catalog jurisdiction."}
            {!pinnedEdition && catalog.nextAfterID &&
              <Button variant="secondary" size="compact" onPress={catalog.loadMore}>Load more editions</Button>}
          </Notice>}
        {reviewEdition && !reviewSourceReady && catalog.state === "ready" &&
          <Notice tone="warning">The source URL or content rights cannot be verified for approval.</Notice>}
      </section>
      <div className="cs-program-standards__reason">
        <TextArea label={"Decision rationale for " + selectedReview.framework_code + " " + selectedReview.edition_code}
          isRequired rows={3} isDisabled={busy} value={reviewRationale} onChange={setReviewRationale}/>
      </div>
      <div className="cs-program-standards__actions">
        {approveOperation?.can_act && <Button variant="primary" isLoading={busy}
          isDisabled={!reviewRationale.trim() || reloadRequired || !reviewSourceReady} onPress={() => void decide(selectedReview.id, "approve")}>Approve standard</Button>}
        {rejectOperation?.can_act && <Button variant="secondary" isDisabled={busy || !reviewRationale.trim() || reloadRequired}
          onPress={() => void decide(selectedReview.id, "reject")}>Reject standard</Button>}
        <Button variant="quiet" isDisabled={busy} onPress={closeReview}>Cancel</Button>
      </div>
      {error && <Notice tone="error"><span>{error}</span>
        <Button variant="secondary" size="compact" onPress={onReload}>Reload Program</Button>
      </Notice>}
      </div>
    </FocusedSheet>}

    {addOpen && <div className="cs-program-standards__browser">
      <div className="cs-program-standards__section-head">
        <div><h4>Framework library</h4><p>{catalog.items.length} loaded{catalog.nextAfterID ? " · More available" : ""}</p></div>
        <Button variant="quiet" isDisabled={busy} onPress={() => { setAddOpen(false); setError(""); }}>Close</Button>
      </div>
      <div className="cs-program-standards__filters">
        <SearchField label="Search loaded standards" placeholder="Search loaded standards" value={search} onChange={setSearch}/>
        <label><span>Jurisdiction</span>
          <select value={jurisdiction} onChange={(event) => { setJurisdiction(event.target.value); setEditionID(""); setSearch(""); }}>
            <option value="GLOBAL">Global</option>
            {program.jurisdiction && program.jurisdiction.toUpperCase() !== "GLOBAL" &&
              <option value={program.jurisdiction.toUpperCase()}>{program.jurisdiction}</option>}
          </select>
        </label>
      </div>
      {catalog.state === "loading" && <p role="status">Loading published editions…</p>}
      {catalog.state === "denied" && <p role="alert">Published standards are unavailable in this scope. Select an authorized legal entity.</p>}
      {catalog.state === "error" && <div role="alert" className="cs-program-standards__notice">
        Published editions could not be loaded. <Button variant="secondary" size="compact" onPress={catalog.retry}>Retry</Button>
      </div>}
      {catalog.state === "ready" && catalog.items.length === 0 && <p role="status">No published standards available for this jurisdiction.</p>}
      {catalog.state === "ready" && editionID === frameworkEdition?.editionID && !selectedEdition && <p role="status">{catalog.nextAfterID ? "The selected edition is outside the loaded page. Load more standards to verify its source." : "The selected edition is unavailable in this jurisdiction. Return to the Framework Library."}</p>}
      {catalog.items.length > 0 && loadedMatches.length === 0 && <p role="status">No matches in the loaded standards.</p>}
      {loadedMatches.length > 0 && <fieldset className="cs-program-standards__editions" aria-label="Published editions">
        <legend className="cs-sr-only">Published editions</legend>
        {loadedMatches.map((edition) => {
          const linked = adoptions.some((item) => item.edition_id === edition.id && (item.status === "ACTIVE" || item.status === "PENDING"));
          return <label className="cs-program-standards__edition" key={edition.id} data-selected={editionID === edition.id || undefined}>
            <input type="radio" name="framework-edition" value={edition.id} checked={editionID === edition.id}
              onChange={() => { setEditionID(edition.id); setError(""); }} disabled={busy || linked}/>
            <span className="cs-program-standards__edition-body">
              <strong>{edition.title}</strong>
              <small>{edition.publisher_code} · {edition.framework_code} · Edition {edition.edition_code}</small>
              <small>{edition.jurisdiction} · Published {edition.source_published_on} · {edition.license_basis === "CITATIONS_ONLY" ? "Citations only" : "Licensed summaries"}</small>
            </span>
            {linked && <StatusBadge tone="neutral">Already linked</StatusBadge>}
          </label>;
        })}
      </fieldset>}
      {catalog.nextAfterID && <Button variant="secondary" size="compact" isDisabled={catalog.state === "loading"}
        onPress={catalog.loadMore}>Load more standards</Button>}
      {selectedEdition && <div className="cs-program-standards__selection" aria-label="Selected edition">
        <div className="cs-program-standards__section-head"><h4>{selectedEdition.title}</h4><StatusBadge tone="info">Published edition</StatusBadge></div>
        <FrameworkEditionProvenance edition={selectedEdition}/>
      </div>}
      <form className="cs-program-standards__proposal" onSubmit={(event) => void submitProposal(event)}>
      <div className="cs-program-standards__reason">
        <TextArea label="Reason for adoption" isRequired rows={3} isDisabled={busy}
          value={proposalRationale} onChange={setProposalRationale}/>
      </div>
      <div className="cs-program-standards__actions">
        <Button type="submit" variant="primary" isLoading={busy}
          isDisabled={!selectedEdition || alreadyLinked || !proposalRationale.trim() || reloadRequired ||
            catalog.state !== "ready" || catalog.loadedJurisdiction !== jurisdiction}>Submit for approval</Button>
        <Button variant="quiet" isDisabled={busy} onPress={() => { setAddOpen(false); setError(""); }}>Cancel</Button>
      </div>
      </form>
    </div>}
    {error && !selectedReview && <div role="alert" className="cs-program-standards__notice">
      <span>{error}</span> <Button variant="secondary" size="compact" onPress={onReload}>Reload Program</Button>
    </div>}
  </section>;
}
