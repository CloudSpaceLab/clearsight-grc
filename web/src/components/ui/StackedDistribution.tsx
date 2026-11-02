export type DistributionTone = "error" | "warning" | "success" | "neutral" | "unknown";

export type DistributionSegment = {
  id: string;
  label: string;
  count: number;
  tone: DistributionTone;
};

type Props = {
  ariaLabel: string;
  segments: readonly DistributionSegment[];
};

export function StackedDistribution({ ariaLabel, segments }: Props) {
  const total = segments.reduce((sum, segment) => sum + Math.max(0, segment.count), 0);
  const description = segments.map(({ count, label }) => `${count} ${label.toLowerCase()}`).join(", ");

  return <div className="cs-stacked-distribution">
    <div className="cs-stacked-distribution__track" role="img" aria-label={`${ariaLabel}: ${description}`}>
      {total > 0 && segments.filter((segment) => segment.count > 0).map((segment) =>
        <span key={segment.id} className={`cs-stacked-distribution__part cs-stacked-distribution__part--${segment.tone}`}
          style={{ flexGrow: segment.count }} aria-hidden="true"/>
      )}
    </div>
    <div className="cs-stacked-distribution__legend" aria-hidden="true">
      {segments.filter((segment) => segment.count > 0).map((segment) =>
        <div key={segment.id} className="cs-stacked-distribution__item">
          <span className={`cs-stacked-distribution__dot cs-stacked-distribution__dot--${segment.tone}`}/>
          <span><strong>{segment.count}</strong> {segment.label}</span>
        </div>
      )}
    </div>
  </div>;
}
