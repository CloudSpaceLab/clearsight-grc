import { useEffect, useState } from "react";
import { loadFormTemplateItem } from "../formsApi";
import type { FormLibraryItem } from "../formsTypes";
import { FormActions } from "./FormActions";
import { APIQuestionWizard } from "./APIQuestionWizard";
import "./form-detail.css";

export function FormDetail({ id, approvalVersion }: { id: string; approvalVersion?: number }) {
  const [item, setItem] = useState<FormLibraryItem>();
  const [failed, setFailed] = useState(false);
  const [retry, setRetry] = useState(0);
  const [receipt, setReceipt] = useState<{ id: string; version: number; status: "ACTIVE" | "REJECTED" }>();
  useEffect(() => { let active = true; setItem(undefined); setFailed(false); loadFormTemplateItem(id).then((value) => { if (active) { if (value.template?.id !== id) setFailed(true); else setItem(value); } }).catch(() => { if (active) setFailed(true); }); return () => { active = false; }; }, [id, retry]);
  if (failed) return <section>{receipt && <p role="status">{receipt.status === "ACTIVE" ? "Form approved." : "Form rejected."}</p>}<h1>Form unavailable</h1><button onClick={() => setRetry(retry + 1)}>Try again</button></section>;
  if (!item) return <p role="status">Loading form…</p>;
  const { template } = item;
  const approvalChanged = approvalVersion !== undefined && template.version !== approvalVersion;
  const approvalClosed = approvalVersion !== undefined && !approvalChanged && template.status !== "PENDING_APPROVAL";
  const approvalUnavailable = approvalChanged || approvalClosed;
  const receiptMatches = receipt?.id === template.id && receipt.version === template.version && receipt.status === template.status;
  const canTestQuestions = template.status === "DRAFT" && item.authority_available && item.operations?.some((operation) => operation.command === "forms.template.revise" && operation.can_act);
  return <section className="v-form-detail"><a className="v-back" href="#/forms">← Forms</a><header className="v-page-heading"><div><p className="v-eyebrow">Version {template.version}</p><h1>{template.name}</h1></div><span className="v-status">{template.status.toLowerCase().replaceAll("_", " ")}</span></header>{receiptMatches && <p role="status">{receipt.status === "ACTIVE" ? "Form approved." : "Form rejected."}</p>}{approvalUnavailable && !receiptMatches && <div className="v-panel" role="alert"><p>{approvalChanged ? "Form version changed. Review the latest version before deciding." : "This form is no longer awaiting approval."}</p><a href={`#/forms/${encodeURIComponent(template.id)}`}>Review latest form</a></div>}{!approvalUnavailable && item.authority_available && item.operations?.some((operation) => operation.command === "forms.template.revise" && operation.can_act) && <p><a className="v-primary" href={`#/forms/${encodeURIComponent(template.id)}/edit`}>Edit form</a></p>}{!approvalUnavailable && <FormActions item={item} onSaved={(result) => { if (approvalVersion !== undefined && template.status === "PENDING_APPROVAL" && (result.status === "ACTIVE" || result.status === "REJECTED")) setReceipt({ id: result.id, version: result.version, status: result.status }); }} onReload={() => setRetry((value) => value + 1)}/>}<div className="v-form-detail-layout"><section className="v-panel v-form-detail-questions" aria-labelledby="v-form-detail-questions"><h2 id="v-form-detail-questions">Questions</h2>{!template.fields ? <p>Questions unavailable.</p> : template.fields.length === 0 ? <p>No questions recorded.</p> : <ul className="v-attention">{template.fields.map((field) => <li key={field.id}><div><strong>{field.label}</strong><small>{field.type.replaceAll("_", " ")}{field.required ? " · Required" : ""}</small>{!approvalUnavailable && canTestQuestions && (field.type === "yes_no" || field.type === "decimal" || field.type === "percentage") && <APIQuestionWizard formID={template.id} version={template.version} fieldID={field.id} label={field.label} resultType={field.type === "yes_no" ? "BOOLEAN" : "NUMBER"}/>}</div></li>)}</ul>}</section><section className="v-panel v-form-detail-facts" aria-labelledby="v-form-detail-facts"><h2 id="v-form-detail-facts">Form details</h2><dl><div><dt>Purpose</dt><dd>{template.purpose || "Not recorded"}</dd></div><div><dt>Version</dt><dd>{template.version}</dd></div><div><dt>Questions</dt><dd>{Array.isArray(template.fields) ? template.fields.length : "Unavailable"}</dd></div></dl></section></div></section>;
}
