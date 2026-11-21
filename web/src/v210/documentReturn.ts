// Return links only accept the Documents route; source links must not become
// arbitrary navigation targets when a URL is copied or modified.
export function documentReturnDestination(): string | undefined {
  const params = new URLSearchParams(window.location.hash.split("?")[1] ?? "");
  const destination = params.get("return_to");
  return destination && destination.length <= 1200 && /^#\/documents(?:\?[^#]*)?$/.test(destination) ? destination : undefined;
}

export function withDocumentReturn(target: string): string {
  const current = window.location.hash;
  if (current.length > 1200 || !/^#\/documents(?:\?[^#]*)?$/.test(current)) return target;
  return `${target}?${new URLSearchParams({ return_to: current })}`;
}

export function preserveDocumentReturn(target: string): string {
  const destination = documentReturnDestination();
  return destination ? `${target}?${new URLSearchParams({ return_to: destination })}` : target;
}
