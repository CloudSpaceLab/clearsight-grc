import type { ReactElement, ReactNode } from "react";
import { Button as AriaButton, Dialog, DialogTrigger, Popover, type Placement } from "react-aria-components";

export type PopoverDialogProps = {
  label: string;
  trigger?: ReactElement;
  triggerLabel?: string;
  triggerChildren?: ReactNode;
  triggerClassName?: string;
  triggerDisabled?: boolean;
  children: ReactNode;
  isOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  placement?: Placement;
};

export function PopoverDialog({
  label,
  trigger,
  triggerLabel,
  triggerChildren,
  triggerClassName,
  triggerDisabled = false,
  children,
  isOpen,
  onOpenChange,
  placement = "bottom start",
}: PopoverDialogProps) {
  const resolvedTrigger = trigger ?? <AriaButton
    aria-label={triggerLabel ?? label}
    className={triggerClassName}
    isDisabled={triggerDisabled}
  >{triggerChildren ?? label}</AriaButton>;

  return <DialogTrigger isOpen={isOpen} onOpenChange={onOpenChange}>
    {resolvedTrigger}
    <Popover className="cs-popover" placement={placement}>
      <Dialog className="cs-popover__dialog" aria-label={label}>{children}</Dialog>
    </Popover>
  </DialogTrigger>;
}
