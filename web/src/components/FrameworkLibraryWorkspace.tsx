import { useMemo, useState } from "react";
import type { FormEvent } from "react";
import { useFrameworkEditionCatalog } from "../hooks/useFrameworkEditionCatalog";
import { useMediaQuery } from "../hooks/useMediaQuery";
import type { FrameworkEditionOption } from "../programOperationsApi";
import { Button, EmptyState, FilterBar, FocusedSheet, Notice, SearchField, SelectableRecord, StatusBadge, TextField, SelectField } from "./ui";
import { FrameworkEditionProvenance, frameworkEditionHasVerifiableSource, frameworkEditionRightsLabel } from "./FrameworkEditionProvenance";
import "../framework-library.css";
import "../program-standards.css";

type RightsFilter = "ALL" | "CITATIONS_ONLY" | "SUMMARIES_PERMITTED" | "UNVERIFIED";
const rightsOptions: readonly {id: RightsFilter; label: string}[] = [
  { id: "ALL", label: "All content rights" },
  { id: "CITATIONS_ONLY", label: "Citations only" },
  { id: "SUMMARIES_PERMITTED", label: "Summaries permitted" },
  { id: "UNVERIFIED", label: "Not verified" },
];

type Props = {
  onBack: () => void;
  onChooseProgram: (edition: FrameworkEditionOption) => void;
};

function publisherMark(value: string): string {
  return value.trim().toUpperCase().replace(/[^A-Z0-9]/g, "").slice(0, 4) || "REF";
}

function normalizeJurisdiction(value: string): string | null {
  const code = value.trim().toUpperCase();
  return /^[A-Z][A-Z0-9._-]{0,63}$/.test(code) ? code : null;
}

