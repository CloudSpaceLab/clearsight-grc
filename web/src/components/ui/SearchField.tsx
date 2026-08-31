import { useId, type ChangeEventHandler } from "react";

export type SearchFieldProps = {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  isDisabled?: boolean;
  isLoading?: boolean;
};

export function SearchField({ label, value, onChange, placeholder, isDisabled = false, isLoading = false }: SearchFieldProps) {
  const id = useId();
  const change: ChangeEventHandler<HTMLInputElement> = (event) => onChange(event.target.value);
  return <label className="cs-search-field" htmlFor={id} data-disabled={isDisabled || undefined}>
    <span className="cs-sr-only">{label}</span>
    <svg className="cs-search-field__icon" viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="11" cy="11" r="6.5"/>
      <path d="m16 16 4 4"/>
    </svg>
    <input
      id={id}
      className="cs-search-field__control"
      type="search"
      value={value}
      placeholder={placeholder}
      disabled={isDisabled}
      aria-busy={isLoading || undefined}
      onChange={change}
    />
  </label>;
}
