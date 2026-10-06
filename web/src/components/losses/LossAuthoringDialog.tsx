import { useState, type FormEvent } from "react";
import { createLoss, recordLossRecovery } from "../../lossApi";
import type {
  LossCreateInput,
  LossEventType,
  LossRecord,
  LossRecoveryInput,
  LossRecoveryKind,
  LossRecoveryResponse,
  LossTotals,
} from "../../lossTypes";
import { Button, FocusedDialog, Notice, SelectField, TextArea, TextField } from "../ui";
import { formatLossMoney, lossEventLabel, lossMajorToMinor } from "./lossPresentation";

const eventTypes: readonly LossEventType[] = [
  "INTERNAL_FRAUD",
  "EXTERNAL_FRAUD",
  "EMPLOYMENT_PRACTICES",
  "CLIENT_PRODUCTS_BUSINESS_PRACTICES",
  "DAMAGE_TO_PHYSICAL_ASSETS",
  "BUSINESS_DISRUPTION_SYSTEM_FAILURES",
  "EXECUTION_DELIVERY_PROCESS_MANAGEMENT",
  "OTHER",
];

const eventOptions = eventTypes.map((id) => ({ id, label: lossEventLabel(id) }));

type LossEntryDialogProps = {
  organizationScopeID?: string;
  organizationScopeName?: string;
  legalEntityName?: string;
  onClose: () => void;
  onCreated: (loss: LossRecord) => void;
  submit?: (input: LossCreateInput) => Promise<LossRecord>;
};

export function LossEntryDialog({
  organizationScopeID,
  organizationScopeName,
  legalEntityName,
  onClose,
  onCreated,
  submit = createLoss,
}: LossEntryDialogProps) {
  const [code, setCode] = useState("");
  const [title, setTitle] = useState("");
  const [eventType, setEventType] = useState<LossEventType>();
  const [cause, setCause] = useState("");
  const [description, setDescription] = useState("");
  const [grossAmount, setGrossAmount] = useState("");
  const [currency, setCurrency] = useState("NGN");
  const [occurredAt, setOccurredAt] = useState("");
  const [discoveredAt, setDiscoveredAt] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function save(event: FormEvent) {
    event.preventDefault();
    if (busy) return;

    const normalizedCurrency = currency.trim().toUpperCase();
    const grossMinor = lossMajorToMinor(grossAmount, normalizedCurrency);
    const occurred = eventTime(occurredAt);
    const discovered = eventTime(discoveredAt);
    if (!code.trim() || !title.trim() || !eventType || !cause.trim()) {
      setError("Complete the required loss fields.");
      return;
    }
    if (grossMinor === undefined) {
      setError("Enter a valid loss amount for the selected currency.");
      return;
    }
    if (!occurred || !discovered || Date.parse(discovered) < Date.parse(occurred)) {
      setError("Discovery time must be the same as or later than occurrence time.");
      return;
    }

    setBusy(true);
    setError("");
    try {
      const loss = await submit({
        organization_scope_id: organizationScopeID,
        code: code.trim().toUpperCase(),
        title: title.trim(),
        event_type: eventType,
        cause: cause.trim(),
        description: description.trim() || undefined,
        gross_amount_minor: grossMinor,
        currency: normalizedCurrency,
        occurred_at: occurred,
        discovered_at: discovered,
      });
      onCreated(loss);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The loss could not be recorded.");
    } finally {
      setBusy(false);
    }
  }

  const scope = organizationScopeName || legalEntityName || "Current legal entity";

  return <FocusedDialog label="Record loss" closeLabel="Close loss form" onClose={onClose}>
    <form className="loss-authoring" onSubmit={(event) => void save(event)}>
      <header className="loss-authoring__header">
        <span className="eyebrow">Operational loss</span>
        <h2>Record loss</h2>
        <p>{scope}</p>
      </header>

      <div className="loss-authoring__grid">
        <TextField label="Loss code" value={code} onChange={setCode} placeholder="LOSS-2026-001" maxLength={80} isRequired/>
        <TextField label="Title" value={title} onChange={setTitle} placeholder="Short business description" maxLength={200} isRequired/>
        <SelectField label="Event type" value={eventType} placeholder="Choose event type" options={eventOptions} onChange={setEventType} isRequired allowsEmpty={false}/>
        <TextField label="Currency" value={currency} onChange={(value) => setCurrency(value.toUpperCase())} placeholder="NGN" maxLength={3} isRequired/>
        <TextField label="Gross loss" value={grossAmount} onChange={setGrossAmount} type="text" inputMode="decimal" placeholder="0.00" isRequired/>
        <TextField label="Occurred" value={occurredAt} onChange={setOccurredAt} type="datetime-local" isRequired/>
        <TextField label="Discovered" value={discoveredAt} onChange={setDiscoveredAt} type="datetime-local" isRequired/>
        <TextField label="Cause" value={cause} onChange={setCause} placeholder="Primary cause" maxLength={500} isRequired/>
      </div>
      <TextArea label="Description" value={description} onChange={setDescription} placeholder="Optional supporting context" maxLength={4000}/>

      {error && <Notice tone="error">{error}</Notice>}
      <div className="loss-authoring__actions">
        <Button type="submit" isDisabled={busy} isLoading={busy}>Record loss</Button>
        <Button type="button" variant="secondary" onPress={onClose} isDisabled={busy}>Cancel</Button>
      </div>
    </form>
  </FocusedDialog>;
}

