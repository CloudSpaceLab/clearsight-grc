import type { MatterAction } from "./types";
import { openFinding, openRiskAction, overdueRiskAction, type VendorRiskFinding } from "./vendorRiskWork";

export type VendorExceptionFilter = "ATTENTION" | "OVERDUE" | "OPEN" | "ALL";
export type VendorExceptionBand = "OVERDUE" | "BLOCKED" | "INCOMPLETE" | "DUE_SOON" | "OPEN" | "CLOSED";
export type VendorExceptionRow = {
  item: VendorRiskFinding;
  band: VendorExceptionBand;
  nextAction?: MatterAction;
  openActionCount: number;
  overdueActionCount: number;
  owner?: string;
  sourceRating?: string;
};

const bandOrder: Record<VendorExceptionBand, number> = {
  OVERDUE: 0,
  BLOCKED: 1,
  INCOMPLETE: 2,
  DUE_SOON: 3,
  OPEN: 4,
  CLOSED: 5,
};

function recordedText(value: unknown) {
  return typeof value === "string" && value.trim() ? value.trim() : undefined;
}

function actionDeadline(action?: MatterAction) {
  if (!action?.due_at) return Number.POSITIVE_INFINITY;
  const parsed = Date.parse(action.due_at);
  return Number.isNaN(parsed) ? Number.POSITIVE_INFINITY : parsed;
}

function present(item: VendorRiskFinding, now: number): VendorExceptionRow {
  const facts = item.record.matter.known_facts ?? {};
  const openActions = item.record.actions.filter(openRiskAction).sort((left, right) => actionDeadline(left) - actionDeadline(right) || left.title.localeCompare(right.title));
  const overdueActions = openActions.filter(action => overdueRiskAction(action, now));
  const nextAction = openActions[0];
  const owner = recordedText(facts.source_owner);
  const sourceRating = recordedText(facts.source_rating);
  let band: VendorExceptionBand = "OPEN";
  if (!openFinding(item)) band = "CLOSED";
  else if (overdueActions.length) band = "OVERDUE";
  else if (openActions.some(action => action.status === "BLOCKED")) band = "BLOCKED";
  else if (!nextAction || !owner || !Number.isFinite(actionDeadline(nextAction))) band = "INCOMPLETE";
  else if (actionDeadline(nextAction) <= now + 30 * 24 * 60 * 60 * 1000) band = "DUE_SOON";
  return { item, band, nextAction, openActionCount: openActions.length, overdueActionCount: overdueActions.length, owner, sourceRating };
}

export function presentVendorExceptions(items: VendorRiskFinding[], filter: VendorExceptionFilter, now = Date.now()): VendorExceptionRow[] {
  return items
    .map(item => present(item, now))
    .filter(row => {
      if (filter === "ALL") return true;
      if (filter === "OPEN") return row.band !== "CLOSED";
      if (filter === "OVERDUE") return row.band === "OVERDUE";
      return ["OVERDUE", "BLOCKED", "INCOMPLETE", "DUE_SOON"].includes(row.band);
    })
    .sort((left, right) => bandOrder[left.band] - bandOrder[right.band]
      || actionDeadline(left.nextAction) - actionDeadline(right.nextAction)
      || left.item.record.matter.title.localeCompare(right.item.record.matter.title));
}
