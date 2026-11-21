import type { ReactNode } from "react";
import type { VendorPage } from "../appRouting";
import type { VendorFormSummary } from "../vendorFormsApi";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import { VendorBrandIcon } from "./VendorBrandIcon";
import { ActionLink, Notice, SelectableRecord, StatusBadge } from "./ui";

type CollectionPage = Exclude<VendorPage, "overview" | "register">;
type Props = {
  page: CollectionPage;
  records: VendorRelationshipAggregate[];
  summaries: Map<string, VendorFormSummary>;
  summaryState: "loading" | "live" | "unavailable";
  hasMore: boolean;
  controls: ReactNode;
  footer: ReactNode;
  onOpen: (record: VendorRelationshipAggregate, page: CollectionPage) => void;
  onOpenRegister: () => void;
};

const content = {
  onboarding: { title: "Vendor onboarding", detail: "Review ownership, evidence and approvals before activation.", action: "Review onboarding" },
  assessments: { title: "Vendor assessments", detail: "Review submitted forms and bank assessments for each service.", action: "Review forms" },
  documents: { title: "Vendor documents", detail: "Open submitted documents and their review status by service.", action: "Open documents" },
} as const;

export function VendorPortfolioCollection({ page, records, summaries, summaryState, hasMore, controls, footer, onOpen, onOpenRegister }: Props) {
  const view = content[page];
  const visible = page === "onboarding" ? records.filter((record) => record.relationship.status === "PROPOSED" || record.relationship.status === "UNDER_REVIEW") : records;
  return <section className={`vendor-collection vendor-collection--${page}`} aria-label={view.title}>
    <header className="vendor-collection-heading"><div><h2>{view.title}</h2><p>{view.detail}</p></div></header>
    {controls}
    <p className="vendor-collection-population">{visible.length} of {records.length} loaded {records.length === 1 ? "service" : "services"}{page === "onboarding" ? " proposed or under review" : " shown"}{hasMore ? "; more services are available" : ""}.</p>
    {page === "assessments" && summaryState === "unavailable" && <Notice tone="warning">Some form summaries are unavailable. Open the service to check its forms.</Notice>}
    {visible.length === 0 ? <div className="vendor-collection-empty"><h3>{page === "onboarding" ? "No proposed services or services under review in the loaded results." : "No vendor services match the current search."}</h3><p>{hasMore ? "Load more services or change the search." : "Change the search or open the register to add a vendor service."}</p></div>
      : <ul className="vendor-collection-list">{visible.map((record) => {
        const summary = summaries.get(record.relationship.id);
        return <li key={record.relationship.id}>
          <SelectableRecord variant="list" title={record.vendor.legal_name} metadata={record.relationship.service_name}
            aria-label={`${view.action} for ${record.vendor.legal_name}, ${record.relationship.service_name}`}
            startContent={<VendorBrandIcon vendorID={record.vendor.id} legalName={record.vendor.legal_name} brand={record.brand} decorative/>}
            endContent={<span>{view.action} <span aria-hidden="true">→</span></span>}
            onPress={() => onOpen(record, page)}/>
          <div className="vendor-collection-context">
            {page === "onboarding" ? <><StatusBadge tone="neutral">{record.relationship.status === "PROPOSED" ? "Proposed" : "Under review"}</StatusBadge><span>{record.relationship.privacy_role === "NONE" ? "No processing role" : record.relationship.privacy_role === "PROCESSOR" ? "Processor" : "Joint controller"}</span><span>Review required gates before activation.</span></>
              : page === "assessments" ? summary ? <><span><strong>{summary.submitted_forms}</strong> submitted</span><span><strong>{summary.awaiting_review}</strong> awaiting review</span><span><strong>{summary.assessed_forms}</strong> assessed</span>{!!summary.partially_replaced_forms && <span>{summary.partially_replaced_forms} partly replaced</span>}<span>Checked <time dateTime={summary.observed_at}>{formatChecked(summary.observed_at)}</time></span></> : <span>{summaryState === "loading" ? "Loading form counts…" : "Form counts unavailable"}</span>
                : <span>Open submitted files for this service.</span>}
          </div>
        </li>;
      })}</ul>}
    <div className="vendor-collection-footer">{footer}<ActionLink onPress={onOpenRegister}>Open register</ActionLink></div>
  </section>;
}

function formatChecked(value: string) {
  const timestamp = Date.parse(value);
  return Number.isFinite(timestamp) ? new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" }).format(timestamp) : "Time unavailable";
}