export function FrameworkLibraryWorkspace({ onBack, onChooseProgram }: Props) {
  const [jurisdictionDraft, setJurisdictionDraft] = useState("GLOBAL");
  const [jurisdiction, setJurisdiction] = useState("GLOBAL");
  const [jurisdictionError, setJurisdictionError] = useState("");
  const [search, setSearch] = useState("");
  const [rightsFilter, setRightsFilter] = useState<RightsFilter>("ALL");
  const [selectedID, setSelectedID] = useState("");
  const [sheetOpen, setSheetOpen] = useState(false);
  const [jurisdictionExpanded, setJurisdictionExpanded] = useState(false);
  const compact = useMediaQuery("(max-width: 900px)");
  const catalog = useFrameworkEditionCatalog(true, jurisdiction);

  const visible = useMemo(() => {
    const query = search.trim().toLocaleLowerCase();
    return catalog.items.filter((item) =>
      (rightsFilter === "ALL" || (rightsFilter === "UNVERIFIED"
        ? !["CITATIONS_ONLY", "SUMMARIES_PERMITTED"].includes(item.license_basis)
        : item.license_basis === rightsFilter)) &&
      (!query || [item.title, item.publisher_code, item.framework_code, item.edition_code, item.jurisdiction]
        .some((value) => value.toLocaleLowerCase().includes(query))));
  }, [catalog.items, search, rightsFilter]);

  // Source inspection requires an explicit record selection. Filtering out the
  // selected edition also removes its action until that record is selected again.
  const selected = visible.find((item) => item.id === selectedID) ?? null;
  // The approved desktop library opens with a populated reference inspector.
  // Previewing is read-only: a user must still select an exact edition before
  // proceeding into the governed Program adoption handoff.
  const preview = selected ?? (catalog.state === "ready" && catalog.loadedJurisdiction === jurisdiction
    ? visible[0] ?? null : null);

  function applyJurisdiction(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const code = normalizeJurisdiction(jurisdictionDraft);
    if (!code) {
      setJurisdictionError("Enter a valid jurisdiction code.");
      return;
    }
    setJurisdictionError("");
    setSearch("");
    setRightsFilter("ALL");
    setSelectedID("");
    setSheetOpen(false);
    setJurisdictionDraft(code);
    setJurisdiction(code);
    setJurisdictionExpanded(false);
  }

  function inspect(edition: FrameworkEditionOption) {
    setSelectedID(edition.id);
    setSheetOpen(compact);
  }

  const inspector = preview && <div className="cs-framework-library__sheet">
    <div className="cs-framework-library__detail-band" data-selected={Boolean(selected)} aria-hidden="true">
      <span className="cs-framework-library__publisher-mark">{publisherMark(preview.publisher_code)}</span>
      <span className="cs-framework-library__detail-state">
        <strong>{selected ? "Selected edition" : "Edition preview"}</strong>
        <small>Published · {preview.source_published_on}</small>
      </span>
    </div>
    <div className="cs-framework-library__detail-body">
      <h2>{preview.title}</h2>
      <p className="cs-framework-library__detail-subtitle">{preview.publisher_code} · {preview.framework_code}</p>
      <div className="cs-framework-library__detail-chips" aria-label="Edition classification">
        <StatusBadge tone="info">{preview.jurisdiction}</StatusBadge>
        <StatusBadge tone="neutral">Edition {preview.edition_code}</StatusBadge>
        <StatusBadge tone={frameworkEditionHasVerifiableSource(preview) ? "neutral" : "unknown"}>
          {frameworkEditionRightsLabel(preview.license_basis)}
        </StatusBadge>
      </div>
      <section className="cs-framework-library__detail-section" aria-labelledby="cs-framework-library-facts-heading">
        <h3 id="cs-framework-library-facts-heading">Edition details</h3>
        <FrameworkEditionProvenance edition={preview}/>
        <p className="cs-framework-library__disclaimer">Publication does not establish Program adoption or compliance.</p>
      </section>
      {!frameworkEditionHasVerifiableSource(preview) && <Notice tone="warning">
        Source or reuse rights are unverified. This edition cannot be selected for a Program.
      </Notice>}
      <section className="cs-framework-library__detail-section cs-framework-library__adoption" aria-labelledby="cs-framework-adoption-heading">
        <h3 id="cs-framework-adoption-heading">Use in a Program</h3>
        <ol className="cs-framework-library__steps">
          <li>Select a Program</li>
          <li>Confirm applicable requirements</li>
          <li>Review control and evidence links</li>
        </ol>
        <div className="cs-program-standards__actions">
          <Button variant="primary" isDisabled={!selected || catalog.state !== "ready" || catalog.loadedJurisdiction !== jurisdiction || !frameworkEditionHasVerifiableSource(preview)}
            onPress={() => { if (selected) onChooseProgram(selected); }}>Choose Program</Button>
        </div>
        {!selected && <p className="cs-framework-library__preview-note">Select an edition in the list to continue.</p>}
      </section>
    </div>
  </div>;

  return <section className="cs-framework-library portfolio-workspace" aria-label="Framework Library workspace" id="framework-library-workspace">
    <header className="cs-framework-library__page-heading" aria-label="Framework Library heading">
      <div className="cs-framework-library__page-copy">
        <span className="eyebrow">Compliance / Reference library</span>
        <h1>Framework Library</h1>
        <p>Review an edition before adding it to a Program.</p>
      </div>
      <div className="cs-framework-library__page-actions">
        <span className="cs-framework-library__catalog-label">Published reference catalog</span>
        <Button variant="secondary" onPress={onBack}>View Programs</Button>
      </div>
    </header>
    <header className="cs-framework-library__header">
      <nav className="cs-framework-library__views" aria-label="Library views">
        <span aria-current="page" className="cs-framework-library__view cs-framework-library__view--active">Explore</span>
        <button type="button" className="cs-framework-library__view" onClick={onBack}>In my Programs</button>
        <button type="button" className="cs-framework-library__view" disabled
          title="Edition updates are not available from the current catalog">Edition updates</button>
      </nav>
    </header>

    <FilterBar label="Framework filters"
      onClear={search || rightsFilter !== "ALL" || jurisdiction !== "GLOBAL" ? () => {
        setSearch("");
        setRightsFilter("ALL");
        setSelectedID("");
        setSheetOpen(false);
        setJurisdiction("GLOBAL");
        setJurisdictionDraft("GLOBAL");
        setJurisdictionExpanded(false);
        setJurisdictionError("");
      } : undefined}
      fields={<>
        <SearchField label="Search loaded standards" placeholder="Search standards or publishers" value={search}
          onChange={(value) => { setSearch(value); setSelectedID(""); setSheetOpen(false); }}
          isLoading={catalog.state === "loading"}/>
        <SelectField label="Content rights" value={rightsFilter} placeholder="All content rights" isLabelHidden={compact}
          allowsEmpty={false} options={rightsOptions} onChange={(value) => {
            setRightsFilter(value ?? "ALL");
            setSelectedID("");
            setSheetOpen(false);
          }}/>
        {compact && <Button variant="secondary" size="compact" aria-expanded={jurisdictionExpanded}
          onPress={() => {
            if (jurisdictionExpanded) { setJurisdictionDraft(jurisdiction); setJurisdictionError(""); }
            setJurisdictionExpanded((value) => !value);
          }}>
          {jurisdictionExpanded ? "Hide jurisdiction filter" : "Jurisdiction: " + jurisdiction}
        </Button>}
        {(!compact || jurisdictionExpanded) && <form className="cs-framework-library__jurisdiction" onSubmit={applyJurisdiction}>
          <TextField id="framework-jurisdiction" label="Jurisdiction code" autoComplete="off"
            value={jurisdictionDraft} maxLength={64} onChange={setJurisdictionDraft}
            isInvalid={Boolean(jurisdictionError)} errorMessage={jurisdictionError}/>
          <Button type="submit" variant="secondary" size="compact">Apply</Button>
        </form>}
      </>}/>

    <div className="cs-framework-library__layout">
      <div className="cs-framework-library__register">

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
            <h2>Reference editions</h2>
            <span role="status">{visible.length} matching loaded · {catalog.items.length} loaded editions · {jurisdiction}{catalog.nextAfterID ? " · More available" : ""}</span>
          </div>
          <ul className="cs-framework-library__list" aria-label="Published framework editions">
            {visible.map((item) => <li key={item.id}>
              <SelectableRecord
                title={item.title}
                startContent={<span className="cs-framework-library__publisher-mark">{publisherMark(item.publisher_code)}</span>}
                metadata={compact
                  ? item.publisher_code + " · " + item.framework_code + " · Edition " + item.edition_code
                  : item.framework_code}
                description={compact
                  ? item.jurisdiction + " · " + frameworkEditionRightsLabel(item.license_basis)
                  : item.jurisdiction + " · Published " + item.source_published_on}
                endContent={compact ? undefined : <span className="cs-framework-library__record-facts">
                  <span className="cs-framework-library__record-edition">
                    <strong>{item.publisher_code}</strong>
                    <small>Edition {item.edition_code}</small>
                  </span>
                  <StatusBadge tone={frameworkEditionRightsLabel(item.license_basis) === "Not verified" ? "unknown" : "neutral"}>
                    {frameworkEditionRightsLabel(item.license_basis)}
                  </StatusBadge>
                </span>}
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
