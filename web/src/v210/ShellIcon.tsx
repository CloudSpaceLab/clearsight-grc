type ShellIconName = "home" | "frameworks" | "programs" | "forms" | "vendors" | "risks" | "privacy" | "documents" | "work" | "reports" | "settings" | "menu";

const paths: Record<ShellIconName, string> = {
  home: "M3 10.5 12 3l9 7.5M5 9.5V21h5v-6h4v6h5V9.5",
  frameworks: "m3 7 9-4 9 4-9 5-9-5Zm0 5 9 5 9-5M3 17l9 5 9-5",
  programs: "M3 6h7l2 3h9v10a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V6Z",
  forms: "M6 3h12v18H6zM9 8h6M9 12h6M9 16h4",
  vendors: "M3 21V8l7-4v17M10 10h10v11M6 11h1M6 15h1M14 14h2M14 18h2",
  risks: "M12 3 2.5 20h19L12 3Zm0 6v5m0 3v.01",
  privacy: "M5 10h14v11H5zM8 10V7a4 4 0 0 1 8 0v3M12 14v3",
  documents: "M5 2h9l5 5v15H5zM14 2v6h5M8 12h8M8 16h6",
  work: "M4 6h16v15H4zM9 6V3h6v3M8 14l3 3 5-6",
  reports: "M4 20V10m6 10V4m6 16v-7m6 8H2",
  settings: "M12 2v3m0 14v3M2 12h3m14 0h3M5 5l2 2m10 10 2 2M5 19l2-2M17 7l2-2M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0",
  menu: "M4 6h16M4 12h16M4 18h16",
};

export function ShellIcon({ name }: { name: ShellIconName }) {
  return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round"><path d={paths[name]}/></svg>;
}
