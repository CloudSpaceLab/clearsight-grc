export type View = "today" | "programs" | "work" | "imports" | "explore" | "configure";
export type WorkTab = "matters" | "evidence";
export type WorkspaceTarget = {
  programID?: string;
  matterID?: string;
  evidenceID?: string;
  openFirstProgram?: boolean;
  openFirstMatter?: boolean;
  openFirstEvidence?: boolean;
};

export function parseRoute(hash: string): { view: View; workTab?: WorkTab; target: WorkspaceTarget } {
  const parts = hash.replace(/^#\/?/, "").split("/").filter(Boolean);
  const allowed: View[] = ["today", "programs", "work", "imports", "explore", "configure"];
  const view = allowed.includes(parts[0] as View) ? parts[0] as View : "today";
  if (view === "programs") return { view, target: { programID: parts[1] } };
  if (view === "work") {
    const workTab: WorkTab = parts[1] === "evidence" ? "evidence" : "matters";
    const target = workTab === "evidence" ? { evidenceID: parts[2] } : { matterID: parts[2] };
    return { view, workTab, target };
  }
  return { view, target: {} };
}

export function routeHash(view: View, target: WorkspaceTarget, workTab: WorkTab) {
  if (view === "programs" && target.programID) return `#programs/${encodeURIComponent(target.programID)}`;
  if (view === "work") {
    const id = workTab === "evidence" ? target.evidenceID : target.matterID;
    return `#work/${workTab}${id ? `/${encodeURIComponent(id)}` : ""}`;
  }
  return `#${view}`;
}
