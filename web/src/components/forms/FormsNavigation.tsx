import type { ReactNode } from "react";
import { Tabs } from "../ui";

export const formsTabs = ["Templates", "Sent forms", "Responses", "Imports", "Communications"] as const;
export type FormsTab = typeof formsTabs[number];
const items = formsTabs.map((id) => ({ id, label: id }));

export function FormsNavigation({ activeTab, onChange, children }: { activeTab: FormsTab; onChange: (tab: FormsTab) => void; children: ReactNode }) {
  return <Tabs ariaLabel="Forms sections" items={items} selectedKey={activeTab} onSelectionChange={onChange}>{() => children}</Tabs>;
}
