import type { ReactNode } from "react";
import { Button as AriaButton, type ButtonProps as AriaButtonProps } from "react-aria-components";

export type ActionCardProps = Omit<AriaButtonProps, "children" | "className" | "style" | "isDisabled" | "aria-label"> & {
  title: string;
  description: string;
  icon?: ReactNode;
  isDisabled?: boolean;
};

export function ActionCard({ title, description, icon, isDisabled = false, ...props }: ActionCardProps) {
  return <AriaButton {...props} aria-label={title} isDisabled={isDisabled} className="cs-action-card">
    {icon && <span className="cs-action-card__icon" aria-hidden="true">{icon}</span>}
    <span className="cs-action-card__copy">
      <strong>{title}</strong>
      <small>{description}</small>
    </span>
    <span className="cs-action-card__arrow" aria-hidden="true">›</span>
  </AriaButton>;
}
