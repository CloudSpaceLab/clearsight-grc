import { describe, expect, it } from "vitest";
import { formatLossMoneyCompact, formatLossMoneyExact, lossCurrencyFractionDigits, lossMajorToMinor } from "./lossPresentation";

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

  it("formats exact minor units beyond Number.MAX_SAFE_INTEGER without losing digits", () => {
    const formatted = formatLossMoneyExact("9223372036854775807", "USD");
    expect(formatted.replace(/\D/g, "")).toBe("9223372036854775807");
  });

  it("preserves signed money flows", () => {
    const formatted = formatLossMoneyExact("-450", "NGN");
    expect(formatted).toContain("-");
    expect(formatted.replace(/\D/g, "")).toBe("450");
  });

  it("formats rounded summary amounts with the currency's minor-unit precision", () => {
    expect(formatLossMoneyCompact("1040000000", "NGN", "en-NG")).toBe("₦10.4M");
    expect(formatLossMoneyCompact("10400000", "JPY", "en-US")).toBe("¥10.4M");
    expect(formatLossMoneyCompact("10400000000", "BHD", "en-US")).toMatch(/10\.4M$/);
    expect(formatLossMoneyCompact("-1040000000", "NGN", "en-NG")).toBe("-₦10.4M");
  });

  it("preserves exact small amounts and locale-specific noncompact values", () => {
    expect(formatLossMoneyCompact("-45", "NGN", "en-NG")).toBe("-₦0.45");
    expect(formatLossMoneyCompact("123", "BHD", "en-US")).toMatch(/0\.123$/);
    expect(formatLossMoneyCompact("0", "JPY", "en-US")).toBe("¥0");
    expect(formatLossMoneyCompact("123456", "EUR", "de-DE")).toBe(formatLossMoneyExact("123456", "EUR", "de-DE"));
  });

  it("rounds summary boundaries without changing the exact stored amount", () => {
    expect(formatLossMoneyCompact("104999", "NGN", "en-NG")).toBe("₦1K");
    expect(formatLossMoneyCompact("105000", "NGN", "en-NG")).toBe("₦1.1K");
    expect(formatLossMoneyCompact("104999999", "NGN", "en-NG")).toBe("₦1M");
    expect(formatLossMoneyCompact("105000000", "NGN", "en-NG")).toBe("₦1.1M");
    expect(formatLossMoneyCompact("99995000", "NGN", "en-NG")).toBe("₦1M");
    expect(formatLossMoneyExact("104999", "NGN", "en-NG")).toBe("₦1,049.99");
  });

  it("keeps large integer precision and honest malformed-value fallbacks", () => {
    expect(formatLossMoneyCompact(9007199254740993123456789012345n, "USD", "en-US")).toBe("$90,071,992,547,409,931.2T");
    expect(formatLossMoneyCompact("not-an-amount", "USD", "en-US")).toBe("not-an-amount USD");
    expect(formatLossMoneyCompact("100", "N", "en-US")).toBe("100 N");
  });
});
