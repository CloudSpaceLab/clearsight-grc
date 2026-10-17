import { useMemo, useState } from "react";
import type { FormEvent } from "react";
import { useFrameworkEditionCatalog } from "../hooks/useFrameworkEditionCatalog";
import type { FrameworkEditionOption } from "../programOperationsApi";
import { Button, DataTable, EmptyState, FilterBar, FocusedSheet, SearchField, StatusBadge } from "./ui";
import type { DataColumn } from "./ui";
import { FrameworkEditionProvenance } from "./FrameworkEditionProvenance";
import "../framework-library.css";
import "../program-standards.css";

type Props = {
  onBack: () => void;
  onChooseProgram: (edition: FrameworkEditionOption) => void;
};

const columns: DataColumn<FrameworkEditionOption>[] = [
  {
    id: "standard", header: "Standard", mobileLayout: "full-width",
    render: (item) => <span className="cs-framework-library__identity">
      <strong>{item.title}</strong>
      <small>{item.publisher_code} · {item.framework_code}</small>
    </span>,
    accessibleText: (item) => item.title + ", " + item.publisher_code + " " + item.framework_code,
  },
  { id: "edition", header: "Edition",
    render: (item) => item.edition_code,
    accessibleText: (item) => item.edition_code,
  },
  { id: "jurisdiction", header: "Jurisdiction",
    render: (item) => item.jurisdiction,
    accessibleText: (item) => item.jurisdiction,
  },
  { id: "published", header: "Published",
    render: (item) => <time dateTime={item.source_published_on}>{item.source_published_on}</time>,
    accessibleText: (item) => item.source_published_on,
  },
  { id: "rights", header: "Rights",
    render: (item) => <StatusBadge tone="neutral">{item.license_basis === "CITATIONS_ONLY" ? "Citations only"
      : item.license_basis === "SUMMARIES_PERMITTED" ? "Summaries permitted" : "Not verified"}</StatusBadge>,
    accessibleText: (item) => item.license_basis,
  },
];

function normalizeJurisdiction(value: string): string | null {
  const code = value.trim().toUpperCase();
  return /^[A-Z][A-Z0-9._-]{0,63}$/.test(code) ? code : null;
}

export function FrameworkLibraryWorkspace({ onBack, onChooseProgram }: Props) {
  const [jurisdictionDraft, setJurisdictionDraft] = useState("GLOBAL");
  const [jurisdiction, setJurisdiction] = useState("GLOBAL");
  const [jurisdictionError, setJurisdictionError] = useState("");
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<FrameworkEditionOption | null>(null);
  const catalog = useFrameworkEditionCatalog(true, jurisdiction);

  const visible = useMemo(() => {
    const query = search.trim().toLocaleLowerCase();
    if (!query) return catalog.items;
    return catalog.items.filter((item) =>
      [item.title, item.publisher_code, item.framework_code, item.edition_code, item.jurisdiction]
        .some((value) => value.toLocaleLowerCase().includes(query)));
  }, [catalog.items, search]);

  function applyJurisdiction(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const code = normalizeJurisdiction(jurisdictionDraft);
    if (!code) {
      setJurisdictionError("Enter a valid jurisdiction code.");
      return;
    }
    setJurisdictionError("");
    setSearch("");
    setSelected(null);
    setJurisdictionDraft(code);
    setJurisdiction(code);
  }

  return <section className="cs-framework-library portfolio-workspace" aria-label="Framework Library workspace" id="framework-library-workspace">
    <header className="cs-register-header cs-framework-library__header">
      <div>
        <span className="eyebrow">Standards</span>
        <h2>Published editions</h2>
        <p>{catalog.items.length} loaded{catalog.nextAfterID ? " · More available" : ""} · {jurisdiction}</p>
      </div>
      <Button variant="secondary" onPress={onBack}>Back to Programs</Button>
    </header>

    <FilterBar label="Framework filters" resultCount={visible.length}
      resultLabel={(count) => count + " matching loaded edition" + (count === 1 ? "" : "s")}
      onClear={search ? () => setSearch("") : undefined}
      fields={<>
        <SearchField label="Search loaded standards" placeholder="Search loaded standards" value={search}
          onChange={setSearch} isLoading={catalog.state === "loading"}/>
        <form className="cs-framework-library__jurisdiction" onSubmit={applyJurisdiction}>
          <label htmlFor="framework-jurisdiction">Jurisdiction code</label>
          <div className="cs-framework-library__jurisdiction-controls">
            <input id="framework-jurisdiction" type="text" autoComplete="off"
              value={jurisdictionDraft} maxLength={64} onChange={(event) => setJurisdictionDraft(event.target.value)}
              aria-invalid={Boolean(jurisdictionError)} aria-describedby={jurisdictionError ? "framework-jurisdiction-error" : undefined}/>
            <Button type="submit" variant="secondary" size="compact">Apply</Button>
          </div>
        </form>
      </>}/>
    {jurisdictionError && <p role="alert" id="framework-jurisdiction-error">{jurisdictionError}</p>}

    {catalog.state === "loading" && <p role="status">Loading published editions…</p>}
    {catalog.state === "denied" && <EmptyState role="alert" population="Framework Library"
      title="Published editions unavailable" description="Select an authorized legal entity to view published standards."/>}
    {catalog.state === "error" && <EmptyState role="alert" population="Framework Library"
      title="Published editions could not be loaded" description="The current catalog is unavailable."
      action={<Button variant="secondary" onPress={catalog.retry}>Retry catalog</Button>}/>}
    {catalog.state === "ready" && catalog.items.length === 0 && <EmptyState population="Framework Library"
      title="No published editions" description={"No verified editions are published for " + jurisdiction + "."}/>}
    {catalog.state === "ready" && catalog.items.length > 0 && visible.length === 0 && <EmptyState population="Framework Library"
      title="No matching loaded editions" description="Clear the search or load additional editions."
      action={<Button variant="secondary" onPress={() => setSearch("")}>Clear search</Button>}/>}

    {catalog.items.length > 0 && catalog.state !== "denied" && <DataTable ariaLabel="Published framework editions"
      rows={visible} rowKey={(item) => item.id} rowName={(item) => item.title + " " + item.edition_code}
      columns={columns} onRowAction={setSelected} rowActionLabel="Inspect edition"
      isLoading={catalog.state === "loading"}/>}

    {catalog.nextAfterID && catalog.state !== "denied" && <div className="cs-framework-library__pagination">
      <span>{catalog.items.length} loaded · More available</span>
      <Button variant="secondary" onPress={catalog.loadMore} isDisabled={catalog.state === "loading"}>Load more editions</Button>
    </div>}
    {selected && catalog.items.some((item) => item.id === selected.id) && <FocusedSheet label={"Edition details: " + selected.title} onClose={() => setSelected(null)} size="wide">
      <div className="cs-framework-library__sheet">
        <span className="eyebrow">Published reference</span>
        <h2>{selected.title}</h2>
        <StatusBadge tone="info">Edition {selected.edition_code}</StatusBadge>
        <FrameworkEditionProvenance edition={selected}/>
        <p className="cs-framework-library__disclaimer">Publication does not establish Program adoption or compliance.</p>
        <div className="cs-program-standards__actions">
          <Button variant="primary" onPress={() => onChooseProgram(selected)}>Choose Program</Button>
          <Button variant="quiet" onPress={() => setSelected(null)}>Close</Button>
        </div>
      </div>
    </FocusedSheet>}
  </section>;
}
