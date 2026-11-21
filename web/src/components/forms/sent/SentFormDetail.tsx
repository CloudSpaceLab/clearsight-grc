import type { DistributionDetail, DistributionRecipient } from "../../../formsDistributionApi";
import { Button, Notice, StatusBadge } from "../../ui";
import { accessPolicyLabel, distributionStatusLabel, distributionStatusTone, formatDistributionDateTime } from "./distributionPresentation";

type LifecycleAction = "lock" | "reopen" | "revoke";

export function SentFormDetail({ detail, error, busy, onLifecycle, onAmend, onSupersede }: { detail: DistributionDetail; error?: string; busy?: string; onLifecycle: (action: LifecycleAction) => void; onAmend: () => void; onSupersede: () => void }) {
  const to = detail.recipients.filter((value) => value.role === "TO" && value.state !== "REVOKED").length;
  const cc = detail.recipients.filter((value) => value.role === "CC" && value.state !== "REVOKED").length;
  const completed = detail.recipients.filter((value) => value.role === "TO" && value.state === "COMPLETED").length;
  const distribution = detail.distribution;
  return <div className="forms-sent-detail">
    <p className="forms-sent-detail__type">{subjectLabel(distribution.subject_type)}</p>
    <h3>{distribution.title}</h3>
    <p>{distribution.purpose}</p>
    {error && <Notice tone="error">{error} Review the selected sent form again.</Notice>}
    <dl>
      <div><dt>Status</dt><dd><StatusBadge tone={distributionStatusTone[distribution.status]}>{distributionStatusLabel[distribution.status]}</StatusBadge></dd></div>
      <div><dt>Recipients</dt><dd>{to} To · {cc} CC</dd></div>
      <div><dt>Completed</dt><dd>{completed}/{to}</dd></div>
      <div><dt>Deadline</dt><dd>{formatDistributionDateTime(distribution.deadline)}</dd></div>
      <div><dt>Access</dt><dd>{accessPolicyLabel[distribution.access_policy]}</dd></div>
      <div><dt>Workspace</dt><dd>{distributionStatusLabel[detail.workspace.status]} · v{detail.workspace.version}</dd></div>
      <div><dt>Form</dt><dd>{distribution.form_template_id} · v{distribution.form_template_version}</dd></div>
    </dl>
    <section className="forms-sent-detail__recipients" aria-labelledby="sent-form-recipients">
      <h4 id="sent-form-recipients">Recipients</h4>
      {detail.recipients.length ? <ul aria-label="Recipients">{detail.recipients.map((recipient) => <li key={recipient.id}>
        <div><strong>{recipientName(recipient)}</strong><span>{recipient.role === "CC" ? "Copy recipient" : "Respondent"}</span></div>
        <StatusBadge tone={recipient.state === "REVOKED" ? "warning" : "neutral"}>{recipientState(recipient)}</StatusBadge>
      </li>)}</ul> : <p>No recipients recorded.</p>}
    </section>
    <div className="forms-sent-detail__actions">
      {distribution.status === "OPEN" && <Button isLoading={busy === "lock"} onPress={() => onLifecycle("lock")}>Lock responses</Button>}
      {distribution.status === "LOCKED" && <Button isLoading={busy === "reopen"} onPress={() => onLifecycle("reopen")}>Reopen responses</Button>}
      {!(["REVOKED", "COMPLETED", "SUPERSEDED"] as const).includes(distribution.status as "REVOKED") && <Button variant="destructive" isLoading={busy === "revoke"} onPress={() => onLifecycle("revoke")}>Revoke access</Button>}
      {distribution.status === "OPEN" && <Button variant="secondary" onPress={onAmend}>Amend distribution</Button>}
      {distribution.status === "OPEN" && <Button variant="secondary" onPress={onSupersede}>Replace form version</Button>}
    </div>
  </div>;
}

function recipientName(recipient: DistributionRecipient) {
  return recipient.contact_label?.trim() || (recipient.type === "INTERNAL_PRINCIPAL" && recipient.principal_id ? `Name unavailable (${recipient.principal_id})` : "External recipient");
}

function recipientState(recipient: DistributionRecipient) {
  return ({ PENDING: "Pending", DELIVERED: "Delivered", VERIFIED: "Access verified", REVOKED: "Access revoked", COMPLETED: recipient.role === "TO" ? "Response submitted" : "Completed" })[recipient.state] ?? "Status unavailable";
}

function subjectLabel(type: string) {
  return ({ PROGRAM: "Program", MATTER: "Issue or change", VENDOR_RELATIONSHIP: "Vendor service", VENDOR: "Vendor", CONTROL: "Control" } as Record<string, string>)[type] ?? "Subject";
}
