import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";
export type FileKind = "PDF" | "IMAGE" | "WORD" | "SPREADSHEET" | "OTHER";
export type DocumentOccurrence = {
  id: string; artifact_id: string; request_id: string; submission_id: string; field_id: string;
  response_revision_id?: string; distribution_id?: string; relationship_id?: string;
  assessment_id?: string; work_request_id?: string; form_template_id?: string;
  form_template_version: number; form_title: string; field_label: string;
  file_name: string; media_type: string; file_kind: FileKind; size_bytes: number;
  sha256: string; artifact_status: string; uploaded_at: string; uploaded_by?: string;
  demo_preview_available?: boolean;
  demo_unscanned_allowed?: boolean;
  submitted_at: string; submitted_by?: string; submission_channel?: string; expires_on?: string; current: boolean;
  review?: { id: string; status: string; reviewed_by: string; reviewed_at?: string; source: "VENDOR_ASSESSMENT" };
};
export type DocumentQuery = {
  file_kind?: FileKind; query?: string; form_template_id?: string; relationship_id?: string;
  response_revision_id?: string; current_only?: boolean; cursor?: string; limit?: number;
};
export type DocumentPage = { items: DocumentOccurrence[]; next_cursor?: string };

export function loadDocuments(query: DocumentQuery, signal?: AbortSignal): Promise<DocumentPage> {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries({ ...query, limit: query.limit ?? 25 })) {
    if (value !== undefined && value !== "") params.set(key, String(value));
  }
  return requestJSON(apiBase, `/api/v1/forms/documents?${params}`, { signal });
}

type ContentOccurrence = Pick<DocumentOccurrence, "submission_id" | "field_id" | "artifact_id" | "response_revision_id">;
export function documentContentURL(file: ContentOccurrence, download = false) {
  const params = new URLSearchParams();
  if (file.response_revision_id) params.set("response_revision_id", file.response_revision_id);
  if (download) params.set("download", "true");
  const path = [file.submission_id, file.field_id, file.artifact_id].map(encodeURIComponent).join("/");
  return `${apiBase}/api/v1/forms/documents/${path}/content${params.size ? `?${params}` : ""}`;
}

export type DocumentAvailability = Pick<DocumentOccurrence, "artifact_status" | "demo_preview_available" | "demo_unscanned_allowed">;
export function demoUnscannedAllowed(file: DocumentAvailability): boolean {
  return file.artifact_status === "STORED_UNSCANNED" && file.demo_unscanned_allowed === true;
}
// Preview-only sample access never grants eligibility for evidence review.
export function documentReviewAllowed(file: DocumentAvailability): boolean {
  return file.artifact_status === "AVAILABLE" || demoUnscannedAllowed(file);
}
export function documentEligibility(file: DocumentAvailability): "available" | "demo" | undefined {
  if (file.artifact_status === "AVAILABLE") return "available";
  if (demoUnscannedAllowed(file) || (file.artifact_status === "STORED_UNSCANNED" && file.demo_preview_available === true)) return "demo";
  return undefined;
}

export function previewKind(file: DocumentAvailability & Pick<DocumentOccurrence, "media_type">): "pdf" | "image" | undefined {
  if (!documentEligibility(file)) return undefined;
  if (file.media_type === "application/pdf") return "pdf";
  if (["image/png", "image/jpeg", "image/gif", "image/webp"].includes(file.media_type)) return "image";
  return undefined;
}
