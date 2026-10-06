import type { HomeMetricMember, HomeMetricMemberPage } from "../../metricApi";
import { Button, DataTable, EmptyState, Notice, type DataColumn } from "../ui";

type Props = {
  label: string;
  state: "loading" | "live" | "unavailable";
  page: HomeMetricMemberPage | null;
  hasPrevious: boolean;
  onPrevious: () => void;
  onNext: () => void;
  onRetry: () => void;
  onOpenMatter?: (id: string) => void;
  onOpenProgram?: (id: string) => void;
  onOpenRisk?: (id: string) => void;
  onOpenLoss?: (id: string) => void;
};

export function MetricMemberDrill({
  label,
  state,
  page,
  hasPrevious,
  onPrevious,
  onNext,
  onRetry,
  onOpenMatter,
  onOpenProgram,
  onOpenRisk,
  onOpenLoss,
}: Props) {
  const columns: readonly DataColumn<HomeMetricMember>[] = [
    {
      id: "record",
      header: "Record",
      mobileLayout: "full-width",
      render: (item) => <span className="oversight-member-record"><strong>{item.target_title}</strong><small>{item.accessible ? recordTypeLabel(item.target_type) : "Access changed"}</small></span>,
      accessibleText: (item) => `${item.target_title}, ${item.accessible ? recordTypeLabel(item.target_type) : "access changed"}`,
    },
    {
      id: "state",
      header: "Snapshot state",
      render: (item) => humanize(item.state),
      accessibleText: (item) => humanize(item.state),
    },
  ];

  if (state === "loading" && !page) {
    return <p className="oversight-result-count" role="status">Loading exact snapshot members…</p>;
  }
  if (state === "unavailable") {
    return <Notice tone="warning">
      <span>Exact snapshot detail is unavailable. The card value is unchanged.</span>{" "}
      <Button variant="secondary" size="compact" onPress={onRetry}>Try again</Button>
    </Notice>;
  }
  if (!page) return null;

  if (page.count === 0) {
    return <EmptyState
      population={`Exact retained members for ${label.toLowerCase()}`}
      title="No records were counted in this snapshot"
      description="The retained metric membership is empty for this card revision."
    />;
  }

  return <>
    <p className="oversight-result-count" aria-live="polite">
      {page.count} exact {page.count === 1 ? "record" : "records"} in this metric snapshot.
    </p>
    <DataTable
      ariaLabel={`Exact ${label.toLowerCase()} snapshot members`}
      rows={page.items}
      rowKey={(item) => item.member_id}
      rowName={(item) => `${item.target_title}, ${recordTypeLabel(item.target_type)}`}
      columns={columns}
      onRowAction={(item) => {
        if (!item.accessible || !item.target_id) return;
        if (item.target_type === "MATTER") onOpenMatter?.(item.target_id);
        else if (item.target_type === "PROGRAM") onOpenProgram?.(item.target_id);
        else if (item.target_type === "RISK") onOpenRisk?.(item.target_id);
        else onOpenLoss?.(item.target_id);
      }}
      isRowActionDisabled={(item) => !item.accessible || !item.target_id || !hasOpenAction(item.target_type, { onOpenMatter, onOpenProgram, onOpenRisk, onOpenLoss })}
      rowActionLabel="Open record"
      isLoading={state === "loading"}
      pagination={(hasPrevious || page.next_cursor) ? {
        label: `${label} member pages`,
        previousLabel: "Previous page",
        nextLabel: "Next page",
        onPrevious: hasPrevious ? onPrevious : undefined,
        onNext: page.next_cursor ? onNext : undefined,
        isLoading: state === "loading",
      } : undefined}
    />
    {page.items.some((item) => !item.accessible) &&
      <Notice tone="info">Some records are still part of this historical count but their current access has changed.</Notice>}
    {page.items.some((item) => item.accessible && !hasOpenAction(item.target_type, { onOpenMatter, onOpenProgram, onOpenRisk, onOpenLoss })) &&
      <Notice tone="warning">Some record drills are not available from this surface.</Notice>}
  </>;
}

function recordTypeLabel(value: HomeMetricMember["target_type"]) {
  if (value === "PROGRAM") return "Program";
  if (value === "RISK") return "Risk";
  if (value === "LOSS") return "Loss";
  return "Issue or change";
}

function hasOpenAction(
  value: HomeMetricMember["target_type"],
  actions: Pick<Props, "onOpenMatter" | "onOpenProgram" | "onOpenRisk" | "onOpenLoss">,
) {
  if (value === "MATTER") return Boolean(actions.onOpenMatter);
  if (value === "PROGRAM") return Boolean(actions.onOpenProgram);
  if (value === "RISK") return Boolean(actions.onOpenRisk);
  return Boolean(actions.onOpenLoss);
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
