import { useEffect, useMemo, useState } from "react";
import {
  createReportDefinition as createReportDefinitionRequest,
  listReportDefinitions as listReportDefinitionsRequest,
  transitionReportDefinition as transitionReportDefinitionRequest,
} from "../reportingApi";
import type {
  ReportDefinition,
  ReportDefinitionAction,
  ReportDefinitionTransitionInput,
  ReportDefinitionStatus,
} from "../reportingTypes";
import {
  buildReportSetupInput,
  reportSetupArea,
  reportSetupAreaLabel,
  reportSetupAreaOptions,
  reportSetupFocus,
  reportSetupFocusLabel,
  reportSetupFocusOptions,
  type ReportSetupArea,
  type ReportSetupFocus,
} from "./reports/reportTemplatePresets";
import { Button, DataTable, EmptyState, Notice, SelectField, StatusBadge, TextField, type DataColumn, type StatusTone } from "./ui";
import "./reports/reports.css";

type LoadState = "loading" | "live" | "error";

export type ReportingPageProps = {
  organizationName?: string;
  legalEntityName?: string;
  embedded?: boolean;
  onBack?: () => void;
  loadDefinitions?: typeof listReportDefinitionsRequest;
  createDefinition?: typeof createReportDefinitionRequest;
  transitionDefinition?: typeof transitionReportDefinitionRequest;
};

