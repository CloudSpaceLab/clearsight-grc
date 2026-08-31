import { useId } from "react";
import { Checkbox as AriaCheckbox, type CheckboxProps as AriaCheckboxProps } from "react-aria-components";

export type CheckboxFieldProps = Omit<AriaCheckboxProps, "children" | "className" | "style" | "aria-label" | "aria-describedby"> & {
  label: string;
  description?: string;
  isLabelHidden?: boolean;
};

export function CheckboxField({ label, description, isLabelHidden = false, ...props }: CheckboxFieldProps) {
  const descriptionID = useId();
  return <AriaCheckbox
    {...props}
    aria-label={label}
    aria-describedby={description ? descriptionID : undefined}
    className={`cs-checkbox-field${isLabelHidden ? " cs-checkbox-field--label-hidden" : ""}`}
  >
    {({ isSelected, isIndeterminate }) => <>
      <span className="cs-checkbox-field__box" aria-hidden="true">
        <span>{isIndeterminate ? "−" : isSelected ? "✓" : ""}</span>
      </span>
      <span className="cs-checkbox-field__content">
        <span className="cs-checkbox-field__label">{label}</span>
        {description && <span id={descriptionID} className="cs-checkbox-field__description">{description}</span>}
      </span>
    </>}
  </AriaCheckbox>;
}
