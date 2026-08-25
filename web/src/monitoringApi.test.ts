import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadContext } from "./api";
import { requestJSON } from "./http";
import { createFormTemplate } from "./monitoringApi";

vi.mock("./api", () => ({ loadContext: vi.fn() }));
vi.mock("./http", () => ({ requestJSON: vi.fn() }));

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadContext).mockResolvedValue({ tenant: { id: "bank-1" } } as Awaited<ReturnType<typeof loadContext>>);
});

describe("monitoring API", () => {
  it("sends presentation, sections and typed fields when a form draft is created", async () => {
    vi.mocked(requestJSON).mockResolvedValue({ id: "form-1" });
    const input = {
      code: "VENDOR-DUE-DILIGENCE",
      name: "Vendor due diligence",
      purpose: "Collect information required for the vendor review.",
      presentation: { default_mode: "AUTOMATIC" as const, allow_mode_switch: true },
      sections: [{ id: "profile", title: "Vendor profile" }],
      fields: [{ id: "contact", section_id: "profile", label: "Primary contact email", type: "email" as const, required: true }],
    };

    await createFormTemplate(input);

    expect(requestJSON).toHaveBeenCalledWith("", "/api/v1/form-templates?tenant_id=bank-1", {
      method: "POST",
      body: JSON.stringify(input),
    });
  });
});
