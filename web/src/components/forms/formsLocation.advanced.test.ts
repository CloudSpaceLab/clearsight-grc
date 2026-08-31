import { beforeEach, describe, expect, it } from "vitest";
import type { FormTemplateQuery } from "../../formsTypes";
import { readFormsQuery, writeFormsLocation } from "./formsLocation";

beforeEach(() => {
  window.history.replaceState(null, "", "#forms");
});

describe("advanced Forms location state", () => {
  it("round-trips a valid bounded expression through the canonical hash", () => {
    const query: FormTemplateQuery = {
      search: "vendor",
      filter: {
        kind: "group",
        operator: "or",
        children: [
          { kind: "condition", field: "status", operator: "is", value: "ACTIVE" },
          { kind: "condition", field: "tag", operator: "is", value: "third-party" },
        ],
      },
      limit: 50,
    };

    writeFormsLocation(query);
    const restored = readFormsQuery(window.location.hash);
    expect(restored).toEqual(query);
  });

  it("drops malformed or unsupported expression state instead of sending it to the server", () => {
    window.history.replaceState(null, "", "#forms?filter=%7B%22kind%22%3A%22condition%22%2C%22field%22%3A%22reviewer%22%2C%22operator%22%3A%22is%22%2C%22value%22%3A%22person-a%22%7D");
    expect(readFormsQuery(window.location.hash).filter).toBeUndefined();
  });
});
