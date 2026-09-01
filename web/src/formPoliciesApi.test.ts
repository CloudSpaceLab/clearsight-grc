import { beforeEach, describe, expect, it, vi } from "vitest";
import { requestJSON } from "./http";
import { previewFormScore, submitFormResponsePolicy } from "./formPoliciesApi";

vi.mock("./http", () => ({ requestJSON: vi.fn() }));

beforeEach(() => vi.mocked(requestJSON).mockReset());

describe("form policy API", () => {
  it("encodes the template ID and sends the exact stored revision for score preview", async () => {
    vi.mocked(requestJSON).mockResolvedValue({});
    await previewFormScore("form/a", 7, { certified: { text: "No" } });
    expect(requestJSON).toHaveBeenCalledWith("", "/api/v1/config/form-templates/form%2Fa/score-preview", expect.objectContaining({ method: "POST", body: JSON.stringify({ form_template_version: 7, answers: { certified: { text: "No" } } }) }));
  });

  it("sends only optimistic version and simulation receipt for submission", async () => {
    vi.mocked(requestJSON).mockResolvedValue({});
    await submitFormResponsePolicy("policy/a", 3, "simulation-1");
    const [, path, init] = vi.mocked(requestJSON).mock.calls[0]!;
    expect(path).toBe("/api/v1/config/form-response-policies/policy%2Fa/submit");
    expect(JSON.parse(String(init?.body))).toEqual({ expected_version: 3, simulation_id: "simulation-1" });
  });
});
