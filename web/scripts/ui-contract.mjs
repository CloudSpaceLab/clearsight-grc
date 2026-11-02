import { readFile, readdir } from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { createScanner, LanguageVariant, SyntaxKind } from "typescript/unstable/ast";

const nativeAlternatives = {
  button: "use Button or IconButton",
  select: "use SelectField",
  input: "use TextField or an approved field component",
  textarea: "use TextArea",
};

const normalize = (value) => value.replaceAll("\\", "/");
const lineAt = (source, index) => source.slice(0, index).split("\n").length;
const controlHeightToken = /^--cs-(?:button|field|tabs|select-item|filter-chip|scope-item)-height$/;
const controlHeightValue = /^var\(--cs-(?:control-height(?:-comfortable|-compact)?|target-min|(?:button|field|tabs|select-item|filter-chip|scope-item)-height)\)$/;
const controlSelector = /\.cs-(?:button|field|control|(?:search-)?field__control|tabs?__tab|select-field__(?:option|trigger)|filter-chip|scope-bar__item)(?![\w-])/i;
const componentLayoutProperties = new Set([
  "display", "position", "top", "right", "bottom", "left", "inset", "inset-inline", "inset-block",
  "width", "min-width", "max-width", "height", "min-height", "max-height",
  "inline-size", "min-inline-size", "max-inline-size", "block-size", "min-block-size", "max-block-size",
  "grid-template-columns", "grid-template-rows", "grid-column", "grid-row", "grid-area", "grid-auto-flow",
  "flex", "flex-basis", "flex-grow", "flex-shrink", "flex-direction", "flex-wrap",
  "align-items", "align-content", "align-self", "justify-items", "justify-content", "justify-self",
  "place-items", "place-content", "place-self", "gap", "row-gap", "column-gap",
  "margin", "margin-inline", "margin-block", "margin-inline-start", "margin-inline-end", "margin-block-start", "margin-block-end",
  "padding", "padding-inline", "padding-block", "padding-inline-start", "padding-inline-end", "padding-block-start", "padding-block-end",
  "font-size", "font-weight", "line-height", "letter-spacing", "text-align", "white-space", "overflow-wrap", "word-break",
  "overflow", "overflow-x", "overflow-y", "scrollbar-width", "vertical-align",
]);

function componentGeometryToken(property, componentTokens) {
  return componentTokens.has(property)
    && !/-(?:border|radius|shadow|focus|indicator|disabled|selected|hover|pressed|backdrop|transition|animation|opacity)(?:-|$)/.test(property)
    && /-(?:padding(?:-inline|-block)?|gap|(?:min-|max-)?height|(?:min-|max-)?width|font-size|heading-size|heading-weight|value-size|icon-size|icon-inset|heading-columns|label-transform|label-spacing)$/.test(property);
}

function featureLayoutSelectorIssue(selector, sharedComponentsOnly = false) {
  for (const part of selector.split(",")) {
    const component = /\.cs-[\w-]+/.exec(part);
    if (!component && sharedComponentsOnly) continue;
    const scope = component ? part.slice(0, component.index) : part;
    if (!/\.(?!cs-)[\w-]+/.test(scope)) return "shared component layout must be scoped by a feature class";
    const structural = part.replace(/:(?:first|last|only)-(?:child|of-type)\b|:nth-(?:last-)?(?:child|of-type)\([^)]*\)/g, "");
    if (/[:\[]/.test(structural)) return "shared component focus, interaction and state selectors belong in the component stylesheet";
  }
  return undefined;
}

