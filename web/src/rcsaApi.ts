import { requestJSON } from "./http";
import type { RCSACycleDetail, RCSACyclePage, RCSAStatus } from "./rcsaTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type RCSAListParams = {
  status?: RCSAStatus;
  cursor?: string;
  limit?: number;
};

export function listRCSACycles(params: RCSAListParams = {}, signal?: AbortSignal): Promise<RCSACyclePage> {
  const query = new URLSearchParams();
  if (params.status) query.set("status", params.status);
  if (params.cursor?.trim()) query.set("cursor", params.cursor.trim());
  if (params.limit !== undefined) query.set("limit", String(params.limit));
  const suffix = query.size ? `?${query.toString()}` : "";
  return requestJSON<RCSACyclePage>(apiBase, `/api/v1/rcsa/cycles${suffix}`, signal ? { signal } : undefined);
}

export function getRCSACycle(id: string, signal?: AbortSignal): Promise<RCSACycleDetail> {
  return requestJSON<RCSACycleDetail>(
    apiBase,
    `/api/v1/rcsa/cycles/${encodeURIComponent(id)}`,
    signal ? { signal } : undefined,
  );
}
