// Render-only diagnostic for layout regressions in the authenticated shell.
export async function overflowElements(page) {
  return page.evaluate(() => {
    const viewport = document.documentElement.clientWidth;
    return [...document.body.querySelectorAll("*")].flatMap((node) => {
      const { left, right, width } = node.getBoundingClientRect();
      if (!width || (right <= viewport + 1 && left >= -1)) return [];
      return [{
        element: node.tagName.toLowerCase(),
        className: typeof node.className === "string" ? node.className.slice(0, 140) : "",
        left: Math.round(left), right: Math.round(right), width: Math.round(width),
      }];
    }).sort((a, b) => b.right - a.right).slice(0, 12);
  });
}
