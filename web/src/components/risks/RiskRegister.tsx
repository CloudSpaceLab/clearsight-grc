import { useEffect, useState } from "react";
import { listRisks, type RiskListParams } from "../../riskApi";
import type { RiskAppetitePosition, RiskPage, RiskStatus, RiskSummary } from "../../riskTypes";
import { Button, DataTable, EmptyState, FilterBar, Notice, SearchField, SelectField, StatusBadge, type DataColumn } from "../ui";
import { assessmentKindLabel, currentAppetiteLabel, currentAppetiteTone, formatRiskDate, riskStatusLabel, riskStatusTone } from "./riskPresentation";

type Props = {
  organizationName?: string;
  legalEntityName?: string;
  organizationScopeID?: string;
  organizationScopeName?: string;
  onOpenRisk: (id: string) => void;
  loadPage?: (params?: RiskListParams, signal?: AbortSignal) => Promise<RiskPage>;
};

type LoadState = "loading" | "live" | "error";

const statusOptions: ReadonlyArray<{ id: RiskStatus; label: string }> = [
  { id: "ACTIVE", label: "Active" },
  { id: "DRAFT", label: "Draft" },
  { id: "RETIRED", label: "Retired" },
];

const appetiteOptions: ReadonlyArray<{ id: RiskAppetitePosition; label: string }> = [
  { id: "BREACHED", label: "Outside appetite" },
  { id: "APPROACHING", label: "Near appetite limit" },
  { id: "WITHIN", label: "Within appetite" },
  { id: "UNKNOWN", label: "Unknown" },
];

