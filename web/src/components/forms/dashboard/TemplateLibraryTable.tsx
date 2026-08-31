import type { FormLibraryItem } from "../../../formsTypes";
import type { LifecycleStatus } from "../../../monitoringTypes";
import { Button, CheckboxField, DataTable, StatusBadge, type DataColumn, type StatusTone } from "../../ui";

type Props = {
  items: FormLibraryItem[];
  selectedIDs: Set<string>;
  targetID?: string;
  onToggle: (id: string) => void;
  onOpen: (id: string) => void;
};

export function TemplateLibraryTable({ items, selectedIDs, targetID, onToggle, onOpen }: Props) {
  const columns: readonly DataColumn<FormLibraryItem>[] = [
    {
      id: "selection",
      header: "Select",
      kind: "action",
      render: ({ template }) => <CheckboxField label={`Select ${template.name}`} isLabelHidden isSelected={selectedIDs.has(template.id)} onChange={() => onToggle(template.id)}/>,
      accessibleText: ({ template }) => selectedIDs.has(template.id) ? "Selected" : "Not selected",
    },
    {
      id: "form",
      header: "Form",
      render: ({ template }) => <div className="forms-library-record"><strong>{template.name}</strong><span>{template.purpose || template.code}</span></div>,
      accessibleText: ({ template }) => `${template.name}. ${template.purpose || template.code}`,
    },
    { id: "state", header: "State", kind: "status", render: ({ template }) => <StatusPill status={template.status}/>, accessibleText: ({ template }) => statusLabel(template.status) },
    { id: "revision", header: "Revision", kind: "number", render: (item) => <div className="forms-library-revision"><strong>v{item.template.version}</strong><span>{item.active_version ? `Reusable v${item.active_version}` : "No reusable revision"}</span></div>, accessibleText: (item) => `Version ${item.template.version}. ${item.active_version ? `Reusable version ${item.active_version}` : "No reusable revision"}` },
    { id: "owner", header: "Owner", render: ({ template }) => ownerLabel(template), accessibleText: ({ template }) => ownerLabel(template) },
    { id: "updated", header: "Updated", render: ({ template }) => formatDate(template.updated_at), accessibleText: ({ template }) => formatDate(template.updated_at) },
    { id: "open", header: "Open", kind: "action", render: ({ template }) => <Button variant="quiet" aria-label={`Open ${template.name}`} onPress={() => onOpen(template.id)}>Open</Button>, accessibleText: ({ template }) => `Open ${template.name}` },
  ];
  return <DataTable
    ariaLabel="Form templates"
    rows={items}
    rowKey={(item) => item.template.id}
    rowName={(item) => `${item.template.name}, ${statusLabel(item.template.status)}, version ${item.template.version}, owned by ${ownerLabel(item.template)}`}
    columns={columns}
    selectedKey={targetID}
  />;
}

export function StatusPill({ status }: { status: LifecycleStatus }) {
  return <StatusBadge tone={statusTone(status)}>{statusLabel(status)}</StatusBadge>;
}

function statusTone(status: LifecycleStatus): StatusTone {
  if (status === "ACTIVE") return "success";
  if (status === "PENDING_APPROVAL" || status === "PAUSED") return "warning";
  if (status === "REJECTED") return "error";
  if (status === "RETIRED") return "unknown";
  return "neutral";
}

export function statusLabel(status: LifecycleStatus) {
  if (status === "PENDING_APPROVAL") return "Awaiting approval";
  return status.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

export function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Unknown";
  return new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric" }).format(date);
}

function ownerLabel(template: FormLibraryItem["template"]) {
  return template.responsible_team || (template.owner_principal_id ? "Assigned owner" : "Not assigned");
}
