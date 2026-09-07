import { useCallback, useEffect, useMemo, useState } from "react";
import type { AIGovernanceWorkload } from "../../types";
import { loadGatewayBaselines, type GatewayBaselinePolicy } from "../../aiGovernanceControlApi";
import {
  loadGatewayTransportState,
  simulateGateway,
  type GatewayEnvironment,
  type GatewaySimulationFact,
  type GatewaySimulationFixture,
  type GatewaySimulationResult,
  type GatewayTransportRevision,
} from "../../aiGatewayTransportApi";
import "./AIGatewaySimulationPanel.css";

type LoadState = "loading" | "live" | "unavailable";

type FixtureOption = {
  value: GatewaySimulationFixture;
  label: string;
  note: string;
};

const environments: GatewayEnvironment[] = ["PRODUCTION", "TEST", "DEVELOPMENT"];
const fixtures: FixtureOption[] = [
  { value: "SAFE", label: "Known safe request", note: "Baseline and workload rules should permit a compliant request." },
  { value: "INSTRUCTION_EXFILTRATION", label: "Instruction exfiltration", note: "Attempts to reveal hidden system/developer instructions." },
  { value: "HOSTILE_UNTRUSTED_CONTENT", label: "Hostile retrieved content", note: "Untrusted retrieved text carries an instruction-boundary attack." },
  { value: "UNAVAILABLE_PROVIDER", label: "Provider unavailable", note: "Policy may allow the request while routing fails closed operationally." },
  { value: "FORBIDDEN_RESIDENCY_FALLBACK", label: "Forbidden residency fallback", note: "No fallback may cross a simulated residency boundary." },
  { value: "UNKNOWN_WORKLOAD", label: "Unknown workload", note: "Unregistered production callers remain outside governed authority." },
];

