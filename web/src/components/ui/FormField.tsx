import { useId, type ReactNode } from "react";

export type FieldControlProps = {
  id: string;
  "aria-describedby"?: string;
  "aria-invalid"?: true;
  "aria-busy"?: true;
};

export type FormFieldProps = {
  label: string;
  description?: string;
  errorMessage?: string;
  isInvalid?: boolean;
  isRequired?: boolean;
  isLoading?: boolean;
  children: (props: FieldControlProps) => ReactNode;
};

export function FormField({ label, description, errorMessage, isInvalid = false, isRequired = false, isLoading = false, children }: FormFieldProps) {
  const controlID = useId();
  const descriptionID = useId();
  const errorID = useId();
  const describedBy = [description ? descriptionID : undefined, isInvalid && errorMessage ? errorID : undefined].filter(Boolean).join(" ") || undefined;

  return <div className="cs-field" data-invalid={isInvalid || undefined} data-required={isRequired || undefined}>
    <label className="cs-field__label" htmlFor={controlID}>
      {label}{isRequired && <><span className="cs-field__required" aria-hidden="true"> *</span><span className="cs-sr-only"> required</span></>}
    </label>
    {children({
      id: controlID,
      "aria-describedby": describedBy,
      "aria-invalid": isInvalid || undefined,
      "aria-busy": isLoading || undefined,
    })}
    {description && <p className="cs-field__description" id={descriptionID}>{description}</p>}
    {isInvalid && errorMessage && <p className="cs-field__error" id={errorID} role="alert">{errorMessage}</p>}
  </div>;
}
