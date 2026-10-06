export type View = "today" | "oversight" | "programs" | "risks" | "rcsa" | "losses" | "forms" | "vendors" | "ropa" | "reports" | "insights" | "work" | "people" | "imports" | "explore" | "configure";
export type WorkTab = "assigned" | "matters" | "evidence";
export type ProgramSection = "overview" | "requirements-controls" | "monitoring" | "evidence-results" | "issues-actions" | "history";
export type ProgramItemTarget = { kind: "requirement" | "control-objective"; id: string };
export type VendorPage = "overview" | "register";
export type RopaPage = "register" | "reports";
export type OversightMetric = "critical-high" | "overdue" | "routing-gaps" | "outcome-failures";
export type HomeTab = "oversight" | "attention" | "my-work";
export type OversightScopeMode = "group";
export type InsightKind = "KRI" | "KCI";
export type WorkspaceTarget = {
  programID?: string;
  riskID?: string;
  lossID?: string;
  rcsaCycleID?: string;
  indicatorID?: string;
  indicatorKind?: InsightKind;
  formTemplateID?: string;
  programSection?: ProgramSection;
  programItem?: ProgramItemTarget;
  matterID?: string;
	evidenceID?: string;
	personID?: string;
  vendorRelationshipID?: string;
  vendorPage?: VendorPage;
  ropaPage?: RopaPage;
  ropaActivityID?: string;
  oversightMetric?: OversightMetric;
  homeTab?: HomeTab;
  oversightScope?: OversightScopeMode;
  documentID?: string;
  openFirstProgram?: boolean;
  openFirstMatter?: boolean;
  openFirstEvidence?: boolean;
};

