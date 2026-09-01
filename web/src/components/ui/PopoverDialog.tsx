import type { ReactElement, ReactNode } from "react";
import { Dialog, DialogTrigger, Popover, type Placement } from "react-aria-components";

export type PopoverDialogProps = {
  label: string;
  trigger: ReactElement;
  children: ReactNode;
  isOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  placement?: Placement;
};

export function PopoverDialog({ label, trigger, children, isOpen, onOpenChange, placement = "bottom start" }: PopoverDialogProps) {
  return <DialogTrigger isOpen={isOpen} onOpenChange={onOpenChange}>
    {trigger}
    <Popover className="cs-popover" placement={placement}>
      <Dialog className="cs-popover__dialog" aria-label={label}>{children}</Dialog>
    </Popover>
  </DialogTrigger>;
}
