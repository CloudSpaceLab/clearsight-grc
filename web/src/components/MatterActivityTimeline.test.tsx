import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MatterActivityTimeline } from "./MatterActivityTimeline";

const api = vi.hoisted(() => ({ loadMatterActivity: vi.fn(), addMatterComment: vi.fn() }));
vi.mock("../matterCollaborationApi", () => api);

describe("Matter activity timeline", () => {
  it("loads older activity without replacing the current entries", async () => {
    api.loadMatterActivity.mockResolvedValueOnce({ items: [{ event_id: "3", event_type: "MATTER_COMMENT_ADDED", occurred_at: "2026-09-10T10:00:00Z", actor_id: "hakeem", actor_type: "PERSON", matter_version: 3, comment: { body: "Latest update" } }], next_before_version: 3 })
      .mockResolvedValueOnce({ items: [{ event_id: "2", event_type: "MATTER_COMMENT_ADDED", occurred_at: "2026-09-10T09:00:00Z", actor_id: "blessing", actor_type: "PERSON", matter_version: 2, comment: { body: "Earlier update" } }] });
    render(<MatterActivityTimeline matterID="matter-1" matterVersion={3} candidates={[]}/>);
    expect(await screen.findByText("Latest update")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Load older activity" }));
    expect(await screen.findByText("Earlier update")).toBeTruthy();
    expect(screen.getByText("Latest update")).toBeTruthy();
  });
});