export function ReportingPage({
  organizationName,
  legalEntityName,
  embedded = false,
  onBack,
  loadDefinitions = listReportDefinitionsRequest,
  createDefinition = createReportDefinitionRequest,
  transitionDefinition = transitionReportDefinitionRequest,
}: ReportingPageProps) {
  const [definitions, setDefinitions] = useState<ReportDefinition[]>([]);
  const [state, setState] = useState<LoadState>("loading");
  const [loadError, setLoadError] = useState<string>();
  const [refreshKey, setRefreshKey] = useState(0);
  const [selectedID, setSelectedID] = useState<string>();
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [area, setArea] = useState<ReportSetupArea>("VENDORS");
  const [focus, setFocus] = useState<ReportSetupFocus>("OVERVIEW");
  const [command, setCommand] = useState<"idle" | "saving" | "transitioning">("idle");
  const [message, setMessage] = useState<string>();
  const [error, setError] = useState<string>();

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    setLoadError(undefined);
    void loadDefinitions(true, controller.signal).then((items) => {
      if (controller.signal.aborted) return;
      setDefinitions(items);
      setSelectedID((current) => items.some((item) => item.id === current) ? current : items.find((item) => item.status !== "RETIRED")?.id ?? items[0]?.id);
      setState("live");
    }).catch((reason: unknown) => {
      if (controller.signal.aborted || isAbortError(reason)) return;
      setState("error");
      setLoadError(readError(reason, "Saved report setups could not be loaded. Try again."));
    });
    return () => controller.abort();
  }, [loadDefinitions, refreshKey]);

  const selected = definitions.find((item) => item.id === selectedID);
  const activeCount = definitions.filter((item) => item.status === "ACTIVE" && item.effective).length;
  const approvalCount = definitions.filter((item) => item.status === "PENDING_REVIEW" || item.status === "REVIEWED").length;

  function beginCreate() {
    setName("");
    setArea("VENDORS");
    setFocus("OVERVIEW");
    setError(undefined);
    setMessage(undefined);
    setShowCreate(true);
  }

  async function saveSetup() {
    const trimmed = name.trim();
    if (trimmed.length < 3) {
      setError("Enter a setup name.");
      return;
    }
    setCommand("saving");
    setError(undefined);
    setMessage(undefined);
    try {
      const created = await createDefinition(buildReportSetupInput(trimmed, area, focus));
      setDefinitions((current) => [created, ...current.filter((item) => item.id !== created.id)]);
      setSelectedID(created.id);
      setShowCreate(false);
      setMessage(`${created.name} saved.`);
    } catch (reason: unknown) {
      setError(readCommandError(reason, "Couldn’t save setup. Retry."));
    } finally {
      setCommand("idle");
    }
  }

  async function transition(action: ReportDefinitionAction) {
    if (!selected) return;
    setCommand("transitioning");
    setError(undefined);
    setMessage(undefined);
    const input: ReportDefinitionTransitionInput = {
      expected_version: selected.version,
      checksum_seen: selected.checksum,
    };
    try {
      const updated = await transitionDefinition(selected.id, action, input);
      setDefinitions((current) => current.map((item) => item.id === updated.id ? updated : item));
      setMessage(transitionMessage(action));
    } catch (reason: unknown) {
      setError(readCommandError(reason, transitionFailure(action)));
    } finally {
      setCommand("idle");
    }
  }

  const columns = useMemo<readonly DataColumn<ReportDefinition>[]>(() => [
    {
      id: "name",
      header: "Report setup",
      render: (definition) => <span className="report-setup-name"><strong>{definition.name}</strong><small>{setupSummary(definition)}</small></span>,
      accessibleText: (definition) => `${definition.name}, ${setupSummary(definition)}`,
    },
    {
      id: "area",
      header: "Area",
      render: (definition) => reportSetupAreaLabel(reportSetupArea(definition)),
      accessibleText: (definition) => reportSetupAreaLabel(reportSetupArea(definition)),
    },
    {
      id: "focus",
      header: "Shows",
      render: (definition) => reportSetupFocusLabel(reportSetupFocus(definition)),
      accessibleText: (definition) => reportSetupFocusLabel(reportSetupFocus(definition)),
    },
    {
      id: "status",
      header: "Status",
      kind: "status",
      render: (definition) => <StatusBadge tone={setupStatusTone(definition.status)}>{setupStatusLabel(definition.status)}</StatusBadge>,
      accessibleText: (definition) => setupStatusLabel(definition.status),
    },
    {
      id: "updated",
      header: "Updated",
      render: (definition) => formatDate(definition.updated_at),
      accessibleText: (definition) => formatDate(definition.updated_at),
    },
  ], []);

  return <section className="report-setup-workspace" aria-label={embedded ? "Saved setups" : undefined} aria-labelledby={embedded ? undefined : "report-setup-heading"}>
    <header className="report-setup-header">
      <div>
        <span className="eyebrow">{organizationName || "ClearSight"} · {legalEntityName || "Current legal entity"}</span>
        <h2 id={embedded ? undefined : "report-setup-heading"}>Saved setups</h2>
      </div>
      <div className="report-setup-actions">
        {onBack && <Button variant="secondary" onPress={onBack}>Generated reports</Button>}
        <Button variant="secondary" onPress={() => setRefreshKey((value) => value + 1)} isLoading={state === "loading"}>Refresh</Button>
        <Button variant="primary" onPress={showCreate ? () => setShowCreate(false) : beginCreate}>{showCreate ? "Close" : "New setup"}</Button>
      </div>
    </header>

    <section className="report-setup-summary" aria-label="Setup summary">
      <div><span>Ready</span><strong>{activeCount}</strong></div>
      <div><span>Approval</span><strong>{approvalCount}</strong></div>
      <div><span>Total</span><strong>{definitions.filter((item) => item.status !== "RETIRED").length}</strong></div>
    </section>

    {message && <Notice tone="success"><span>{message}</span></Notice>}
    {error && <Notice tone="error"><span>{error}</span></Notice>}
    {state === "error" && <Notice tone="error"><span>{loadError}</span> <Button variant="secondary" size="compact" onPress={() => setRefreshKey((value) => value + 1)}>Retry</Button></Notice>}

    {showCreate && <section className="report-setup-create" aria-labelledby="new-report-setup-heading">
      <div className="report-setup-create__heading">
        <span className="eyebrow">New setup</span>
        <h3 id="new-report-setup-heading">New report setup</h3>
      </div>

      <TextField
        label="Name"
        value={name}
        onChange={(value) => { setName(value); setError(undefined); }}
        placeholder="e.g. Monthly vendor overview"
        maxLength={120}
        isRequired
      />

      <SelectField
        label="Area"
        value={area}
        placeholder="Choose an area"
        allowsEmpty={false}
        options={reportSetupAreaOptions.map((option) => ({ id: option.id, label: option.label, description: option.description }))}
        onChange={(value) => value && setArea(value as ReportSetupArea)}
      />

      <fieldset className="report-focus-picker">
        <legend>Report type</legend>
        <div>
          {reportSetupFocusOptions.map((option) => <button
            key={option.id}
            type="button"
            className="report-focus-option"
            data-selected={focus === option.id ? "true" : undefined}
            aria-pressed={focus === option.id}
            onClick={() => setFocus(option.id)}
          >
            <strong>{option.label}</strong>
            <span>{option.description}</span>
          </button>)}
        </div>
      </fieldset>

      <div className="report-setup-create__preview">
        <span>Output</span>
        <strong>{reportSetupAreaLabel(area)} · {reportSetupFocusLabel(focus)}</strong>
        <small>Summary · chart · supporting detail</small>
      </div>

      <div className="report-setup-create__footer">
        <Button variant="secondary" onPress={() => setShowCreate(false)}>Cancel</Button>
        <Button variant="primary" isLoading={command === "saving"} onPress={() => void saveSetup()}>Save setup</Button>
      </div>
    </section>}

    {state === "loading" && definitions.length === 0 && <p role="status" className="reports-library__loading">Loading setups…</p>}
    {state === "live" && definitions.length === 0 && !showCreate && <EmptyState
      population="saved setups"
      title="No saved setups"
      description="Create a setup to generate reports."
    />}

    {definitions.length > 0 && <div className="report-setup-layout">
      <DataTable
        ariaLabel="Saved setups"
        rows={definitions.filter((item) => item.status !== "RETIRED")}
        rowKey={(definition) => definition.id}
        rowName={(definition) => definition.name}
        columns={columns}
        selectedKey={selected?.id}
        onSelectionChange={(definition) => setSelectedID(definition.id)}
        onRowAction={(definition) => setSelectedID(definition.id)}
      />
      {selected && selected.status !== "RETIRED" && <SetupDetail
        definition={selected}
        busy={command === "transitioning"}
        onAction={(action) => void transition(action)}
      />}
    </div>}
  </section>;
}

