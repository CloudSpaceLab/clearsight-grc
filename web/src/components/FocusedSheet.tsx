import { useEffect, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { CloseIcon } from "./CloseIcon";

type Props = {
  label: string;
  onClose: () => void;
  children: ReactNode;
  panelClassName?: string;
  backdropClassName?: string;
  closeLabel?: string;
};

export function FocusedSheet({ label, onClose, children, panelClassName = "", backdropClassName = "", closeLabel = "Close" }: Props) {
  const panelRef = useRef<HTMLElement>(null);

  useEffect(() => {
    const panel = panelRef.current;
    if (!panel) return;
    const activePanel = panel;
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const background = Array.from(document.querySelectorAll<HTMLElement>(".app-shell > .sidebar, .app-shell > main, .app-shell > .mobile-nav, .app-shell > .guide-launcher"));
    const previous = background.map((element) => ({
      element,
      inert: element.hasAttribute("inert"),
      ariaHidden: element.getAttribute("aria-hidden"),
    }));
    for (const { element } of previous) {
      element.setAttribute("inert", "");
      element.setAttribute("aria-hidden", "true");
    }

    const focusable = () => Array.from(activePanel.querySelectorAll<HTMLElement>("button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), summary, [tabindex]:not([tabindex='-1'])"));
    focusable()[0]?.focus();

    function keydown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        event.preventDefault();
        onClose();
        return;
      }
      if (event.key !== "Tab") return;
      const values = focusable();
      if (!values.length) {
        event.preventDefault();
        activePanel.focus();
        return;
      }
      const first = values[0];
      const last = values[values.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last?.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first?.focus();
      }
    }

    activePanel.addEventListener("keydown", keydown);
    return () => {
      activePanel.removeEventListener("keydown", keydown);
      for (const item of previous) {
        if (!item.inert) item.element.removeAttribute("inert");
        if (item.ariaHidden == null) item.element.removeAttribute("aria-hidden");
        else item.element.setAttribute("aria-hidden", item.ariaHidden);
      }
      previousFocus?.focus?.();
    };
  }, [onClose]);

  return createPortal(<div className={`panel-backdrop ${backdropClassName}`.trim()} onMouseDown={onClose}>
    <aside ref={panelRef} className={`side-panel ${panelClassName}`.trim()} role="dialog" aria-modal="true" aria-label={label} tabIndex={-1} onMouseDown={(event) => event.stopPropagation()}>
      <button className="panel-close icon-button" type="button" onClick={onClose} aria-label={closeLabel}><CloseIcon/></button>
      {children}
    </aside>
  </div>, document.body);
}
