import { useCallback, useEffect, useState } from "react";
import { loadAIGovernancePolicies, loadAIGovernanceWorkloads } from "../../api";
import type { AIGovernancePolicy, AIGovernanceWorkload } from "../../types";
import { AIGovernancePanel } from "../AIGovernancePanel";
import { AIGatewayControlPlane } from "./AIGatewayControlPlane";
import { AIGatewayTransportControl } from "./AIGatewayTransportControl";

type LoadState = "loading" | "live" | "unavailable";

export function AIGovernanceSection() {
  const [policies, setPolicies] = useState<AIGovernancePolicy[]>([]);
  const [policyState, setPolicyState] = useState<LoadState>("loading");
  const [workloads, setWorkloads] = useState<AIGovernanceWorkload[]>([]);
  const [workloadState, setWorkloadState] = useState<LoadState>("loading");

  const load = useCallback(async () => {
    setPolicyState("loading");
    setWorkloadState("loading");
    const [policyResult, workloadResult] = await Promise.allSettled([
      loadAIGovernancePolicies(),
      loadAIGovernanceWorkloads(),
    ]);
    if (policyResult.status === "fulfilled") {
      setPolicies(policyResult.value);
      setPolicyState("live");
    } else {
      setPolicies([]);
      setPolicyState("unavailable");
    }
    if (workloadResult.status === "fulfilled") {
      setWorkloads(workloadResult.value);
      setWorkloadState("live");
    } else {
      setWorkloads([]);
      setWorkloadState("unavailable");
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const degraded = policyState === "unavailable" || workloadState === "unavailable";
  return <section className="configure-domain" aria-labelledby="ai-governance-heading">
    <header className="configure-domain-header">
      <div><span className="eyebrow">Configuration · AI governance</span><h2 id="ai-governance-heading">AI governance</h2><p>Operate the organization AI proxy, configure non-bypassable gateway guardrails, and review governed workloads and policy rollout. Human decisions remain in Today and Work.</p></div>
      {degraded && <button className="secondary-button" type="button" onClick={() => void load()}>Retry unavailable AI data</button>}
    </header>
    <AIGatewayTransportControl/>
    <AIGatewayControlPlane onChanged={() => void load()}/>
    <AIGovernancePanel policies={policies} policyState={policyState} workloads={workloads} workloadState={workloadState}/>
  </section>;
}
