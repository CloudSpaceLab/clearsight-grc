import { Button as AriaButton } from "react-aria-components";
import type { StatusTone } from "./StatusBadge";

export type MetricCardQuality = "current" | "stale" | "partial" | "unknown";

export type MetricCardProps = {
  label: string;
  value: string | number;
  detail?: string;
  meta?: string;
  delta?: string;
  tone?: StatusTone;
  quality?: MetricCardQuality;
  qualityLabel?: string;
  actionLabel?: string;
  isSelected?: boolean;
  isDisabled?: boolean;
  ariaControls?: string;
  onPress?: () => void;
};

export function MetricCard({
  label,
  value,
  detail,
  meta,
  delta,
  tone = "neutral",
  quality = "current",
  qualityLabel,
  actionLabel,
  isSelected,
  isDisabled = false,
  ariaControls,
  onPress,
}: MetricCardProps) {
  const effectiveTone = metricTone(tone, quality);
  const status = qualityLabel || metricQualityLabel(quality);
  const className = `cs-metric-card cs-tone--${effectiveTone}`;
  const accessibleName = [
    `${label}: ${String(value)}`,
    delta,
    status,
    detail,
    meta,
    actionLabel,
  ].filter(Boolean).join(". ");

  const content = <>
    <span className="cs-metric-card__heading">
      <span className="cs-metric-card__label">{label}</span>
      <span className="cs-metric-card__quality"><span aria-hidden="true"/> {status}</span>
    </span>
    <span className="cs-metric-card__measure">
      <strong>{value}</strong>
      {delta && <span className="cs-metric-card__delta">{delta}</span>}
    </span>
    {detail && <small className="cs-metric-card__detail">{detail}</small>}
    {meta && <small className="cs-metric-card__meta">{meta}</small>}
    {actionLabel && <span className="cs-metric-card__action">{actionLabel}</span>}
  </>;

  if (onPress) {
    return <AriaButton
      aria-label={accessibleName}
      aria-controls={ariaControls}
      aria-pressed={isSelected}
      isDisabled={isDisabled}
      onPress={onPress}
      className={className}
      data-quality={quality}
    >
      {content}
    </AriaButton>;
  }

  return <article className={className} aria-label={accessibleName} data-quality={quality}>
    {content}
  </article>;
}

function metricTone(tone: StatusTone, quality: MetricCardQuality): StatusTone {
  if (quality === "stale") return "warning";
  if (quality === "partial" || quality === "unknown") return "unknown";
  return tone;
}

function metricQualityLabel(quality: MetricCardQuality) {
  switch (quality) {
    case "stale": return "Stale";
    case "partial": return "Coverage incomplete";
    case "unknown": return "Coverage unknown";
    default: return "Current";
  }
}
