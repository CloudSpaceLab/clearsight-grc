import type { FormTemplate } from "../../../monitoringTypes";
import { hasRequiredSignOff, type AuthoringField, type FormQualityIssue } from "../formAuthoring";
import type { BuilderSelection } from "./builderSelection";

type Props = {
  issues: FormQualityIssue[];
  fields: AuthoringField[];
  initialValue?: FormTemplate;
  onFix: (selection: BuilderSelection) => void;
  onAddRequiredSignOff: () => void;
  onClose: () => void;
};

export function reviewIssueCount(issues: FormQualityIssue[], fields: AuthoringField[]): number {
  return issues.length + (hasRequiredSignOff(fields) ? 0 : 1);
}

export function FormReviewDrawer({ issues, fields, initialValue, onFix, onAddRequiredSignOff, onClose }: Props) {
  const blocking = issues.filter((issue) => issue.blocking);
  const recommended = hasRequiredSignOff(fields) ? 0 : 1;

  return <div className="form-review-backdrop" onMouseDown={onClose}>
    <aside className="form-review-drawer" aria-label="Form review" onMouseDown={(event) => event.stopPropagation()}>
      <header className="form-review-heading">
        <div><span className="eyebrow">Review</span><h3>Approval readiness</h3><p>{blocking.length} blocking · {recommended} recommended</p></div>
        <button type="button" className="icon-button" aria-label="Close review" onClick={onClose}>×</button>
      </header>

      <div className="form-review-content">
        {blocking.length > 0 ? <section aria-labelledby="blocking-review-title">
          <h4 id="blocking-review-title">Blocking</h4>
          <ul className="form-review-list">
            {blocking.map((issue) => <li key={issue.id}>
              <span className="form-review-marker blocking" aria-hidden="true"/>
              <span>{issue.message}</span>
              <button type="button" onClick={() => { onFix(issueSelection(issue)); onClose(); }}>Fix →</button>
            </li>)}
          </ul>
        </section> : <section className="form-review-ready" aria-label="Approval checks passed">
          <strong>Deterministic approval checks pass</strong>
          <p>No blocking contract issue is present in the current draft.</p>
        </section>}

        <section aria-labelledby="recommended-review-title">
          <h4 id="recommended-review-title">Recommended</h4>
          {recommended ? <ul className="form-review-list">
            <li>
              <span className="form-review-marker" aria-hidden="true"/>
              <span><strong>No required sign-off yet</strong><small>Add one only when respondents must explicitly attest to the submitted information.</small></span>
              <button type="button" onClick={onAddRequiredSignOff}>Add →</button>
            </li>
          </ul> : <div className="form-review-recommendation-complete"><strong>Required sign-off included</strong><span>A required attestation or signature is present.</span></div>}
        </section>
      </div>

      <footer className="form-review-footer">
        <span>{initialValue ? `Revision v${initialValue.version}` : "New draft"}</span>
        <span>{initialValue?.status ? formatState(initialValue.status) : "Draft"}</span>
      </footer>
    </aside>
  </div>;
}

function issueSelection(issue: FormQualityIssue): BuilderSelection {
  if (issue.fieldID) return { kind: "field", fieldID: issue.fieldID };
  if (issue.sectionID) return { kind: "section", sectionID: issue.sectionID };
  return { kind: "overview" };
}

function formatState(value: string): string {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
