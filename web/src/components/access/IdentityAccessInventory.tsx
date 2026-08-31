import type { GroupRoleBinding, IdentityGroup, IdentityPerson, IdentitySource } from "../../identityAccessApi";

type SignIn = { mode: string; issuer?: string; assurance_level?: string };

type Props = {
  signIn: SignIn;
  sources: IdentitySource[];
  bindings: GroupRoleBinding[];
  people: IdentityPerson[];
  groups: IdentityGroup[];
  canConfigure: boolean;
  busy: boolean;
  onAddSource: () => void;
  onAddMapping: () => void;
  onRotateSource: (source: IdentitySource) => void;
  onRevokeSource: (source: IdentitySource) => void;
  onRetireBinding: (binding: GroupRoleBinding) => void;
};

export function IdentityAccessInventory({ signIn, sources, bindings, people, groups, canConfigure, busy, onAddSource, onAddMapping, onRotateSource, onRevokeSource, onRetireBinding }: Props) {
  return <>
    <article className="config-card identity-source-card">
      <div className="section-header identity-card-header">
        <div><h3>Sign-in & provisioning</h3><p>{signIn.mode === "oidc" ? `OIDC${signIn.issuer ? ` · ${signIn.issuer}` : ""}` : humanize(signIn.mode)} · {signIn.assurance_level || "assurance not reported"}</p></div>
        {canConfigure && <button className="secondary-button" type="button" disabled={busy} onClick={onAddSource}>Add source</button>}
      </div>
      <div className="identity-list">{sources.length ? sources.map((source) => <div className="identity-row" key={source.id}>
        <div><strong>{source.code}</strong><span>{source.active_users} users · {source.active_groups} groups · {source.subject_attribute}</span></div>
        <mark>{humanize(source.status)}</mark>
        {canConfigure && source.status === "ACTIVE" && <div className="identity-row-actions">
          <button className="text-button" type="button" disabled={busy} onClick={() => onRotateSource(source)}>Rotate token</button>
          <button className="text-button danger-text" type="button" disabled={busy} onClick={() => onRevokeSource(source)}>Revoke</button>
        </div>}
      </div>) : <p className="muted-copy">No SCIM source has been configured.</p>}</div>
    </article>

    <article className="config-card">
      <div className="section-header identity-card-header">
        <div><h3>Group → role mappings</h3><p>Map directory groups to workspace roles. Approval authority is configured separately.</p></div>
        {canConfigure && <button className="secondary-button" type="button" disabled={busy} onClick={onAddMapping}>Add mapping</button>}
      </div>
      <div className="identity-list">{bindings.length ? bindings.map((binding) => <div className="identity-row" key={binding.id}>
        <div><strong>{binding.group_name} → {binding.role_code}</strong><span>{binding.department_path.length ? binding.department_path.join(" / ") : "Legal entity wide"} · {binding.legal_entity}</span></div>
        {canConfigure && <button className="text-button" disabled={busy} type="button" onClick={() => onRetireBinding(binding)}>Retire</button>}
      </div>) : <p className="muted-copy">No active directory group role mappings in this legal entity.</p>}</div>
    </article>

    <article className="config-card">
      <div className="section-header"><div><h3>People & groups</h3><p>People and groups available from connected directories.</p></div></div>
      <div className="identity-directory-columns">
        <div><h4>People</h4>{people.slice(0, 8).map((person) => <div className="identity-mini-row" key={person.id}><strong>{person.display_name}</strong><span>{person.source_code ? `${person.source_code} · ${person.source_state}` : "Local principal"}</span></div>)}{!people.length && <p className="muted-copy">No people found.</p>}</div>
        <div><h4>Groups</h4>{groups.slice(0, 8).map((group) => <div className="identity-mini-row" key={group.id}><strong>{group.display_name}</strong><span>{group.member_count} members · {group.source_code} · {group.source_state}</span></div>)}{!groups.length && <p className="muted-copy">No directory groups found.</p>}</div>
      </div>
    </article>
  </>;
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
