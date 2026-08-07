import { useEffect, useRef, useState } from "react";

type Props = {
  value: string;
  label: string;
  attestation?: string;
  onChange: (value: string) => void;
};

type Mode = "draw" | "type";

export function SignatureCapture({ value, label, attestation, onChange }: Props) {
  const [open, setOpen] = useState(false);
  const [mode, setMode] = useState<Mode>("draw");
  const [typedName, setTypedName] = useState("");
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const drawing = useRef(false);

  useEffect(() => {
    if (!open || mode !== "draw") return;
    const canvas = canvasRef.current;
    const context = canvas?.getContext("2d");
    if (!canvas || !context) return;
    context.lineCap = "round";
    context.lineJoin = "round";
    context.lineWidth = 2.2;
    context.strokeStyle = getComputedStyle(document.documentElement).getPropertyValue("--text").trim() || "#111827";
  }, [open, mode]);

  function point(event: React.PointerEvent<HTMLCanvasElement>) {
    const canvas = event.currentTarget;
    const rect = canvas.getBoundingClientRect();
    return {
      x: (event.clientX - rect.left) * (canvas.width / Math.max(rect.width, 1)),
      y: (event.clientY - rect.top) * (canvas.height / Math.max(rect.height, 1)),
    };
  }

  function startDraw(event: React.PointerEvent<HTMLCanvasElement>) {
    const context = event.currentTarget.getContext("2d");
    if (!context) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    const next = point(event);
    drawing.current = true;
    context.beginPath();
    context.moveTo(next.x, next.y);
  }

  function continueDraw(event: React.PointerEvent<HTMLCanvasElement>) {
    if (!drawing.current) return;
    const context = event.currentTarget.getContext("2d");
    if (!context) return;
    const next = point(event);
    context.lineTo(next.x, next.y);
    context.stroke();
  }

  function stopDraw(event: React.PointerEvent<HTMLCanvasElement>) {
    if (!drawing.current) return;
    drawing.current = false;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
  }

  function clearDraw() {
    const canvas = canvasRef.current;
    canvas?.getContext("2d")?.clearRect(0, 0, canvas.width, canvas.height);
  }

  function useDrawnSignature() {
    const canvas = canvasRef.current;
    if (!canvas || isCanvasBlank(canvas)) return;
    onChange(canvas.toDataURL("image/png"));
    setOpen(false);
  }

  function useTypedSignature() {
    const name = typedName.trim();
    if (!name) return;
    const canvas = document.createElement("canvas");
    canvas.width = 720;
    canvas.height = 180;
    const context = canvas.getContext("2d");
    if (!context) return;
    context.clearRect(0, 0, canvas.width, canvas.height);
    context.fillStyle = getComputedStyle(document.documentElement).getPropertyValue("--text").trim() || "#111827";
    context.font = "italic 54px Georgia, 'Times New Roman', serif";
    context.textBaseline = "middle";
    context.fillText(name, 28, canvas.height / 2, canvas.width - 56);
    onChange(canvas.toDataURL("image/png"));
    setOpen(false);
  }

  return <fieldset className="capture-field signature-field">
    <legend>{label}</legend>
    {attestation && <p className="field-help">{attestation}</p>}
    {value ? <div className="signature-preview"><img src={value} alt="Your signature"/><div><strong>Signature added</strong><button type="button" className="text-button" onClick={() => setOpen(true)}>Change</button></div></div> : <button type="button" className="secondary-button signature-add" onClick={() => setOpen(true)}>Add signature</button>}
    {open && <div className="signature-editor" role="dialog" aria-modal="false" aria-label={`Add ${label.toLowerCase()}`}>
      <div className="signature-mode" role="group" aria-label="Signature method"><button type="button" className={mode === "draw" ? "active" : ""} aria-pressed={mode === "draw"} onClick={() => setMode("draw")}>Draw</button><button type="button" className={mode === "type" ? "active" : ""} aria-pressed={mode === "type"} onClick={() => setMode("type")}>Type</button></div>
      {mode === "draw" ? <>
        <canvas ref={canvasRef} className="signature-canvas" width={720} height={180} aria-label="Draw your signature" onPointerDown={startDraw} onPointerMove={continueDraw} onPointerUp={stopDraw} onPointerCancel={stopDraw}/>
        <div className="signature-actions"><button type="button" className="text-button" onClick={clearDraw}>Clear</button><button type="button" className="primary-button" onClick={useDrawnSignature}>Use signature</button></div>
      </> : <>
        <label className="field compact"><span>Your name</span><input value={typedName} autoComplete="name" onChange={(event) => setTypedName(event.target.value)} placeholder="Type your full name"/></label>
        <div className="typed-signature-preview" aria-hidden="true">{typedName || "Your name"}</div>
        <div className="signature-actions"><button type="button" className="secondary-button" onClick={() => setOpen(false)}>Cancel</button><button type="button" className="primary-button" disabled={!typedName.trim()} onClick={useTypedSignature}>Use signature</button></div>
      </>}
    </div>}
  </fieldset>;
}

function isCanvasBlank(canvas: HTMLCanvasElement) {
  const context = canvas.getContext("2d");
  if (!context) return true;
  const pixels = context.getImageData(0, 0, canvas.width, canvas.height).data;
  for (let index = 3; index < pixels.length; index += 4) if (pixels[index] !== 0) return false;
  return true;
}
