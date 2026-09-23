import { loadContext } from "./api";
import { continuityCommand } from "./continuityCommands";
import { requestJSON } from "./http";
import type { MatterActivityPage, MatterAggregate } from "./types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export async function loadMatterActivity(matterID: string, beforeVersion?: number): Promise<MatterActivityPage> {
  const context = await loadContext();
  const query = new URLSearchParams({ tenant_id: context.tenant.id, limit: "20" });
  if (beforeVersion) query.set("before_version", String(beforeVersion));
  return requestJSON<MatterActivityPage>(apiBase, `/api/v1/matters/${encodeURIComponent(matterID)}/activity?${query}`);
}

export function addMatterComment(matterID: string, expectedVersion: number, body: string, mentionedPrincipalIDs: string[] = []): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/comments`, { expected_version: expectedVersion, body, mentioned_principal_ids: mentionedPrincipalIDs });
}

export function requestMatterActionUpdate(matterID: string, actionID: string, expectedVersion: number, message: string, dueAt?: string): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/actions/${encodeURIComponent(actionID)}/update-requests`, { expected_version: expectedVersion, message, due_at: dueAt });
}
