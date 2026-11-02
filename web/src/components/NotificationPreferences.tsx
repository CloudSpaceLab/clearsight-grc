import { useEffect, useRef, useState } from "react";
import { apiErrorKind } from "../http";
import {
  loadNotificationPreferences,
  saveNotificationPreferences,
  type NotificationPreferences as NotificationPreferencesValue,
} from "../notificationPreferencesApi";
import { Button, CheckboxField, Notice, StatusBadge, TextField } from "./ui";

type State = "loading" | "live" | "saving" | "unavailable";

export function NotificationPreferences({
  load = loadNotificationPreferences,
  save = saveNotificationPreferences,
}: {
  load?: typeof loadNotificationPreferences;
  save?: typeof saveNotificationPreferences;
}) {
  const [state, setState] = useState<State>("loading");
  const [value, setValue] = useState<NotificationPreferencesValue>();
  const [draft, setDraft] = useState<NotificationPreferencesValue>();
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);
  const activeLoad = useRef<AbortController | undefined>(undefined);
  const savingFor = useRef<AbortController | undefined>(undefined);

  useEffect(() => {
    const controller = new AbortController();
    activeLoad.current = controller;
    setState("loading");
    setValue(undefined);
    setDraft(undefined);
    setError("");
    setSaved(false);
    void load(controller.signal).then((loaded) => {
      if (controller.signal.aborted) return;
      const normalized = loaded.version === 0 && loaded.time_zone === "UTC"
        ? { ...loaded, time_zone: browserTimeZone() }
        : loaded;
      setValue(loaded);
      setDraft(normalized);
      setState("live");
    }).catch(() => {
      if (!controller.signal.aborted) setState("unavailable");
    });
    return () => controller.abort();
  }, [load, reloadKey]);

  function patch(change: Partial<NotificationPreferencesValue>) {
    setDraft((current) => current ? { ...current, ...change } : current);
    setError("");
    setSaved(false);
  }

  async function persist() {
    const controller = activeLoad.current;
    if (!draft || !value || !controller || controller.signal.aborted || savingFor.current === controller) return;
    savingFor.current = controller;
    setState("saving");
    setError("");
    setSaved(false);
    try {
      const saved = await save({ ...draft, version: value.version });
      if (controller.signal.aborted || activeLoad.current !== controller) return;
      setValue(saved);
      setDraft(saved);
      setState("live");
      setSaved(true);
    } catch (error) {
      if (controller.signal.aborted || activeLoad.current !== controller) return;
      setState("live");
      setError(apiErrorKind(error) === "conflict"
        ? "Email preferences changed. Reload them before saving again."
        : "Could not save email preferences. Your changes have not been saved. Try again.");
    } finally {
      if (savingFor.current === controller) savingFor.current = undefined;
    }
  }

  return <section className="notification-preferences" aria-labelledby="notification-email-heading">
    <div className="notification-preferences__heading">
      <div>
        <strong id="notification-email-heading">Email delivery</strong>
        <span>Critical changes and daily summary.</span>
      </div>
      {draft?.critical_email_required && <StatusBadge tone="warning">Critical required</StatusBadge>}
    </div>

    {state === "loading" && <div role="status" aria-busy="true">Loading email preferences…</div>}
    {state === "unavailable" && <>
      <Notice tone="warning">Email preferences are unavailable. Try again to load your settings.</Notice>
      <div className="notification-preferences__actions"><Button onPress={() => setReloadKey((current) => current + 1)}>Try again</Button></div>
    </>}
    {draft && (state === "live" || state === "saving") && <>
    <CheckboxField
      label="Daily digest"
      description="One email with material changes and assigned work."
      isSelected={draft.daily_digest_enabled}
      isDisabled={state === "saving"}
      onChange={(selected) => patch({ daily_digest_enabled: selected })}
    />

    {draft.daily_digest_enabled && <div className="notification-preferences__grid">
      <TextField
        label="Digest time"
        type="time"
        value={minuteToTime(draft.digest_minute)}
        isDisabled={state === "saving"}
        onChange={(next) => {
          const minute = timeToMinute(next);
          if (minute !== undefined) patch({ digest_minute: minute });
        }}
      />
      <TextField
        label="Time zone"
        value={draft.time_zone}
        maxLength={64}
        isDisabled={state === "saving"}
        onChange={(time_zone) => patch({ time_zone })}
      />
    </div>}

    <CheckboxField
      label="Quiet hours"
      description="Digests pause during quiet hours; critical emails continue."
      isSelected={draft.quiet_hours_enabled}
      isDisabled={state === "saving"}
      onChange={(selected) => patch({ quiet_hours_enabled: selected })}
    />

    {draft.quiet_hours_enabled && <div className="notification-preferences__grid">
      <TextField
        label="Starts"
        type="time"
        value={minuteToTime(draft.quiet_start_minute)}
        isDisabled={state === "saving"}
        onChange={(next) => {
          const minute = timeToMinute(next);
          if (minute !== undefined) patch({ quiet_start_minute: minute });
        }}
      />
      <TextField
        label="Ends"
        type="time"
        value={minuteToTime(draft.quiet_end_minute)}
        isDisabled={state === "saving"}
        onChange={(next) => {
          const minute = timeToMinute(next);
          if (minute !== undefined) patch({ quiet_end_minute: minute });
        }}
      />
    </div>}

    {error && <Notice tone="warning">{error}</Notice>}
    {saved && <Notice tone="success">Email preferences saved.</Notice>}
    <div className="notification-preferences__actions">
      {error && <Button size="compact" onPress={() => setReloadKey((current) => current + 1)}>Reload preferences</Button>}
      <Button size="compact" onPress={() => void persist()} isLoading={state === "saving"}>Save email preferences</Button>
    </div>
    </>}
  </section>;
}

function minuteToTime(value: number) {
  const safe = Math.max(0, Math.min(1439, Math.trunc(value)));
  return `${String(Math.floor(safe / 60)).padStart(2, "0")}:${String(safe % 60).padStart(2, "0")}`;
}

function timeToMinute(value: string) {
  const match = /^(\d{2}):(\d{2})$/.exec(value);
  if (!match) return undefined;
  const hour = Number(match[1]);
  const minute = Number(match[2]);
  if (hour > 23 || minute > 59) return undefined;
  return hour * 60 + minute;
}

function browserTimeZone() {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
}
