import { useCallback, useEffect, useState } from "react";
import { loadAutomationPolicies } from "../../api";
import type { AutomationPolicy } from "../../types";
import { AutomationPolicies } from "../AutomationPolicies";

type LoadState = "loading" | "live" | "unavailable";

export function AutomationSection() {
  const [policies, setPolicies] = useState<AutomationPolicy[]>([]);
  const [state, setState] = useState<LoadState>("loading");

  const load = useCallback(async () => {
    setState("loading");
    try {
      setPolicies(await loadAutomationPolicies());
      setState("live");
    } catch {
      setPolicies([]);
      setState("unavailable");
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  return <section className="configure-domain" aria-labelledby="automation-heading">
    <header className="configure-domain-header">
      <div><span className="eyebrow">Configuration · governed execution</span><h2 id="automation-heading">Automation</h2><p>Review approved automation guardrails without mixing them with AI model and workload governance.</p></div>
      {state === "unavailable" && <button className="secondary-button" type="button" onClick={() => void load()}>Retry automation data</button>}
    </header>
    <AutomationPolicies policies={policies} state={state}/>
  </section>;
}
