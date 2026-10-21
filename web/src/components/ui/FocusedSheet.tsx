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
  placement?: "start" | "end";
  isDismissable?: boolean;
};

export function FocusedSheet({ label, onClose, children, panelClassName = "", closeLabel = "Close", size = "default", placement = "end", isDismissable = true }: FocusedSheetProps) {
  useOverlayLifecycle();
  // The legacy alias forces right alignment outside the component layer. Keep it
  // only for the existing end placement; start uses the shared overlay contract.
  const legacyOverlayClass = placement === "end" ? " panel-backdrop" : "";

  return <ModalOverlay
    isOpen
    isDismissable={isDismissable}
    className={`cs-sheet__overlay cs-sheet__overlay--${placement}${legacyOverlayClass}`}
    onOpenChange={(open) => { if (!open && isDismissable) onClose(); }}
    onMouseDown={(event) => { if (isDismissable && event.target === event.currentTarget) onClose(); }}
  >
    <Modal className={`cs-sheet cs-sheet--${size} cs-sheet--${placement} ${panelClassName}`.trim()}>
      <Dialog aria-label={label} className="cs-sheet__dialog">
        <div className="cs-sheet__close">
          <IconButton autoFocus aria-label={closeLabel} isDisabled={!isDismissable} onPress={onClose} variant="quiet"><CloseIcon/></IconButton>
        </div>
        <div className="cs-sheet__content">{children}</div>
      </Dialog>
    </Modal>
  </ModalOverlay>;
}
