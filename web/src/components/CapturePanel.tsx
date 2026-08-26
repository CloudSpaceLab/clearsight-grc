import { useEffect, useMemo, useRef, useState } from "react";
import { loadCaptureDraft, saveCaptureDraft, submitInternalCaptureRequest, uploadInternalCaptureArtifact, type CaptureArtifact, type CaptureReceipt } from "../captureApi";
import { apiErrorKind, type ApiErrorKind } from "../http";
import type { CaptureAnswerValue, CaptureAnswers, CaptureField, CapturePresentationMode, CaptureRequest } from "../types";
import { CaptureForm } from "./capture/CaptureForm";
import type { CaptureAttachment } from "./capture/CaptureFieldControl";
import { CaptureReview } from "./capture/CaptureReview";
import { captureContract, effectivePresentationMode, keepVisibleAnswers, normalizeFieldType, visibleCaptureFields } from "./capture/contract";
import { initialSourceAnswers } from "./capture/sourceProvenance";
import { EmptyState } from "./EmptyState";

export type CaptureLoadState = "loading" | "live" | "unavailable" | "forbidden" | "not-found";
type SubmitResult = Pick<CaptureReceipt, "submitted_at"> & Partial<CaptureReceipt>;
type DraftState = "idle" | "loading" | "saving" | "saved" | "failed" | "ended";

type Props = {
  request: CaptureRequest | null;
  state?: CaptureLoadState;
  onReload?: () => void;
  external?: boolean;
  sessionToken?: string;
  onSubmit?: (request: CaptureRequest, answers: CaptureAnswers) => Promise<SubmitResult>;
  onUploadArtifact?: (requestID: string, file: File, fieldID?: string) => Promise<CaptureArtifact>;
};

