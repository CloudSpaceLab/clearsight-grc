import {
  FormWorkspaceConflictError,
  normalizeCaptureAnswer,
  saveFormResponseWorkspace,
  type FormResponseWorkspace,
  type FormWorkspaceConflict,
  type FormWorkspaceEditInput,
  type SaveFormResponseWorkspaceInput,
} from "./captureApi";
import {
  CaptureRecovery,
  rebaseRecoveryEnvelope,
  type CaptureRecoveryContext,
  type RecoveryDeltaOperation,
  type RecoverySaveResult,
} from "./captureRecovery";
import { apiErrorKind } from "./http";
import type { CaptureAnswerValue, CaptureAnswers, CaptureField, CapturePresentationMode } from "./types";

export type CaptureWorkspaceSaveState = "saved_server" | "saving" | "saved_device" | "conflict" | "failed";
export type CaptureWorkspaceConflictChoice = "server" | "local";

export type CaptureWorkspaceConflict = {
  fieldID: string;
  serverValue: CaptureAnswerValue;
  localValue: CaptureAnswerValue;
  localOperation: RecoveryDeltaOperation;
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
  filesToReselect: string[];
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
  private readonly sessionFieldIDs = new Set<string>();
  private saveState: CaptureWorkspaceSaveState = "saved_server";
  private localRecoveryComplete = false;
  private localWrite: Promise<void> = Promise.resolve();
  private flushPromise: Promise<boolean> | null = null;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private disposed = false;

  private constructor(options: CaptureWorkspaceSyncOptions) {
    this.sessionToken = options.sessionToken;
    this.fields = options.fields;
    this.serverWorkspace = cloneWorkspace(options.workspace);
    this.latestAnswers = cloneAnswers(options.workspace.answers);
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
      answers: cloneAnswers(this.latestAnswers),
      presentationMode: this.latestMode,
      page: this.page,
      saveState: this.saveState,
      filesToReselect: [...this.filesToReselect],
      conflicts: this.conflicts.map((conflict) => ({
        ...conflict,
        serverValue: cloneAnswer(conflict.serverValue),
        localValue: cloneAnswer(conflict.localValue),
      })),
    };
  }

  currentWorkspace(): FormResponseWorkspace {
    return cloneWorkspace(this.serverWorkspace);
  }

  change(answers: CaptureAnswers, presentationMode: CapturePresentationMode, page = this.page): void {
    if (this.disposed) return;
    const nextPage = Math.max(0, Math.trunc(page));
    const pageChanged = nextPage !== this.page;
    this.latestAnswers = cloneAnswers(answers);
    this.latestMode = presentationMode;
    this.page = nextPage;
    this.clearCompletedReselections();
    this.localSequence += 1;
    const snapshot = this.localSnapshot();
    if (pageChanged) this.emit();

    if (!this.recovery) {
      if (this.conflicts.length === 0 && this.hasUnsyncedChanges()) this.setSaveState("saving");
      this.scheduleFlush();
      return;
    }

    this.localWrite = this.localWrite.catch(() => undefined).then(async () => {
      const result = await this.persistSnapshot(snapshot);
      this.localRecoveryComplete = Boolean(result?.complete);
      if (!this.disposed && this.conflicts.length === 0 && snapshot.localSequence === this.localSequence) {
        this.setSaveState(this.saveStateForPendingChanges());
      }
    }).catch((error) => {
      this.localRecoveryComplete = false;
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
    if (this.filesToReselect.length > 0) {
      this.setSaveState(this.localRecoveryComplete ? "saved_device" : "failed");
      return false;
    }
    if (this.flushPromise) {
      const previous = await this.flushPromise;
      if (!previous || this.conflicts.length > 0 || this.filesToReselect.length > 0) return false;
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

  async reload(workspace: FormResponseWorkspace): Promise<void> {
    if (this.disposed) return;
    this.clearTimer();
    await this.localWrite.catch(() => undefined);

    const previous = this.serverWorkspace;
    const local = this.localSnapshot();
    const claimed = new Set(this.sessionFieldIDs);
    for (const fieldID of new Set([...Object.keys(previous.answers), ...Object.keys(local.answers)])) {
      if (!sameOptionalAnswer(previous.answers[fieldID], local.answers[fieldID])) claimed.add(fieldID);
    }
    for (const fieldID of local.filesToReselect) claimed.add(fieldID);
    for (const conflict of this.conflicts) claimed.add(conflict.fieldID);

    const nextServer = cloneWorkspace(workspace);
    const answers = cloneAnswers(nextServer.answers);
    const nextConflicts: CaptureWorkspaceConflict[] = [];
    for (const fieldID of claimed) {
      const localValue = local.answers[fieldID];
      const serverValue = nextServer.answers[fieldID];
      const awaitingReselection = local.filesToReselect.includes(fieldID);
      const operation: RecoveryDeltaOperation = awaitingReselection
        ? "reselect"
        : hasAnswer(localValue) ? "set" : "delete";
      const serverChanged = (nextServer.field_sequences[fieldID] ?? 0) !== (previous.field_sequences[fieldID] ?? 0);

      if (serverChanged && !localOperationSatisfied(operation, localValue, serverValue ?? {})) {
        nextConflicts.push({
          fieldID,
          serverValue: serverValue ? cloneAnswer(serverValue) : {},
          localValue: localValue ? cloneAnswer(localValue) : {},
          localOperation: operation,
          sequence: nextServer.field_sequences[fieldID] ?? 0,
        });
      }
      if (operation === "delete") delete answers[fieldID];
      else if (hasAnswer(localValue)) answers[fieldID] = cloneAnswer(localValue ?? {});
    }

    this.serverWorkspace = nextServer;
    this.recoveryContext = { ...this.recoveryContext, serverVersion: nextServer.workspace.version };
    this.latestAnswers = answers;
    this.latestMode = local.presentationMode;
    this.page = local.page;
    this.filesToReselect = [...local.filesToReselect];
    this.conflicts = nextConflicts;
    this.localSequence += 1;
    await this.persistCurrentAgainstServer();
    this.setSaveState(this.conflicts.length > 0 ? "conflict" : this.saveStateForPendingChanges());
    this.emit();
    if (this.conflicts.length === 0 && this.hasUnsyncedChanges()) this.scheduleFlush();
  }

  async resolveConflict(fieldID: string, choice: CaptureWorkspaceConflictChoice): Promise<void> {
    if (this.disposed) return;
    const conflict = this.conflicts.find((item) => item.fieldID === fieldID);
    if (!conflict) return;

    if (choice === "server") {
      this.filesToReselect = this.filesToReselect.filter((id) => id !== fieldID);
      if (hasAnswer(conflict.serverValue)) this.latestAnswers[fieldID] = cloneAnswer(conflict.serverValue);
      else delete this.latestAnswers[fieldID];
    } else if (conflict.localOperation === "reselect") {
      delete this.latestAnswers[fieldID];
      if (!this.filesToReselect.includes(fieldID)) this.filesToReselect = [...this.filesToReselect, fieldID].sort();
    } else if (conflict.localOperation === "delete") {
      delete this.latestAnswers[fieldID];
    } else if (hasAnswer(conflict.localValue)) {
      this.latestAnswers[fieldID] = cloneAnswer(conflict.localValue);
    }

    this.conflicts = this.conflicts.filter((item) => item.fieldID !== fieldID);
    this.localSequence += 1;
    await this.persistCurrentAgainstServer();
    if (this.conflicts.length > 0) {
      this.setSaveState("conflict");
    } else {
      this.setSaveState(this.saveStateForPendingChanges());
      this.scheduleFlush();
    }
    this.emit();
  }

  async clearRecovery(): Promise<void> {
    this.clearTimer();
    await this.localWrite.catch(() => undefined);
    if (this.recovery) await this.recovery.clear(this.recoveryContext);
    this.localRecoveryComplete = false;
  }

  dispose(): void {
    this.disposed = true;
    this.clearTimer();
  }

  private async restore(): Promise<void> {
    if (!this.recovery) return;
    const restored = await this.recovery.restore(this.recoveryContext);
    if (restored.status !== "restored") return;

    const rebased = rebaseRecoveryEnvelope({
      answers: this.serverWorkspace.answers,
      fieldSequences: this.serverWorkspace.field_sequences,
      presentationMode: this.serverWorkspace.presentation_mode,
    }, restored.envelope);
    this.localSequence = restored.envelope.localSequence;
    this.page = rebased.page;
    this.latestAnswers = rebased.answers;
    this.latestMode = rebased.presentationMode;
    this.filesToReselect = rebased.filesToReselect;
    this.conflicts = rebased.conflicts.map((conflict) => ({
      fieldID: conflict.fieldID,
      serverValue: conflict.serverValue,
      localValue: conflict.localValue,
      localOperation: conflict.operation,
      sequence: conflict.sequence,
    }));
    this.localRecoveryComplete = restored.envelope.complete;
    this.saveState = this.conflicts.length > 0 ? "conflict" : this.saveStateForPendingChanges();
  }

  private async performFlush(): Promise<boolean> {
    const sent = this.localSnapshot();
    const edits = workspaceEdits(this.serverWorkspace, sent.answers, sent.filesToReselect);
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
      for (const edit of edits) this.sessionFieldIDs.add(edit.field_id);
      this.serverWorkspace = cloneWorkspace(saved);
      this.recoveryContext = { ...this.recoveryContext, serverVersion: saved.workspace.version };
      await this.persistCurrentAgainstServer();
      if (sameLocalSnapshot(sent, this.localSnapshot())) {
        this.setSaveState("saved_server");
      } else {
        this.setSaveState(this.saveStateForPendingChanges());
        this.scheduleFlush();
      }
      return true;
    } catch (error) {
      if (this.disposed) return false;
      if (error instanceof FormWorkspaceConflictError) {
        this.applyServerConflict(error.conflict);
        await this.persistCurrentAgainstServer();
        if (this.conflicts.length > 0) {
          this.setSaveState("conflict");
        } else {
          this.setSaveState(this.saveStateForPendingChanges());
          this.scheduleFlush();
        }
        return false;
      }
      const kind = apiErrorKind(error);
      this.setSaveState(this.localRecoveryComplete && kind !== "validation" ? "saved_device" : "failed");
      this.onError?.(error);
      return false;
    }
  }

  private persistSnapshot(snapshot: LocalSnapshot): Promise<RecoverySaveResult | undefined> {
    if (!this.recovery) return Promise.resolve(undefined);
    return this.recovery.save(
      { ...this.recoveryContext, serverVersion: this.serverWorkspace.workspace.version },
      this.fields,
      {
        answers: this.serverWorkspace.answers,
        fieldSequences: this.serverWorkspace.field_sequences,
        presentationMode: this.serverWorkspace.presentation_mode,
        serverVersion: this.serverWorkspace.workspace.version,
      },
      {
        answers: snapshot.answers,
        presentationMode: snapshot.presentationMode,
        page: snapshot.page,
        filesToReselect: snapshot.filesToReselect,
        localSequence: snapshot.localSequence,
      },
    );
  }

  private async persistCurrentAgainstServer(): Promise<void> {
    if (!this.recovery) {
      this.localRecoveryComplete = false;
      return;
    }
    try {
      const result = await this.persistSnapshot(this.localSnapshot());
      this.localRecoveryComplete = Boolean(result?.complete);
    } catch (error) {
      this.localRecoveryComplete = false;
      this.onError?.(error);
    }
  }

  private applyServerConflict(conflict: FormWorkspaceConflict): void {
    const previousWorkspace = this.serverWorkspace;
    const answers = cloneAnswers(previousWorkspace.answers);
    const sequences = { ...previousWorkspace.field_sequences };
    const nextConflicts: CaptureWorkspaceConflict[] = [];

    for (const changed of conflict.changed_fields) {
      const oldServerValue = previousWorkspace.answers[changed.field_id];
      const localValue = this.latestAnswers[changed.field_id];
      const awaitingReselection = this.filesToReselect.includes(changed.field_id);
      const localOperation: RecoveryDeltaOperation | null = awaitingReselection
        ? "reselect"
        : sameOptionalAnswer(oldServerValue, localValue)
          ? null
          : hasAnswer(localValue) ? "set" : "delete";
      const serverValue = normalizeCaptureAnswer(changed.server_value);

      if (hasAnswer(serverValue)) answers[changed.field_id] = cloneAnswer(serverValue);
      else delete answers[changed.field_id];
      sequences[changed.field_id] = changed.sequence;

      if (localOperation === null || localOperationSatisfied(localOperation, localValue, serverValue)) {
        if (hasAnswer(serverValue)) this.latestAnswers[changed.field_id] = cloneAnswer(serverValue);
        else delete this.latestAnswers[changed.field_id];
        if (awaitingReselection) this.filesToReselect = this.filesToReselect.filter((id) => id !== changed.field_id);
        continue;
      }

      nextConflicts.push({
        fieldID: changed.field_id,
        serverValue: cloneAnswer(serverValue),
        localValue: localValue ? cloneAnswer(localValue) : {},
        localOperation,
        sequence: changed.sequence,
      });
    }

    this.serverWorkspace = {
      ...previousWorkspace,
      workspace: { ...previousWorkspace.workspace, version: conflict.current_version },
      answers,
      field_sequences: sequences,
    };
    this.recoveryContext = { ...this.recoveryContext, serverVersion: conflict.current_version };
    this.conflicts = nextConflicts;
  }

  private clearCompletedReselections(): void {
    if (this.filesToReselect.length === 0) return;
    this.filesToReselect = this.filesToReselect.filter((fieldID) => {
      const local = this.latestAnswers[fieldID];
      const server = this.serverWorkspace.answers[fieldID];
      return !hasFileReference(local) || sameOptionalAnswer(server, local);
    });
  }

  private hasUnsyncedChanges(): boolean {
    return this.filesToReselect.length > 0
      || workspaceEdits(this.serverWorkspace, this.latestAnswers, this.filesToReselect).length > 0
      || this.latestMode !== this.serverWorkspace.presentation_mode;
  }

  private saveStateForPendingChanges(): CaptureWorkspaceSaveState {
    if (!this.hasUnsyncedChanges()) return "saved_server";
    return this.localRecoveryComplete ? "saved_device" : "failed";
  }

  private localSnapshot(): LocalSnapshot {
    return {
      answers: cloneAnswers(this.latestAnswers),
      presentationMode: this.latestMode,
      page: this.page,
      localSequence: this.localSequence,
      filesToReselect: [...this.filesToReselect],
    };
  }

  private scheduleFlush(): void {
    this.clearTimer();
    if (this.disposed || this.conflicts.length > 0 || this.filesToReselect.length > 0) return;
    if (!this.hasUnsyncedChanges()) return;
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

export function workspaceEdits(workspace: FormResponseWorkspace, answers: CaptureAnswers, filesToReselect: readonly string[] = []): FormWorkspaceEditInput[] {
  const waiting = new Set(filesToReselect);
  const fieldIDs = new Set([...Object.keys(workspace.answers), ...Object.keys(answers)]);
  return [...fieldIDs].sort().flatMap((fieldID) => {
    if (waiting.has(fieldID)) return [];
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

function sameLocalSnapshot(left: LocalSnapshot, right: LocalSnapshot): boolean {
  return left.presentationMode === right.presentationMode
    && left.page === right.page
    && left.localSequence === right.localSequence
    && JSON.stringify(left.answers) === JSON.stringify(right.answers)
    && JSON.stringify(left.filesToReselect) === JSON.stringify(right.filesToReselect);
}

function localOperationSatisfied(operation: RecoveryDeltaOperation, localValue: CaptureAnswerValue | undefined, serverValue: CaptureAnswerValue): boolean {
  if (operation === "delete") return !hasAnswer(serverValue);
  if (operation === "set" && localValue) return sameOptionalAnswer(localValue, serverValue);
  return false;
}

function sameOptionalAnswer(left?: CaptureAnswerValue, right?: CaptureAnswerValue): boolean {
  if (left === undefined || right === undefined) return !hasAnswer(left) && !hasAnswer(right);
  return JSON.stringify(normalizeComparable(left)) === JSON.stringify(normalizeComparable(right));
}

function normalizeComparable(value: CaptureAnswerValue): CaptureAnswerValue {
  return cloneAnswer(value);
}

function hasAnswer(value?: CaptureAnswerValue): boolean {
  return Boolean(value?.text || value?.values?.length || value?.artifact_ids?.length || value?.document);
}

function hasFileReference(value?: CaptureAnswerValue): boolean {
  return Boolean(value?.artifact_ids?.length || value?.document?.artifact_id);
}

function cloneAnswer(value: CaptureAnswerValue): CaptureAnswerValue {
  return {
    ...(value.text !== undefined ? { text: value.text } : {}),
    ...(value.values !== undefined ? { values: [...value.values] } : {}),
    ...(value.artifact_ids !== undefined ? { artifact_ids: [...value.artifact_ids] } : {}),
    ...(value.document !== undefined ? { document: { ...value.document } } : {}),
  };
}

function cloneAnswers(answers: CaptureAnswers): CaptureAnswers {
  return Object.fromEntries(Object.entries(answers).map(([fieldID, value]) => [fieldID, cloneAnswer(value)]));
}

function cloneWorkspace(workspace: FormResponseWorkspace): FormResponseWorkspace {
  return {
    ...workspace,
    workspace: { ...workspace.workspace },
    answers: cloneAnswers(workspace.answers),
    field_sequences: { ...workspace.field_sequences },
    current_revision: workspace.current_revision ? { ...workspace.current_revision } : undefined,
  };
}
