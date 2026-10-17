import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SelectableRecord } from "./SelectableRecord";
import { StatusBadge } from "./StatusBadge";

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

  it("keeps trailing record status visible and selection intact", () => {
    const onPress = vi.fn();
    render(<SelectableRecord title="Published edition" metadata="GLOBAL · 2026"
      description="Publisher reference" endContent={<StatusBadge tone="neutral">Citations only</StatusBadge>}
      isSelected onPress={onPress}/>);
    const record = screen.getByRole("button", {name: /Published edition/});
    expect(record.getAttribute("data-has-end-content")).toBe("true");
    expect(screen.getByText("Citations only")).toBeTruthy();
    expect(record.getAttribute("aria-pressed")).toBe("true");
    fireEvent.click(record);
    expect(onPress).toHaveBeenCalledOnce();
  });

  it("arranges an optional leading mark and trailing status without changing record selection", () => {
    const onPress = vi.fn();
    render(<SelectableRecord title="ISO/IEC 27001" metadata="Global · Edition 2022"
      startContent={<span>ISO</span>} endContent={<StatusBadge tone="neutral">Citations only</StatusBadge>}
      isSelected onPress={onPress}/>);
    const button = screen.getByRole("button", {name: /ISO\/IEC 27001/});
    expect(button.getAttribute("data-has-start-content")).toBe("true");
    expect(button.getAttribute("data-has-end-content")).toBe("true");
    expect(screen.getByText("ISO")).toBeTruthy();
    expect(screen.getByText("Citations only")).toBeTruthy();
    expect(button.getAttribute("aria-pressed")).toBe("true");
    fireEvent.click(button);
    expect(onPress).toHaveBeenCalledOnce();
  });

  it("supports a concise accessible name without hiding visible context", () => {
    render(<SelectableRecord title="Revision 2" metadata="Current" aria-label="Review revision 2, current" onPress={() => undefined}/>);

    expect(screen.getByRole("button", { name: "Review revision 2, current" }).getAttribute("aria-pressed")).toBe("false");
    expect(screen.getByText("Revision 2")).toBeTruthy();
  });
});
