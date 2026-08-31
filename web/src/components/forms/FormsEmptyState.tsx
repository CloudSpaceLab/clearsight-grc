import type { ReactNode } from "react";
import { EmptyState } from "../ui";

type Props = {
  population: string;
  title: string;
  detail: string;
  actions?: ReactNode;
};

export function FormsEmptyState({ population, title, detail, actions }: Props) {
  return <EmptyState population={population} title={title} description={detail} action={actions}/>;
}
