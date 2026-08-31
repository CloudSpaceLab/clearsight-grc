import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FormCanvas } from "./FormCanvas";

describe("FormCanvas response type", () => {
  it("uses the themed select contract instead of a native select", () => {
    render(<FormCanvas
      draft={{ name: "Vendor review", purpose: "Collect evidence", code: "VENDOR", presentation: "AUTOMATIC", allowModeSwitch: false, scoringMode: "NONE", sections: [{ id: "section-1", title: "Identity" }], fields: [{ id: "field-1", section_id: "section-1", label: "Legal name", type: "short_text", required: true }] }}
      selection={{ kind: "field", fieldID: "field-1" }}
      onPatch={vi.fn()} onSectionsChange={vi.fn()} onFieldChange={vi.fn()} onFieldTypeChange={vi.fn()} onSelect={vi.fn()} onAddField={vi.fn()} onAddSection={vi.fn()} onMoveField={vi.fn()} onReorderField={vi.fn()} onDuplicateField={vi.fn()} onRemoveField={vi.fn()}
    />);

    expect(screen.getByRole("button", { name: /Response type/ }).classList.contains("cs-select-field__trigger")).toBe(true);
    expect(document.querySelector("select[aria-label='Response type']")).toBeNull();
  });
});
