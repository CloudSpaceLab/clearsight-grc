import type { ReactNode } from "react";
import { Button } from "./Button";

export type DataColumn<Row> = {
  id: string;
  header: string;
  kind?: "text" | "number" | "status" | "action";
  mobileLayout?: "full-width";
  render: (row: Row) => ReactNode;
  accessibleText: (row: Row) => string;
};

export type DataTablePagination = {
  label: string;
  previousLabel?: string;
  nextLabel?: string;
  onPrevious?: () => void;
  onNext?: () => void;
  isLoading?: boolean;
};

export type DataTableProps<Row> = {
  ariaLabel: string;
  rows: readonly Row[];
  rowKey: (row: Row) => string;
  rowName: (row: Row) => string;
  columns: readonly DataColumn<Row>[];
  selectedKey?: string;
  onSelectionChange?: (row: Row) => void;
  onRowAction?: (row: Row) => void;
  isLoading?: boolean;
  pagination?: DataTablePagination;
  responsiveTo?: "viewport" | "container";
};

export function DataTable<Row>({ ariaLabel, rows, rowKey, rowName, columns, selectedKey, onSelectionChange, onRowAction, isLoading = false, pagination, responsiveTo = "viewport" }: DataTableProps<Row>) {
  const activeKey = rows.some((row) => rowKey(row) === selectedKey) ? selectedKey : rows[0] && rowKey(rows[0]);
  return <div className="cs-data-table" data-responsive-to={responsiveTo}>
    <div className="cs-data-table__viewport">
      <table aria-label={ariaLabel} aria-busy={isLoading || undefined}>
        <thead><tr>{columns.map((column) => <th key={column.id} scope="col" data-kind={column.kind ?? "text"}>{column.header}</th>)}</tr></thead>
        <tbody>{rows.map((row, index) => {
          const key = rowKey(row);
          const selected = selectedKey === key;
          return <tr key={key} tabIndex={onSelectionChange ? (key === activeKey ? 0 : -1) : 0}
            aria-label={rowName(row)} aria-selected={selected || undefined} data-selected={selected || undefined}
            onFocus={(event) => { if (event.target === event.currentTarget) onSelectionChange?.(row); }}
            onClick={(event) => {
              if (isRowSurface(event.target)) event.currentTarget.focus();
            }}
            onDoubleClick={(event) => { if (isRowSurface(event.target)) onRowAction?.(row); }}
            onKeyDown={(event) => {
              if (event.target !== event.currentTarget) return;
              if (onRowAction && (event.key === " " || event.key === "Enter")) {
                event.preventDefault(); onRowAction(row);
              }
              if (onSelectionChange && (event.key === "ArrowDown" || event.key === "ArrowUp")) {
                event.preventDefault();
                const next = Math.max(0, Math.min(rows.length - 1, index + (event.key === "ArrowDown" ? 1 : -1)));
                const target = event.currentTarget.parentElement?.children[next];
                if (target instanceof HTMLElement) target.focus();
              }
            }}>
            {columns.map((column) => <td key={column.id} data-label={column.header} data-kind={column.kind ?? "text"} data-mobile-layout={column.mobileLayout} aria-label={`${column.header}: ${column.accessibleText(row)}`}>{column.render(row)}</td>)}
          </tr>;
        })}</tbody>
      </table>
    </div>
    {isLoading && <span className="cs-sr-only" role="status" aria-label={`Loading ${ariaLabel.toLowerCase()}`}/>} 
    {pagination && <nav className="cs-data-table__pagination" aria-label={pagination.label}>
      {pagination.onPrevious && <Button variant="secondary" isDisabled={pagination.isLoading} onPress={pagination.onPrevious}>{pagination.previousLabel ?? "Previous page"}</Button>}
      {pagination.onNext && <Button variant="secondary" isLoading={pagination.isLoading} onPress={pagination.onNext}>{pagination.nextLabel ?? "Next page"}</Button>}
    </nav>}
  </div>;
}

function isRowSurface(target: EventTarget) {
  return target instanceof Element && !target.closest("button, a, input, select, textarea, [role=button], [contenteditable=true]");
}
