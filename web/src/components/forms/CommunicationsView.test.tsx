import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import CommunicationsView from "./CommunicationsView";

const api = vi.hoisted(() => ({
  createCommunicationProfile: vi.fn(), createCommunicationTemplate: vi.fn(), impactCommunicationTemplate: vi.fn(),
  loadCommunicationProfiles: vi.fn(), loadCommunicationTemplates: vi.fn(), previewCommunicationTemplate: vi.fn(),
  rollbackCommunicationProfile: vi.fn(), rollbackCommunicationTemplate: vi.fn(), testSendCommunicationTemplate: vi.fn(),
  transitionCommunicationProfile: vi.fn(), transitionCommunicationTemplate: vi.fn(),
}));
vi.mock("../../formsCommunicationApi", () => api);

const profile = {
  id: "profile-1", legal_entity_id: "bank-1", version: 2, default_locale: "en", bank_name: "Clear Bank Nigeria",
  support_contact: "compliance@clearbank.example", brand_asset_id: "brand-1", status: "ACTIVE", effective_from: "2026-08-31T18:00:00Z",
  maker_id: "maker-1", created_at: "2026-08-31T18:00:00Z", updated_at: "2026-08-31T18:00:00Z",
} as const;
const template = {
  id: "template-1", legal_entity_id: "bank-1", action: "INVITATION", locale: "en", version: 3,
  subject_template: "Complete {{form_title}}", document: [{ type: "paragraph", text: "Hello {{recipient_name}}" }],
  status: "ACTIVE", effective_from: "2026-08-31T18:00:00Z", maker_id: "maker-1",
  created_at: "2026-08-31T18:00:00Z", updated_at: "2026-08-31T18:00:00Z",
} as const;

beforeEach(() => {
  for (const value of Object.values(api)) value.mockReset();
  api.loadCommunicationProfiles.mockResolvedValue([profile]);
  api.loadCommunicationTemplates.mockResolvedValue([template]);
});

describe("Communications workspace", () => {
  it("opens profile revision in a cancellable focused dialog without replacing the workspace", async () => {
    render(<CommunicationsView/>);
    fireEvent.click(await screen.findByRole("button", { name: "Create profile revision" }));
    expect(screen.getByRole("heading", { name: "Communications", hidden: true })).toBeTruthy();
    expect(screen.getByRole("dialog", { name: "Create profile revision" })).toBeTruthy();
    expect(screen.getByLabelText(/Effective from/).getAttribute("type")).toBe("datetime-local");
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.queryByRole("dialog", { name: "Create profile revision" })).toBeNull();
  });

  it("opens template revision in a wide focused dialog and preserves the selected template", async () => {
    render(<CommunicationsView/>);
    expect((await screen.findByRole("button", { name: /Invitation.*v3/ })).getAttribute("aria-pressed")).toBe("true");
    fireEvent.click(screen.getByRole("button", { name: "Create template revision" }));
    expect(screen.getByRole("heading", { name: "Communications", hidden: true })).toBeTruthy();
    expect(screen.getByRole("dialog", { name: "Create template revision" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeTruthy();
  });
});
