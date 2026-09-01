import { useDemoSessionTools } from "./SessionGate";

export function DemoEnvironmentMenu({ onOpenReferenceJourneys }: { onOpenReferenceJourneys?: () => void }) {
  const session = useDemoSessionTools();
  const alternateAccounts = session?.accounts.filter((account) => account.label !== session.currentAccountLabel) ?? [];
  const summaryText = session ? `Viewing as ${session.currentAccountLabel}` : "Demo environment";
  const summaryLabel = session ? `${summaryText}. Open demo account menu` : summaryText;

  return <details className="demo-environment-menu">
    <summary role="button" aria-label={summaryLabel}>{summaryText}</summary>
    <div className="shell-popover" role="group" aria-label="Demo environment tools">
      <div><strong>{onOpenReferenceJourneys ? "Non-production sample data" : "Demo account"}</strong><span>{onOpenReferenceJourneys ? "Use the same enterprise workspace with reference records and optional scenario guidance." : "Change the signed-in demo account to review this workspace with different responsibilities and access."}</span></div>
      {session && <div className="demo-environment-account" aria-live="polite">
        <span>Viewing as</span>
        <strong>{session.currentAccountLabel}</strong>
      </div>}
      {alternateAccounts.length > 0 && <div className="demo-environment-switches" aria-label="Switch demo account">
        <span>Switch account</span>
        {alternateAccounts.map((account) => <button key={account.username} type="button" onClick={() => void session?.switchAccount(account)}>Switch to {account.label}</button>)}
      </div>}
      {onOpenReferenceJourneys && <button type="button" onClick={onOpenReferenceJourneys}>Reference journeys</button>}
    </div>
  </details>;
}
