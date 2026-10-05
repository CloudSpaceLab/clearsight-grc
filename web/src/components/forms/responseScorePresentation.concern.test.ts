import { describe, expect, it } from "vitest";
import { scorePresentation } from "./responseScorePresentation";

describe("response concern score meaning", () => {
  it("uses points for risk scores without rounding across concern bands", () => {
    expect(scorePresentation({ state: "FINAL", mode: "RISK", raw_score: 74.99, band: "HIGH" })).toEqual({ value: "74.99 / 100 concern points", meaning: "High concern" });
  });

  it("preserves compliance percentage semantics", () => {
    expect(scorePresentation({ state: "FINAL", mode: "COMPLIANCE", raw_score: 88, band: "LOW" })).toEqual({ value: "88% compliance", meaning: "Meets expected level" });
  });

  it("does not confuse zero concern with missing results", () => {
    expect(scorePresentation({ state: "FINAL", mode: "RISK", raw_score: 0, band: "LOW" }).value).toBe("0 / 100 concern points");
    expect(scorePresentation({ state: "PROVISIONAL", mode: "RISK" }).value).toBe("Score unavailable");
  });

  it.each([NaN, Infinity, -Infinity, -1, 101])("rejects invalid risk score %s", (raw_score) => {
    expect(scorePresentation({ state: "FINAL", mode: "RISK", raw_score, band: "LOW" })).toEqual({ value: "Score unavailable", meaning: "Score not recorded" });
  });
});
