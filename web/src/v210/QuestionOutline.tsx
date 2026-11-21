import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { FocusedSheet } from "./FocusedSheet";

type Question = { id: string; label: string };

export function focusAuthoringQuestion(index: number) {
  const field = document.getElementById(`v-authoring-question-${index}`);
  field?.focus({ preventScroll: true });
  field?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
}

function Outline({ questions, onChoose, showHeading = true }: { questions: Question[]; onChoose: (index: number) => void; showHeading?: boolean }) {
  return <nav className="v-panel v-question-outline" aria-label="Question outline">
    {showHeading && <h2>Questions</h2>}
    {questions.length === 0 ? <p>No questions recorded.</p> : <ol>{questions.map((question, index) => {
      const label = question.label.trim() || "Untitled question";
      return <li key={question.id}><button type="button" aria-label={`Go to question ${index + 1}: ${label}`} onClick={() => onChoose(index)}><span aria-hidden="true">{String(index + 1).padStart(2, "0")}</span><span>{label}</span></button></li>;
    })}</ol>}
  </nav>;
}

export function QuestionOutline({ questions }: { questions: Question[] }) {
  const [open, setOpen] = useState(false);
  const opener = useRef<HTMLButtonElement>(null);
  const focusTarget = useRef<number | "opener" | "main" | null>(null);
  useLayoutEffect(() => {
    if (open || focusTarget.current === null) return;
    const target = focusTarget.current; focusTarget.current = null;
    if (target === "opener") opener.current?.focus(); else if (target === "main") document.getElementById("v-main")?.focus(); else focusAuthoringQuestion(target);
  }, [open]);
  useEffect(() => {
    if (!open) return;
    const resize = () => { if (window.innerWidth > 760) { focusTarget.current = "main"; setOpen(false); } };
    window.addEventListener("resize", resize);
    return () => window.removeEventListener("resize", resize);
  }, [open]);
  return <>
    <div className="v-question-outline-desktop"><Outline questions={questions} onChoose={focusAuthoringQuestion}/></div>
    <button className="v-question-outline-open" type="button" ref={opener} aria-haspopup="dialog" aria-expanded={open} onClick={() => setOpen(true)}>Show questions</button>
    {open && <FocusedSheet title="Questions" onDismiss={() => { focusTarget.current = "opener"; setOpen(false); }}><Outline questions={questions} showHeading={false} onChoose={(index) => { focusTarget.current = index; setOpen(false); }}/></FocusedSheet>}
  </>;
}
