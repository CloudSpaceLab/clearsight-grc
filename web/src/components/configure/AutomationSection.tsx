import { useEffect, useState } from "react";
import { loadAutomationPolicyPage, type RuntimeContext } from "../../api";
import { apiErrorKind } from "../../http";
import type { AutomationPolicyPage } from "../../types";
import { AutomationPolicies, type AutomationLoadState } from "../AutomationPolicies";
import { ActionLink, Button, Tabs } from "../ui";
import { automationHash, automationRoute, type AutomationRoute, type AutomationView } from "./automationRouting";

type Props = { hash: string; onNavigate: (hash: string) => void; runtimeContext?: RuntimeContext | null };
const tabs: Array<{ id: AutomationView; label: string }> = [
  { id: "automations", label: "Automations" }, { id: "schedules", label: "Schedules" },
  { id: "escalations", label: "Escalations" }, { id: "activity", label: "Activity" },
];

export function AutomationSection(props: Props) {
  const context = props.runtimeContext;
  // Retire records during render, including when a session or capability changes
  // without changing the actor. The obsolete request is also cancelled below.
  const identity = context ? JSON.stringify([context.tenant.id, context.legal_entity.id, context.actor.id, context.actor.session_id, context.capabilities?.config_read]) : "pending";
  return <AutomationWorkspace key={identity} {...props}/>;
}

function AutomationWorkspace({ hash, onNavigate, runtimeContext }: Props) {
  const canRead = runtimeContext?.capabilities?.config_read === true;
  const [snapshot, setSnapshot] = useState<{ state: AutomationLoadState; page?: AutomationPolicyPage }>({ state: !runtimeContext || canRead ? "loading" : "denied" });
  const [refresh, setRefresh] = useState(0);
  const tenantID = runtimeContext?.tenant.id;
  const route = automationRoute(hash);

  useEffect(() => {
    if (!canRead || !tenantID) return;
    const controller = new AbortController();
    let current = true;
    void loadAutomationPolicyPage(controller.signal).then((page) => {
      if (!current) return;
      if (page.tenant_id !== tenantID || page.scope !== "TENANT" || page.revision_basis !== "LATEST_PER_CODE"
        || !Array.isArray(page.items) || !Number.isInteger(page.limit) || page.limit < 1 || page.limit > 200
        || page.items.length > page.limit || page.returned_count !== page.items.length || typeof page.truncated !== "boolean") {
        setSnapshot({ state: "unavailable" });
        return;
      }
      setSnapshot({ state: "live", page });
    }).catch((error: unknown) => {
      if (!current) return;
      const kind = apiErrorKind(error);
      setSnapshot({ state: kind === "forbidden" || kind === "unauthorized" ? "denied" : "unavailable" });
    });
    return () => { current = false; controller.abort(); };
  }, [canRead, tenantID, refresh]);

  function reload() {
    setSnapshot({ state: "loading" });
    setRefresh((value) => value + 1);
  }
  function navigate(next: AutomationRoute) { onNavigate(automationHash(next)); }

  return <section className="automation-workspace" aria-labelledby="automation-heading">
    <header className="automation-workspace__heading">
      <div>
        <div className="automation-workspace__crumb"><ActionLink href="#configure">Settings</ActionLink><span aria-hidden="true">›</span><span>Automation</span></div>
        <h1 id="automation-heading">Automation &amp; governance</h1>
      </div>
      {canRead && route.tab === "automations" && <Button variant="secondary" onPress={reload} isDisabled={snapshot.state === "loading"}>Refresh policies</Button>}
    </header>
    <Tabs ariaLabel="Automation and governance views" variant="surface" items={tabs} selectedKey={route.tab} onSelectionChange={(tab) => navigate({ ...route, tab })}>
      {(view) => view === "automations" ? <>
        {runtimeContext && canRead && <p className="automation-workspace__scope">{runtimeContext.tenant.name} · Organization-wide policies</p>}
        <AutomationPolicies page={snapshot.page} state={snapshot.state} route={route} onRouteChange={navigate} onRefresh={canRead ? reload : undefined}/>
      </> : view === "schedules" ? <section className="automation-workspace__unavailable" aria-labelledby="automation-schedules-heading">
        <h2 id="automation-schedules-heading">Automation schedules are unavailable</h2>
        <p>Due runs and reminder schedules are not available in this view. Policy effective dates do not show when an action will run.</p>
        <Button variant="secondary" onPress={() => navigate({ ...route, tab: "automations" })}>Review policy revisions</Button>
      </section> : view === "escalations" ? <section className="automation-workspace__unavailable" aria-labelledby="automation-escalations-heading">
        <h2 id="automation-escalations-heading">Escalation routes</h2>
        <p>Review recipient roles, thresholds and approval requirements in People &amp; access. A notification does not authorize a decision.</p>
        <ActionLink href="#configure/access/escalation">Open escalation routes</ActionLink>
      </section> : <section className="automation-workspace__unavailable" aria-labelledby="automation-activity-heading">
        <h2 id="automation-activity-heading">Automation activity is unavailable</h2>
        <p>Execution receipts, delivery results and earlier policy revisions are not available in this view. A policy’s status does not confirm that an action ran or its outcome was verified.</p>
        <Button variant="secondary" onPress={() => navigate({ ...route, tab: "automations" })}>Review policy revisions</Button>
      </section>}
    </Tabs>
  </section>;
}
