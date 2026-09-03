import { useEffect, useRef, useState } from "react";
import {
  loadFormResponseWorkspace,
  submitFormResponseWorkspace,
  uploadCaptureSessionArtifact,
  type FormResponseWorkspace,
  type FormResponseWorkspacePayload,
  type RedeemedFormAccessSession,
} from "../captureApi";
import { CaptureRecovery } from "../captureRecovery";
import { buildCaptureRecoveryContext } from "../captureRecoveryContext";
import { IndexedDBRecoveryStore } from "../captureRecoveryStore";
import {
  CaptureWorkspaceSync,
  type CaptureWorkspaceConflictChoice,
  type CaptureWorkspaceSyncSnapshot,
} from "../captureWorkspaceSync";
import { apiErrorKind, ApiError } from "../http";
import type { CaptureAnswers, CaptureRequest } from "../types";
import { CapturePanel, type CaptureWorkspacePersistence } from "./CapturePanel";
import { CaptureWorkspaceRecoveryProvider } from "./capture/CaptureWorkspaceRecoveryContext";
import { ExternalAccessGate } from "./capture/ExternalAccessGate";
import { WorkspaceConflictPanel } from "./capture/WorkspaceConflictPanel";

type ExternalCaptureState = "access" | "loading" | "live" | "recoverable" | "terminal" | "submitted";

