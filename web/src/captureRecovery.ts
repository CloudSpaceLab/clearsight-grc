import { decryptRecoveryEnvelope, encryptRecoveryEnvelope, type RecoveryCryptoContext } from "./captureRecoveryCrypto";
import type { RecoveryStore } from "./captureRecoveryStore";
import type { CaptureAnswerInputs, CaptureAnswerValue, CaptureAnswers, CaptureField, CapturePresentationMode } from "./types";

export const CAPTURE_RECOVERY_MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000;

const RECOVERY_PAYLOAD_VERSION = 2 as const;
const RECOVERY_STORAGE_PREFIX = "capture-recovery-v2";
const LEGACY_RECOVERY_STORAGE_PREFIX = "capture-recovery-v1";

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

export type RecoveryDeltaOperation = "set" | "delete" | "reselect";

export type RecoveryFieldDelta = {
  fieldID: string;
  baseSequence: number;
  operation: RecoveryDeltaOperation;
  value?: CaptureAnswerValue;
};

export type RecoveryEnvelope = {
  payloadVersion: typeof RECOVERY_PAYLOAD_VERSION;
  distributionID: string;
  workspaceID: string;
  schemaVersion: number;
  serverVersion: number;
  page: number;
  presentationMode: CapturePresentationMode;
  basePresentationMode: CapturePresentationMode;
  presentationModeDirty: boolean;
  edits: RecoveryFieldDelta[];
  complete: boolean;
  localSequence: number;
  updatedAt: string;
  expiresAt: string;
};

export type RecoveryBaseline = {
  answers: CaptureAnswerInputs;
  fieldSequences: Record<string, number>;
  presentationMode: CapturePresentationMode;
  serverVersion: number;
};

export type RecoveryLocalState = {
  answers: CaptureAnswerInputs;
  presentationMode: CapturePresentationMode;
  page?: number;
  filesToReselect?: string[];
  localSequence?: number;
};

export type RecoverySaveOptions = {
  now?: Date;
};

export type RecoverySaveResult = {
  envelope: RecoveryEnvelope;
  complete: boolean;
  hasUnsyncedChanges: boolean;
  recoverableChanges: number;
};

export type RecoveryRestoreResult =
  | { status: "empty" }
  | { status: "restored"; envelope: RecoveryEnvelope }
  | { status: "denied"; reason: "unauthorized" | "disabled" }
  | { status: "discarded"; reason: "expired" | "schema_mismatch" | "invalid" | "corrupt" };

export type RecoveryMergeConflict = {
  fieldID: string;
  serverValue: CaptureAnswerValue;
  localValue: CaptureAnswerValue;
  operation: RecoveryDeltaOperation;
  baseSequence: number;
  sequence: number;
};

export type RecoveryRebaseResult = {
  answers: CaptureAnswers;
  presentationMode: CapturePresentationMode;
  page: number;
  filesToReselect: string[];
  conflicts: RecoveryMergeConflict[];
};

export class CaptureRecovery {
  constructor(private readonly store: RecoveryStore) {}

  async save(
    context: CaptureRecoveryContext,
    fields: CaptureField[],
    baseline: RecoveryBaseline,
    local: RecoveryLocalState,
    options: RecoverySaveOptions = {},
  ): Promise<RecoverySaveResult | undefined> {
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

    const delta = buildRecoveryDelta(fields, baseline, local);
    const envelope: RecoveryEnvelope = {
      payloadVersion: RECOVERY_PAYLOAD_VERSION,
      distributionID: context.distributionID,
      workspaceID: context.workspaceID,
      schemaVersion: context.schemaVersion,
      serverVersion: baseline.serverVersion,
      page: Math.max(0, Math.trunc(local.page ?? 0)),
      presentationMode: local.presentationMode,
      basePresentationMode: baseline.presentationMode,
      presentationModeDirty: local.presentationMode !== baseline.presentationMode,
      edits: delta.edits,
      complete: delta.complete,
      localSequence: Math.max(0, Math.trunc(local.localSequence ?? 0)),
      updatedAt: now.toISOString(),
      expiresAt: expiresAt.toISOString(),
    };
    const encrypted = await encryptRecoveryEnvelope(this.store, context, envelope, envelope.expiresAt);
    await this.store.put(recoveryStorageKey(context), encrypted);
    return {
      envelope,
      complete: envelope.complete,
      hasUnsyncedChanges: delta.hasUnsyncedChanges || envelope.presentationModeDirty,
      recoverableChanges: envelope.edits.length + (envelope.presentationModeDirty ? 1 : 0),
    };
  }

