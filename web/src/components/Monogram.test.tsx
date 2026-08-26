import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Monogram, initials } from "./Monogram";

describe("Monogram", () => {
  it("uses stable first and last initials for names", () => {
    expect(initials("Acme Processing Limited")).toBe("AL");
    expect(initials("A")).toBe("A");
    expect(initials("  ")).toBe("—");
  });

  it("can be decorative when an adjacent label names the subject", () => {
    render(<Monogram name="Acme Processing Limited" decorative/>);
    const monogram = screen.getByText("AL");
    expect(monogram.getAttribute("aria-hidden")).toBe("true");
  });
});
