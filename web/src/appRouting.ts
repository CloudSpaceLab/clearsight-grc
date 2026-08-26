export type View = "today" | "programs" | "vendors" | "work" | "imports" | "explore" | "configure";
export type WorkTab = "matters" | "evidence";
export type WorkspaceTarget = {
  programID?: string;
  matterID?: string;
  evidenceID?: string;
  vendorRelationshipID?: string;
  documentID?: string;
  openFirstProgram?: boolean;
  openFirstMatter?: boolean;
  openFirstEvidence?: boolean;
};

export function parseRoute(hash: string): { view: View; workTab?: WorkTab; target: WorkspaceTarget } {
  const parts = hash.replace(/^#\/?/, "").split("/").filter(Boolean);
  const allowed: View[] = ["today", "programs", "vendors", "work", "imports", "explore", "configure"];
  const view = allowed.includes(parts[0] as View) ? parts[0] as View : "today";
  if (view === "programs") return { view, target: { programID: parts[1] } };
  if (view === "vendors") return { view, target: { vendorRelationshipID: parts[1] } };
  if (view === "imports") return { view, target: { documentID: parts[1] } };
  if (view === "work") {
    const workTab: WorkTab = parts[1] === "evidence" ? "evidence" : "matters";
    const target = workTab === "evidence" ? { evidenceID: parts[2] } : { matterID: parts[2] };
    return { view, workTab, target };
  }
  return { view, target: {} };
}

export function routeHash(view: View, target: WorkspaceTarget, workTab: WorkTab) {
  if (view === "programs" && target.programID) return `#programs/${encodeURIComponent(target.programID)}`;
  if (view === "vendors" && target.vendorRelationshipID) return `#vendors/${encodeURIComponent(target.vendorRelationshipID)}`;
  if (view === "imports" && target.documentID) return `#imports/${encodeURIComponent(target.documentID)}`;
  if (view === "work") {
    const id = workTab === "evidence" ? target.evidenceID : target.matterID;
    return `#work/${workTab}${id ? `/${encodeURIComponent(id)}` : ""}`;
  }
  return `#${view}`;
}
