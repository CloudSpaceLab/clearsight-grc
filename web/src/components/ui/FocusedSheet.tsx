import type { ReactNode } from "react";
import { useOverlayLifecycle } from "./useOverlayLifecycle";
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
  isDismissable?: boolean;
};

export function FocusedSheet({ label, onClose, children, panelClassName = "", closeLabel = "Close", size = "default", isDismissable = true }: FocusedSheetProps) {
  useOverlayLifecycle();

  return <ModalOverlay
    isOpen
    isDismissable={isDismissable}
    className="cs-sheet__overlay panel-backdrop"
    onOpenChange={(open) => { if (!open && isDismissable) onClose(); }}
    onMouseDown={(event) => { if (isDismissable && event.target === event.currentTarget) onClose(); }}
  >
    <Modal className={`cs-sheet cs-sheet--${size} ${panelClassName}`.trim()}>
      <Dialog aria-label={label} className="cs-sheet__dialog">
        <div className="cs-sheet__close">
          <IconButton autoFocus aria-label={closeLabel} isDisabled={!isDismissable} onPress={onClose} variant="quiet"><CloseIcon/></IconButton>
        </div>
        <div className="cs-sheet__content">{children}</div>
      </Dialog>
    </Modal>
  </ModalOverlay>;
}
