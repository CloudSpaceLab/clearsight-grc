import type { CaptureAnswerValue, CaptureDocumentAnswer, CaptureField } from "../../types";
import { FileDropzone } from "../FileDropzone";
import { SignatureCapture } from "../SignatureCapture";
import { answerText, answerValues, normalizeFieldType } from "./contract";

export type CaptureAttachment = { id?: string; file_name: string; media_type: string; size_bytes: number; preview_url?: string };

type Props = {
  field: CaptureField;
  value?: CaptureAnswerValue;
  attachments?: CaptureAttachment[];
  uploading: boolean;
  external: boolean;
  error?: string;
  onChange: (value: CaptureAnswerValue) => void;
  onUpload: (files: File[], previewURL?: string) => void;
  onRemove: (attachmentID: string) => void;
};

export function CaptureFieldControl({ field, value, attachments = [], uploading, external, error, onChange, onUpload, onRemove }: Props) {
  const type = normalizeFieldType(field.type);
  if (!type) return null;
  const text = answerText(value);
  const label = `${field.label}${field.required ? " *" : ""}`;
  const inputID = `capture-input-${field.id}`;
  const helpID = `capture-help-${field.id}`;
  const errorID = `capture-error-${field.id}`;
  const describedBy = [field.description || type === "currency" || type === "multi_select" ? helpID : "", error ? errorID : ""].filter(Boolean).join(" ") || undefined;
  const common = { id: inputID, required: field.required, "aria-invalid": Boolean(error), "aria-describedby": describedBy } as const;
  const constraints = field.constraints ?? {};
  const inputHelp = [field.description, type === "currency" ? `Enter the amount in ${constraints.currency || "the request currency"}.` : ""].filter(Boolean).join(" ");

	const attachment = attachments[0];
	if (type === "signature") return <SignatureCapture value={attachment?.preview_url} label={label} attestation={field.attestation || field.description} busy={uploading} onCapture={(file, previewURL) => onUpload([file], previewURL)}/>;
	if (type === "vendor_document") return <VendorDocumentControl field={field} value={value} attachment={attachment} uploading={uploading} error={error} onChange={onChange} onUpload={(file) => onUpload([file])}/>;
  if (type === "photo" || type === "file") {
    const photo = type === "photo";
    const accept = normalizedAcceptedFormats(field.accepted_formats) ?? (photo ? "image/*" : undefined);
    return <div className="capture-field"><FileDropzone
      label={label}
      description={fileDescription(field)}
      accept={accept}
      capture={photo ? "environment" : undefined}
      compact={!photo}
      busy={uploading}
	  multiple={!photo && (field.constraints?.max_files ?? 1) > 1}
	  actionLabel={photo ? "Take or add photo" : (field.constraints?.max_files ?? 1) > 1 ? "Add files" : "Choose file"}
      replaceLabel={photo ? "Replace photo" : "Replace file"}
	  fileName={(photo || (field.constraints?.max_files ?? 1) === 1) ? attachment?.file_name : undefined}
	  fileSize={(photo || (field.constraints?.max_files ?? 1) === 1) ? attachment?.size_bytes : undefined}
      previewUrl={attachment?.preview_url}
	  onSelect={(file) => onUpload([file])}
	  onSelectMany={(files) => onUpload(files)}
	/>{!photo && attachments.length > 0 && <AttachmentList attachments={attachments} onRemove={onRemove}/>} {error && <p id={errorID} className="field-error">{error}</p>}</div>;
  }
  if (type === "yes_no") return <ChoiceField field={field} values={["Yes", "No"]} selected={text ? [text] : []} describedBy={describedBy} error={error} onSelect={(selected) => onChange({ text: selected[0] ?? "" })}/>;
  if (type === "single_select" && (field.options?.length ?? 0) <= 4) return <ChoiceField field={field} values={field.options ?? []} selected={text ? [text] : []} describedBy={describedBy} error={error} onSelect={(selected) => onChange({ text: selected[0] ?? "" })}/>;
  if (type === "multi_select") return <ChoiceField field={field} values={field.options ?? []} selected={answerValues(value)} multiple describedBy={describedBy} error={error} onSelect={(values) => onChange({ values })}/>;
  if (type === "single_select") return <label className="capture-field" htmlFor={inputID}><span>{label}</span>{field.description && <small id={helpID} className="field-help">{field.description}</small>}<select {...common} value={text} onChange={(event) => onChange({ text: event.target.value })}><option value="">Choose one</option>{field.options?.map((option) => <option key={option}>{option}</option>)}</select>{error && <small id={errorID} className="field-error">{error}</small>}</label>;
  if (type === "checkbox" || type === "attestation") {
    const checkboxLabel = type === "attestation" ? field.attestation || field.description || field.label : field.label;
    return <fieldset className="capture-field capture-check-field" aria-describedby={describedBy}><legend>{label}</legend>{field.description && type !== "attestation" && <p id={helpID} className="field-help">{field.description}</p>}<label className="capture-check-option"><input {...common} type="checkbox" checked={text === "true"} onChange={(event) => onChange({ text: event.target.checked ? "true" : "" })}/><span>{checkboxLabel}</span></label>{error && <p id={errorID} className="field-error">{error}</p>}</fieldset>;
  }
  if (type === "long_text" && external && !field.required) return <details className="capture-optional-field"><summary>{field.label}</summary><label className="capture-field" htmlFor={inputID}>{field.description && <small id={helpID} className="field-help">{field.description}</small>}<textarea {...common} aria-label={field.label} minLength={constraints.min_length} maxLength={constraints.max_length} value={text} onChange={(event) => onChange({ text: event.target.value })}/>{error && <small id={errorID} className="field-error">{error}</small>}</label></details>;
  if (type === "long_text") return <label className="capture-field" htmlFor={inputID}><span>{label}</span>{field.description && <small id={helpID} className="field-help">{field.description}</small>}<textarea {...common} minLength={constraints.min_length} maxLength={constraints.max_length} value={text} onChange={(event) => onChange({ text: event.target.value })}/>{error && <small id={errorID} className="field-error">{error}</small>}</label>;

  const inputType = type === "email" ? "email" : type === "telephone" ? "tel" : type === "url" ? "url" : type === "date" ? "date" : ["integer", "decimal", "percentage", "currency"].includes(type) ? "number" : "text";
  const inputMode = type === "integer" ? "numeric" : ["decimal", "percentage", "currency"].includes(type) ? "decimal" : type === "telephone" ? "tel" : type === "email" ? "email" : type === "url" ? "url" : undefined;
  return <label className="capture-field" htmlFor={inputID}><span>{label}</span>{inputHelp && <small id={helpID} className="field-help">{inputHelp}</small>}<input
    {...common}
    type={inputType}
    inputMode={inputMode}
    minLength={constraints.min_length}
    maxLength={constraints.max_length}
    min={inputType === "date" ? constraints.min_date : constraints.minimum ?? (type === "percentage" ? 0 : undefined)}
    max={inputType === "date" ? constraints.max_date : constraints.maximum ?? (type === "percentage" ? 100 : undefined)}
    step={type === "integer" ? constraints.step ?? 1 : constraints.step ?? (constraints.decimal_precision !== undefined ? 10 ** -constraints.decimal_precision : undefined)}
    value={text}
    onChange={(event) => onChange({ text: event.target.value })}
  />{error && <small id={errorID} className="field-error">{error}</small>}</label>;
}

