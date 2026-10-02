import { requestJSON } from "./http";
import type { RiskControlDefinition, RiskControlImplementationStatus } from "./riskTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type ControlCatalogCandidate = {
  catalog_link_id: string;
  definition: RiskControlDefinition;
  program_id: string;
  program_name: string;
  implementation_id: string;
  implementation_name: string;
  implementation_status: RiskControlImplementationStatus;
};

export type ControlCatalogCandidatePage = {
  items: ControlCatalogCandidate[];
  complete: boolean;
};

export type ControlCatalogPromotion = {
  definition: RiskControlDefinition;
  implementation_link: {
    id: string;
    definition_id: string;
    program_id: string;
    implementation_id: string;
  };
};

export function listControlCatalogCandidates(programID?: string, signal?: AbortSignal): Promise<ControlCatalogCandidatePage> {
  const query = new URLSearchParams({ limit: "100" });
  if (programID?.trim()) query.set("program_id", programID.trim());
  return requestJSON<ControlCatalogCandidatePage>(
    apiBase,
    `/api/v1/control-catalog/implementation-links?${query.toString()}`,
    signal ? { signal } : undefined,
  );
}

export function promoteProgramControl(programID: string, implementationID: string, expectedVersion: number): Promise<ControlCatalogPromotion> {
  return requestJSON<ControlCatalogPromotion>(
    apiBase,
    `/api/v1/programs/${encodeURIComponent(programID)}/control-implementations/${encodeURIComponent(implementationID)}/catalog`,
    {
      method: "POST",
      body: JSON.stringify({ expected_version: expectedVersion }),
    },
  );
}
