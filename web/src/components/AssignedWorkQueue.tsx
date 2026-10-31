import { useEffect, useState } from "react";
import { useActorWorkSearch } from "./useActorWorkSearch";
import { isSourceWorkStatus } from "./workSourceFilters";
import { Button, DataTable, FilterBar, FocusedSheet, MetricCard, SearchField, SelectField, StatusBadge, Tabs, type DataColumn } from "./ui";
import { countLoadedWork, filterLoadedWork, type WorkDueWindow, type WorkQueueFilter } from "./workQueuePresentation";
import { EmptyState } from "./EmptyState";
import { WorkItemIcon } from "./WorkItemIcon";
import type { AttentionItem, InterventionClass } from "../types";

export type AssignedWorkState = "loading" | "live" | "unavailable";
type FocusedAttentionItem = AttentionItem & { action_target_sub_id?: string };

export type AssignedWorkQueueProps = {
  items: AttentionItem[];
  state: AssignedWorkState;
  asOf?: string;
  hasMore?: boolean;
  onOpenItem: (item: AttentionItem) => void;
  onInspectAuthority?: (item: AttentionItem) => void;
  onRetry?: () => void;
  headingLevel?: "h2" | "h3";
  presentation?: "brief" | "workspace";
  sourceSearchScopeKey?: string;
};