export function CapturePanel({ request, state = "live", onReload, external = false, sessionToken, onSubmit, onUploadArtifact }: Props) {
  const [answers, setAnswers] = useState<CaptureAnswers>({});
	const [attachments, setAttachments] = useState<Record<string, CaptureAttachment[]>>({});
  const [uploadingField, setUploadingField] = useState<string | null>(null);
  const [mode, setMode] = useState<CapturePresentationMode>("AUTOMATIC");
  const [reviewing, setReviewing] = useState(false);
  const [receipt, setReceipt] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [errorKind, setErrorKind] = useState<ApiErrorKind | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [draftState, setDraftState] = useState<DraftState>("idle");
  const [draftReady, setDraftReady] = useState(false);
  const previewURLs = useRef<Record<string, string>>({});
  const mounted = useRef(true);
  const activeRequestKey = useRef("");
  const answersRef = useRef<CaptureAnswers>({});
  const modeRef = useRef<CapturePresentationMode>("AUTOMATIC");
  const draftVersion = useRef(0);
  const draftReadyRef = useRef(false);
  const draftTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const draftSaveInFlight = useRef(false);
  const draftSavePromise = useRef<Promise<boolean> | null>(null);
  const lastSavedDraft = useRef("");
  const locallyEditedFields = useRef(new Set<string>());
  const locallyChangedMode = useRef(false);
  const requestKey = request ? `${request.id}:${request.version}` : "";
  activeRequestKey.current = requestKey;
  answersRef.current = answers;
  modeRef.current = mode;
  const contract = useMemo(() => request ? captureContract(request) : null, [request]);

  useEffect(() => {
    revokeAllPreviews(previewURLs.current);
    previewURLs.current = {};
    setAnswers(request ? initialSourceAnswers(request) : {});
    setAttachments({});
    setUploadingField(null);
    setMode(request?.presentation?.default_mode ?? "AUTOMATIC");
    setReviewing(false);
    setReceipt(null);
    setError(null);
    setErrorKind(null);
    setSubmitting(false);
    setDraftState("idle");
    setDraftReady(false);
    draftReadyRef.current = false;
    draftVersion.current = 0;
    lastSavedDraft.current = "";
    locallyEditedFields.current.clear();
    locallyChangedMode.current = false;
    if (draftTimer.current) clearTimeout(draftTimer.current);
    draftTimer.current = null;
  }, [request?.id, request?.version]);

  useEffect(() => {
    if (!request || !contract || !external || !sessionToken) return;
    const loadRequestKey = requestKey;
    setDraftState("loading");
    void loadCaptureDraft(sessionToken).then((draft) => {
      if (!currentRequest(mounted.current, activeRequestKey.current, loadRequestKey)) return;
      draftVersion.current = draft.version;
      const restoredMode = locallyChangedMode.current ? modeRef.current : draft.presentation_mode;
      setMode(restoredMode);
      setAnswers((current) => {
        const restored = { ...initialSourceAnswers(request), ...draft.answers };
        for (const fieldID of locallyEditedFields.current) {
          if (current[fieldID]) restored[fieldID] = current[fieldID];
          else delete restored[fieldID];
        }
        const visible = keepVisibleAnswers(contract, restored);
        lastSavedDraft.current = captureDraftSnapshot(visible, restoredMode);
        return visible;
      });
	  setAttachments((current) => {
		const restored = restoredCaptureAttachments(draft.answers);
		for (const fieldID of locallyEditedFields.current) {
		  if (current[fieldID]) restored[fieldID] = current[fieldID];
		  else delete restored[fieldID];
		}
		return restored;
	  });
      setDraftReady(true);
      draftReadyRef.current = true;
      setDraftState(draft.version > 0 ? "saved" : "idle");
    }).catch((cause) => {
      if (!currentRequest(mounted.current, activeRequestKey.current, loadRequestKey)) return;
      setDraftReady(false);
      draftReadyRef.current = false;
      setDraftState(apiErrorKind(cause) === "unauthorized" ? "ended" : "failed");
    });
  }, [requestKey, external, sessionToken, contract]);

  useEffect(() => {
    if (!external || !sessionToken || !draftReady || receipt || submitting) return;
    if (captureDraftSnapshot(answers, mode) === lastSavedDraft.current || draftSaveInFlight.current) return;
    scheduleDraftSave(500);
    return () => {
      if (draftTimer.current) clearTimeout(draftTimer.current);
      draftTimer.current = null;
    };
  }, [answers, mode, external, sessionToken, draftReady, receipt, submitting]);

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      if (draftTimer.current) clearTimeout(draftTimer.current);
      revokeAllPreviews(previewURLs.current);
      previewURLs.current = {};
    };
  }, []);

  if (state === "loading") return <div className="panel-content"><span className="eyebrow">Evidence request</span><h2>Loading request</h2><p aria-live="polite" aria-busy="true">Getting the latest request…</p></div>;
  if (state === "forbidden") return <div className="panel-content"><EmptyState kind="forbidden" label="Request" title="You cannot open this request" description="Your current access does not allow you to view it."/></div>;
  if (state === "not-found") return <div className="panel-content"><EmptyState kind="not-found" label="Request" title="This request is no longer available" description="It may have been replaced, cancelled, or moved outside your access."/></div>;
  if (state === "unavailable") return <div className="panel-content"><EmptyState kind="unavailable" label="Request" title="The request could not be loaded" description="Try again. No response has been recorded." action={onReload ? "Try again" : undefined} onAction={onReload}/></div>;
  if (!request || !contract) return <div className="panel-content"><EmptyState label="Request" title="No request selected" description="Open a request from the evidence list."/></div>;

  const effectiveStatus = isPastDeadline(request) && ["READY", "IN_PROGRESS"].includes(request.status) ? "EXPIRED" : request.status;
  if (effectiveStatus !== "READY" && effectiveStatus !== "IN_PROGRESS") return <TerminalRequest request={request} status={effectiveStatus}/>;

  function updateAnswer(fieldID: string, value: CaptureAnswerValue) {
    if (!contract) return;
    locallyEditedFields.current.add(fieldID);
    setAnswers((current) => keepVisibleAnswers(contract, { ...current, [fieldID]: value }));
  }

  function changeMode(nextMode: CapturePresentationMode) {
    locallyChangedMode.current = true;
    setMode(nextMode);
  }

  function scheduleDraftSave(delay: number) {
    if (draftTimer.current) clearTimeout(draftTimer.current);
    draftTimer.current = setTimeout(() => void performDraftSave(), delay);
  }

  async function performDraftSave(): Promise<boolean> {
    if (!sessionToken || !draftReadyRef.current) return false;
    if (draftSavePromise.current) return draftSavePromise.current;
    const operation = (async () => {
      const saveRequestKey = requestKey;
      const saveAnswers = answersRef.current;
      const saveMode = modeRef.current;
      const snapshot = captureDraftSnapshot(saveAnswers, saveMode);
      if (snapshot === lastSavedDraft.current) return true;
      draftSaveInFlight.current = true;
      setDraftState("saving");
      try {
        const saved = await saveCaptureDraft(sessionToken, { answers: saveAnswers, presentation_mode: saveMode, expected_version: draftVersion.current });
        if (!currentRequest(mounted.current, activeRequestKey.current, saveRequestKey)) return false;
        draftVersion.current = saved.version;
        lastSavedDraft.current = snapshot;
        setDraftState("saved");
        if (captureDraftSnapshot(answersRef.current, modeRef.current) !== snapshot) scheduleDraftSave(500);
        return true;
      } catch (cause) {
        if (!currentRequest(mounted.current, activeRequestKey.current, saveRequestKey)) return false;
        const kind = apiErrorKind(cause);
        draftReadyRef.current = false;
        setDraftReady(false);
        setDraftState(kind === "unauthorized" || kind === "forbidden" ? "ended" : "failed");
        return false;
      } finally {
        draftSaveInFlight.current = false;
      }
    })();
    draftSavePromise.current = operation;
    try {
      return await operation;
    } finally {
      if (draftSavePromise.current === operation) draftSavePromise.current = null;
    }
  }

  async function saveBeforeSectionNavigation() {
    if (!external || !sessionToken) return true;
    if (draftTimer.current) clearTimeout(draftTimer.current);
    draftTimer.current = null;
    const activeSave = draftSavePromise.current;
    if (activeSave && !await activeSave) return false;
    if (!draftReadyRef.current) return false;
    return performDraftSave();
  }

  async function retryDraftSave() {
    if (!sessionToken || draftSaveInFlight.current) return;
    setDraftState("loading");
    try {
      const current = await loadCaptureDraft(sessionToken);
      draftVersion.current = current.version;
      setDraftReady(true);
      draftReadyRef.current = true;
      lastSavedDraft.current = "";
      await performDraftSave();
    } catch (cause) {
      draftReadyRef.current = false;
      setDraftReady(false);
      setDraftState(apiErrorKind(cause) === "unauthorized" ? "ended" : "failed");
    }
  }

	async function upload(field: CaptureField, files: File[], preferredPreviewURL?: string) {
    if (!request || uploadingField) return;
	locallyEditedFields.current.add(field.id);
	const currentAttachments = attachments[field.id] ?? [];
	const validationError = validateSelectedFiles(field, currentAttachments, files);
	if (validationError) {
	  setErrorKind("validation");
	  setError(validationError);
	  return;
	}
    const uploadRequestKey = requestKey;
	const single = normalizeFieldType(field.type) !== "file" || (field.constraints?.max_files ?? 1) === 1;
	const previousAttachments = currentAttachments;
    const previousObjectPreview = previewURLs.current[field.id];
    setUploadingField(field.id);
    setError(null);
    setErrorKind(null);
	let completed = single ? [] as CaptureAttachment[] : [...previousAttachments];
	let failedFile = "";
	for (const file of files) {
	  let previewURL = files.length === 1 ? preferredPreviewURL : undefined;
	  let createdObjectPreview = false;
	  try {
		const artifact = await (onUploadArtifact ?? uploadInternalCaptureArtifact)(request.id, file, field.id);
		if (!currentRequest(mounted.current, activeRequestKey.current, uploadRequestKey)) return;
		if (single && !previewURL && file.type.startsWith("image/") && typeof URL.createObjectURL === "function") {
		  previewURL = URL.createObjectURL(file);
		  createdObjectPreview = true;
		}
		const attachment: CaptureAttachment = { id: artifact.id, file_name: artifact.file_name, media_type: artifact.media_type, size_bytes: artifact.size_bytes, preview_url: previewURL };
		completed = single ? [attachment] : [...completed, attachment];
		if (single) {
		  if (previousObjectPreview && previousObjectPreview !== previewURL) URL.revokeObjectURL(previousObjectPreview);
		  if (previewURL) previewURLs.current[field.id] = previewURL;
		  else delete previewURLs.current[field.id];
		}
		setAttachments((current) => ({ ...current, [field.id]: completed }));
		setAnswers((current) => {
		  const artifactIDs = completed.flatMap((item) => item.id ? [item.id] : []);
		  const nextValue: CaptureAnswerValue = normalizeFieldType(field.type) === "vendor_document"
			? { document: { ...current[field.id]?.document, artifact_id: artifact.id, document_type: current[field.id]?.document?.document_type ?? "" } }
			: { artifact_ids: artifactIDs };
		  return contract ? keepVisibleAnswers(contract, { ...current, [field.id]: nextValue }) : current;
		});
	  } catch (cause) {
		if (createdObjectPreview && previewURL) URL.revokeObjectURL(previewURL);
		if (!currentRequest(mounted.current, activeRequestKey.current, uploadRequestKey)) return;
		failedFile = file.name;
		setErrorKind(apiErrorKind(cause));
		break;
	  }
	}
	if (failedFile) {
	  setError(completed.length > previousAttachments.length
		? `${failedFile} could not be uploaded. The files uploaded before it remain selected.`
		: single && previousAttachments.length ? `${failedFile} could not be uploaded. The previous file remains selected.` : `${failedFile} could not be uploaded. Your other answers remain on this screen.`);
	}
	if (currentRequest(mounted.current, activeRequestKey.current, uploadRequestKey)) setUploadingField(null);
  }

	function removeAttachment(field: CaptureField, attachmentID: string) {
	  locallyEditedFields.current.add(field.id);
	  setAttachments((current) => {
		const remaining = (current[field.id] ?? []).filter((attachment) => attachment.id !== attachmentID);
		const next = { ...current };
		if (remaining.length) next[field.id] = remaining;
		else delete next[field.id];
		return next;
	  });
	  setAnswers((current) => {
		const remaining = current[field.id]?.artifact_ids?.filter((id) => id !== attachmentID) ?? [];
		const next = { ...current };
		if (remaining.length) next[field.id] = { artifact_ids: remaining };
		else delete next[field.id];
		return contract ? keepVisibleAnswers(contract, next) : next;
	  });
	}

  async function submit() {
    if (!request || !contract || submitting) return;
    const submitRequestKey = requestKey;
    const submissionAnswers = keepVisibleAnswers(contract, answers);
    setError(null);
    setErrorKind(null);
    setSubmitting(true);
    if (draftTimer.current) clearTimeout(draftTimer.current);
    try {
      const result = onSubmit ? await onSubmit(request, submissionAnswers) : await submitInternalCaptureRequest(request.id, request.version, submissionAnswers);
      if (!currentRequest(mounted.current, activeRequestKey.current, submitRequestKey)) return;
      setReceipt(new Date(result.submitted_at).toLocaleString());
      setReviewing(false);
    } catch (cause) {
      if (!currentRequest(mounted.current, activeRequestKey.current, submitRequestKey)) return;
      const kind = apiErrorKind(cause);
      setErrorKind(kind);
      setError(errorMessage(kind, cause));
    } finally {
      if (currentRequest(mounted.current, activeRequestKey.current, submitRequestKey)) setSubmitting(false);
    }
  }

  if (receipt) return <div className="panel-content response-receipt"><span className="eyebrow">Receipt</span><div className="receipt-mark" aria-hidden="true">✓</div><h2>{external ? "Submitted" : "Response submitted"}</h2><p>{receipt}</p><p>{external ? "Your response was recorded." : "The response was recorded for evidence review."}</p></div>;

  if (reviewing) return <CaptureReview request={request} fields={visibleCaptureFields(contract, answers)} answers={answers} attachments={attachments} external={external} submitting={submitting} error={error} errorKind={errorKind} onEdit={() => setReviewing(false)} onReload={onReload} onSubmit={() => void submit()}/>;

  return <div className="panel-content">
    <span className="eyebrow">Evidence request · about {request.estimated_minutes} min</span><h2>{request.title}</h2><p>{request.purpose}</p>
    {external && <div className="external-request-context" aria-label="Request details">
      <strong>Due {formatCaptureDeadline(request.deadline)}</strong>
      <span>Your answers and files are shared with the organization that sent this request.</span>
      <span>For changes to the request or your access, contact the person who sent this link.</span>
    </div>}
    <div className="why-you"><strong>Why this was sent to you</strong><span>{request.why_you}</span></div>
    {Object.keys(request.known_facts).length > 0 && <><h3>Already filled in</h3><dl className="known-facts">{Object.entries(request.known_facts).map(([key, value]) => <div key={key}><dt>{humanize(key)}</dt><dd>{value}</dd></div>)}</dl></>}
    {external && sessionToken && <DraftStatus state={draftState} onRetry={() => void retryDraftSave()}/>}
	<CaptureForm contract={contract} answers={answers} attachments={attachments} mode={effectivePresentationMode(contract, answers, mode)} external={external} uploadingField={uploadingField} onAnswer={updateAnswer} onUpload={(field, files, previewURL) => void upload(field, files, previewURL)} onRemoveAttachment={removeAttachment} onModeChange={changeMode} onBeforeSectionNavigation={external && sessionToken ? saveBeforeSectionNavigation : undefined} onReview={() => { setError(null); setErrorKind(null); setReviewing(true); }}/>
    {error && <p className="error-text" role="alert">{error}</p>}
  </div>;
}

