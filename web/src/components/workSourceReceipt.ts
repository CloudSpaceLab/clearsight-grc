import type { ActorWorkDueCounts, ActorWorkPage, ActorWorkScope } from "../api";

// Source timestamps are UTC with at most PostgreSQL's microsecond precision.
export function validWorkTimestamp(value: string): boolean {
  return /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/.test(value) &&
    Number.isFinite(Date.parse(value));
}

function matchesScope(value: ActorWorkPage | ActorWorkDueCounts, scope: ActorWorkScope): boolean {
  return Boolean(value && value.coverage === "WORKFLOW_ASSIGNMENTS_ONLY" &&
    value.scope?.tenant_id === scope.tenantID &&
    value.scope?.principal_id === scope.principalID &&
    value.scope?.legal_entity_id === scope.legalEntityID && validWorkTimestamp(value.as_of));
}

export function validWorkCounts(value: ActorWorkDueCounts, scope: ActorWorkScope): boolean {
  if (!matchesScope(value, scope) || value.count_basis !== "SOURCE_SNAPSHOT") return false;
  const { total, overdue, due_soon: dueSoon, undated } = value;
  return [total, overdue, dueSoon, undated].every((count) => Number.isSafeInteger(count) && count >= 0) &&
    overdue + dueSoon + undated <= total;
}

export function validWorkPage(value: ActorWorkPage, scope: ActorWorkScope): boolean {
  return matchesScope(value, scope) && value.count_basis === "NOT_AVAILABLE" &&
    Array.isArray(value.items) && value.items.length <= 50 &&
    typeof value.has_more === "boolean" &&
    (value.has_more
      ? value.items.length > 0 && typeof value.next_cursor === "string" && value.next_cursor.length > 0
      : !value.next_cursor);
}
