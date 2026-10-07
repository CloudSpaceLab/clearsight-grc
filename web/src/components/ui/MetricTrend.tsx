export type MetricTrendDatum = {
  id: string;
  at: string;
  label: string;
  value: number;
  displayValue?: string;
};

export type MetricTrendProps = {
  ariaLabel: string;
  points: readonly MetricTrendDatum[];
  gapThresholdMs?: number;
};

export function MetricTrend({
  ariaLabel,
  points,
  gapThresholdMs = 36 * 60 * 60 * 1000,
}: MetricTrendProps) {
  const values = [...points]
    .map((point) => ({ ...point, timestamp: Date.parse(point.at) }))
    .filter((point) => Number.isFinite(point.timestamp))
    .sort((left, right) => left.timestamp - right.timestamp);

  if (values.length === 0) return null;

  const first = values[0]!;
  const last = values[values.length - 1]!;
  const minTime = first.timestamp;
  const maxTime = last.timestamp;
  const minValue = Math.min(...values.map((point) => point.value));
  const maxValue = Math.max(...values.map((point) => point.value));
  const timeRange = maxTime - minTime;
  const valueRange = maxValue - minValue;
  const width = 1000;
  const height = 180;
  const insetX = 20;
  const insetY = 18;
  const plotWidth = width - insetX * 2;
  const plotHeight = height - insetY * 2;

  const plotted = values.map((point) => ({
    ...point,
    x: timeRange === 0 ? width / 2 : insetX + ((point.timestamp - minTime) / timeRange) * plotWidth,
    y: valueRange === 0 ? height / 2 : insetY + (1 - ((point.value - minValue) / valueRange)) * plotHeight,
  }));

  const segments: typeof plotted[] = [];
  let current: typeof plotted = [];
  for (const point of plotted) {
    const previous = current[current.length - 1];
    if (previous && point.timestamp - previous.timestamp > gapThresholdMs) {
      segments.push(current);
      current = [];
    }
    current.push(point);
  }
  if (current.length) segments.push(current);

  return <figure className="cs-metric-trend">
    <svg
      viewBox={`0 0 ${width} ${height}`}
      role="img"
      aria-label={ariaLabel}
      preserveAspectRatio="none"
      className="cs-metric-trend__plot"
    >
      <line className="cs-metric-trend__baseline" x1={insetX} y1={height - insetY} x2={width - insetX} y2={height - insetY}/>
      {segments.map((segment, index) => segment.length > 1
        ? <polyline
          key={`segment-${index}`}
          className="cs-metric-trend__line"
          points={segment.map((point) => `${point.x},${point.y}`).join(" ")}
          fill="none"
          vectorEffect="non-scaling-stroke"
        />
        : null)}
      {plotted.map((point) => <circle
        key={point.id}
        className="cs-metric-trend__point"
        cx={point.x}
        cy={point.y}
        r="4"
        vectorEffect="non-scaling-stroke"
      />)}
    </svg>
    <figcaption className="cs-metric-trend__range">
      <span>{first.label}</span>
      <span>{last.label}</span>
    </figcaption>
    <table className="cs-sr-only">
      <caption>{ariaLabel}</caption>
      <thead><tr><th>Date</th><th>Value</th></tr></thead>
      <tbody>{values.map((point) => <tr key={`row-${point.id}`}><td>{point.label}</td><td>{point.displayValue ?? point.value}</td></tr>)}</tbody>
    </table>
  </figure>;
}
