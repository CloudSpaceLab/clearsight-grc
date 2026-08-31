import type { FormLibraryItem } from "../../../formsTypes";
import type { LifecycleStatus } from "../../../monitoringTypes";

type Props = {
  items: FormLibraryItem[];
  selectedIDs: Set<string>;
  targetID?: string;
  onToggle: (id: string) => void;
  onOpen: (id: string) => void;
};

export function TemplateLibraryTable({ items, selectedIDs, targetID, onToggle, onOpen }: Props) {
  return <div className="forms-table-wrap forms-library-table-wrap">
    <table className="forms-table forms-library-table">
      <thead>
        <tr>
          <th><span className="sr-only">Select</span></th>
          <th>Form</th>
          <th>State</th>
          <th>Revision</th>
          <th>Owner</th>
          <th>Updated</th>
          <th><span className="sr-only">Open</span></th>
        </tr>
      </thead>
      <tbody>
        {items.map((item) => {
          const form = item.template;
          const owner = form.responsible_team || form.owner_principal_id || "Not assigned";
          return <tr key={form.id} className={targetID === form.id ? "selected" : ""}>
            <td className="forms-library-select">
              <input
                type="checkbox"
                aria-label={`Select ${form.name}`}
                checked={selectedIDs.has(form.id)}
                onChange={() => onToggle(form.id)}
              />
            </td>
            <td className="forms-library-name">
              <button type="button" className="forms-row-open" onClick={() => onOpen(form.id)}>
                <strong>{form.name}</strong>
                <span>{form.purpose || form.code}</span>
              </button>
            </td>
            <td><StatusPill status={form.status}/></td>
            <td className="forms-library-revision">
              <strong>v{form.version}</strong>
              <span>{item.active_version ? `Reusable v${item.active_version}` : "No reusable revision"}</span>
            </td>
            <td className="forms-library-owner">{owner}</td>
            <td className="forms-library-updated">{formatDate(form.updated_at)}</td>
            <td><button type="button" className="forms-row-action" aria-label={`Open ${form.name}`} onClick={() => onOpen(form.id)}>Open</button></td>
          </tr>;
        })}
      </tbody>
    </table>
  </div>;
}

export function StatusPill({ status }: { status: LifecycleStatus }) {
  return <span className={`forms-status status-${status.toLowerCase().replaceAll("_", "-")}`}>{statusLabel(status)}</span>;
}

export function statusLabel(status: LifecycleStatus) {
  return status.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

export function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Unknown";
  return new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric" }).format(date);
}
