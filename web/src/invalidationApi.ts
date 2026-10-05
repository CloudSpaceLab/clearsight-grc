const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type InvalidationEvent = {
  revision: string;
};

export function subscribeInvalidations(onInvalidate: (event: InvalidationEvent) => void): () => void {
  if (typeof EventSource === "undefined") return () => {};
  const source = new EventSource(`${apiBase}/api/v1/updates/stream`, { withCredentials: true });
  const handler = (event: MessageEvent<string>) => {
    try {
      const value = JSON.parse(event.data) as Partial<InvalidationEvent>;
      const revision = typeof value.revision === "string" ? value.revision.trim() : "";
      if (revision) onInvalidate({ revision });
    } catch {
      // Ignore malformed transport events; ordinary reads remain authoritative.
    }
  };
  source.addEventListener("invalidate", handler as EventListener);
  return () => {
    source.removeEventListener("invalidate", handler as EventListener);
    source.close();
  };
}
