import { useCallback, useEffect, useState } from "react";
import { loadAIGovernancePolicies, loadAIGovernanceWorkloads, loadAutomationPolicies } from "../../api";
import type { AIGovernancePolicy, AIGovernanceWorkload, AutomationPolicy } from "../../types";
import { AIGovernancePanel } from "../AIGovernancePanel";
import { AutomationPolicies } from "../AutomationPolicies";

type LoadState = "loading" | "live" | "unavailable";

export function AutomationAISection() {
  const [automationPolicies, setAutomationPolicies] = useState<AutomationPolicy[]>([]);
  const [automationState, setAutomationState] = useState<LoadState>("loading");
  const [aiPolicies, setAIPolicies] = useState<AIGovernancePolicy[]>([]);
  const [aiPolicyState, setAIPolicyState] = useState<LoadState>("loading");
  const [aiWorkloads, setAIWorkloads] = useState<AIGovernanceWorkload[]>([]);
  const [aiWorkloadState, setAIWorkloadState] = useState<LoadState>("loading");

  const load = useCallback(async () => {
    setAutomationState("loading");
    setAIPolicyState("loading");
    setAIWorkloadState("loading");
    const [automationResult, policyResult, workloadResult] = await Promise.allSettled([
      loadAutomationPolicies(), loadAIGovernancePolicies(), loadAIGovernanceWorkloads(),
    ]);
    if (automationResult.status === "fulfilled") { setAutomationPolicies(automationResult.value); setAutomationState("live"); }
    else { setAutomationPolicies([]); setAutomationState("unavailable"); }
    if (policyResult.status === "fulfilled") { setAIPolicies(policyResult.value); setAIPolicyState("live"); }
    else { setAIPolicies([]); setAIPolicyState("unavailable"); }
    if (workloadResult.status === "fulfilled") { setAIWorkloads(workloadResult.value); setAIWorkloadState("live"); }
    else { setAIWorkloads([]); setAIWorkloadState("unavailable"); }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const degraded = [automationState, aiPolicyState, aiWorkloadState].some((state) => state === "unavailable");
  return <section className="configure-domain" aria-labelledby="automation-ai-heading">
    <header className="configure-domain-header">
      <div><span className="eyebrow">Configuration · governed execution</span><h2 id="automation-ai-heading">Automation & AI</h2><p>Review approved automation guardrails and governed AI workload/policy state. Human approvals remain in Today and Work.</p></div>
      {degraded && <button className="secondary-button" type="button" onClick={() => void load()}>Retry unavailable data</button>}
    </header>
    <AutomationPolicies policies={automationPolicies} state={automationState}/>
    <AIGovernancePanel policies={aiPolicies} policyState={aiPolicyState} workloads={aiWorkloads} workloadState={aiWorkloadState}/>
  </section>;
}
