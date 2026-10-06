import { useEffect, useMemo, useState } from "react";
import { loadIndicatorPortfolio, type IndicatorPortfolioItem, type IndicatorPortfolioPage } from "../../indicatorInsightsApi";
import type { IndicatorKind, IndicatorState } from "../../indicatorTypes";
import type { MonitoringResult } from "../../monitoringTypes";
import { Button, DataTable, EmptyState, FocusedSheet, Notice, SearchField, SelectField, StatusBadge, type DataColumn } from "../ui";
import { IndicatorDetail } from "../indicators/IndicatorDetail";
import { IndicatorValue, indicatorValueAccessibleText } from "../indicators/IndicatorValue";
import { formatIndicatorDate, indicatorStateLabel, indicatorTone } from "../indicators/indicatorPresentation";
import "./insights.css";

type LoadState = "loading" | "live" | "unavailable";

type Props = {
  organizationName: string;
  legalEntityName: string;
  refreshToken?: string;
  onOpenProgram?: (programID: string) => void;
  loadPortfolio?: typeof loadIndicatorPortfolio;
  loadIndicatorResults?: (checkID: string, version?: number) => Promise<MonitoringResult[]>;
};

export function InsightsWorkspace({
  organizationName,
  legalEntityName,
  refreshToken,
  onOpenProgram,
  loadPortfolio = loadIndicatorPortfolio,
  loadIndicatorResults,
}: Props) {
  const [search, setSearch] = useState("");
  const [kind, setKind] = useState<IndicatorKind | "ALL">("ALL");
  const [condition, setCondition] = useState<IndicatorState | "ALL">("ALL");
  const [page, setPage] = useState<IndicatorPortfolioPage>();
  const [state, setState] = useState<LoadState>("loading");
  const [cursorStack, setCursorStack] = useState<string[]>([]);
  const [selected, setSelected] = useState<IndicatorPortfolioItem>();
  const [retry, setRetry] = useState(0);
  const cursor = cursorStack[cursorStack.length - 1];

  useEffect(() => {
    setCursorStack([]);
  }, [kind, condition, search]);

  useEffect(() => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      setState("loading");
      void loadPortfolio({
        kind: kind === "ALL" ? undefined : kind,
        state: condition === "ALL" ? undefined : condition,
        search,
        cursor,
        limit: 50,
      }, controller.signal).then((value) => {
        if (controller.signal.aborted) return;
        setPage(value);
        setState("live");
      }).catch((error: unknown) => {
        if (controller.signal.aborted || isAbortError(error)) return;
        setPage(undefined);
        setState("unavailable");
      });
    }, search.trim() ? 250 : 0);
    return () => {
      controller.abort();
      window.clearTimeout(timer);
    };
  }, [condition, cursor, kind, loadPortfolio, refreshToken, retry, search]);

  const columns = useMemo<readonly DataColumn<IndicatorPortfolioItem>[]>(() => [
    {
      id: "indicator",
      header: "Indicator",
      mobileLayout: "full-width",
      render: (item) => <span className="insights-indicator__stack"><strong>{item.kind} · {item.check_name}</strong><small>{item.check_code}</small></span>,
      accessibleText: (item) => `${item.kind}, ${item.check_name}, ${item.check_code}`,
    },
    {
      id: "risks",
      header: "Risk",
      mobileLayout: "full-width",
      render: (item) => <span className="insights-indicator__stack"><strong>{riskSummary(item)}</strong><small>{item.risks.length} linked {item.risks.length === 1 ? "risk" : "risks"}</small></span>,
      accessibleText: (item) => item.risks.length ? item.risks.map((risk) => risk.name).join(", ") : "No linked risk",
    },
    {
      id: "scope",
      header: "Program / owner",
      mobileLayout: "full-width",
      render: (item) => <span className="insights-indicator__stack"><strong>{item.program_name}</strong><small>{item.owner_display_name || "Owner not assigned"}</small></span>,
      accessibleText: (item) => `${item.program_name}; ${item.owner_display_name || "owner not assigned"}`,
    },
    {
      id: "value",
      header: "Value / limit",
      mobileLayout: "full-width",
      render: (item) => <IndicatorValue measurement={item.native_measurement} score={item.score} denominator={100}/>,
      accessibleText: (item) => indicatorValueAccessibleText(item.native_measurement, item.score, 100),
    },
    {
      id: "condition",
      header: "Condition",
      kind: "status",
      render: (item) => <StatusBadge tone={indicatorTone(item.state)}>{indicatorStateLabel(item.state)}</StatusBadge>,
      accessibleText: (item) => indicatorStateLabel(item.state),
    },
    {
      id: "updated",
      header: "Updated",
      mobileLayout: "full-width",
      render: (item) => <span className="insights-indicator__stack"><strong>{item.evaluated_at ? formatIndicatorDate(item.evaluated_at) : "No result"}</strong><small>{item.reason}</small></span>,
      accessibleText: (item) => `${item.evaluated_at ? formatIndicatorDate(item.evaluated_at) : "No result"}. ${item.reason}`,
    },
  ], []);

  const activeFilters = Number(kind !== "ALL") + Number(condition !== "ALL") + Number(search.trim() !== "");

  return <section className="insights-workspace">
    <header className="insights-header">
      <div>
        <span className="eyebrow">{organizationName} · {legalEntityName}</span>
        <h1>Insights</h1>
        <p>Indicators across the current legal entity. Native values stay separate from normalized concern.</p>
      </div>
      {page?.generated_at && <small>Updated {formatIndicatorDate(page.generated_at)}</small>}
    </header>

    <section className="insights-indicators" aria-labelledby="insights-indicators-heading">
      <div className="section-header">
        <div>
          <h2 id="insights-indicators-heading">KRI & KCI</h2>
          <p>Current governed indicators, their approved limits and linked risks.</p>
        </div>
        {activeFilters > 0 && <Button variant="quiet" size="compact" onPress={() => { setSearch(""); setKind("ALL"); setCondition("ALL"); }}>Clear filters</Button>}
      </div>

      <div className="insights-indicator__filters" aria-label="Indicator filters">
        <SearchField label="Search indicators" value={search} onChange={setSearch} placeholder="Indicator, Program or Risk"/>
        <SelectField
          label="Type"
          value={kind}
          placeholder="All types"
          allowsEmpty={false}
          options={[{ id: "ALL", label: "All types" }, { id: "KRI", label: "KRI" }, { id: "KCI", label: "KCI" }]}
          onChange={(value) => setKind((value ?? "ALL") as IndicatorKind | "ALL")}
        />
        <SelectField
          label="Condition"
          value={condition}
          placeholder="All conditions"
          allowsEmpty={false}
          options={[
            { id: "ALL", label: "All conditions" },
            { id: "BREACH", label: "Breach" },
            { id: "WATCH", label: "Watch" },
            { id: "NORMAL", label: "Normal" },
            { id: "UNKNOWN", label: "Unknown" },
          ]}
          onChange={(value) => setCondition((value ?? "ALL") as IndicatorState | "ALL")}
        />
      </div>

      {state === "unavailable" && <Notice tone="warning"><span>Indicator insights are unavailable.</span> <Button variant="secondary" size="compact" onPress={() => setRetry((value) => value + 1)}>Try again</Button></Notice>}

      {state === "loading" && !page && <p className="insights-indicator__status" role="status">Loading indicators…</p>}

      {page?.items.length ? <DataTable
        ariaLabel="KRI and KCI indicators"
        rows={page.items}
        rowKey={(item) => `${item.check_id}:${item.check_version}:${item.kind}`}
        rowName={(item) => `${item.kind}, ${item.check_name}, ${indicatorStateLabel(item.state)}`}
        columns={columns}
        onRowAction={setSelected}
        rowActionLabel="Open indicator"
        isLoading={state === "loading"}
        pagination={(cursorStack.length > 0 || page.next_cursor) ? {
          label: "Indicator pages",
          onPrevious: cursorStack.length > 0 ? () => setCursorStack((current) => current.slice(0, -1)) : undefined,
          onNext: page.next_cursor ? () => setCursorStack((current) => [...current, page.next_cursor!]) : undefined,
          isLoading: state === "loading",
        } : undefined}
      /> : state === "live" ? <EmptyState
        population="Indicators in this legal entity"
        title={activeFilters ? "No indicator matches these filters" : "No governed indicators"}
        description={activeFilters ? "Clear or change the filters to review another indicator." : "Indicators appear after an active Program monitoring check is linked to a Risk."}
      /> : null}
    </section>

    {selected && <FocusedSheet label={`${selected.kind} · ${selected.check_name}`} size="wide" onClose={() => setSelected(undefined)}>
      <IndicatorDetail indicator={selected} onOpenProgram={onOpenProgram} loadResults={loadIndicatorResults}/>
    </FocusedSheet>}
  </section>;
}

function riskSummary(item: IndicatorPortfolioItem) {
  if (!item.risks.length) return "No linked risk";
  if (item.risks.length === 1) return item.risks[0]!.name;
  return `${item.risks[0]!.name} · +${item.risks.length - 1}`;
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}
