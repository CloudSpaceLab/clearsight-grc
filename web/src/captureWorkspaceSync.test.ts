import { describe, expect, it, vi } from "vitest";
import {
  FormWorkspaceConflictError,
  type FormResponseWorkspace,
} from "./captureApi";
import { CaptureRecovery, recoveryStorageKey } from "./captureRecovery";
import { MemoryRecoveryStore } from "./captureRecoveryStore";
import { CaptureWorkspaceSync, workspaceEdits } from "./captureWorkspaceSync";
import { ApiError } from "./http";
import type { CaptureField } from "./types";

const fields: CaptureField[] = [
  { id: "note", label: "Note", type: "long_text", required: false },
  { id: "other", label: "Other", type: "short_text", required: false },
  { id: "secret", label: "Secret", type: "short_text", required: false, browser_cache_policy: "NO_BROWSER_CACHE" } as CaptureField,
  { id: "proof", label: "Proof", type: "file", required: false },
];

const recoveryContext = {
  origin: "https://forms.example.test",
  legalEntityID: "entity-1",
  distributionID: "distribution-1",
  schemaVersion: 4,
  workspaceID: "workspace-1",
  serverVersion: 1,
  authorized: true,
  deadline: "2030-01-08T00:00:00Z",
  routeExpiresAt: "2030-01-07T00:00:00Z",
} as const;

function workspace(overrides: Partial<FormResponseWorkspace> = {}): FormResponseWorkspace {
  return {
    workspace: {
      id: "workspace-1",
      distribution_id: "distribution-1",
      status: "OPEN",
      version: 1,
      created_at: "2030-01-01T00:00:00Z",
      updated_at: "2030-01-01T00:00:00Z",
    },
    answers: {},
    presentation_mode: "AUTOMATIC",
    field_sequences: {},
    ...overrides,
  };
}

