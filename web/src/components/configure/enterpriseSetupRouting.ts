import { peopleAccessHash, peopleAccessRoute, type PeopleAccessRoute } from "./peopleAccessRouting";

export type EnterpriseSetupView = "organization" | "groups" | "jurisdictions";
export type EnterpriseSetupRoute = {
  view: EnterpriseSetupView;
  unit?: string;
  selections: PeopleAccessRoute["selections"];
};

function boundedIdentity(value: string | undefined | null): string | undefined {
  return value && value.trim() && value.length <= 256 && !/[\u0000-\u001f\u007f]/.test(value) ? value : undefined;
}

export function enterpriseSetupRoute(hash: string): EnterpriseSetupRoute | null {
  const [path, query = ""] = hash.replace(/^#\/?/, "").split("?", 2);
  if (path !== "configure/access/positions") return null;
  const params = new URLSearchParams(query);
  const setup = params.get("setup");
  const selections = Object.fromEntries(Object.entries(peopleAccessRoute(hash).selections)
    .filter(([, value]) => boundedIdentity(value))) as PeopleAccessRoute["selections"];
  return {
    view: setup === "groups" || setup === "jurisdictions" ? setup : "organization",
    unit: boundedIdentity(params.get("unit")),
    selections,
  };
}

export function enterpriseSetupHash(view: EnterpriseSetupView, unit?: string, selections: PeopleAccessRoute["selections"] = {}): string {
  const safeSelections = Object.fromEntries(Object.entries(selections)
    .filter(([, value]) => boundedIdentity(value))) as PeopleAccessRoute["selections"];
  const [path, query = ""] = peopleAccessHash("organization", safeSelections, "positions").split("?", 2);
  const params = new URLSearchParams(query);
  if (view !== "organization") params.set("setup", view);
  const selectedUnit = boundedIdentity(unit);
  if (selectedUnit) params.set("unit", selectedUnit);
  return `${path}${params.size ? `?${params.toString()}` : ""}`;
}
