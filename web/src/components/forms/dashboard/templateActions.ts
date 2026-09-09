import type { FormLibraryItem } from "../../../formsTypes";

export function canEditTemplate(item: FormLibraryItem) {
  return item.authority_available === true && item.template.status !== "PENDING_APPROVAL"
    && item.operations?.some((operation) => operation.command === "forms.template.revise" && operation.can_act) === true;
}

export function templateEditLabel(item: FormLibraryItem) {
  return item.template.status === "DRAFT" ? "Edit draft" : "Edit form";
}
