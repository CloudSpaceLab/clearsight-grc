import type { VendorFormsFilter } from "./vendorFormsApi";

export const vendorWorkFilters = [
  { id: "AWAITING_VENDOR", label: "Awaiting vendor" },
  { id: "AWAITING_REVIEW", label: "Awaiting review" },
  { id: "WITH_RISKS", label: "Submitted with risks" },
  { id: "HIGH_RISK", label: "High or critical concern" },
  { id: "OVERDUE", label: "Overdue" },
  { id: "NOT_ASSESSED", label: "Not assessed" },
] satisfies Array<{ id: VendorFormsFilter; label: string }>;

export const assessmentStateLabels: Record<string, string> = {
  NOT_REQUIRED: "Review not required", AWAITING_REVIEW: "Awaiting review", IN_REVIEW: "In review", ASSESSED: "Reviewed",
};