function SetupDetail({ definition, busy, onAction }: { definition: ReportDefinition; busy: boolean; onAction: (action: ReportDefinitionAction) => void }) {
  const action = nextSetupAction(definition.status);
  return <aside className="report-setup-detail" aria-label="Selected report setup">
    <div className="report-setup-detail__heading">
      <div>
        <h3>{definition.name}</h3>
        <p>{setupSummary(definition)}</p>
      </div>
      <StatusBadge tone={setupStatusTone(definition.status)}>{setupStatusLabel(definition.status)}</StatusBadge>
    </div>

    <dl>
      <div><dt>Area</dt><dd>{reportSetupAreaLabel(reportSetupArea(definition))}</dd></div>
      <div><dt>Shows</dt><dd>{reportSetupFocusLabel(reportSetupFocus(definition))}</dd></div>
      <div><dt>Updated</dt><dd>{formatDate(definition.updated_at)}</dd></div>
    </dl>

    <p className="report-setup-detail__guidance">{setupGuidance(definition.status)}</p>

    <div className="report-setup-detail__actions">
      {action && <Button variant="primary" isLoading={busy} onPress={() => onAction(action.action)}>{action.label}</Button>}
      {definition.status === "ACTIVE" && <Button variant="quiet" isLoading={busy} onPress={() => onAction("retire")}>Retire setup</Button>}
    </div>
  </aside>;
}

function nextSetupAction(status: ReportDefinitionStatus): { action: ReportDefinitionAction; label: string } | undefined {
  if (status === "DRAFT") return { action: "submit", label: "Send for review" };
  if (status === "PENDING_REVIEW") return { action: "review", label: "Record review" };
  if (status === "REVIEWED") return { action: "activate", label: "Activate setup" };
  return undefined;
}

function setupGuidance(status: ReportDefinitionStatus) {
  if (status === "DRAFT") return "Send for review.";
  if (status === "PENDING_REVIEW") return "Awaiting review.";
  if (status === "REVIEWED") return "Ready to activate.";
  if (status === "ACTIVE") return "Ready to generate.";
  return "Retired.";
}

function setupSummary(definition: ReportDefinition) {
  return `${reportSetupAreaLabel(reportSetupArea(definition))} · ${reportSetupFocusLabel(reportSetupFocus(definition))}`;
}

function setupStatusLabel(status: ReportDefinitionStatus) {
  if (status === "DRAFT") return "Draft";
  if (status === "PENDING_REVIEW") return "Needs review";
  if (status === "REVIEWED") return "Ready to activate";
  if (status === "ACTIVE") return "Ready";
  return "Retired";
}

function setupStatusTone(status: ReportDefinitionStatus): StatusTone {
  if (status === "ACTIVE") return "success";
  if (status === "PENDING_REVIEW") return "warning";
  if (status === "REVIEWED") return "info";
  if (status === "RETIRED") return "neutral";
  return "unknown";
}

function transitionMessage(action: ReportDefinitionAction) {
  if (action === "submit") return "Sent for review.";
  if (action === "review") return "Review recorded.";
  if (action === "activate") return "Activated.";
  if (action === "retire") return "Setup retired.";
  return "Report setup updated.";
}

function transitionFailure(action: ReportDefinitionAction) {
  if (action === "review") return "Independent reviewer required.";
  if (action === "activate") return "Authorized approver required.";
  return "Action failed.";
}

function readCommandError(error: unknown, fallback: string) {
  if (typeof error === "object" && error !== null && "code" in error && error.code === "report_governance_blocked") return fallback;
  return readError(error, fallback);
}

function readError(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback;
}

function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(date);
}
