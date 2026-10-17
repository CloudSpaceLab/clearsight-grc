import { useMemo, useState } from "react";
import type { FormEvent } from "react";
import { useFrameworkEditionCatalog } from "../hooks/useFrameworkEditionCatalog";
import { useMediaQuery } from "../hooks/useMediaQuery";
import type { FrameworkEditionOption } from "../programOperationsApi";
import { Button, EmptyState, FilterBar, FocusedSheet, Notice, SearchField, SelectableRecord, StatusBadge, TextField } from "./ui";
import { FrameworkEditionProvenance, frameworkEditionRightsLabel } from "./FrameworkEditionProvenance";
import "../framework-library.css";
import "../program-standards.css";

type Props = {
  onBack: () => void;
  onChooseProgram: (edition: FrameworkEditionOption) => void;
};

function normalizeJurisdiction(value: string): string | null {
  const code = value.trim().toUpperCase();
  return /^[A-Z][A-Z0-9._-]{0,63}$/.test(code) ? code : null;
}

export function FrameworkLibraryWorkspace({ onBack, onChooseProgram }: Props) {
  const [jurisdictionDraft, setJurisdictionDraft] = useState("GLOBAL");
  const [jurisdiction, setJurisdiction] = useState("GLOBAL");
  const [jurisdictionError, setJurisdictionError] = useState("");
  const [search, setSearch] = useState("");
  const [selectedID, setSelectedID] = useState("");
  const [sheetOpen, setSheetOpen] = useState(false);
  const compact = useMediaQuery("(max-width: 900px)");
  const catalog = useFrameworkEditionCatalog(true, jurisdiction);

  const visible = useMemo(() => {
    const query = search.trim().toLocaleLowerCase();
    if (!query) return catalog.items;
    return catalog.items.filter((item) =>
      [item.title, item.publisher_code, item.framework_code, item.edition_code, item.jurisdiction]
        .some((value) => value.toLocaleLowerCase().includes(query)));
  }, [catalog.items, search]);

  // Source inspection requires an explicit record selection. Filtering out the
  // selected edition also removes its action until that record is selected again.
  const selected = visible.find((item) => item.id === selectedID) ?? null;

  function applyJurisdiction(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const code = normalizeJurisdiction(jurisdictionDraft);
    if (!code) {
      setJurisdictionError("Enter a valid jurisdiction code.");
      return;
    }
    setJurisdictionError("");
    setSearch("");
    setSelectedID("");
    setSheetOpen(false);
    setJurisdictionDraft(code);
    setJurisdiction(code);
  }

  function inspect(edition: FrameworkEditionOption) {
    setSelectedID(edition.id);
    setSheetOpen(compact);
  }

  const inspector = selected && <div className="cs-framework-library__sheet">
    <span className="eyebrow">Published reference</span>
    <h2>{selected.title}</h2>
    <StatusBadge tone="info">Edition {selected.edition_code}</StatusBadge>
    <FrameworkEditionProvenance edition={selected}/>
    <p className="cs-framework-library__disclaimer">Publication does not establish Program adoption or compliance.</p>
    <div className="cs-program-standards__actions">
      <Button variant="primary" isDisabled={catalog.state !== "ready"}
        onPress={() => onChooseProgram(selected)}>Choose Program</Button>
    </div>
  </div>;

  return <section className="cs-framework-library portfolio-workspace" aria-label="Framework Library workspace" id="framework-library-workspace">
    <header className="cs-register-header cs-framework-library__header">
      <div>
        <span className="eyebrow">Compliance / Reference library</span>
        <h2>Framework Library</h2>
        <p>{catalog.items.length} loaded{catalog.nextAfterID ? " · More available" : ""} · {jurisdiction}</p>
      </div>
      <Button variant="secondary" onPress={onBack}>Back to Programs</Button>
    </header>

    <div className="cs-framework-library__layout">
      <div className="cs-framework-library__register">
        <FilterBar label="Framework filters" resultCount={visible.length}
          resultLabel={(count) => count + " matching loaded edition" + (count === 1 ? "" : "s")}
          onClear={search ? () => setSearch("") : undefined}
          fields={<>
            <SearchField label="Search loaded standards" placeholder="Search loaded standards" value={search}
              onChange={setSearch} isLoading={catalog.state === "loading"}/>
            <form className="cs-framework-library__jurisdiction" onSubmit={applyJurisdiction}>
              <TextField id="framework-jurisdiction" label="Jurisdiction code" autoComplete="off"
                value={jurisdictionDraft} maxLength={64} onChange={setJurisdictionDraft}
                isInvalid={Boolean(jurisdictionError)} errorMessage={jurisdictionError}/>
              <Button type="submit" variant="secondary" size="compact">Apply</Button>
            </form>
          </>}/>
        {catalog.state === "loading" && <p role="status">{catalog.items.length ? "Loading additional editions…" : "Loading published editions…"}</p>}
        {catalog.state === "denied" && <EmptyState role="alert" population="Framework Library"
          title="Published editions unavailable" description="Select an authorized legal entity to view published standards."/>}
        {catalog.state === "error" && (catalog.items.length
          ? <Notice tone="warning">Additional editions unavailable. {catalog.items.length} loaded editions remain visible.
            <Button variant="secondary" size="compact" onPress={catalog.retry}>Retry catalog</Button>
          </Notice>
          : <EmptyState role="alert" population="Framework Library"
              title="Published editions could not be loaded" description="The current catalog is unavailable."
              action={<Button variant="secondary" onPress={catalog.retry}>Retry catalog</Button>}/>)} 
        {catalog.state === "ready" && catalog.items.length === 0 && <EmptyState population="Framework Library"
          title="No published editions" description={"No verified editions are published for " + jurisdiction + "."}/>}
        {catalog.state === "ready" && catalog.items.length > 0 && visible.length === 0 && <EmptyState population="Framework Library"
          title="No matching loaded editions" description="Clear the search or load additional editions."
          action={<Button variant="secondary" onPress={() => setSearch("")}>Clear search</Button>}/>}

        {catalog.items.length > 0 && catalog.state !== "denied" && <section aria-label="Published framework editions">
          <div className="cs-framework-library__results-head">
            <h3>Reference editions</h3>
            <span>{visible.length} matching loaded</span>
          </div>
          <ul className="cs-framework-library__list" aria-label="Published framework editions">
            {visible.map((item) => <li key={item.id}>
              <SelectableRecord
                title={item.title}
                metadata={item.publisher_code + " · " + item.framework_code + " · Edition " + item.edition_code}
                description={item.jurisdiction + " · Published " + item.source_published_on}
                endContent={<StatusBadge tone={frameworkEditionRightsLabel(item.license_basis) === "Not verified" ? "unknown" : "neutral"}>
                  {frameworkEditionRightsLabel(item.license_basis)}
                </StatusBadge>}
                isSelected={selected?.id === item.id}
                aria-label={"Inspect edition for " + item.title + " " + item.edition_code}
                onPress={() => inspect(item)}
              />
            </li>)}
          </ul>
        </section>}

        {catalog.nextAfterID && catalog.state !== "denied" && <div className="cs-framework-library__pagination">
          <span>{catalog.items.length} loaded · More available</span>
          <Button variant="secondary" onPress={catalog.loadMore} isDisabled={catalog.state === "loading"}>Load more editions</Button>
        </div>}
      </div>

      {!compact && catalog.items.length > 0 && catalog.state !== "denied" && <aside className="cs-framework-library__inspector" aria-label="Framework edition details">
        {inspector ?? <EmptyState compact population="Framework Library" title="Select an edition"
          description="Choose a published edition to inspect its source and rights."/>}
      </aside>}
    </div>
    {compact && sheetOpen && selected && <FocusedSheet label={"Edition details: " + selected.title}
      closeLabel="Close edition details" onClose={() => setSheetOpen(false)} size="wide">
      {inspector}
      <Button variant="quiet" onPress={() => setSheetOpen(false)}>Close</Button>
    </FocusedSheet>}
  </section>;
}
