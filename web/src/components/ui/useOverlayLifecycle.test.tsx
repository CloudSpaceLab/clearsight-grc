import { act, render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useOverlayLifecycle } from "./useOverlayLifecycle";

function Overlay() { useOverlayLifecycle(); return <button>Overlay action</button>; }

describe("shared overlay lifecycle", () => {
  it("keeps scrolling locked until the last overlay closes, including out-of-order removal", async () => {
    document.body.style.overflow = "auto";
    document.body.style.paddingRight = "12px";
    const first = render(<Overlay/>);
    const second = render(<Overlay/>);
    first.unmount();
    expect(document.body.style.overflow).toBe("hidden");
    second.unmount();
    await act(async () => {});
    expect(document.body.style.overflow).toBe("auto");
    expect(document.body.style.paddingRight).toBe("12px");
    document.body.style.overflow = "";
    document.body.style.paddingRight = "";
  });

  it("does not restore focus to an invoker made inert while open", async () => {
    const host = document.createElement("div");
    const invoker = document.createElement("button");
    host.append(invoker); document.body.append(host); invoker.focus();
    const overlay = render(<Overlay/>);
    const action = overlay.getByRole("button", { name: "Overlay action" }); action.focus();
    host.setAttribute("inert", "");
    overlay.unmount();
    await act(async () => {});
    expect(document.activeElement).not.toBe(invoker);
    host.remove();
  });
});
