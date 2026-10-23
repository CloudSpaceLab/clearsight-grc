import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { routeHash } from "../../appRouting";
import { riskLossDestinations } from "../../riskLossNavigation";
import { RiskLossNavigation } from "./RiskLossNavigation";

afterEach(() => { cleanup(); window.location.hash = ""; });

it("uses the approved shared destinations in source order", () => {
  render(<RiskLossNavigation selectedKey="rcsa"/>);
  const menu = screen.getByRole("navigation", { name: "Risk & Losses sections" });
  expect(within(menu).getAllByRole("button").map((item) => item.textContent)).toEqual(riskLossDestinations.map((item) => item.label));
  expect(within(menu).getByRole("button", { name: "RCSA" }).getAttribute("aria-current")).toBe("page");
});

it.each(riskLossDestinations)("navigates to the existing $label route", ({ label, view }) => {
  render(<RiskLossNavigation selectedKey={view === "rcsa" ? "risks" : "rcsa"}/>);
  fireEvent.click(within(screen.getByRole("navigation", { name: "Risk & Losses sections" })).getByRole("button", { name: label }));
  expect(window.location.hash).toBe(routeHash(view, {}, "assigned"));
});

it("exposes the selected section through the compact selector", async () => {
  render(<RiskLossNavigation selectedKey="losses"/>);
  fireEvent.click(screen.getByRole("button", { name: /Risk & Losses section/ }));
  fireEvent.click(await screen.findByRole("option", { name: "RCSA" }));
  expect(window.location.hash).toBe("#rcsa");
});
