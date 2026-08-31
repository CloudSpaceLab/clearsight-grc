import type { DistributionAccessPolicy, DistributionDueState, DistributionStatus } from "../../../formsDistributionApi";
import type { StatusTone } from "../../ui";

export const distributionStatusLabel: Record<DistributionStatus, string> = {
  DRAFT: "Draft",
  READY: "Ready to send",
  OPEN: "Responses open",
  LOCKED: "Responses locked",
  COMPLETED: "Completed",
  EXPIRED: "Expired",
  REVOKED: "Access revoked",
  SUPERSEDED: "Replaced",
};

export const distributionStatusTone: Record<DistributionStatus, StatusTone> = {
  DRAFT: "neutral",
  READY: "info",
  OPEN: "info",
  LOCKED: "warning",
  COMPLETED: "success",
  EXPIRED: "warning",
  REVOKED: "error",
  SUPERSEDED: "unknown",
};

export const distributionDueStateLabel: Record<DistributionDueState, string> = {
  OPEN: "Due later",
  OVERDUE: "Overdue",
  CLOSED: "Closed",
};

export const accessPolicyLabel: Record<DistributionAccessPolicy, string> = {
  DIRECT_MAGIC_LINK: "Direct secure link",
  SHARED_LINK_EMAIL_OTP: "Shared link with email code",
  DIRECT_LINK_EMAIL_OTP: "Direct link with email code",
};

export function formatDistributionDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "Date unavailable" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date);
}
