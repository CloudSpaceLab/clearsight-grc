import { useEffect, useState } from "react";
import { listLosses, type LossListParams } from "../../lossApi";
import { summarizeLoadedLossExposure } from "../../lossExposurePresentation";
import type { LossEventType, LossPage, LossRecoveryStatus, LossStatus, LossSummary } from "../../lossTypes";
import { Button, DataTable, EmptyState, FilterBar, Notice, RankedBarList, SearchField, SelectField, StatusBadge, TextField, type DataColumn } from "../ui";
import { formatLossDate, formatLossMoney, lossEventLabel, lossEventOptions, lossStatusLabel, lossStatusTone, recoveryStatusLabel, recoveryStatusTone } from "./lossPresentation";

type Props = {
  organizationName?: string;
  legalEntityName?: string;
  organizationScopeID?: string;
  organizationScopeName?: string;
  onOpenLoss: (id: string) => void;
  onRecordLoss?: () => void;
  loadPage?: (params?: LossListParams, signal?: AbortSignal) => Promise<LossPage>;
};

type LoadState = "loading" | "live" | "error";

const statusOptions: ReadonlyArray<{ id: LossStatus; label: string }> = [
  { id: "ACTIVE", label: "Active" },
  { id: "VOIDED", label: "Voided" },
];

const recoveryOptions: ReadonlyArray<{ id: LossRecoveryStatus; label: string }> = [
  { id: "NONE", label: "No recovery" },
  { id: "PARTIAL", label: "Partly recovered" },
  { id: "FULL", label: "Fully recovered" },
];

