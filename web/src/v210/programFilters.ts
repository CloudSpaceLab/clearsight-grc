export const programStates = [
  ["CURRENT", "Up to date"], ["AT_RISK", "At risk"], ["GAP_IDENTIFIED", "Gap identified"],
  ["EVIDENCE_INSUFFICIENT", "Evidence incomplete"], ["IMPLEMENTATION_PENDING", "Implementation pending"],
  ["OVERDUE", "Overdue"], ["UNDER_REVIEW", "Under review"], ["NOT_APPLICABLE", "Not applicable"], ["UNKNOWN", "Unknown"],
] as const;

export function readProgramFilters(hash: string) {
  const params = new URLSearchParams(hash.split("?")[1] ?? "");
  const state = params.get("overall_state") ?? "";
  return { search: params.get("q") ?? "", overallState: programStates.some(([value]) => value === state) ? state : "" };
}

export function programFilterQuery(search: string, overallState: string) {
  const params = new URLSearchParams();
  if (search) params.set("q", search);
  if (overallState) params.set("overall_state", overallState);
  return params.size ? `?${params.toString()}` : "";
}
