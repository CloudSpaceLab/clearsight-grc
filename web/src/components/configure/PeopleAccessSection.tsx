import { IdentityAccessPanel } from "../IdentityAccessPanel";

export function PeopleAccessSection() {
  return <section className="configure-domain" aria-labelledby="organization-access-heading">
    <header className="configure-domain-header">
      <div><span className="eyebrow">Configuration · organization & access</span><h2 id="organization-access-heading">Organization & access</h2><p>Review departments, positions, reporting lines, directory provisioning and workspace access.</p></div>
    </header>
    <IdentityAccessPanel/>
  </section>;
}
