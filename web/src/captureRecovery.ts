import { decryptRecoveryEnvelope, encryptRecoveryEnvelope, type RecoveryCryptoContext } from "./captureRecoveryCrypto";
import type { RecoveryStore } from "./captureRecoveryStore";
import type { CaptureAnswerInputs, CaptureAnswerValue, CaptureField } from "./types";

export const CAPTURE_RECOVERY_MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000;

type RecoveryAwareField = CaptureField & {
  browser_cache_policy?: "ALLOWED" | "ENCRYPTED_BROWSER_CACHE" | "NO_BROWSER_CACHE";
};

export type CaptureRecoveryContext = RecoveryCryptoContext & {
  workspaceID: string;
  serverVersion: number;
  authorized: boolean;
  deadline?: string;
  routeExpiresAt?: string;
  cachePolicy?: "ENCRYPTED_BROWSER_CACHE" | "NO_BROWSER_CACHE";
};

export type RecoveryEnvelope = {
  distributionID: string;
  workspaceID: string;
  schemaVersion: number;
  serverVersion: number;
  page: number;
  answers: CaptureAnswerInputs;
  filesToReselect: string[];
  localSequence: number;
  updatedAt: string;
  expiresAt: string;
};

export type RecoverySaveOptions = {
  page?: number;
  localSequence?: number;
  now?: Date;
};

export type RecoveryRestoreResult =
  | { status: "empty" }
  | { status: "restored"; envelope: RecoveryEnvelope }
  | { status: "denied"; reason: "unauthorized" | "disabled" }
  | { status: "discarded"; reason: "expired" | "schema_mismatch" | "invalid" | "corrupt" };

export type RecoveryMergeConflict = {
  fieldID: string;
  serverValue: CaptureAnswerValue | string;
  localValue: CaptureAnswerValue | string;
};

export type RecoveryMergeResult = {
  answers: CaptureAnswerInputs;
  conflicts: RecoveryMergeConflict[];
};

export class CaptureRecovery {
  constructor(private readonly store: RecoveryStore) {}

  async save(
    context: CaptureRecoveryContext,
    fields: CaptureField[],
    answers: CaptureAnswerInputs,
    options: RecoverySaveOptions = {},
  ): Promise<RecoveryEnvelope | undefined> {
    if (context.cachePolicy === "NO_BROWSER_CACHE") {
      await this.clear(context);
      return undefined;
    }
    if (!context.authorized) throw new Error("Recovery may only be saved for an authorized distribution session");

    const now = options.now ?? new Date();
    const expiresAt = recoveryExpiry(context, now);
    if (expiresAt.getTime() <= now.getTime()) {
      await this.clear(context);
      return undefined;
    }

    const sanitized = sanitizeRecoveryAnswers(fields, answers);
    const envelope: RecoveryEnvelope = {
      distributionID: context.distributionID,
      workspaceID: context.workspaceID,
      schemaVersion: context.schemaVersion,
      serverVersion: context.serverVersion,
      page: Math.max(0, Math.trunc(options.page ?? 0)),
      answers: sanitized.answers,
      filesToReselect: sanitized.filesToReselect,
      localSequence: Math.max(0, Math.trunc(options.localSequence ?? 0)),
      updatedAt: now.toISOString(),
      expiresAt: expiresAt.toISOString(),
    };
    const encrypted = await encryptRecoveryEnvelope(this.store, context, envelope, envelope.expiresAt);
    await this.store.put(recoveryStorageKey(context), encrypted);
    return envelope;
  }

  async restore(context: CaptureRecoveryContext, now = new Date()): Promise<RecoveryRestoreResult> {
    if (!context.authorized) return { status: "denied", reason: "unauthorized" };
    if (context.cachePolicy === "NO_BROWSER_CACHE") {
      await this.clear(context);
      return { status: "denied", reason: "disabled" };
    }

    const key = recoveryStorageKey(context);
    const encrypted = await this.store.get(key);
    if (!encrypted) return { status: "empty" };
    if (encrypted.schemaVersion !== context.schemaVersion) {
      await this.store.delete(key);
      return { status: "discarded", reason: "schema_mismatch" };
    }
    if (!validFutureTime(encrypted.expiresAt, now)) {
      await this.store.delete(key);
      return { status: "discarded", reason: "expired" };
    }

    let envelope: RecoveryEnvelope;
    try {
      envelope = await decryptRecoveryEnvelope<RecoveryEnvelope>(this.store, context, encrypted);
    } catch {
      await this.store.delete(key);
      return { status: "discarded", reason: "corrupt" };
    }

    if (!validEnvelope(context, envelope, now)) {
      await this.store.delete(key);
      return { status: "discarded", reason: "invalid" };
    }
    return { status: "restored", envelope };
  }

  clear(context: Pick<CaptureRecoveryContext, "origin" | "legalEntityID" | "distributionID" | "schemaVersion">): Promise<void> {
    return this.store.delete(recoveryStorageKey(context));
  }
}

