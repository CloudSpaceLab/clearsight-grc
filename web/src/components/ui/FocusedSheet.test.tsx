import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { FocusedSheet } from "./index";

describe("FocusedSheet", () => {
  it("labels focused work, contains focus and restores the invoker", async () => {
    function Harness() {
      const [open, setOpen] = useState(false);
      return <div className="app-shell">
        <main><button type="button" onClick={() => setOpen(true)}>Review evidence</button></main>
        {open && <FocusedSheet label="Evidence review" onClose={() => setOpen(false)}>
          <button type="button">First action</button>
          <button type="button">Last action</button>
        </FocusedSheet>}
      </div>;
    }
    render(<Harness/>);
    const invoker = screen.getByRole("button", { name: "Review evidence" });
    invoker.focus();
    fireEvent.click(invoker);
    const dialog = await screen.findByRole("dialog", { name: "Evidence review" });
    expect(document.body.contains(dialog)).toBe(true);
    expect(document.body.style.overflow).toBe("hidden");
    await waitFor(() => expect(document.activeElement).toBe(screen.getByRole("button", { name: "Close" })));

    fireEvent.keyDown(dialog, { key: "Escape" });
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    await waitFor(() => expect(document.activeElement).toBe(invoker));
    expect(document.body.style.overflow).toBe("");
  });

  it("dismisses from the overlay but not from panel interaction", async () => {
    const close = vi.fn();
    render(<FocusedSheet label="Review" onClose={close}><p>Review content</p></FocusedSheet>);
    fireEvent.mouseDown(await screen.findByRole("dialog", { name: "Review" }));
    expect(close).not.toHaveBeenCalled();
    fireEvent.mouseDown(document.querySelector(".cs-sheet__overlay") as HTMLElement);
    expect(close).toHaveBeenCalledTimes(1);
  });
});
