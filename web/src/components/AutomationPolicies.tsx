import { useEffect, useRef } from "react";
import type { AutomationPolicyPage } from "../types";
import type { AutomationRoute } from "./configure/automationRouting";
import "../automation-policies.css";
import { Button, MetricCard, SearchField, SelectableRecord, SelectField, StatusBadge, type StatusTone } from "./ui";

export type AutomationLoadState = "loading" | "live" | "unavailable" | "denied";
type Props = {
  page?: AutomationPolicyPage;
  state: AutomationLoadState;
  route: AutomationRoute;
  onRouteChange: (route: AutomationRoute) => void;
  onRefresh?: () => void;
};

export function AutomationPolicies({ page, state, route, onRouteChange, onRefresh }: Props) {
  const records = page?.items ?? [];
  const query = route.q.trim().toLocaleLowerCase();
  const filtered = records.filter((policy) => (!route.status || policy.status === route.status) && (!route.action || policy.action_class === route.action)
    && (!query || `${policy.name} ${policy.code} ${humanize(policy.action_class)}`.toLocaleLowerCase().includes(query)));
  const selected = route.policy ? filtered.find((policy) => policy.id === route.policy) : filtered[0];
  const context = useRef<HTMLElement>(null);
  const list = useRef<HTMLElement>(null);
  const focusSelection = useRef<string | undefined>(undefined);
  const returnSelection = useRef<string | undefined>(undefined);
  const hasFilters = Boolean(route.q || route.action || route.status);

  useEffect(() => {
    if (focusSelection.current && focusSelection.current === selected?.id) {
      focusSelection.current = undefined;
      if (window.matchMedia("(max-width: 1100px)").matches) {
        context.current?.focus({ preventScroll: true });
        context.current?.scrollIntoView({ block: "start", behavior: "instant" });
      }
    }
  }, [selected?.id, route.policy]);

  function change(filters: Partial<AutomationRoute>) { onRouteChange({ ...route, ...filters, policy: undefined }); }
  function select(id: string) {
    returnSelection.current = id;
    if (selected?.id === id && route.policy === id) {
      if (window.matchMedia("(max-width: 1100px)").matches) { context.current?.focus({ preventScroll: true }); context.current?.scrollIntoView({ block: "start", behavior: "instant" }); }
    } else { focusSelection.current = id; onRouteChange({ ...route, policy: id }); }
  }
  function backToPolicies() {
    const row = Array.from(list.current?.querySelectorAll<HTMLButtonElement>("button[data-policy-id]") ?? []).find((node) => node.dataset.policyId === (returnSelection.current ?? selected?.id));
    const target = row ?? list.current;
    target?.focus({ preventScroll: true });
    target?.scrollIntoView({ block: "center", behavior: "instant" });
  }
  const options = (values: string[], current?: string) => [...new Set([...values, ...(current ? [current] : [])])].sort().map((id) => ({ id, label: humanize(id) }));

  return <div className="automation-policies">
    <div className="automation-workspace__metrics" role="group" aria-label="Loaded policy revision counts">
      {[
        { label: "Policy revisions", status: undefined }, { label: "Active", status: "ACTIVE" },
        { label: "Draft", status: "DRAFT" }, { label: "Suspended", status: "SUSPENDED" },
      ].map(({ label, status }) => <MetricCard key={label} label={label} value={page ? status ? records.filter((record) => record.status === status).length : records.length : "—"}
        quality={page ? page.truncated ? "partial" : "current" : "unknown"} qualityLabel={page ? "Loaded" : "Unavailable"}
        isDisabled={state !== "live"} isSelected={!route.q && !route.action && route.status === status}
        onPress={() => change({ q: "", action: undefined, status })} ariaControls="automation-policy-register"/>) }
    </div>
    {page && <div className="automation-workspace__population">
      <p>{page.returned_count} latest policy revisions loaded. A newer draft can replace an active revision in this list.</p>
      {page.truncated && <p role="status">Only the first {page.limit} policy codes are shown. Search, filters and counts cover these loaded revisions.</p>}
    </div>}
    <div className="automation-workspace__layout">
      <section className="automation-workspace__register" id="automation-policy-register" ref={list} tabIndex={-1} aria-label="Automation policies">
        <div className="automation-workspace__list-heading"><h2>Automation policies</h2>{page && <span>{filtered.length} of {records.length} loaded</span>}{hasFilters && <Button variant="quiet" onPress={() => change({ q: "", status: undefined, action: undefined })}>Clear filters</Button>}</div>
        <div className="automation-workspace__filters">
          <SearchField label="Search loaded policies" placeholder="Search loaded policies" value={route.q} onChange={(q) => change({ q: q.slice(0, 256) })} isLoading={state === "loading"}/>
          <SelectField label="Action class" isLabelHidden placeholder="All actions" value={route.action} options={options(records.map((policy) => policy.action_class), route.action)} onChange={(action) => change({ action })}/>
          <SelectField label="Status" isLabelHidden placeholder="All statuses" value={route.status} options={options(records.map((policy) => policy.status), route.status)} onChange={(status) => change({ status })}/>
        </div>
        {state === "loading" ? <p className="automation-workspace__message" role="status" aria-busy="true">Loading automation policies…</p>
          : state === "denied" ? <div className="automation-workspace__message" role="status"><h3>Automation policies are restricted</h3><p>Configuration read access is required to review these policies.</p></div>
            : state === "unavailable" ? <div className="automation-workspace__message" role="alert"><h3>Automation policies could not be loaded</h3><p>Refresh the policy list to try again.</p>{onRefresh && <Button variant="secondary" onPress={onRefresh}>Retry automation data</Button>}</div>
              : !records.length ? <div className="automation-workspace__message"><h3>No policy revisions returned</h3><p>No latest policy revisions were returned for this organization.</p></div>
                : !filtered.length ? <div className="automation-workspace__message"><h3>No loaded policies match</h3><p>Change the search or clear the filters to review the loaded revisions.</p></div>
                  : <div className="automation-policy-list">{filtered.map((policy) => <SelectableRecord key={policy.id} variant="list" title={policy.name} metadata={`${humanize(policy.action_class)} · ${policy.code} · Revision ${policy.version}`}
                    endContent={<StatusBadge tone={policyTone(policy.status)}>{humanize(policy.status)}</StatusBadge>} isSelected={selected?.id === policy.id} aria-label={`Inspect ${policy.name}`} data-policy-id={policy.id} onPress={() => select(policy.id)}/>)}</div>}
      </section>
      <aside className="automation-workspace__context" aria-label="Selected automation policy" ref={context} tabIndex={-1}>
        {state !== "live" ? <div className="automation-workspace__message"><h2>Policy details</h2><p>{state === "loading" ? "Policy details will appear when the list loads." : "Policy details are unavailable until the list can be reviewed."}</p></div>
          : selected ? <>
            <div className="automation-workspace__context-heading"><span className="eyebrow">Selected policy</span><StatusBadge tone={policyTone(selected.status)}>{humanize(selected.status)}</StatusBadge><h2>{selected.name}</h2><p>{selected.code} · Revision {selected.version}</p></div>
            <div className="automation-workspace__context-return"><Button variant="secondary" onPress={backToPolicies}>Back to policies</Button></div>
            <div className="automation-guardrails">
              <GuardrailGroup title="Eligible when" value={selected.eligibility}/>
              <GuardrailGroup title="Maximum scope" value={selected.blast_radius_limit}/>
              <GuardrailGroup title="Outcome check" value={selected.verification_contract}/>
            </div>
            <dl className="automation-workspace__facts">
              <Fact label="Action class" value={humanize(selected.action_class)}/>
              <Fact label="Effective from" value={formatDate(selected.effective_from)}/>
              <Fact label="Effective until" value={formatDate(selected.effective_until)}/>
              <Fact label="Last updated" value={formatDate(selected.updated_at)}/>
              {selected.rollout_mode && <Fact label="Rollout mode" value={humanize(selected.rollout_mode)}/>}
            </dl>
            <p className="automation-workspace__context-note">Policy editing, simulation and execution history are unavailable here.</p>
          </> : <div className="automation-workspace__message"><h2>{route.policy ? "Selected policy is not in this list" : "Policy details"}</h2><p>{route.policy ? "The selected revision may be outside the loaded policies or filters. Choose a policy from the list." : "Choose a policy to review its revision and action limits."}</p>{route.policy && <Button variant="secondary" onPress={() => onRouteChange({ ...route, policy: undefined })}>Return to loaded policies</Button>}</div>}
      </aside>
    </div>
  </div>;
}