export function AssignedWorkQueue({ items, state, asOf, hasMore, onOpenItem, onInspectAuthority, onRetry = () => window.location.reload(), headingLevel = "h2", presentation = "brief", sourceSearchScopeKey }: AssignedWorkQueueProps) {
  const Heading = headingLevel;
  const [filter, setFilter] = useState<WorkQueueFilter>("ALL");
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<string>();
  const [dueWindow, setDueWindow] = useState<WorkDueWindow>("ALL");
  const [selectedID, setSelectedID] = useState<string>();
  const [inspectingItem, setInspectingItem] = useState<AttentionItem>();
  const sourceSearch = presentation === "workspace" && Boolean(sourceSearchScopeKey) && search.trim() !== "";
  useEffect(() => {
    if (sourceSearch && status && !isSourceWorkStatus(status)) setStatus(undefined);
  }, [sourceSearch, status]);
  const searched = useActorWorkSearch(search, sourceSearchScopeKey ?? "", Boolean(sourceSearchScopeKey) && presentation === "workspace", status, dueWindow);
  const currentItems = sourceSearch ? searched.items : items;
  const currentState = sourceSearch ? searched.state === "idle" ? "loading" : searched.state : state;
  const currentHasMore = sourceSearch ? searched.hasMore : hasMore;
  const count = currentItems.length === 1 ? "1 item needs your action" : `${currentItems.length} items need your action`;
  const title = currentState === "loading" ? "Loading assigned work" : currentState === "unavailable" ? "Assigned work is unavailable" : count;
  const snapshotTime = Date.parse((sourceSearch ? searched.asOf : asOf) ?? "");
  const now = Number.isFinite(snapshotTime) ? snapshotTime : Date.now();
  const counts = countLoadedWork(currentItems, now);
  const statuses = sourceSearch ? sourceWorkStatuses : [...new Set(currentItems.map((item) => item.state).filter(Boolean))]
    .sort((a, b) => a.localeCompare(b)).map((value) => ({ id: value, label: value }));
  const filtered = filterLoadedWork(currentItems, filter, sourceSearch ? "" : search, sourceSearch ? undefined : status, now, sourceSearch ? "ALL" : dueWindow);
  const selected = filtered.find((item) => item.id === selectedID) ?? filtered[0];
  // Inspection references are invalidated when the source population changes.
  const inspecting = currentItems.find((item) => item === inspectingItem);

  return <section className={`intervention-brief${presentation === "workspace" ? " assigned-workspace" : ""}`} aria-labelledby="assigned-work-heading">
    {(presentation !== "workspace" || (currentState !== "live" && !sourceSearch)) && <header className="intervention-heading">
      <div><span className="eyebrow">Assigned work</span><Heading id="assigned-work-heading">{title}</Heading></div>
    </header>}
    {currentState === "loading" && !sourceSearch
      ? <div className="workspace-loading" aria-live="polite" aria-busy="true">Loading assigned work…</div>
      : currentState === "unavailable" && !sourceSearch
        ? <EmptyState kind="unavailable" label="Assigned work" title="Assigned work could not be loaded" description="Retry." action="Try again" onAction={sourceSearch ? searched.retry : onRetry}/>
        : presentation === "workspace"
          ? <>
            <Tabs ariaLabel="Assigned work queues" items={workQueues}
              selectedKey={filter === "APPROVALS" ? "APPROVALS" : "ALL"}
              onSelectionChange={(key) => setFilter(key)}>
              {() => <>
                {(!sourceSearch || currentState === "live") && <div className="assigned-workspace__signals" aria-label="Loaded work measures">
                  {workSignals.map(({ key, label, tone }) => <MetricCard
                    key={key} label={label} value={counts[key]} qualityLabel="Loaded work" tone={tone}
                    isSelected={filter === key} ariaControls="attention-list" onPress={() => setFilter(filter === key ? "ALL" : key)}
                  />)}
                </div>}
                <header className="intervention-heading">
                  <div><span className="eyebrow">Assigned work</span><Heading id="assigned-work-heading">Assigned to me</Heading>
                    {(!sourceSearch || currentState === "live") && <p>{currentItems.length} {currentItems.length === 1 ? "item" : "items"} loaded</p>}
                    {sourceSearch
                      ? <p role="status">Workflow assignments only · title matches · loaded results, not total work.</p>
                      : currentHasMore && <p role="status">More assignments exist. Counts and search cover loaded work only.</p>}
                  </div>
                  <div className="assigned-workspace__due-window">
                    <SelectField label="Due-date window" value={dueWindow} placeholder="All due dates"
                      options={dueWindows} allowsEmpty={false}
                      onChange={(value) => setDueWindow(value ?? "ALL")}/>
                  </div>
                </header>
                <FilterBar label={sourceSearch ? "Filter matching assignments" : "Filter loaded work"}
                  fields={<>
                    <SearchField label={sourceSearchScopeKey ? "Search assignment titles" : "Search loaded work"} value={search} onChange={(value) => {
                      setSearch(value.slice(0, 120));
                      if (value.trim() && sourceSearchScopeKey && status && !isSourceWorkStatus(status)) setStatus(undefined);
                    }} placeholder={sourceSearchScopeKey ? "Search assignment titles" : "Search loaded work"}/>
                    <SelectField label="Work status" isLabelHidden value={status} placeholder="All statuses" options={statuses} onChange={setStatus}/>
                  </>}
                  resultCount={filtered.length}
                  resultLabel={(count) => sourceSearch && currentState !== "live" ? currentState === "loading" ? "Searching assignment titles" : "Assignment search unavailable" : `${count} of ${currentItems.length} loaded ${sourceSearch ? "assignment matches" : "items"} · ${workFilterLabel(filter)}`}
                  onClear={search || status || filter !== "ALL" || dueWindow !== "ALL" ? () => { setSearch(""); setStatus(undefined); setDueWindow("ALL"); setFilter("ALL"); } : undefined}
                />
                {sourceSearch && currentState !== "live"
                  ? <div id="attention-list">{currentState === "loading"
                    ? <div className="workspace-loading" aria-live="polite" aria-busy="true">Searching assignment titles…</div>
                    : <EmptyState kind="unavailable" label="Assigned work" title="Assignment search unavailable" description="Retry the search." action="Try again" onAction={searched.retry}/>}</div>
                  : filtered.length ? <div className="assigned-workspace__grid" id="attention-list">
                  <DataTable ariaLabel="Assigned work" rows={filtered} rowKey={(item) => item.id}
                    rowName={(item) => item.title} columns={workColumns} selectedKey={selected?.id}
                    onSelectionChange={(item) => setSelectedID(item.id)}
                    onRowAction={(item) => { setSelectedID(item.id); setInspectingItem(item); }}
                    rowActionLabel="Inspect" responsiveTo="container"/>
                  {selected && <WorkContext item={selected} onOpen={onOpenItem} onInspectAuthority={onInspectAuthority}/>}
                </div> : <div id="attention-list"><EmptyState label="Assigned work" title={sourceSearch ? currentHasMore ? "No matching work loaded yet" : "No matching assignments" : filter === "ALL" && !search && !status ? "Nothing needs your action right now" : "No matching work"}
                  description={sourceSearch ? currentHasMore ? "Load more assignments to check remaining records." : "No workflow assignments match the selected filters." : filter === "ALL" && !search && !status ? "No open assigned work." : "No matching items in the loaded queue."}/></div>}
                {sourceSearch && currentHasMore && <div className="workspace-recovery-actions">
                  <Button variant="secondary" isDisabled={searched.loadingMore} onPress={searched.loadMore}>{searched.loadingMore ? "Loading assignments…" : "Load more assignments"}</Button>
                </div>}
                {sourceSearch && searched.loadMoreFailed && <p role="alert">More assignments could not be loaded. Select Load more assignments to retry.</p>}
              </>}
            </Tabs>
          </>
          : items.length
            ? <div className="intervention-list" id="attention-list">{items.map((item) => <InterventionRow key={item.id} item={item} onOpen={onOpenItem} onInspectAuthority={onInspectAuthority}/>)}</div>
            : <div id="attention-list"><EmptyState label="Assigned work" title="Nothing needs your action right now" description="No open assigned work."/></div>}
    {presentation === "workspace" && currentState === "live" && inspecting && <FocusedSheet label={`Selected work: ${inspecting.title}`} onClose={() => setInspectingItem(undefined)}><WorkContext item={inspecting} onOpen={onOpenItem} onInspectAuthority={onInspectAuthority}/></FocusedSheet>}
  </section>;
}

