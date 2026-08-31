import { Button, type ButtonProps } from "react-aria-components";

export type SelectableRecordProps = Omit<ButtonProps, "children" | "className" | "aria-pressed"> & {
  title: string;
  metadata: string;
  description?: string;
  isSelected?: boolean;
};

export function SelectableRecord({ title, metadata, description, isSelected = false, ...props }: SelectableRecordProps) {
  return <Button
    {...props}
    className="cs-selectable-record"
    aria-pressed={isSelected}
    data-selected={isSelected || undefined}
  >
    <span className="cs-selectable-record__title">{title}</span>
    <span className="cs-selectable-record__metadata">{metadata}</span>
    {description && <span className="cs-selectable-record__description">{description}</span>}
  </Button>;
}
