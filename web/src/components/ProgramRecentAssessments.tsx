import { useEffect, useRef, useState } from "react";
import { loadCompletedResponses, type CompletedResponseSummary } from "../formsDistributionApi";
import { concernText, concernTone } from "./forms/responseScorePresentation";
import { Button, DataTable, Notice, StatusBadge, type DataColumn } from "./ui";

type Props = {
  programID: string;
  onOpenAssessment: (responseID: string) => void;
  onViewAll: () => void;
};

type Period = 7 | 30 | 90;
type LoadState = "loading" | "ready" | "unavailable";
const periods: readonly Period[] = [7, 30, 90];

export function ProgramRecentAssessments({ programID, onOpenAssessment, onViewAll }: Props) {
  const [period, setPeriod] = useState<Period>(90);
  const [items, setItems] = useState<CompletedResponseSummary[]>([]);
  const [loadedKey, setLoadedKey] = useState("");
  const [state, setState] = useState<LoadState>("loading");
  const [reloadKey, setReloadKey] = useState(0);
  const sequence = useRef(0);
  const key = programID + ":" + period;

  useEffect(() => {
    const request = ++sequence.current;
    setState("loading");
    const completedFrom = new Date(Date.now() - period * 86400000).toISOString();
    void loadCompletedResponses({
      subject_type: "PROGRAM", subject_id: programID, current_only: true,
      completed_from: completedFrom, sort: "COMPLETED_DESC", limit: 5,
    }).then((page) => {
      if (request !== sequence.current) return;
      if (page.items.some((item) =>
        item.subject_type !== "PROGRAM" || item.subject_id !== programID ||
        !item.current)) {
        throw new Error("Returned assessments do not match the selected Program.");
      }
      setItems(page.items);
      setLoadedKey(key);
      setState("ready");
    }).catch(() => {
      if (request !== sequence.current) return;
      setItems([]);
      setLoadedKey(key);
      setState("unavailable");
    });
    return () => { sequence.current++; };
  }, [programID, period, key, reloadKey]);

  const current = loadedKey === key;
  const visible = current && state === "ready" ? items : [];
  const unavailable = current && state === "unavailable";
  const columns: readonly DataColumn<CompletedResponseSummary>[] = [
    { id: "assessment", header: "Assessment", render: (item) => <strong>{item.title}</strong>,
      accessibleText: (item) => item.title },
    { id: "updated", header: "Updated", render: (item) => <time dateTime={item.completed_at}>{displayDate(item.completed_at)}</time>,
      accessibleText: (item) => displayDate(item.completed_at) },
    { id: "status", header: "Result", kind: "status",
      render: (item) => <StatusBadge tone={item.state === "PROVISIONAL" ? "warning" : concernTone(item.score)}>{item.state === "PROVISIONAL" ? "Provisional" : concernText(item.score)}</StatusBadge>,
      accessibleText: (item) => item.state === "PROVISIONAL" ? "Provisional" : concernText(item.score) },
    { id: "action", header: "Review", kind: "action",
      render: (item) => <Button variant="quiet" aria-label={`Review ${item.title}`} onPress={() => onOpenAssessment(item.id)}>Review</Button>,
      accessibleText: (item) => `Review ${item.title}` },
  ];

  return <section className="program-overview-recent" aria-labelledby="program-overview-recent-heading">
    <header className="program-overview-recent__header">
      <div>
        <span className="eyebrow">Activity</span>
        <h2 id="program-overview-recent-heading">Recent assessments</h2>
        <p>Submitted Program responses · last {period} days</p>
      </div>
      <Button variant="quiet" onPress={onViewAll}>View assessments</Button>
    </header>
    <div className="program-overview-recent__periods" role="group" aria-label="Assessment activity period">
      {periods.map((days) => <button key={days} type="button"
        className={period === days ? "program-overview-recent__period is-active" : "program-overview-recent__period"}
        aria-pressed={period === days} onClick={() => setPeriod(days)}>{days} days</button>)}
    </div>
    {(!current || state === "loading") && <p role="status">Loading recent assessments…</p>}
    {unavailable && <Notice tone="warning">Recent assessments could not be loaded.
      <Button variant="secondary" size="compact" onPress={() => setReloadKey((value) => value + 1)}>Retry assessments</Button>
    </Notice>}
    {current && state === "ready" && visible.length === 0 && <p role="status">No submitted assessments in this period.</p>}
    {visible.length > 0 && <DataTable
      ariaLabel="Recent Program assessments"
      rows={visible} rowKey={(item) => item.id}
      rowName={(item) => `${item.title}, submitted ${displayDate(item.completed_at)}`}
      columns={columns} responsiveTo="container"
    />}
    {current && state === "ready" && visible.length > 0 && <p className="program-overview-recent__foot">
      Showing {visible.length} {visible.length === 1 ? "response" : "responses"}
    </p>}
  </section>;
}

function displayDate(value: string) {
  const parsed = new Date(value);
  return Number.isFinite(parsed.getTime())
    ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(parsed)
    : "Submission date unavailable";
}
