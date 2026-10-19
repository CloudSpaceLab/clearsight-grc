// Render-only diagnostic for layout regressions in the authenticated shell.
export async function overflowElements(page) {
  return page.evaluate(() => {
    const viewport = document.documentElement.clientWidth;
    const describe = (node) => {
      if (!node) return null;
      const rect = node.getBoundingClientRect();
      const computed = getComputedStyle(node);
      return {
        element: node.tagName.toLowerCase(),
        className: typeof node.className === "string" ? node.className.slice(0, 90) : "",
        left: Math.round(rect.left), right: Math.round(rect.right), width: Math.round(rect.width),
        scrollWidth: node.scrollWidth,
        display: computed.display, minWidth: computed.minWidth,
        columns: computed.gridTemplateColumns,
      };
    };
    const selectors = [".app-shell", ".app-shell > .sidebar", ".app-shell > main", ".context-bar",
      ".today-topbar", "#today-brief", ".intervention-brief", ".intervention-list",
      ".intervention-row", ".monitoring-record"];
    const layout = Object.fromEntries(selectors.map((selector) => [selector, describe(document.querySelector(selector))]));
    const offenders = [...document.body.querySelectorAll("*")].flatMap((node) => {
      const { left, right, width } = node.getBoundingClientRect();
      if (!width || (right <= viewport + 1 && left >= -1)) return [];
      return [describe(node)];
    }).sort((a, b) => b.right - a.right).slice(0, 12);
    return { layout, offenders };
  });
}
