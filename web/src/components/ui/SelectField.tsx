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
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";

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
  const [isOpen, setIsOpen] = useState(false);
  const [portalContainer, setPortalContainer] = useState<Element>();
  const openScrollPosition = useRef<{ x: number; y: number } | undefined>(undefined);
  const openedAt = useRef(0);
  const restoringScroll = useRef(false);
  const allowClose = useRef(false);
  const restoreReleaseTimer = useRef<number | undefined>(undefined);
  const selectRef = useCallback((node: HTMLDivElement | null) => {
    if (!node) return;
    // Keep a modal as the option list's accessibility and dismissal boundary.
    // Outside a modal, the fixed application overlay root keeps the list out
    // of sticky, scrollable workspace layouts while retaining the main
    // landmark boundary. Portalling directly into a layout landmark can make
    // focusing a selected option scroll the document and immediately close
    // the list through React Aria's close-on-scroll behavior.
    const dialog = node.closest('[role="dialog"]');
    const nextContainer = dialog ?? document.getElementById("cs-overlay-root") ?? undefined;
    setPortalContainer((current) => current === nextContainer ? current : nextContainer);
  }, []);
  useEffect(() => () => window.clearTimeout(restoreReleaseTimer.current), []);
  useLayoutEffect(() => {
    if (!isOpen) return;
    function restoreOpeningPosition() {
      const position = openScrollPosition.current;
      if (!position || performance.now() - openedAt.current >= 250) return;
      const scrollShifted = Math.abs(window.scrollX - position.x) > 0.5 || Math.abs(window.scrollY - position.y) > 0.5;
      if (!scrollShifted) return;
      restoringScroll.current = true;
      window.scrollTo({ left: position.x, top: position.y, behavior: "instant" });
      window.clearTimeout(restoreReleaseTimer.current);
      restoreReleaseTimer.current = window.setTimeout(() => { restoringScroll.current = false; }, 100);
    }
    document.addEventListener("scroll", restoreOpeningPosition, true);
    return () => document.removeEventListener("scroll", restoreOpeningPosition, true);
  }, [isOpen]);

  function finishClose() {
    allowClose.current = false;
    restoringScroll.current = false;
    openScrollPosition.current = undefined;
    setIsOpen(false);
  }

  function handleOpenChange(nextOpen: boolean) {
    if (nextOpen) {
      window.clearTimeout(restoreReleaseTimer.current);
      openScrollPosition.current = { x: window.scrollX, y: window.scrollY };
      openedAt.current = performance.now();
      restoringScroll.current = false;
      allowClose.current = false;
      setIsOpen(true);
      return;
    }
    if (allowClose.current) {
      finishClose();
      return;
    }

    const position = openScrollPosition.current;
    const scrollShifted = position && (Math.abs(window.scrollX - position.x) > 0.5 || Math.abs(window.scrollY - position.y) > 0.5);
    const initialPositioningClose = scrollShifted && performance.now() - openedAt.current < 250;
    if (position && (restoringScroll.current || initialPositioningClose)) {
      restoringScroll.current = true;
      window.scrollTo({ left: position.x, top: position.y, behavior: "instant" });
      window.clearTimeout(restoreReleaseTimer.current);
      restoreReleaseTimer.current = window.setTimeout(() => { restoringScroll.current = false; }, 100);
      return;
    }
    finishClose();
  }

  function permitClose(event: { key: string }) {
    if (event.key === "Escape" || event.key === "Tab") allowClose.current = true;
  }

  function change(key: Key | null) {
    allowClose.current = true;
    finishClose();
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
    isOpen={isOpen}
    onOpenChange={handleOpenChange}
  >
    <Label className="cs-select-field__label">{label}{isRequired && <span className="cs-field__required" aria-hidden="true"> *</span>}</Label>
    <AriaButton className="cs-select-field__trigger" onPressStart={() => { if (isOpen) allowClose.current = true; }} onKeyDown={permitClose}>
      <SelectValue className="cs-select-field__value"/>
      <span className="cs-select-field__chevron" aria-hidden="true">⌄</span>
    </AriaButton>
    {description && <Text className="cs-select-field__description" slot="description">{description}</Text>}
    {isInvalid && errorMessage && <FieldError className="cs-select-field__error">{errorMessage}</FieldError>}
    <Popover className="cs-select-field__popover" isNonModal UNSTABLE_portalContainer={portalContainer}>
      <div onKeyDownCapture={permitClose}>
        <ListBox className="cs-select-field__listbox">
          {allowsEmpty && <ListBoxItem id={emptyKey} textValue={placeholder} className="cs-select-field__option">{placeholder}</ListBoxItem>}
          {options.map((option) => <ListBoxItem key={option.id} id={option.id} textValue={option.label} className="cs-select-field__option">
            <span>{option.label}</span>
            {option.description && <span className="cs-select-field__option-description">{option.description}</span>}
          </ListBoxItem>)}
        </ListBox>
      </div>
    </Popover>
  </Select>;
}
