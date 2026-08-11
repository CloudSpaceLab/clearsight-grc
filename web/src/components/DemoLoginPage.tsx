import { FormEvent, useMemo, useState } from "react";
import { loginDemo, type DemoAccount } from "../api";

export function DemoLoginPage({ accounts, onAuthenticated }: { accounts: DemoAccount[]; onAuthenticated: () => Promise<void> }) {
  const initial = accounts[0];
  const [username, setUsername] = useState(initial?.username ?? "");
  const [password, setPassword] = useState(initial?.password ?? "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const selected = useMemo(() => accounts.find((account) => account.username === username), [accounts, username]);

  function choose(account: DemoAccount) {
    setUsername(account.username);
    setPassword(account.password);
    setError("");
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      await loginDemo(username, password);
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
        <h1 id="demo-login-title">See the bank from a real role</h1>
        <p>Choose a default demo identity to inspect the exact work, visibility and configuration that role receives. These accounts exist only while demo mode is enabled.</p>
      </div>

      <div className="demo-login-layout">
        <div className="demo-account-list" aria-label="Available demo roles">
          {accounts.map((account) => <button
            key={account.username}
            type="button"
            className={`demo-account-card${selected?.username === account.username ? " selected" : ""}`}
            aria-pressed={selected?.username === account.username}
            onClick={() => choose(account)}
          >
            <span className="demo-account-role">{account.label}</span>
            <strong>{account.role_codes.map(humanize).join(" · ")}</strong>
            <span className="demo-account-credential"><b>User</b><code>{account.username}</code></span>
            <span className="demo-account-credential"><b>Password</b><code>{account.password}</code></span>
          </button>)}
        </div>

        <form className="demo-login-form" onSubmit={(event) => void submit(event)}>
          <div><span className="eyebrow">Selected identity</span><h2>{selected?.label ?? "Demo sign-in"}</h2><p>Credentials are intentionally visible because this is a non-production role simulator.</p></div>
          <label>Email<input type="email" autoComplete="username" value={username} onChange={(event) => setUsername(event.target.value)} required/></label>
          <label>Password<input type="password" autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} required/></label>
          {error && <div className="demo-login-error" role="alert">{error}</div>}
          <button className="primary-button" type="submit" disabled={busy}>{busy ? "Signing in…" : "Enter demo"}</button>
          <small>No production password, OIDC token or customer identity is exposed by this page.</small>
        </form>
      </div>
    </section>
  </main>;
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
