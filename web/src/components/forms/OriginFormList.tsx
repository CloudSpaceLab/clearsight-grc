import { useEffect, useState } from "react";
import { loadFormTemplatePage } from "../../formsApi";
import type { FormLibraryItem } from "../../formsTypes";
import type { FormOrigin } from "../../monitoringTypes";
import { ActionLink, Button, Notice } from "../ui";
import { StatusPill } from "./dashboard/TemplateLibraryTable";

type Props = {
  originType: FormOrigin["type"];
  originID: string;
  subjectLabel: string;
  limit?: number;
};

type LoadState = "loading" | "live" | "unavailable";

export function OriginFormList({ originType, originID, subjectLabel, limit = 6 }: Props) {
  const [state, setState] = useState<LoadState>("loading");
  const [items, setItems] = useState<FormLibraryItem[]>([]);
  const [cursor, setCursor] = useState<string>();
  const [loadingMore, setLoadingMore] = useState(false);
  const [pageError, setPageError] = useState("");
  const [reload, setReload] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    setItems([]);
    setCursor(undefined);
    setPageError("");

    void loadFormTemplatePage({
      origin_type: originType,
      origin_id: originID,
      limit,
    }, controller.signal).then((page) => {
      if (controller.signal.aborted) return;
      setItems(page.items);
      setCursor(page.next_cursor);
      setState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setState("unavailable");
    });

    return () => controller.abort();
  }, [limit, originID, originType, reload]);

  async function loadMore() {
    if (!cursor || loadingMore) return;
    setLoadingMore(true);
    setPageError("");
    try {
      const page = await loadFormTemplatePage({
        origin_type: originType,
        origin_id: originID,
        cursor,
        limit,
      });
      setItems((current) => appendUniqueByID(current, page.items));
      setCursor(page.next_cursor);
    } catch {
      setPageError("More linked forms could not be loaded.");
    } finally {
      setLoadingMore(false);
    }
  }

  return <section className="subject-form-activity__group" aria-label={`Forms linked to ${subjectLabel}`}>
    <header>
      <h3>Linked forms</h3>
      {state === "live" && <span>{items.length} shown</span>}
    </header>
    {state === "loading" && <p role="status">Loading linked forms…</p>}
    {state === "unavailable" && <Notice tone="warning">
      Linked forms are unavailable. Other issue work remains available.{" "}
      <Button variant="secondary" size="compact" onPress={() => setReload((value) => value + 1)}>Retry linked forms</Button>
    </Notice>}
    {state === "live" && items.length === 0 && <p>No linked forms recorded.</p>}
    {items.length > 0 && <ul>{items.map((item) => <li key={item.template.id}>
      <div>
        <strong>{item.template.name}</strong>
        <span>Revision v{item.template.version}</span>
      </div>
      <div className="subject-form-activity__actions">
        <StatusPill status={item.template.status}/>
        <ActionLink href={`#forms/${encodeURIComponent(item.template.id)}`}>Open form</ActionLink>
      </div>
    </li>)}</ul>}
    {pageError && <Notice tone="warning">
      {pageError}{" "}
      <Button variant="secondary" size="compact" isLoading={loadingMore} onPress={() => void loadMore()}>Retry linked forms</Button>
    </Notice>}
    {cursor && !pageError && <Button variant="secondary" size="compact" isLoading={loadingMore} onPress={() => void loadMore()}>Load more linked forms</Button>}
  </section>;
}

function appendUniqueByID(current: FormLibraryItem[], incoming: FormLibraryItem[]) {
  if (incoming.length === 0) return current;
  const seen = new Set(current.map((item) => item.template.id));
  return [...current, ...incoming.filter((item) => !seen.has(item.template.id))];
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}
