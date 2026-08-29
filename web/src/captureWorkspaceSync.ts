import {
  FormWorkspaceConflictError,
  normalizeCaptureAnswer,
  normalizeCaptureAnswers,
  saveFormResponseWorkspace,
  type FormResponseWorkspace,
  type FormWorkspaceConflict,
  type FormWorkspaceEditInput,
  type SaveFormResponseWorkspaceInput,
} from "./captureApi";
import {
  CaptureRecovery,
  mergeRecoveredAnswers,
  type CaptureRecoveryContext,
  type RecoveryMergeConflict,
} from "./captureRecovery";
import { apiErrorKind } from "./http";
import type { CaptureAnswerValue, CaptureAnswers, CaptureField, CapturePresentationMode } from "./types";

export type CaptureWorkspaceSaveState = "saved_server" | "saving" | "saved_device" | "conflict" | "failed";

export type CaptureWorkspaceConflict = {
  fieldID: string;
  serverValue: CaptureAnswerValue;
  localValue: CaptureAnswerValue;
  sequence: number;
};

export type CaptureWorkspaceSyncSnapshot = {
  answers: CaptureAnswers;
  presentationMode: CapturePresentationMode;
  page: number;
  saveState: CaptureWorkspaceSaveState;
  filesToReselect: string[];
  conflicts: CaptureWorkspaceConflict[];
};

type SaveWorkspace = (sessionToken: string, input: SaveFormResponseWorkspaceInput) => Promise<FormResponseWorkspace>;

type CaptureWorkspaceSyncOptions = {
  sessionToken: string;
  fields: CaptureField[];
  workspace: FormResponseWorkspace;
  recovery?: CaptureRecovery;
  recoveryContext: CaptureRecoveryContext;
  debounceMs?: number;
  saveWorkspace?: SaveWorkspace;
  onStateChange?: (snapshot: CaptureWorkspaceSyncSnapshot) => void;
  onError?: (error: unknown) => void;
};

type LocalSnapshot = {
  answers: CaptureAnswers;
  presentationMode: CapturePresentationMode;
  page: number;
  localSequence: number;
};

export class CaptureWorkspaceSync {
  private readonly sessionToken: string;
  private readonly fields: CaptureField[];
  private readonly recovery?: CaptureRecovery;
  private readonly debounceMs: number;
  private readonly saveWorkspace: SaveWorkspace;
  private readonly onStateChange?: (snapshot: CaptureWorkspaceSyncSnapshot) => void;
  private readonly onError?: (error: unknown) => void;
  private recoveryContext: CaptureRecoveryContext;
  private serverWorkspace: FormResponseWorkspace;
  private latestAnswers: CaptureAnswers;
  private latestMode: CapturePresentationMode;
  private page = 0;
  private localSequence = 0;
  private filesToReselect: string[] = [];
  private conflicts: CaptureWorkspaceConflict[] = [];
  private saveState: CaptureWorkspaceSaveState = "saved_server";
  private localRecoveryAvailable = false;
  private localWrite: Promise<void> = Promise.resolve();
  private flushPromise: Promise<boolean> | null = null;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private disposed = false;

  private constructor(options: CaptureWorkspaceSyncOptions) {
    this.sessionToken = options.sessionToken;
    this.fields = options.fields;
    this.serverWorkspace = cloneWorkspace(options.workspace);
    this.latestAnswers = { ...options.workspace.answers };
    this.latestMode = options.workspace.presentation_mode;
    this.recovery = options.recovery;
    this.recoveryContext = { ...options.recoveryContext, serverVersion: options.workspace.workspace.version };
    this.debounceMs = Math.max(0, options.debounceMs ?? 500);
    this.saveWorkspace = options.saveWorkspace ?? saveFormResponseWorkspace;
    this.onStateChange = options.onStateChange;
    this.onError = options.onError;
  }

  static async create(options: CaptureWorkspaceSyncOptions): Promise<CaptureWorkspaceSync> {
    const controller = new CaptureWorkspaceSync(options);
    await controller.restore();
    controller.emit();
    return controller;
  }

  snapshot(): CaptureWorkspaceSyncSnapshot {
    return {
      answers: { ...this.latestAnswers },
      presentationMode: this.latestMode,
      page: this.page,
      saveState: this.saveState,
      filesToReselect: [...this.filesToReselect],
      conflicts: this.conflicts.map((conflict) => ({
        ...conflict,
        serverValue: { ...conflict.serverValue },
        localValue: { ...conflict.localValue },
      })),
    };
  }

  currentWorkspace(): FormResponseWorkspace {
    return cloneWorkspace(this.serverWorkspace);
  }

