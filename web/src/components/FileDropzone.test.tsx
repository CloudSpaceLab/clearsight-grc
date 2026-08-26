import { render } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { FileDropzone } from "./FileDropzone";

it("distinguishes selected document rows from selected image preview rows", () => {
  const document = render(<FileDropzone label="Document" fileName="policy.pdf" fileSize={13} onSelect={vi.fn()}/>);
  expect(document.container.querySelector(".file-dropzone")?.classList.contains("has-file")).toBe(true);
  expect(document.container.querySelector(".file-dropzone")?.classList.contains("has-preview")).toBe(false);
  document.unmount();

  const image = render(<FileDropzone label="Site photo" fileName="site.png" previewUrl="blob:site" onSelect={vi.fn()}/>);
  expect(image.container.querySelector(".file-dropzone")?.classList.contains("has-preview")).toBe(true);
});
