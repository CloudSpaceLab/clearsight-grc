import type { FormLibraryFacets, FormTemplateQuery } from "../../../formsTypes";
import type { LifecycleStatus } from "../../../monitoringTypes";
import { withStatusScope } from "./filterModel";

const scopes: readonly { status: LifecycleStatus; label: string }[] = [
  { status: "DRAFT", label: "Draft" },
  { status: "PENDING_APPROVAL", label: "Awaiting approval" },
  { status: "ACTIVE", label: "Active" },
  { status: "PAUSED", label: "Paused" },
  { status: "REJECTED", label: "Rejected" },
  { status: "RETIRED", label: "Retired" },
];

type Props = {
  query: FormTemplateQuery;
  facets?: FormLibraryFacets;
  onChange: (query: FormTemplateQuery) => void;
};

export function FormStatusScopes({ query, facets, onChange }: Props) {
  const status = facets?.status;
  if (!status) return null;
  const total = scopes.reduce((sum, scope) => sum + (status[scope.status] ?? 0), 0);

  return <nav className="forms-status-scopes" aria-label="Form status scopes">
    <button
      type="button"
      className={!query.status ? "active" : ""}
      aria-current={!query.status ? "page" : undefined}
      aria-label={`All ${total}`}
      onClick={() => onChange(withStatusScope(query))}
    >
      <span>All</span><strong>{total}</strong>
    </button>
    {scopes.map((scope) => {
      const count = status[scope.status] ?? 0;
      if (count === 0 && query.status !== scope.status) return null;
      const active = query.status === scope.status;
      return <button
        type="button"
        className={active ? "active" : ""}
        aria-current={active ? "page" : undefined}
        aria-label={`${scope.label} ${count}`}
        key={scope.status}
        onClick={() => onChange(withStatusScope(query, scope.status))}
      >
        <span>{scope.label}</span><strong>{count}</strong>
      </button>;
    })}
  </nav>;
}
