import { useEffect, useState } from "react";
import { loadMatterFormLinks, unlinkExistingForm, type FormSubjectLink } from "../../formSubjectLinksApi";
import { ActionLink, Button, Notice } from "../ui";
import { StatusPill } from "./dashboard/TemplateLibraryTable";
import { FormLinkPicker } from "./FormLinkPicker";
import "../../form-subject-links.css";

type Props = {
  matterID: string;
  pickerOpen: boolean;
  onClosePicker: () => void;
  onLinkedIDsChange: (ids: ReadonlySet<string>) => void;
  onRequestResponse?: (link: FormSubjectLink) => void;
};

function appendUnique(current: FormSubjectLink[], next: FormSubjectLink[]) {
  const ids = new Set(current.map((link) => link.id));
  return [...current, ...next.filter((link) => !ids.has(link.id))];
}

export function MatterFormLinksPanel({ matterID, pickerOpen, onClosePicker, onLinkedIDsChange, onRequestResponse }: Props) {
  const [links, setLinks] = useState<FormSubjectLink[]>([]);
  const [cursor, setCursor] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState("");
  const [refresh, setRefresh] = useState(0);
  const [endingID, setEndingID] = useState("");
  const [reason, setReason] = useState("");
  const [ending, setEnding] = useState(false);
  const [notice, setNotice] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    setLinks([]);
    setCursor(undefined);
    setError("");
    setLoading(true);
    void loadMatterFormLinks(matterID, undefined, controller.signal).then((page) => {
      if (controller.signal.aborted) return;
      setLinks(page.items);
      setCursor(page.next_cursor);
      setLoading(false);
    }).catch(() => {
      if (controller.signal.aborted) return;
      setError("Linked forms could not be loaded.");
      setLoading(false);
    });
    return () => controller.abort();
  }, [matterID, refresh]);

  useEffect(() => {
    onLinkedIDsChange(new Set(links.map((link) => link.form_template_id)));
  }, [links, onLinkedIDsChange]);

  async function loadMore() {
    if (!cursor || loadingMore) return;
    setLoadingMore(true);
    setError("");
    try {
      const page = await loadMatterFormLinks(matterID, cursor);
      setLinks((current) => appendUnique(current, page.items));
      setCursor(page.next_cursor);
    } catch {
      setError("More linked forms could not be loaded.");
    } finally {
      setLoadingMore(false);
    }
  }

  function onLinked(link: FormSubjectLink) {
    setLinks((current) => appendUnique(current, [link]));
    setNotice("Form linked. No request was sent.");
    onClosePicker();
  }

  async function unlink(link: FormSubjectLink) {
    if (!reason.trim() || ending) return;
    setEnding(true);
    setError("");
    try {
      await unlinkExistingForm(matterID, link, reason.trim());
      setLinks((current) => current.filter((item) => item.id !== link.id));
      setEndingID("");
      setReason("");
      setNotice("Form link ended. Request and response history is unchanged.");
    } catch {
      setError("The form link could not be ended. Refresh the list and try again.");
    } finally {
      setEnding(false);
    }
  }

  const ids = new Set(links.map((link) => link.form_template_id));
  return <>
    {notice && <Notice tone="success">{notice}</Notice>}
    {(loading || error || links.length > 0) && <div className="subject-form-activity__group subject-form-activity__card" aria-label="Linked approved forms">
      <header><h3>Linked forms</h3>{links.length > 0 && <span>{links.length} shown</span>}</header>
      {loading && <p role="status">Loading linked forms…</p>}
      {error && <Notice tone="warning">{error} <Button variant="secondary" size="compact" onPress={() => setRefresh((value) => value + 1)}>Retry</Button></Notice>}
      {links.length > 0 && <ul>{links.map((link) => <li key={link.id}>
        <div><strong>{link.form_name}</strong><span>Revision {link.form_template_version}</span></div>
        <div className="subject-form-activity__actions">
          <StatusPill status={link.form_status}/>
          <ActionLink href={`#forms/${encodeURIComponent(link.form_template_id)}`}>Open form</ActionLink>
          {onRequestResponse && link.form_status === "ACTIVE" && <Button variant="secondary" size="compact" onPress={() => onRequestResponse(link)}>Request response</Button>}
          <Button variant="secondary" size="compact" onPress={() => { setEndingID(link.id); setReason(""); setError(""); }}>Unlink</Button>
        </div>
        {endingID === link.id && <div className="form-link-retirement">
          <label>Reason for unlinking<textarea rows={2} maxLength={1000} value={reason} onChange={(event) => setReason(event.target.value)}/></label>
          <div><Button variant="secondary" size="compact" onPress={() => setEndingID("")}>Keep link</Button>
          <Button isDisabled={ending || !reason.trim()} isLoading={ending} size="compact" onPress={() => void unlink(link)}>Confirm unlink</Button></div>
        </div>}
      </li>)}</ul>}
      {cursor && !loading && <Button variant="secondary" size="compact" isLoading={loadingMore} onPress={() => void loadMore()}>Load more linked forms</Button>}
    </div>}
    {pickerOpen && <FormLinkPicker matterID={matterID} linkedFormIDs={ids} onLinked={onLinked} onClose={onClosePicker}/>}
  </>;
}
