import type { EncryptedRecoveryEnvelope, RecoveryStore } from "./captureRecoveryStore";

export type RecoveryCryptoContext = {
  origin: string;
  legalEntityID: string;
  distributionID: string;
  schemaVersion: number;
};

const encoder = new TextEncoder();
const decoder = new TextDecoder();

export function recoveryAAD(context: RecoveryCryptoContext): Uint8Array {
  return encoder.encode([
    "clearsight.capture-recovery.v1",
    context.origin,
    context.legalEntityID,
    context.distributionID,
    String(context.schemaVersion),
  ].join("\n"));
}

export function recoveryDeviceKeyName(context: Pick<RecoveryCryptoContext, "origin">): string {
  return `capture-recovery-device:${context.origin}`;
}

export async function encryptRecoveryEnvelope<T extends object>(
  store: RecoveryStore,
  context: RecoveryCryptoContext,
  value: T,
  expiresAt: string,
): Promise<EncryptedRecoveryEnvelope> {
  const key = await store.getOrCreateDeviceKey(recoveryDeviceKeyName(context));
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv, additionalData: recoveryAAD(context), tagLength: 128 },
    key,
    encoder.encode(JSON.stringify(value)),
  );
  return {
    version: 1,
    algorithm: "AES-GCM",
    iv: iv.buffer.slice(iv.byteOffset, iv.byteOffset + iv.byteLength),
    ciphertext,
    expiresAt,
    schemaVersion: context.schemaVersion,
  };
}

export async function decryptRecoveryEnvelope<T>(
  store: RecoveryStore,
  context: RecoveryCryptoContext,
  value: EncryptedRecoveryEnvelope,
): Promise<T> {
  if (value.version !== 1 || value.algorithm !== "AES-GCM" || value.schemaVersion !== context.schemaVersion) {
    throw new Error("Recovery envelope is incompatible");
  }
  const key = await store.getOrCreateDeviceKey(recoveryDeviceKeyName(context));
  const plaintext = await crypto.subtle.decrypt(
    {
      name: "AES-GCM",
      iv: new Uint8Array(value.iv),
      additionalData: recoveryAAD(context),
      tagLength: 128,
    },
    key,
    value.ciphertext,
  );
  return JSON.parse(decoder.decode(plaintext)) as T;
}
