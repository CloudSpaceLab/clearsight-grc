import { useEffect, useState } from "react";
import { loadFormPolicyExecutionResult, type FormPolicyExecutionResult } from "../../formPoliciesApi";
import { ActionLink, Button, Notice } from "../ui";

export function PolicyExecutionResult({ policyID, executionID }: { policyID: string; executionID: string }) {
  const [result, setResult] = useState<FormPolicyExecutionResult>();
  const [state, setState] = useState<"loading" | "live" | "error">("loading");
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    let current = true;
    setResult(undefined); setState("loading");
    void loadFormPolicyExecutionResult(policyID, executionID).then((value) => { if (current) { setResult(value); setState("live"); } }, () => { if (current) setState("error"); });
    return () => { current = false; };
  }, [policyID, executionID, retry]);
  return <section aria-label="Policy result"><h3>Policy result</h3>
    {state === "loading" && <p role="status">Checking access to the issue and submitted response…</p>}
    {state === "error" && <><Notice tone="warning">The policy result could not be loaded. Retry to check this activity.</Notice><Button onPress={() => setRetry((value) => value + 1)}>Retry result</Button></>}
    {state === "live" && !result?.targets.length && <p>No issue or response is available to open under your current access.</p>}
    {state === "live" && result?.targets.map((target) => <div key={target.type + target.id}><p>{target.title}</p><ActionLink href={target.type === "MATTER" ? `#work/matters/${encodeURIComponent(target.id)}` : `#forms?section=responses&response=${encodeURIComponent(target.id)}`}>{target.type === "MATTER" ? "Open issue" : "Review response"}</ActionLink></div>)}
  </section>;
}