export function recoveryStorageKey(context: Pick<CaptureRecoveryContext, "origin" | "legalEntityID" | "distributionID" | "schemaVersion">): string {
  return ["capture-recovery-v1", context.origin, context.legalEntityID, context.distributionID, context.schemaVersion].join("|");
}

export function recoveryExpiry(context: Pick<CaptureRecoveryContext, "deadline" | "routeExpiresAt">, now = new Date()): Date {
  const candidates = [now.getTime() + CAPTURE_RECOVERY_MAX_AGE_MS];
  for (const value of [context.deadline, context.routeExpiresAt]) {
    if (!value) continue;
    const parsed = Date.parse(value);
    if (Number.isFinite(parsed)) candidates.push(parsed);
  }
  return new Date(Math.min(...candidates));
}

export function sanitizeRecoveryAnswers(fields: CaptureField[], answers: CaptureAnswerInputs): {
  answers: CaptureAnswerInputs;
  filesToReselect: string[];
} {
  const fieldMetadata = new Map(fields.map((field) => {
    const recoveryField = field as RecoveryAwareField;
    return [field.id, { type: field.type, cachePolicy: recoveryField.browser_cache_policy }] as const;
  }));
  const safe: CaptureAnswerInputs = {};
  const filesToReselect = new Set<string>();

  for (const [fieldID, raw] of Object.entries(answers)) {
    const metadata = fieldMetadata.get(fieldID);
    if (!metadata || metadata.cachePolicy === "NO_BROWSER_CACHE") continue;
    const type = metadata.type;
    if (isBinaryOrSignatureField(type)) {
      if (isFileField(type) && hasAnswer(raw)) filesToReselect.add(fieldID);
      continue;
    }
    if (typeof raw === "string") {
      safe[fieldID] = raw;
      continue;
    }
    if (!raw || raw.artifact_ids?.length || raw.document) {
      if (raw?.artifact_ids?.length || raw?.document) filesToReselect.add(fieldID);
      continue;
    }
    const value: CaptureAnswerValue = {};
    if (typeof raw.text === "string") value.text = raw.text;
    if (Array.isArray(raw.values) && raw.values.every((entry) => typeof entry === "string")) value.values = [...raw.values];
    if (Object.keys(value).length > 0) safe[fieldID] = value;
  }

  return { answers: safe, filesToReselect: [...filesToReselect].sort() };
}

export function mergeRecoveredAnswers(serverAnswers: CaptureAnswerInputs, localAnswers: CaptureAnswerInputs): RecoveryMergeResult {
  const merged: CaptureAnswerInputs = { ...serverAnswers };
  const conflicts: RecoveryMergeConflict[] = [];

  for (const [fieldID, localValue] of Object.entries(localAnswers)) {
    const serverValue = serverAnswers[fieldID];
    if (serverValue === undefined) {
      merged[fieldID] = localValue;
      continue;
    }
    if (sameAnswer(serverValue, localValue)) continue;
    conflicts.push({ fieldID, serverValue, localValue });
  }

  return { answers: merged, conflicts };
}

function validEnvelope(context: CaptureRecoveryContext, envelope: RecoveryEnvelope, now: Date): boolean {
  return envelope.distributionID === context.distributionID
    && envelope.workspaceID === context.workspaceID
    && envelope.schemaVersion === context.schemaVersion
    && Number.isInteger(envelope.serverVersion)
    && envelope.serverVersion >= 0
    && Number.isInteger(envelope.localSequence)
    && envelope.localSequence >= 0
    && Number.isInteger(envelope.page)
    && envelope.page >= 0
    && validFutureTime(envelope.expiresAt, now);
}

function validFutureTime(value: string, now: Date): boolean {
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) && parsed > now.getTime();
}

function isBinaryOrSignatureField(type?: string): boolean {
  return type === "file" || type === "photo" || type === "vendor_document" || type === "signature";
}

function isFileField(type?: string): boolean {
  return type === "file" || type === "photo" || type === "vendor_document";
}

function hasAnswer(value: CaptureAnswerValue | string): boolean {
  if (typeof value === "string") return value.length > 0;
  return Boolean(value.text || value.values?.length || value.artifact_ids?.length || value.document);
}

function sameAnswer(left: CaptureAnswerValue | string, right: CaptureAnswerValue | string): boolean {
  return JSON.stringify(normalizeComparable(left)) === JSON.stringify(normalizeComparable(right));
}

function normalizeComparable(value: CaptureAnswerValue | string): CaptureAnswerValue {
  if (typeof value === "string") return { text: value };
  const normalized: CaptureAnswerValue = {};
  if (value.text !== undefined) normalized.text = value.text;
  if (value.values !== undefined) normalized.values = [...value.values];
  if (value.artifact_ids !== undefined) normalized.artifact_ids = [...value.artifact_ids];
  if (value.document !== undefined) normalized.document = { ...value.document };
  return normalized;
}