export function validateTsxSource({ file, source, exceptions = [], scanNative = true, scanAriaImport = true }) {
  const diagnostics = [];
  const normalizedFile = normalize(file);
  const scanner = createScanner(true, LanguageVariant.JSX, source);
  const tokens = [];
  for (let kind = scanner.scan(); kind !== SyntaxKind.EndOfFile; kind = scanner.scan()) {
    tokens.push({ kind, text: scanner.getTokenText(), value: scanner.getTokenValue(), pos: scanner.getTokenStart() });
  }

  for (let index = 0; index < tokens.length; index += 1) {
    const token = tokens[index];
    if (scanAriaImport && token.kind === SyntaxKind.ImportKeyword && !normalizedFile.includes("/components/ui/")) {
      const specifier = tokens.slice(index + 1).find((item) => item.kind === SyntaxKind.StringLiteral || item.kind === SyntaxKind.SemicolonToken);
      if (specifier?.kind === SyntaxKind.StringLiteral && specifier.value === "react-aria-components") diagnostics.push(`${normalizedFile}:${lineAt(source, token.pos)} import shared controls from components/ui instead of react-aria-components`);
    }
    if (!scanNative || token.kind !== SyntaxKind.LessThanToken) continue;
    const name = tokens[index + 1];
    if (!name || name.kind !== SyntaxKind.Identifier || !Object.hasOwn(nativeAlternatives, name.value)) continue;
    const tag = name.value;
    let inputType = tag === "input" ? "text" : undefined;
    for (let cursor = index + 2; cursor < tokens.length && tokens[cursor].kind !== SyntaxKind.GreaterThanToken; cursor += 1) {
      if (tag === "input" && tokens[cursor].value === "type" && tokens[cursor + 1]?.kind === SyntaxKind.EqualsToken && tokens[cursor + 2]?.kind === SyntaxKind.StringLiteral) inputType = tokens[cursor + 2].value;
    }
    const permitted = exceptions.some((item) => normalize(item.file) === normalizedFile && item.tag === tag && (tag !== "input" || item.inputType === inputType) && typeof item.reason === "string" && item.reason.trim().length > 0);
    if (!permitted) diagnostics.push(`${normalizedFile}:${lineAt(source, token.pos)} raw <${tag}> is outside the migrated contract; ${nativeAlternatives[tag]}`);
  }
  return diagnostics;
}

export function validateCssSource({ file, source, componentTokens = new Set() }) {
  const diagnostics = [];
  const normalizedFile = normalize(file);
  const componentOwner = normalizedFile.startsWith("src/design-system/components/");
  // Comments are not selectors/declarations; retain their line offsets in diagnostics.
  const css = source.replace(/\/\*[\s\S]*?\*\//g, (comment) => comment.replace(/[^\n]/g, " "));
  const blocks = /([^{}]+)\{([^{}]*)\}/g;
  for (const block of css.matchAll(blocks)) {
    const selector = block[1].trim();
    const body = block[2];
    const blockIndex = block.index ?? 0;
    const sharedLayout = !componentOwner && /\.cs-[\w-]+/.test(selector);
    const sharedSelectorIssue = sharedLayout ? featureLayoutSelectorIssue(selector, true) : undefined;
    if (sharedSelectorIssue) diagnostics.push(`${normalizedFile}:${lineAt(source, blockIndex)} ${sharedSelectorIssue}`);
    for (const declaration of body.matchAll(/([\w-]+)\s*:\s*([^;]+);?/g)) {
      const property = declaration[1].toLowerCase();
      const value = declaration[2].trim();
      const at = blockIndex + selector.length + 1 + (declaration.index ?? 0);
      const diagnostic = (message) => diagnostics.push(`${normalizedFile}:${lineAt(source, at)} ${message}`);
      if (!componentOwner && property.startsWith("--cs-")) {
        if (!componentGeometryToken(property, componentTokens)) diagnostic(`${property} is not an existing component geometry token; palette, focus and state tokens belong in the shared layer`);
        const issue = featureLayoutSelectorIssue(selector);
        if (issue && issue !== sharedSelectorIssue) diagnostic(issue);
        if (controlHeightToken.test(property) && !controlHeightValue.test(value)) diagnostic(`${property} must use a control-height token`);
      } else if (sharedLayout && !componentLayoutProperties.has(property)) {
        diagnostic(`${property} is owned by the shared component; feature selectors may configure layout only`);
      }
      if (sharedLayout || (!componentOwner && property.startsWith("--cs-"))) {
        if (/!important/i.test(value)) diagnostic("shared component layout must not override the cascade with !important");
        if (property === "display" && /^(?:none|contents)$/i.test(value)) diagnostic("feature layout must not hide or remove the shared component's semantic box");
        if (/^overflow(?:-[xy])?$/.test(property) && /\b(?:hidden|clip)\b/.test(value)) diagnostic("shared component layout must preserve content and focus visibility");
      }
      if ((property === "color" || property.includes("background") || property.includes("border-color") || property === "fill" || property === "stroke") && /#[\da-f]{3,8}\b|(?:rgb|hsl)a?\(/i.test(value)) diagnostic(`${property} must use a design token`);
      if (property.includes("border-radius") && !tokenOrZero(value)) diagnostic(`${property} must use a radius token`);
      if (property.includes("box-shadow") && !tokenOrZero(value) && value !== "none") diagnostic(`${property} must use a shadow token`);
      if (property === "z-index" && !/^var\(--cs-[^)]+\)$/.test(value) && value !== "auto") diagnostic("z-index must use a z-index token");
      if ((property.includes("transition") || property.includes("animation")) && /\b\d*\.?\d+(?:ms|s)\b/.test(value)) diagnostic(`${property} must use a duration token`);
      if (/^(?:height|min-height|block-size|min-block-size)$/.test(property) && controlSelector.test(selector)
        && (componentOwner ? /\b\d+(?:\.\d+)?px\b/.test(value) : !controlHeightValue.test(value))) diagnostic(`${property} must use a control-height token`);
    }
  }
  return diagnostics;
}

function tokenOrZero(value) {
  const remainder = value.replace(/var\(--cs-[^)]+\)/g, "").replace(/\b0\b/g, "").replace(/\s|\/|,/g, "");
  return remainder.length === 0 || /^inset$/.test(remainder);
}