describe("CaptureWorkspaceSync", () => {
  it("emits deletion edits for answers that existed on the server", () => {
    expect(workspaceEdits(workspace({
      answers: { note: { text: "server value" } },
      field_sequences: { note: 7 },
    }), {})).toEqual([{ field_id: "note", value: {}, base_sequence: 7 }]);
  });

  it("encrypts permitted scalar edits immediately and retains them when server sync is offline", async () => {
    const store = new MemoryRecoveryStore();
    const saveWorkspace = vi.fn().mockRejectedValue(new ApiError(503, "offline"));
    const controller = await CaptureWorkspaceSync.create({
      sessionToken: "memory-only-session",
      fields,
      workspace: workspace(),
      recovery: new CaptureRecovery(store),
      recoveryContext,
      debounceMs: 60_000,
      saveWorkspace,
    });

    controller.change({ note: { text: "private local answer" } }, "AUTOMATIC");
    expect(await controller.flush()).toBe(false);
    expect(controller.snapshot().saveState).toBe("saved_device");
    const raw = store.readRaw(recoveryStorageKey(recoveryContext));
    expect(raw).toBeDefined();
    expect(raw).not.toContain("private local answer");
    expect(raw).not.toContain("memory-only-session");
    controller.dispose();
  });

  it("does not claim complete device recovery for an offline NO_BROWSER_CACHE-only edit", async () => {
    const store = new MemoryRecoveryStore();
    const controller = await CaptureWorkspaceSync.create({
      sessionToken: "session",
      fields,
      workspace: workspace(),
      recovery: new CaptureRecovery(store),
      recoveryContext,
      debounceMs: 60_000,
      saveWorkspace: vi.fn().mockRejectedValue(new ApiError(503, "offline")),
    });

    controller.change({ secret: { text: "never persist me" } }, "AUTOMATIC");
    expect(await controller.flush()).toBe(false);
    expect(controller.snapshot().saveState).toBe("failed");
    expect(store.readRaw(recoveryStorageKey(recoveryContext))).not.toContain("never persist me");
    controller.dispose();
  });

  it("rebases a server-only concurrent field update without inventing a conflict", async () => {
    const store = new MemoryRecoveryStore();
    const saveWorkspace = vi.fn()
      .mockRejectedValueOnce(new FormWorkspaceConflictError({
        current_version: 2,
        changed_fields: [{ field_id: "note", server_value: { text: "server changed" }, sequence: 2 }],
      }))
      .mockResolvedValueOnce(workspace({
        workspace: { ...workspace().workspace, version: 3 },
        answers: { note: { text: "server changed" }, other: { text: "my edit" } },
        field_sequences: { note: 2, other: 1 },
      }));
    const controller = await CaptureWorkspaceSync.create({
      sessionToken: "session",
      fields,
      workspace: workspace({ answers: { note: { text: "server old" } }, field_sequences: { note: 1 } }),
      recovery: new CaptureRecovery(store),
      recoveryContext,
      debounceMs: 60_000,
      saveWorkspace,
    });

    controller.change({ note: { text: "server old" }, other: { text: "my edit" } }, "AUTOMATIC");
    expect(await controller.flush()).toBe(false);
    expect(controller.snapshot()).toMatchObject({
      answers: { note: { text: "server changed" }, other: { text: "my edit" } },
      conflicts: [],
    });
    expect(await controller.flush()).toBe(true);
    expect(saveWorkspace).toHaveBeenLastCalledWith("session", {
      expected_version: 2,
      presentation_mode: "AUTOMATIC",
      edits: [{ field_id: "other", value: { text: "my edit" }, base_sequence: 0 }],
    });
    controller.dispose();
  });

  it("surfaces a true same-field conflict and lets the respondent explicitly keep the local answer", async () => {
    const store = new MemoryRecoveryStore();
    const saveWorkspace = vi.fn()
      .mockRejectedValueOnce(new FormWorkspaceConflictError({
        current_version: 2,
        changed_fields: [{ field_id: "note", server_value: { text: "server changed" }, sequence: 2 }],
      }))
      .mockResolvedValueOnce(workspace({
        workspace: { ...workspace().workspace, version: 3 },
        answers: { note: { text: "my changed answer" } },
        field_sequences: { note: 3 },
      }));
    const controller = await CaptureWorkspaceSync.create({
      sessionToken: "session",
      fields,
      workspace: workspace({ answers: { note: { text: "server old" } }, field_sequences: { note: 1 } }),
      recovery: new CaptureRecovery(store),
      recoveryContext,
      debounceMs: 60_000,
      saveWorkspace,
    });

    controller.change({ note: { text: "my changed answer" } }, "AUTOMATIC");
    expect(await controller.flush()).toBe(false);
    expect(controller.snapshot()).toMatchObject({
      saveState: "conflict",
      conflicts: [{
        fieldID: "note",
        serverValue: { text: "server changed" },
        localValue: { text: "my changed answer" },
        localOperation: "set",
        sequence: 2,
      }],
    });

    await controller.resolveConflict("note", "local");
    expect(controller.snapshot().conflicts).toEqual([]);
    expect(await controller.flush()).toBe(true);
    expect(saveWorkspace).toHaveBeenLastCalledWith("session", {
      expected_version: 2,
      presentation_mode: "AUTOMATIC",
      edits: [{ field_id: "note", value: { text: "my changed answer" }, base_sequence: 2 }],
    });
    controller.dispose();
  });

  it("lets the respondent explicitly accept the server answer", async () => {
    const controller = await CaptureWorkspaceSync.create({
      sessionToken: "session",
      fields,
      workspace: workspace({ answers: { note: { text: "server old" } }, field_sequences: { note: 1 } }),
      recoveryContext,
      debounceMs: 60_000,
      saveWorkspace: vi.fn().mockRejectedValue(new FormWorkspaceConflictError({
        current_version: 2,
        changed_fields: [{ field_id: "note", server_value: { text: "server changed" }, sequence: 2 }],
      })),
    });

    controller.change({ note: { text: "my changed answer" } }, "AUTOMATIC");
    expect(await controller.flush()).toBe(false);
    await controller.resolveConflict("note", "server");
    expect(controller.snapshot()).toMatchObject({ answers: { note: { text: "server changed" } }, conflicts: [], saveState: "saved_server" });
    controller.dispose();
  });

  it("restores the encrypted wizard page and dirty scalar after a refresh", async () => {
    const store = new MemoryRecoveryStore();
    const offline = vi.fn().mockRejectedValue(new ApiError(503, "offline"));
    const first = await CaptureWorkspaceSync.create({
      sessionToken: "session-one",
      fields,
      workspace: workspace(),
      recovery: new CaptureRecovery(store),
      recoveryContext,
      debounceMs: 60_000,
      saveWorkspace: offline,
    });
    first.change({ note: { text: "resume me" } }, "WIZARD", 2);
    expect(await first.flush()).toBe(false);
    first.dispose();

    const restored = await CaptureWorkspaceSync.create({
      sessionToken: "session-two",
      fields,
      workspace: workspace(),
      recovery: new CaptureRecovery(store),
      recoveryContext,
      debounceMs: 60_000,
      saveWorkspace: offline,
    });
    expect(restored.snapshot()).toMatchObject({
      answers: { note: { text: "resume me" } },
      presentationMode: "WIZARD",
      page: 2,
      saveState: "saved_device",
    });
    restored.dispose();
  });

  it("requires reselection only for an unsynced file replacement and never sends it as a deletion", async () => {
    const store = new MemoryRecoveryStore();
    const server = workspace({
      answers: { proof: { artifact_ids: ["server-file"] } },
      field_sequences: { proof: 4 },
    });
    const first = await CaptureWorkspaceSync.create({
      sessionToken: "session-one",
      fields,
      workspace: server,
      recovery: new CaptureRecovery(store),
      recoveryContext,
      debounceMs: 60_000,
      saveWorkspace: vi.fn().mockRejectedValue(new ApiError(503, "offline")),
    });
    first.change({ proof: { artifact_ids: ["local-file"] } }, "AUTOMATIC");
    expect(await first.flush()).toBe(false);
    first.dispose();

    const restoredSave = vi.fn();
    const restored = await CaptureWorkspaceSync.create({
      sessionToken: "session-two",
      fields,
      workspace: server,
      recovery: new CaptureRecovery(store),
      recoveryContext,
      debounceMs: 60_000,
      saveWorkspace: restoredSave,
    });
    expect(restored.snapshot()).toMatchObject({ answers: {}, filesToReselect: ["proof"], saveState: "saved_device" });
    expect(workspaceEdits(server, restored.snapshot().answers, restored.snapshot().filesToReselect)).toEqual([]);
    expect(await restored.flush()).toBe(false);
    expect(restoredSave).not.toHaveBeenCalled();
    restored.dispose();
  });

  it("does not mark an unchanged server-confirmed file for reselection", async () => {
    const store = new MemoryRecoveryStore();
    const server = workspace({ answers: { proof: { artifact_ids: ["server-file"] } }, field_sequences: { proof: 4 } });
    const first = await CaptureWorkspaceSync.create({
      sessionToken: "session-one",
      fields,
      workspace: server,
      recovery: new CaptureRecovery(store),
      recoveryContext,
      debounceMs: 60_000,
      saveWorkspace: vi.fn(),
    });
    first.change(server.answers, "AUTOMATIC", 1);
    await new Promise((resolve) => setTimeout(resolve, 0));
    first.dispose();

    const restored = await CaptureWorkspaceSync.create({
      sessionToken: "session-two",
      fields,
      workspace: server,
      recovery: new CaptureRecovery(store),
      recoveryContext,
      debounceMs: 60_000,
      saveWorkspace: vi.fn(),
    });
    expect(restored.snapshot().filesToReselect).toEqual([]);
    expect(restored.snapshot().answers.proof).toEqual({ artifact_ids: ["server-file"] });
    restored.dispose();
  });

  it("moves to Saved to ClearSight after the server confirms an edit and only explicit clear purges recovery", async () => {
    const store = new MemoryRecoveryStore();
    const saveWorkspace = vi.fn().mockResolvedValue(workspace({
      workspace: { ...workspace().workspace, version: 2, updated_at: "2030-01-01T00:01:00Z" },
      answers: { note: { text: "synced" } },
      field_sequences: { note: 1 },
    }));
    const controller = await CaptureWorkspaceSync.create({
      sessionToken: "session",
      fields,
      workspace: workspace(),
      recovery: new CaptureRecovery(store),
      recoveryContext,
      debounceMs: 60_000,
      saveWorkspace,
    });

    controller.change({ note: { text: "synced" } }, "AUTOMATIC");
    expect(await controller.flush()).toBe(true);
    expect(controller.snapshot().saveState).toBe("saved_server");
    expect(store.readRaw(recoveryStorageKey(recoveryContext))).toBeDefined();
    await controller.clearRecovery();
    expect(store.readRaw(recoveryStorageKey(recoveryContext))).toBeUndefined();
    controller.dispose();
  });
});
