import type { OrganizationScope } from "../../identityAccessApi";

export type OrganizationHierarchyRow = {
  scope: OrganizationScope;
  depth: number;
  ancestors: string[];
  childCount: number;
  issue?: "missing-parent" | "cycle";
};

/** A display forest of the returned population; it does not infer authorization or missing ancestry. */
export function organizationHierarchy(scopes: OrganizationScope[]): OrganizationHierarchyRow[] {
  const byID = new Map(scopes.map((scope) => [scope.id, scope]));
  const parents = new Map<string, string | undefined>();
  const issues = new Map<string, OrganizationHierarchyRow["issue"]>();
  for (const scope of byID.values()) {
    const parent = scope.parent_scope_id;
    parents.set(scope.id, parent && byID.has(parent) ? parent : undefined);
    if (parent && !byID.has(parent)) issues.set(scope.id, "missing-parent");
  }
  // Break one display edge per cycle, while marking every member. Every loaded
  // area remains discoverable; the actual parent remains in its record detail.
  const resolved = new Set<string>();
  for (const id of byID.keys()) {
    const path: string[] = [];
    const index = new Map<string, number>();
    let cursor: string | undefined = id;
    while (cursor && !resolved.has(cursor) && !index.has(cursor)) {
      index.set(cursor, path.length);
      path.push(cursor);
      cursor = parents.get(cursor);
    }
    if (cursor && index.has(cursor)) {
      const cycle = path.slice(index.get(cursor));
      for (const member of cycle) issues.set(member, "cycle");
      parents.set([...cycle].sort()[0]!, undefined);
    }
    for (const member of path) resolved.add(member);
  }
  const children = new Map<string | undefined, OrganizationScope[]>();
  const ordered = [...byID.values()].sort((a, b) => a.department_path.join("/").localeCompare(b.department_path.join("/")) || a.name.localeCompare(b.name) || a.id.localeCompare(b.id));
  for (const scope of ordered) {
    const parent = parents.get(scope.id);
    const siblings = children.get(parent) ?? [];
    siblings.push(scope);
    children.set(parent, siblings);
  }
  const rows: OrganizationHierarchyRow[] = [];
  const stack = [...(children.get(undefined) ?? [])].reverse().map((scope) => ({ scope, ancestors: [] as string[] }));
  while (stack.length) {
    const next = stack.pop()!;
    const descendants = children.get(next.scope.id) ?? [];
    rows.push({ ...next, depth: next.ancestors.length, childCount: descendants.length, issue: issues.get(next.scope.id) });
    for (let i = descendants.length - 1; i >= 0; i--) stack.push({ scope: descendants[i]!, ancestors: [...next.ancestors, next.scope.id] });
  }
  return rows;
}

export function filterOrganizationHierarchy(rows: OrganizationHierarchyRow[], query: string, collapsed: ReadonlySet<string>) {
  const normalized = query.trim().toLocaleLowerCase();
  if (!normalized) return { rows: rows.filter((row) => !row.ancestors.some((id) => collapsed.has(id))), matches: rows.length };
  const matching = rows.filter(({ scope }) => [scope.name, scope.code, scopeKindLabel(scope.kind), scope.department_path.join(" ")].some((value) => value.toLocaleLowerCase().includes(normalized)));
  const included = new Set(matching.flatMap((row) => [row.scope.id, ...row.ancestors]));
  return { rows: rows.filter((row) => included.has(row.scope.id)), matches: matching.length };
}

export function scopeKindLabel(value: OrganizationScope["kind"]) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
