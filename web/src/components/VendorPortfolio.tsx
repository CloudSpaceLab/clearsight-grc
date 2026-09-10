import type { VendorFormSummary, VendorFormsFilter } from "../vendorFormsApi";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import { Button } from "./ui";
import "./vendor-portfolio.css";

export function VendorPortfolio({ records, summaries, summaryState, hasMore, onFilter }: {
  records: VendorRelationshipAggregate[];
  summaries: Map<string, VendorFormSummary>;
  summaryState: "loading" | "live" | "unavailable";
  hasMore: boolean;
  onFilter: (filter: VendorFormsFilter | "") => void;
}) {
  const available = records.flatMap(record => {
    const summary = summaries.get(record.relationship.id);
    return summary ? [summary] : [];
  });
  const complete = summaryState === "live" && available.length === records.length;
  const sum = (key: "outstanding_forms" | "awaiting_review" | "overdue_forms" | "submitted_forms" | "assessed_forms") => available.reduce((total, item) => total + item[key], 0);
  const oldest = available.map(item => item.observed_at).filter(value => Number.isFinite(Date.parse(value))).sort((a, b) => Date.parse(a) - Date.parse(b))[0];
  const assessed = sum("assessed_forms"), submitted = sum("submitted_forms");
  const criticalities = ["CRITICAL", "IMPORTANT", "STANDARD"] as const;
  const metrics = [
    { label: "Outstanding forms", count: sum("outstanding_forms"), detail: "Vendor response due", filter: "AWAITING_VENDOR", action: "Review outstanding forms", tone: "neutral" },
    { label: "Awaiting review", count: sum("awaiting_review"), detail: "Bank review required", filter: "AWAITING_REVIEW", action: "Review submitted forms", tone: "accent" },
    { label: "Overdue forms", count: sum("overdue_forms"), detail: "Response deadline passed", filter: "OVERDUE", action: "Review overdue forms", tone: "warning" },
  ] as const;
  return <section className="vendor-portfolio" aria-label="Vendor portfolio metrics">
    <div className="vendor-portfolio-scope"><span>{records.length} loaded services{hasMore ? " · More services available" : ""}</span><span>{complete && oldest ? <>Checked from <time dateTime={oldest}>{new Date(oldest).toLocaleString()}</time></> : `${available.length} of ${records.length} service summaries available`}</span></div>
    <div className="vendor-metrics">
      <div className="vendor-metric vendor-metric--portfolio" role="group" aria-label="Vendor services"><span className="vendor-metric-label">Vendor services</span><strong className="vendor-metric-value">{records.length}</strong><span>{new Set(records.map(record => record.vendor.id)).size} vendors · Current search</span><Button variant="quiet" onPress={() => onFilter("")}>Review all services</Button></div>
      {metrics.map(metric => <div key={metric.label} className={`vendor-metric vendor-metric--${metric.tone}`} role="group" aria-label={metric.label}><span className="vendor-metric-label">{metric.label}</span><strong className="vendor-metric-value">{complete ? metric.count : "Unknown"}</strong><span>{metric.detail}</span>{complete ? <Button variant="quiet" onPress={() => onFilter(metric.filter)}>{metric.action}</Button> : <span className="vendor-metric-recovery">{summaryState === "loading" ? "Checking form work…" : "Form summaries incomplete"}</span>}</div>)}
    </div>
    <div className="vendor-portfolio-analysis">
      <section className="vendor-analysis-card" aria-labelledby="vendor-criticality-title"><header><h2 id="vendor-criticality-title">Service criticality</h2><span>{records.length} services</span></header><div className="vendor-criticality-bars">{criticalities.map(value => {
        const count = records.filter(record => record.relationship.criticality === value).length;
        return <div className={`vendor-criticality-bar criticality-${value.toLowerCase()}`} key={value}><span>{value[0]}{value.slice(1).toLowerCase()}</span><div className="vendor-bar-track" aria-hidden="true"><div style={{ width: `${records.length ? count / records.length * 100 : 0}%` }}/></div><strong>{count}</strong></div>;
      })}</div></section>
      <section className="vendor-analysis-card" aria-labelledby="vendor-review-title"><header><h2 id="vendor-review-title">Response review</h2><span>Submitted forms</span></header>{complete ? <><div className="vendor-review-total"><strong>{assessed}</strong><span>{assessed} assessed / {submitted} submitted</span></div><div className="vendor-review-track" aria-hidden="true"><div style={{ width: `${submitted ? Math.min(100, assessed / submitted * 100) : 0}%` }}/></div><p>{submitted ? "Assessment does not establish vendor approval." : "No submitted forms in this population."}</p></> : <p>Review coverage unknown until all service summaries are available.</p>}</section>
    </div>
  </section>;
}