  change(answers: CaptureAnswers, presentationMode: CapturePresentationMode, page = this.page): void {
    if (this.disposed) return;
    this.latestAnswers = { ...answers };
    this.latestMode = presentationMode;
    this.page = Math.max(0, Math.trunc(page));
    this.localSequence += 1;
    const snapshot = this.localSnapshot();

    if (!this.recovery) {
      if (this.conflicts.length === 0) this.setSaveState("saving");
      this.scheduleFlush();
      return;
    }

    this.localWrite = this.localWrite.catch(() => undefined).then(async () => {
      const envelope = await this.recovery!.save(
        { ...this.recoveryContext, serverVersion: this.serverWorkspace.workspace.version },
        this.fields,
        snapshot.answers,
        { page: snapshot.page, localSequence: snapshot.localSequence },
      );
      this.localRecoveryAvailable = envelope !== undefined;
      if (!this.disposed && this.conflicts.length === 0 && snapshot.localSequence === this.localSequence) {
        this.setSaveState(this.localRecoveryAvailable ? "saved_device" : "saving");
      }
    }).catch((error) => {
      this.localRecoveryAvailable = false;
      if (!this.disposed) {
        this.setSaveState("failed");
        this.onError?.(error);
      }
    });
    this.scheduleFlush();
  }

  async flush(): Promise<boolean> {
    if (this.disposed) return false;
    this.clearTimer();
    await this.localWrite.catch(() => undefined);
    if (this.conflicts.length > 0) {
      this.setSaveState("conflict");
      return false;
    }
    if (this.flushPromise) {
      const previous = await this.flushPromise;
      if (!previous || this.conflicts.length > 0) return false;
      if (this.hasUnsyncedChanges()) return this.flush();
      return true;
    }

    const operation = this.performFlush();
    this.flushPromise = operation;
    try {
      return await operation;
    } finally {
      if (this.flushPromise === operation) this.flushPromise = null;
    }
  }

  retry(): Promise<boolean> {
    return this.flush();
  }

  async clearRecovery(): Promise<void> {
    this.clearTimer();
    await this.localWrite.catch(() => undefined);
    if (this.recovery) await this.recovery.clear(this.recoveryContext);
    this.localRecoveryAvailable = false;
  }

  dispose(): void {
    this.disposed = true;
    this.clearTimer();
  }

  private async restore(): Promise<void> {
    if (!this.recovery) return;
    const restored = await this.recovery.restore(this.recoveryContext);
    if (restored.status !== "restored") return;

    this.localSequence = restored.envelope.localSequence;
    this.page = restored.envelope.page;
    this.filesToReselect = [...restored.envelope.filesToReselect];
    this.localRecoveryAvailable = true;

    if (restored.envelope.serverVersion === this.serverWorkspace.workspace.version) {
      this.latestAnswers = normalizeCaptureAnswers({ ...this.serverWorkspace.answers, ...restored.envelope.answers });
      this.saveState = this.hasUnsyncedChanges() ? "saved_device" : "saved_server";
      return;
    }

    const merged = mergeRecoveredAnswers(this.serverWorkspace.answers, restored.envelope.answers);
    this.latestAnswers = normalizeCaptureAnswers(merged.answers);
    this.conflicts = merged.conflicts.map((conflict) => recoveryConflict(this.serverWorkspace, conflict));
    this.saveState = this.conflicts.length > 0 ? "conflict" : this.hasUnsyncedChanges() ? "saved_device" : "saved_server";
  }

  private async performFlush(): Promise<boolean> {
    const sent = this.localSnapshot();
    const edits = workspaceEdits(this.serverWorkspace, sent.answers);
    const modeChanged = sent.presentationMode !== this.serverWorkspace.presentation_mode;
    if (edits.length === 0 && !modeChanged) {
      this.setSaveState("saved_server");
      return true;
    }

    this.setSaveState("saving");
    try {
      const saved = await this.saveWorkspace(this.sessionToken, {
        expected_version: this.serverWorkspace.workspace.version,
        presentation_mode: sent.presentationMode,
        edits,
      });
      if (this.disposed) return false;
      this.serverWorkspace = cloneWorkspace(saved);
      this.recoveryContext = { ...this.recoveryContext, serverVersion: saved.workspace.version };
      await this.persistCurrentAgainstServer();
      if (sameLocalSnapshot(sent, this.localSnapshot())) {
        this.setSaveState("saved_server");
      } else {
        this.setSaveState(this.localRecoveryAvailable ? "saved_device" : "saving");
        this.scheduleFlush();
      }
      return true;
    } catch (error) {
      if (this.disposed) return false;
      if (error instanceof FormWorkspaceConflictError) {
        this.applyServerConflict(error.conflict);
        await this.persistCurrentAgainstServer();
        this.setSaveState("conflict");
        return false;
      }
      const kind = apiErrorKind(error);
      this.setSaveState(this.localRecoveryAvailable && kind !== "validation" ? "saved_device" : "failed");
      this.onError?.(error);
      return false;
    }
  }

  private async persistCurrentAgainstServer(): Promise<void> {
    if (!this.recovery) {
      this.localRecoveryAvailable = false;
      return;
    }
    try {
      const envelope = await this.recovery.save(
        { ...this.recoveryContext, serverVersion: this.serverWorkspace.workspace.version },
        this.fields,
        this.latestAnswers,
        { page: this.page, localSequence: this.localSequence },
      );
      this.localRecoveryAvailable = envelope !== undefined;
    } catch (error) {
      this.localRecoveryAvailable = false;
      this.onError?.(error);
    }
  }

