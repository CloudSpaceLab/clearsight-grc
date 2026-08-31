import { IdentityAccessPanel } from "../IdentityAccessPanel";

export function PeopleAccessSection() {
  return <section className="configure-domain" aria-labelledby="people-access-heading">
    <header className="configure-domain-header">
      <div><span className="eyebrow">Configuration · people & access</span><h2 id="people-access-heading">People & access</h2><p>Manage sign-in, directory provisioning, people, groups and workspace role mappings without changing material decision authority.</p></div>
    </header>
    <IdentityAccessPanel/>
  </section>;
}
