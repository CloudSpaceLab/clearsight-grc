import type { DocumentImport, ProposalStatus } from "./documentTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

async function parse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { message?: string; error?: { message?: string } } | null;
    throw new Error(body?.error?.message ?? body?.message ?? `Request failed with ${response.status}`);
  }
  return (await response.json()) as T;
}

export async function loadDocumentImports(): Promise<DocumentImport[]> {
  return (await parse<{ items: DocumentImport[] }>(await fetch(`${apiBase}/api/v1/document-imports?limit=50`))).items;
}

export async function loadDocumentImport(id: string): Promise<DocumentImport> {
  return parse<DocumentImport>(await fetch(`${apiBase}/api/v1/document-imports/${encodeURIComponent(id)}`));
}

export async function importDocument(file: File, purpose: string, sourceType: string): Promise<DocumentImport> {
  const body = new FormData();
  body.set("file", file);
  body.set("purpose", purpose);
  body.set("source_type", sourceType);
  return parse<DocumentImport>(await fetch(`${apiBase}/api/v1/document-imports`, { method: "POST", body }));
}

export async function reviewDocumentProposal(documentID: string, proposalID: string, status: ProposalStatus, expectedVersion: number, note = ""): Promise<DocumentImport> {
  return parse<DocumentImport>(await fetch(`${apiBase}/api/v1/document-imports/${encodeURIComponent(documentID)}/proposals/${encodeURIComponent(proposalID)}/review`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ status, note, expected_version: expectedVersion }),
  }));
}
