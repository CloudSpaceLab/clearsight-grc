import { describe, expect, it } from "vitest";
import { runtimePresentation } from "./runtimePresentation";

describe("runtimePresentation", () => {
  it.each(["", "?demo=0", "?demo=false", "?demo="])("keeps enterprise presentation for %s", (search) => {
    expect(runtimePresentation(search)).toBe("enterprise");
  });

  it("selects demo presentation only for the explicit demo=1 value", () => {
    expect(runtimePresentation("?tour=off&demo=1")).toBe("demo");
  });
});