type LossRecoveryDialogProps = {
  loss: LossRecord;
  totals: LossTotals;
  onClose: () => void;
  onRecorded: (response: LossRecoveryResponse) => void;
  submit?: (id: string, input: LossRecoveryInput) => Promise<LossRecoveryResponse>;
};

export function LossRecoveryDialog({
  loss,
  totals,
  onClose,
  onRecorded,
  submit = recordLossRecovery,
}: LossRecoveryDialogProps) {
  const canRecover = totals.net_loss_minor > 0;
  const canReverse = totals.recovered_amount_minor > 0;
  const initialKind: LossRecoveryKind = canRecover ? "RECOVERY" : "REVERSAL";
  const [kind, setKind] = useState<LossRecoveryKind>(initialKind);
  const [amount, setAmount] = useState("");
  const [reference, setReference] = useState("");
  const [recoveredAt, setRecoveredAt] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const options: Array<{ id: LossRecoveryKind; label: string; description: string }> = [];
  if (canRecover) options.push({ id: "RECOVERY", label: "Recovery", description: "Reduce the current net loss." });
  if (canReverse) options.push({ id: "REVERSAL", label: "Reversal", description: "Reverse a previously recorded recovery." });

  async function save(event: FormEvent) {
    event.preventDefault();
    if (busy) return;

    const amountMinor = lossMajorToMinor(amount, loss.currency);
    const recovered = eventTime(recoveredAt);
    const ceiling = kind === "RECOVERY" ? totals.net_loss_minor : totals.recovered_amount_minor;
    if (amountMinor === undefined) {
      setError("Enter a valid amount for this loss currency.");
      return;
    }
    if (amountMinor > ceiling) {
      setError(kind === "RECOVERY" ? "Recovery exceeds the current net loss." : "Reversal exceeds the recovered amount.");
      return;
    }
    if (!recovered || Date.parse(recovered) < Date.parse(loss.occurred_at)) {
      setError("Entry time cannot be earlier than the loss occurrence.");
      return;
    }

    setBusy(true);
    setError("");
    try {
      const response = await submit(loss.id, {
        expected_version: loss.version,
        kind,
        amount_minor: amountMinor,
        reference: reference.trim() || undefined,
        recovered_at: recovered,
      });
      onRecorded(response);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The ledger entry could not be recorded.");
    } finally {
      setBusy(false);
    }
  }

  const available = kind === "RECOVERY" ? totals.net_loss_minor : totals.recovered_amount_minor;

  return <FocusedDialog label="Record recovery entry" closeLabel="Close recovery form" onClose={onClose}>
    <form className="loss-authoring" onSubmit={(event) => void save(event)}>
      <header className="loss-authoring__header">
        <span className="eyebrow">{loss.code}</span>
        <h2>Record recovery</h2>
        <p>{loss.title}</p>
      </header>

      <SelectField
        label="Entry"
        value={kind}
        placeholder="Choose entry"
        options={options}
        onChange={(value) => value && setKind(value)}
        isRequired
        allowsEmpty={false}
      />
      <TextField
        label="Amount"
        value={amount}
        onChange={setAmount}
        type="text"
        inputMode="decimal"
        placeholder="0.00"
        description={`Available: ${formatLossMoney(available, loss.currency)}`}
        isRequired
      />
      <TextField label="Recorded at" value={recoveredAt} onChange={setRecoveredAt} type="datetime-local" isRequired/>
      <TextField label="Reference" value={reference} onChange={setReference} placeholder="Settlement, insurer or adjustment reference" maxLength={200}/>

      {error && <Notice tone="error">{error}</Notice>}
      <div className="loss-authoring__actions">
        <Button type="submit" isDisabled={busy} isLoading={busy}>{kind === "REVERSAL" ? "Record reversal" : "Record recovery"}</Button>
        <Button type="button" variant="secondary" onPress={onClose} isDisabled={busy}>Cancel</Button>
      </div>
    </form>
  </FocusedDialog>;
}

function eventTime(value: string): string | undefined {
  if (!value.trim()) return undefined;
  const parsed = new Date(value);
  return Number.isFinite(parsed.getTime()) ? parsed.toISOString() : undefined;
}
