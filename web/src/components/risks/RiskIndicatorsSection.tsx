import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { loadProgramSummaries } from "../../api";
import { concernScoreText } from "../../concernScorePresentation";
import { loadMonitoringChecks } from "../../monitoringApi";
import type { MonitoringCheck } from "../../monitoringTypes";
import { linkRiskIndicator, type LinkRiskIndicatorResponse } from "../../riskApi";
import type { RiskIndicatorDetail, RiskIndicatorKind, RiskIndicatorLink, RiskRecord } from "../../riskTypes";
import type { ProgramSummary, SummaryPage } from "../../summaryTypes";
import { Button, DataTable, EmptyState, Notice, SearchField, SelectField, StatusBadge, type DataColumn, type StatusTone } from "../ui";

type Props = {
  risk: RiskRecord;
  actorID?: string;
  indicators: RiskIndicatorLink[];
  details: RiskIndicatorDetail[];
  detailsComplete: boolean;
  onReload: () => Promise<void>;
  onOpenProgram?: (programID: string) => void;
  onOpenMatter?: (matterID: string) => void;
  searchPrograms?: (query: string) => Promise<SummaryPage<ProgramSummary>>;
  loadChecks?: (programID: string) => Promise<MonitoringCheck[]>;
  linkIndicator?: (
    riskID: string,
    expectedRiskVersion: number,
    monitoringCheckID: string,
    monitoringCheckVersion: number,
    kind: RiskIndicatorKind,
  ) => Promise<LinkRiskIndicatorResponse>;
};

type CandidateState = "idle" | "loading" | "live" | "error";

function searchIndicatorPrograms(query: string) {
  return loadProgramSummaries({ q: query, status: "ACTIVE", limit: 20 });
}

