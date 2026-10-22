import { Button as AriaButton } from "react-aria-components";

export type RankedBarItem = {
  id: string;
  label: string;
  value: number;
  displayValue?: string;
  meta?: string;
  actionLabel?: string;
  isDisabled?: boolean;
};

export type RankedBarListProps = {
  ariaLabel: string;
  items: readonly RankedBarItem[];
  layout?: "stacked" | "inline";
  valueLabel?: (value: number) => string;
  onAction?: (item: RankedBarItem) => void;
};

export function RankedBarList({
  ariaLabel,
  items,
  layout = "stacked",
  valueLabel = (value) => String(value),
  onAction,
}: RankedBarListProps) {
  const max = Math.max(1, ...items.map((item) => item.value));

  return <ol className="cs-ranked-bars" data-layout={layout} aria-label={ariaLabel}>
    {items.map((item) => {
      const interactive = Boolean(onAction && !item.isDisabled);
      const renderedValue = item.displayValue ?? valueLabel(item.value);
      const label = <span className="cs-ranked-bars__label">{item.label}</span>;
      const value = <strong className="cs-ranked-bars__value">{renderedValue}</strong>;
      const track = <span className="cs-ranked-bars__track" aria-hidden="true">
        <span className="cs-ranked-bars__fill" style={{ inlineSize: item.value <= 0 ? "0%" : `${Math.max(2, item.value / max * 100)}%` }}/>
      </span>;
      const content = <>
        {layout === "inline" ? <>{label}{track}{value}</> : <>
          <span className="cs-ranked-bars__heading">{label}{value}</span>
          {track}
        </>}
        {item.meta && <small>{item.meta}</small>}
      </>;

      return <li key={item.id}>
        {interactive
          ? <AriaButton
            className="cs-ranked-bars__row"
            aria-label={item.actionLabel || `${item.label}: ${renderedValue}`}
            onPress={() => onAction?.(item)}
          >{content}</AriaButton>
          : <div className="cs-ranked-bars__row" aria-label={`${item.label}: ${renderedValue}`}>{content}</div>}
      </li>;
    })}
  </ol>;
}