async function sourceFiles(directory) {
  const result = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const target = path.join(directory, entry.name);
    if (entry.isDirectory()) result.push(...await sourceFiles(target));
    else if (/\.(?:ts|tsx)$/.test(entry.name) && !/\.test\.(?:ts|tsx)$/.test(entry.name)) result.push(target);
  }
  return result;
}

export async function validateProject(root) {
  const manifest = JSON.parse(await readFile(path.join(root, "ui-contract-migrations.json"), "utf8"));
  const componentTokenSource = await readFile(path.join(root, "src/design-system/tokens/components.css"), "utf8");
  const componentTokens = new Set([...componentTokenSource.matchAll(/(--cs-[\w-]+)\s*:/g)].map((match) => match[1]));
  const diagnostics = [];
  const exceptions = manifest.nativeControlExceptions ?? [];
  for (const file of manifest.migratedTsx) {
    diagnostics.push(...validateTsxSource({ file, source: await readFile(path.join(root, file), "utf8"), exceptions, scanAriaImport: false }));
  }
  for (const absolute of await sourceFiles(path.join(root, "src"))) {
    const file = normalize(path.relative(root, absolute));
    const source = await readFile(absolute, "utf8");
    if (source.includes("react-aria-components")) diagnostics.push(...validateTsxSource({ file, source, exceptions, scanNative: false }));
  }
  for (const file of manifest.migratedCss) diagnostics.push(...validateCssSource({ file, source: await readFile(path.join(root, file), "utf8"), componentTokens }));

  const [barrel, gallery, design] = await Promise.all([
    readFile(path.join(root, "src/components/ui/index.ts"), "utf8"),
    readFile(path.join(root, "src/components/ui-gallery/UIComponentGallery.tsx"), "utf8"),
    readFile(path.join(root, "../DESIGN.md"), "utf8"),
  ]);
  for (const family of Object.keys(manifest.componentFamilies)) {
    if (!new RegExp(`\\b${family}\\b`).test(barrel)) diagnostics.push(`src/components/ui/index.ts:1 missing ${family} export`);
    if (!gallery.includes(`data-component-contract={family}`) && !gallery.includes(`family="${family}"`)) diagnostics.push(`src/components/ui-gallery/UIComponentGallery.tsx:1 missing ${family} component contract`);
    if (!design.includes(`\`${family}\``)) diagnostics.push(`../DESIGN.md:1 missing ${family} documentation`);
  }
  return diagnostics;
}

if (process.argv[1] && normalize(process.argv[1]) === normalize(new URL(import.meta.url).pathname).replace(/^\/(?:[A-Za-z]:)/, (match) => match.slice(1))) {
  const root = path.resolve(path.dirname(new URL(import.meta.url).pathname.replace(/^\/(?:[A-Za-z]:)/, (match) => match.slice(1))), "..");
  const diagnostics = await validateProject(root);
  if (diagnostics.length) {
    console.error(diagnostics.join("\n"));
    process.exitCode = 1;
  } else console.log("UI contracts passed.");
}