export function RiskIndicatorsSection({
  risk,
  actorID,
  indicators,
  details,
  detailsComplete,
  onReload,
  onOpenProgram,
  onOpenMatter,
  searchPrograms = searchIndicatorPrograms,
  loadChecks = loadMonitoringChecks,
  linkIndicator = linkRiskIndicator,
}: Props) {
  const [linkMode, setLinkMode] = useState(false);
  const [query, setQuery] = useState("");
  const [programState, setProgramState] = useState<CandidateState>("idle");
  const [programPage, setProgramPage] = useState<SummaryPage<ProgramSummary>>({ items: [], generated_at: "" });
  const [selectedProgramID, setSelectedProgramID] = useState<string>();
  const [checkState, setCheckState] = useState<CandidateState>("idle");
  const [checks, setChecks] = useState<MonitoringCheck[]>([]);
  const [selectedCheckKey, setSelectedCheckKey] = useState<string>();
  const [kind, setKind] = useState<RiskIndicatorKind>("KRI");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const canLink = Boolean(actorID && risk.owner_principal_id && actorID === risk.owner_principal_id);
  const linkedKeys = useMemo(
    () => new Set(indicators.map((item) => checkKey(item.monitoring_check_id, item.monitoring_check_version))),
    [indicators],
  );
  const eligibleChecks = useMemo(
    () => checks.filter((check) => check.status === "ACTIVE" && check.is_current && !linkedKeys.has(checkKey(check.id, check.version))),
    [checks, linkedKeys],
  );

  useEffect(() => {
    if (!linkMode) return;
    let active = true;
    const timer = window.setTimeout(() => {
      setProgramState("loading");
      setError("");
      void searchPrograms(query.trim()).then((page) => {
        if (!active) return;
        setProgramPage(page);
        setProgramState("live");
      }).catch((caught: unknown) => {
        if (!active) return;
        setProgramPage({ items: [], generated_at: "" });
        setProgramState("error");
        setError(caught instanceof Error ? caught.message : "Programs could not be loaded.");
      });
    }, 250);
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [linkMode, query, searchPrograms]);

  useEffect(() => {
    if (!selectedProgramID) {
      setChecks([]);
      setCheckState("idle");
      setSelectedCheckKey(undefined);
      return;
    }
    let active = true;
    setCheckState("loading");
    setError("");
    void loadChecks(selectedProgramID).then((values) => {
      if (!active) return;
      setChecks(values);
      const first = values.find((check) => check.status === "ACTIVE" && check.is_current && !linkedKeys.has(checkKey(check.id, check.version)));
      setSelectedCheckKey(first ? checkKey(first.id, first.version) : undefined);
      setCheckState("live");
    }).catch((caught: unknown) => {
      if (!active) return;
      setChecks([]);
      setSelectedCheckKey(undefined);
      setCheckState("error");
      setError(caught instanceof Error ? caught.message : "Monitoring checks could not be loaded.");
    });
    return () => { active = false; };
  }, [linkedKeys, loadChecks, selectedProgramID]);

  const programOptions = programPage.items.map(({ program }) => ({
    id: program.id,
    label: `${program.name} · ${program.code}`,
    description: program.status === "ACTIVE" ? undefined : program.status,
  }));
  const checkOptions = eligibleChecks.map((check) => ({
    id: checkKey(check.id, check.version),
    label: `${check.name} · ${check.code}`,
    description: `${check.input_kind === "SOURCE" ? "Connected data" : "Form"} · v${check.version}`,
  }));
  const selectedCheck = eligibleChecks.find((check) => checkKey(check.id, check.version) === selectedCheckKey);

  const columns: readonly DataColumn<RiskIndicatorDetail>[] = [
    {
      id: "indicator",
      header: "Indicator",
      mobileLayout: "full-width",
      render: (item) => <span className="risk-record__stack"><strong>{item.link.kind} · {item.check_name}</strong><small>{item.check_code}</small><small>{item.owner_display_name ? `Owner: ${item.owner_display_name}` : "Owner name unavailable"}</small></span>,
      accessibleText: (item) => `${item.link.kind}, ${item.check_name}, ${item.check_code}. ${item.owner_display_name ? `Owner: ${item.owner_display_name}` : "Owner name unavailable"}`,
    },
    {
      id: "program",
      header: "Program",
      render: (item) => item.program_name,
      accessibleText: (item) => item.program_name,
    },
    {
      id: "state",
      header: "State",
      kind: "status",
      render: (item) => <StatusBadge tone={indicatorTone(item.state)}>{indicatorStateLabel(item.state)}</StatusBadge>,
      accessibleText: (item) => indicatorStateLabel(item.state),
    },
    {
      id: "score",
      header: "Concern score",
      kind: "number",
      render: (item) => <span className="risk-record__stack"><span>{concernScoreText(item.score, "compact")}</span>{item.state === "UNKNOWN" && item.score != null && Number.isFinite(item.score) && item.score >= 0 && item.score <= 100 && <small>Last recorded</small>}</span>,
      accessibleText: (item) => `${concernScoreText(item.score)}${item.state === "UNKNOWN" ? ". Current state unknown." : ""}`,
    },
    {
      id: "coverage",
      header: "Coverage",
      render: (item) => item.coverage === undefined ? "—" : `${formatPercent(item.coverage)} · min ${formatPercent(item.minimum_coverage)}`,
      accessibleText: (item) => item.coverage === undefined ? "No current coverage" : `${formatPercent(item.coverage)}, minimum ${formatPercent(item.minimum_coverage)}`,
    },
    {
      id: "freshness",
      header: "Current result",
      mobileLayout: "full-width",
      render: (item) => <span className="risk-record__stack"><strong>{item.evaluated_at ? formatIndicatorDate(item.evaluated_at) : "No result"}</strong><small>{item.reason}</small></span>,
      accessibleText: (item) => `${item.evaluated_at ? formatIndicatorDate(item.evaluated_at) : "No result"}. ${item.reason}`,
    },
    {
      id: "intervention",
      header: "Intervention",
      mobileLayout: "full-width",
      render: (item) => item.intervention
        ? <span className="risk-record__intervention"><span className="risk-record__stack"><strong>{item.intervention.reference}</strong><small>{matterStatusLabel(item.intervention.status)}</small></span>{onOpenMatter && <Button variant="secondary" size="compact" onPress={() => onOpenMatter(item.intervention!.matter_id)}>Open issue</Button>}</span>
        : "No open issue",
      accessibleText: (item) => item.intervention ? `${item.intervention.reference}, ${matterStatusLabel(item.intervention.status)}` : "No open issue",
    },
  ];

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!selectedCheck || busy) return;
    setBusy(true);
    setError("");
    try {
      await linkIndicator(risk.id, risk.version, selectedCheck.id, selectedCheck.version, kind);
      await onReload();
      closeLinker();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The indicator could not be linked.");
    } finally {
      setBusy(false);
    }
  }

  function closeLinker() {
    setLinkMode(false);
    setQuery("");
    setSelectedProgramID(undefined);
    setSelectedCheckKey(undefined);
    setChecks([]);
    setError("");
    setProgramState("idle");
    setCheckState("idle");
  }

  return <section className="risk-record__history" aria-labelledby="risk-indicators-heading">
    <div className="section-header">
      <div>
        <h2 id="risk-indicators-heading">Indicators</h2>
        <p>Linked indicators and their latest results.</p>
      </div>
      {canLink && !linkMode && <Button variant="secondary" size="compact" onPress={() => setLinkMode(true)}>Link indicator</Button>}
    </div>

    {linkMode && <div className="risk-record__link-form">
      <div><h3>Link existing indicator source</h3><p>Choose a Program and one active monitoring check. ClearSight uses its approved thresholds, freshness and coverage rules.</p></div>
      <SearchField label="Search Programs" value={query} onChange={setQuery} placeholder="Search Programs" isLoading={programState === "loading"}/>
      {programState === "error" && <Notice tone="error">{error || "Programs could not be loaded."}</Notice>}
      {programState === "live" && programPage.next_cursor && <Notice tone="info">Showing the first 20 matching Programs. Refine the search to find another Program.</Notice>}
      {programState === "live" && programOptions.length === 0 && <p>No matching Programs are available in this legal entity.</p>}
      {programOptions.length > 0 && <SelectField
        label="Program"
        value={selectedProgramID}
        placeholder="Choose a Program"
        options={programOptions}
        onChange={(value) => { setSelectedProgramID(value); setSelectedCheckKey(undefined); }}
      />}
      {checkState === "loading" && <p role="status">Loading monitoring checks…</p>}
      {checkState === "error" && <Notice tone="error">{error || "Monitoring checks could not be loaded."}</Notice>}
      {checkState === "live" && checkOptions.length === 0 && <p>No unlinked active monitoring checks are available for this Program.</p>}
      {checkOptions.length > 0 && <form onSubmit={(event) => void submit(event)}>
        <SelectField label="Monitoring check" value={selectedCheckKey} placeholder="Choose a check" options={checkOptions} onChange={setSelectedCheckKey}/>
        <SelectField<RiskIndicatorKind>
          label="Indicator type"
          value={kind}
          placeholder="Choose a type"
          options={[
            { id: "KRI", label: "KRI · Risk indicator" },
            { id: "KCI", label: "KCI · Control indicator" },
          ]}
          onChange={(value) => setKind(value ?? "KRI")}
          allowsEmpty={false}
        />
        {selectedCheck && <p className="risk-indicator-link__contract">
          Concern score: <strong>0–100 points</strong>. Check v{selectedCheck.version}, minimum coverage {formatPercent(selectedCheck.minimum_coverage)}, freshness {selectedCheck.freshness_minutes} minutes.
        </p>}
        {error && <Notice tone="error">{error}</Notice>}
        <div className="risk-record__link-actions">
          <Button type="submit" isDisabled={busy || !selectedCheckKey}>{busy ? "Linking…" : "Link indicator"}</Button>
          <Button variant="secondary" type="button" onPress={closeLinker}>Cancel</Button>
        </div>
      </form>}
      {checkOptions.length === 0 && checkState !== "loading" && <Button variant="secondary" size="compact" onPress={closeLinker}>Close</Button>}
    </div>}

    {!detailsComplete && <Notice tone="warning">Some linked Indicator details are unavailable in the current scope.</Notice>}
    {details.length ? <DataTable
      ariaLabel="Risk indicators"
      rows={details}
      rowKey={(item) => item.link.id}
      rowName={(item) => `${item.link.kind}, ${item.check_name}, ${indicatorStateLabel(item.state)}`}
      columns={columns}
      onRowAction={onOpenProgram ? (item) => onOpenProgram(item.program_id) : undefined}
      rowActionLabel="Open Program"
    /> : <EmptyState population={risk.name} title="No linked indicators" description="No governed Program monitoring check is linked to this risk."/>}
  </section>;
}

function checkKey(id: string, version: number) {
  return `${id}::${version}`;
}

function indicatorStateLabel(state: RiskIndicatorDetail["state"]) {
  if (state === "NORMAL") return "Normal";
  if (state === "WATCH") return "Watch";
  if (state === "BREACH") return "Breach";
  return "Unknown";
}

function indicatorTone(state: RiskIndicatorDetail["state"]): StatusTone {
  if (state === "NORMAL") return "success";
  if (state === "WATCH") return "warning";
  if (state === "BREACH") return "error";
  return "unknown";
}

function formatPercent(value: number) {
  return `${Math.round(value * 100)}%`;
}

function formatIndicatorDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return value;
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date);
}


function matterStatusLabel(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}