function AttachmentList({ attachments, onRemove }: { attachments: CaptureAttachment[]; onRemove: (attachmentID: string) => void }) {
	return <ul className="capture-attachment-list" aria-label="Selected files">{attachments.map((attachment) => <li key={attachment.id ?? attachment.file_name}><span><strong>{attachment.file_name}</strong><small>{formatBytes(attachment.size_bytes)}</small></span>{attachment.id && <button type="button" onClick={() => onRemove(attachment.id!)}>Remove</button>}</li>)}</ul>;
}

function ChoiceField({ field, values, selected, multiple = false, describedBy, error, onSelect }: { field: CaptureField; values: string[]; selected: string[]; multiple?: boolean; describedBy?: string; error?: string; onSelect: (values: string[]) => void }) {
  const helpID = `capture-help-${field.id}`;
  const errorID = `capture-error-${field.id}`;
  const minimum = field.constraints?.min_selections;
  const maximum = field.constraints?.max_selections;
  const selectionHelp = multiple && (minimum !== undefined || maximum !== undefined)
    ? minimum !== undefined && maximum !== undefined ? `Select ${minimum} to ${maximum}.` : minimum !== undefined ? `Select at least ${minimum}.` : `Select no more than ${maximum}.`
    : field.description;
  return <fieldset className="capture-field" aria-describedby={[selectionHelp ? helpID : "", describedBy, error ? errorID : ""].filter(Boolean).join(" ") || undefined} aria-invalid={Boolean(error)}><legend>{field.label}{field.required ? " *" : ""}</legend>{selectionHelp && <p id={helpID} className="field-help">{selectionHelp}</p>}<div className="choice-grid">{values.map((option) => {
    const checked = selected.includes(option);
    return <label className={checked ? "choice-option selected" : "choice-option"} key={option}><input type={multiple ? "checkbox" : "radio"} name={field.id} value={option} checked={checked} onChange={() => onSelect(multiple ? checked ? selected.filter((value) => value !== option) : [...selected, option] : [option])}/><span>{option}</span></label>;
  })}</div>{error && <p id={errorID} className="field-error">{error}</p>}</fieldset>;
}

