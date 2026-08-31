import type { FormTemplate } from "../../../monitoringTypes";
import { FocusedSheet } from "../../FocusedSheet";
import { hasRequiredSignOff, type AuthoringField, type FormQualityIssue } from "../formAuthoring";

type Props = {
  issues: FormQualityIssue[];
  fields: AuthoringField[];
  initialValue?: FormTemplate;
  onFix: (issue: FormQualityIssue) => void;
  onAddRequiredSignOff: () => void;
  onClose: () => void;
};

export function reviewIssueCount(issues: FormQualityIssue[], fields: AuthoringField[]): number {
  return issues.length + (hasRequiredSignOff(fields) ? 0 : 1);
}

export function FormReviewDrawer({ issues, fields, initialValue, onFix, onAddRequiredSignOff, onClose }: Props) {
  const blocking = issues.filter((issue) => issue.blocking);
  const recommended = hasRequiredSignOff(fields) ? 0 : 1;

  return <FocusedSheet
    label="Form review"
    closeLabel="Close review"
    panelClassName="form-review-drawer"
    backdropClassName="form-review-backdrop"
    onClose={onClose}
  >
    <header className="form-review-heading">
      <div><span className="eyebrow">Review</span><h3>Approval readiness</h3><p>{blocking.length} blocking · {recommended} recommended</p></div>
    </header>

    <div className="form-review-content">
      {blocking.length > 0 ? <section aria-labelledby="blocking-review-title">
        <h4 id="blocking-review-title">Blocking</h4>
        <ul className="form-review-list">
          {blocking.map((issue) => <li key={issue.id}>
            <span className="form-review-marker blocking" aria-hidden="true"/>
            <span>{issue.message}</span>
            <button type="button" onClick={() => onFix(issue)}>Fix →</button>
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
  </FocusedSheet>;
}

function formatState(value: string): string {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
