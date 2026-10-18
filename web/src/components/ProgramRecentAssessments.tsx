import { useEffect, useRef, useState } from "react";
import { loadCompletedResponses, type CompletedResponseSummary } from "../formsDistributionApi";
import type { ProgramSection } from "../appRouting";
import { concernText } from "./forms/responseScorePresentation";
import { Button, Notice } from "./ui";

type State = "loading" | "ready" | "unavailable";
type PeriodDays = 7 | 30 | 90;

function submittedDate(value: string) {
  const date = new Date(value);
  return Number.isFinite(date.getTime())
    ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(date)
    : "Date unavailable";
}

// Approved Program Overview: recent assessments are *submitted responses*,
// not approved compliance or verified evidence. Reuse one authorized,
// bounded Forms query and preserve the independent Assessments workspace.
export function ProgramRecentAssessments({ programID, onNavigate, onOpenAssessment }: {
  programID: string;
  onNavigate: (section: ProgramSection) => void;
  onOpenAssessment?: (response: CompletedResponseSummary) => void;
}) {
  const [period, setPeriod] = useState<PeriodDays>(90);
  const [state, setState] = useState<State>("loading");
  const [items, setItems] = useState<CompletedResponseSummary[]>([]);
  const [error, setError] = useState("");
  const [retry, setRetry] = useState(0);
  const generation = useRef(0);

  useEffect(() => {
    const request = ++generation.current;
    setItems([]);
    setError("");
    setState("loading");
    const completedFrom = new Date(Date.now() - period * 86_400_000).toISOString();
    void loadCompletedResponses({
      subject_type: "PROGRAM", subject_id: programID, current_only: true,
      sort: "COMPLETED_DESC", completed_from: completedFrom, limit: 3,
    }).then((page) => {
      if (generation.current !== request) return;
      if (page.items.some((item) => item.subject_type !== "PROGRAM" ||
          item.subject_id !== programID || !item.current)) {
        throw new Error("The returned assessment population does not match this Program.");
      }
      setItems(page.items);
      setState("ready");
    }).catch(() => {
      if (generation.current !== request) return;
      setState("unavailable");
      setError("Recent submitted responses are unavailable.");
    });
    return () => { generation.current += 1; };
  }, [programID, period, retry]);

  return <section className="program-recent-assessments" aria-labelledby="program-recent-assessments-heading">
    <header className="program-recent-assessments__head">
      <div><h2 id="program-recent-assessments-heading">Recent assessments</h2>
        <p>Submitted responses · Last {period} days</p>
      </div>
      <div className="program-recent-assessments__actions">
        <label>Activity period
          <select aria-label="Assessment activity period" value={period}
            onChange={(event) => setPeriod(Number(event.target.value) as PeriodDays)}>
            <option value={7}>7 days</option>
            <option value={30}>30 days</option>
            <option value={90}>90 days</option>
          </select>
        </label>
        <Button variant="quiet" onPress={() => onNavigate("assessments")}>View assessments</Button>
      </div>
    </header>
    {state === "loading" && <p role="status">Loading recent submissions…</p>}
    {state === "unavailable" && <Notice tone="warning">{error}
      <Button variant="secondary" size="compact" onPress={() => setRetry((n) => n + 1)}>Retry assessments</Button>
    </Notice>}
    {state === "ready" && items.length === 0 && <p>No submitted responses in this period.</p>}
    {state === "ready" && items.length > 0 && <ul className="program-recent-assessments__list">
      {items.map((item) => <li key={item.id} className={onOpenAssessment ? "program-recent-assessments__row has-review" : "program-recent-assessments__row"}>
        <div><strong>{item.title}</strong><small>Form revision {item.form_template_version} · Response revision {item.revision}</small></div>
        <time dateTime={item.completed_at}>{submittedDate(item.completed_at)}</time>
        <span>{item.state === "FINAL" ? concernText(item.score) : "Provisional response"}</span>
        {onOpenAssessment && <Button variant="quiet" aria-label={`Review ${item.title}`}
          onPress={() => onOpenAssessment(item)}>Review</Button>}
      </li>)}
    </ul>}
    <p className="program-recent-assessments__foot">Submission scores do not establish independent review or evidence acceptance.</p>
  </section>;
}
