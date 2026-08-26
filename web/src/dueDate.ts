export function selectedDateEndOfLocalDay(value: string): string | undefined {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value.trim());
  if (!match) return undefined;

  const year = Number(match[1]);
  const monthIndex = Number(match[2]) - 1;
  const day = Number(match[3]);
  const localDeadline = new Date(year, monthIndex, day, 23, 59, 59, 999);
  if (
    localDeadline.getFullYear() !== year
    || localDeadline.getMonth() !== monthIndex
    || localDeadline.getDate() !== day
  ) return undefined;

  return localDeadline.toISOString();
}

export function storedDeadlineLocalDate(value?: string): string {
  if (!value) return "";
  const stored = new Date(value);
  if (!Number.isFinite(stored.valueOf())) return "";
  return [
    String(stored.getFullYear()).padStart(4, "0"),
    String(stored.getMonth() + 1).padStart(2, "0"),
    String(stored.getDate()).padStart(2, "0"),
  ].join("-");
}
