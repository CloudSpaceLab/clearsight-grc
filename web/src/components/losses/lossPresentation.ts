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

export const lossEventTypes: readonly LossEventType[] = [
  "INTERNAL_FRAUD",
  "EXTERNAL_FRAUD",
  "EMPLOYMENT_PRACTICES",
  "CLIENT_PRODUCTS_BUSINESS_PRACTICES",
  "DAMAGE_TO_PHYSICAL_ASSETS",
  "BUSINESS_DISRUPTION_SYSTEM_FAILURES",
  "EXECUTION_DELIVERY_PROCESS_MANAGEMENT",
  "OTHER",
];

export const lossEventOptions = lossEventTypes.map((id) => ({ id, label: eventLabels[id] }));

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

export function formatLossMoneyExact(minorUnits: string | bigint, currency: string) {
  const normalized = currency.trim().toUpperCase();
  const digits = lossCurrencyFractionDigits(normalized);
  if (digits === undefined) return `${minorUnits} ${normalized || "currency"}`;

  let minor: bigint;
  try {
    minor = typeof minorUnits === "bigint" ? minorUnits : BigInt(minorUnits.trim());
  } catch {
    return `${minorUnits} ${normalized}`;
  }

  const scale = 10n ** BigInt(digits);
  const negative = minor < 0n;
  const absolute = negative ? -minor : minor;
  const whole = absolute / scale;
  const fraction = absolute % scale;

  const formatter = new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: normalized,
    currencyDisplay: "narrowSymbol",
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  });
  const signedWhole: bigint | number = negative
    ? whole === 0n ? -0 : -whole
    : whole;
  const exactFraction = digits ? fraction.toString().padStart(digits, "0") : "";

  return formatter.formatToParts(signedWhole).map((part) => {
    if (part.type === "fraction") return exactFraction;
    return part.value;
  }).join("");
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
