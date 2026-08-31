import { useEffect, useRef, type ReactNode } from "react";
import { Dialog, Modal, ModalOverlay } from "react-aria-components";
import { CloseIcon } from "../CloseIcon";
import { IconButton } from "./Button";

export type FocusedSheetProps = {
  label: string;
  onClose: () => void;
  children: ReactNode;
  panelClassName?: string;
  closeLabel?: string;
  size?: "default" | "wide";
};

export function FocusedSheet({ label, onClose, children, panelClassName = "", closeLabel = "Close", size = "default" }: FocusedSheetProps) {
  const invokerRef = useRef(document.activeElement instanceof HTMLElement ? document.activeElement : null);

  useEffect(() => {
    const overflow = document.body.style.overflow;
    const paddingRight = document.body.style.paddingRight;
    const documentWidth = document.documentElement.clientWidth;
    const scrollbarWidth = documentWidth > 0 ? Math.max(0, window.innerWidth - documentWidth) : 0;
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

  return <ModalOverlay
    isOpen
    isDismissable
    className="cs-sheet__overlay panel-backdrop"
    onOpenChange={(open) => { if (!open) onClose(); }}
    onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}
  >
    <Modal className={`cs-sheet cs-sheet--${size} ${panelClassName}`.trim()}>
      <Dialog aria-label={label} className="cs-sheet__dialog">
        <div className="cs-sheet__close">
          <IconButton autoFocus aria-label={closeLabel} onPress={onClose} variant="quiet"><CloseIcon/></IconButton>
        </div>
        <div className="cs-sheet__content">{children}</div>
      </Dialog>
    </Modal>
  </ModalOverlay>;
}
