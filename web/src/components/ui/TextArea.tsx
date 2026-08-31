import type { ChangeEventHandler } from "react";
import { FormField } from "./FormField";

export type TextAreaProps = {
  label: string;
  value: string;
  onChange: (value: string) => void;
  description?: string;
  errorMessage?: string;
  placeholder?: string;
  name?: string;
  rows?: number;
  isDisabled?: boolean;
  isReadOnly?: boolean;
  isRequired?: boolean;
  isInvalid?: boolean;
  isLoading?: boolean;
};

export function TextArea({ label, value, onChange, description, errorMessage, placeholder, name, rows = 4, isDisabled = false, isReadOnly = false, isRequired = false, isInvalid = false, isLoading = false }: TextAreaProps) {
  const change: ChangeEventHandler<HTMLTextAreaElement> = (event) => onChange(event.target.value);
  return <FormField label={label} description={description} errorMessage={errorMessage} isInvalid={isInvalid} isRequired={isRequired} isLoading={isLoading}>
    {(control) => <textarea
      {...control}
      className="cs-field__control"
      name={name}
      value={value}
      placeholder={placeholder}
      rows={rows}
      disabled={isDisabled}
      readOnly={isReadOnly}
      required={isRequired}
      onChange={change}
    />}
  </FormField>;
}
