import { useId, type ReactNode } from "react";

type EmptyStateProps = {
  population: string;
  title: string;
  description: string;
  action?: ReactNode;
};

export function EmptyState({ population, title, description, action }: EmptyStateProps) {
  const titleID = useId();
  return <section className="cs-empty-state" aria-labelledby={titleID}>
    <p className="cs-empty-state__population">{population}</p>
    <h3 className="cs-empty-state__title" id={titleID}>{title}</h3>
    <p className="cs-empty-state__description">{description}</p>
    {action && <div className="cs-empty-state__actions">{action}</div>}
  </section>;
}
