import { render, screen, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { DistributionDetail } from "../../../formsDistributionApi";
import { SentFormDetail } from "./SentFormDetail";

it("shows the named recipients and their delivery or response state without inventing a review", () => {
  const detail = {
    distribution: { id: "dist", subject_type: "PROGRAM", title: "Branch check", purpose: "Confirm controls", status: "OPEN", deadline: "2026-12-01T10:00:00Z", access_policy: "DIRECT_MAGIC_LINK", form_template_id: "form", form_template_version: 1 },
    workspace: { status: "OPEN", version: 1 },
    recipients: [
      { id: "a", role: "TO", type: "INTERNAL_PRINCIPAL", principal_id: "person-a", contact_label: "Ada Okafor", state: "COMPLETED", version: 1 },
      { id: "b", role: "TO", type: "EXTERNAL_AUDIENCE", state: "PENDING", version: 1 },
      { id: "c", role: "CC", type: "INTERNAL_PRINCIPAL", contact_label: "Bola Ade", state: "DELIVERED", version: 1 },
    ],
  } as DistributionDetail;
  render(<SentFormDetail detail={detail} onLifecycle={vi.fn()} onAmend={vi.fn()} onSupersede={vi.fn()}/>);
  const list = screen.getByRole("list", { name: "Recipients" });
  expect(within(list).getByText("Ada Okafor")).toBeTruthy();
  expect(within(list).getByText("Response submitted")).toBeTruthy();
  expect(within(list).getByText("External recipient")).toBeTruthy();
  expect(within(list).getByText("Bola Ade")).toBeTruthy();
  expect(within(list).getByText("Copy recipient")).toBeTruthy();
  expect(within(list).queryByText(/reviewed|approved/i)).toBeNull();
  expect(screen.getByText("Program")).toBeTruthy();
  expect(screen.queryByText("PROGRAM")).toBeNull();
});
