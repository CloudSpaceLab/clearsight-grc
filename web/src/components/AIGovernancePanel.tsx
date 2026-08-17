import type { AIGovernancePolicy, AIGovernanceWorkload } from "../types";
import { EmptyState } from "./EmptyState";

type LoadState = "loading" | "live" | "unavailable";

type Props = {
  policies: AIGovernancePolicy[];
  policyState: LoadState;
  workloads: AIGovernanceWorkload[];
  workloadState: LoadState;
};

export function AIGovernancePanel({ policies, policyState, workloads, workloadState }: Props) {
  const activePolicies = policies.filter((policy) => policy.status === "ACTIVE");
  const enforcing = activePolicies.filter((policy) => policy.rollout_mode === "ENFORCE").length;
  const shadow = activePolicies.filter((policy) => policy.rollout_mode === "SHADOW").length;
  const activeWorkloads = workloads.filter((workload) => workload.state === "ACTIVE");

  return <section className="ai-governance" aria-labelledby="ai-governance-heading">
    <header className="section-header ai-governance-header">
      <div>
        <span className="eyebrow">AI governance</span>
        <h2 id="ai-governance-heading">Governed model access</h2>
        <p>Registered workloads, exact policy revisions and rollout state. Approval work stays in Matters and Today.</p>
      </div>
      {policyState === "live" && workloadState === "live" && <div className="ai-governance-summary" aria-label="AI governance summary">
        <span><strong>{activeWorkloads.length}</strong> active workloads</span>
        <span><strong>{shadow}</strong> shadow</span>
        <span><strong>{enforcing}</strong> enforcing</span>
      </div>}
    </header>

    <div className="ai-governance-grid">
      <article className="ai-governance-card">
        <div className="ai-governance-card-title"><div><h3>Policy revisions</h3><p>Shadow precedes enforcement; maker and checker remain distinct.</p></div></div>
        <PolicyList policies={policies} state={policyState}/>
      </article>
      <article className="ai-governance-card">
        <div className="ai-governance-card-title"><div><h3>Workloads</h3><p>Server-verified identity, approved models and bounded request budgets.</p></div></div>
        <WorkloadList workloads={workloads} state={workloadState}/>
      </article>
    </div>

    <p className="ai-governance-note">Routine allowed traffic does not create Today work. A governed action that requires approval must be backed by an approved Matter decision, and its execution grant is bound to the exact action and can be used once.</p>
  </section>;
}

function PolicyList({ policies, state }: { policies: AIGovernancePolicy[]; state: LoadState }) {
  if (state === "loading") return <div className="workspace-loading compact" aria-live="polite" aria-busy="true">Loading AI policies…</div>;
  if (state === "unavailable") return <EmptyState kind="unavailable" label="AI policies" title="AI policies are unavailable" description="Try again before changing or relying on an AI enforcement rollout."/>;
  if (!policies.length) return <EmptyState label="AI policies" title="No AI policies in this scope" description="No governed model policy has been registered for the current bank scope."/>;
  return <div className="ai-governance-list">{policies.map((policy) => <div className="ai-governance-row" key={policy.id}>
    <div><strong>{policy.name}</strong><span>{policy.code} · v{policy.version}</span></div>
    <div className="ai-governance-badges"><mark className={`ai-rollout-${policy.rollout_mode.toLowerCase()}`}>{humanize(policy.rollout_mode)}</mark><mark>{humanize(policy.status)}</mark></div>
  </div>)}</div>;
}

function WorkloadList({ workloads, state }: { workloads: AIGovernanceWorkload[]; state: LoadState }) {
  if (state === "loading") return <div className="workspace-loading compact" aria-live="polite" aria-busy="true">Loading AI workloads…</div>;
  if (state === "unavailable") return <EmptyState kind="unavailable" label="AI workloads" title="AI workloads are unavailable" description="Try again before relying on the registered model allow-list or request budgets."/>;
  if (!workloads.length) return <EmptyState label="AI workloads" title="No AI workloads in this scope" description="No workload or agent is registered for governed model access."/>;
  return <div className="ai-governance-list">{workloads.map((workload) => <div className="ai-governance-row ai-workload-row" key={workload.id}>
    <div><strong>{workload.name}</strong><span>{workload.environment || "Environment not set"} · {workload.purpose}</span><small>{workload.allowed_models.join(", ")} · {workload.requests_per_minute}/min · {workload.max_concurrent} concurrent</small></div>
    <div className="ai-governance-badges"><mark>{humanize(workload.state)}</mark><span>v{workload.version}</span></div>
  </div>)}</div>;
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
