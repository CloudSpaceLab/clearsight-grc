import type { ChangeEventHandler, FocusEventHandler } from "react";
import { FormField } from "./FormField";

export type TextFieldType = "text" | "search" | "email" | "url" | "tel" | "number" | "date" | "time" | "datetime-local";

export type TextFieldProps = {
  label: string;
  value: string;
  onChange: (value: string) => void;
  description?: string;
  errorMessage?: string;
  placeholder?: string;
  type?: TextFieldType;
  name?: string;
  autoComplete?: string;
  min?: number;
  max?: number;
  step?: number;
  maxLength?: number;
  isDisabled?: boolean;
  isReadOnly?: boolean;
  isRequired?: boolean;
  isInvalid?: boolean;
  isLoading?: boolean;
  isLabelHidden?: boolean;
  onBlur?: FocusEventHandler<HTMLInputElement>;
};

export function TextField({
  label,
  value,
  onChange,
  description,
  errorMessage,
  placeholder,
  type = "text",
  name,
  autoComplete,
  min,
  max,
  step,
  maxLength,
  isDisabled = false,
  isReadOnly = false,
  isRequired = false,
  isInvalid = false,
  isLoading = false,
  isLabelHidden = false,
  onBlur,
}: TextFieldProps) {
  const change: ChangeEventHandler<HTMLInputElement> = (event) => onChange(event.target.value);
  return <FormField label={label} description={description} errorMessage={errorMessage} isInvalid={isInvalid} isRequired={isRequired} isLoading={isLoading} isLabelHidden={isLabelHidden}>
    {(control) => <input
      {...control}
      className="cs-field__control"
      type={type}
      name={name}
      value={value}
      placeholder={placeholder}
      autoComplete={autoComplete}
      min={min}
      max={max}
      step={step}
      maxLength={maxLength}
      disabled={isDisabled}
      readOnly={isReadOnly}
      required={isRequired}
      onChange={change}
      onBlur={onBlur}
    />}
  </FormField>;
}
