import type { AutomationPolicy } from "../types";
import "../automation-policies.css";
import { EmptyState } from "./EmptyState";
import { StatusBadge, type StatusTone } from "./ui";

type LoadState = "loading" | "live" | "unavailable";

type Props = {
  policies: AutomationPolicy[];
  state: LoadState;
};

export function AutomationPolicies({ policies, state }: Props) {
  if (state === "loading") {
    return <section className="automation-policies workspace-loading" aria-live="polite" aria-busy="true">Loading automation policies…</section>;
  }
  if (state === "unavailable") {
    return <section className="automation-policies"><EmptyState label="Automation policy" title="Automation policies could not be loaded" description="Try again before reviewing or changing automated actions."/></section>;
  }

  return <section className="automation-policies" aria-labelledby="automation-policy-heading">
    <header className="section-header">
      <div><h2 id="automation-policy-heading">Automation policies</h2><p>Approved automated actions, eligibility rules, limits and outcome checks.</p></div>
    </header>
    {!policies.length
      ? <EmptyState label="Automation policy" title="No automation policies in this scope" description="No automated actions are approved for the current scope."/>
      : <div className="automation-policy-list">{policies.map((policy) => <PolicyRow key={policy.id} policy={policy}/>)}</div>}
    <p className="automation-policy-note">Only active policies can run actions. Review execution history for completed actions and outcome checks.</p>
  </section>;
}

function PolicyRow({ policy }: { policy: AutomationPolicy }) {
  return <article className="automation-policy-row">
    <div className="automation-policy-main">
      <div><span className="eyebrow">{humanize(policy.action_class)}</span><h3>{policy.name}</h3><p>{policy.code} · version {policy.version}</p></div>
      <div className="automation-policy-state"><StatusBadge tone={policyTone(policy.status)}>{humanize(policy.status)}</StatusBadge>{policy.effective_until && <span>Ends {formatDate(policy.effective_until)}</span>}</div>
    </div>
    <details>
      <summary>View limits</summary>
      <div className="automation-guardrails">
        <GuardrailGroup title="Eligible when" value={policy.eligibility}/>
        <GuardrailGroup title="Maximum scope" value={policy.blast_radius_limit}/>
        <GuardrailGroup title="Outcome check" value={policy.verification_contract}/>
      </div>
    </details>
  </article>;
}

function policyTone(status: AutomationPolicy["status"]): StatusTone {
  if (status === "ACTIVE") return "success";
  if (status === "SUSPENDED" || status === "EXPIRED") return "warning";
  return "neutral";
}

function GuardrailGroup({ title, value }: { title: string; value: Record<string, unknown> }) {
  const entries = Object.entries(value ?? {});
  return <section><h4>{title}</h4>{entries.length ? <dl>{entries.map(([key, entry]) => <div key={key}><dt>{humanize(key)}</dt><dd>{formatValue(entry)}</dd></div>)}</dl> : <p>No additional condition is recorded.</p>}</section>;
}

function formatValue(value: unknown): string {
  if (typeof value === "boolean") return value ? "Yes" : "No";
  if (typeof value === "string") return /^[A-Z0-9_]+$/.test(value) || value.includes("_") ? humanize(value) : value;
  if (Array.isArray(value)) return value.map(formatValue).join(", ");
  if (value && typeof value === "object") return Object.entries(value as Record<string, unknown>).map(([key, entry]) => `${humanize(key)}: ${formatValue(entry)}`).join(" · ");
  if (value === null || value === undefined || value === "") return "Not set";
  return String(value);
}

function formatDate(value: string) {
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(parsed)) : "date unavailable";
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
