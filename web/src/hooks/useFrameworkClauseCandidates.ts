import { useCallback, useEffect, useRef, useState } from "react";
import { loadFrameworkClauseCandidates } from "../programFrameworkCandidatesApi";
import type { FrameworkClauseCandidatePage } from "../programFrameworkCandidatesApi";

// Shared Program-standard review state. A stale request cannot replace a newer
// Program, edition or page selection.
export function useFrameworkClauseCandidates(programID: string, programVersion: number) {
  const requestID = useRef(0);
  const [adoptionID, setAdoptionID] = useState("");
  const [page, setPage] = useState<FrameworkClauseCandidatePage | null>(null);
  const [status, setStatus] = useState<"idle" | "loading" | "loaded" | "error">("idle");

  useEffect(() => {
    requestID.current += 1;
    setAdoptionID("");
    setPage(null);
    setStatus("idle");
    return () => { requestID.current += 1; };
  }, [programID, programVersion]);

  const load = useCallback(async (nextAdoptionID: string, afterCode = "") => {
    const current = ++requestID.current;
    setAdoptionID(nextAdoptionID);
    setPage(null);
    if (!nextAdoptionID) {
      setStatus("idle");
      return;
    }
    setStatus("loading");
    try {
      const result = await loadFrameworkClauseCandidates(programID, nextAdoptionID, afterCode);
      if (requestID.current !== current) return;
      if (result.program_id !== programID || result.adoption_id !== nextAdoptionID ||
          result.program_version !== programVersion) {
        setStatus("error");
        return;
      }
      setPage(result);
      setStatus("loaded");
    } catch {
      if (requestID.current === current) setStatus("error");
    }
  }, [programID, programVersion]);

  return { adoptionID, page, status, load };
}
