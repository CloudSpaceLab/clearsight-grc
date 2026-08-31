import { describe, expect, it } from "vitest";
import { accessPolicyLabel, distributionDueStateLabel, distributionStatusLabel, distributionStatusTone } from "./distributionPresentation";

describe("distribution presentation", () => {
  it("translates every API state into working language", () => {
    expect(distributionStatusLabel).toEqual({ DRAFT: "Draft", READY: "Ready to send", OPEN: "Responses open", LOCKED: "Responses locked", COMPLETED: "Completed", EXPIRED: "Expired", REVOKED: "Access revoked", SUPERSEDED: "Replaced" });
    expect(distributionDueStateLabel).toEqual({ OPEN: "Due later", OVERDUE: "Overdue", CLOSED: "Closed" });
    expect(accessPolicyLabel).toEqual({ DIRECT_MAGIC_LINK: "Direct secure link", SHARED_LINK_EMAIL_OTP: "Shared link with email code", DIRECT_LINK_EMAIL_OTP: "Direct link with email code" });
    expect(distributionStatusTone.OPEN).toBe("info");
    expect(distributionStatusTone.REVOKED).toBe("error");
  });
});
