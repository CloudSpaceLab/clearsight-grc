import { IconButton } from "./ui";
import type { NotificationInbox } from "./useNotificationInbox";
import "../notification-center.css";

export function NotificationCenter({ inbox, onOpen, isCurrent = false }: {
  inbox: Pick<NotificationInbox, "state" | "unreadCount">;
  onOpen: () => void;
  isCurrent?: boolean;
}) {
  const unread = inbox.state === "live" ? inbox.unreadCount : undefined;
  const buttonLabel = inbox.state === "unavailable"
    ? "Notifications unavailable"
    : inbox.state === "loading" ? "Notifications loading"
      : unread !== undefined && unread > 0 ? `Notifications, ${unread} unread` : "Notifications";

  return <span className="notification-center__launcher">
    <IconButton aria-label={buttonLabel} aria-current={isCurrent ? "page" : undefined} variant="quiet" onPress={onOpen}>
      <BellGlyph/>
    </IconButton>
    {unread !== undefined && unread > 0 && <span className="notification-center__count" aria-hidden="true">{unread > 99 ? "99+" : unread}</span>}
  </span>;
}

export function safeNotificationPath(value: string) {
  return /^#(?:oversight|work|programs|risks|losses|vendors|ropa|forms|people|configure|imports|reports)(?:[/?#]|$)/.test(value) && !/[\r\n]/.test(value);
}

function BellGlyph() {
  return <svg className="compact-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9"/>
    <path d="M10 21h4"/>
  </svg>;
}
