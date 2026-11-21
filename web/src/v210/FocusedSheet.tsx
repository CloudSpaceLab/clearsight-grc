import { useId, useLayoutEffect, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import "./focused-sheet.css";

/** Native modal dialog supplies background isolation and keyboard containment. */
export function FocusedSheet({ title, onDismiss, children }: { title: string; onDismiss: () => void; children: ReactNode }) {
  const dialog = useRef<HTMLDialogElement>(null);
  const close = useRef<HTMLButtonElement>(null);
  const headingID = useId();
  useLayoutEffect(() => {
    const element = dialog.current;
    if (!element) return;
    const previousOverflow = document.body.style.overflow;
    element.showModal();
    document.body.style.overflow = "hidden";
    close.current?.focus();
    return () => { if (element.open) element.close(); document.body.style.overflow = previousOverflow; };
  }, []);
  return createPortal(<dialog ref={dialog} className="v-focused-sheet" aria-labelledby={headingID} onCancel={(event) => { event.preventDefault(); onDismiss(); }} onClick={(event) => { const bounds = event.currentTarget.getBoundingClientRect(); if (event.target === event.currentTarget && (event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom)) onDismiss(); }}>
    <header className="v-focused-sheet-header"><h2 id={headingID}>{title}</h2><button ref={close} type="button" aria-label={`Close ${title.toLowerCase()}`} onClick={onDismiss}>×</button></header>
    <div className="v-focused-sheet-content">{children}</div>
  </dialog>, document.body);
}