function Fact({ label, value }: { label: string; value: string }) { return <div><dt>{label}</dt><dd>{value}</dd></div>; }
function policyTone(status: string): StatusTone {
  if (status === "ACTIVE") return "success";
  if (status === "SUSPENDED" || status === "EXPIRED") return "warning";
  return "neutral";
}
function GuardrailGroup({ title, value }: { title: string; value: Record<string, unknown> }) {
  const entries = Object.entries(value ?? {});
  return <section><h3>{title}</h3>{entries.length ? <dl>{entries.map(([key, entry]) => <Fact key={key} label={humanize(key)} value={formatValue(entry)}/>)}</dl> : <p>Not recorded.</p>}</section>;
}
function formatValue(value: unknown): string {
  if (typeof value === "boolean") return value ? "Yes" : "No";
  if (Array.isArray(value)) return value.length ? value.map(formatValue).join(", ") : "Not recorded";
  if (value && typeof value === "object") return Object.entries(value as Record<string, unknown>).map(([key, entry]) => `${humanize(key)}: ${formatValue(entry)}`).join(" · ") || "Not recorded";
  return value === null || value === undefined || value === "" ? "Not recorded" : String(value);
}
function formatDate(value?: string) {
  if (!value) return "Not recorded";
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(parsed)) : "Date unavailable";
}
function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }
