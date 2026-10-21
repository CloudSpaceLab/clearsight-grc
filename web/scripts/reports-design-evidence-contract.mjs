import { reportsDesignEvidenceScenarios, reportsDesignViewports } from "./reports-design-evidence-scenarios.mjs";

const stableTabs = ["Overview", "Explore", "Reports", "Schedules"];
const same = (first, second) => JSON.stringify(first) === JSON.stringify(second);
const rectangle = (value) => value && ["x", "y", "width", "height", "right", "bottom"].every((key) => Number.isFinite(value[key]))
  && value.width > 0 && value.height > 0 && Math.abs(value.x + value.width - value.right) <= 1 && Math.abs(value.y + value.height - value.bottom) <= 1;

export function reportExploreFixtureSource(fixture, isRisk, scoped = false) {
  const prefix = fixture === "report-explore-empty" ? isRisk ? "8f720000" : "8f7a0000"
    : fixture === "report-explore-mixed-currency" ? isRisk ? "8f730000" : "8f7b0000" : isRisk ? "8f710000" : "8f790000";
  return `${prefix}-0000-4000-8000-00000000000${scoped ? 2 : 1}`;
}

function validBuilderCommands(flow, blocked) {
  const commands = flow?.commands;
  if (!Array.isArray(commands) || commands.length !== (blocked ? 3 : 5) || commands.some((command) => command.method !== "POST")) return false;
  const create = commands[0];
  const definition = create.result;
  const filters = ["AT_RISK", "GAP_IDENTIFIED", "EVIDENCE_INSUFFICIENT", "IMPLEMENTATION_PENDING", "OVERDUE", "UNDER_REVIEW", "UNKNOWN"];
  if (create.path !== "/api/v1/reports/definitions" || create.status !== 201
      || create.body?.name !== "Sample Program exceptions" || create.body.dataset !== "PROGRAMS" || create.body.scope_kind !== "LEGAL_ENTITY" || create.body.format !== "XLSX"
      || create.body.filter?.kind !== "group" || create.body.filter.operator !== "or"
      || !same(create.body.filter.children?.map((item) => [item.kind, item.field, item.operator, item.value]), filters.map((value) => ["condition", "overall_state", "is", value]))
      || definition?.id !== "sample-report-builder-1" || definition.name !== create.body.name || definition.status !== "DRAFT"
      || definition.version !== 1 || definition.current_version !== 1 || !/^[0-9a-f]{64}$/.test(definition.checksum ?? "")) return false;
  const actions = blocked ? ["submit", "review"] : ["submit", "review", "activate"];
  const statuses = ["PENDING_REVIEW", "REVIEWED", "ACTIVE"];
  if (!actions.every((action, index) => {
    const command = commands[index + 1];
    return command.path === `/api/v1/reports/definitions/${definition.id}/${action}`
      && command.body?.expected_version === index + 1 && command.body?.checksum_seen === definition.checksum
      && (blocked && action === "review" ? command.status === 403 && command.result?.error?.code === "report_governance_blocked"
        : command.status === 200 && command.result?.id === definition.id && command.result?.status === statuses[index]
          && command.result?.version === index + 2 && command.result?.current_version === 1 && command.result?.checksum === definition.checksum);
  })) return false;
  if (blocked) return true;
  const run = commands.at(-1);
  return run.path === "/api/v1/reports/runs" && run.status === 201 && run.body?.definition_id === definition.id
    && run.body?.expected_definition_version === 1 && same(run.body?.parameters, { start_date: "2026-09-01", end_date: "2026-09-30", owner_principal_id: "owner-program" })
    && run.result?.id === "sample-report-builder-run-1" && run.result?.status === "QUEUED"
    && run.result?.definition_id === definition.id && run.result?.definition_version === 1 && run.result?.definition_checksum === definition.checksum;
}

