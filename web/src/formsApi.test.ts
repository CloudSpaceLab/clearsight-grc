import { beforeEach, describe, expect, it, vi } from "vitest";
import { createLibraryFormDraft, loadFormTemplatePage, saveFormView, transitionFormTemplateRevision } from "./formsApi";

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  fetchMock.mockResolvedValue(new Response(JSON.stringify({ items: [], next_cursor: "next" }), { status: 200, headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetchMock);
});

describe("Forms API", () => {
  it("serializes only bounded defined library filters", async () => {
    await loadFormTemplatePage({ search: "vendor review", status: "ACTIVE", tag: "third-party", limit: 25 });
    const url = String(fetchMock.mock.calls[0]?.[0]);
    expect(url).toContain("/api/v1/forms/templates?");
    expect(url).toContain("search=vendor+review");
    expect(url).toContain("status=ACTIVE");
    expect(url).toContain("tag=third-party");
    expect(url).toContain("limit=25");
    expect(url).not.toContain("owner=");
    expect(url).not.toContain("tenant_id=");
  });

  it("keeps material form transitions version-pinned", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ id: "form-a", version: 2, status: "PENDING_APPROVAL" }), { status: 200, headers: { "Content-Type": "application/json" } }));
    await transitionFormTemplateRevision("form/a", 1, "PENDING_APPROVAL");
    expect(fetchMock.mock.calls[0]?.[0]).toContain("/api/v1/forms/templates/form%2Fa/transition");
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual({ expected_version: 1, to: "PENDING_APPROVAL" });
  });

  it("creates ordinary drafts and principal-owned saved filters without scope fields", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ id: "form-a", version: 1, status: "DRAFT" }), { status: 201, headers: { "Content-Type": "application/json" } }));
    await createLibraryFormDraft({
      code: "VENDOR", name: "Vendor review", purpose: "Collect evidence.",
      presentation: { default_mode: "AUTOMATIC", allow_mode_switch: true },
      sections: [{ id: "general", title: "General" }],
      fields: [{ id: "question_1", section_id: "general", label: "Question", type: "short_text", required: true }],
    });
    expect(String(fetchMock.mock.calls[0]?.[1]?.body)).not.toContain("tenant_id");
    expect(String(fetchMock.mock.calls[0]?.[1]?.body)).not.toContain("legal_entity_id");

    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ id: "view-a", name: "Vendor forms", filter: {} }), { status: 200, headers: { "Content-Type": "application/json" } }));
    await saveFormView("Vendor forms", { search: "vendor", owner: "owner-a", program: "program-a", limit: 25 });
    const body = JSON.parse(String(fetchMock.mock.calls[1]?.[1]?.body));
    expect(body.filter).toMatchObject({ search: "vendor", owner_principal_id: "owner-a", program_id: "program-a", limit: 25 });
    expect(body.filter.tenant_id).toBeUndefined();
    expect(body.filter.legal_entity_id).toBeUndefined();
  });
});
