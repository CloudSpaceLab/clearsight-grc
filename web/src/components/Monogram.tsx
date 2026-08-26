type Props = { name: string; decorative?: boolean; className?: string };

export function initials(value: string) {
  const parts = value.trim().split(/\s+/).filter(Boolean);
  if (!parts.length) return "—";
  const first = parts[0]?.at(0) ?? "";
  const last = parts.length > 1 ? parts.at(-1)?.at(0) ?? "" : parts[0]?.at(1) ?? "";
  return `${first}${last}`.toUpperCase();
}

export function Monogram({ name, decorative = false, className = "" }: Props) {
  return <span className={className} aria-hidden={decorative ? "true" : undefined} aria-label={decorative ? undefined : `${name} initials`}>{initials(name)}</span>;
}
