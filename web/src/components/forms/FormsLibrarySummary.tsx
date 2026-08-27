import type { FormLibraryItem } from "../../formsTypes";

export function FormsLibrarySummary({ items }: { items: FormLibraryItem[] }) {
  const drafts = items.filter((item) => item.template.status === "DRAFT").length;
  const pending = items.filter((item) => item.template.status === "PENDING_APPROVAL").length;
  const reusable = items.filter((item) => item.active_version && item.active_status === "ACTIVE").length;
  return <section className="forms-library-summary" aria-label="Current form library results">
    <SummaryMetric label="Templates" value={items.length} detail="Current results"/>
    <SummaryMetric label="Drafts" value={drafts} detail="Still editable"/>
    <SummaryMetric label="In review" value={pending} detail="Awaiting decision"/>
    <SummaryMetric label="Reusable" value={reusable} detail="Active revisions" emphasized/>
  </section>;
}

function SummaryMetric({ label, value, detail, emphasized = false }: { label: string; value: number; detail: string; emphasized?: boolean }) {
  return <article className={emphasized ? "forms-summary-metric emphasized" : "forms-summary-metric"}>
    <span>{label}</span>
    <strong>{value}</strong>
    <small>{detail}</small>
  </article>;
}
