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
  const [custom, setCustom] = useState(false);
  const [startDate, setStartDate] = useState(period.start_date);

  useEffect(() => { setStartDate(period.start_date); }, [period.start_date]);

  const minimum = minimumReportingStart(period);
  const invalid = !startDate || startDate < minimum || startDate > period.end_date;

  function handleOpenChange(next: boolean) {
    setOpen(next);
    if (!next) {
      setCustom(false);
      setStartDate(period.start_date);
    }
  }

  function apply(query: ReportingPeriodQuery) {
    setOpen(false);
    setCustom(false);
    onApply(query);
  }

  return <div className="oversight-period-control">
    <PopoverDialog
      label="Choose reporting period"
      isOpen={open}
      onOpenChange={handleOpenChange}
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
        <div className="oversight-period-picker__intro">
          <strong>Reporting period</strong>
          <p>Current posture stays current. History measures use this window.</p>
        </div>

        <div className="oversight-period-presets" aria-label="Reporting period presets">
          {presets.map((days) => <Button
            key={days}
            size="compact"
            variant="secondary"
            isDisabled={isChanging}
            onPress={() => apply({ start_date: startDateForDays(period.end_date, days), end_date: period.end_date })}
          >{days} days</Button>)}
        </div>

        {!custom && <Button
          size="compact"
          variant="quiet"
          isDisabled={isChanging}
          onPress={() => setCustom(true)}
        >Custom start date</Button>}

        {custom && <div className="oversight-period-custom">
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
          <div className="oversight-period-end" aria-label={`To ${period.end_date}`}>
            <span>To</span>
            <strong>{formatReportingDate(period.end_date)}</strong>
            <small>Current reporting date</small>
          </div>
          <div className="oversight-period-actions">
            <Button size="compact" variant="quiet" onPress={() => { setCustom(false); setStartDate(period.start_date); }}>Back</Button>
            <Button size="compact" isDisabled={invalid || isChanging} onPress={() => apply({ start_date: startDate, end_date: period.end_date })}>
              {isChanging ? "Applying…" : "Apply"}
            </Button>
          </div>
        </div>}
      </div>
    </PopoverDialog>
    <small>{freshness === "CURRENT" ? "Current" : "Needs refresh"} · Updated {formatDateTime(generatedAt)}</small>
    {error && <Notice tone="warning">{error}</Notice>}
  </div>;
}

function formatReportingDate(value: string) {
  return new Date(`${value}T00:00:00Z`).toLocaleDateString([], { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" });
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString([], { dateStyle: "medium", timeStyle: "short" });
}
