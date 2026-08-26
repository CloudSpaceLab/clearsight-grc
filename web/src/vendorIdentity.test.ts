import { describe, expect, it } from "vitest";
import { validateWebsiteDomain } from "./vendorIdentity";

describe("validateWebsiteDomain", () => {
  it("accepts a DNS hostname and leaves an empty optional value valid", () => {
    expect(validateWebsiteDomain("Vendor.Example")).toBeUndefined();
    expect(validateWebsiteDomain("intranet")).toBeUndefined();
    expect(validateWebsiteDomain("2130706433.example")).toBeUndefined();
    expect(validateWebsiteDomain("127.1.vendor.example")).toBeUndefined();
    expect(validateWebsiteDomain("")).toBeUndefined();
  });

  it.each([
    "https://vendor.example",
    "vendor.example/path",
    "user@vendor.example",
    "vendor.example:443",
    "127.0.0.1",
    "127.1",
    "2130706433",
    "0177.0.0.1",
    "0x7f000001",
    "[2001:db8::1]",
    "-vendor.example",
    "vendor_.example",
    "vendor..example",
  ])("rejects non-hostname website input %s", (value) => {
    expect(validateWebsiteDomain(value)).toMatch(/hostname only/i);
  });

  it("rejects an overlong hostname", () => {
    expect(validateWebsiteDomain(`${"a".repeat(63)}.${"b".repeat(63)}.${"c".repeat(63)}.${"d".repeat(62)}.example`)).toMatch(/253 characters/i);
  });
});
