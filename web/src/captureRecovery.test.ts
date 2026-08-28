import { describe, expect, it } from "vitest";
import { CaptureRecovery, CAPTURE_RECOVERY_MAX_AGE_MS, mergeRecoveredAnswers, recoveryExpiry, recoveryStorageKey, type CaptureRecoveryContext } from "./captureRecovery";
import { MemoryRecoveryStore } from "./captureRecoveryStore";
import type { CaptureField } from "./types";

const fields: CaptureField[] = [
  { id: "legal_name", label: "Legal name", type: "short_text", required: true },
  { id: "services", label: "Services", type: "multi_select", required: false },
  { id: "certificate", label: "Certificate", type: "file", required: false },
  { id: "signoff", label: "Sign", type: "signature", required: false },
];

function context(overrides: Partial<CaptureRecoveryContext> = {}): CaptureRecoveryContext {
  return {
    origin: "https://forms.example.test",
    legalEntityID: "entity-a",
    distributionID: "distribution-a",
    workspaceID: "workspace-a",
    schemaVersion: 7,
    serverVersion: 3,
    authorized: true,
    deadline: "2026-09-20T12:00:00.000Z",
    routeExpiresAt: "2026-09-10T12:00:00.000Z",
    ...overrides,
  };
}

describe("capture recovery", () => {
  it("never places secrets, files or signatures in the recovery envelope", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const current = context();

    await recovery.save(current, fields, {
      legal_name: { text: "Example Ltd" },
      services: { values: ["Payments"] },
      certificate: { artifact_ids: ["file-bytes"] },
      signoff: { text: "signature-data" },
    }, { now: new Date("2026-09-01T12:00:00.000Z") });

    const raw = store.readRaw(recoveryStorageKey(current));
    expect(raw).toBeDefined();
    expect(raw).not.toContain("Example Ltd");
    expect(raw).not.toContain("file-bytes");
    expect(raw).not.toContain("signature-data");
    expect(raw).not.toContain("session-token");

    const restored = await recovery.restore(current, new Date("2026-09-01T12:01:00.000Z"));
    expect(restored).toMatchObject({
      status: "restored",
      envelope: {
        answers: { legal_name: { text: "Example Ltd" }, services: { values: ["Payments"] } },
        filesToReselect: ["certificate"],
      },
    });
  });

  it("binds encrypted recovery to origin, legal entity, distribution and schema", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const source = context();
    await recovery.save(source, fields, { legal_name: "Example Ltd" }, { now: new Date("2026-09-01T12:00:00.000Z") });

    for (const changed of [
      context({ origin: "https://evil.example" }),
      context({ legalEntityID: "entity-b" }),
      context({ distributionID: "distribution-b" }),
      context({ schemaVersion: 8 }),
    ]) {
      expect(await recovery.restore(changed, new Date("2026-09-01T12:01:00.000Z"))).not.toMatchObject({ status: "restored" });
    }
    expect(await recovery.restore(source, new Date("2026-09-01T12:01:00.000Z"))).toMatchObject({ status: "restored" });
  });

  it("does not restore before the current server session authorizes the exact distribution", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const authorized = context();
    await recovery.save(authorized, fields, { legal_name: "Example Ltd" }, { now: new Date("2026-09-01T12:00:00.000Z") });

    expect(await recovery.restore(context({ authorized: false }), new Date("2026-09-01T12:01:00.000Z"))).toEqual({
      status: "denied",
      reason: "unauthorized",
    });
  });

  it("honours NO_BROWSER_CACHE and clears an existing envelope", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const enabled = context();
    await recovery.save(enabled, fields, { legal_name: "Example Ltd" }, { now: new Date("2026-09-01T12:00:00.000Z") });

    const disabled = context({ cachePolicy: "NO_BROWSER_CACHE" });
    await recovery.save(disabled, fields, { legal_name: "Changed" }, { now: new Date("2026-09-01T12:01:00.000Z") });
    expect(store.readRaw(recoveryStorageKey(enabled))).toBeUndefined();
    expect(await recovery.restore(disabled)).toEqual({ status: "denied", reason: "disabled" });
  });

  it("caps recovery at seven days, the deadline, or route expiry, whichever is earliest", () => {
    const now = new Date("2026-09-01T12:00:00.000Z");
    expect(recoveryExpiry({}, now).getTime()).toBe(now.getTime() + CAPTURE_RECOVERY_MAX_AGE_MS);
    expect(recoveryExpiry({ deadline: "2026-09-03T12:00:00.000Z" }, now).toISOString()).toBe("2026-09-03T12:00:00.000Z");
    expect(recoveryExpiry({ deadline: "2026-09-05T12:00:00.000Z", routeExpiresAt: "2026-09-02T12:00:00.000Z" }, now).toISOString()).toBe("2026-09-02T12:00:00.000Z");
  });

  it("discards expired and corrupt envelopes instead of returning untrusted data", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const current = context({ routeExpiresAt: "2026-09-02T12:00:00.000Z" });
    await recovery.save(current, fields, { legal_name: "Example Ltd" }, { now: new Date("2026-09-01T12:00:00.000Z") });
    expect(await recovery.restore(current, new Date("2026-09-03T12:00:00.000Z"))).toEqual({ status: "discarded", reason: "expired" });

    await recovery.save(current, fields, { legal_name: "Example Ltd" }, { now: new Date("2026-09-01T12:00:00.000Z") });
    const key = recoveryStorageKey(current);
    const encrypted = await store.get(key);
    if (!encrypted) throw new Error("missing test envelope");
    const tampered = new Uint8Array(encrypted.ciphertext.slice(0));
    tampered[0] ^= 0xff;
    store.replaceRaw(key, { ...encrypted, ciphertext: tampered.buffer });
    expect(await recovery.restore(current, new Date("2026-09-01T12:01:00.000Z"))).toEqual({ status: "discarded", reason: "corrupt" });
    expect(store.readRaw(key)).toBeUndefined();
  });

  it("merges non-overlapping answers and surfaces changed answers field by field", () => {
    expect(mergeRecoveredAnswers(
      { legal_name: { text: "Server Ltd" }, unchanged: { text: "same" } },
      { services: { values: ["Payments"] }, legal_name: { text: "Local Ltd" }, unchanged: "same" },
    )).toEqual({
      answers: {
        legal_name: { text: "Server Ltd" },
        unchanged: { text: "same" },
        services: { values: ["Payments"] },
      },
      conflicts: [{
        fieldID: "legal_name",
        serverValue: { text: "Server Ltd" },
        localValue: { text: "Local Ltd" },
      }],
    });
  });

  it("stores a non-exportable AES-GCM device key", async () => {
    const store = new MemoryRecoveryStore();
    const key = await store.getOrCreateDeviceKey("device");
    expect(key.algorithm).toMatchObject({ name: "AES-GCM", length: 256 });
    expect(key.extractable).toBe(false);
    await expect(crypto.subtle.exportKey("raw", key)).rejects.toBeDefined();
  });
});
