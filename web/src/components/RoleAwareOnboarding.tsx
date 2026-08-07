import { useCallback, useEffect, useState } from "react";
import { loadGuideState, loadRoleGuide, saveGuideState } from "../onboardingApi";
import type { GuideStep, OnboardingGuide, OnboardingState } from "../types";
import { IntroGuide } from "./IntroGuide";

type Runtime = {
  tenant: { id: string };
  actor: { id: string; role_codes?: string[] };
};

type Props = {
  runtime: Runtime | null;
  onStep: (step: GuideStep) => void | Promise<void>;
};

export function RoleAwareOnboarding({ runtime, onStep }: Props) {
  const [guide, setGuide] = useState<OnboardingGuide | null>(null);
  const [state, setState] = useState<OnboardingState | null>(null);
  const [busy, setBusy] = useState(false);
  const [open, setOpen] = useState(false);

  const load = useCallback(async () => {
    if (!runtime) return;
    try {
      const resolved = await loadRoleGuide();
      const saved = await loadGuideState(resolved.code);
      const tourMode = new URLSearchParams(window.location.search).get("tour");
      setGuide(resolved);
      setState(saved);
      setOpen(tourMode === "on" || (tourMode !== "off" && !saved.completed && !saved.dismissed));
    } catch {
      setGuide(null);
      setState(null);
      setOpen(false);
    }
  }, [runtime]);

  useEffect(() => { void load(); }, [load]);

  async function persist(next: OnboardingState) {
    if (!guide) return;
    const saved = await saveGuideState(guide.code, next).catch(async () => {
      const current = await loadGuideState(guide.code);
      return saveGuideState(guide.code, { ...next, version: current.version });
    });
    setState(saved);
    setOpen(!saved.completed && !saved.dismissed);
  }

  async function advance(step: GuideStep, next: OnboardingState) {
    setBusy(true);
    try {
      await onStep(step);
      highlight(step.target);
      await persist(next);
    } finally {
      setBusy(false);
    }
  }

  async function back(next: OnboardingState) {
    setBusy(true);
    try {
      await persist(next);
    } finally {
      setBusy(false);
    }
  }

  async function dismiss() {
    if (!state) return;
    setBusy(true);
    try {
      await persist({ ...state, completed: false, dismissed: true });
    } finally {
      setBusy(false);
    }
  }

  async function restart() {
    if (!state) return;
    setBusy(true);
    try {
      const next = await saveGuideState(state.guide_code, { current_step: 0, completed: false, dismissed: false, version: state.version });
      setState(next);
      setOpen(true);
    } catch {
      await load();
      setOpen(true);
    } finally {
      setBusy(false);
    }
  }

  if (!guide || !state) return null;
  return <>
    <button className="guide-launcher" type="button" onClick={() => void restart()} aria-label={`Restart ${guide.role} introduction`} disabled={busy}>
      <span aria-hidden="true">?</span><strong>Guide</strong>
    </button>
    {open && <IntroGuide guide={guide} state={state} busy={busy} onAdvance={advance} onBack={back} onDismiss={dismiss}/>} 
  </>;
}

function highlight(target?: string) {
  if (!target) return;
  window.setTimeout(() => {
    const escaped = typeof CSS !== "undefined" && CSS.escape ? CSS.escape(target) : target.replace(/[^a-zA-Z0-9_-]/g, "");
    const element = document.getElementById(target) ?? document.querySelector<HTMLElement>(`.${escaped}`);
    if (!element) return;
    element.classList.add("guide-highlight");
    element.scrollIntoView({ behavior: "smooth", block: "center" });
    if (!element.hasAttribute("tabindex")) element.setAttribute("tabindex", "-1");
    element.focus({ preventScroll: true });
    window.setTimeout(() => element.classList.remove("guide-highlight"), 3200);
  }, 160);
}
