import { useId, type ReactNode } from "react";
import {
  Button as AriaButton,
  Link as AriaLink,
  type ButtonProps as AriaButtonProps,
  type LinkProps as AriaLinkProps,
} from "react-aria-components";

export type ButtonVariant = "primary" | "secondary" | "quiet" | "destructive";
export type ButtonSize = "comfortable" | "compact";

export type ButtonProps = Omit<AriaButtonProps, "children" | "className" | "style" | "isDisabled"> & {
  variant?: ButtonVariant;
  size?: ButtonSize;
  isDisabled?: boolean;
  isLoading?: boolean;
  children: ReactNode;
};

export function Button({
  variant = "secondary",
  size = "comfortable",
  isDisabled = false,
  isLoading = false,
  children,
  ...props
}: ButtonProps) {
  const progressID = useId();
  return <>
    <AriaButton
      {...props}
      isDisabled={isDisabled || isLoading}
      aria-describedby={isLoading ? progressID : props["aria-describedby"]}
      className={`cs-button cs-button--${variant} cs-button--${size}`}
    >
      {isLoading && <span className="cs-button__spinner" aria-hidden="true"/>}
      <span>{children}</span>
    </AriaButton>
    {isLoading && <span id={progressID} className="cs-sr-only" role="status" aria-label={`${textLabel(children)} in progress`}/>} 
  </>;
}

export type IconButtonProps = Omit<ButtonProps, "children"> & {
  "aria-label": string;
  children: ReactNode;
};

export function IconButton({ children, variant = "secondary", ...props }: IconButtonProps) {
  return <AriaButton {...props} className={`cs-icon-button cs-button--${variant}`}>{children}</AriaButton>;
}

export type ActionLinkProps = Omit<AriaLinkProps, "children" | "className" | "style"> & {
  children: ReactNode;
};

export function ActionLink({ children, ...props }: ActionLinkProps) {
  return <AriaLink {...props} className="cs-action-link">{children}</AriaLink>;
}

function textLabel(children: ReactNode) {
  return typeof children === "string" || typeof children === "number" ? String(children) : "Action";
}
