// Runs inside the browser. Canvas parsing supports color(srgb ...) and color-mix().
// Estimates do not model overlapping siblings, pseudo-element paint or blur.
export function inspectRenderedContrast() {
  const canvas = document.createElement('canvas');
  canvas.width = canvas.height = 1;
  const context = canvas.getContext('2d', { willReadFrequently: true });
  const cache = new Map();
  const parse = (value) => {
    if (!value || !CSS.supports('color', value)) return null;
    if (!cache.has(value)) {
      context.clearRect(0, 0, 1, 1);
      context.fillStyle = value;
      context.fillRect(0, 0, 1, 1);
      const rgba = [...context.getImageData(0, 0, 1, 1).data];
      cache.set(value, [...rgba.slice(0, 3), rgba[3] / 255]);
    }
    return [...cache.get(value)];
  };
  const over = (foreground, background) => [0, 1, 2].map((i) => foreground[i] * foreground[3] + background[i] * (1 - foreground[3])).concat(1);
  const luminance = (color) => color.slice(0, 3).map((v) => v / 255).map((v) => v <= .04045 ? v / 12.92 : ((v + .055) / 1.055) ** 2.4).reduce((sum, v, i) => sum + v * [.2126, .7152, .0722][i], 0);
  const ratio = (a, b) => (Math.max(luminance(a), luminance(b)) + .05) / (Math.min(luminance(a), luminance(b)) + .05);
  const hex = (color) => '#' + color.slice(0, 3).map((v) => Math.round(v).toString(16).padStart(2, '0')).join('');
  function background(element) {
    const chain = [], effects = [];
    for (let node = element; node; node = node.parentElement) {
      const style = getComputedStyle(node);
      chain.unshift(style);
      if (+style.opacity !== 1) effects.push('element/group opacity');
      if (style.filter !== 'none' || style.backdropFilter !== 'none') effects.push('filter');
    }
    let color = [255, 255, 255, 1], uncertain = [];
    for (const style of chain) {
      const paint = parse(style.backgroundColor);
      if (!paint) uncertain.push('unsupported background color');
      else {
        if (paint[3] === 1) uncertain = [];
        color = over(paint, color);
      }
      if (style.backgroundImage !== 'none') uncertain.push('background image');
    }
    return { color, uncertain: [...new Set([...uncertain, ...effects])] };
  }
  const measurements = [];
  const add = (reading) => measurements.push({ ...reading, outcome: reading.disabled ? 'INACTIVE' : reading.ratio == null ? 'UNSUPPORTED' : reading.uncertain.length ? 'INCOMPLETE' : reading.ratio < reading.minimum ? reading.essential === false ? 'REVIEW' : 'FAIL' : 'PASS' });
  for (const element of document.querySelectorAll('body *')) {
    const style = getComputedStyle(element), box = element.getBoundingClientRect();
    if (!element.checkVisibility({ checkVisibilityCSS: true, checkOpacity: true }) || !box.width || !box.height || element.closest('[aria-hidden="true"], [inert]')) continue;
    const bg = background(element);
    const base = {
      selector: element.id ? '#' + CSS.escape(element.id) : element.tagName.toLowerCase() + [...element.classList].map((c) => '.' + CSS.escape(c)).join(''),
      parentSelector: element.parentElement ? element.parentElement.tagName.toLowerCase() + [...element.parentElement.classList].map((c) => '.' + CSS.escape(c)).join('') : null,
      disabled: !!element.closest(':disabled,[aria-disabled="true"],[data-disabled]'),
      background: hex(bg.color), uncertain: bg.uncertain,
      box: { x: box.x, y: box.y, width: box.width, height: box.height },
      fontSize: style.fontSize, fontWeight: style.fontWeight,
    };
    const text = [...element.childNodes].filter((node) => node.nodeType === Node.TEXT_NODE).map((node) => node.textContent).join(' ').trim();
    if (text) {
      const fg = parse(style.color);
      const large = parseFloat(style.fontSize) >= 24 || (parseFloat(style.fontSize) >= 18.66 && +style.fontWeight >= 700);
      add({ ...base, kind: 'text', text: text.slice(0, 120), foreground: fg ? hex(over(fg, bg.color)) : null, ratio: fg ? ratio(over(fg, bg.color), bg.color) : null, minimum: large ? 3 : 4.5 });
    }
    if (element.matches('input[placeholder],textarea[placeholder]') && !element.value) {
      const placeholder = getComputedStyle(element, '::placeholder'), fg = parse(placeholder.color);
      if (fg) fg[3] *= +placeholder.opacity;
      add({ ...base, kind: 'placeholder', text: element.getAttribute('placeholder'), foreground: fg ? hex(over(fg, bg.color)) : null, ratio: fg ? ratio(over(fg, bg.color), bg.color) : null, minimum: 4.5 });
    }
    // Enforce the shared field contract only. Other borders require human interpretation.
    const essential = element.matches('.cs-field__control,.cs-search-field__control,.cs-select-field__trigger,.cs-checkbox-field__box');
    if (essential || element.matches('input,textarea,select,button,[role="button"],[role="checkbox"],[role="combobox"]')) {
      const adjacent = background(element.parentElement), border = parse(style.borderTopColor);
      const painted = border && parseFloat(style.borderTopWidth) > 0 && !['none', 'hidden'].includes(style.borderTopStyle);
      if (!essential && !painted) continue;
      const borderOnFill = painted ? over(border, bg.color) : bg.color;
      const outsideRatio = ratio(borderOnFill, adjacent.color), insideRatio = ratio(borderOnFill, bg.color), fillRatio = ratio(bg.color, adjacent.color);
      add({ ...base, kind: 'control-border', essential, foreground: hex(borderOnFill), background: hex(adjacent.color), uncertain: [...new Set([...base.uncertain, ...adjacent.uncertain])], ratio: Math.max(Math.min(outsideRatio, insideRatio), fillRatio), outsideRatio, insideRatio, fillRatio, minimum: 3,
        interpretation: essential ? 'Shared field identification boundary, checked against fill and adjacent surface.' : 'Review whether this boundary identifies the control; a low reading alone is not a violation.' });
    }
  }
  return measurements;
}