function DraftStatus({ state, onRetry }: { state: DraftState; onRetry: () => void }) {
  if (state === "idle" || state === "loading") return null;
  if (state === "saving") return <div className="capture-draft-status" aria-live="polite"><span>Saving</span></div>;
  if (state === "saved") return <div className="capture-draft-status" aria-live="polite"><span>Saved</span></div>;
  if (state === "ended") return <div className="capture-draft-status capture-draft-status-error" aria-live="polite"><span><strong>Access ended</strong> Ask the sender for a new link.</span><button type="button" onClick={onRetry}>Check access</button></div>;
  return <div className="capture-draft-status capture-draft-status-error" aria-live="polite"><span><strong>Could not save</strong> Your entries remain on this screen.</span><button type="button" onClick={onRetry}>Try again</button></div>;
}

function TerminalRequest({ request, status }: { request: CaptureRequest; status: string }) {
  const copy: Record<string, [string, string]> = { SUBMITTED: ["Response already submitted", "This request already has a response."], EXPIRED: ["This request has expired", "The deadline has passed. Ask the sender to extend or replace the request."], CANCELLED: ["This request was cancelled", "No further response can be submitted."], DRAFT: ["This request is not ready", "The sender has not released it for response yet."] };
  const [title, description] = copy[status] ?? ["This request is read-only", `The request is currently ${humanize(status)} and cannot accept a response.`];
  return <div className="panel-content"><EmptyState kind="not-found" label="Request" title={title} description={description}/><p className="request-terminal-context">{request.title}</p></div>;
}

