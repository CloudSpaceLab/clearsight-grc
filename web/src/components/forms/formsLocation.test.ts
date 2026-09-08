import { beforeEach, describe, expect, it, vi } from "vitest";
import { readFormsQuery, readFormsSection, writeFormsLocation } from "./formsLocation";

beforeEach(() => window.history.replaceState(null, "", "#forms"));

describe("Forms section locations", () => {
  it.each([
    ["templates", "Templates"], ["sent-forms", "Sent forms"], ["responses", "Responses"],
    ["documents", "Documents"], ["policies", "Policies"], ["imports", "Imports"], ["communications", "Communications"],
  ] as const)("round-trips only the %s section slug", (slug, tab) => {
    expect(readFormsSection(`#forms/template-a?section=${slug}`)).toBe(tab);
    writeFormsLocation({ search: "vendor", limit: 25 }, "template-a", false, tab);
    expect(window.location.hash).toBe(`#forms/template-a?search=vendor${tab === "Templates" ? "" : `&section=${slug}`}`);
    expect(readFormsSection(window.location.hash)).toBe(tab);
    expect(readFormsQuery(window.location.hash)).toEqual({ search: "vendor", limit: 25 });
  });

  it.each(["#forms", "#forms/documents", "#forms?section=", "#forms?section=Documents", "#forms?section=unknown", "#forms?section=toString", "#forms?section=__proto__"])(
    "defaults an absent or unsupported section to Templates in %s", (hash) => {
      expect(readFormsSection(hash)).toBe("Templates");
    },
  );

  it("keeps legacy template IDs and exact filter URLs with replace as the default", () => {
    const push = vi.spyOn(window.history, "pushState");
    const replace = vi.spyOn(window.history, "replaceState");
    try {
      writeFormsLocation({ search: "outsourcing", status: "ACTIVE", tag: "third-party" }, "template-a");
      expect(window.location.hash).toBe("#forms/template-a?search=outsourcing&status=ACTIVE&tag=third-party");
      writeFormsLocation({}, "documents");
      expect(window.location.hash).toBe("#forms/documents");
      expect(readFormsSection(window.location.hash)).toBe("Templates");
      writeFormsLocation({}, "template/a?b");
      expect(window.location.hash).toBe("#forms/template%2Fa%3Fb");
      expect(replace).toHaveBeenCalledTimes(3);
      expect(push).not.toHaveBeenCalled();
    } finally {
      vi.restoreAllMocks();
    }
  });
});
