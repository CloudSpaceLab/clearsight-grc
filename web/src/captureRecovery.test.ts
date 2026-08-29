import { describe, expect, it } from "vitest";
import {
  CaptureRecovery,
  CAPTURE_RECOVERY_MAX_AGE_MS,
  rebaseRecoveryEnvelope,
  recoveryExpiry,
  recoveryStorageKey,
  type CaptureRecoveryContext,
  type RecoveryBaseline,
} from "./captureRecovery";
import { MemoryRecoveryStore } from "./captureRecoveryStore";
import type { CaptureField } from "./types";

const fields: CaptureField[] = [
  { id: "legal_name", label: "Legal name", type: "short_text", required: true },
  { id: "services", label: "Services", type: "multi_select", required: false },
  { id: "private_note", label: "Private note", type: "short_text", required: false, browser_cache_policy: "NO_BROWSER_CACHE" } as CaptureField,
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

function baseline(overrides: Partial<RecoveryBaseline> = {}): RecoveryBaseline {
  return {
    answers: {},
    fieldSequences: {},
    presentationMode: "AUTOMATIC",
    serverVersion: 3,
    ...overrides,
  };
}

const saveTime = new Date("2026-09-01T12:00:00.000Z");

describe("capture recovery", () => {
  it("encrypts only dirty recoverable intent and never persists secrets, file references or signatures in plaintext", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const current = context();

    const saved = await recovery.save(current, fields, baseline(), {
      answers: {
        legal_name: { text: "Example Ltd" },
        private_note: { text: "never cache me" },
        certificate: { artifact_ids: ["artifact-secret"] },
        signoff: { text: "signature-data" },
      },
      presentationMode: "WIZARD",
      page: 2,
      localSequence: 9,
    }, { now: saveTime });

    expect(saved).toMatchObject({ complete: false, hasUnsyncedChanges: true });
    expect(saved?.envelope.edits).toEqual([
      { fieldID: "certificate", baseSequence: 0, operation: "reselect" },
      { fieldID: "legal_name", baseSequence: 0, operation: "set", value: { text: "Example Ltd" } },
    ]);
    const raw = store.readRaw(recoveryStorageKey(current));
    expect(raw).toBeDefined();
    expect(raw).not.toContain("Example Ltd");
    expect(raw).not.toContain("never cache me");
    expect(raw).not.toContain("artifact-secret");
    expect(raw).not.toContain("signature-data");
    expect(raw).not.toContain("session-token");
  });

  it("uses a fresh AES-GCM IV for equivalent recovery writes", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const current = context();
    const local = { answers: { legal_name: { text: "Example Ltd" } }, presentationMode: "AUTOMATIC" as const };

    await recovery.save(current, fields, baseline(), local, { now: saveTime });
    const first = store.readRaw(recoveryStorageKey(current));
    await recovery.save(current, fields, baseline(), local, { now: new Date("2026-09-01T12:00:01.000Z") });
    const second = store.readRaw(recoveryStorageKey(current));

    expect(first).toBeDefined();
    expect(second).toBeDefined();
    expect(second).not.toBe(first);
  });

  it("fails authentication when ciphertext is replayed under a different AAD binding", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const source = context();
    const wrongEntity = context({ legalEntityID: "entity-b" });
    await recovery.save(source, fields, baseline(), { answers: { legal_name: { text: "Example Ltd" } }, presentationMode: "AUTOMATIC" }, { now: saveTime });
    const encrypted = await store.get(recoveryStorageKey(source));
    if (!encrypted) throw new Error("missing test envelope");
    await store.put(recoveryStorageKey(wrongEntity), encrypted);

    expect(await recovery.restore(wrongEntity, new Date("2026-09-01T12:01:00.000Z"))).toEqual({ status: "discarded", reason: "corrupt" });
    expect(await recovery.restore(source, new Date("2026-09-01T12:01:00.000Z"))).toMatchObject({ status: "restored" });
  });

  it("does not restore before the current server session authorizes the exact distribution", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const authorized = context();
    await recovery.save(authorized, fields, baseline(), { answers: { legal_name: { text: "Example Ltd" } }, presentationMode: "AUTOMATIC" }, { now: saveTime });

    expect(await recovery.restore(context({ authorized: false }), new Date("2026-09-01T12:01:00.000Z"))).toEqual({
      status: "denied",
      reason: "unauthorized",
    });
  });

  it("honours NO_BROWSER_CACHE and clears existing v2 recovery", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const enabled = context();
    await recovery.save(enabled, fields, baseline(), { answers: { legal_name: { text: "Example Ltd" } }, presentationMode: "AUTOMATIC" }, { now: saveTime });

    const disabled = context({ cachePolicy: "NO_BROWSER_CACHE" });
    expect(await recovery.save(disabled, fields, baseline(), { answers: { legal_name: { text: "Changed" } }, presentationMode: "AUTOMATIC" }, { now: saveTime })).toBeUndefined();
    expect(store.readRaw(recoveryStorageKey(enabled))).toBeUndefined();
    expect(await recovery.restore(disabled)).toEqual({ status: "denied", reason: "disabled" });
  });

  it("caps recovery at seven days, the deadline, or route expiry, whichever is earliest", () => {
    const now = new Date("2026-09-01T12:00:00.000Z");
    expect(recoveryExpiry({}, now).getTime()).toBe(now.getTime() + CAPTURE_RECOVERY_MAX_AGE_MS);
    expect(recoveryExpiry({ deadline: "2026-09-03T12:00:00.000Z" }, now).toISOString()).toBe("2026-09-03T12:00:00.000Z");
    expect(recoveryExpiry({ deadline: "2026-09-05T12:00:00.000Z", routeExpiresAt: "2026-09-02T12:00:00.000Z" }, now).toISOString()).toBe("2026-09-02T12:00:00.000Z");
  });

  it("discards expired and tampered envelopes instead of returning untrusted data", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const current = context({ routeExpiresAt: "2026-09-02T12:00:00.000Z" });
    await recovery.save(current, fields, baseline(), { answers: { legal_name: { text: "Example Ltd" } }, presentationMode: "AUTOMATIC" }, { now: saveTime });
    expect(await recovery.restore(current, new Date("2026-09-03T12:00:00.000Z"))).toEqual({ status: "discarded", reason: "expired" });

    await recovery.save(current, fields, baseline(), { answers: { legal_name: { text: "Example Ltd" } }, presentationMode: "AUTOMATIC" }, { now: saveTime });
    const key = recoveryStorageKey(current);
    const encrypted = await store.get(key);
    if (!encrypted) throw new Error("missing test envelope");
    const tampered = new Uint8Array(encrypted.ciphertext.slice(0));
    tampered[0] = (tampered[0] ?? 0) ^ 0xff;
    store.replaceRaw(key, { ...encrypted, ciphertext: tampered.buffer });
    expect(await recovery.restore(current, new Date("2026-09-01T12:01:00.000Z"))).toEqual({ status: "discarded", reason: "corrupt" });
    expect(store.readRaw(key)).toBeUndefined();
  });

  it("stores only local deltas so untouched server changes rebase without a false conflict", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const current = context();
    const saved = await recovery.save(current, fields, baseline({
      answers: { legal_name: { text: "Server Ltd" }, services: { values: ["Old"] } },
      fieldSequences: { legal_name: 4, services: 2 },
    }), {
      answers: { legal_name: { text: "Server Ltd" }, services: { values: ["Payments"] } },
      presentationMode: "AUTOMATIC",
      page: 1,
    }, { now: saveTime });
    if (!saved) throw new Error("missing recovery");

    expect(saved.envelope.edits).toEqual([{ fieldID: "services", baseSequence: 2, operation: "set", value: { values: ["Payments"] } }]);
    expect(rebaseRecoveryEnvelope({
      answers: { legal_name: { text: "Server renamed" }, services: { values: ["Old"] } },
      fieldSequences: { legal_name: 5, services: 2 },
      presentationMode: "AUTOMATIC",
    }, saved.envelope)).toEqual({
      answers: { legal_name: { text: "Server renamed" }, services: { values: ["Payments"] } },
      presentationMode: "AUTOMATIC",
      page: 1,
      filesToReselect: [],
      conflicts: [],
    });
  });

  it("surfaces only true same-field sequence races", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const saved = await recovery.save(context(), fields, baseline({
      answers: { legal_name: { text: "Server Ltd" } },
      fieldSequences: { legal_name: 4 },
    }), {
      answers: { legal_name: { text: "Local Ltd" } },
      presentationMode: "AUTOMATIC",
    }, { now: saveTime });
    if (!saved) throw new Error("missing recovery");

    expect(rebaseRecoveryEnvelope({
      answers: { legal_name: { text: "Server changed" } },
      fieldSequences: { legal_name: 5 },
      presentationMode: "AUTOMATIC",
    }, saved.envelope).conflicts).toEqual([{
      fieldID: "legal_name",
      serverValue: { text: "Server changed" },
      localValue: { text: "Local Ltd" },
      operation: "set",
      baseSequence: 4,
      sequence: 5,
    }]);
  });

  it("marks only an unsynced changed file for reselection", async () => {
    const store = new MemoryRecoveryStore();
    const recovery = new CaptureRecovery(store);
    const base = baseline({
      answers: { certificate: { artifact_ids: ["server-file"] } },
      fieldSequences: { certificate: 3 },
    });

    const unchanged = await recovery.save(context(), fields, base, {
      answers: { certificate: { artifact_ids: ["server-file"] } },
      presentationMode: "AUTOMATIC",
      page: 1,
    }, { now: saveTime });
    expect(unchanged?.envelope.edits).toEqual([]);

    const changed = await recovery.save(context(), fields, base, {
      answers: { certificate: { artifact_ids: ["local-file"] } },
      presentationMode: "AUTOMATIC",
      page: 1,
    }, { now: saveTime });
    expect(changed?.envelope.edits).toEqual([{ fieldID: "certificate", baseSequence: 3, operation: "reselect" }]);
  });

  it("stores a non-exportable AES-GCM device key", async () => {
    const store = new MemoryRecoveryStore();
    const key = await store.getOrCreateDeviceKey("device");
    expect(key.algorithm).toMatchObject({ name: "AES-GCM", length: 256 });
    expect(key.extractable).toBe(false);
    await expect(crypto.subtle.exportKey("raw", key)).rejects.toBeDefined();
  });
});
