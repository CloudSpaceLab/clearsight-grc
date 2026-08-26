import { expect, it } from "vitest";

it("forces every review runner to use the managed current-build preview", async () => {
  let runtime;
  try {
    const modulePath = "./ui-review-runtime.mjs";
    runtime = await import(/* @vite-ignore */ modulePath);
  } catch {
    runtime = undefined;
  }

  expect(runtime?.reviewRunnerEnvironment).toBeTypeOf("function");
  const environment = runtime.reviewRunnerEnvironment({ PAGE_URL: "http://127.0.0.1:4173", EXISTING: "kept" }, "http://127.0.0.1:5199");
  expect(environment.PAGE_URL).toBe("http://127.0.0.1:5199");
  expect(environment.EXISTING).toBe("kept");
});
