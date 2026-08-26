import { useRef, useState } from "react";

type Props = {
  label: string;
  description?: string;
  accept?: string;
  capture?: "user" | "environment";
  disabled?: boolean;
  busy?: boolean;
  compact?: boolean;
  actionLabel?: string;
  replaceLabel?: string;
  fileName?: string;
  fileSize?: number;
  previewUrl?: string;
  multiple?: boolean;
  onSelect: (file: File) => void;
  onSelectMany?: (files: File[]) => void;
};

export function FileDropzone({
  label,
  description,
  accept,
  capture,
  disabled = false,
  busy = false,
  compact = false,
  actionLabel = "Choose file",
  replaceLabel = "Replace file",
  fileName,
  fileSize,
  previewUrl,
  multiple = false,
  onSelect,
  onSelectMany,
}: Props) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);
  const inactive = disabled || busy;

  function select(files?: FileList | File[]) {
    if (inactive || !files?.length) return;
    const selected = Array.from(files);
    if (multiple && onSelectMany) onSelectMany(selected);
    else onSelect(selected[0]!);
  }

  return <div
    className={`file-dropzone${compact ? " compact" : ""}${dragging ? " dragging" : ""}${fileName ? " has-file" : ""}`}
    onDragEnter={(event) => {
      event.preventDefault();
      if (!inactive) setDragging(true);
    }}
    onDragOver={(event) => {
      event.preventDefault();
      if (!inactive) setDragging(true);
    }}
    onDragLeave={(event) => {
      event.preventDefault();
      if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setDragging(false);
    }}
    onDrop={(event) => {
      event.preventDefault();
      setDragging(false);
      select(event.dataTransfer.files);
    }}
  >
    <input
      ref={inputRef}
      className="file-dropzone-input"
      type="file"
      aria-label={label}
      accept={accept}
      capture={capture}
      multiple={multiple}
      disabled={inactive}
      tabIndex={-1}
      onChange={(event) => {
        select(event.target.files ?? undefined);
        event.currentTarget.value = "";
      }}
    />
    {previewUrl && <img className="file-dropzone-preview" src={previewUrl} alt="Selected file preview"/>}
    <div className="file-dropzone-copy">
      <strong>{label}</strong>
      {description && <span>{description}</span>}
      {fileName && <small>{fileName}{fileSize ? ` · ${formatBytes(fileSize)}` : ""}</small>}
    </div>
    <button className="file-dropzone-action" type="button" disabled={inactive} onClick={() => inputRef.current?.click()}>
      {busy ? "Uploading…" : fileName ? replaceLabel : actionLabel}
    </button>
  </div>;
}

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
