import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { FocusedSheet, SelectField, TextArea } from "./index";

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
    expect(document.body.style.paddingRight).not.toBe(`${window.innerWidth}px`);
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

  it("supports a closed wide composition for multi-column focused work", () => {
    render(<FocusedSheet label="Create a form" size="wide" onClose={() => undefined}><p>Creation choices</p></FocusedSheet>);

    const sheet = screen.getByRole("dialog", { name: "Create a form" }).closest(".cs-sheet");
    expect(sheet?.className).toContain("cs-sheet--wide");
    expect(sheet?.className).not.toContain("side-panel");
  });

  it("blocks Escape, backdrop and close controls while consequential work is running", async () => {
    const close = vi.fn();
    render(<FocusedSheet label="Sending request" isDismissable={false} onClose={close}><p>Sending…</p></FocusedSheet>);
    const dialog = screen.getByRole("dialog", { name: "Sending request" });

    fireEvent.keyDown(dialog, { key: "Escape" });
    fireEvent.mouseDown(document.querySelector(".cs-sheet__overlay") as HTMLElement);
    fireEvent.click(screen.getByRole("button", { name: "Close" }));

    expect(close).not.toHaveBeenCalled();
    expect(screen.getByRole("dialog", { name: "Sending request" })).toBeTruthy();
    expect((screen.getByRole("button", { name: "Close" }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("keeps the sheet open when a portalled select option is chosen", async () => {
    function Harness() {
      const [open, setOpen] = useState(true);
      const [owner, setOwner] = useState<string>("owner-2");
      const [reason, setReason] = useState("");
      return open ? <FocusedSheet label="Change issue owner" onClose={() => setOpen(false)}>
        <SelectField label="New issue owner" value={owner} placeholder="Choose an owner" options={[{ id: "owner-2", label: "Privacy Operations Lead" }]} onChange={(value) => setOwner(value ?? "")}/>
        <TextArea label="Reason for reassignment" value={reason} onChange={setReason}/>
      </FocusedSheet> : null;
    }

    render(<Harness/>);
    const dialog = screen.getByRole("dialog", { name: "Change issue owner" });
    fireEvent.click(screen.getByRole("button", { name: /New issue owner/i }));
    expect((await screen.findByRole("listbox")).closest('[role="dialog"]')).toBe(dialog);
    fireEvent.click(screen.getByRole("option", { name: "Privacy Operations Lead" }));
    await waitFor(() => expect(screen.queryByRole("listbox")).toBeNull());

    expect(screen.getByRole("dialog", { name: "Change issue owner" })).toBeTruthy();
    expect(screen.getByLabelText("Reason for reassignment")).toBeTruthy();
  });
});
