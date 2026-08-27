import { describe, expect, it } from "vitest";
import { normalizeWebsiteDomain, validateWebsiteDomain } from "./vendorIdentity";

describe("validateWebsiteDomain", () => {
  it("accepts a DNS hostname or safe HTTPS URL and leaves an empty optional value valid", () => {
    expect(validateWebsiteDomain("Vendor.Example")).toBeUndefined();
    expect(validateWebsiteDomain("intranet")).toBeUndefined();
    expect(validateWebsiteDomain("2130706433.example")).toBeUndefined();
    expect(validateWebsiteDomain("127.1.vendor.example")).toBeUndefined();
    expect(validateWebsiteDomain("BÜCHER.Example")).toBeUndefined();
    expect(validateWebsiteDomain("")).toBeUndefined();
    expect(normalizeWebsiteDomain(" BÜCHER.Example ")).toBe("xn--bcher-kva.example");
    expect(validateWebsiteDomain("https://Vendor.Example/about?source=register#company")).toBeUndefined();
    expect(normalizeWebsiteDomain("https://Vendor.Example/about?source=register#company")).toBe("vendor.example");
  });

  it.each([
    "http://vendor.example",
    "https://user@vendor.example",
    "https://vendor.example:443/about",
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
  ])("rejects unsafe website input %s", (value) => {
    expect(validateWebsiteDomain(value)).toMatch(/website hostname or full HTTPS URL/i);
  });

  it("rejects an overlong hostname", () => {
    expect(validateWebsiteDomain(`${"a".repeat(63)}.${"b".repeat(63)}.${"c".repeat(63)}.${"d".repeat(62)}.example`)).toMatch(/253 characters/i);
  });
});