  async restore(context: CaptureRecoveryContext, now = new Date()): Promise<RecoveryRestoreResult> {
    if (!context.authorized) return { status: "denied", reason: "unauthorized" };
    if (context.cachePolicy === "NO_BROWSER_CACHE") {
      await this.clear(context);
      return { status: "denied", reason: "disabled" };
    }

    const key = recoveryStorageKey(context);
    const encrypted = await this.store.get(key);
    if (!encrypted) {
      await this.store.delete(legacyRecoveryStorageKey(context));
      return { status: "empty" };
    }
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

  async clear(context: Pick<CaptureRecoveryContext, "origin" | "legalEntityID" | "distributionID" | "schemaVersion">): Promise<void> {
    await Promise.all([
      this.store.delete(recoveryStorageKey(context)),
      this.store.delete(legacyRecoveryStorageKey(context)),
    ]);
  }
}

export function recoveryStorageKey(context: Pick<CaptureRecoveryContext, "origin" | "legalEntityID" | "distributionID" | "schemaVersion">): string {
  return [RECOVERY_STORAGE_PREFIX, context.origin, context.legalEntityID, context.distributionID, context.schemaVersion].join("|");
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

export function rebaseRecoveryEnvelope(
  server: {
    answers: CaptureAnswerInputs;
    fieldSequences: Record<string, number>;
    presentationMode: CapturePresentationMode;
  },
  envelope: RecoveryEnvelope,
): RecoveryRebaseResult {
  const answers = normalizeAnswers(server.answers);
  const filesToReselect = new Set<string>();
  const conflicts: RecoveryMergeConflict[] = [];

  for (const edit of envelope.edits) {
    const sequence = server.fieldSequences[edit.fieldID] ?? 0;
    const serverValue = answers[edit.fieldID];
    if (sequence === edit.baseSequence) {
      applyRecoveryEdit(answers, filesToReselect, edit);
      continue;
    }
    if (recoveryEditSatisfied(edit, serverValue)) continue;
    conflicts.push({
      fieldID: edit.fieldID,
      serverValue: serverValue ? { ...serverValue } : {},
      localValue: edit.value ? { ...edit.value, values: edit.value.values ? [...edit.value.values] : undefined } : {},
      operation: edit.operation,
      baseSequence: edit.baseSequence,
      sequence,
    });
  }

  const presentationMode = envelope.presentationModeDirty && server.presentationMode === envelope.basePresentationMode
    ? envelope.presentationMode
    : server.presentationMode;

  return {
    answers,
    presentationMode,
    page: envelope.page,
    filesToReselect: [...filesToReselect].sort(),
    conflicts,
  };
}

export function sanitizeRecoveryAnswers(fields: CaptureField[], answers: CaptureAnswerInputs): {
  answers: CaptureAnswerInputs;
  filesToReselect: string[];
} {
  const fieldMetadata = recoveryFieldMetadata(fields);
  const safe: CaptureAnswerInputs = {};
  const filesToReselect = new Set<string>();

  for (const [fieldID, raw] of Object.entries(answers)) {
    const metadata = fieldMetadata.get(fieldID);
    if (!metadata || metadata.cachePolicy === "NO_BROWSER_CACHE") continue;
    if (isFileField(metadata.type)) {
      if (hasAnswer(raw)) filesToReselect.add(fieldID);
      continue;
    }
    if (metadata.type === "signature") continue;
    const value = safeRecoveryValue(raw);
    if (value && hasAnswer(value)) safe[fieldID] = value;
  }

  return { answers: safe, filesToReselect: [...filesToReselect].sort() };
}

function buildRecoveryDelta(fields: CaptureField[], baseline: RecoveryBaseline, local: RecoveryLocalState): {
  edits: RecoveryFieldDelta[];
  complete: boolean;
  hasUnsyncedChanges: boolean;
} {
  const metadata = recoveryFieldMetadata(fields);
  const forcedReselection = new Set(local.filesToReselect ?? []);
  const fieldIDs = new Set([...Object.keys(baseline.answers), ...Object.keys(local.answers), ...forcedReselection]);
  const edits: RecoveryFieldDelta[] = [];
  let complete = true;
  let hasUnsyncedChanges = false;

  for (const fieldID of [...fieldIDs].sort()) {
    const field = metadata.get(fieldID);
    const baseValue = baseline.answers[fieldID];
    const localValue = local.answers[fieldID];
    const forceReselect = forcedReselection.has(fieldID);
    if (!forceReselect && sameOptionalAnswer(baseValue, localValue)) continue;

    hasUnsyncedChanges = true;
    if (!field || field.cachePolicy === "NO_BROWSER_CACHE") {
      complete = false;
      continue;
    }
    const baseSequence = baseline.fieldSequences[fieldID] ?? 0;
    if (forceReselect) {
      if (isFileField(field.type)) edits.push({ fieldID, baseSequence, operation: "reselect" });
      else complete = false;
      continue;
    }
    if (field.type === "signature") {
      complete = false;
      continue;
    }
    if (isFileField(field.type)) {
      edits.push({ fieldID, baseSequence, operation: hasAnswer(localValue) ? "reselect" : "delete" });
      continue;
    }
    if (!hasAnswer(localValue)) {
      edits.push({ fieldID, baseSequence, operation: "delete" });
      continue;
    }
    const safeValue = safeRecoveryValue(localValue);
    if (!safeValue) {
      complete = false;
      continue;
    }
    edits.push({ fieldID, baseSequence, operation: "set", value: safeValue });
  }

  return { edits, complete, hasUnsyncedChanges };
}

function recoveryFieldMetadata(fields: CaptureField[]) {
  return new Map(fields.map((field) => {
    const recoveryField = field as RecoveryAwareField;
    return [field.id, { type: field.type, cachePolicy: recoveryField.browser_cache_policy }] as const;
  }));
}

function applyRecoveryEdit(answers: CaptureAnswers, filesToReselect: Set<string>, edit: RecoveryFieldDelta) {
  if (edit.operation === "set" && edit.value) {
    answers[edit.fieldID] = cloneAnswer(edit.value);
    return;
  }
  delete answers[edit.fieldID];
  if (edit.operation === "reselect") filesToReselect.add(edit.fieldID);
}

function recoveryEditSatisfied(edit: RecoveryFieldDelta, serverValue?: CaptureAnswerValue): boolean {
  if (edit.operation === "delete") return !hasAnswer(serverValue);
  if (edit.operation === "set" && edit.value) return sameOptionalAnswer(edit.value, serverValue);
  return false;
}

function validEnvelope(context: CaptureRecoveryContext, envelope: RecoveryEnvelope, now: Date): boolean {
  return envelope?.payloadVersion === RECOVERY_PAYLOAD_VERSION
    && envelope.distributionID === context.distributionID
    && envelope.workspaceID === context.workspaceID
    && envelope.schemaVersion === context.schemaVersion
    && Number.isInteger(envelope.serverVersion)
    && envelope.serverVersion >= 0
    && Number.isInteger(envelope.localSequence)
    && envelope.localSequence >= 0
    && Number.isInteger(envelope.page)
    && envelope.page >= 0
    && validPresentationMode(envelope.presentationMode)
    && validPresentationMode(envelope.basePresentationMode)
    && typeof envelope.presentationModeDirty === "boolean"
    && typeof envelope.complete === "boolean"
    && Array.isArray(envelope.edits)
    && envelope.edits.every(validRecoveryDelta)
    && validFutureTime(envelope.expiresAt, now);
}

function validRecoveryDelta(delta: RecoveryFieldDelta): boolean {
  if (!delta || typeof delta.fieldID !== "string" || !delta.fieldID || !Number.isInteger(delta.baseSequence) || delta.baseSequence < 0) return false;
  if (!["set", "delete", "reselect"].includes(delta.operation)) return false;
  if (delta.operation === "set") return Boolean(delta.value && safeRecoveryValue(delta.value) && hasAnswer(delta.value));
  return delta.value === undefined;
}

function validPresentationMode(value: unknown): value is CapturePresentationMode {
  return value === "CLASSIC" || value === "WIZARD" || value === "AUTOMATIC";
}

function validFutureTime(value: string, now: Date): boolean {
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) && parsed > now.getTime();
}

function legacyRecoveryStorageKey(context: Pick<CaptureRecoveryContext, "origin" | "legalEntityID" | "distributionID" | "schemaVersion">): string {
  return [LEGACY_RECOVERY_STORAGE_PREFIX, context.origin, context.legalEntityID, context.distributionID, context.schemaVersion].join("|");
}

function isFileField(type?: string): boolean {
  return type === "file" || type === "photo" || type === "vendor_document";
}

function safeRecoveryValue(raw: CaptureAnswerValue | string | undefined): CaptureAnswerValue | undefined {
  if (raw === undefined) return undefined;
  const normalized = typeof raw === "string" ? { text: raw } : raw;
  if (normalized.artifact_ids?.length || normalized.document) return undefined;
  const value: CaptureAnswerValue = {};
  if (typeof normalized.text === "string") value.text = normalized.text;
  if (Array.isArray(normalized.values) && normalized.values.every((entry) => typeof entry === "string")) value.values = [...normalized.values];
  return Object.keys(value).length > 0 ? value : undefined;
}

function normalizeAnswers(answers: CaptureAnswerInputs): CaptureAnswers {
  return Object.fromEntries(Object.entries(answers).map(([fieldID, value]) => [fieldID, normalizeAnswer(value)]));
}

function normalizeAnswer(value: CaptureAnswerValue | string): CaptureAnswerValue {
  return typeof value === "string" ? { text: value } : cloneAnswer(value);
}

function cloneAnswer(value: CaptureAnswerValue): CaptureAnswerValue {
  return {
    ...(value.text !== undefined ? { text: value.text } : {}),
    ...(value.values !== undefined ? { values: [...value.values] } : {}),
    ...(value.artifact_ids !== undefined ? { artifact_ids: [...value.artifact_ids] } : {}),
    ...(value.document !== undefined ? { document: { ...value.document } } : {}),
  };
}

function hasAnswer(value: CaptureAnswerValue | string | undefined): boolean {
  if (typeof value === "string") return value.length > 0;
  return Boolean(value?.text || value?.values?.length || value?.artifact_ids?.length || value?.document);
}

function sameOptionalAnswer(left?: CaptureAnswerValue | string, right?: CaptureAnswerValue | string): boolean {
  if (left === undefined || right === undefined) return !hasAnswer(left) && !hasAnswer(right);
  return JSON.stringify(normalizeComparable(left)) === JSON.stringify(normalizeComparable(right));
}

function normalizeComparable(value: CaptureAnswerValue | string): CaptureAnswerValue {
  return normalizeAnswer(value);
}
