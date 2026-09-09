import type { ReactNode } from "react";
import { useOverlayLifecycle } from "./useOverlayLifecycle";
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
  useOverlayLifecycle();

  return <ModalOverlay isOpen isDismissable className="cs-dialog__overlay" onOpenChange={(open) => { if (!open) onClose(); }}>
    <Modal className={`cs-dialog cs-dialog--${size} ${panelClassName}`.trim()}>
      <Dialog aria-label={label} className="cs-dialog__body">
        <div className="cs-dialog__close"><IconButton autoFocus aria-label={closeLabel} onPress={onClose} variant="quiet"><CloseIcon/></IconButton></div>
        <div className="cs-dialog__content">{children}</div>
      </Dialog>
    </Modal>
  </ModalOverlay>;
}