function currentRequest(isMounted: boolean, activeKey: string, operationKey: string) { return isMounted && activeKey === operationKey; }
function revokeAllPreviews(values: Record<string, string>) { if (typeof URL.revokeObjectURL !== "function") return; for (const value of Object.values(values)) URL.revokeObjectURL(value); }
function isPastDeadline(request: CaptureRequest) { const deadline = Date.parse(request.deadline); return Number.isFinite(deadline) && deadline <= Date.now(); }
function formatCaptureDeadline(value: string) {
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) return "not recorded";
  return new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit", timeZoneName: "short" }).format(new Date(parsed));
}
function errorMessage(kind: ApiErrorKind, cause: unknown) { if (kind === "conflict") return "This request changed while you were working. Reload it before submitting. Your current entries remain on this screen."; if (kind === "forbidden" || kind === "unauthorized") return "Your access to this request has ended. Ask the sender to confirm your access or send a new link."; if (kind === "not_found") return "This request is no longer available."; if (kind === "unavailable") return "The response could not be submitted right now. Your entries remain on this screen."; return cause instanceof Error ? cause.message : "The response could not be submitted."; }
function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }
function captureDraftSnapshot(answers: CaptureAnswers, mode: CapturePresentationMode) { return JSON.stringify({ answers, mode }); }

