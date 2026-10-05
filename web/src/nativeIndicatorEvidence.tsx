import type { MonitoringResult } from "./monitoringTypes";
import type { RiskIndicatorDetail } from "./riskTypes";
import { IndicatorDetail } from "./components/indicators/IndicatorDetail";

const indicator: RiskIndicatorDetail = {
  link: {
    id: "sample-indicator-link",
    risk_id: "sample-risk",
    risk_version: 3,
    program_id: "sample-channel-program",
    monitoring_check_id: "sample-mobile-success",
    monitoring_check_version: 4,
    kind: "KRI",
    measurement: "MONITORING_RISK_SCORE",
    created_at: "2026-10-05T08:00:00Z",
  },
  program_id: "sample-channel-program",
  program_name: "Digital channels",
  check_id: "sample-mobile-success",
  check_code: "MOBILE-SUCCESS",
  check_name: "Mobile transaction success rate",
  claim: "Mobile transaction success remains at or above the approved operating limit.",
  check_status: "ACTIVE",
  check_version: 4,
  input_kind: "SOURCE",
  owner_display_name: "Channel Operations",
  reviewer_display_name: "Technology Risk",
  measurement: "MONITORING_RISK_SCORE",
  unit: "RISK_POINTS",
  denominator: 100,
  native_measurement: {
    field: "success_rate",
    label: "Success rate",
    unit: "PERCENT",
    precision: 2,
    value: "98.70",
    limits: [{ operator: "GREATER_OR_EQUAL", expected: "99.50" }],
  },
  state: "BREACH",
  reason: "Latest native measurement is outside its approved limit.",
  score: 100,
  band: "CRITICAL",
  coverage: 1,
  minimum_coverage: 0.95,
  freshness_minutes: 60,
  result_id: "sample-result-3",
  evaluated_at: "2026-10-05T10:00:00Z",
  intervention: {
    matter_id: "sample-matter",
    reference: "MAT-2041",
    status: "ASSESSMENT",
    priority: 3,
    created_at: "2026-10-05T10:05:00Z",
  },
};

const history: MonitoringResult[] = [
  {
    id: "sample-result-3",
    monitoring_check_id: indicator.check_id,
    monitoring_check_version: indicator.check_version,
    evaluated_at: "2026-10-05T10:00:00Z",
    evaluation: {
      score: 100,
      band: "CRITICAL",
      coverage: 1,
      measurement: {
        field: "success_rate",
        label: "Success rate",
        unit: "PERCENT",
        precision: 2,
        value: "98.70",
        limits: [{ operator: "GREATER_OR_EQUAL", expected: "99.50" }],
        condition: "BREACHED",
      },
    },
  },
  {
    id: "sample-result-2",
    monitoring_check_id: indicator.check_id,
    monitoring_check_version: indicator.check_version,
    evaluated_at: "2026-10-05T09:00:00Z",
    evaluation: {
      score: 100,
      band: "CRITICAL",
      coverage: 1,
      measurement: {
        field: "success_rate",
        label: "Success rate",
        unit: "PERCENT",
        precision: 2,
        value: "99.10",
        limits: [{ operator: "GREATER_OR_EQUAL", expected: "99.50" }],
        condition: "BREACHED",
      },
    },
  },
  {
    id: "sample-result-1",
    monitoring_check_id: indicator.check_id,
    monitoring_check_version: indicator.check_version,
    evaluated_at: "2026-10-05T08:00:00Z",
    evaluation: {
      score: 0,
      band: "LOW",
      coverage: 1,
      measurement: {
        field: "success_rate",
        label: "Success rate",
        unit: "PERCENT",
        precision: 2,
        value: "99.80",
        limits: [{ operator: "GREATER_OR_EQUAL", expected: "99.50" }],
        condition: "WITHIN",
      },
    },
  },
];

export function NativeIndicatorEvidencePage() {
  return <main className="native-indicator-evidence">
    <header className="topbar">
      <div>
        <span className="eyebrow">Sample data · Digital channels</span>
        <h1>Risk indicator</h1>
        <p>Sample observations dated 5 October 2026.</p>
      </div>
    </header>
    <IndicatorDetail
      indicator={indicator}
      loadResults={async () => history}
    />
  </main>;
}
