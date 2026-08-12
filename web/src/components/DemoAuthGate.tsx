import { useEffect, useState, type ReactNode } from "react";
import { loadContext, loadDemoAccounts, logoutDemo, type DemoAccount, type RuntimeContext } from "../api";
import { apiErrorKind } from "../http";
import { DemoLoginPage } from "./DemoLoginPage";

type GateState = "checking" | "ready" | "login";
type DemoRuntime = RuntimeContext & { demo_mode?: boolean };

export function DemoAuthGate({ children }: { children: ReactNode }) {
  const [state, setState] = useState<GateState>("checking");
  const [accounts, setAccounts] = useState<DemoAccount[]>([]);
  const [demoMode, setDemoMode] = useState(false);
  const [switchError, setSwitchError] = useState("");

  async function enter() {
    setState("checking");
    setSwitchError("");
    try {
      const context = await loadContext() as DemoRuntime;
      setDemoMode(context.demo_mode === true);
      setState("ready");
      return;
    } catch (error) {
      if (apiErrorKind(error) !== "unauthorized") {
        // Preserve the existing application's degraded-state handling when the
        // problem is not an unauthenticated demo session.
        setDemoMode(false);
        setState("ready");
        return;
      }
    }

    const available = await loadDemoAccounts().catch(() => []);
    if (available.length) {
      setAccounts(available);
      setDemoMode(true);
      setState("login");
      return;
    }
    // Preserve the existing application's degraded-state handling.
    // Production OIDC/signed deployments do not expose the demo catalogue;
    // defer to the existing application behavior there.
    setDemoMode(false);
    setState("ready");
  }

  useEffect(() => { void enter(); }, []);

  async function switchRole() {
    setSwitchError("");
    try {
      await logoutDemo();
      const available = await loadDemoAccounts();
      setAccounts(available);
      setState("login");
    } catch (error) {
      setSwitchError(error instanceof Error ? error.message : "Demo role could not be changed.");
    }
  }

  if (state === "checking") {
    return <main className="demo-login-shell"><section className="demo-login-panel compact" aria-live="polite" aria-busy="true"><span className="eyebrow">ClearSight</span><h1>Opening your workspace…</h1></section></main>;
  }
  if (state === "login") return <DemoLoginPage accounts={accounts} onAuthenticated={enter}/>;

  const canSwitchRole = demoMode && import.meta.env.VITE_STATIC_DEMO !== "true";
  return <>
    {children}
    {canSwitchRole && <div className="demo-role-switch-wrap">
      <button className="demo-role-switch" type="button" onClick={() => void switchRole()}>Switch demo role</button>
      {switchError && <span role="alert">{switchError}</span>}
    </div>}
  </>;
}
