export type View = "today" | "programs" | "vendors" | "work" | "imports" | "explore" | "configure";
export type WorkTab = "matters" | "evidence";
export type ProgramSection = "overview" | "requirements-controls" | "monitoring" | "evidence-results" | "issues-actions" | "history";
export type WorkspaceTarget = {
  programID?: string;
  programSection?: ProgramSection;
  matterID?: string;
  evidenceID?: string;
  vendorRelationshipID?: string;
  openFirstProgram?: boolean;
  openFirstMatter?: boolean;
  openFirstEvidence?: boolean;
};

export function parseRoute(hash: string): { view: View; workTab?: WorkTab; target: WorkspaceTarget } {
  const parts = hash.replace(/^#\/?/, "").split("/").filter(Boolean);
  const allowed: View[] = ["today", "programs", "vendors", "work", "imports", "explore", "configure"];
  const view = allowed.includes(parts[0] as View) ? parts[0] as View : "today";
  if (view === "programs") {
    if (!parts[1]) return { view, target: {} };
    const allowedSections: ProgramSection[] = ["overview", "requirements-controls", "monitoring", "evidence-results", "issues-actions", "history"];
    const programSection = allowedSections.includes(parts[2] as ProgramSection) ? parts[2] as ProgramSection : "overview";
    return { view, target: { programID: parts[1], programSection } };
  }
  if (view === "vendors") return { view, target: { vendorRelationshipID: parts[1] } };
  if (view === "work") {
    const workTab: WorkTab = parts[1] === "evidence" ? "evidence" : "matters";
    const target = workTab === "evidence" ? { evidenceID: parts[2] } : { matterID: parts[2] };
    return { view, workTab, target };
  }
  return { view, target: {} };
}

export function routeHash(view: View, target: WorkspaceTarget, workTab: WorkTab) {
  if (view === "programs" && target.programID) return `#programs/${encodeURIComponent(target.programID)}/${target.programSection ?? "overview"}`;
  if (view === "vendors" && target.vendorRelationshipID) return `#vendors/${encodeURIComponent(target.vendorRelationshipID)}`;
  if (view === "work") {
    const id = workTab === "evidence" ? target.evidenceID : target.matterID;
    return `#work/${workTab}${id ? `/${encodeURIComponent(id)}` : ""}`;
  }
  return `#${view}`;
}
