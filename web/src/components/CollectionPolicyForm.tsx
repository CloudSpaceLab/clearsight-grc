import { useState } from "react";
import type { FormEvent } from "react";
import type { CollectionPolicy } from "../monitoringTypes";

type Props = {
  initialValue?: CollectionPolicy;
  onSave: (policy: CollectionPolicy) => Promise<void>;
  onCancel: () => void;
  submitLabel?: string;
};

export function CollectionPolicyForm({ initialValue, onSave, onCancel, submitLabel = "Add collection to Program" }: Props) {
  const [validityMonths, setValidityMonths] = useState(String(initialValue?.validity_months ?? 12));
  const [renewalWindowDays, setRenewalWindowDays] = useState(String(initialValue?.renewal_window_days ?? 30));
  const [reminderCount, setReminderCount] = useState(String(initialValue?.reminder_count ?? 3));
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const policy = { validity_months: Number(validityMonths), renewal_window_days: Number(renewalWindowDays), reminder_count: Number(reminderCount) };

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    if (!Number.isInteger(policy.validity_months) || policy.validity_months < 1 || policy.validity_months > 120) {
      setError("Response expiry must be between 1 and 120 months.");
      return;
    }
    if (!Number.isInteger(policy.renewal_window_days) || policy.renewal_window_days < 1 || policy.renewal_window_days > 90) {
      setError("Renewal must start between 1 and 90 days before expiry.");
      return;
    }
    if (policy.renewal_window_days > policy.validity_months * 28 - 1) {
      setError("Renewal must start before the response can expire.");
      return;
    }
    if (!Number.isInteger(policy.reminder_count) || policy.reminder_count < 1 || policy.reminder_count > 5) {
      setError("Choose between 1 and 5 reminders.");
      return;
    }
    setSaving(true);
    try {
      await onSave(policy);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The collection schedule could not be saved.");
    } finally {
      setSaving(false);
    }
  }

  const reminderWord = policy.reminder_count === 1 ? "reminder" : "reminders";
  return <form className="collection-policy-form" onSubmit={submit}>
    <div className="collection-policy-grid">
      <label htmlFor="collection-validity">Response expires after</label>
      <div><input id="collection-validity" type="number" min="1" max="120" value={validityMonths} onChange={(event) => setValidityMonths(event.target.value)}/><span>months</span></div>
      <label htmlFor="collection-renewal-window">Renewal starts</label>
      <div><input id="collection-renewal-window" type="number" min="1" max="90" value={renewalWindowDays} onChange={(event) => setRenewalWindowDays(event.target.value)}/><span>days before expiry</span></div>
      <label htmlFor="collection-reminders">Reminders during renewal</label>
      <div><input id="collection-reminders" type="number" min="1" max="5" value={reminderCount} onChange={(event) => setReminderCount(event.target.value)}/><span>{reminderWord}</span></div>
    </div>
    <p className="collection-policy-preview">Responses will be renewed {policy.renewal_window_days} days before they reach {policy.validity_months} months old. The initial request is followed by up to {policy.reminder_count} {reminderWord}.</p>
    {error && <p className="inline-form-error" role="alert">{error}</p>}
    <div className="monitoring-form-actions"><button className="text-button" type="button" onClick={onCancel}>Cancel</button><button className="primary-button" type="submit" disabled={saving}>{saving ? "Saving…" : submitLabel}</button></div>
  </form>;
}