function validateSelectedFiles(field: CaptureField, current: CaptureAttachment[], selected: File[]) {
	const single = normalizeFieldType(field.type) !== "file" || (field.constraints?.max_files ?? 1) === 1;
	const maximumFiles = normalizeFieldType(field.type) === "file" ? field.constraints?.max_files ?? 1 : 1;
	const retained = single ? [] : current;
	if (selected.length === 0) return "Choose at least one file.";
	if (retained.length + selected.length > maximumFiles) return `${field.label} accepts no more than ${maximumFiles} file${maximumFiles === 1 ? "" : "s"}.`;
	for (const file of selected) {
	  if (field.constraints?.max_file_bytes && file.size > field.constraints.max_file_bytes) return `${file.name} exceeds the ${formatFileBytes(field.constraints.max_file_bytes)} per-file limit.`;
	  if (!clientFileFormatAllowed(file, field.accepted_formats)) return `${file.name} does not use a permitted file type.`;
	  if (retained.some((attachment) => attachment.file_name === file.name && attachment.size_bytes === file.size)) return `${file.name} is already selected.`;
	}
	const total = [...retained.map((attachment) => attachment.size_bytes), ...selected.map((file) => file.size)].reduce((sum, size) => sum + size, 0);
	if (field.constraints?.max_total_file_bytes && total > field.constraints.max_total_file_bytes) return `${field.label} exceeds the ${formatFileBytes(field.constraints.max_total_file_bytes)} combined limit.`;
	return null;
}

