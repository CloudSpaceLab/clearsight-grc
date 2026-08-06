import { useEffect, useRef } from "react";
import type { GuideStep, OnboardingGuide, OnboardingState } from "../types";
import { PremiumIllustration } from "./PremiumIllustration";

type Props = {
  guide: OnboardingGuide;
  state: OnboardingState;
  busy?: boolean;
  onAdvance: (step: GuideStep, next: OnboardingState) => void | Promise<void>;
  onBack: (next: OnboardingState) => void | Promise<void>;
  onDismiss: () => void | Promise<void>;
};

export function IntroGuide({ guide, state, busy = false, onAdvance, onBack, onDismiss }: Props) {
  const cardRef = useRef<HTMLElement>(null);
  const index = Math.min(state.current_step, Math.max(guide.steps.length - 1, 0));
  const step = guide.steps[index] ?? guide.steps[0];
  const final = index === guide.steps.length - 1;

  useEffect(() => {
    const card = cardRef.current;
    if (!card) return;
    const previous = document.activeElement as HTMLElement | null;
    const focusable = () => Array.from(card.querySelectorAll<HTMLElement>("button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex='-1'])"));
    focusable()[0]?.focus();
    function keydown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        event.preventDefault();
        void onDismiss();
        return;
      }
      if (event.key !== "Tab") return;
      const values = focusable();
      if (!values.length) return;
      const first = values[0];
      const last = values[values.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last?.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first?.focus();
      }
    }
    card.addEventListener("keydown", keydown);
    return () => {
      card.removeEventListener("keydown", keydown);
      previous?.focus?.();
    };
  }, [index, onDismiss]);

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

  return <div className="guide-backdrop" data-guide-profile={guide.profile ?? "general"}>
    <section ref={cardRef} className="guide-card" aria-modal="true" role="dialog" aria-labelledby="guide-title" aria-describedby="guide-description">
      <button className="panel-close" type="button" onClick={() => void onDismiss()} aria-label="Skip introduction" disabled={busy}>×</button>
      <div className="guide-art"><PremiumIllustration variant="guided"/></div>
      <div className="guide-copy">
        <span className="eyebrow">{guide.role} · Step {index + 1} of {guide.steps.length}</span>
        <h2 id="guide-title">{step.title}</h2>
        <p id="guide-description">{index === 0 ? `${guide.description} ${step.description}` : step.description}</p>
        <ol className="guide-progress" aria-label="Introduction progress">
          {guide.steps.map((item, itemIndex) => <li key={item.id} className={itemIndex < index ? "complete" : itemIndex === index ? "current" : ""}><span className="sr-only">{itemIndex < index ? "Completed" : itemIndex === index ? "Current" : "Upcoming"}: </span>{item.title}</li>)}
        </ol>
        <div className="guide-actions">
          <button className="text-button" type="button" onClick={() => void onDismiss()} disabled={busy}>Skip for now</button>
          <div className="guide-primary-actions">
            {index > 0 && <button className="secondary-button" type="button" onClick={() => void onBack(previous)} disabled={busy}>Back</button>}
            <button className="primary-button" type="button" onClick={() => void onAdvance(step, next)} disabled={busy}>{busy ? "Opening…" : final ? "Finish introduction" : step.action ?? "Continue"}</button>
          </div>
        </div>
      </div>
    </section>
  </div>;
}
