export type WorkspaceFilterValue = string | number | boolean | undefined;

export function readWorkspaceFilters(hash: string): Record<string, string> {
  const queryIndex = hash.indexOf("?");
  if (queryIndex < 0) return {};
  return Object.fromEntries(new URLSearchParams(hash.slice(queryIndex + 1)));
}

export function workspaceHash(base: string, values: Record<string, WorkspaceFilterValue>): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(values)) {
    if (value !== undefined && value !== "" && value !== false && value !== 0) params.set(key, String(value));
  }
  const cleanBase = base.split("?", 1)[0] ?? base;
  const query = params.toString();
  return query ? `${cleanBase}?${query}` : cleanBase;
}

export function replaceWorkspaceHash(hash: string) {
  window.history.replaceState(window.history.state, "", `${window.location.pathname}${window.location.search}${hash}`);
}
