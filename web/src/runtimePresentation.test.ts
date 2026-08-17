import { describe, expect, it } from "vitest";
import { runtimePresentation } from "./runtimePresentation";

describe("runtimePresentation", () => {
  it.each(["", "?demo=1", "?demo=false", "?demo="])("keeps demo presentation for %s", (search) => {
    expect(runtimePresentation(search)).toBe("demo");
  });

  it("selects live preview only for the exact demo=0 value", () => {
    expect(runtimePresentation("?tour=off&demo=0")).toBe("live-preview");
  });
});
