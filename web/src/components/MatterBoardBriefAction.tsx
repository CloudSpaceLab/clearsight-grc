import { useEffect, useState } from "react";
import { createReportRun, getMatterBoardBriefAvailability } from "../reportingApi";
import type { MatterBoardBriefAvailability } from "../reportingTypes";
import { ActionLink, Button, Notice } from "./ui";

type Props = {
  matterID: string;
};

type LoadState = "loading" | "live" | "unavailable";

export function MatterBoardBriefAction({ matterID }: Props) {
  const [state, setState] = useState<LoadState>("loading");
  const [availability, setAvailability] = useState<MatterBoardBriefAvailability>();
  const [generating, setGenerating] = useState(false);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    setAvailability(undefined);
    setNotice("");
    setError("");
    void getMatterBoardBriefAvailability(matterID, controller.signal)
      .then((value) => {
        if (controller.signal.aborted) return;
        setAvailability(value);
        setState("live");
      })
      .catch(() => {
        if (controller.signal.aborted) return;
        setState("unavailable");
      });
    return () => controller.abort();
  }, [matterID]);

  if (state === "loading") return null;
  if (state === "unavailable") {
    return <p className="matter-board-brief-state">Board brief availability could not be checked.</p>;
  }
  if (!availability?.definition) return null;
  if (!availability.authority_available) {
    return <p className="matter-board-brief-state">{availability.reason || "Board brief authority could not be checked."}</p>;
  }
  if (!availability.can_run) {
    return <p className="matter-board-brief-state">{availability.reason || "Board brief generation is not assigned to you."}</p>;
  }

  async function generate() {
    const definition = availability?.definition;
    if (!definition || generating) return;
    setGenerating(true);
    setError("");
    setNotice("");
    try {
      await createReportRun(definition.id, definition.current_version);
      setNotice("Board brief queued.");
    } catch {
      setError("Board brief could not be queued.");
    } finally {
      setGenerating(false);
    }
  }

  return <div className="matter-board-brief-action">
    <Button variant="secondary" onPress={() => void generate()} isLoading={generating}>Generate board brief</Button>
    {notice && <Notice tone="success">{notice} <ActionLink href="#reports">Open Reports</ActionLink></Notice>}
    {error && <Notice tone="error">{error}</Notice>}
  </div>;
}
