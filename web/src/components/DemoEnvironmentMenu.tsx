import { useDemoSessionTools } from "./SessionGate";

export function DemoEnvironmentMenu({ onOpenReferenceJourneys }: { onOpenReferenceJourneys: () => void }) {
  const session = useDemoSessionTools();
  const alternateAccounts = session?.accounts.filter((account) => account.label !== session.currentAccountLabel) ?? [];
  const summaryLabel = session ? `Viewing as ${session.currentAccountLabel}. Demo environment` : "Demo environment";

  return <details className="demo-environment-menu">
    <summary aria-label={summaryLabel}>Demo environment</summary>
    <div className="shell-popover" role="group" aria-label="Demo environment tools">
      <div><strong>Non-production sample data</strong><span>Use the same enterprise workspace with reference records and optional scenario guidance.</span></div>
      {session && <div className="demo-environment-account" aria-live="polite">
        <span>Viewing as</span>
        <strong>{session.currentAccountLabel}</strong>
      </div>}
      {alternateAccounts.length > 0 && <div className="demo-environment-switches" aria-label="Switch demo account">
        <span>Switch account</span>
        {alternateAccounts.map((account) => <button key={account.username} type="button" onClick={() => void session?.switchAccount(account)}>Switch to {account.label}</button>)}
      </div>}
      <button type="button" onClick={onOpenReferenceJourneys}>Reference journeys</button>
    </div>
  </details>;
}
