import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { FormCanvas } from "./FormCanvas";

describe("FormCanvas response type", () => {
  it("uses the themed select contract instead of a native select", () => {
    render(<FormCanvas
      draft={{ name: "Vendor review", purpose: "Collect evidence", code: "VENDOR", presentation: "AUTOMATIC", allowModeSwitch: false, scoringMode: "NONE", sections: [{ id: "section-1", title: "Identity" }], fields: [{ id: "field-1", section_id: "section-1", label: "Legal name", type: "short_text", required: true }] }}
      selection={{ kind: "field", fieldID: "field-1" }}
      onPatch={vi.fn()} onSectionsChange={vi.fn()} onFieldChange={vi.fn()} onFieldTypeChange={vi.fn()} onSelect={vi.fn()} onAddField={vi.fn()} onAddSection={vi.fn()} onMoveField={vi.fn()} onReorderField={vi.fn()} onDuplicateField={vi.fn()} onRemoveField={vi.fn()}
    />);

    expect(screen.getByRole("region", { name: "Form canvas" })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Response type/ }).classList.contains("cs-select-field__trigger")).toBe(true);
    expect(document.querySelector("select[aria-label='Response type']")).toBeNull();
  });

  it("opens the response type list without the question-card click closing it", async () => {
    const changeType = vi.fn();
    const selectQuestion = vi.fn();
    function Harness() {
      const [selection, setSelection] = useState({ kind: "field" as const, fieldID: "field-1" });
      return <FormCanvas
        draft={{ name: "Vendor review", purpose: "Collect evidence", code: "VENDOR", presentation: "AUTOMATIC", allowModeSwitch: false, scoringMode: "NONE", sections: [{ id: "section-1", title: "Identity" }], fields: [{ id: "field-1", section_id: "section-1", label: "Legal name", type: "short_text", required: true }] }}
        selection={selection}
        onPatch={vi.fn()} onSectionsChange={vi.fn()} onFieldChange={vi.fn()} onFieldTypeChange={changeType} onSelect={(next) => { selectQuestion(next); setSelection(next as typeof selection); }} onAddField={vi.fn()} onAddSection={vi.fn()} onMoveField={vi.fn()} onReorderField={vi.fn()} onDuplicateField={vi.fn()} onRemoveField={vi.fn()}
      />;
    }
    render(<Harness/>);

    fireEvent.click(screen.getByRole("button", { name: /Short answer.*Response type/ }));
    expect(await screen.findByRole("listbox")).toBeTruthy();
    expect(selectQuestion).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("option", { name: "Long answer" }));
    expect(selectQuestion).toHaveBeenCalledTimes(1);
    expect(changeType).toHaveBeenCalledWith(0, "long_text");
  });
});
