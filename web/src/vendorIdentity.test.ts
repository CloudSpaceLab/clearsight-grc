import { describe, expect, it } from "vitest";
import { normalizeWebsiteDomain, validateWebsiteDomain } from "./vendorIdentity";

describe("validateWebsiteDomain", () => {
  it("accepts a DNS hostname and leaves an empty optional value valid", () => {
    expect(validateWebsiteDomain("Vendor.Example")).toBeUndefined();
    expect(validateWebsiteDomain("intranet")).toBeUndefined();
    expect(validateWebsiteDomain("2130706433.example")).toBeUndefined();
    expect(validateWebsiteDomain("127.1.vendor.example")).toBeUndefined();
    expect(validateWebsiteDomain("BÜCHER.Example")).toBeUndefined();
    expect(validateWebsiteDomain("")).toBeUndefined();
    expect(normalizeWebsiteDomain(" BÜCHER.Example ")).toBe("xn--bcher-kva.example");
  });

  it.each([
    "https://vendor.example",
    "vendor.example/path",
    "vendor.example\\path",
    "vendor.example.",
    "%76endor.example",
    "vendor%2Eexample",
    "user@vendor.example",
    "vendor.example:443",
    "127.0.0.1",
    "127.1",
    "2130706433",
    "0177.0.0.1",
    "0x7f000001",
    "0x7f.0.0.0x1",
    "0300.0250.0001.0001",
    "127.0x0.01",
    "１２７．０．０．１",
    "０ｘ７ｆ．０．０．１",
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