const workQueues = [
  { id: "ALL", label: "My work" },
  { id: "APPROVALS", label: "Approvals" },
] as const;

const sourceWorkStatuses = [
  { id: "Ready", label: "Ready" },
  { id: "In Progress", label: "In progress" },
  { id: "Blocked", label: "Blocked" },
  { id: "Escalated", label: "Escalated" },
];

const dueWindows: readonly { id: WorkDueWindow; label: string }[] = [
  { id: "ALL", label: "All due dates" },
  { id: "7", label: "Next 7 days + overdue" },
  { id: "30", label: "Next 30 days + overdue" },
  { id: "OVERDUE", label: "Overdue only" },
];

const workSignals = [
  { key: "OVERDUE", label: "Overdue", tone: "warning" },
  { key: "DUE_SOON", label: "Due next 7 days", tone: "neutral" },
  { key: "APPROVALS", label: "Your approvals", tone: "info" },
  { key: "EVIDENCE", label: "Evidence needed", tone: "neutral" },
] as const;

function workFilterLabel(filter: WorkQueueFilter): string {
  return ({ ALL: "My work", OVERDUE: "Overdue", DUE_SOON: "Due next 7 days", APPROVALS: "Approvals", EVIDENCE: "Evidence needed" })[filter];
}

const workColumns: readonly DataColumn<AttentionItem>[] = [
  { id: "work", header: "Work", mobileLayout: "full-width", render: (item) => <span className="assigned-workspace__record"><WorkItemIcon type={item.type}/><strong>{item.title}</strong></span>, accessibleText: (item) => item.title },
  { id: "status", header: "Status", kind: "status", render: (item) => <StatusBadge tone="neutral">{item.state || "Unknown"}</StatusBadge>, accessibleText: (item) => item.state || "Unknown" },
  { id: "due", header: "Due", render: (item) => formatDue(item.due_at), accessibleText: (item) => formatDue(item.due_at) },
  { id: "owner", header: "Owner", render: (item) => item.owner || "Unassigned", accessibleText: (item) => item.owner || "Unassigned" },
];

function WorkContext({ item, onOpen, onInspectAuthority }: { item: AttentionItem; onOpen: (item: AttentionItem) => void; onInspectAuthority?: (item: AttentionItem) => void }) {
  const target = item.action_target_type;
  const canOpen = Boolean(target && item.action_target_id);
  const canInspect = Boolean(canOpen && target !== "DOCUMENT_IMPORT" && item.authority && onInspectAuthority);
  return <aside className="assigned-workspace__context" aria-label="Selected work">
    <span className="eyebrow">{gateLabel(item.intervention_class, target)}</span>
    <h3>{item.title}</h3>
    <StatusBadge tone="neutral">{item.state || "Unknown"}</StatusBadge>
    <p>{item.material_conclusion || item.why_now}</p>
    <dl>
      <div><dt>Due</dt><dd>{formatDue(item.due_at)}</dd></div>
      <div><dt>Scope</dt><dd>{item.scope || "Not specified"}</dd></div>
      <div><dt>Owner</dt><dd>{item.owner || "Unassigned"}</dd></div>
      <div><dt>Evidence</dt><dd>{item.evidence || "Not recorded"}</dd></div>
    </dl>
    <div className="assigned-workspace__next"><span>{item.recommendation ? "Recommended action" : "Next action"}</span><strong>{item.recommendation?.proposed_action || item.primary_action}</strong></div>
    {item.verification && <VerificationContext item={item}/>}
    <div className="assigned-workspace__actions">
      {canOpen
        ? <Button variant="primary" onPress={() => openItem(item, onOpen)}>{openLabel(target, item.action_target_id)}</Button>
        : <p>No linked record is available.</p>}
      {canInspect && <Button variant="secondary" onPress={() => onInspectAuthority?.(item)}>Check authority</Button>}
    </div>
  </aside>;
}

