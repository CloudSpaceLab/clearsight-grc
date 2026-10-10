import { useEffect, useState } from "react";
import { listRisks, type RiskListParams } from "../../riskApi";
import { riskAppetiteDistribution } from "../../riskAppetiteDistribution";
import type { RiskAppetitePosition, RiskPage, RiskStatus, RiskSummary } from "../../riskTypes";
import { Button, DataTable, EmptyState, FilterBar, Notice, SearchField, SelectField, StackedDistribution, StatusBadge, type DataColumn } from "../ui";
import { assessmentKindLabel, assessmentRatingLabel, assessmentRatingTone, currentAppetiteLabel, currentAppetiteTone, formatRiskDate, riskStatusLabel, riskStatusTone } from "./riskPresentation";

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
  const appetiteSegments = state === "live" ? riskAppetiteDistribution(page.items) : [];
  const outsideCount = appetiteSegments.find((segment) => segment.id === "breached")?.count ?? 0;
  const nearCount = appetiteSegments.find((segment) => segment.id === "approaching")?.count ?? 0;
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
      id: "rating",
      header: "Rating",
      kind: "status",
      render: (item) => <StatusBadge tone={assessmentRatingTone(item.latest_assessment, item.risk.version)}>{assessmentRatingLabel(item.latest_assessment, item.risk.version)}</StatusBadge>,
      accessibleText: (item) => assessmentRatingLabel(item.latest_assessment),
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
      header: "Current assessment",
      render: (item) => item.latest_assessment
        ? <span className="risk-register__assessment"><strong>{assessmentKindLabel(item.latest_assessment.kind)}</strong><small>{item.latest_assessment.method_code} · {item.latest_assessment.method_version}</small></span>
        : "No current assessment",
      accessibleText: (item) => item.latest_assessment ? `${assessmentKindLabel(item.latest_assessment.kind)}, ${item.latest_assessment.method_code} ${item.latest_assessment.method_version}` : "No current assessment",
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
        <p>Risk assessments and appetite limits for {scope}.</p>
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
    {state === "live" && page.items.length > 0 && <section className="risk-register__position" aria-label="Loaded risk appetite position">
      <div className="risk-register__position-header">
        <div><span className="eyebrow">Risks</span><h2>Risk appetite</h2></div>
        <span>{page.items.length} shown{page.next_cursor ? " · More available" : ""}</span>
      </div>
      <StackedDistribution ariaLabel="Risk appetite for displayed records" segments={appetiteSegments}/>
      {(outsideCount > 0 || nearCount > 0) && <div className="risk-register__position-actions">
        {outsideCount > 0 && appetite !== "BREACHED" && <Button variant="secondary" size="compact" onPress={() => changeAppetite("BREACHED")}>Review {outsideCount} outside appetite</Button>}
        {nearCount > 0 && appetite !== "APPROACHING" && <Button variant="quiet" size="compact" onPress={() => changeAppetite("APPROACHING")}>Review {nearCount} near limit</Button>}
      </div>}
      <p>{page.next_cursor ? "More risks available. " : ""}Based on current assessments for the risks shown.</p>
    </section>}
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
