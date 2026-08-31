import type { DistributionDueState, DistributionQuery, DistributionStatus } from "../../../formsDistributionApi";
import { FilterBar, SelectField, TextField } from "../../ui";
import { distributionDueStateLabel, distributionStatusLabel } from "./distributionPresentation";

const statusOptions = (Object.keys(distributionStatusLabel) as DistributionStatus[]).map((id) => ({ id, label: distributionStatusLabel[id] }));
const dueOptions = (Object.keys(distributionDueStateLabel) as DistributionDueState[]).map((id) => ({ id, label: distributionDueStateLabel[id] }));

export function SentFormsFilters({ query, resultCount, onChange, onClear }: { query: DistributionQuery; resultCount?: number; onChange: (patch: Partial<DistributionQuery>) => void; onClear: () => void }) {
  return <FilterBar
    label="Sent-form filters"
    resultCount={resultCount}
    resultLabel={(count) => `${count} sent ${count === 1 ? "form" : "forms"} on this page`}
    clearLabel="Clear sent-form filters"
    onClear={onClear}
    fields={<>
      <SelectField label="Status" value={query.status} placeholder="All states" options={statusOptions} onChange={(status) => onChange({ status })}/>
      <SelectField label="Due state" value={query.due_state} placeholder="Any deadline" options={dueOptions} onChange={(due_state) => onChange({ due_state })}/>
      <TextField label="Subject type" value={query.subject_type ?? ""} onChange={(subject_type) => onChange({ subject_type: subject_type.trim() ? subject_type : undefined })}/>
      <TextField label="Subject ID" value={query.subject_id ?? ""} onChange={(subject_id) => onChange({ subject_id: subject_id.trim() ? subject_id : undefined })}/>
      <TextField label="Owner" value={query.owner ?? ""} onChange={(owner) => onChange({ owner: owner.trim() ? owner : undefined })}/>
    </>}
  />;
}