export function RiskRegister({ organizationName, legalEntityName, organizationScopeID, organizationScopeName, onOpenRisk, loadPage = listRisks }: Props) {
  const scope = organizationScopeName || legalEntityName || "this legal entity";
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<RiskStatus>();
  const [appetite, setAppetite] = useState<RiskAppetitePosition>();
  const [cursors, setCursors] = useState<string[]>([]);
  const [page, setPage] = useState<RiskPage>({ items: [] });
  const [state, setState] = useState<LoadState>("loading");
  const [retry, setRetry] = useState(0);
  const currentCursor = cursors[cursors.length - 1];

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    setPage({ items: [] });
    void loadPage({
      search: search.trim() || undefined,
      status,
      appetitePosition: appetite,
      organizationScopeID,
      cursor: currentCursor,
      limit: 25,
    }, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setPage(value);
      setState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setState("error");
    });
    return () => controller.abort();
  }, [appetite, currentCursor, loadPage, organizationScopeID, retry, search, status]);

  function changeSearch(value: string) {
    setSearch(value);
    setCursors([]);
  }

  function changeStatus(value: RiskStatus | undefined) {
    setStatus(value);
    setCursors([]);
  }

  function changeAppetite(value: RiskAppetitePosition | undefined) {
    setAppetite(value);
    setCursors([]);
  }

  function clearFilters() {
    setSearch("");
    setStatus(undefined);
    setAppetite(undefined);
    setCursors([]);
  }

  const hasFilters = Boolean(search.trim() || status || appetite);
  const pagination = cursors.length > 0 || page.next_cursor ? {
    label: "Risk register pages",
    previousLabel: "Load previous page",
    nextLabel: "Load next page",
    onPrevious: cursors.length > 0 ? () => setCursors((value) => value.slice(0, -1)) : undefined,
    onNext: page.next_cursor ? () => setCursors((value) => [...value, page.next_cursor!]) : undefined,
    isLoading: state === "loading",
  } : undefined;

  const columns: readonly DataColumn<RiskSummary>[] = [
    {
      id: "risk",
      header: "Risk",
      mobileLayout: "full-width",
      render: ({ risk }) => <span className="risk-register__identity"><strong>{risk.name}</strong><small>{[risk.code, risk.category].filter(Boolean).join(" · ")}</small></span>,
      accessibleText: ({ risk }) => [risk.name, risk.code, risk.category].filter(Boolean).join(", "),
    },
    {
      id: "appetite",
      header: "Appetite",
      kind: "status",
      render: (item) => <StatusBadge tone={currentAppetiteTone(item.risk.version, item.latest_assessment, item.active_appetite)}>{currentAppetiteLabel(item.risk.version, item.latest_assessment, item.active_appetite)}</StatusBadge>,
      accessibleText: (item) => currentAppetiteLabel(item.risk.version, item.latest_assessment, item.active_appetite),
    },
    {
      id: "assessment",
      header: "Latest assessment",
      render: (item) => item.latest_assessment
        ? <span className="risk-register__assessment"><strong>{assessmentKindLabel(item.latest_assessment.kind)}</strong><small>{item.latest_assessment.method_code} · {item.latest_assessment.method_version}</small></span>
        : "Not assessed",
      accessibleText: (item) => item.latest_assessment ? `${assessmentKindLabel(item.latest_assessment.kind)}, ${item.latest_assessment.method_code} ${item.latest_assessment.method_version}` : "Not assessed",
    },
    {
      id: "status",
      header: "Status",
      kind: "status",
      render: ({ risk }) => <StatusBadge tone={riskStatusTone(risk.status)}>{riskStatusLabel(risk.status)}</StatusBadge>,
      accessibleText: ({ risk }) => riskStatusLabel(risk.status),
    },
    {
      id: "updated",
      header: "Updated",
      render: ({ risk }) => formatRiskDate(risk.updated_at),
      accessibleText: ({ risk }) => formatRiskDate(risk.updated_at),
    },
  ];

  return <section className="risk-register-page" aria-labelledby="risk-register-heading">
    <header className="topbar risk-page-header">
      <div>
        <span className="eyebrow">{organizationName || "Risk portfolio"}</span>
        <h1 id="risk-register-heading">Risks</h1>
        <p>Current risk statements and appetite position for {scope}.</p>
      </div>
    </header>

    <FilterBar
      label="Risk filters"
      fields={<>
        <SearchField label="Search risks" value={search} onChange={changeSearch} placeholder="Risk name, code, category or impact" isLoading={state === "loading"}/>
        <SelectField label="Risk status" value={status} placeholder="All statuses" options={statusOptions} onChange={changeStatus}/>
        <SelectField label="Appetite position" value={appetite} placeholder="All appetite positions" options={appetiteOptions} onChange={changeAppetite}/>
      </>}
      resultCount={state === "live" ? page.items.length : undefined}
      resultLabel={(count) => `${count} shown`}
      onClear={hasFilters ? clearFilters : undefined}
    />

    {state === "loading" && page.items.length === 0 && <p className="risk-load-state" role="status">Loading risks…</p>}
    {state === "error" && <Notice tone="error"><span>Risk register unavailable.</span> <Button variant="secondary" size="compact" onPress={() => setRetry((value) => value + 1)}>Try again</Button></Notice>}
    {state === "live" && page.items.length === 0 && <EmptyState
      population={hasFilters ? `${scope} · current risk filters` : `${scope} · current risk register`}
      title={hasFilters ? "No matching risks" : "No risks in this scope"}
      description={hasFilters ? "Change the filters or search." : `No current risk records were returned for ${scope}.`}
    />}
    {page.items.length > 0 && <DataTable
      ariaLabel="Risk register"
      rows={page.items}
      rowKey={(item) => item.risk.id}
      rowName={(item) => `${item.risk.name}, ${item.risk.code}, ${currentAppetiteLabel(item.risk.version, item.latest_assessment, item.active_appetite)}`}
      columns={columns}
      onRowAction={(item) => onOpenRisk(item.risk.id)}
      rowActionLabel="Open risk"
      isLoading={state === "loading"}
      pagination={pagination}
    />}
  </section>;
}

function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}
