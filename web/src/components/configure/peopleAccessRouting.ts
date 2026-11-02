import type { IdentityAccessArea } from "../IdentityAccessPanel";

export type PeopleAccessView = "people" | "roles" | "policies" | "delegations" | "organization" | "checks";
export type PeopleAccessSelection = "person" | "role" | "policy" | "delegation";
export type PeopleAccessRoute = {
  view: PeopleAccessView;
  area: IdentityAccessArea;
  selections: Partial<Record<PeopleAccessSelection, string>>;
};

const organizationAreas = ["positions", "reporting", "directory", "escalation"] as const;
const selectionKeys: PeopleAccessSelection[] = ["person", "role", "policy", "delegation"];

export function peopleAccessRoute(hash: string): PeopleAccessRoute {
  const [path, query = ""] = hash.replace(/^#\/?/, "").split("?", 2);
  const [, section, leaf] = (path ?? "").split("/");
  const params = new URLSearchParams(query);
  const selections: PeopleAccessRoute["selections"] = {};
  for (const key of selectionKeys) {
    const value = params.get(key);
    if (value && value.length <= 256) selections[key] = value;
  }
  if (section === "authority") return { view: leaf === "delegations" ? "delegations" : leaf === "checks" ? "checks" : "policies", area: "people", selections };
  if (organizationAreas.some((area) => area === leaf)) return { view: "organization", area: leaf as IdentityAccessArea, selections };
  return { view: leaf === "roles" ? "roles" : "people", area: leaf === "roles" ? "roles" : "people", selections };
}

export function peopleAccessHash(view: PeopleAccessView, selections: PeopleAccessRoute["selections"] = {}, area: IdentityAccessArea = "positions"): string {
  const paths: Record<PeopleAccessView, string> = {
    people: "#configure/access", roles: "#configure/access/roles", policies: "#configure/authority",
    delegations: "#configure/authority/delegations", organization: `#configure/access/${organizationAreas.some((value) => value === area) ? area : "positions"}`,
    checks: "#configure/authority/checks",
  };
  const params = new URLSearchParams();
  for (const key of selectionKeys) if (selections[key]) params.set(key, selections[key]!);
  return `${paths[view]}${params.size ? `?${params.toString()}` : ""}`;
}
