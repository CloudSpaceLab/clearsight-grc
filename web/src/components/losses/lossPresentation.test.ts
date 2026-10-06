import { describe, expect, it } from "vitest";
import { lossCurrencyFractionDigits, lossMajorToMinor } from "./lossPresentation";

describe("loss amount presentation", () => {
  it("converts decimal major units to exact integer minor units", () => {
    expect(lossMajorToMinor("1234.56", "NGN")).toBe(123456);
    expect(lossMajorToMinor("1.2", "USD")).toBe(120);
    expect(lossMajorToMinor("100", "JPY")).toBe(100);
  });

  it("rejects ambiguous, over-precise and unsafe ledger amounts", () => {
    expect(lossMajorToMinor("1.001", "NGN")).toBeUndefined();
    expect(lossMajorToMinor("-10", "NGN")).toBeUndefined();
    expect(lossMajorToMinor("1.1", "JPY")).toBeUndefined();
    expect(lossMajorToMinor("9007199254740992", "JPY")).toBeUndefined();
    expect(lossMajorToMinor("10", "N")).toBeUndefined();
  });

  it("uses the currency's declared minor-unit precision", () => {
    expect(lossCurrencyFractionDigits("ngn")).toBe(2);
    expect(lossCurrencyFractionDigits("JPY")).toBe(0);
  });
});
