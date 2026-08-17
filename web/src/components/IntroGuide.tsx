import type { GuideStep, OnboardingGuide, OnboardingState } from "../types";

type Props = {
  guide: OnboardingGuide;
  state: OnboardingState;
  busy?: boolean;
  persistenceError?: string | null;
  onAdvance: (step: GuideStep, next: OnboardingState) => void | Promise<void>;
  onBack: (next: OnboardingState) => void | Promise<void>;
  onDismiss: () => void | Promise<void>;
};

export function IntroGuide({ guide, state, busy = false, onAdvance, onBack, onDismiss }: Props) {
  const index = Math.min(state.current_step, Math.max(guide.steps.length - 1, 0));
  const step = guide.steps[index] ?? guide.steps[0];
  const final = index === guide.steps.length - 1;

  if (!step) return null;
  const next: OnboardingState = {
    ...state,
    current_step: final ? guide.steps.length : index + 1,
    completed: final,
    dismissed: false,
  };
  const previous: OnboardingState = {
    ...state,
    current_step: Math.max(0, index - 1),
    completed: false,
    dismissed: false,
  };

  return <aside className="guide-panel" data-guide-profile={guide.profile ?? "general"} aria-label="Getting started">
    <button className="guide-close" type="button" onClick={() => void onDismiss()} aria-label="Dismiss guide" disabled={busy}>×</button>
    <div className="guide-copy">
      <div className="guide-meta"><span className="eyebrow">Getting started</span><span>Step {index + 1} of {guide.steps.length}</span></div>
      <h2>{step.title}</h2>
      <p>{step.description}</p>
      <progress className="guide-progress" value={index + 1} max={guide.steps.length} aria-label={`Guide progress: step ${index + 1} of ${guide.steps.length}`}/>
    </div>
    <div className="guide-actions">
      <button className="text-button" type="button" onClick={() => void onDismiss()} disabled={busy}>Dismiss</button>
      {index > 0 && <button className="secondary-button" type="button" onClick={() => void onBack(previous)} disabled={busy}>Back</button>}
      <button className="primary-button" type="button" onClick={() => void onAdvance(step, next)} disabled={busy}>{busy ? "Working…" : step.action ?? (final ? "Done" : "Continue")}</button>
    </div>
  </aside>;
}