export function LossRegister({
  organizationName,
  legalEntityName,
  organizationScopeID,
  organizationScopeName,
  onOpenLoss,
  onRecordLoss,
  loadPage = listLosses,
}: Props) {
  const scope = organizationScopeName || legalEntityName || "this legal entity";
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<LossStatus>();
  const [recoveryStatus, setRecoveryStatus] = useState<LossRecoveryStatus>();
  const [eventType, setEventType] = useState<LossEventType>();
  const [currency, setCurrency] = useState("");
  const [cursors, setCursors] = useState<string[]>([]);
  const [page, setPage] = useState<LossPage>({ items: [] });
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
      recoveryStatus,
      eventType,
      currency: currency.trim().toUpperCase() || undefined,
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
  }, [currency, currentCursor, eventType, loadPage, organizationScopeID, recoveryStatus, retry, search, status]);

  const hasFilters = Boolean(search.trim() || status || recoveryStatus || eventType || currency.trim());
  const exposure = state === "live" && page.items.length > 0 ? summarizeLoadedLossExposure(page.items) : undefined;
  const resetPage = () => setCursors([]);
  const pagination = cursors.length > 0 || page.next_cursor ? {
    label: "Loss register pages",
    previousLabel: "Load previous page",
    nextLabel: "Load next page",
    onPrevious: cursors.length > 0 ? () => setCursors((value) => value.slice(0, -1)) : undefined,
    onNext: page.next_cursor ? () => setCursors((value) => [...value, page.next_cursor!]) : undefined,
    isLoading: state === "loading",
  } : undefined;

  const columns: readonly DataColumn<LossSummary>[] = [
    {
      id: "loss",
      header: "Loss",
      mobileLayout: "full-width",
      render: ({ loss }) => <span className="loss-register__identity"><strong>{loss.title}</strong><small>{loss.code} · {lossEventLabel(loss.event_type)}</small></span>,
      accessibleText: ({ loss }) => `${loss.title}, ${loss.code}, ${lossEventLabel(loss.event_type)}`,
    },
    {
      id: "impact",
      header: "Financial impact",
      render: ({ totals }) => <span className="loss-register__money"><strong>{formatLossMoney(totals.net_loss_minor, totals.currency)} net</strong><small>Gross {formatLossMoney(totals.gross_amount_minor, totals.currency)} · Recovered {formatLossMoney(totals.recovered_amount_minor, totals.currency)}</small></span>,
      accessibleText: ({ totals }) => `Net ${formatLossMoney(totals.net_loss_minor, totals.currency)}, gross ${formatLossMoney(totals.gross_amount_minor, totals.currency)}, recovered ${formatLossMoney(totals.recovered_amount_minor, totals.currency)}`,
    },
    {
      id: "recovery",
      header: "Recovery",
      kind: "status",
      render: ({ totals }) => <StatusBadge tone={recoveryStatusTone(totals.recovery_status)}>{recoveryStatusLabel(totals.recovery_status)}</StatusBadge>,
      accessibleText: ({ totals }) => recoveryStatusLabel(totals.recovery_status),
    },
    {
      id: "occurred",
      header: "Occurred",
      render: ({ loss }) => formatLossDate(loss.occurred_at),
      accessibleText: ({ loss }) => formatLossDate(loss.occurred_at),
    },
    {
      id: "status",
      header: "Status",
      kind: "status",
      render: ({ loss }) => <StatusBadge tone={lossStatusTone(loss.status)}>{lossStatusLabel(loss.status)}</StatusBadge>,
      accessibleText: ({ loss }) => lossStatusLabel(loss.status),
    },
  ];

  return <section className="loss-register-page" aria-labelledby="loss-register-heading">
    <header className="topbar loss-page-header">
      <div>
        <span className="eyebrow">{organizationName || "Risk portfolio"}</span>
        <h1 id="loss-register-heading">Losses</h1>
        <p>Recorded losses and recoveries for {scope}.</p>
      </div>
      {onRecordLoss && <div className="topbar-actions"><Button onPress={onRecordLoss}>Record loss</Button></div>}
    </header>

    <FilterBar
      label="Loss filters"
      fields={<>
        <SearchField label="Search losses" value={search} onChange={(value) => { setSearch(value); resetPage(); }} placeholder="Title, code, cause or description" isLoading={state === "loading"}/>
        <SelectField label="Status" value={status} placeholder="All statuses" options={statusOptions} onChange={(value) => { setStatus(value); resetPage(); }}/>
        <SelectField label="Recovery" value={recoveryStatus} placeholder="All recovery states" options={recoveryOptions} onChange={(value) => { setRecoveryStatus(value); resetPage(); }}/>
        <SelectField label="Event type" value={eventType} placeholder="All event types" options={lossEventOptions} onChange={(value) => { setEventType(value); resetPage(); }}/>
        <TextField label="Currency" value={currency} onChange={(value) => { setCurrency(value.toUpperCase()); resetPage(); }} placeholder="All currencies" maxLength={3}/>
      </>}
      resultCount={state === "live" ? page.items.length : undefined}
      resultLabel={(count) => `${count} shown`}
      onClear={hasFilters ? () => { setSearch(""); setStatus(undefined); setRecoveryStatus(undefined); setEventType(undefined); setCurrency(""); resetPage(); } : undefined}
    />

    {state === "loading" && page.items.length === 0 && <p className="loss-load-state" role="status">Loading losses…</p>}
    {state === "error" && <Notice tone="error"><span>Loss register unavailable.</span> <Button variant="secondary" size="compact" onPress={() => setRetry((value) => value + 1)}>Try again</Button></Notice>}
    {state === "live" && page.items.length === 0 && <EmptyState
      population={hasFilters ? `${scope} · current loss filters` : `${scope} · current loss register`}
      title={hasFilters ? "No matching losses" : "No losses in this scope"}
      description={hasFilters ? "Change the filters or search." : "No operational losses were returned for this legal entity."}
    />}
    {exposure && <section className="loss-exposure" aria-label="Loaded loss financial exposure">
      <div className="loss-exposure__heading">
        <div><span className="eyebrow">Losses</span><h2>Financial impact</h2></div>
        <span>{page.items.length} shown{page.next_cursor ? " · More available" : ""}</span>
      </div>
      <p>Active losses shown, by currency. No currency conversion.</p>
      {exposure.groups.length > 0
        ? <div className="loss-exposure__groups">
            {exposure.groups.map((group) => <section className="loss-exposure__currency" key={group.currency} aria-label={group.currency + " loss exposure"}>
              <div className="loss-exposure__currency-heading"><h3>{group.currency}</h3><small>{group.count} loss{group.count === 1 ? "" : "es"}</small></div>
              {currency !== group.currency && <Button variant="quiet" size="compact" onPress={() => { setCurrency(group.currency); resetPage(); }}>Show {group.currency} losses</Button>}
              <RankedBarList ariaLabel={group.currency + " financial exposure from loaded losses"} items={group.rows}/>
            </section>)}
          </div>
        : <p>No valid loss amounts in the records shown.</p>}
      {(exposure.moreCurrencies > 0 || exposure.excluded > 0 || exposure.voided > 0) && <p className="loss-exposure__limits">
        {exposure.moreCurrencies > 0 && exposure.moreCurrencies + " other currencies. Filter by currency to review. "}
        {exposure.excluded > 0 && exposure.excluded + " records omitted from totals; check their amounts below. "}
        {exposure.voided > 0 && exposure.voided + " voided loss" + (exposure.voided === 1 ? "" : "es") + " not counted."}
      </p>}
    </section>}
    {page.items.length > 0 && <DataTable
      ariaLabel="Operational loss register"
      rows={page.items}
      rowKey={(item) => item.loss.id}
      rowName={(item) => `${item.loss.title}, ${item.loss.code}, ${formatLossMoney(item.totals.net_loss_minor, item.totals.currency)} net`}
      columns={columns}
      onRowAction={(item) => onOpenLoss(item.loss.id)}
      rowActionLabel="Open loss"
      isLoading={state === "loading"}
      pagination={pagination}
    />}
  </section>;
}

function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}
