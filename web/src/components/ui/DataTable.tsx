import type { ReactNode } from "react";
import { Button } from "./Button";

export type DataColumn<Row> = {
  id: string;
  header: string;
  kind?: "text" | "number" | "status" | "action";
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
  isLoading?: boolean;
  pagination?: DataTablePagination;
};

export function DataTable<Row>({ ariaLabel, rows, rowKey, rowName, columns, selectedKey, isLoading = false, pagination }: DataTableProps<Row>) {
  return <div className="cs-data-table">
    <div className="cs-data-table__viewport">
      <table aria-label={ariaLabel} aria-busy={isLoading || undefined}>
        <thead><tr>{columns.map((column) => <th key={column.id} scope="col" data-kind={column.kind ?? "text"}>{column.header}</th>)}</tr></thead>
        <tbody>{rows.map((row) => {
          const key = rowKey(row);
          const selected = selectedKey === key;
          return <tr key={key} tabIndex={0} aria-label={rowName(row)} aria-selected={selected || undefined} data-selected={selected || undefined}>
            {columns.map((column) => <td key={column.id} data-label={column.header} data-kind={column.kind ?? "text"} aria-label={`${column.header}: ${column.accessibleText(row)}`}>{column.render(row)}</td>)}
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
