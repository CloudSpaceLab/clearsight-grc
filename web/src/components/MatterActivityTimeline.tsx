import { useEffect, useState } from "react";
import type { AuthorityPrincipal, MatterActivityItem } from "../types";
import { addMatterComment, loadMatterActivity } from "../matterCollaborationApi";
import { Button, Notice, TextArea } from "./ui";

type Props = { matterID: string; matterVersion: number; candidates: AuthorityPrincipal[]; onUpdated?: () => void };

function labelFor(item: MatterActivityItem, candidates: AuthorityPrincipal[]) {
  return candidates.find((candidate) => candidate.id === item.actor_id)?.display_name ?? (item.actor_type === "SYSTEM" ? "Automated process" : "Recorded person");
}

function activityLabel(item: MatterActivityItem) {
  if (item.comment) return "Comment";
  if (item.update_request) return "Status update requested";
  return item.event_type.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, value => value.toUpperCase());
}

export function MatterActivityTimeline({ matterID, matterVersion, candidates, onUpdated }: Props) {
  const [items, setItems] = useState<MatterActivityItem[]>([]);
  const [nextBefore, setNextBefore] = useState<number>();
  const [loading, setLoading] = useState(true);
  const [loadingOlder, setLoadingOlder] = useState(false);
  const [body, setBody] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [mentionedIDs, setMentionedIDs] = useState<string[]>([]);
  const [selectedMentionID, setSelectedMentionID] = useState("");
  const uniqueCandidates = candidates.filter((candidate, index) => candidates.findIndex((value) => value.id === candidate.id) === index);

  useEffect(() => {
    let active = true;
    setLoading(true); setError("");
    void loadMatterActivity(matterID).then(page => { if (active) { setItems(page.items); setNextBefore(page.next_before_version); } }).catch(() => { if (active) setError("Issue activity could not be loaded."); }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [matterID]);

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (!body.trim()) return;
    setSaving(true); setError("");
    try {
      await addMatterComment(matterID, matterVersion, body.trim(), mentionedIDs);
      setBody("");
      setMentionedIDs([]);
      onUpdated?.();
      const page = await loadMatterActivity(matterID);
      setItems(page.items); setNextBefore(page.next_before_version);
    } catch (cause) { setError(cause instanceof Error ? cause.message : "The comment could not be recorded."); } finally { setSaving(false); }
  }

  async function loadOlder() {
    if (!nextBefore) return;
    setLoadingOlder(true); setError("");
    try { const page = await loadMatterActivity(matterID, nextBefore); setItems(current => [...current, ...page.items]); setNextBefore(page.next_before_version); }
    catch { setError("Earlier issue activity could not be loaded."); } finally { setLoadingOlder(false); }
  }

  return <aside className="matter-activity" aria-label="Issue activity">
    <header><div><span className="eyebrow">Activity</span><h2>Updates and history</h2></div><span>{items.length} recent entries</span></header>
    <form className="matter-activity-composer" onSubmit={submit}>
      <TextArea label="Add internal comment" value={body} onChange={setBody} rows={3} description="Comments are recorded in the issue history."/>
      {uniqueCandidates.length > 0 && <div className="matter-comment-mentions">
        <label><span>Mention colleague</span><select value={selectedMentionID} onChange={(event) => {
          const value = event.target.value;
          setSelectedMentionID("");
          if (value) setMentionedIDs((current) => current.includes(value) ? current : [...current, value]);
        }}><option value="">Select colleague</option>{uniqueCandidates.filter((candidate) => !mentionedIDs.includes(candidate.id)).map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name}</option>)}</select></label>
        {mentionedIDs.length > 0 && <div className="matter-comment-mention-list" aria-label="Mentioned colleagues">{mentionedIDs.map((id) => <button key={id} type="button" onClick={() => setMentionedIDs((current) => current.filter((value) => value !== id))}>@{uniqueCandidates.find((candidate) => candidate.id === id)?.display_name ?? "Colleague"} ×</button>)}</div>}
      </div>}
      <Button type="submit" variant="primary" isDisabled={!body.trim()} isLoading={saving}>Add comment</Button>
    </form>
    {error && <Notice tone="error">{error}</Notice>}
    {loading ? <p className="matter-activity-state" role="status">Loading issue activity…</p> : <ol className="matter-activity-list">{items.map(item => <li key={item.event_id}>
      <div className="matter-activity-entry-heading"><strong>{activityLabel(item)}</strong><time dateTime={item.occurred_at}>{new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" }).format(new Date(item.occurred_at))}</time></div>
      <p>{item.comment?.body ?? item.update_request?.message ?? "Issue state changed."}</p>
      <small>{labelFor(item, uniqueCandidates)}{item.comment?.mentioned_principal_ids?.length ? ` · ${item.comment.mentioned_principal_ids.map((id) => `@${uniqueCandidates.find((candidate) => candidate.id === id)?.display_name ?? "Colleague"}`).join(" ")}` : ""}</small>
    </li>)}</ol>}
    {nextBefore && <Button variant="secondary" onPress={() => void loadOlder()} isLoading={loadingOlder}>Load older activity</Button>}
  </aside>;
}
