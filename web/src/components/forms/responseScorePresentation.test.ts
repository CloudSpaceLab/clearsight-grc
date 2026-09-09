import { describe, expect, it } from "vitest";
import { scorePresentation } from "./responseScorePresentation";
import type { ResponseScore } from "../../formsDistributionApi";

describe("response score presentation", () => {
  it("separates missing score metadata from a recorded no-scoring configuration", () => {
    expect(scorePresentation().value).toBe("Score unavailable");
    expect(scorePresentation({ state: "NOT_CONFIGURED" } as ResponseScore).value).toBe("Not scored");
  });
  it("does not present a null recorded score as zero", () => {
    expect(scorePresentation({ state: "CALCULATED", mode: "RISK", raw_score: null } as unknown as ResponseScore).value).toBe("Score unavailable");
  });
});
