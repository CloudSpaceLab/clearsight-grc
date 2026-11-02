import { useCallback, useEffect, useRef, useState } from "react";
import { loadFrameworkClauseCandidates, loadProgramFrameworkMappings } from "../programFrameworkCandidatesApi";
import type { FrameworkClauseCandidatePage, ProgramFrameworkMappingPage } from "../programFrameworkCandidatesApi";

// Shared Program-standard review state. A stale request cannot replace a newer
// Program, edition or page selection.
export function useFrameworkClauseCandidates(programID: string, programVersion: number) {
  const requestID = useRef(0);
  const mappingRequestID = useRef(0);
  const [adoptionID, setAdoptionID] = useState("");
  const [page, setPage] = useState<FrameworkClauseCandidatePage | null>(null);
  const [mappingPage, setMappingPage] = useState<ProgramFrameworkMappingPage | null>(null);
  const [mappingStatus, setMappingStatus] = useState<"idle" | "loading" | "loaded" | "unavailable">("idle");
  const [status, setStatus] = useState<"idle" | "loading" | "loaded" | "error">("idle");

  useEffect(() => {
    requestID.current += 1;
    mappingRequestID.current += 1;
    setMappingPage(null);
    setMappingStatus("idle");
    setAdoptionID("");
    setPage(null);
    setStatus("idle");
    return () => { requestID.current += 1; mappingRequestID.current += 1; };
  }, [programID, programVersion]);

  const load = useCallback(async (nextAdoptionID: string, afterCode = "") => {
    const current = ++requestID.current;
    setAdoptionID(nextAdoptionID);
    setPage(null);
    setMappingPage(null);
    const mappingRequest = ++mappingRequestID.current;
    if (!nextAdoptionID) {
      setStatus("idle");
      return;
    }
    setStatus("loading");
    setMappingStatus("loading");
    void loadProgramFrameworkMappings(programID, nextAdoptionID).then((mappings) => {
      if (mappingRequest !== mappingRequestID.current) return;
      if (mappings.adoption_id !== nextAdoptionID || mappings.program_version !== programVersion) {
        setMappingStatus("unavailable");
        return;
      }
      setMappingPage(mappings);
      setMappingStatus("loaded");
    }).catch(() => {
      if (mappingRequest === mappingRequestID.current) setMappingStatus("unavailable");
    });
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

  const loadMoreMappings = useCallback(async () => {
    if (!mappingPage?.next_after_id || !adoptionID) return;
    const current = ++mappingRequestID.current;
    setMappingStatus("loading");
    try {
      const next = await loadProgramFrameworkMappings(programID, adoptionID, mappingPage.next_after_id);
      if (current !== mappingRequestID.current) return;
      if (next.program_version !== programVersion || next.adoption_id !== adoptionID) {
        setMappingStatus("unavailable");
        return;
      }
      setMappingPage({ ...next, items: [...mappingPage.items, ...next.items] });
      setMappingStatus("loaded");
    } catch {
      if (current === mappingRequestID.current) setMappingStatus("unavailable");
    }
  }, [programID, programVersion, adoptionID, mappingPage]);

  return { adoptionID, page, status, mappingPage, mappingStatus, load, loadMoreMappings };
}
