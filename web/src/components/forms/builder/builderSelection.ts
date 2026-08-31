export type BuilderSelection =
  | { kind: "overview" }
  | { kind: "section"; sectionID: string }
  | { kind: "field"; fieldID: string };

export const overviewSelection: BuilderSelection = { kind: "overview" };

export function selectionKey(selection: BuilderSelection): string {
  if (selection.kind === "overview") return "overview";
  return `${selection.kind}:${selection.kind === "section" ? selection.sectionID : selection.fieldID}`;
}
