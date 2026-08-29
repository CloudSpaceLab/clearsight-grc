import { describe, expect, it, vi } from "vitest";
import {
  FormWorkspaceConflictError,
  type FormResponseWorkspace,
} from "./captureApi";
import { CaptureRecovery, recoveryStorageKey, sanitizeRecoveryAnswers } from "./captureRecovery";
import { MemoryRecoveryStore } from "./captureRecoveryStore";
import { CaptureWorkspaceSync, workspaceEdits } from "./captureWorkspaceSync";
import { ApiError } from "./http";
import type { CaptureField } from "./types";

const fields: CaptureField[] = [
  { id: "note", label: "Note", type: "long_text", required: false },
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
    const edits = workspaceEdits(workspace({
      answers: { note: { text: "server value" } },
      field_sequences: { note: 7 },
    }), {});

    expect(edits).toEqual([{ field_id: "note", value: {}, base_sequence: 7 }]);
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
    expect(JSON.stringify(raw)).not.toContain("private local answer");
    expect(JSON.stringify(raw)).not.toContain("memory-only-session");
    controller.dispose();
  });

  it("surfaces structured server conflicts without last-write-wins", async () => {
    const store = new MemoryRecoveryStore();
    const saveWorkspace = vi.fn().mockRejectedValue(new FormWorkspaceConflictError({
      current_version: 2,
      changed_fields: [{ field_id: "note", server_value: { text: "server changed" }, sequence: 2 }],
    }));
    const controller = await CaptureWorkspaceSync.create({
      sessionToken: "session",
      fields,
      workspace: workspace({ field_sequences: { note: 1 } }),
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
        sequence: 2,
      }],
    });
    expect(controller.currentWorkspace().workspace.version).toBe(2);
    controller.dispose();
  });

  it("moves to Saved to ClearSight state after the shared workspace confirms the edit", async () => {
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
    expect(controller.currentWorkspace().workspace.version).toBe(2);
    controller.dispose();
  });
});

describe("recovery cache policy", () => {
  it("omits NO_BROWSER_CACHE fields while retaining a file reselection marker for ordinary file fields", () => {
    expect(sanitizeRecoveryAnswers(fields, {
      note: { text: "safe scalar" },
      secret: { text: "never cache me" },
      proof: { artifact_ids: ["artifact-1"] },
    })).toEqual({
      answers: { note: { text: "safe scalar" } },
      filesToReselect: ["proof"],
    });
  });
});
