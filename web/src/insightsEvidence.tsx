import type { IndicatorPortfolioPage } from "./indicatorInsightsApi";
import type { MonitoringResult } from "./monitoringTypes";
import { InsightsWorkspace } from "./components/insights/InsightsWorkspace";

const page: IndicatorPortfolioPage = {
  generated_at: "2026-10-06T07:00:00Z",
  items: [
    {
      kind: "KRI",
      program_id: "program-channels",
      program_name: "Digital channels",
      check_id: "check-mobile-success",
      check_code: "MOBILE-SUCCESS",
      check_name: "Mobile transaction success rate",
      claim: "Mobile transaction success remains at or above the approved operating limit.",
      check_status: "ACTIVE",
      check_version: 4,
      input_kind: "SOURCE",
      owner_display_name: "Channel Operations",
      reviewer_display_name: "Technology Risk",
      native_measurement: {
        field: "success_rate",
        label: "Success rate",
        unit: "PERCENT",
        precision: 2,
        value: "98.70",
        limits: [{ operator: "GREATER_OR_EQUAL", expected: "99.50" }],
        condition: "BREACHED",
      },
      state: "BREACH",
      reason: "Latest native measurement is outside its approved limit.",
      score: 100,
      band: "CRITICAL",
      coverage: 1,
      minimum_coverage: 0.95,
      freshness_minutes: 60,
      result_id: "result-mobile-3",
      evaluated_at: "2026-10-06T06:55:00Z",
      risks: [
        { id: "risk-mobile-availability", name: "Mobile channel availability" },
        { id: "risk-digital-failure", name: "Digital transaction failure" },
      ],
    },
    {
      kind: "KCI",
      program_id: "program-access",
      program_name: "Privileged access",
      check_id: "check-pam",
      check_code: "PAM-EXCEPTIONS",
      check_name: "Privileged access exceptions",
      claim: "Unresolved privileged-access exceptions remain within the approved limit.",
      check_status: "ACTIVE",
      check_version: 2,
      input_kind: "SOURCE",
      owner_display_name: "Security Operations",
      reviewer_display_name: "Technology Risk",
      native_measurement: {
        field: "exceptions",
        label: "Open exceptions",
        unit: "COUNT",
        precision: 0,
        value: "2",
        limits: [{ operator: "LESS_OR_EQUAL", expected: "3" }],
        condition: "WITHIN",
      },
      state: "NORMAL",
      reason: "Latest native measurement is within its approved limit.",
      score: 0,
      band: "LOW",
      coverage: 1,
      minimum_coverage: 1,
      freshness_minutes: 120,
      result_id: "result-pam-2",
      evaluated_at: "2026-10-06T06:45:00Z",
      risks: [{ id: "risk-privileged-misuse", name: "Privileged access misuse" }],
    },
    {
      kind: "KRI",
      program_id: "program-recovery",
      program_name: "Service continuity",
      check_id: "check-recovery-time",
      check_code: "RECOVERY-TIME",
      check_name: "Recovery test duration",
      claim: "Recovery exercises complete within the approved recovery-time objective.",
      check_status: "ACTIVE",
      check_version: 3,
      input_kind: "FORM",
      owner_display_name: "Service Continuity",
      reviewer_display_name: "Technology Risk",
      native_measurement: {
        field: "recovery_minutes",
        label: "Recovery duration",
        unit: "DURATION",
        duration_unit: "MINUTES",
        precision: 0,
        value: "145",
        limits: [{ operator: "LESS_OR_EQUAL", expected: "120" }],
        condition: "BREACHED",
        reporting_period_start: "2026-09-01T00:00:00Z",
        reporting_period_end: "2026-09-30T23:59:59Z",
      },
      state: "BREACH",
      reason: "Latest native measurement is outside its approved limit.",
      score: 100,
      band: "CRITICAL",
      coverage: 1,
      minimum_coverage: 1,
      freshness_minutes: 10080,
      result_id: "result-recovery-1",
      evaluated_at: "2026-10-05T13:20:00Z",
      risks: [{ id: "risk-service-recovery", name: "Critical service recovery failure" }],
    },
  ],
};

const history: MonitoringResult[] = [{
  id: "result-mobile-3",
  monitoring_check_id: "check-mobile-success",
  monitoring_check_version: 4,
  evaluated_at: "2026-10-06T06:55:00Z",
  evaluation: {
    score: 100,
    band: "CRITICAL",
    coverage: 1,
    measurement: page.items[0]!.native_measurement,
  },
}];

export function InsightsEvidencePage() {
  return <main className="insights-evidence">
    <InsightsWorkspace
      organizationName="Clear Bank"
      legalEntityName="Nigeria"
      loadPortfolio={async () => page}
      loadIndicatorResults={async () => history}
    />
  </main>;
}
