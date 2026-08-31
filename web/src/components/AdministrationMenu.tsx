export function AdministrationMenu({ enabled, onOpen }: { enabled: boolean; onOpen: () => void }) {
  if (!enabled) return null;
  return <details className="administration-menu">
    <summary aria-label="Administration">Admin</summary>
    <div className="shell-popover" role="group" aria-label="Administration menu">
      <div><strong>Configuration</strong><span>People, authority, data, automation and system operations.</span></div>
      <button type="button" onClick={onOpen}>Open Configuration</button>
    </div>
  </details>;
}
