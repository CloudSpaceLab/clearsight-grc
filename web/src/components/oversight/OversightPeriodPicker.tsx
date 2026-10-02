import { useEffect, useState } from "react";
import type { ReportingPeriod, ReportingPeriodQuery } from "../../reportingPeriod";
import { minimumReportingStart, reportingPeriodLabel, startDateForDays } from "../../reportingPeriod";
import { Button, Notice, PopoverDialog, TextField } from "../ui";

type Props = {
  period: ReportingPeriod;
  freshness: "CURRENT" | "STALE";
  generatedAt: string;
  isChanging?: boolean;
  error?: string;
  onApply: (period: ReportingPeriodQuery) => void;
};

const presets = [30, 90, 180] as const;

export function OversightPeriodPicker({ period, freshness, generatedAt, isChanging = false, error, onApply }: Props) {
  const [open, setOpen] = useState(false);
  const [startDate, setStartDate] = useState(period.start_date);

  useEffect(() => { setStartDate(period.start_date); }, [period.start_date]);

  const minimum = minimumReportingStart(period);
  const invalid = !startDate || startDate < minimum || startDate > period.end_date;

  function apply(query: ReportingPeriodQuery) {
    setOpen(false);
    onApply(query);
  }

  return <div className="oversight-period-control">
    <PopoverDialog
      label="Choose reporting period"
      isOpen={open}
      onOpenChange={setOpen}
      placement="bottom end"
      triggerLabel={`Reporting period, ${reportingPeriodLabel(period)}`}
      triggerClassName="oversight-period-trigger"
      triggerDisabled={isChanging}
      triggerChildren={<>
        <span className="oversight-period-trigger__label">Period</span>
        <strong>{reportingPeriodLabel(period)}</strong>
        <span aria-hidden="true">⌄</span>
      </>}
    >
      <div className="oversight-period-picker">
        <div>
          <strong>Analysis period</strong>
          <p>Current risk posture stays current. Completion, resolution and operating-history measures use this period.</p>
        </div>
        <div className="oversight-period-presets" aria-label="Reporting period presets">
          {presets.map((days) => <Button
            key={days}
            size="compact"
            variant="secondary"
            isDisabled={isChanging}
            onPress={() => apply({ start_date: startDateForDays(period.end_date, days), end_date: period.end_date })}
          >Last {days} days</Button>)}
        </div>
        <div className="oversight-period-fields">
          <TextField
            label="From"
            type="date"
            value={startDate}
            min={minimum}
            max={period.end_date}
            onChange={setStartDate}
            isDisabled={isChanging}
            isInvalid={invalid}
            errorMessage={invalid ? `Choose a date from ${minimum} through ${period.end_date}.` : undefined}
          />
          <TextField
            label="To"
            type="date"
            value={period.end_date}
            onChange={() => undefined}
            min={period.end_date}
            max={period.end_date}
            isReadOnly
            description="Current reporting date. Historical end dates are not reconstructed."
          />
        </div>
        <div className="oversight-period-actions">
          <Button variant="secondary" onPress={() => setOpen(false)}>Cancel</Button>
          <Button isDisabled={invalid || isChanging} onPress={() => apply({ start_date: startDate, end_date: period.end_date })}>
            {isChanging ? "Applying…" : "Apply"}
          </Button>
        </div>
      </div>
    </PopoverDialog>
    <small>{freshness === "CURRENT" ? "Current" : "Needs refresh"} · Updated {formatDateTime(generatedAt)}</small>
    {error && <Notice tone="warning">{error}</Notice>}
  </div>;
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString([], { dateStyle: "medium", timeStyle: "short" });
}
