import { useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent } from "react";
import { apiErrorKind } from "../http";
import { decideProgramFrameworkAdoption, proposeProgramFrameworkAdoption } from "../programOperationsApi";
import type { ProgramOperation } from "../programOperationsApi";
import type { ProgramAggregate } from "../types";
import { useFrameworkEditionCatalog } from "../hooks/useFrameworkEditionCatalog";
import { Button, SearchField, StatusBadge } from "./ui";
import "../program-standards.css";

type Props = {
  aggregate: ProgramAggregate;
  operations: ProgramOperation[];
  onUpdated: (value: ProgramAggregate) => void;
  onReload: () => void;
};

function adoptionStatus(status: string) {
  if (status === "ACTIVE") return { label: "Active", tone: "success" as const };
  if (status === "PENDING") return { label: "Awaiting approval", tone: "warning" as const };
  return { label: status === "REJECTED" ? "Rejected" : status === "SUPERSEDED" ? "Superseded" : "Retired", tone: "neutral" as const };
}

export function ProgramStandardsPanel({ aggregate, operations, onUpdated, onReload }: Props) {
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
  const [reviewRationale, setReviewRationale] = useState("");
  const [busy, setBusy] = useState(false);
  const inFlight = useRef(false);
  const [error, setError] = useState("");
  const [reloadRequired, setReloadRequired] = useState(false);
  const catalog = useFrameworkEditionCatalog(addOpen && Boolean(proposeOperation?.can_act), jurisdiction);

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
  const selectedReview = pending.find((item) => item.id === reviewingID);
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
    setReviewRationale("");
    setError("");
    setReloadRequired(false);
  }

  async function submitProposal(event: FormEvent) {
    event.preventDefault();
    if (inFlight.current || !proposeOperation?.can_act || !selectedEdition || alreadyLinked ||
        !proposalRationale.trim() || reloadRequired || catalog.state !== "ready") return;
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
    if (inFlight.current || !allowed || reviewingID !== id || !selectedReview || !reviewRationale.trim() || reloadRequired) return;
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
      {proposeOperation?.can_act && !addOpen &&
        <Button variant="secondary" onPress={openCatalog}>Add standard</Button>}
    </div>

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

    {selectedReview && canReview && <div className="cs-program-standards__review" aria-label="Standard adoption review">
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
      <label className="cs-program-standards__reason">
        <span>Decision rationale</span>
        <textarea required value={reviewRationale} onChange={(event) => setReviewRationale(event.target.value)}
          aria-label={"Decision rationale for " + selectedReview.framework_code + " " + selectedReview.edition_code}/>
      </label>
      <div className="cs-program-standards__actions">
        {approveOperation?.can_act && <Button variant="primary" isLoading={busy}
          isDisabled={!reviewRationale.trim() || reloadRequired} onPress={() => void decide(selectedReview.id, "approve")}>Approve standard</Button>}
        {rejectOperation?.can_act && <Button variant="secondary" isDisabled={busy || !reviewRationale.trim() || reloadRequired}
          onPress={() => void decide(selectedReview.id, "reject")}>Reject standard</Button>}
        <Button variant="quiet" isDisabled={busy} onPress={() => { setReviewingID(null); setReviewRationale(""); setError(""); }}>Cancel</Button>
      </div>
    </div>}

    {addOpen && <form className="cs-program-standards__browser" onSubmit={(event) => void submitProposal(event)}>
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
        <dl className="cs-program-standards__facts">
          <div><dt>Publisher</dt><dd>{selectedEdition.publisher_code}</dd></div>
          <div><dt>Edition</dt><dd>{selectedEdition.edition_code}</dd></div>
          <div><dt>Jurisdiction</dt><dd>{selectedEdition.jurisdiction}</dd></div>
          <div><dt>Published</dt><dd>{selectedEdition.source_published_on}</dd></div>
          <div><dt>Source rights</dt><dd>{selectedEdition.license_basis === "CITATIONS_ONLY" ? "Citations only" : "Licensed summaries"}</dd></div>
          {selectedEdition.effective_on && <div><dt>Effective</dt><dd>{selectedEdition.effective_on}</dd></div>}
          {selectedEdition.approval_reference && <div><dt>Release approval</dt><dd>{selectedEdition.approval_reference}</dd></div>}
        </dl>
        {selectedEdition.source_url.startsWith("https://") ?
          <a className="cs-program-standards__source" href={selectedEdition.source_url} target="_blank" rel="noopener noreferrer">Show published source ↗</a>
          : <span className="cs-program-standards__meta">Source reference unavailable</span>}
      </div>}
      <label className="cs-program-standards__reason"><span>Reason for adoption</span>
        <textarea required value={proposalRationale} onChange={(event) => setProposalRationale(event.target.value)}/>
      </label>
      <div className="cs-program-standards__actions">
        <Button type="submit" variant="primary" isLoading={busy}
          isDisabled={!selectedEdition || alreadyLinked || !proposalRationale.trim() || reloadRequired || catalog.state !== "ready"}>Submit for approval</Button>
        <Button variant="quiet" isDisabled={busy} onPress={() => { setAddOpen(false); setError(""); }}>Cancel</Button>
      </div>
    </form>}
    {error && <div role="alert" className="cs-program-standards__notice">
      <span>{error}</span> <Button variant="secondary" size="compact" onPress={onReload}>Reload Program</Button>
    </div>}
  </section>;
}
