import { useEffect, useRef, useState, type FormEvent } from "react";
import { listEvidenceRecipientCandidates, type EvidenceRecipientCandidate } from "../evidenceRequestAdminApi";

export function EvidenceRecipientCandidateSelect({ requestID, value, disabled = false, onChange }: { requestID: string; value: string; disabled?: boolean; onChange: (principalID: string) => void }) {
  const [candidates, setCandidates] = useState<EvidenceRecipientCandidate[]>([]);
  const [query, setQuery] = useState("");
  const [hasMore, setHasMore] = useState(false);
  const [state, setState] = useState<"loading" | "ready" | "unavailable">("loading");
  const generation = useRef(0);

  function load(search: string) {
    const current = ++generation.current;
    setCandidates([]);
    setHasMore(false);
    setState("loading");
    void listEvidenceRecipientCandidates(requestID, search).then((page) => {
      if (current !== generation.current) return;
      setCandidates(page.items);
      setHasMore(page.has_more);
      setState("ready");
    }).catch(() => {
      if (current !== generation.current) return;
      setState("unavailable");
    });
  }

  useEffect(() => {
    setQuery("");
    load("");
    return () => { generation.current += 1; };
  }, [requestID]);

  function search(event: FormEvent) {
    event.preventDefault();
    if (disabled || state === "loading") return;
    onChange("");
    load(query.trim());
  }

  return <>
    <form onSubmit={search}>
      <label className="capture-field"><span>Find eligible person</span><input maxLength={100} value={query} disabled={disabled} onChange={(event) => setQuery(event.target.value)} placeholder="Search by name, position or directory role"/></label>
      <button className="secondary-button" type="submit" disabled={disabled || state === "loading"}>{state === "loading" ? "Loading people…" : "Search people"}</button>
    </form>
    <label className="capture-field"><span>New assigned person</span><select value={value} disabled={disabled || state !== "ready"} onChange={(event) => onChange(event.target.value)}>
      <option value="">{state === "loading" ? "Loading eligible people…" : "Choose an eligible person"}</option>
      {candidates.map((candidate) => <option key={candidate.principal_id} value={candidate.principal_id}>{candidate.context_label ? `${candidate.display_name} · ${candidate.context_label}` : candidate.display_name}</option>)}
    </select></label>
    {state === "unavailable" && <p className="field-help" role="status">Eligible people could not be loaded. Reload the evidence request before changing the assigned person.</p>}
    {state === "ready" && candidates.length === 0 && <p className="field-help" role="status">No eligible people are available for this request. Ask the access administrator to check who can view this Program or issue before reassigning it.</p>}
    {state === "ready" && hasMore && <p className="field-help" role="status">More eligible people match this request. Search by name, position or directory role to narrow the list.</p>}
  </>;
}
