import type { ReactNode } from "react";
import { Button as AriaButton } from "react-aria-components";
import { StatusBadge, type StatusTone } from "./StatusBadge";

export type MetricCardProps = {
  label: string;
  value: number | string;
  status: string;
  tone?: StatusTone;
  detail?: string;
  meta?: string;
  actionLabel?: string;
  isSelected?: boolean;
  ariaControls?: string;
  onPress?: () => void;
};

export function MetricCard({
  label,
  value,
  status,
  tone = "neutral",
  detail,
  meta,
  actionLabel,
  isSelected = false,
  ariaControls,
  onPress,
}: MetricCardProps) {
  const body: ReactNode = <>
    <span className="cs-metric-card__heading">
      <span className="cs-metric-card__label">{label}</span>
      <StatusBadge tone={tone}>{status}</StatusBadge>
    </span>
    <strong className="cs-metric-card__value">{typeof value === "number" ? value.toLocaleString() : value}</strong>
    {detail && <span className="cs-metric-card__detail">{detail}</span>}
    {meta && <small className="cs-metric-card__meta">{meta}</small>}
    {actionLabel && <span className="cs-metric-card__action">{actionLabel}</span>}
  </>;

  if (!onPress) return <article className={`cs-metric-card cs-tone--${tone}`}>{body}</article>;

  return <AriaButton
    className={`cs-metric-card cs-metric-card--interactive cs-tone--${tone}`}
    aria-pressed={isSelected}
    aria-controls={ariaControls}
    onPress={onPress}
  >
    {body}
  </AriaButton>;
}
