import type { FormResponseWorkspacePayload } from "./captureApi";
import type { CaptureRecoveryContext } from "./captureRecovery";

function boundedString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function validTimestamp(value: string): boolean {
  return value.length > 0 && Number.isFinite(Date.parse(value));
}

export function buildCaptureRecoveryContext(
  payload: FormResponseWorkspacePayload,
  origin: string,
): CaptureRecoveryContext {
  const authority = payload.recovery_context;
  const legalEntityID = boundedString(authority?.legal_entity_id);
  const distributionID = boundedString(authority?.distribution_id);
  const schemaVersion = Number.isInteger(authority?.schema_version) && authority.schema_version > 0
    ? authority.schema_version
    : 0;
  const routeExpiresAt = boundedString(authority?.route_expires_at);
  const sessionDistributionID = boundedString(payload.session?.distribution_id);
  const workspaceDistributionID = boundedString(payload.workspace?.workspace?.distribution_id);
  const workspaceID = boundedString(payload.workspace?.workspace?.id);
  const normalizedOrigin = boundedString(origin);
  const deadline = boundedString(payload.request?.deadline);
  const authorized = Boolean(
    normalizedOrigin
      && legalEntityID
      && distributionID
      && schemaVersion > 0
      && workspaceID
      && distributionID === sessionDistributionID
      && distributionID === workspaceDistributionID
      && validTimestamp(deadline)
      && validTimestamp(routeExpiresAt),
  );

  return {
    origin: normalizedOrigin,
    legalEntityID,
    distributionID,
    schemaVersion,
    workspaceID,
    serverVersion: payload.workspace.workspace.version,
    authorized,
    deadline,
    routeExpiresAt,
    cachePolicy: authorized ? "ENCRYPTED_BROWSER_CACHE" : "NO_BROWSER_CACHE",
  };
}
