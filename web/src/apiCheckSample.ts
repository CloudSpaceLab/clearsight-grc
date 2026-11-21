export type APICheckResultType = "boolean" | "number";

/** Validate a local example; this does not validate a live source or its authority. */
export function validateAPICheckSample(sample: string, resultType: APICheckResultType): string | null {
  if (new TextEncoder().encode(sample).length > 65536) return "Sample response is too large.";
  let value: unknown;
  try { value = JSON.parse(sample); } catch { return "Enter valid JSON."; }
  if (value === null || typeof value !== "object" || Array.isArray(value)) return "The response must be a JSON object.";
  if (hasDuplicateKeys(sample)) return "Response contains duplicate fields.";
  const response = value as Record<string, unknown>;
  if (Object.keys(response).some((key) => key !== "result" && key !== "message")) return "Response contains an unsupported field.";
  if (resultType === "boolean" && typeof response.result !== "boolean") return "result must be true or false.";
  if (resultType === "number" && (typeof response.result !== "number" || !Number.isFinite(response.result))) return "result must be a finite number.";
  if (Object.hasOwn(response, "message") && typeof response.message !== "string") return "message must be text.";
  if (typeof response.message === "string" && (Array.from(response.message).length > 1000 || response.message.includes("\0"))) return "message must be no more than 1,000 characters and cannot contain a null character.";
  return null;
}

// JSON.parse discards duplicate keys, so a contradictory result must be
// rejected before it can be treated as a valid example.
function hasDuplicateKeys(sample: string): boolean {
  let depth = 0;
  let expectingKey = false;
  const seen = new Set<string>();
  for (let index = 0; index < sample.length; index++) {
    const char = sample[index];
    if (char === '"') {
      const start = index;
      for (index++; index < sample.length; index++) {
        if (sample[index] === "\\") { index++; continue; }
        if (sample[index] === '"') break;
      }
      if (depth === 1 && expectingKey) {
        const key = JSON.parse(sample.slice(start, index + 1)) as string;
        if (seen.has(key)) return true;
        seen.add(key);
        expectingKey = false;
      }
    } else if (char === "{" || char === "[") {
      depth++;
      if (depth === 1) expectingKey = true;
    } else if (char === "}" || char === "]") {
      depth--;
    } else if (char === "," && depth === 1) {
      expectingKey = true;
    }
  }
  return false;
}