  private applyServerConflict(conflict: FormWorkspaceConflict): void {
    const answers = { ...this.serverWorkspace.answers };
    const sequences = { ...this.serverWorkspace.field_sequences };
    for (const changed of conflict.changed_fields) {
      if (hasAnswer(changed.server_value)) answers[changed.field_id] = normalizeCaptureAnswer(changed.server_value);
      else delete answers[changed.field_id];
      sequences[changed.field_id] = changed.sequence;
    }
    this.serverWorkspace = {
      ...this.serverWorkspace,
      workspace: { ...this.serverWorkspace.workspace, version: conflict.current_version },
      answers,
      field_sequences: sequences,
    };
    this.recoveryContext = { ...this.recoveryContext, serverVersion: conflict.current_version };
    this.conflicts = conflict.changed_fields.flatMap((changed) => {
      const localValue = this.latestAnswers[changed.field_id] ?? {};
      const serverValue = normalizeCaptureAnswer(changed.server_value);
      return sameAnswer(serverValue, localValue) ? [] : [{
        fieldID: changed.field_id,
        serverValue,
        localValue: normalizeCaptureAnswer(localValue),
        sequence: changed.sequence,
      }];
    });
  }

  private hasUnsyncedChanges(): boolean {
    return workspaceEdits(this.serverWorkspace, this.latestAnswers).length > 0
      || this.latestMode !== this.serverWorkspace.presentation_mode;
  }

  private localSnapshot(): LocalSnapshot {
    return {
      answers: { ...this.latestAnswers },
      presentationMode: this.latestMode,
      page: this.page,
      localSequence: this.localSequence,
    };
  }

  private scheduleFlush(): void {
    this.clearTimer();
    if (this.disposed || this.conflicts.length > 0) return;
    this.timer = setTimeout(() => void this.flush(), this.debounceMs);
  }

  private clearTimer(): void {
    if (this.timer) clearTimeout(this.timer);
    this.timer = null;
  }

  private setSaveState(saveState: CaptureWorkspaceSaveState): void {
    if (this.saveState === saveState) return;
    this.saveState = saveState;
    this.emit();
  }

  private emit(): void {
    this.onStateChange?.(this.snapshot());
  }
}

export function workspaceEdits(workspace: FormResponseWorkspace, answers: CaptureAnswers): FormWorkspaceEditInput[] {
  const fieldIDs = new Set([...Object.keys(workspace.answers), ...Object.keys(answers)]);
  return [...fieldIDs].sort().flatMap((fieldID) => {
    const serverValue = workspace.answers[fieldID];
    const localValue = answers[fieldID];
    if (sameOptionalAnswer(serverValue, localValue)) return [];
    return [{
      field_id: fieldID,
      value: localValue ? normalizeCaptureAnswer(localValue) : {},
      base_sequence: workspace.field_sequences[fieldID] ?? 0,
    }];
  });
}

function recoveryConflict(workspace: FormResponseWorkspace, conflict: RecoveryMergeConflict): CaptureWorkspaceConflict {
  return {
    fieldID: conflict.fieldID,
    serverValue: normalizeCaptureAnswer(conflict.serverValue),
    localValue: normalizeCaptureAnswer(conflict.localValue),
    sequence: workspace.field_sequences[conflict.fieldID] ?? 0,
  };
}

function sameLocalSnapshot(left: LocalSnapshot, right: LocalSnapshot): boolean {
  return left.presentationMode === right.presentationMode
    && left.page === right.page
    && left.localSequence === right.localSequence
    && JSON.stringify(left.answers) === JSON.stringify(right.answers);
}

function sameOptionalAnswer(left?: CaptureAnswerValue, right?: CaptureAnswerValue): boolean {
  if (left === undefined || right === undefined) return left === right;
  return sameAnswer(left, right);
}

function sameAnswer(left: CaptureAnswerValue, right: CaptureAnswerValue): boolean {
  return JSON.stringify(normalizeComparable(left)) === JSON.stringify(normalizeComparable(right));
}

function normalizeComparable(value: CaptureAnswerValue): CaptureAnswerValue {
  const normalized: CaptureAnswerValue = {};
  if (value.text !== undefined) normalized.text = value.text;
  if (value.values !== undefined) normalized.values = [...value.values];
  if (value.artifact_ids !== undefined) normalized.artifact_ids = [...value.artifact_ids];
  if (value.document !== undefined) normalized.document = { ...value.document };
  return normalized;
}

function hasAnswer(value: CaptureAnswerValue): boolean {
  return Boolean(value.text || value.values?.length || value.artifact_ids?.length || value.document);
}

function cloneWorkspace(workspace: FormResponseWorkspace): FormResponseWorkspace {
  return {
    ...workspace,
    workspace: { ...workspace.workspace },
    answers: Object.fromEntries(Object.entries(workspace.answers).map(([fieldID, value]) => [fieldID, normalizeCaptureAnswer(value)])),
    field_sequences: { ...workspace.field_sequences },
    current_revision: workspace.current_revision ? { ...workspace.current_revision } : undefined,
  };
}
