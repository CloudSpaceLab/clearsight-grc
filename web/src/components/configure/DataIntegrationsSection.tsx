export function DataIntegrationsSection({ importsEnabled, onOpenImports }: { importsEnabled: boolean; onOpenImports: () => void }) {
  return <section className="configure-domain" aria-labelledby="data-integrations-heading">
    <header className="configure-domain-header">
      <div><span className="eyebrow">Configuration · data</span><h2 id="data-integrations-heading">Data & integrations</h2><p>Review imports and connected sources.</p></div>
    </header>
    <div className="configure-area-list compact">
      <button className="configure-area-row" type="button" disabled={!importsEnabled} onClick={onOpenImports}>
        <span className="configure-area-copy"><strong>Document imports</strong><small>Compare source documents with current Programs, controls and evidence.</small></span>
        <span className="configure-area-state"><span>{importsEnabled ? "Available" : "Unavailable in this deployment"}</span><b aria-hidden="true">›</b></span>
      </button>
      <div className="configure-area-row passive">
        <span className="configure-area-copy"><strong>Connected sources</strong><small>Connection settings are unavailable in this workspace.</small></span>
      </div>
    </div>
  </section>;
}
