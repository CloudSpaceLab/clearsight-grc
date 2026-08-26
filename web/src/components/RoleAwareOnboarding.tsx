import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { loadGuideState, loadRoleGuide, saveGuideState } from "../onboardingApi";
import type { GuideStep, GuideSurface, OnboardingGuide, OnboardingState } from "../types";
import { CinematicGuidePanel } from "./CinematicGuidePanel";
import { IntroGuide } from "./IntroGuide";

type Runtime = {
  tenant: { id: string };
  actor: { id: string; role_codes?: string[] };
};

type Props = {
  runtime: Runtime | null;
  surface: GuideSurface;
  onStep: (step: GuideStep) => void | Promise<void>;
};

export function RoleAwareOnboarding({ runtime, surface, onStep }: Props) {
  const [guide, setGuide] = useState<OnboardingGuide | null>(null);
  const [state, setState] = useState<OnboardingState | null>(null);
  const [busy, setBusy] = useState(false);
  const [open, setOpen] = useState(false);
  const [introduced, setIntroduced] = useState(false);
  const [guideError, setGuideError] = useState("");
  const loadID = useRef(0);

  const load = useCallback(async () => {
    const requestID = ++loadID.current;
    setGuide(null);
    setState(null);
    setOpen(false);
    setIntroduced(false);
    setGuideError("");
    setBusy(false);
    if (!runtime) return;
    try {
      const resolved = await loadRoleGuide(surface);
      if (requestID !== loadID.current) return;
      const saved = await loadGuideState(resolved.code);
      if (requestID !== loadID.current) return;
      const tourMode = new URLSearchParams(window.location.search).get("tour");
      setGuide(resolved);
      setState(saved);
      setIntroduced(false);
      setOpen(tourMode === "on" || (tourMode !== "off" && !saved.completed && !saved.dismissed));
    } catch {
      if (requestID !== loadID.current) return;
      setGuide(null);
      setState(null);
      setOpen(false);
    }
  }, [runtime, surface]);

  useLayoutEffect(() => { void load(); }, [load]);
  useEffect(() => () => { loadID.current += 1; }, []);

  async function persist(next: OnboardingState, epoch: number, guideCode: string) {
    let saved: OnboardingState;
    try {
      saved = await saveGuideState(guideCode, next);
    } catch {
      if (epoch !== loadID.current) return false;
      const current = await loadGuideState(guideCode);
      if (epoch !== loadID.current) return false;
      saved = await saveGuideState(guideCode, { ...next, version: current.version });
    }
    if (epoch !== loadID.current) return false;
    setState(saved);
    setOpen(!saved.completed && !saved.dismissed);
    return true;
  }

  async function advance(step: GuideStep, next: OnboardingState) {
    if (!guide) return;
    const epoch = loadID.current;
    const guideCode = guide.code;
    setGuideError("");
    setBusy(true);
    try {
      try {
        await onStep(step);
      } catch {
        if (epoch === loadID.current) setGuideError("This guide step could not be opened. Try again.");
        return;
      }
      if (epoch !== loadID.current) return;
      if (step.intent !== "open-vendor-due-diligence" && step.intent !== "open-vendor-work" && step.intent !== "open-vendor-next-action") highlight(step.target);
      try {
        await persist(next, epoch, guideCode);
      } catch {
        if (epoch === loadID.current) setGuideError("Guide progress could not be saved. Your workspace remains available; try again.");
      }
    } finally {
      if (epoch === loadID.current) setBusy(false);
    }
  }

  async function back(next: OnboardingState) {
    if (!guide) return;
    const epoch = loadID.current;
    const guideCode = guide.code;
    setGuideError("");
    setBusy(true);
    try {
      await persist(next, epoch, guideCode);
    } catch {
      if (epoch === loadID.current) setGuideError("Guide progress could not be saved. Your workspace remains available; try again.");
    } finally {
      if (epoch === loadID.current) setBusy(false);
    }
  }

  async function dismiss() {
    if (!state || !guide) return;
    const epoch = loadID.current;
    const guideCode = guide.code;
    setGuideError("");
    setBusy(true);
    setOpen(false);
    try {
      await persist({ ...state, completed: false, dismissed: true }, epoch, guideCode);
    } catch {
      if (epoch === loadID.current) setGuideError("Guide dismissal could not be saved. The guide is closed for this session; resume it to try again.");
    } finally {
      if (epoch === loadID.current) {
        setOpen(false);
        setBusy(false);
      }
    }
  }

  async function restart() {
    if (!state || !guide) return;
    const epoch = loadID.current;
    const guideCode = guide.code;
    setGuideError("");
    setBusy(true);
    try {
      const next = await saveGuideState(guideCode, { current_step: 0, completed: false, dismissed: false, version: state.version });
      if (epoch !== loadID.current) return;
      setState(next);
      setIntroduced(false);
      setOpen(true);
    } catch {
      if (epoch === loadID.current) setGuideError("The guide could not be restarted. Your workspace remains available; try again.");
    } finally {
      if (epoch === loadID.current) setBusy(false);
    }
  }

  function resume() {
    setGuideError("");
    setIntroduced(true);
    setOpen(true);
  }

  if (!guide || !state) return null;
  const completed = state.completed;
  return <>
    {guideError && <p className="inline-error" role="alert">{guideError}</p>}
    {!open && <button className="guide-launcher" type="button" onClick={completed ? () => void restart() : resume} aria-label={`${completed ? "Restart" : "Resume"} ${guide.role} guide`} disabled={busy}>
      <span aria-hidden="true">?</span><strong>{completed ? "Restart guide" : "Resume guide"}</strong>
    </button>}
    {open && !introduced && <CinematicGuidePanel
      variant={guide.surface === "VENDORS" ? "vendors" : "today"}
      role={guide.role}
      title={guide.title}
      description={guide.description}
      busy={busy}
      onStart={() => setIntroduced(true)}
      onSkip={dismiss}
    />}
    {open && introduced && <IntroGuide guide={guide} state={state} busy={busy} onAdvance={advance} onBack={back} onDismiss={dismiss}/>}
  </>;
}

function highlight(target?: string) {
  if (!target) return;
  window.setTimeout(() => {
    const escaped = typeof CSS !== "undefined" && CSS.escape ? CSS.escape(target) : target.replace(/[^a-zA-Z0-9_-]/g, "");
    const element = document.getElementById(target) ?? document.querySelector<HTMLElement>(`.${escaped}`);
    if (!element) return;
    element.classList.add("guide-highlight");
    const reducedMotion = typeof window.matchMedia === "function" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    element.scrollIntoView({ behavior: reducedMotion ? "auto" : "smooth", block: "center" });
    if (!element.hasAttribute("tabindex")) element.setAttribute("tabindex", "-1");
    element.focus({ preventScroll: true });
    window.setTimeout(() => element.classList.remove("guide-highlight"), 3200);
  }, 160);
}
