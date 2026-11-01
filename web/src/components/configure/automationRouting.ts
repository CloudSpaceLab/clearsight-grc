export type AutomationView = "automations" | "schedules" | "escalations" | "activity";
export type AutomationRoute = {
  tab: AutomationView;
  policy?: string;
  q: string;
  status?: string;
  action?: string;
};

const views: AutomationView[] = ["automations", "schedules", "escalations", "activity"];

export function automationRoute(hash: string): AutomationRoute {
  const params = new URLSearchParams(hash.split("?", 2)[1] ?? "");
  const tab = params.get("tab") as AutomationView;
  const value = (key: string) => {
    const entry = params.get(key);
    return entry && entry.length <= 256 ? entry : undefined;
  };
  return { tab: views.includes(tab) ? tab : "automations", policy: value("policy"), q: value("q") ?? "", status: value("status"), action: value("action") };
}

export function automationHash(route: AutomationRoute): string {
  const params = new URLSearchParams();
  if (route.tab !== "automations") params.set("tab", route.tab);
  if (route.policy) params.set("policy", route.policy);
  if (route.q) params.set("q", route.q);
  if (route.status) params.set("status", route.status);
  if (route.action) params.set("action", route.action);
  return `#configure/automation${params.size ? `?${params}` : ""}`;
}
