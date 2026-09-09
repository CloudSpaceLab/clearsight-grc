import { useEffect, useRef } from "react";

// Share the body lock across nested sheets/dialogs, including out-of-order removal.
let locks = 0;
let originalOverflow = "";
let originalPadding = "";

export function useOverlayLifecycle() {
  const invoker = useRef(document.activeElement instanceof HTMLElement ? document.activeElement : null);
  useEffect(() => {
    if (locks++ === 0) {
      originalOverflow = document.body.style.overflow;
      originalPadding = document.body.style.paddingRight;
      const documentWidth = document.documentElement.clientWidth;
      const scrollbar = documentWidth > 0 ? Math.max(0, window.innerWidth - documentWidth) : 0;
      const padding = Number.parseFloat(window.getComputedStyle(document.body).paddingRight) || 0;
      document.body.style.overflow = "hidden";
      if (scrollbar > 0) document.body.style.paddingRight = `${padding + scrollbar}px`;
    }
    return () => {
      if (--locks === 0) {
        document.body.style.overflow = originalOverflow;
        document.body.style.paddingRight = originalPadding;
      }
      const target = invoker.current;
      queueMicrotask(() => {
        if (target?.isConnected && !target.closest('[inert], [aria-hidden="true"]')) target.focus();
      });
    };
  }, []);
}
