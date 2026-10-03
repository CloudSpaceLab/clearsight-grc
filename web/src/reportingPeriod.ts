export type ReportingPeriod = {
  start_date: string;
  end_date: string;
  mode: "CURRENT_WINDOW";
  max_days: number;
  historical_end_supported: false;
};

export type ReportingPeriodQuery = {
  start_date: string;
  end_date: string;
};

export function reportingPeriodPath(path: string, period?: ReportingPeriodQuery, organizationScopeID?: string) {
  const query = new URLSearchParams();
  if (period) {
    query.set("start_date", period.start_date);
    query.set("end_date", period.end_date);
  }
  if (organizationScopeID?.trim()) query.set("organization_scope_id", organizationScopeID.trim());
  const encoded = query.toString();
  return encoded ? `${path}?${encoded}` : path;
}

export function reportingPeriodLabel(period: ReportingPeriod) {
  const days = utcDayDifference(period.start_date, period.end_date);
  const calendarDays = days + 1;
  if (calendarDays === 30 || calendarDays === 90 || calendarDays === 180) return `Last ${calendarDays} days`;
  return `${shortDate(period.start_date)} – ${shortDate(period.end_date)}`;
}

export function startDateForDays(endDate: string, days: number) {
  const value = parseDate(endDate);
  value.setUTCDate(value.getUTCDate() - Math.max(0, days - 1));
  return value.toISOString().slice(0, 10);
}

export function minimumReportingStart(period: ReportingPeriod) {
  return startDateForDays(period.end_date, period.max_days);
}

export function utcDayDifference(startDate: string, endDate: string) {
  return Math.round((parseDate(endDate).getTime() - parseDate(startDate).getTime()) / 86_400_000);
}

function parseDate(value: string) {
  return new Date(`${value}T00:00:00Z`);
}

function shortDate(value: string) {
  return parseDate(value).toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" });
}
