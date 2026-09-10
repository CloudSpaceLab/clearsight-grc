export type View = "today" | "oversight" | "programs" | "forms" | "vendors" | "work" | "people" | "imports" | "explore" | "configure";
export type WorkTab = "matters" | "evidence";
export type ProgramSection = "overview" | "requirements-controls" | "monitoring" | "evidence-results" | "issues-actions" | "history";
export type ProgramItemTarget = { kind: "requirement" | "control-objective"; id: string };
export type WorkspaceTarget = {
  programID?: string;
  formTemplateID?: string;
  programSection?: ProgramSection;
  programItem?: ProgramItemTarget;
  matterID?: string;
	evidenceID?: string;
	personID?: string;
  vendorRelationshipID?: string;
  documentID?: string;
  openFirstProgram?: boolean;
  openFirstMatter?: boolean;
  openFirstEvidence?: boolean;
};

export function parseRoute(hash: string): { view: View; workTab?: WorkTab; target: WorkspaceTarget } {
  const route = hash.replace(/^#\/?/, "").split("?", 1)[0] ?? "";
  const parts = route.split("/").filter(Boolean);
  const decodeTarget = (value?: string) => {
    if (!value) return undefined;
    try { return decodeURIComponent(value); } catch { return value; }
  };
	const allowed: View[] = ["today", "oversight", "programs", "forms", "vendors", "work", "people", "imports", "explore", "configure"];
  const view = allowed.includes(parts[0] as View) ? parts[0] as View : "today";
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
	if (view === "forms") return { view, target: { formTemplateID: decodeTarget(parts[1]) } };
	if (view === "people") return { view, target: { personID: decodeTarget(parts[1]) } };
  if (view === "vendors") return { view, target: { vendorRelationshipID: decodeTarget(parts[1]) } };
  if (view === "imports") return { view, target: { documentID: decodeTarget(parts[1]) } };
  if (view === "work") {
    const workTab: WorkTab = parts[1] === "evidence" ? "evidence" : "matters";
    const target = workTab === "evidence" ? { evidenceID: decodeTarget(parts[2]) } : { matterID: decodeTarget(parts[2]) };
    return { view, workTab, target };
  }
  return { view, target: {} };
}

export function routeHash(view: View, target: WorkspaceTarget, workTab: WorkTab) {
  if (view === "programs" && target.programID) {
    const section = target.programSection ?? "overview";
    const item = section === "requirements-controls" ? target.programItem : undefined;
    return `#programs/${encodeURIComponent(target.programID)}/${section}${item?.id.trim() ? `/${item.kind}/${encodeURIComponent(item.id)}` : ""}`;
  }
	if (view === "forms" && target.formTemplateID) return `#forms/${encodeURIComponent(target.formTemplateID)}`;
	if (view === "people" && target.personID) return `#people/${encodeURIComponent(target.personID)}`;
  if (view === "vendors" && target.vendorRelationshipID) return `#vendors/${encodeURIComponent(target.vendorRelationshipID)}`;
  if (view === "imports" && target.documentID) return `#imports/${encodeURIComponent(target.documentID)}`;
  if (view === "work") {
    const id = workTab === "evidence" ? target.evidenceID : target.matterID;
    return `#work/${workTab}${id ? `/${encodeURIComponent(id)}` : ""}`;
  }
  return `#${view}`;
}
