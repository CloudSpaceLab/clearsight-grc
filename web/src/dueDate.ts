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

export type ActionDeadlinePresentation = {
  dateTime?: string;
  label: string;
  overdue: boolean;
};

const dayMilliseconds = 86_400_000;

function localCalendarDay(value: Date): number {
  return Date.UTC(value.getFullYear(), value.getMonth(), value.getDate()) / dayMilliseconds;
}

export function actionDeadlinePresentation(
  value: string | undefined,
  terminal: boolean,
  now = new Date(Date.now()),
): ActionDeadlinePresentation {
  if (!value) return { label: "No action deadline", overdue: false };

  const due = new Date(value);
  if (!Number.isFinite(due.valueOf())) return { label: "No action deadline", overdue: false };

  const dateTime = due.toISOString();
  const formattedDate = `${due.getDate()} ${new Intl.DateTimeFormat("en-US", {
    month: "short",
  }).format(due)} ${due.getFullYear()}`;

  if (terminal || due.valueOf() >= now.valueOf()) {
    return { dateTime, label: `Due ${formattedDate}`, overdue: false };
  }

  const elapsedDays = Math.max(0, localCalendarDay(now) - localCalendarDay(due));
  const elapsedLabel = elapsedDays === 0
    ? "overdue today"
    : `${elapsedDays} day${elapsedDays === 1 ? "" : "s"} overdue`;

  return {
    dateTime,
    label: `Due ${formattedDate} · ${elapsedLabel}`,
    overdue: true,
  };
}
