import { beforeEach, describe, expect, it, vi } from "vitest";
import { createCommunicationTemplate, impactCommunicationTemplate, loadCommunicationTemplates, previewCommunicationTemplate, testSendCommunicationTemplate, transitionCommunicationTemplate } from "./formsCommunicationApi";

const fetchMock = vi.fn();
beforeEach(() => { fetchMock.mockReset(); vi.stubGlobal("fetch", fetchMock); });

const template = { id: "comm-a", legal_entity_id: "entity-a", action: "INVITATION" as const, locale: "en", version: 3, subject_template: "{{form_title}}", document: [{ type: "primary-action", text: "Open", href: "{{secure_form_link}}" }], status: "DRAFT" as const, effective_from: "2026-08-28T10:00:00Z", maker_id: "maker", created_at: "2026-08-28T10:00:00Z", updated_at: "2026-08-28T10:00:00Z" };

describe("forms communication API", () => {
  it("uses server-backed template filters", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ items: [template] }), { status: 200, headers: { "Content-Type": "application/json" } }));
    await expect(loadCommunicationTemplates({ action: "INVITATION", locale: "en", status: "DRAFT" })).resolves.toEqual([template]);
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain("/api/v1/forms/communications/templates?action=INVITATION&locale=en&status=DRAFT");
  });

  it("creates immutable template revisions using the server document contract", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify(template), { status: 201, headers: { "Content-Type": "application/json" } }));
    await createCommunicationTemplate({ action: "INVITATION", locale: "en", subject_template: "{{form_title}}", document: [{ type: "primary-action", text: "Open", href: "{{secure_form_link}}" }], effective_from: "2026-08-28T10:00:00Z" });
    const body = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body));
    expect(body).toMatchObject({ action: "INVITATION", locale: "en", document: [{ type: "primary-action", href: "{{secure_form_link}}" }] });
    expect(body.id).toBeUndefined();
    expect(body.version).toBeUndefined();
  });

  it("never fabricates preview or impact locally", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ subject: "Rendered", plain_text: "Server copy", html: "<p>Server copy</p>" }), { status: 200, headers: { "Content-Type": "application/json" } }));
    await expect(previewCommunicationTemplate(template)).resolves.toMatchObject({ subject: "Rendered", plain_text: "Server copy" });
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain("/INVITATION/en/revisions/3/preview");
    expect(fetchMock.mock.calls[0]?.[1]?.method).toBe("POST");

    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ action: "INVITATION", locale: "en", candidate_version: 3, subject_changed: true, document_changed: false, effective_window_changed: false }), { status: 200, headers: { "Content-Type": "application/json" } }));
    await impactCommunicationTemplate(template);
    expect(String(fetchMock.mock.calls[1]?.[0])).toContain("/INVITATION/en/revisions/3/impact");
  });

  it("keeps test-send and lifecycle changes on governed server endpoints", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ accepted: true }), { status: 200, headers: { "Content-Type": "application/json" } }));
    await testSendCommunicationTemplate(template, "test@example.com");
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual({ address: "test@example.com" });

    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ ...template, status: "PENDING_APPROVAL" }), { status: 200, headers: { "Content-Type": "application/json" } }));
    await transitionCommunicationTemplate(template, "PENDING_APPROVAL");
    expect(JSON.parse(String(fetchMock.mock.calls[1]?.[1]?.body))).toEqual({ expected_version: 3, to: "PENDING_APPROVAL" });
  });
});
