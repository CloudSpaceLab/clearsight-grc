import { ActionLink, Button, Notice, Surface, Tabs } from "../ui";
import "./data-integrations.css";

type View = "sources" | "mapping" | "history";
const views: readonly { id: View; label: string }[] = [
  { id: "sources", label: "Sources" }, { id: "mapping", label: "Field mapping" }, { id: "history", label: "Sync history" },
];

export function DataIntegrationsSection({ hash, onNavigate, importsEnabled, onOpenImports }: {
  hash: string; onNavigate: (hash: string) => void; importsEnabled: boolean; onOpenImports: () => void;
}) {
  const requested = new URLSearchParams(hash.split("?", 2)[1] ?? "").get("tab");
  const selected: View = requested === "mapping" || requested === "history" ? requested : "sources";
  return <section className="integrations-workspace" aria-labelledby="integrations-heading">
    <header className="integrations-workspace__heading">
      <div>
        <div className="integrations-workspace__crumb"><ActionLink href="#configure">Settings</ActionLink><span aria-hidden="true">›</span><span>Data connections</span></div>
        <h1 id="integrations-heading">Integrations &amp; data sources</h1>
      </div>
    </header>
    <Tabs ariaLabel="Integration views" items={views} selectedKey={selected}
      onSelectionChange={(view) => onNavigate(view === "sources" ? "#configure/data" : `#configure/data?tab=${view}`)}>
      {(view) => view === "sources" ? <div className="integrations-workspace__layout">
        <section aria-labelledby="integration-sources-heading"><Surface>
          <div className="integrations-workspace__panel">
            <h2 id="integration-sources-heading">Data sources</h2>
            <Notice tone="info">Connected sources are unavailable.</Notice>
          </div>
        </Surface></section>
        <aside aria-labelledby="integration-imports-heading"><Surface>
          <div className="integrations-workspace__panel">
            <h2 id="integration-imports-heading">Document imports</h2>
            {!importsEnabled && <p>Document imports are unavailable in this deployment.</p>}
            <Button variant="secondary" isDisabled={!importsEnabled} onPress={onOpenImports}>Open document imports</Button>
          </div>
        </Surface></aside>
      </div> : <section className="integrations-workspace__unavailable" aria-labelledby="integration-view-heading"><Surface>
        <div className="integrations-workspace__panel">
          <h2 id="integration-view-heading">{view === "mapping" ? "Field mapping" : "Sync history"}</h2>
          <Notice tone="info">{view === "mapping" ? "Field mappings are unavailable." : "Sync history is unavailable."}</Notice>
        </div>
      </Surface></section>}
    </Tabs>
  </section>;
}