function InterventionRow({ item, onOpen, onInspectAuthority }: { item: AttentionItem; onOpen: (item: AttentionItem) => void; onInspectAuthority?: (item: AttentionItem) => void }) {
  const due = formatDue(item.due_at);
  const nextAction = item.recommendation?.proposed_action || item.primary_action;
  const nextActionLabel = item.recommendation ? "Recommended action" : "Next action";
  const conclusion = item.material_conclusion || item.why_now;
  const targetType = item.action_target_type as string | undefined;
  const canOpen = Boolean(targetType && item.action_target_id);
  const canInspectAuthority = Boolean(canOpen && targetType !== "DOCUMENT_IMPORT" && item.authority && onInspectAuthority);
  return <article className="intervention-row">
    <div className="intervention-main">
      <div className="intervention-kicker"><span className="intervention-kind"><WorkItemIcon type={item.type}/>{gateLabel(item.intervention_class, targetType)}</span><span>{item.state}</span><time>{due}</time></div>
      <h3>{item.title}</h3>
      <p className="intervention-conclusion">{conclusion}</p>
      {item.change_summary && item.change_summary !== conclusion && <p className="intervention-change"><strong>Changed:</strong> {item.change_summary}</p>}
      <div className="intervention-meta"><span>{item.scope}</span><span>{item.evidence}</span><span>{item.owner}</span></div>
    </div>
    <div className="intervention-next">
      <span>{nextActionLabel}</span>
      <strong>{nextAction}</strong>
      {item.recommendation?.rationale && item.recommendation.rationale !== conclusion && <small>{item.recommendation.rationale}</small>}
      {item.verification && <VerificationContext item={item}/>}
      {canOpen ? <button className="primary-button" type="button" onClick={() => openItem(item, onOpen)}>{openLabel(targetType, item.action_target_id)}</button> : <small>No linked record is available.</small>}
      {canInspectAuthority && <button className="text-button" type="button" onClick={() => onInspectAuthority?.(item)}>Check authority</button>}
    </div>
  </article>;
}

function openItem(item: AttentionItem, fallback: (item: AttentionItem) => void) {
  const focused = item as FocusedAttentionItem;
  if ((focused.action_target_type as string | undefined) === "DOCUMENT_IMPORT" && focused.action_target_id) {
    const proposal = focused.action_target_sub_id ? `/${encodeURIComponent(focused.action_target_sub_id)}` : "";
    window.location.hash = `imports/${encodeURIComponent(focused.action_target_id)}${proposal}`;
    return;
  }
  if (focused.action_target_type === "CONFIGURE" && focused.action_target_id) {
    window.location.hash = `configure/${encodeURIComponent(focused.action_target_id)}`;
    return;
  }
  fallback(item);
}

function VerificationContext({ item }: { item: AttentionItem }) {
  const verification = item.verification;
  if (!verification) return null;
  const timing = formatCheckTime(verification.next_check_at);
  return <details className="intervention-verification">
    <summary>Outcome check details</summary>
    <dl>
      <div><dt>Expected outcome</dt><dd>{verification.expected_outcome || "Not provided"}</dd></div>
      <div><dt>Method</dt><dd>{verification.method || "Outcome review"}</dd></div>
      <div><dt>Check</dt><dd>{timing}</dd></div>
    </dl>
  </details>;
}

function openLabel(target?: string, targetID?: string) {
  if (target === "PROGRAM") return "Open program";
  if (target === "MATTER") return "Open issue";
  if (target === "EVIDENCE_REQUEST") return "Open request";
  if (target === "DOCUMENT_IMPORT") return "Open proposal";
  if (target === "CONFIGURE") return targetID === "operations" ? "Open system operations" : "Open access settings";
  return "Open item";
}

function gateLabel(value?: InterventionClass, target?: string) {
  switch (value) {
    case "DECISION": return "Decision";
    case "AUTHORIZATION": return "Approval";
    case "EVIDENCE_EXCEPTION": return "Evidence needed";
    case "ESCALATION": return "Escalated";
    case "VERIFICATION": return "Outcome check";
    case "EXTERNAL_REPRESENTATION": return "External response";
    case "REVIEW": return "Review";
    default: return target === "EVIDENCE_REQUEST" ? "Evidence request" : "Review";
  }
}

function formatDue(value: string) {
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) return "No deadline";
  const date = new Date(parsed);
  if (date.getUTCFullYear() < 2000) return "No deadline";
  return new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }).format(date);
}

function formatCheckTime(value?: string) {
  const parsed = Date.parse(value ?? "");
  if (!Number.isFinite(parsed) || new Date(parsed).getUTCFullYear() < 2000) return "Not scheduled";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(parsed));
}
