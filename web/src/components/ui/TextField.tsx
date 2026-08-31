import type { ChangeEventHandler, FocusEventHandler } from "react";
import { FormField } from "./FormField";

export type TextFieldType = "text" | "search" | "email" | "url" | "tel";

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
  isDisabled?: boolean;
  isReadOnly?: boolean;
  isRequired?: boolean;
  isInvalid?: boolean;
  isLoading?: boolean;
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
  isDisabled = false,
  isReadOnly = false,
  isRequired = false,
  isInvalid = false,
  isLoading = false,
  onBlur,
}: TextFieldProps) {
  const change: ChangeEventHandler<HTMLInputElement> = (event) => onChange(event.target.value);
  return <FormField label={label} description={description} errorMessage={errorMessage} isInvalid={isInvalid} isRequired={isRequired} isLoading={isLoading}>
    {(control) => <input
      {...control}
      className="cs-field__control"
      type={type}
      name={name}
      value={value}
      placeholder={placeholder}
      autoComplete={autoComplete}
      disabled={isDisabled}
      readOnly={isReadOnly}
      required={isRequired}
      onChange={change}
      onBlur={onBlur}
    />}
  </FormField>;
}