export function reviewReportsDesignCaptures(captures) {
  const failures = [];
  const byName = new Map(captures.map((capture) => [capture.name, capture]));
  const missing = reportsDesignEvidenceScenarios.filter((scenario) => !byName.has(scenario.name));
  if (missing.length) failures.push(`Reports design state evidence is missing: ${missing.map(({ state }) => state).join(", ")}`);
  for (const view of ["explore", "builder"]) {
    const matrix = reportsDesignEvidenceScenarios.filter((scenario) => scenario.reportDesignMatrix && scenario[`openReport${view === "explore" ? "Explore" : "Builder"}`]);
    const present = matrix.filter((scenario) => byName.has(scenario.name));
    const expectedVariants = ["light", "dark"].flatMap((theme) => reportsDesignViewports.map(({ width, height }) => `${theme}-${width}x${height}`));
    const actualVariants = present.map((scenario) => byName.get(scenario.name)).map((capture) => `${capture.theme}-${capture.viewport?.width}x${capture.viewport?.height}`);
    if (present.length !== 10 || new Set(actualVariants).size !== 10 || expectedVariants.some((variant) => !actualVariants.includes(variant))) {
      failures.push(`Reports ${view === "explore" ? "Explore" : "Builder"} requires both themes at 1440, 1280, 1024, 768 and 390px`);
    }
  }
  for (const scenario of reportsDesignEvidenceScenarios) {
    const capture = byName.get(scenario.name);
    if (!capture) continue; // Registry absence is also reported by the full flow manifest.
    const layout = capture.reportsLayout;
    if (["route", "fixture", "state", "theme", "density"].some((key) => capture[key] !== scenario[key])
        || capture.viewport?.width !== scenario.viewport.width || capture.viewport?.height !== scenario.viewport.height) {
      failures.push(`${scenario.name}: Reports design evidence metadata does not match its registered state`);
    }
    if (scenario.reportBuilderFlow) {
      const flow = layout?.flow;
      const blocked = scenario.fixture === "report-builder-blocked";
      if (layout?.kind !== "builder-flow" || !flow || !validBuilderCommands(flow, blocked)
          || flow.evidenceMode !== "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION" || flow.definitionID !== "sample-report-builder-1"
          || flow.definitionName !== "Sample Program exceptions" || flow.dataset !== "PROGRAMS" || flow.format !== "XLSX" || flow.scopeKind !== "LEGAL_ENTITY"
          || flow.saveCount !== 1 || flow.checksumRetained !== true || flow.versionChecksMatched !== true || flow.savedDefinitionRetained !== true
          || !same(flow.actions, blocked ? ["create", "submit", "review"] : ["create", "submit", "review", "activate", "generate"])
          || !same(flow.commandStatuses, blocked ? [201, 200, 403] : [201, 200, 200, 200, 201])
          || flow.finalStatus !== (blocked ? "PENDING_REVIEW" : "ACTIVE")
          || (blocked ? flow.blockedReviewPreservedSetup !== true || flow.runID !== null
            : flow.generationHandoff !== true || flow.runID !== "sample-report-builder-run-1" || flow.runStatus !== "QUEUED"
              || flow.generationFocusContained !== true || flow.dateFiltersAreRecordCreation !== true || flow.queuedRecordVisible !== true)) {
        failures.push(`${scenario.name}: real setup mutation, governance rejection or generation handoff evidence is missing or inconsistent`);
      }
      continue;
    }
    const kind = scenario.openReportExplore ? "explore" : "builder";
    const expectedTabs = [...stableTabs, ...(kind === "builder" ? ["Build report"] : [])];
    if (layout?.kind !== kind || !same(layout.localTabs, expectedTabs) || layout.selectedTab !== (kind === "builder" ? "Build report" : "Explore")
        || !rectangle(layout.heading) || !rectangle(layout.tabs) || layout.tabs.bottom > layout.heading.y + 2
        || !rectangle(layout.selectedTabBounds) || layout.selectedTabVisible !== true
        || layout.selectedTabBounds.x < Math.max(0, layout.tabs.x) - 1 || layout.selectedTabBounds.right > Math.min(scenario.viewport.width, layout.tabs.right) + 1
        || !rectangle(layout.firstControl) || layout.firstControl.y < 0 || layout.firstControl.bottom > scenario.viewport.height
        || layout.firstUsefulContentVisible !== true || !(layout.horizontalOverflow <= 1)
        || (scenario.touch && (!(layout.minTargetWidth >= 44) || !(layout.minTargetHeight >= 44)))) {
      failures.push(`${scenario.name}: Reports local tabs, measured first content or touch geometry is missing or inconsistent`);
      continue;
    }
    if (kind === "explore") {
      const beside = scenario.viewport.width > 700;
      const filterWidth = scenario.viewport.width <= 900 ? 200 : 240;
      if (!rectangle(layout.filters) || !rectangle(layout.chart) || layout.placement !== (beside ? "beside" : "above")
          || layout.heading.bottom > layout.filters.y + 1
          || (beside ? Math.abs(layout.filterWidth - filterWidth) > 1 || Math.abs(layout.filters.width - filterWidth) > 1
            || layout.chart.x < layout.filters.right || Math.abs(layout.chart.y - layout.filters.y) > 2
            : layout.chart.y < layout.filters.bottom)
          || (layout.table && (!rectangle(layout.table) || layout.table.y < layout.chart.bottom))) {
        failures.push(`${scenario.name}: Explore Data rail, chart and exact breakdown lack source replacement geometry`);
      }
      const measure = scenario.reportExploreMeasure ?? "Risks outside appetite";
      const isRisk = measure === "Risks outside appetite";
      const unavailable = ["report-definitions", "report-overview-denied"].includes(scenario.fixture)
        || scenario.fixture === "report-overview-partial" && !isRisk;
      const empty = scenario.fixture === "report-explore-empty";
      const mixed = scenario.fixture === "report-explore-mixed-currency";
      const scope = scenario.reportOrganizationScope ?? null;
      const total = isRisk ? empty ? "0" : scope ? "2" : "4" : measure === "Loss events" ? empty ? "0" : "1"
        : empty ? "—" : mixed ? scenario.reportExploreCurrency === "USD" ? "-$4.50" : "₦9,007,199,254,740,993.01" : "₦10,400,000.00";
      const rows = empty ? [] : isRisk ? [{ label: "Risk Operations", value: "2" }, ...(!scope ? [{ label: "Legal entity", value: "2" }] : [])]
        : [{ label: "Risk Operations", value: total }];
      if (layout.measure !== measure || layout.chartLabel !== `${measure} chart` || layout.organizationScopeID !== scope || layout.exactValuesContained !== true
          || layout.timeBasis !== (isRisk ? "CURRENT_POSTURE" : "PERIOD_FLOW") || layout.unsupportedControlsDisabled !== true
          || (unavailable ? !["unavailable", "denied"].includes(layout.chartState) || layout.total !== null || layout.table
            || layout.chartRows?.length || layout.tableRows?.length
            : layout.chartState !== "live" || layout.total !== total || !layout.table || !same(layout.chartRows, rows) || !same(layout.tableRows, rows)
              || layout.sourceConsistent !== true || layout.sourceID !== reportExploreFixtureSource(scenario.fixture, isRisk, Boolean(scope))
              || layout.definitionRevision !== (isRisk ? "enterprise-domain-v1" : "operational-loss-period-v1"))) {
        failures.push(`${scenario.name}: Explore chosen measure, exact values, source or unavailable semantics are missing or inconsistent`);
      }
      if (!unavailable && isRisk && (layout.dataQuality !== (scenario.fixture === "report-overview-partial" ? "partial" : "complete")
          || !layout.coverageText?.includes(`${scope ? 5 : 12} assessed`)
          || (scenario.fixture === "report-overview-partial" && !layout.coverageText.includes("Risk coverage is partial.")))) {
        failures.push(`${scenario.name}: Explore assessed population and coverage disclosure are missing`);
      }
      if (empty && (layout.emptyStateConfirmed !== true || layout.emptyHeading !== (isRisk ? "No risks outside appetite" : measure === "Loss events" ? "No new loss events" : "No monetary loss flow"))) {
        failures.push(`${scenario.name}: a real empty measure must be distinguished from unavailable data`);
      }
      if (mixed && (layout.currency !== scenario.reportExploreCurrency || layout.currencyAmountsSeparate !== true
          || !layout.coverageText?.includes("Amounts are not converted or combined."))) {
        failures.push(`${scenario.name}: mixed-currency analysis must show one exact currency without combining amounts`);
      }
      if (scenario.verifyReportCurrency && (layout.currencyInteraction?.selectionPreserved !== true
          || !same(layout.currencyInteraction?.verified, [{ currency: "USD", total: "-$4.50" }, { currency: "NGN", total: "₦9,007,199,254,740,993.01" }]))) {
        failures.push(`${scenario.name}: currency switching and retained selection were not verified`);
      }
      if (scenario.openReportExploreMembers) {
        const drill = layout.memberDrill;
        if (!drill || drill.sourceID !== layout.sourceID || drill.definitionRevision !== layout.definitionRevision
            || drill.metricID !== "risks_outside_appetite" || drill.count !== 4 || drill.rowCount !== 4
            || drill.organizationScopeID !== null || drill.limit !== 50 || drill.focusContained !== true
            || drill.escapeRestoredFocus !== true || drill.captureOpen !== true) {
          failures.push(`${scenario.name}: underlying records require a source-matched member request and sheet keyboard proof`);
        }
      }
    } else {
      const { setup, paper, output } = layout;
      const placement = scenario.viewport.width > 1380 ? "three-columns" : scenario.viewport.width > 900 ? "output-below" : scenario.viewport.width > 700 ? "controls-above" : "stacked";
      if (!rectangle(setup) || !rectangle(paper) || !rectangle(output) || layout.placement !== placement
          || layout.controlsOrder !== (placement === "controls-above" ? "output-setup" : "setup-output")
          || layout.heading.bottom > Math.min(setup.y, output.y) + 1
          || (placement === "three-columns" && (Math.abs(layout.setupWidth - 210) > 1 || Math.abs(setup.width - 210) > 1 || Math.abs(output.width - 210) > 1
            || paper.x < setup.right || output.x < paper.right || Math.abs(output.y - setup.y) > 2 || Math.abs(paper.y - setup.y) > 2))
          || (placement === "output-below" && (Math.abs(layout.setupWidth - 210) > 1 || paper.x < setup.right || output.y < Math.max(setup.bottom, paper.bottom)))
          || (placement === "controls-above" && (setup.x < output.right || Math.abs(setup.y - output.y) > 2 || paper.y < Math.max(setup.bottom, output.bottom)))
          || (placement === "stacked" && (paper.y < setup.bottom || output.y < paper.bottom))) {
        failures.push(`${scenario.name}: Builder setup, preview and output lack source replacement geometry`);
      }
      if (layout.previewKind !== "SETUP_CONTENTS_ONLY" || layout.unsaved !== true || layout.manualDeliveryOnly !== true
          || layout.sectionCount !== 2 || layout.renderedDataRows !== 0 || layout.previewTitle !== "New report"
          || !same(layout.contents, ["Summary", "Data", "Report information"])) {
        failures.push(`${scenario.name}: unsaved builder preview must describe actual output without fabricated generated content`);
      }
    }
    if (scenario.verifyReportNavigation) {
      const interaction = layout.interaction;
      const expectedSequence = kind === "explore" ? ["Explore", "Reports", "Schedules", "Overview", "Explore"]
        : ["Build report", "Schedules", "Reports", "Explore", "Overview"];
      if (!interaction || !same(interaction.stableTabs, stableTabs) || !same(interaction.contextualTabs, [...stableTabs, "Build report"])
          || !same(interaction.arrowSequence, expectedSequence) || interaction.hiddenBuilderRemoved !== true || interaction.baselineRestored !== true
          || (kind === "explore" ? interaction.measureKeyboardSelection !== true || interaction.selectedMeasurePreserved !== true
            : interaction.draftNamePreserved !== true || interaction.areaPreserved !== true || interaction.reportTypePreserved !== true
              || interaction.savedSetupInspectionPreservedDraft !== true || interaction.singleRetainedBuilder !== true)) {
        failures.push(`${scenario.name}: contextual Reports tabs require keyboard and retained input evidence`);
      }
    }
  }
  return { failures, check: { name: "Reports Explore and Builder source layout", status: failures.length ? "FAIL" : "PASS",
    detail: `${reportsDesignEvidenceScenarios.filter((scenario) => byName.has(scenario.name)).length}/${reportsDesignEvidenceScenarios.length} paired-layout, state and real-workflow captures; whole-screen acceptance not assessed` } };
}
