/** A normalized concern score is not a probability or a native measurement. */
export function concernScoreText(
  score: number | null | undefined,
  presentation: "labelled" | "compact" = "labelled",
): string {
  if (score == null) return "Not assessed";
  if (!Number.isFinite(score) || score < 0 || score > 100) return "Score unavailable";
  // Preserve the supplied score instead of rounding it across a band boundary.
  return `${score} / 100${presentation === "labelled" ? " concern points" : ""}`;
}
