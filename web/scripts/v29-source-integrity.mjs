import { createHash } from "node:crypto";
import { readFile, readdir } from "node:fs/promises";
import path from "node:path";

const sha256 = (bytes) => createHash("sha256").update(bytes).digest("hex");

async function referenceFiles(root, relative = "") {
  const files = [];
  for (const entry of await readdir(path.join(root, relative), { withFileTypes: true })) {
    if (entry.name === "__pycache__") continue;
    const name = relative ? `${relative}/${entry.name}` : entry.name;
    if (entry.isSymbolicLink()) throw new Error(`Reference contains a symbolic link: ${name}`);
    if (entry.isDirectory()) files.push(...await referenceFiles(root, name));
    else if (entry.isFile() && name !== "MANIFEST.json") files.push(name);
    else if (!entry.isFile()) throw new Error(`Unsupported reference entry: ${name}`);
  }
  return files;
}

export async function verifyReference(root, expectedManifestHash, expectedRelease = "2.9") {
  const manifestBytes = await readFile(path.join(root, "MANIFEST.json"));
  if (sha256(manifestBytes) !== expectedManifestHash) throw new Error("Reference manifest does not match its pinned digest");
  const manifest = JSON.parse(manifestBytes.toString("utf8"));
  if (manifest.release !== expectedRelease || !Array.isArray(manifest.files) || !manifest.files.length) {
    throw new Error(`Invalid ${expectedRelease} reference manifest`);
  }
  const expected = new Map();
  for (const entry of manifest.files) {
    const name = entry.path;
    if (typeof name !== "string" || !name || name.includes("\\") || name.includes(":") || name.startsWith("/") ||
        name.split("/").some((part) => !part || part === "." || part === "..") ||
        !Number.isSafeInteger(entry.bytes) || entry.bytes < 0 || !/^[a-f0-9]{64}$/.test(entry.sha256) || expected.has(name)) {
      throw new Error(`Unsafe or duplicate reference manifest entry: ${name}`);
    }
    expected.set(name, entry);
  }
  const actual = await referenceFiles(root);
  const actualSet = new Set(actual);
  const missing = [...expected.keys()].filter((name) => !actualSet.has(name));
  const extra = actual.filter((name) => !expected.has(name));
  if (missing.length || extra.length) throw new Error(`Reference file set differs; missing: ${missing.join(", ")}; extra: ${extra.join(", ")}`);
  for (const [name, entry] of expected) {
    const bytes = await readFile(path.join(root, ...name.split("/")));
    if (bytes.length !== entry.bytes || sha256(bytes) !== entry.sha256) throw new Error(`Changed reference file: ${name}`);
  }
  return expected.size;
}
