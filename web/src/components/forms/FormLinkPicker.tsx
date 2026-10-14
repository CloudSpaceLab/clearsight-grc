import { useEffect, useState } from "react";
import { loadFormTemplatePage } from "../../formsApi";
import type { FormLibraryItem } from "../../formsTypes";
import { linkExistingForm, type FormSubjectLink } from "../../formSubjectLinksApi";
import { FocusedSheet } from "../FocusedSheet";
import { ActionLink, Button, Notice } from "../ui";

type Props = {
  matterID: string;
  linkedFormIDs: ReadonlySet<string>;
  onLinked: (value: FormSubjectLink) => void;
  onClose: () => void;
};

const pageSize = 20;

function appendUnique(current: FormLibraryItem[], incoming: FormLibraryItem[]) {
  const found = new Set(current.map((item) => item.template.id));
  return [...current, ...incoming.filter((item) => !found.has(item.template.id))];
}

export function FormLinkPicker({ matterID, linkedFormIDs, onLinked, onClose }: Props) {
  const [query, setQuery] = useState("");
  const [forms, setForms] = useState<FormLibraryItem[]>([]);
  const [cursor, setCursor] = useState<string>();
  const [selectedID, setSelectedID] = useState("");
  const [loading, setLoading] = useState(true);
  const [moreBusy, setMoreBusy] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      setLoading(true);
      setSelectedID("");
      setForms([]);
      setCursor(undefined);
      setError("");
      void loadFormTemplatePage({ search: query.trim() || undefined, status: "ACTIVE", limit: pageSize }, controller.signal)
        .then((page) => {
          if (controller.signal.aborted) return;
          setForms(page.items);
          setCursor(page.next_cursor);
          setLoading(false);
        })
        .catch(() => {
          if (controller.signal.aborted) return;
          setError("Approved forms could not be loaded. Try searching again.");
          setLoading(false);
        });
    }, query ? 250 : 0);
    return () => { controller.abort(); window.clearTimeout(timer); };
  }, [query]);

  async function loadMore() {
    if (!cursor || moreBusy) return;
    setMoreBusy(true);
    setError("");
    try {
      const page = await loadFormTemplatePage({ search: query.trim() || undefined, status: "ACTIVE", limit: pageSize, cursor });
      setForms((current) => appendUnique(current, page.items));
      setCursor(page.next_cursor);
    } catch {
      setError("More forms could not be loaded. Try again.");
    } finally {
      setMoreBusy(false);
    }
  }

  async function link() {
    const form = forms.find((item) => item.template.id === selectedID);
    if (!form?.active_version || form.active_status !== "ACTIVE" || linkedFormIDs.has(form.template.id) || saving) return;
    setSaving(true);
    setError("");
    try {
      const linked = await linkExistingForm(matterID, form.template.id, form.active_version);
      onLinked(linked);
    } catch {
      setError("The form could not be linked. Check its current approval and try again.");
    } finally {
      setSaving(false);
    }
  }

  const eligible = forms.filter((item) => item.active_status === "ACTIVE" && item.active_version);
  const selected = eligible.find((item) => item.template.id === selectedID);

  return <FocusedSheet label="Link existing form" closeLabel="Close form selection" size="wide" onClose={onClose}>
    <div className="cs-sheet-heading">
      <span className="eyebrow">Forms library</span>
      <h2>Link existing form</h2>
      <p>Choose an approved form. Linking does not send a request or change the form.</p>
    </div>
    <div className="form-link-picker">
      <label className="form-link-picker__search">Search approved forms
        <input type="search" value={query} placeholder="Form name or code" onChange={(event) => setQuery(event.target.value)}/>
      </label>
      {error && <Notice tone="warning">{error}</Notice>}
      {loading ? <p role="status">Loading approved forms…</p> : eligible.length === 0
        ? <p>No approved forms found. Try another search or open <ActionLink href="#forms">Forms</ActionLink>.</p>
        : <fieldset className="form-link-picker__results"><legend>Approved forms</legend>
          {eligible.map((item) => {
            const disabled = linkedFormIDs.has(item.template.id);
            return <label key={item.template.id} className={selectedID === item.template.id ? "selected" : ""}>
              <input type="radio" name="form-link-choice" value={item.template.id} disabled={disabled}
                checked={selectedID === item.template.id} onChange={() => setSelectedID(item.template.id)}/>
              <span><strong>{item.template.name}</strong><small>Revision {item.active_version}{disabled ? " · Already linked" : ""}</small></span>
            </label>;
          })}
        </fieldset>}
      {cursor && !loading && <Button variant="secondary" size="compact" isLoading={moreBusy} onPress={() => void loadMore()}>Load more forms</Button>}
      <div className="form-link-picker__actions">
        <Button variant="secondary" onPress={onClose}>Cancel</Button>
        <Button isDisabled={!selected || saving} isLoading={saving} onPress={() => void link()}>Link form</Button>
      </div>
    </div>
  </FocusedSheet>;
}
