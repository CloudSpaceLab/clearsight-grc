import { requestJSON } from "./http";
import type { IndicatorDetailModel, IndicatorKind, IndicatorRiskReference, IndicatorState } from "./indicatorTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type IndicatorPortfolioItem = IndicatorDetailModel & {
  risks: IndicatorRiskReference[];
};

export type IndicatorPortfolioPage = {
  generated_at: string;
  items: IndicatorPortfolioItem[];
  next_cursor?: string;
};

export type IndicatorPortfolioFilter = {
  kind?: IndicatorKind;
  state?: IndicatorState;
  search?: string;
  cursor?: string;
  limit?: number;
};

export function loadIndicatorPortfolio(filter: IndicatorPortfolioFilter = {}, signal?: AbortSignal): Promise<IndicatorPortfolioPage> {
  const query = new URLSearchParams();
  if (filter.kind) query.set("kind", filter.kind);
  if (filter.state) query.set("state", filter.state);
  if (filter.search?.trim()) query.set("search", filter.search.trim());
  if (filter.cursor) query.set("cursor", filter.cursor);
  if (filter.limit !== undefined) query.set("limit", String(filter.limit));
  const suffix = query.size ? `?${query.toString()}` : "";
  return requestJSON<IndicatorPortfolioPage>(
    apiBase,
    `/api/v1/metrics/indicators${suffix}`,
    signal ? { signal } : undefined,
  );
}
