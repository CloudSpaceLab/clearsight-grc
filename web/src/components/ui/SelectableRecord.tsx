import type { ReactNode } from "react";
import { Button, type ButtonProps } from "react-aria-components";

export type SelectableRecordProps = Omit<ButtonProps, "children" | "className" | "aria-pressed"> & {
  title: string;
  metadata: string;
  description?: string;
  endContent?: ReactNode;
  isSelected?: boolean;
};

export function SelectableRecord({ title, metadata, description, endContent, isSelected = false, ...props }: SelectableRecordProps) {
  return <Button
    {...props}
    className="cs-selectable-record"
    aria-pressed={isSelected}
    data-selected={isSelected || undefined}
    data-has-end-content={endContent == null ? undefined : "true"}
  >
    <span className="cs-selectable-record__title">{title}</span>
    <span className="cs-selectable-record__metadata">{metadata}</span>
    {description && <span className="cs-selectable-record__description">{description}</span>}
    {endContent != null && <span className="cs-selectable-record__end">{endContent}</span>}
  </Button>;
}
