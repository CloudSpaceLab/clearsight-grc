import { useEffect, useState, type ReactNode } from "react";
import type { CaptureAnswerValue, CaptureField } from "../../types";

type Props = {
  field: CaptureField;
  value?: CaptureAnswerValue;
  editor: ReactNode;
  onChange: (value: CaptureAnswerValue) => void;
};

export function HeldValueField({ field, value, editor, onChange }: Props) {
  const baseline = field.record_baseline;
  const replacement = field.collection_intent === "REPLACE_HELD_DOCUMENT";
  const changed = replacement ? Boolean(value?.document?.artifact_id) : Boolean(value?.text !== undefined && value.text !== baseline?.display_value);
  const [editing, setEditing] = useState(changed);

  useEffect(() => setEditing(changed), [field.id, changed]);
  if (!baseline) return <>{editor}</>;

  function confirm() {
    if (!replacement) onChange({ text: baseline!.display_value });
    setEditing(false);
  }

  return <section className="held-value-field" aria-labelledby={`held-value-${field.id}`}>
    <div className="held-value-heading"><div><span className="eyebrow">Current held record</span><h3 id={`held-value-${field.id}`}>{field.label}</h3></div><span className={baseline.expires_at && Date.parse(baseline.expires_at) <= Date.now() ? "held-value-state expired" : "held-value-state"}>{baseline.expires_at ? `Expires ${formatDate(baseline.expires_at)}` : `Version ${baseline.record_version}`}</span></div>
    <p className="held-value-current">{baseline.display_value || "No value is currently recorded."}</p>
    <p className="held-value-source">{baseline.source_label} · checked {formatDate(baseline.observed_or_confirmed_at)}</p>
    {!editing && <div className="held-value-actions">{!replacement && <button type="button" className="primary-button" onClick={confirm}>Confirm this is accurate</button>}<button type="button" className={replacement ? "primary-button" : "secondary-button"} onClick={() => setEditing(true)}>{replacement ? "Replace held document" : "Update this information"}</button></div>}
    {editing && <div className="held-value-editor">{editor}<button type="button" className="text-button" onClick={() => { onChange(replacement ? {} : { text: baseline.display_value }); setEditing(false); }}>Keep current record</button></div>}
    {!replacement && value?.text === baseline.display_value && !editing && <p className="held-value-confirmed" role="status">Confirmed against the held version above.</p>}
  </section>;
}

function formatDate(value: string) {
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(parsed)) : value;
}
