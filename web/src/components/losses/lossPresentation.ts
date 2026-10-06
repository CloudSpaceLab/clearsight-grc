import type { LossEventType, LossRecoveryStatus, LossStatus } from "../../lossTypes";

const eventLabels: Record<LossEventType, string> = {
  INTERNAL_FRAUD: "Internal fraud",
  EXTERNAL_FRAUD: "External fraud",
  EMPLOYMENT_PRACTICES: "Employment practices",
  CLIENT_PRODUCTS_BUSINESS_PRACTICES: "Clients, products & business practices",
  DAMAGE_TO_PHYSICAL_ASSETS: "Physical asset damage",
  BUSINESS_DISRUPTION_SYSTEM_FAILURES: "Business disruption & system failures",
  EXECUTION_DELIVERY_PROCESS_MANAGEMENT: "Execution, delivery & process management",
  OTHER: "Other",
};

export function lossEventLabel(value: LossEventType) {
  return eventLabels[value];
}

export function lossStatusLabel(value: LossStatus) {
  return value === "ACTIVE" ? "Active" : "Voided";
}

export function lossStatusTone(value: LossStatus) {
  return value === "ACTIVE" ? "info" as const : "neutral" as const;
}

export function recoveryStatusLabel(value: LossRecoveryStatus) {
  if (value === "FULL") return "Fully recovered";
  if (value === "PARTIAL") return "Partly recovered";
  return "No recovery";
}

export function recoveryStatusTone(value: LossRecoveryStatus) {
  if (value === "FULL") return "success" as const;
  if (value === "PARTIAL") return "warning" as const;
  return "neutral" as const;
}

export function formatLossMoney(minor: number, currency: string) {
  const formatter = new Intl.NumberFormat(undefined, {
    style: "currency",
    currency,
    currencyDisplay: "narrowSymbol",
  });
  const digits = formatter.resolvedOptions().maximumFractionDigits ?? 2;
  return formatter.format(minor / (10 ** digits));
}

export function formatLossDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return "Unknown";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(date);
}

export function lossCurrencyFractionDigits(currency: string): number | undefined {
  const normalized = currency.trim().toUpperCase();
  if (!/^[A-Z]{3}$/.test(normalized)) return undefined;
  try {
    return new Intl.NumberFormat("en", { style: "currency", currency: normalized }).resolvedOptions().maximumFractionDigits;
  } catch {
    return undefined;
  }
}

export function lossMajorToMinor(value: string, currency: string): number | undefined {
  const digits = lossCurrencyFractionDigits(currency);
  if (digits === undefined) return undefined;
  const match = /^(?:0|[1-9]\d*)(?:\.(\d+))?$/.exec(value.trim());
  if (!match) return undefined;
  const fraction = match[1] ?? "";
  if (fraction.length > digits) return undefined;

  const whole = value.trim().split(".")[0]!;
  const scale = 10n ** BigInt(digits);
  const paddedFraction = digits ? (fraction + "0".repeat(digits)).slice(0, digits) : "";
  const minor = BigInt(whole) * scale + (paddedFraction ? BigInt(paddedFraction) : 0n);
  if (minor <= 0n || minor > BigInt(Number.MAX_SAFE_INTEGER)) return undefined;
  return Number(minor);
}
