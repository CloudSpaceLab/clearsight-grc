import type { OnboardingGuide, OnboardingState } from "../types";
import { PremiumIllustration } from "./PremiumIllustration";

type Props = { guide: OnboardingGuide; state: OnboardingState; onAdvance: (next: OnboardingState) => void; onDismiss: () => void };
export function IntroGuide({ guide, state, onAdvance, onDismiss }: Props) {
  const index = Math.min(state.current_step, guide.steps.length - 1);
  const step = guide.steps[index];
  const final = index === guide.steps.length - 1;
  return <div className="guide-backdrop"><section className="guide-card" aria-modal="true" role="dialog" aria-labelledby="guide-title">
    <button className="panel-close" onClick={onDismiss} aria-label="Skip introduction">×</button>
    <div className="guide-art"><PremiumIllustration variant="guided"/></div>
    <div className="guide-copy"><span className="eyebrow">First-time guide · {index + 1} of {guide.steps.length}</span><h2 id="guide-title">{index === 0 ? guide.title : step.title}</h2><p>{index === 0 ? guide.description : step.description}</p>
      <div className="guide-progress">{guide.steps.map((item, itemIndex) => <span key={item.id} className={itemIndex <= index ? "complete" : ""}/>)}</div>
      <div className="guide-actions"><button className="text-button" onClick={onDismiss}>Skip for now</button><button className="primary-button" onClick={() => onAdvance({ ...state, current_step: final ? guide.steps.length : index + 1, completed: final })}>{final ? "Enter workspace" : step.action ?? "Continue"}</button></div>
    </div>
  </section></div>;
}
