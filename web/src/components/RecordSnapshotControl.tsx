import { useRef, useState } from "react";

type SnapshotSummary = { version: number; status: string; updatedAt: string };
type Props = { recordLabel: "Program" | "issue"; loadSnapshot: (at: string) => Promise<SnapshotSummary> };

function readable(value: string) {
  return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

export function RecordSnapshotControl({ recordLabel, loadSnapshot }: Props) {
  const [at, setAt] = useState("");
  const [snapshot, setSnapshot] = useState<SnapshotSummary | null>(null);
  const [shownAt, setShownAt] = useState("");
  const [state, setState] = useState<"idle" | "loading" | "unavailable">("idle");
  const requestID = useRef(0);

  async function viewSnapshot() {
    if (!at) return;
    const current = ++requestID.current;
    setState("loading");
    try {
      const exact = new Date(at).toISOString();
      const value = await loadSnapshot(exact);
      if (current !== requestID.current) return;
      setSnapshot(value); setShownAt(exact); setState("idle");
    } catch {
      if (current !== requestID.current) return;
      setSnapshot(null); setShownAt(""); setState("unavailable");
    }
  }

  return <details className="record-snapshot-control"><summary>View an earlier {recordLabel} record</summary>
    <p>This retrieves the stored record as it stood at the selected date and time. It is not a list of changes.</p>
    <label><span>Date and time</span><input type="datetime-local" value={at} onChange={(event) => setAt(event.target.value)}/></label>
    <button className="secondary-button" type="button" disabled={!at || state === "loading"} onClick={() => void viewSnapshot()}>{state === "loading" ? "Loading record…" : "View earlier record"}</button>
    {state === "unavailable" && <p role="alert">The earlier {recordLabel} record could not be loaded for that date and time. Choose another time or retry.</p>}
    {snapshot && <section aria-label={`${recordLabel} record at selected time`}><strong>{recordLabel === "Program" ? "Program" : "Issue"} record at {new Date(shownAt).toLocaleString()}</strong><dl><div><dt>Status at that time</dt><dd>{readable(snapshot.status)}</dd></div><div><dt>Record version</dt><dd>{snapshot.version}</dd></div><div><dt>Last change included</dt><dd>{new Date(snapshot.updatedAt).toLocaleString()}</dd></div></dl></section>}
  </details>;
}
