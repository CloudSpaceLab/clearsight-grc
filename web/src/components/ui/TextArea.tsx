import type { ChangeEventHandler } from "react";
import { FormField } from "./FormField";

export type TextAreaProps = {
  label: string;
  id?: string;
  autoComplete?: string;
  value: string;
  onChange: (value: string) => void;
  description?: string;
  errorMessage?: string;
  placeholder?: string;
  name?: string;
  rows?: number;
  maxLength?: number;
  isDisabled?: boolean;
  isReadOnly?: boolean;
  isRequired?: boolean;
  isInvalid?: boolean;
  isLoading?: boolean;
};

export function TextArea({ id, autoComplete, label, value, onChange, description, errorMessage, placeholder, name, rows = 4, maxLength, isDisabled = false, isReadOnly = false, isRequired = false, isInvalid = false, isLoading = false }: TextAreaProps) {
  const change: ChangeEventHandler<HTMLTextAreaElement> = (event) => onChange(event.target.value);
  return <FormField id={id} label={label} description={description} errorMessage={errorMessage} isInvalid={isInvalid} isRequired={isRequired} isLoading={isLoading}>
    {(control) => <textarea
      {...control}
      className="cs-field__control"
      name={name}
      autoComplete={autoComplete}
      value={value}
      placeholder={placeholder}
      rows={rows}
      maxLength={maxLength}
      disabled={isDisabled}
      readOnly={isReadOnly}
      required={isRequired}
      onChange={change}
    />}
  </FormField>;
}
