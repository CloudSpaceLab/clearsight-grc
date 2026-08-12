import { useEffect, useState, type ReactNode } from "react";
import { loadContext, loadDemoAccounts, loadSessionStatus, loginDemo, logoutDemo, type DemoAccount, type RuntimeContext } from "../api";
import { apiErrorKind } from "../http";
import type { RuntimePresentation } from "../runtimePresentation";
import { DemoLoginPage } from "./DemoLoginPage";

type GateState = "checking" | "ready" | "login";
type DemoRuntime = RuntimeContext & { demo_mode?: boolean };

export function DemoAuthGate({ children, presentation = "demo" }: { children: ReactNode; presentation?: RuntimePresentation }) {
  const [state, setState] = useState<GateState>("checking");
  const [accounts, setAccounts] = useState<DemoAccount[]>([]);
  const [demoMode, setDemoMode] = useState(false);
  const [switchError, setSwitchError] = useState("");
  const [loginError, setLoginError] = useState("");
  const [accountMenuOpen, setAccountMenuOpen] = useState(false);
  const [currentAccountLabel, setCurrentAccountLabel] = useState("Demo account");

  async function rememberContext(context: DemoRuntime) {
    const isDemo = context.demo_mode === true;
    setDemoMode(isDemo);
    if (isDemo) {
      const available = await loadDemoAccounts().catch(() => []);
      setAccounts(available);
      const actorRoles = new Set(context.actor.role_codes ?? []);
      const matched = available.find((account) => account.role_codes.some((role) => actorRoles.has(role)));
      setCurrentAccountLabel(matched?.label ?? context.actor.name ?? "Demo account");
      return;
    }
    setCurrentAccountLabel(context.actor.name || "Signed-in account");
  }

  async function enter() {
    setState("checking");
    setSwitchError("");
    setLoginError("");
    let status;
    try {
      status = await loadSessionStatus();
    } catch {
      // Keep compatibility with deployments released before session discovery.
    }

    if (status && !status.authenticated && status.demo_login_available) {
      const available = await loadDemoAccounts().catch(() => []);
      setAccounts(available);
      setDemoMode(true);
      setState(available.length ? "login" : "ready");
      return;
    }
    if (status?.authenticated) {
      try {
        await rememberContext(await loadContext() as DemoRuntime);
      } catch (error) {
        if (apiErrorKind(error) === "unauthorized" && status.demo_login_available) {
          const available = await loadDemoAccounts().catch(() => []);
          setAccounts(available);
          setDemoMode(true);
          setState(available.length ? "login" : "ready");
          return;
        }
        setDemoMode(false);
      }
      setState("ready");
      return;
    }

    try {
      await rememberContext(await loadContext() as DemoRuntime);
      setState("ready");
      return;
    } catch (error) {
      if (apiErrorKind(error) !== "unauthorized") {
        setDemoMode(false);
        setState("ready");
        return;
      }
    }

    const available = await loadDemoAccounts().catch(() => []);
    if (!available.length) {
      setDemoMode(false);
      setState("ready");
      return;
    }
    setAccounts(available);
    setDemoMode(true);
    setState("login");
  }

  useEffect(() => { void enter(); }, []);

  async function switchRole(account: DemoAccount) {
    setSwitchError("");
    setAccountMenuOpen(false);
    try {
      await logoutDemo();
      setState("checking");
      await loginDemo(account.username, account.password);
      await enter();
    } catch (error) {
      setLoginError(error instanceof Error ? error.message : "Demo account could not be changed.");
      setState("login");
    }
  }

  if (state === "checking") {
    return <main className="demo-login-shell"><section className="demo-login-panel compact" aria-live="polite" aria-busy="true"><span className="eyebrow">ClearSight</span><h1>Opening your workspace…</h1></section></main>;
  }
  if (state === "login") return <DemoLoginPage accounts={accounts} onAuthenticated={enter} initialError={loginError}/>;

  const canSwitchRole = demoMode && presentation === "demo" && import.meta.env.VITE_STATIC_DEMO !== "true";
  return <>
    {children}
    {canSwitchRole && <div className="demo-account-menu-wrap">
      <button className="demo-account-menu-trigger" type="button" aria-expanded={accountMenuOpen} aria-controls="demo-account-menu" onClick={() => setAccountMenuOpen((open) => !open)}>Viewing as <strong>{currentAccountLabel}</strong></button>
      {accountMenuOpen && <div className="demo-account-menu" id="demo-account-menu" aria-label="Switch demo account">
        <span>Choose another account</span>
        {accounts.filter((account) => account.label !== currentAccountLabel).map((account) => <button key={account.username} type="button" onClick={() => void switchRole(account)}>Switch to {account.label}</button>)}
      </div>}
      {switchError && <span role="alert">{switchError}</span>}
    </div>}
  </>;
}
