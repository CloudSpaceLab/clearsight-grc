import { useEffect, useRef, type ReactNode } from "react";
import { Dialog, Modal, ModalOverlay } from "react-aria-components";
import { CloseIcon } from "../CloseIcon";
import { IconButton } from "./Button";
import "./FocusedDialog.css";

export type FocusedDialogProps = {
  label: string;
  onClose: () => void;
  children: ReactNode;
  panelClassName?: string;
  closeLabel?: string;
  size?: "default" | "wide";
};

export function FocusedDialog({ label, onClose, children, panelClassName = "", closeLabel = "Close", size = "default" }: FocusedDialogProps) {
  const invokerRef = useRef(document.activeElement instanceof HTMLElement ? document.activeElement : null);

  useEffect(() => {
    const overflow = document.body.style.overflow;
    const paddingRight = document.body.style.paddingRight;
    const scrollbarWidth = Math.max(0, window.innerWidth - document.documentElement.clientWidth);
    const currentPadding = Number.parseFloat(window.getComputedStyle(document.body).paddingRight) || 0;
    document.body.style.overflow = "hidden";
    if (scrollbarWidth > 0) document.body.style.paddingRight = `${currentPadding + scrollbarWidth}px`;
    return () => {
      document.body.style.overflow = overflow;
      document.body.style.paddingRight = paddingRight;
      const invoker = invokerRef.current;
      queueMicrotask(() => invoker?.focus());
    };
  }, []);

  return <ModalOverlay isOpen isDismissable className="cs-dialog__overlay" onOpenChange={(open) => { if (!open) onClose(); }}>
    <Modal className={`cs-dialog cs-dialog--${size} ${panelClassName}`.trim()}>
      <Dialog aria-label={label} className="cs-dialog__body">
        <div className="cs-dialog__close"><IconButton autoFocus aria-label={closeLabel} onPress={onClose} variant="quiet"><CloseIcon/></IconButton></div>
        <div className="cs-dialog__content">{children}</div>
      </Dialog>
    </Modal>
  </ModalOverlay>;
}