export function ExternalCaptureApp({ invitationToken }: { invitationToken: string }) {
  const [sessionToken, setSessionToken] = useState("");
  const [request, setRequest] = useState<CaptureRequest | null>(null);
  const [workspace, setWorkspace] = useState<FormResponseWorkspace | null>(null);
  const [syncSnapshot, setSyncSnapshot] = useState<CaptureWorkspaceSyncSnapshot | null>(null);
  const [audienceHint, setAudienceHint] = useState("");
  const [assurance, setAssurance] = useState("");
  const [panelGeneration, setPanelGeneration] = useState(0);
  const [state, setState] = useState<ExternalCaptureState>(invitationToken ? "access" : "terminal");
  const [error, setError] = useState(invitationToken ? "" : "Ask the sender for a new invitation link.");
  const [terminalTitle, setTerminalTitle] = useState("This request is no longer available");
  const syncRef = useRef<CaptureWorkspaceSync | null>(null);

  useEffect(() => {
    const flushHiddenEdits = () => {
      if (document.visibilityState === "hidden") void syncRef.current?.flush();
    };
    document.addEventListener("visibilitychange", flushHiddenEdits);
    return () => {
      document.removeEventListener("visibilitychange", flushHiddenEdits);
      syncRef.current?.dispose();
      syncRef.current = null;
    };
  }, []);

  async function openRedeemedSession(redeemed: RedeemedFormAccessSession) {
    setSessionToken(redeemed.session_token);
    setAudienceHint(redeemed.audience_hint);
    setAssurance(redeemed.assurance);
    setState("loading");
    setError("");
    try {
      const payload = await loadFormResponseWorkspace(redeemed.session_token);
      if (!resumableRequest(payload.request) || payload.workspace.workspace.status !== "OPEN") {
        endSession(
          payload.request.status === "EXPIRED" ? "This request has expired" : "This request is no longer available",
          "Ask the sender for a new invitation link.",
        );
        return;
      }
      await activateWorkspace(redeemed.session_token, payload);
      setAudienceHint(payload.session.audience_hint || redeemed.audience_hint);
      setAssurance(payload.session.assurance || redeemed.assurance);
      setState("live");
    } catch (cause) {
      if (terminalSessionFailure(cause)) {
        endSession("This request is no longer available", "Ask the sender for a new invitation link.");
        return;
      }
      setError("Check your connection, then try loading this request again.");
      setState("recoverable");
    }
  }

  async function retrySession() {
    if (!sessionToken) {
      endSession("This request is no longer available", "Open the original invitation link again.");
      return;
    }
    setState("loading");
    setError("");
    try {
      const payload = await loadFormResponseWorkspace(sessionToken);
      if (!resumableRequest(payload.request) || payload.workspace.workspace.status !== "OPEN") {
        endSession(
          payload.request.status === "EXPIRED" ? "This request has expired" : "This request is no longer available",
          "Ask the sender for a new invitation link.",
        );
        return;
      }
      const controller = syncRef.current;
      if (controller && request?.id === payload.request.id && workspace?.workspace.id === payload.workspace.workspace.id) {
        await controller.reload(payload.workspace);
        setRequest(payload.request);
        setWorkspace(controller.currentWorkspace());
        setSyncSnapshot(controller.snapshot());
      } else {
        await activateWorkspace(sessionToken, payload);
      }
      setAudienceHint(payload.session.audience_hint);
      setAssurance(payload.session.assurance);
      setState("live");
    } catch (cause) {
      if (terminalSessionFailure(cause)) {
        endSession("This request is no longer available", "Ask the sender for a new invitation link.");
        return;
      }
      setError("Check your connection, then try loading this request again.");
      setState("recoverable");
    }
  }

  async function activateWorkspace(token: string, payload: FormResponseWorkspacePayload) {
    syncRef.current?.dispose();
    syncRef.current = null;
    setSyncSnapshot(null);
    setPanelGeneration(0);

    const recoveryContext = buildCaptureRecoveryContext(
      payload,
      typeof window === "undefined" ? "" : window.location.origin,
    );
    const recovery = browserRecoveryAvailable() && recoveryContext.authorized
      ? new CaptureRecovery(new IndexedDBRecoveryStore())
      : undefined;
    const controller = await CaptureWorkspaceSync.create({
      sessionToken: token,
      fields: payload.request.fields,
      workspace: payload.workspace,
      recovery,
      recoveryContext,
      onStateChange: setSyncSnapshot,
      onError: (cause) => {
        if (terminalSessionFailure(cause)) endSession("This request is no longer available", "Ask the sender for a new invitation link.");
      },
    });
    syncRef.current = controller;
    setRequest(payload.request);
    setWorkspace(controller.currentWorkspace());
    setSyncSnapshot(controller.snapshot());
  }

  async function resolveWorkspaceConflict(fieldID: string, choice: CaptureWorkspaceConflictChoice) {
    const controller = syncRef.current;
    if (!controller) return;
    await controller.resolveConflict(fieldID, choice);
    setSyncSnapshot(controller.snapshot());
    setPanelGeneration((current) => current + 1);
  }

  function changeWorkspacePage(page: number) {
    const controller = syncRef.current;
    if (!controller) return;
    const current = controller.snapshot();
    controller.change(current.answers, current.presentationMode, page);
  }

  async function submit(answers: CaptureAnswers) {
    const controller = syncRef.current;
    if (!workspace || !sessionToken || !controller) throw new Error("Response workspace is unavailable");
    try {
      const currentSnapshot = controller.snapshot();
      controller.change(answers, currentSnapshot.presentationMode, currentSnapshot.page);
      const saved = await controller.flush();
      if (!saved) {
        if (controller.snapshot().saveState === "conflict") {
          throw new ApiError(409, "Resolve changed answers before submitting.", "workspace_conflict");
        }
        throw new ApiError(503, "Save the response before submitting.", "workspace_save_failed");
      }
      const current = controller.currentWorkspace();
      const result = await submitFormResponseWorkspace(sessionToken, { expected_version: current.workspace.version });
      if (result.workspace.status !== "COMPLETED" || !result.revision) {
        throw new ApiError(503, "The final response could not be confirmed.", "submission_unconfirmed");
      }
      await controller.clearRecovery().catch(() => undefined);
      clearAuthorityState();
      setState("submitted");
      return result.submission;
    } catch (cause) {
      if (terminalSessionFailure(cause)) {
        endSession("This request is no longer available", "Ask the sender for a new invitation link.");
      }
      throw cause;
    }
  }

  async function upload(file: File, fieldID?: string) {
    try {
      return await uploadCaptureSessionArtifact(sessionToken, file, fieldID);
    } catch (cause) {
      if (terminalSessionFailure(cause)) {
        endSession("This request is no longer available", "Ask the sender for a new invitation link.");
      }
      throw cause;
    }
  }

  function clearAuthorityState() {
    syncRef.current?.dispose();
    syncRef.current = null;
    setSyncSnapshot(null);
    setSessionToken("");
    setRequest(null);
    setWorkspace(null);
    setAudienceHint("");
    setAssurance("");
  }

  function endSession(title: string, message: string) {
    clearAuthorityState();
    setTerminalTitle(title);
    setError(message);
    setState("terminal");
  }

  const persistence: CaptureWorkspacePersistence | undefined = request && workspace && syncSnapshot && syncRef.current
    ? {
      key: `${request.id}:${workspace.workspace.id}`,
      initialAnswers: syncSnapshot.answers,
      initialPresentationMode: syncSnapshot.presentationMode,
      saveState: syncSnapshot.saveState,
      onChange: (answers, presentationMode) => syncRef.current?.change(answers, presentationMode),
      onFlush: async (answers, presentationMode) => {
        const controller = syncRef.current;
        if (!controller) return false;
        controller.change(answers, presentationMode);
        return controller.flush();
      },
      onRetry: () => void syncRef.current?.retry(),
    }
    : undefined;

  const recoveryUI = syncSnapshot ? {
    initialPage: syncSnapshot.page,
    filesToReselect: syncSnapshot.filesToReselect,
    onPageChange: changeWorkspacePage,
  } : null;

  return <main className="external-capture-shell">
    <header className="external-capture-brand"><div className="brand-mark" aria-label="ClearSight">C</div><div><strong>ClearSight</strong><span>Evidence response</span></div></header>
    {state === "access" ? <ExternalAccessGate routeSelector={invitationToken} onRedeemed={openRedeemedSession}/>
      : state === "loading" ? <section className="external-capture-entry" aria-live="polite" aria-busy="true"><span className="eyebrow">Evidence request</span><h1>Opening request</h1><p>Loading the response workspace…</p></section>
        : state === "recoverable" ? <section className="external-capture-entry" aria-labelledby="external-capture-title"><span className="eyebrow">Evidence request</span><h1 id="external-capture-title">Request could not be loaded</h1><p>{error}</p><button className="primary-button" type="button" onClick={() => void retrySession()}>Try again</button></section>
          : state === "terminal" ? <section className="external-capture-entry" aria-labelledby="external-capture-title"><span className="eyebrow">Evidence request</span><h1 id="external-capture-title">{terminalTitle}</h1><p>{error}</p></section>
            : state === "submitted" ? <section className="external-capture-entry" aria-labelledby="external-capture-title"><span className="eyebrow">Evidence request</span><h1 id="external-capture-title">Submitted</h1><p>Your evidence response was submitted for this request.</p></section>
              : request && workspace && sessionToken && persistence && recoveryUI ? <section className="external-capture-work">
                <div className="external-session-hint">Opened for {audienceHint || "invited respondent"}{assurance === "EMAIL_VERIFIED" ? " · Email verified" : ""}</div>
                <WorkspaceConflictPanel fields={request.fields} conflicts={syncSnapshot?.conflicts ?? []} onResolve={(fieldID, choice) => void resolveWorkspaceConflict(fieldID, choice)}/>
                <CaptureWorkspaceRecoveryProvider value={recoveryUI}>
                  <CapturePanel key={`${request.id}:${workspace.workspace.id}:${panelGeneration}`} request={request} external workspacePersistence={persistence} onReload={() => void retrySession()} onSubmit={(_, answers) => submit(answers)} onUploadArtifact={(_, file, fieldID) => upload(file, fieldID)}/>
                </CaptureWorkspaceRecoveryProvider>
              </section> : null}
  </main>;
}

function browserRecoveryAvailable() {
  return typeof indexedDB !== "undefined" && typeof crypto !== "undefined" && Boolean(crypto.subtle);
}

function resumableRequest(request: CaptureRequest) {
  const deadline = Date.parse(request.deadline);
  return ["READY", "IN_PROGRESS"].includes(request.status) && (!Number.isFinite(deadline) || deadline > Date.now());
}

function terminalSessionFailure(cause: unknown) {
  const kind = apiErrorKind(cause);
  return ["unauthorized", "forbidden", "not_found"].includes(kind)
    || (cause instanceof ApiError && ["request_closed", "workspace_unavailable", "session_unavailable"].includes(cause.code ?? ""));
}