function VendorDocumentControl({ field, value, attachment, uploading, error, onChange, onUpload }: { field: CaptureField; value?: CaptureAnswerValue; attachment?: CaptureAttachment; uploading: boolean; error?: string; onChange: (value: CaptureAnswerValue) => void; onUpload: (file: File) => void }) {
  const document = value?.document ?? { artifact_id: "", document_type: "" };
  const update = (change: Partial<CaptureDocumentAnswer>) => onChange({ document: { ...document, ...change } });
  return <fieldset className="capture-field vendor-document-field"><legend>{field.label}{field.required ? " *" : ""}</legend>{field.description && <p className="field-help">{field.description}</p>}<FileDropzone label={`${field.label} file`} description={fileDescription(field)} accept={normalizedAcceptedFormats(field.accepted_formats)} compact busy={uploading} fileName={attachment?.file_name} fileSize={attachment?.size_bytes} onSelect={onUpload}/><div className="vendor-document-metadata"><label className="capture-field"><span>Document type</span><input value={document.document_type} onChange={(event) => update({ document_type: event.target.value })}/></label><label className="capture-field"><span>Document reference</span><input value={document.reference ?? ""} onChange={(event) => update({ reference: event.target.value })}/></label><label className="capture-field"><span>Issuer</span><input value={document.issued_by ?? ""} onChange={(event) => update({ issued_by: event.target.value })}/></label><label className="capture-field"><span>Issue date</span><input type="date" value={document.issued_on ?? ""} onChange={(event) => update({ issued_on: event.target.value })}/></label><label className="capture-field"><span>Expiry date</span><input type="date" value={document.expires_on ?? ""} onChange={(event) => update({ expires_on: event.target.value })}/></label></div>{error && <p id={`capture-error-${field.id}`} className="field-error">{error}</p>}</fieldset>;
}

function normalizedAcceptedFormats(values?: string[]) {
  const formats = (values ?? []).map((value) => value.split(";", 1)[0]?.trim().toLowerCase()).filter((value): value is string => Boolean(value));
  return formats.length ? formats.join(",") : undefined;
}

function fileDescription(field: CaptureField) {
  const parts = [field.description];
  if (field.constraints?.max_file_bytes) parts.push(`Maximum file size ${formatBytes(field.constraints.max_file_bytes)}.`);
	if (field.constraints?.min_files && field.constraints?.max_files) parts.push(`Select ${field.constraints.min_files} to ${field.constraints.max_files} files.`);
	else if (field.constraints?.min_files) parts.push(`Select at least ${field.constraints.min_files} files.`);
	else if (field.constraints?.max_files && field.constraints.max_files > 1) parts.push(`Up to ${field.constraints.max_files} files.`);
	if (field.constraints?.max_total_file_bytes) parts.push(`Combined limit ${formatBytes(field.constraints.max_total_file_bytes)}.`);
  return parts.filter(Boolean).join(" ") || undefined;
}

function formatBytes(bytes: number) {
  if (bytes < 1024 * 1024) return `${Math.ceil(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(bytes % (1024 * 1024) === 0 ? 0 : 1)} MB`;
}
