import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { loadContext, loadDemoAccounts, loadSessionStatus, loginDemo, logoutDemo, type DemoAccount, type RuntimeContext } from "../api";
import { apiErrorKind } from "../http";
import type { RuntimePresentation } from "../runtimePresentation";
import { DemoLoginPage } from "./DemoLoginPage";

type GateState = "checking" | "ready" | "login";
type SessionRuntime = RuntimeContext & { demo_mode?: boolean };

export type DemoSessionTools = {
  accounts: DemoAccount[];
  currentAccountLabel: string;
  switchAccount: (account: DemoAccount) => Promise<void>;
};

const DemoSessionContext = createContext<DemoSessionTools | null>(null);

export function useDemoSessionTools() {
  return useContext(DemoSessionContext);
}

export function SessionGate({ children }: { children: ReactNode; presentation?: RuntimePresentation }) {
  const [state, setState] = useState<GateState>("checking");
  const [accounts, setAccounts] = useState<DemoAccount[]>([]);
  const [demoMode, setDemoMode] = useState(false);
  const [loginError, setLoginError] = useState("");
  const [currentAccountLabel, setCurrentAccountLabel] = useState("Demo account");

  async function rememberContext(context: SessionRuntime) {
    const isDemo = context.demo_mode === true;
    setDemoMode(isDemo);
    if (!isDemo) {
      setAccounts([]);
      setCurrentAccountLabel(context.actor.name || "Signed-in account");
      return;
    }

    const available = await loadDemoAccounts().catch(() => []);
    setAccounts(available);
    const actorRoles = new Set(context.actor.role_codes ?? []);
    const matchedByName = available.find((account) => account.label === context.actor.name);
    const matchedByRole = available
      .map((account) => ({ account, score: account.role_codes.filter((role) => actorRoles.has(role)).length }))
      .filter((candidate) => candidate.score > 0)
      .sort((left, right) => right.score - left.score || right.account.role_codes.length - left.account.role_codes.length)[0]?.account;
    setCurrentAccountLabel(matchedByName?.label ?? matchedByRole?.label ?? context.actor.name ?? "Demo account");
  }

  async function enter() {
    setState("checking");
    setLoginError("");
    let status;
    try {
      status = await loadSessionStatus();
    } catch {
      setDemoMode(false);
      setState("ready");
      return;
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
        await rememberContext(await loadContext() as SessionRuntime);
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
      await rememberContext(await loadContext() as SessionRuntime);
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

  async function switchAccount(account: DemoAccount) {
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

  const demoTools = useMemo<DemoSessionTools | null>(() => demoMode ? {
    accounts,
    currentAccountLabel,
    switchAccount,
  } : null, [accounts, currentAccountLabel, demoMode]);

  if (state === "checking") {
    return <main className="demo-login-shell"><section className="demo-login-panel compact" aria-live="polite" aria-busy="true"><span className="eyebrow">ClearSight</span><h1>Opening your workspace…</h1></section></main>;
  }
  if (state === "login") return <DemoLoginPage accounts={accounts} onAuthenticated={enter} initialError={loginError}/>;

  return <DemoSessionContext.Provider value={demoTools}>{children}</DemoSessionContext.Provider>;
}