export function parseRoute(hash: string): { view: View; workTab?: WorkTab; target: WorkspaceTarget } {
  const route = hash.replace(/^#\/?/, "").split("?", 1)[0] ?? "";
  const queryIndex = hash.indexOf("?");
  const query = new URLSearchParams(queryIndex >= 0 ? hash.slice(queryIndex + 1) : "");
  const parts = route.split("/").filter(Boolean);
  const decodeTarget = (value?: string) => {
    if (!value) return undefined;
    try { return decodeURIComponent(value); } catch { return value; }
  };
	const allowed: View[] = ["today", "oversight", "programs", "risks", "rcsa", "losses", "forms", "vendors", "ropa", "reports", "insights", "work", "people", "imports", "explore", "configure"];
	const requestedView = allowed.includes(parts[0] as View) ? parts[0] as View : "oversight";
	const view = requestedView === "today" || (requestedView === "ropa" && parts[1] === "reports") ? "oversight" : requestedView;
	if (requestedView === "ropa" && parts[1] === "reports") return { view: "reports", target: {} };
  if (view === "oversight") {
    const metric = query.get("metric");
    const allowedMetrics: OversightMetric[] = ["critical-high", "overdue", "routing-gaps", "outcome-failures"];
    const target: WorkspaceTarget = {};
    const tab = query.get("tab");
    const allowedTabs: HomeTab[] = ["oversight", "attention", "my-work"];
    if (allowedMetrics.includes(metric as OversightMetric)) target.oversightMetric = metric as OversightMetric;
    if (allowedTabs.includes(tab as HomeTab)) target.homeTab = tab as HomeTab;
    if (query.get("scope") === "group") target.oversightScope = "group";
    return { view, target };
  }
  if (view === "programs") {
    if (!parts[1]) return { view, target: {} };
    const allowedSections: ProgramSection[] = ["overview", "requirements-controls", "monitoring", "evidence-results", "issues-actions", "history"];
    const programSection = allowedSections.includes(parts[2] as ProgramSection) ? parts[2] as ProgramSection : "overview";
    const target: WorkspaceTarget = { programID: decodeTarget(parts[1]), programSection };
    const segments = route.split("/");
    if (programSection === "requirements-controls" && segments.length === 5 &&
      (segments[3] === "requirement" || segments[3] === "control-objective") && segments[4]) {
      try {
        const id = decodeURIComponent(segments[4]);
        if (id.trim()) target.programItem = { kind: segments[3], id };
      } catch { /* An invalid item segment must not select another record. */ }
    }
    return { view, target };
  }
	if (view === "risks") return { view, target: { riskID: decodeTarget(parts[1]) } };
	if (view === "rcsa") return { view, target: { rcsaCycleID: decodeTarget(parts[1]) } };
	if (view === "losses") return { view, target: { lossID: decodeTarget(parts[1]) } };
	if (view === "forms") return { view, target: { formTemplateID: decodeTarget(parts[1]) } };
	if (view === "people") return { view, target: { personID: decodeTarget(parts[1]) } };
	if (view === "vendors") {
    if (!parts[1] || parts[1] === "overview") return { view, target: { vendorPage: "overview" } };
    if (parts[1] === "register") return { view, target: { vendorPage: "register", ...(parts[2] ? { vendorRelationshipID: decodeTarget(parts[2]) } : {}) } };
    return { view, target: { vendorPage: "register", vendorRelationshipID: decodeTarget(parts[1]) } };
  }
	if (view === "ropa") {
    if (parts[1] === "activity" && parts[2]) return { view, target: { ropaPage: "register", ropaActivityID: decodeTarget(parts[2]) } };
    if (parts[1] === "reports") return { view, target: { ropaPage: "reports" } };
    return { view, target: { ropaPage: "register" } };
	}
	if (view === "reports") return { view, target: {} };
  if (view === "insights") {
    const indicator = query.get("indicator")?.trim();
    const kind = query.get("kind");
    const target: WorkspaceTarget = {};
    if (indicator) target.indicatorID = indicator;
    if (kind === "KRI" || kind === "KCI") target.indicatorKind = kind;
    return { view, target };
  }
  if (view === "imports") return { view, target: { documentID: decodeTarget(parts[1]) } };
  if (view === "work") {
    const workTab: WorkTab = parts[1] === "evidence" ? "evidence" : parts[1] === "matters" ? "matters" : "assigned";
    const target = workTab === "evidence"
      ? { evidenceID: decodeTarget(parts[2]) }
      : workTab === "matters"
        ? { matterID: decodeTarget(parts[2]) }
        : {};
    return { view, workTab, target };
  }
  return { view, target: {} };
}

export function routeHash(view: View, target: WorkspaceTarget, workTab: WorkTab) {
  if (view === "oversight") {
    const params = new URLSearchParams();
    if (target.homeTab && target.homeTab !== "oversight") params.set("tab", target.homeTab);
    if (target.oversightMetric) params.set("metric", target.oversightMetric);
    if (target.oversightScope === "group") params.set("scope", "group");
    const query = params.toString();
    return query ? `#oversight?${query}` : "#oversight";
  }
  if (view === "programs" && target.programID) {
    const section = target.programSection ?? "overview";
    const item = section === "requirements-controls" ? target.programItem : undefined;
    return `#programs/${encodeURIComponent(target.programID)}/${section}${item?.id.trim() ? `/${item.kind}/${encodeURIComponent(item.id)}` : ""}`;
  }
	if (view === "risks" && target.riskID) return `#risks/${encodeURIComponent(target.riskID)}`;
	if (view === "rcsa" && target.rcsaCycleID) return `#rcsa/${encodeURIComponent(target.rcsaCycleID)}`;
	if (view === "losses" && target.lossID) return `#losses/${encodeURIComponent(target.lossID)}`;
	if (view === "forms" && target.formTemplateID) return `#forms/${encodeURIComponent(target.formTemplateID)}`;
	if (view === "people" && target.personID) return `#people/${encodeURIComponent(target.personID)}`;
  if (view === "vendors") {
    if (target.vendorRelationshipID) return `#vendors/register/${encodeURIComponent(target.vendorRelationshipID)}`;
    if (target.vendorPage) return `#vendors/${target.vendorPage}`;
  }
	if (view === "ropa") {
    if (target.ropaActivityID) return `#ropa/activity/${encodeURIComponent(target.ropaActivityID)}`;
    if (target.ropaPage === "reports") return "#ropa/reports";
	}
	if (view === "reports") return "#reports";
  if (view === "insights") {
    const params = new URLSearchParams();
    if (target.indicatorKind) params.set("kind", target.indicatorKind);
    if (target.indicatorID?.trim()) params.set("indicator", target.indicatorID.trim());
    const query = params.toString();
    return query ? `#insights?${query}` : "#insights";
  }
  if (view === "imports" && target.documentID) return `#imports/${encodeURIComponent(target.documentID)}`;
  if (view === "work") {
    if (workTab === "assigned") return "#work";
    const id = workTab === "evidence" ? target.evidenceID : target.matterID;
    return `#work/${workTab}${id ? `/${encodeURIComponent(id)}` : ""}`;
  }
  return `#${view}`;
}
