export function DemoEnvironmentMenu({ onOpenReferenceJourneys }: { onOpenReferenceJourneys: () => void }) {
  return <details className="demo-environment-menu">
    <summary>Demo environment</summary>
    <div className="shell-popover" role="group" aria-label="Demo environment tools">
      <div><strong>Non-production sample data</strong><span>Use the same enterprise workspace with reference records and optional scenario guidance.</span></div>
      <button type="button" onClick={onOpenReferenceJourneys}>Reference journeys</button>
    </div>
  </details>;
}
