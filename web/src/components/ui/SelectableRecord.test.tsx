import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SelectableRecord } from "./SelectableRecord";

describe("SelectableRecord", () => {
  it("announces selection and preserves the record hierarchy", () => {
    const onPress = vi.fn();
    render(<SelectableRecord
      title="Acme annual vendor review"
      metadata="Open · 26 Sep 2027"
      description="2 submitted versions"
      isSelected
      onPress={onPress}
    />);

    const record = screen.getByRole("button", { name: /Acme annual vendor review/ });
    expect(record.getAttribute("aria-pressed")).toBe("true");
    expect(record.getAttribute("data-selected")).toBe("true");
    expect(screen.getByText("Open · 26 Sep 2027")).toBeTruthy();
    expect(screen.getByText("2 submitted versions")).toBeTruthy();
    fireEvent.click(record);
    expect(onPress).toHaveBeenCalledOnce();
  });

  it("supports a concise accessible name without hiding visible context", () => {
    render(<SelectableRecord title="Revision 2" metadata="Current" aria-label="Review revision 2, current" onPress={() => undefined}/>);

    expect(screen.getByRole("button", { name: "Review revision 2, current" }).getAttribute("aria-pressed")).toBe("false");
    expect(screen.getByText("Revision 2")).toBeTruthy();
  });
});
