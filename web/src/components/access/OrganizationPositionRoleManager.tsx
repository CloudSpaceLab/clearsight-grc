import { useMemo, useState } from "react";
import type {
  IdentityRole,
  OrganizationPosition,
  OrganizationPositionRoleRevision,
  ProposeOrganizationPositionRoleInput,
} from "../../identityAccessApi";
import { Button, FocusedSheet, StatusBadge, TextArea } from "../ui";

type Decision = {
  action: "approve" | "reject";
  revision: OrganizationPositionRoleRevision;
};

type Props = {
  position: OrganizationPosition;
  roles: IdentityRole[];
  revisions: OrganizationPositionRoleRevision[];
  actorPrincipalID: string;
  canConfigure: boolean;
  positionChangePending?: boolean;
  isBusy?: boolean;
  onClose: () => void;
  onPropose: (input: ProposeOrganizationPositionRoleInput) => Promise<boolean>;
  onApprove: (revision: OrganizationPositionRoleRevision, rationale: string) => Promise<boolean>;
  onReject: (revision: OrganizationPositionRoleRevision, rationale: string) => Promise<boolean>;
};

export function OrganizationPositionRoleManager({
  position,
  roles,
  revisions,
  actorPrincipalID,
  canConfigure,
  positionChangePending = false,
  isBusy = false,
  onClose,
  onPropose,
  onApprove,
  onReject,
}: Props) {
  const [decision, setDecision] = useState<Decision>();
  const [rationale, setRationale] = useState("");
  const assigned = useMemo(() => new Set(position.workspace_role_codes ?? []), [position.workspace_role_codes]);
  const pendingByRoleID = useMemo(
    () => new Map(revisions.filter((revision) => revision.status === "PENDING").map((revision) => [revision.role_template_id, revision])),
    [revisions],
  );
  const pending = revisions.filter((revision) => revision.status === "PENDING");

  async function propose(role: IdentityRole) {
    const ok = await onPropose({
      position_id: position.id,
      role_template_id: role.id,
      operation: assigned.has(role.code) ? "RETIRE" : "ADD",
      expected_position_version: position.version,
    });
    if (!ok) return;
  }

  async function decide() {
    if (!decision || !rationale.trim()) return;
    const ok = decision.action === "approve"
      ? await onApprove(decision.revision, rationale.trim())
      : await onReject(decision.revision, rationale.trim());
    if (ok) {
      setDecision(undefined);
      setRationale("");
    }
  }

  return <FocusedSheet label={"Manage roles · " + position.title} onClose={onClose} isDismissable={!isBusy}>
    <div className="cs-sheet-heading">
      <span className="eyebrow">Organization</span>
      <h2>Manage roles</h2>
      <p>{position.title} · {position.code}</p>
    </div>

    <div className="identity-scope-editor">
      {positionChangePending && <div className="inline-notice" role="status">A position change is pending. Role changes can be reviewed, but a new role change cannot be proposed yet.</div>}

      {pending.length > 0 && <div className="identity-scope-changes">
        <strong>Pending role changes</strong>
        <ul>{pending.map((revision) => {
          const own = revision.maker_id === actorPrincipalID;
          return <li key={revision.id}>
            <div>
              <b>{revision.operation === "ADD" ? "Add " : "Remove "}{revision.role_name || revision.role_code}</b>
              <span>{formatCapabilities(revision.capabilities)}</span>
            </div>
            <StatusBadge tone="warning">Pending</StatusBadge>
            {canConfigure && !own && <div className="identity-scope-change-actions">
              <Button size="compact" variant="secondary" onPress={() => { setDecision({ action: "approve", revision }); setRationale(""); }}>Approve</Button>
              <Button size="compact" variant="quiet" onPress={() => { setDecision({ action: "reject", revision }); setRationale(""); }}>Reject</Button>
            </div>}
            {own && <small>Awaiting approval</small>}
          </li>;
        })}</ul>
      </div>}

      <div className="identity-scope-current">
        <span>Workspace roles</span>
        <strong>Eligibility only</strong>
        <small>These roles control workspace capabilities. Decision, review, authorization and routing roles remain governed by their policy.</small>
      </div>

      <ul className="identity-position-role-list" aria-label="Available position roles">
        {roles.map((role) => {
          const isAssigned = assigned.has(role.code);
          const pendingRevision = pendingByRoleID.get(role.id);
          const disabled = !role.workspace_editable || Boolean(pendingRevision) || positionChangePending || isBusy;
          return <li key={role.id}>
            <span className="identity-position-role-code">{role.code}</span>
            <div className="identity-position-role-copy">
              <strong>{role.name}</strong>
              <small>{formatCapabilities(role.capabilities)}</small>
              {!role.workspace_editable && <small>{role.workspace_lock_reason || "Managed by authority policy"}</small>}
            </div>
            {role.workspace_editable
              ? <StatusBadge tone={isAssigned ? "success" : "neutral"}>{isAssigned ? "Assigned" : "Available"}</StatusBadge>
              : <StatusBadge tone="neutral">Policy managed</StatusBadge>}
            {canConfigure && role.workspace_editable && <Button
              size="compact"
              variant={isAssigned ? "quiet" : "secondary"}
              isDisabled={disabled}
              onPress={() => void propose(role)}
            >{pendingRevision ? "Pending" : isAssigned ? "Remove" : "Add"}</Button>}
          </li>;
        })}
      </ul>

      {!roles.length && <div className="identity-empty-state"><strong>No active roles</strong><span>Role templates must exist before they can be assigned to positions.</span></div>}

      {decision && <div className="identity-inline-form">
        <h4>{decision.action === "approve" ? "Approve role change" : "Reject role change"}</h4>
        <p className="muted-copy">{decision.revision.operation === "ADD" ? "Add " : "Remove "}{decision.revision.role_name || decision.revision.role_code}. Material authority remains unchanged.</p>
        <TextArea label="Rationale" value={rationale} onChange={setRationale} rows={3} maxLength={1000} isRequired/>
        <div className="identity-guard-actions">
          <Button size="compact" variant="quiet" onPress={() => { setDecision(undefined); setRationale(""); }} isDisabled={isBusy}>Cancel</Button>
          <Button
            size="compact"
            variant={decision.action === "reject" ? "destructive" : "primary"}
            onPress={() => void decide()}
            isDisabled={!rationale.trim()}
            isLoading={isBusy}
          >{decision.action === "approve" ? "Approve" : "Reject"}</Button>
        </div>
      </div>}

      <div className="identity-scope-editor-actions">
        <Button variant="secondary" onPress={onClose} isDisabled={isBusy}>Close</Button>
      </div>
    </div>
  </FocusedSheet>;
}

function formatCapabilities(capabilities: string[]) {
  return capabilities.length ? capabilities.join(" · ") : "No workspace capabilities";
}
