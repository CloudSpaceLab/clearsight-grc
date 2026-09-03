import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CollectionPolicyForm } from "./CollectionPolicyForm";

describe("collection policy form", () => {
  it("uses the recommended schedule and explains the result", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(<CollectionPolicyForm onSave={onSave} onCancel={vi.fn()}/>);

    expect((screen.getByLabelText("Response expires after") as HTMLInputElement).value).toBe("12");
    expect((screen.getByLabelText("Renewal starts") as HTMLInputElement).value).toBe("30");
    expect((screen.getByLabelText("Reminders during renewal") as HTMLInputElement).value).toBe("3");
    expect(screen.getByText("Responses will be renewed 30 days before they reach 12 months old. The initial request is followed by up to 3 reminders.")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Add collection to Program" }));
    await waitFor(() => expect(onSave).toHaveBeenCalledWith({ validity_months: 12, renewal_window_days: 30, reminder_count: 3 }));
  });

  it("rejects a renewal period that reaches the response expiry", async () => {
    const onSave = vi.fn();
    render(<CollectionPolicyForm onSave={onSave} onCancel={vi.fn()}/>);
    fireEvent.change(screen.getByLabelText("Response expires after"), { target: { value: "1" } });
    fireEvent.change(screen.getByLabelText("Renewal starts"), { target: { value: "30" } });
    fireEvent.click(screen.getByRole("button", { name: "Add collection to Program" }));

    expect((await screen.findByRole("alert")).textContent).toContain("Renewal must start before the response can expire.");
    expect(onSave).not.toHaveBeenCalled();
  });

  it("keeps entered values when saving fails", async () => {
    const onSave = vi.fn().mockRejectedValue(new Error("This record changed. Reload it before saving your update."));
    render(<CollectionPolicyForm onSave={onSave} onCancel={vi.fn()}/>);
    fireEvent.change(screen.getByLabelText("Response expires after"), { target: { value: "18" } });
    fireEvent.change(screen.getByLabelText("Reminders during renewal"), { target: { value: "5" } });
    fireEvent.click(screen.getByRole("button", { name: "Add collection to Program" }));

    expect((await screen.findByRole("alert")).textContent).toContain("This record changed");
    expect((screen.getByLabelText("Response expires after") as HTMLInputElement).value).toBe("18");
    expect((screen.getByLabelText("Reminders during renewal") as HTMLInputElement).value).toBe("5");
  });
});
