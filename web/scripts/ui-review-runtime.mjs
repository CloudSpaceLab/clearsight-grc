import net from "node:net";
import path from "node:path";
import { spawn } from "node:child_process";

export function reviewRunnerEnvironment(environment, pageURL) {
  return { ...environment, PAGE_URL: pageURL };
}

export async function startManagedPreview({ cwd, environment = process.env, config }) {
  const port = await availablePort();
  const url = `http://127.0.0.1:${port}`;
  const vite = path.join(cwd, "node_modules", "vite", "bin", "vite.js");
  const args = [vite, "preview", "--host", "127.0.0.1", "--port", String(port), "--strictPort"];
  if (config) args.push("--config", config);
  const child = spawn(process.execPath, args, {
    cwd,
    env: environment,
    stdio: ["ignore", "pipe", "pipe"],
  });
  let output = "";
  child.stdout?.on("data", (chunk) => { output += String(chunk); });
  child.stderr?.on("data", (chunk) => { output += String(chunk); });

  try {
    await waitForPreview(url, child, () => output);
  } catch (error) {
    child.kill();
    throw error;
  }

  return {
    url,
    output: () => output,
    async stop() {
      if (child.exitCode != null) return;
      child.kill();
      await Promise.race([
        new Promise((resolve) => child.once("exit", resolve)),
        new Promise((resolve) => setTimeout(resolve, 3000)),
      ]);
    },
  };
}

async function availablePort() {
  const server = net.createServer();
  await new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", resolve);
  });
  const address = server.address();
  const port = typeof address === "object" && address ? address.port : 0;
  await new Promise((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
  if (!port) throw new Error("Could not allocate a local UI review port.");
  return port;
}

async function waitForPreview(url, child, output) {
  const deadline = Date.now() + 15000;
  while (Date.now() < deadline) {
    if (child.exitCode != null) throw new Error(`Managed UI preview exited before it was ready.\n${output()}`);
    try {
      const response = await fetch(url);
      if (response.ok) return;
    } catch {
      // The preview process is still starting.
    }
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error(`Managed UI preview did not become ready at ${url}.\n${output()}`);
}
