import { routeHash } from "./appRouting";

// Only stored conversion results may provide destinations; a proposed type is not a receipt.
export function documentResultLink(type?: string, id?: string, programID?: string) {
  if (!id?.trim()) return undefined;
  if (type === "PROGRAM") return { href: routeHash("programs", { programID: id }, "matters"), label: "Open Program", object: "Program" };
  if (type === "MATTER") return { href: routeHash("work", { matterID: id }, "matters"), label: "Open issue", object: "Issue" };
  if ((type === "REQUIREMENT" || type === "CONTROL_OBJECTIVE") && programID?.trim()) {
    const kind = type === "REQUIREMENT" ? "requirement" : "control-objective";
    return {
      href: routeHash("programs", { programID, programSection: "requirements-controls", programItem: { kind, id } }, "matters"),
      label: type === "REQUIREMENT" ? "Open requirement" : "Open control objective",
      object: type === "REQUIREMENT" ? "Requirement" : "Control objective",
    };
  }
  return undefined;
}