function clientFileFormatAllowed(file: File, accepted?: string[]) {
	if (!accepted?.length) return true;
	const name = file.name.toLowerCase();
	const mediaType = file.type.toLowerCase().split(";", 1)[0] ?? "";
	return accepted.some((value) => { const normalized = value.toLowerCase().split(";", 1)[0]?.trim() ?? ""; return normalized.startsWith(".") ? name.endsWith(normalized) : normalized === mediaType || (normalized.endsWith("/*") && mediaType.startsWith(normalized.slice(0, -1))) || fileExtensionMatchesMediaType(name, normalized); });
}

function fileExtensionMatchesMediaType(name: string, mediaType: string) {
	const extensions: Record<string, string[]> = { "application/pdf": [".pdf"], "image/png": [".png"], "image/jpeg": [".jpg", ".jpeg"], "text/plain": [".txt"], "text/csv": [".csv"], "application/vnd.openxmlformats-officedocument.wordprocessingml.document": [".docx"], "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": [".xlsx"] };
	return (extensions[mediaType] ?? []).some((extension) => name.endsWith(extension));
}

function formatFileBytes(bytes: number) { return bytes < 1024 * 1024 ? `${Math.ceil(bytes / 1024)} KB` : `${(bytes / (1024 * 1024)).toFixed(bytes % (1024 * 1024) === 0 ? 0 : 1)} MB`; }

function restoredCaptureAttachments(answers: CaptureAnswers) {
	const restored: Record<string, CaptureAttachment[]> = {};
	for (const [fieldID, answer] of Object.entries(answers)) {
	  const artifactIDs = answer.artifact_ids ?? (answer.document?.artifact_id ? [answer.document.artifact_id] : []);
	  if (artifactIDs.length) restored[fieldID] = artifactIDs.map((id) => ({ id, file_name: "Previously uploaded file", media_type: "", size_bytes: 0 }));
	}
	return restored;
}