export function AIGatewaySimulationPanel({ workloads, workloadState }: { workloads: AIGovernanceWorkload[]; workloadState: LoadState }) {
  const [environment, setEnvironment] = useState<GatewayEnvironment>("PRODUCTION");
  const [fixture, setFixture] = useState<GatewaySimulationFixture>("SAFE");
  const [baselines, setBaselines] = useState<GatewayBaselinePolicy[]>([]);
  const [transports, setTransports] = useState<GatewayTransportRevision[]>([]);
  const [workloadId, setWorkloadId] = useState("");
  const [modelAlias, setModelAlias] = useState("");
  const [loading, setLoading] = useState(true);
  const [running, setRunning] = useState(false);
  const [message, setMessage] = useState("");
  const [result, setResult] = useState<GatewaySimulationResult | null>(null);

  const loadCandidates = useCallback(async () => {
    setLoading(true);
    setMessage("");
    try {
      const [baselineItems, transportState] = await Promise.all([loadGatewayBaselines(), loadGatewayTransportState(environment)]);
      setBaselines(baselineItems);
      setTransports(transportState.revisions);
    } catch (error) {
      setBaselines([]);
      setTransports([]);
      setMessage(error instanceof Error ? error.message : "Simulation candidates could not be loaded.");
    } finally {
      setLoading(false);
    }
  }, [environment]);

  useEffect(() => { void loadCandidates(); }, [loadCandidates]);

  const workloadCandidates = useMemo(
    () => workloads.filter((workload) => workload.environment.toUpperCase() === environment),
    [environment, workloads],
  );
  const selectedWorkload = workloadCandidates.find((workload) => workload.id === workloadId);
  const baseline = baselines[0];
  const transport = transports[0];
  const availableAliases = selectedWorkload?.allowed_models ?? transport?.definition.models.map((model) => model.alias) ?? [];
  const selectedAlias = modelAlias && availableAliases.includes(modelAlias) ? modelAlias : availableAliases[0] ?? "";
  const requiresWorkload = fixture !== "UNKNOWN_WORKLOAD";

  useEffect(() => {
    if (fixture === "UNKNOWN_WORKLOAD") return;
    if (workloadId && workloadCandidates.some((workload) => workload.id === workloadId)) return;
    setWorkloadId(workloadCandidates.find((workload) => workload.state === "ACTIVE")?.id ?? workloadCandidates[0]?.id ?? "");
  }, [fixture, workloadCandidates, workloadId]);

  useEffect(() => {
    if (modelAlias && availableAliases.includes(modelAlias)) return;
    setModelAlias(availableAliases[0] ?? "");
  }, [availableAliases, modelAlias]);

  async function runSimulation() {
    if (running || loading || (requiresWorkload && !workloadId)) return;
    setRunning(true);
    setMessage("");
    setResult(null);
    try {
      setResult(await simulateGateway({
        environment,
        fixture,
        workloadId: requiresWorkload ? workloadId : undefined,
        baselinePolicyId: baseline?.id,
        transportId: transport?.id,
        modelAlias: selectedAlias || undefined,
      }));
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Gateway simulation could not be completed.");
    } finally {
      setRunning(false);
    }
  }

  return <article className="configure-context-panel ai-gateway-simulation" aria-labelledby="gateway-simulation-heading">
    <div className="configure-subheader ai-gateway-simulation__header">
      <div>
        <span className="eyebrow">Gateway · deterministic validation</span>
        <h3 id="gateway-simulation-heading">Test policy before traffic moves</h3>
        <p>Run bounded fixtures through the exact candidate baseline, workload policy and routing revision. Simulation makes no model call, stores no fixture content and creates no normal decision receipt.</p>
      </div>
      <span className="ai-gateway-simulation__state">{running ? "Evaluating…" : result ? title(result.decision.action) : "No provider call"}</span>
    </div>

    <div className="ai-gateway-simulation__controls">
      <label><span>Environment</span><select value={environment} onChange={(event) => { setEnvironment(event.target.value as GatewayEnvironment); setResult(null); }}>{environments.map((value) => <option key={value} value={value}>{title(value)}</option>)}</select></label>
      <label><span>Fixture</span><select value={fixture} onChange={(event) => { setFixture(event.target.value as GatewaySimulationFixture); setResult(null); }}>{fixtures.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select><small>{fixtures.find((option) => option.value === fixture)?.note}</small></label>
      {requiresWorkload && <label><span>Workload</span><select value={workloadId} disabled={workloadState !== "live" || workloadCandidates.length === 0} onChange={(event) => { setWorkloadId(event.target.value); setResult(null); }}><option value="">Select workload</option>{workloadCandidates.map((workload) => <option key={workload.id} value={workload.id}>{workload.name} · {title(workload.state)}</option>)}</select><small>{workloadState === "unavailable" ? "Workload inventory is unavailable." : `${workloadCandidates.length} workload${workloadCandidates.length === 1 ? "" : "s"} in this environment.`}</small></label>}
      {requiresWorkload && <label><span>Logical model</span><select value={selectedAlias} disabled={availableAliases.length === 0} onChange={(event) => { setModelAlias(event.target.value); setResult(null); }}><option value="">Auto-select</option>{availableAliases.map((alias) => <option key={alias} value={alias}>{alias}</option>)}</select><small>Logical alias only; raw provider model selection is never exposed.</small></label>}
    </div>

    <div className="ai-gateway-simulation__candidate" aria-label="Candidate revisions">
      <Candidate label="Organization baseline" value={baseline ? `v${baseline.version} · ${title(baseline.status)} · ${title(baseline.rollout_mode)}` : "No baseline candidate"}/>
      <Candidate label="Transport revision" value={transport ? `v${transport.version} · ${title(transport.status)}` : "No transport candidate"}/>
      <Candidate label="Policy source" value={selectedWorkload ? `${selectedWorkload.code} · policy v${selectedWorkload.policy_version}` : fixture === "UNKNOWN_WORKLOAD" ? "Unregistered caller" : "No workload selected"}/>
    </div>

    <div className="ai-gateway-simulation__actions">
      <button className="primary-button" type="button" onClick={() => void runSimulation()} disabled={running || loading || (requiresWorkload && !workloadId)}>{running ? "Running simulation…" : "Run deterministic test"}</button>
      <span>Fixture text is built into the server test harness and is never returned to the browser.</span>
    </div>

    {result && <SimulationResult result={result}/>} 
    {message && <p className="ai-gateway-simulation__message" aria-live="polite">{message}</p>}
  </article>;
}

function Candidate({ label, value }: { label: string; value: string }) {
  return <div><span>{label}</span><strong>{value}</strong></div>;
}

function SimulationResult({ result }: { result: GatewaySimulationResult }) {
  const proposed = result.decision.proposed_action || result.decision.baseline_proposed_action;
  return <section className="ai-gateway-simulation__result" aria-label="Gateway simulation result">
    <div className="ai-gateway-simulation__verdict">
      <div><span>Decision</span><strong>{title(result.decision.action)}{proposed ? ` → ${title(proposed)}` : ""}</strong><small>{(result.decision.reason_codes ?? []).join(" · ") || "No material policy reason"}</small></div>
      <div><span>Provider call</span><strong>{result.provider_call_would_occur ? "Would proceed" : "Would not occur"}</strong><small>{result.provider_call_blocked_reason ? title(result.provider_call_blocked_reason) : `${result.eligible_routes.length} eligible route${result.eligible_routes.length === 1 ? "" : "s"}`}</small></div>
      <div><span>Exact baseline</span><strong>{result.baseline_policy ? `v${result.baseline_policy.version} · ${title(result.baseline_policy.rollout_mode)}` : "None"}</strong><small>{result.baseline_policy?.code ?? "No tenant baseline selected"}</small></div>
      <div><span>Exact transport</span><strong>{result.transport ? `v${result.transport.version} · ${title(result.transport.status)}` : "None"}</strong><small>{result.transport?.checksum ? `sha256 · ${result.transport.checksum.slice(0, 10)}…` : "No route snapshot selected"}</small></div>
    </div>

    <div className="ai-gateway-simulation__detail-grid">
      <section>
        <span className="eyebrow">Detector facts</span>
        <FactList facts={result.detector_facts}/>
      </section>
      <section>
        <span className="eyebrow">Eligible routes</span>
        {result.eligible_routes.length ? <ul>{result.eligible_routes.map((route) => <li key={route.id}><strong>{route.id}</strong><span>{route.provider_id} · {route.model}{route.regions?.length ? ` · ${route.regions.join(", ")}` : ""}</span></li>)}</ul> : <p>No route is eligible for this fixture.</p>}
      </section>
    </div>

    <section className="ai-gateway-simulation__instructions">
      <div><span className="eyebrow">Effective instruction stack</span><p>{result.instruction_precedence.map(title).join(" → ")}</p></div>
      {result.organization_instructions.length ? <div className="ai-gateway-simulation__instruction-list">{result.organization_instructions.map((instruction) => <div key={`${instruction.rule_id}:${instruction.reason_code}`}><span>{instruction.applied ? "Applied" : instruction.matched ? "Preview only" : "Not matched"}</span><strong>{instruction.content}</strong><small>{instruction.rule_id} · {instruction.reason_code}</small></div>)}</div> : <p>No organization instruction overlay is present in the selected baseline revision.</p>}
      <small>Workload system/developer and user/retrieved content are represented only as precedence boundaries; their text is not returned by simulation.</small>
    </section>
  </section>;
}

function FactList({ facts }: { facts: GatewaySimulationFact[] }) {
  if (!facts.length) return <p>No detector facts were produced.</p>;
  return <ul>{facts.map((fact) => <li key={fact.key}><strong>{title(fact.key.replace("gateway.", ""))}</strong><span>{fact.value ?? title(fact.state)}</span></li>)}</ul>;
}

function title(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replaceAll("-", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
