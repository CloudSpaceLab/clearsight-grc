import type { Distribution } from "../../../formsDistributionApi";
import { Button, DataTable, StatusBadge, type DataColumn } from "../../ui";
import { distributionStatusLabel, distributionStatusTone, formatDistributionDateTime } from "./distributionPresentation";

export function SentFormsTable({ items, selectedID, nextCursor, loadingMore, onSelect, onLoadMore }: { items: readonly Distribution[]; selectedID?: string; nextCursor?: string; loadingMore: boolean; onSelect: (id: string) => void; onLoadMore: () => void }) {
  const columns: readonly DataColumn<Distribution>[] = [
    { id: "distribution", header: "Distribution", render: (value) => <div className="forms-sent__title"><strong>{value.title}</strong><span>{value.purpose}</span></div>, accessibleText: (value) => `${value.title}. ${value.purpose}` },
    { id: "status", header: "Status", kind: "status", render: (value) => <StatusBadge tone={distributionStatusTone[value.status]}>{distributionStatusLabel[value.status]}</StatusBadge>, accessibleText: (value) => distributionStatusLabel[value.status] },
    { id: "revision", header: "Form revision", kind: "number", render: (value) => `v${value.form_template_version}`, accessibleText: (value) => `Version ${value.form_template_version}` },
    { id: "deadline", header: "Deadline", render: (value) => formatDistributionDateTime(value.deadline), accessibleText: (value) => formatDistributionDateTime(value.deadline) },
    { id: "subject", header: "Subject", render: (value) => `${value.subject_type} · ${value.subject_id}`, accessibleText: (value) => `${value.subject_type} ${value.subject_id}` },
    { id: "action", header: "Open", kind: "action", render: (value) => <Button variant="quiet" aria-label={`Open ${value.title}`} onPress={() => onSelect(value.id)}>Open</Button>, accessibleText: (value) => `Open ${value.title}` },
  ];
  return <DataTable
    ariaLabel="Sent-form distributions"
    rows={items}
    rowKey={(value) => value.id}
    rowName={(value) => `${value.title}, ${distributionStatusLabel[value.status]}, form version ${value.form_template_version}, due ${formatDistributionDateTime(value.deadline)}, ${value.subject_type} ${value.subject_id}`}
    columns={columns}
    selectedKey={selectedID}
    pagination={nextCursor ? { label: "Sent-form pages", nextLabel: "Load more sent forms", onNext: onLoadMore, isLoading: loadingMore } : undefined}
  />;
}
