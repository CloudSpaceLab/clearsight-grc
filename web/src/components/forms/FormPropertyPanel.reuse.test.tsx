import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { FormTemplate } from "../../monitoringTypes";
import { FormPropertyPanel } from "./FormPropertyPanel";

function template(id: string, version: number, status: FormTemplate["status"], sectionID: string, title: string): FormTemplate {
  return {
    id,
    tenant_id: "bank-1",
    code: id.toUpperCase(),
    name: title,
    purpose: "Reusable governed section.",
    scoring_mode: "NONE",
    presentation: { default_mode: "CLASSIC", allow_mode_switch: false },
    sections: [{ id: sectionID, title }],
    fields: [{ id: `${sectionID}-q`, section_id: sectionID, label: "Question", type: "short_text", required: true }],
    status,
    is_current: status === "ACTIVE",
    version,
    created_at: "2026-08-27T00:00:00Z",
    updated_at: "2026-08-27T00:00:00Z",
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
}

function renderPanel(loadReusableTemplate: (id: string, version: number) => Promise<FormTemplate>) {
  return render(<FormPropertyPanel
    scoringMode="NONE"
    sections={[{ id: "section_1", title: "Questions" }]}
    fields={[{ id: "question_1", section_id: "section_1", label: "Existing", type: "short_text", required: true }]}
    reusableTemplates={[
      { id: "template-a", code: "A", name: "Template A", version: 2 },
      { id: "template-b", code: "B", name: "Template B", version: 4 },
    ]}
    loadReusableTemplate={loadReusableTemplate}
    onSectionsChange={vi.fn()}
    onFieldChange={vi.fn()}
    onFieldTypeChange={vi.fn()}
    onFieldConstraint={vi.fn()}
    onFieldScoringToggle={vi.fn()}
    onAddSection={vi.fn()}
    onDuplicateSection={vi.fn()}
    onMoveSection={vi.fn()}
    onRemoveSection={vi.fn()}
    onInsertReusableSection={vi.fn()}
    onAddField={vi.fn()}
    onMoveField={vi.fn()}
    onRemoveField={vi.fn()}
  />);
}

describe("FormPropertyPanel reusable revisions", () => {
  it("ignores a slower stale template response after the user selects another revision", async () => {
    const a = deferred<FormTemplate>();
    const b = deferred<FormTemplate>();
    const load = vi.fn((id: string) => id === "template-a" ? a.promise : b.promise);
    renderPanel(load);
    fireEvent.click(screen.getByText("Insert section from active template"));
    const selector = screen.getByLabelText("Active template revision");

    fireEvent.change(selector, { target: { value: "template-a:2" } });
    fireEvent.change(selector, { target: { value: "template-b:4" } });
    b.resolve(template("template-b", 4, "ACTIVE", "b-section", "Current B"));

    const sectionSelector = await screen.findByLabelText("Section to insert");
    expect(within(sectionSelector).getByRole("option", { name: "Current B" })).toBeTruthy();

    a.resolve(template("template-a", 2, "ACTIVE", "a-section", "Stale A"));
    await waitFor(() => expect(screen.queryByRole("option", { name: "Stale A" })).toBeNull());
  });

  it("refuses a revision that is no longer active at insertion time", async () => {
    renderPanel(vi.fn().mockResolvedValue(template("template-a", 2, "RETIRED", "a-section", "Retired A")));
    fireEvent.click(screen.getByText("Insert section from active template"));
    fireEvent.change(screen.getByLabelText("Active template revision"), { target: { value: "template-a:2" } });

    expect(await screen.findByRole("alert")).toHaveTextContent("no longer an active reusable template");
    expect(screen.queryByRole("button", { name: "Insert section" })).toBeNull();
  });
});
