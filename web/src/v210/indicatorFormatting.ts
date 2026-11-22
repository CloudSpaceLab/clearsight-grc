import type { MonitoringNativeMeasurement, SourceOperator } from "../monitoringTypes";

export function measurementValue(value?: MonitoringNativeMeasurement) {
  if (!value) return "Native value unavailable";
  if (!value.value) return "No observation";
  const numeric = exactDecimal(value.value, value.precision);
  if (value.unit === "PERCENT") return `${numeric}%`;
  if (value.unit === "MONEY") return `${value.currency || "Currency"} ${numeric}`;
  if (value.unit === "DURATION") return `${numeric} ${duration(value.duration_unit)}`;
  return numeric;
}
// Preserve source digits and avoid floating-point conversion or unbounded expansion.
function exactDecimal(value: string, precision?: number) {
  const match = /^([+-]?)(\d+)(?:\.(\d+))?(?:[eE]([+-]?\d+))?$/.exec(value);
  if (!match) return value;
  const exponent = Number(match[4] ?? "0");
  if (!Number.isSafeInteger(exponent) || Math.abs(exponent) > 100 || (precision !== undefined && (!Number.isInteger(precision) || precision < 0 || precision > 20))) return value;
  const digits = match[2] + (match[3] ?? "");
  const point = match[2]!.length + exponent;
  const expanded = point <= 0 ? `0.${"0".repeat(-point)}${digits}` : point >= digits.length ? `${digits}${"0".repeat(point - digits.length)}` : `${digits.slice(0, point)}.${digits.slice(point)}`;
  const [integer, sourceFraction = ""] = expanded.split(".");
  const fraction = sourceFraction.padEnd(Math.max(sourceFraction.length, precision ?? 0), "0");
  return `${match[1]}${integer!.replace(/\B(?=(\d{3})+(?!\d))/g, ",")}${fraction ? `.${fraction}` : ""}`;
}
function duration(value?: string) { return ({ SECONDS: "seconds", MINUTES: "minutes", HOURS: "hours", DAYS: "days" } as Record<string, string>)[value ?? ""] ?? "duration"; }
function operator(value: SourceOperator) { return ({ EQUALS: "=", NOT_EQUALS: "≠", GREATER_THAN: ">", GREATER_OR_EQUAL: "≥", LESS_THAN: "<", LESS_OR_EQUAL: "≤" } as Partial<Record<SourceOperator, string>>)[value]; }
export function limitText(value?: MonitoringNativeMeasurement) { if (!value?.limits?.length) return "Not recorded"; const limits = value.limits.map((limit) => { const symbol = operator(limit.operator); return symbol && limit.expected ? `${symbol} ${measurementValue({ ...value, value: limit.expected })}` : "Limit unavailable"; }); return limits.join(" · "); }

export function validConcern(score: unknown, denominator: unknown): boolean {
  return typeof score === "number" && Number.isFinite(score) && typeof denominator === "number" && Number.isFinite(denominator) && denominator > 0 && score >= 0 && score <= denominator;
}
