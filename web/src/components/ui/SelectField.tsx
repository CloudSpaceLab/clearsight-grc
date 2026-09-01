import {
  Button as AriaButton,
  FieldError,
  Label,
  ListBox,
  ListBoxItem,
  Popover,
  Select,
  SelectValue,
  Text,
  type Key,
} from "react-aria-components";
import { useCallback, useState } from "react";

export type SelectOption<T extends string> = {
  id: T;
  label: string;
  description?: string;
};

export type SelectFieldProps<T extends string> = {
  label: string;
  value?: T;
  placeholder: string;
  options: readonly SelectOption<T>[];
  onChange: (value: T | undefined) => void;
  description?: string;
  errorMessage?: string;
  isDisabled?: boolean;
  isRequired?: boolean;
  isInvalid?: boolean;
  allowsEmpty?: boolean;
  isLabelHidden?: boolean;
};

const emptyKey = "__clearsight_empty_selection__";

export function SelectField<T extends string>({ label, value, placeholder, options, onChange, description, errorMessage, isDisabled = false, isRequired = false, isInvalid = false, allowsEmpty = true, isLabelHidden = false }: SelectFieldProps<T>) {
  const selectedKey = value ?? null;
  const [portalContainer, setPortalContainer] = useState<Element>();
  const selectRef = useCallback((node: HTMLDivElement | null) => {
    if (!node) return;
    // A modal remains the option list's accessibility and dismissal boundary.
    // Outside a modal, use the outer workspace landmark: nested builder/canvas
    // landmarks can be sticky or scrollable and must not become the popover's
    // positioning or focus-scroll container.
    const dialog = node.closest('[role="dialog"]');
    let landmark = node.closest('main, [role="main"]');
    while (landmark) {
      const parentLandmark = landmark.parentElement?.closest('main, [role="main"]') ?? null;
      if (!parentLandmark) break;
      landmark = parentLandmark;
    }
    const nextContainer = dialog ?? landmark ?? undefined;
    setPortalContainer((current) => current === nextContainer ? current : nextContainer);
  }, []);
  function change(key: Key | null) {
    onChange(key === null || key === emptyKey ? undefined : String(key) as T);
  }

  return <Select
    ref={selectRef}
    className={`cs-select-field${isLabelHidden ? " cs-select-field--label-hidden" : ""}`}
    selectedKey={selectedKey}
    onSelectionChange={change}
    placeholder={placeholder}
    isDisabled={isDisabled}
    isRequired={isRequired}
    isInvalid={isInvalid}
  >
    <Label className="cs-select-field__label">{label}{isRequired && <span className="cs-field__required" aria-hidden="true"> *</span>}</Label>
    <AriaButton className="cs-select-field__trigger">
      <SelectValue className="cs-select-field__value"/>
      <span className="cs-select-field__chevron" aria-hidden="true">⌄</span>
    </AriaButton>
    {description && <Text className="cs-select-field__description" slot="description">{description}</Text>}
    {isInvalid && errorMessage && <FieldError className="cs-select-field__error">{errorMessage}</FieldError>}
    <Popover className="cs-select-field__popover" isNonModal UNSTABLE_portalContainer={portalContainer}>
      <ListBox className="cs-select-field__listbox">
        {allowsEmpty && <ListBoxItem id={emptyKey} textValue={placeholder} className="cs-select-field__option">{placeholder}</ListBoxItem>}
        {options.map((option) => <ListBoxItem key={option.id} id={option.id} textValue={option.label} className="cs-select-field__option">
          <span>{option.label}</span>
          {option.description && <span className="cs-select-field__option-description">{option.description}</span>}
        </ListBoxItem>)}
      </ListBox>
    </Popover>
  </Select>;
}
