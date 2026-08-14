import { FormEvent, useState } from "react";
import { loginDemo, type DemoAccount } from "../api";

export function DemoLoginPage({ accounts, onAuthenticated, initialError = "" }: { accounts: DemoAccount[]; onAuthenticated: () => Promise<void>; initialError?: string }) {
  const [selected, setSelected] = useState<DemoAccount | undefined>(accounts[0]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(initialError);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!selected) {
      setError("Choose a demo account to continue.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await loginDemo(selected.username, selected.password);
      await onAuthenticated();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Demo sign-in could not be completed.");
    } finally {
      setBusy(false);
    }
  }

  return <main className="demo-login-shell">
    <section className="demo-login-panel" aria-labelledby="demo-login-title">
      <div className="demo-login-intro">
        <div className="demo-login-brand" aria-hidden="true">C</div>
        <span className="eyebrow">ClearSight stakeholder demo</span>
        <h1 id="demo-login-title">Choose a demo account</h1>
        <p>Each account has different responsibilities and access. You can switch accounts from the workspace at any time.</p>
      </div>

      <div className="demo-login-layout">
        <div className="demo-account-list" aria-label="Available demo roles">
          {accounts.map((account) => <button
            key={account.username}
            type="button"
            className={`demo-account-card${selected?.username === account.username ? " selected" : ""}`}
            aria-pressed={selected?.username === account.username}
            onClick={() => { setSelected(account); setError(""); }}
          >
            <strong>{account.label}</strong>
            <span className="demo-account-role">{account.role_codes.map(humanize).join(" · ")}</span>
          </button>)}
        </div>

        <form className="demo-login-form" onSubmit={(event) => void submit(event)}>
          <div><span className="eyebrow">Selected account</span><h2>{selected?.label ?? "Choose an account"}</h2><p>{selected?.username ?? "Select a role from the list."}</p></div>
          {error && <div className="demo-login-error" role="alert">{error}</div>}
          <button className="primary-button" type="submit" disabled={busy || !selected}>{busy ? "Opening workspace…" : `Continue as ${selected?.label ?? "selected account"}`}</button>
          <small>Switch accounts at any time from the account control in the workspace.</small>
        </form>
      </div>
    </section>
  </main>;
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
