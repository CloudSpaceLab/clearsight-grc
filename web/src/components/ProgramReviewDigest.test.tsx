import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../http";
import type { ProgramReviewDigest as Digest } from "../programReviewApi";
import { acceptProgramReview, loadProgramReviewDigest } from "../programReviewApi";
import type { ProgramAggregate } from "../types";
import { ProgramReviewDigest } from "./ProgramReviewDigest";

vi.mock("../programReviewApi", async () => {
  const actual = await vi.importActual<typeof import("../programReviewApi")>("../programReviewApi");
  return { ...actual, loadProgramReviewDigest: vi.fn(), acceptProgramReview: vi.fn() };
});

const aggregate: ProgramAggregate = {
  state_label: "Evidence incomplete",
  program: {
    id: "program-1", tenant_id: "bank-1", code: "AML", name: "AML Programme", type: "REGULATORY", status: "ACTIVE", owning_function: "Compliance",
    scope: {}, effective_from: "2026-01-01T00:00:00Z", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-08-09T10:00:00Z", version: 12,
  },
  requirements: [], applicability: [], control_objectives: [], control_implementations: [], requirement_control_links: [], evidence_contracts: [], evidence_assessments: [], triggers: [],
  current_state: {
    id: "state-3", overall: "EVIDENCE_INSUFFICIENT", dimensions: { evidence_sufficiency: "EVIDENCE_INSUFFICIENT" },
    reasons: [{ code: "EVIDENCE_EXPIRED", summary: "Annual evidence is out of date.", object_type: "EVIDENCE_CONTRACT", object_id: "contract-1" }],
    open_matter_count: 2, generated_at: "2026-08-09T10:00:00Z", program_version: 12, projection_version: 3,
  },
};

const changed: Digest = {
  program_id: "program-1", state: "CHANGED", review_required: true,
  checkpoint: { id: "review-1", tenant_id: "bank-1", program_id: "program-1", principal_id: "actor-1", program_version: 10, projection_version: 2, accepted_at: "2026-08-05T09:00:00Z" },
  current_program_version: 12, current_projection_version: 3, current_overall: "EVIDENCE_INSUFFICIENT", baseline_overall: "CURRENT",
  open_matter_count: 2, open_matter_delta: 2,
  changes: [{ kind: "EVIDENCE", summary: "Evidence for Annual evidence was assessed as expired.", object_type: "EVIDENCE_CONTRACT", object_id: "contract-1" }],
  changes_total: 1, changes_omitted: 0, history_truncated: false,
  current_exceptions: aggregate.current_state!.reasons, current_exceptions_total: 1,
  new_exceptions: aggregate.current_state!.reasons, new_exceptions_total: 1,
  resolved_exceptions: [], resolved_exceptions_total: 0,
};

const current: Digest = {
  ...changed,
  state: "CURRENT", review_required: false,
  checkpoint: { ...changed.checkpoint!, program_version: 12, projection_version: 3, accepted_at: "2026-08-09T10:00:00Z" },
  baseline_overall: "EVIDENCE_INSUFFICIENT", open_matter_delta: 0,
  changes: [], changes_total: 0,
  new_exceptions: [], new_exceptions_total: 0,
};

beforeEach(() => vi.clearAllMocks());

describe("Program review digest", () => {
  it("shows only canonical changes and current exceptions from the actor baseline", async () => {
    vi.mocked(loadProgramReviewDigest).mockResolvedValue(changed);
    render(<ProgramReviewDigest aggregate={aggregate}/>);
    expect(await screen.findByRole("heading", { name: "1 change since your last review" })).toBeTruthy();
    expect(screen.getByText("Evidence for Annual evidence was assessed as expired.")).toBeTruthy();
    expect(screen.getByText("Annual evidence is out of date.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Mark current state reviewed" })).toBeTruthy();
  });

  it("accepts exactly the canonical Program and projection versions currently displayed", async () => {
    vi.mocked(loadProgramReviewDigest).mockResolvedValue(changed);
    vi.mocked(acceptProgramReview).mockResolvedValue(current);
    render(<ProgramReviewDigest aggregate={aggregate}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Mark current state reviewed" }));
    await waitFor(() => expect(acceptProgramReview).toHaveBeenCalledWith("program-1", 12, 3));
    expect(await screen.findByRole("heading", { name: "No changes since your review" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Mark current state reviewed" })).toBeNull();
  });

  it("states when older canonical change events are outside the bounded daily digest", async () => {
    vi.mocked(loadProgramReviewDigest).mockResolvedValue({ ...changed, history_truncated: true });
    render(<ProgramReviewDigest aggregate={aggregate}/>);
    expect(await screen.findByText(/Older Program change events are outside this bounded daily digest/)).toBeTruthy();
  });

  it("fails visibly when the Program moves before acknowledgement", async () => {
    vi.mocked(loadProgramReviewDigest).mockResolvedValue(changed);
    vi.mocked(acceptProgramReview).mockRejectedValue(new ApiError(409, "version conflict", "version_conflict"));
    render(<ProgramReviewDigest aggregate={aggregate}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Mark current state reviewed" }));
    const alert = await screen.findByRole("alert");
    expect(alert.textContent).toContain("changed while you were reviewing");
  });

  it("degrades to current Program truth when actor review history is unavailable", async () => {
    vi.mocked(loadProgramReviewDigest).mockRejectedValue(new ApiError(503, "unavailable"));
    render(<ProgramReviewDigest aggregate={aggregate}/>);
    expect(await screen.findByText(/Review history is unavailable/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Mark current state reviewed" })).toBeNull();
  });
});
