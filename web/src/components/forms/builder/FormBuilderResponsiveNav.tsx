type Props = {
  onOutline: () => void;
  onSettings: () => void;
};

export function FormBuilderResponsiveNav({ onOutline, onSettings }: Props) {
  return <nav className="form-builder-responsive-nav" aria-label="Builder panes">
    <span className="active" aria-current="page">Canvas</span>
    <button type="button" onClick={onOutline}>Outline</button>
    <button type="button" onClick={onSettings}>Settings</button>
  </nav>;
}
