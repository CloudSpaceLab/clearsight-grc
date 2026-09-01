import type { FormLibraryFacets, FormTemplateQuery } from "../../../formsTypes";
import type { LifecycleStatus } from "../../../monitoringTypes";
import { ScopeBar, type ScopeItem } from "../../ui";
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
  const items: ScopeItem<"ALL" | LifecycleStatus>[] = [
    { id: "ALL", label: "All", count: total },
    ...scopes.flatMap((scope) => {
      const count = status[scope.status] ?? 0;
      return count === 0 && query.status !== scope.status ? [] : [{ id: scope.status, label: scope.label, count }];
    }),
  ];

  return <div className="forms-status-scopes"><ScopeBar
      ariaLabel="Form status scopes"
      items={items}
      selectedKey={query.status ?? "ALL"}
      onSelectionChange={(selected) => onChange(withStatusScope(query, selected === "ALL" ? undefined : selected))}
    /></div>;
}
