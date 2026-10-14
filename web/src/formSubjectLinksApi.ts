import { requestJSON } from "./http";
import type { LifecycleStatus } from "./monitoringTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type FormSubjectLink = {
  id: string;
  subject_type: "MATTER";
  subject_id: string;
  form_template_id: string;
  form_template_version: number;
  form_name: string;
  form_status: LifecycleStatus;
  linked_by: string;
  linked_at: string;
  ended_at?: string;
  version: number;
};

export type FormSubjectLinkPage = {
  items: FormSubjectLink[];
  next_cursor?: string;
};

function matterLinkPath(matterID: string) {
  return `/api/v1/matters/${encodeURIComponent(matterID)}/form-links`;
}

export function loadMatterFormLinks(matterID: string, cursor?: string, signal?: AbortSignal): Promise<FormSubjectLinkPage> {
  const params = new URLSearchParams({ limit: "20" });
  if (cursor) params.set("cursor", cursor);
  return requestJSON<FormSubjectLinkPage>(apiBase, `${matterLinkPath(matterID)}?${params}`, signal ? { signal } : undefined);
}

export function linkExistingForm(matterID: string, formID: string, revision: number): Promise<FormSubjectLink> {
  return requestJSON<FormSubjectLink>(apiBase, matterLinkPath(matterID), {
    method: "POST",
    body: JSON.stringify({ form_template_id: formID, form_template_version: revision }),
  });
}

export function unlinkExistingForm(matterID: string, link: FormSubjectLink, reason: string): Promise<FormSubjectLink> {
  return requestJSON<FormSubjectLink>(apiBase, `${matterLinkPath(matterID)}/${encodeURIComponent(link.id)}/retirement`, {
    method: "POST",
    body: JSON.stringify({ expected_version: link.version, reason }),
  });
}
