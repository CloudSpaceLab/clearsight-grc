import { useMemo, useState } from "react";
import { LexicalComposer } from "@lexical/react/LexicalComposer";
import { ContentEditable } from "@lexical/react/LexicalContentEditable";
import { HistoryPlugin } from "@lexical/react/LexicalHistoryPlugin";
import { LinkPlugin } from "@lexical/react/LexicalLinkPlugin";
import { ListPlugin } from "@lexical/react/LexicalListPlugin";
import { OnChangePlugin } from "@lexical/react/LexicalOnChangePlugin";
import { RichTextPlugin } from "@lexical/react/LexicalRichTextPlugin";
import { $createLinkNode, $isLinkNode, LinkNode, TOGGLE_LINK_COMMAND } from "@lexical/link";
import { $createListItemNode, $createListNode, $isListNode, ListItemNode, ListNode, INSERT_UNORDERED_LIST_COMMAND } from "@lexical/list";
import { $createHeadingNode, $isHeadingNode, HeadingNode } from "@lexical/rich-text";
import {
  $createParagraphNode, $createTextNode, $getRoot, $isParagraphNode, $isTextNode, $insertNodes,
  COMMAND_PRIORITY_LOW, FORMAT_TEXT_COMMAND, TextNode, type EditorConfig, type EditorState, type LexicalEditor, type NodeKey, type SerializedTextNode,
} from "lexical";
import type { CommunicationNode, CommunicationTemplate } from "../../formsCommunicationApi";

const variables = ["recipient_name", "bank_name", "form_title", "task_summary", "due_time", "link_expiry", "access_instructions", "support_contact", "secure_form_link"] as const;

type SerializedVariableNode = SerializedTextNode & { type: "communication-variable"; version: 1 };

class VariableNode extends TextNode {
  static getType() { return "communication-variable"; }
  static clone(node: VariableNode) { return new VariableNode(node.__text, node.__key); }
  constructor(text: string, key?: NodeKey) { super(text, key); }
  createDOM(config: EditorConfig) { const element = super.createDOM(config); element.classList.add("forms-variable-token"); return element; }
  static importJSON(value: SerializedVariableNode) { return $createVariableNode(value.text); }
  exportJSON(): SerializedVariableNode { return { ...super.exportJSON(), type: "communication-variable", version: 1 }; }
  isTextEntity() { return true; }
  canInsertTextBefore() { return false; }
  canInsertTextAfter() { return false; }
}
function $createVariableNode(value: string) { return new VariableNode(value).setMode("token"); }

export type CommunicationTemplateDraft = Pick<CommunicationTemplate, "action" | "locale" | "subject_template" | "document" | "effective_from" | "effective_until">;

type Props = { initial?: CommunicationTemplate; onSave: (draft: CommunicationTemplateDraft) => Promise<void>; busy?: boolean };

export function CommunicationTemplateEditor({ initial, onSave, busy }: Props) {
  const [action, setAction] = useState<CommunicationTemplate["action"]>(initial?.action ?? "INVITATION");
  const [locale, setLocale] = useState(initial?.locale ?? "en");
  const [subject, setSubject] = useState(initial?.subject_template ?? "{{form_title}} — action required");
  const [document, setDocument] = useState<CommunicationNode[]>(initial?.document ?? [{ type: "paragraph", text: "Hello {{recipient_name}}," }, { type: "paragraph", text: "{{task_summary}}" }, { type: "primary-action", text: "Open secure form", href: "{{secure_form_link}}" }]);
  const [effectiveFrom, setEffectiveFrom] = useState(toLocal(initial?.effective_from) || toLocal(new Date().toISOString()));
  const [effectiveUntil, setEffectiveUntil] = useState(toLocal(initial?.effective_until));
  const initialConfig = useMemo(() => ({
    namespace: `forms-communication-${initial?.id ?? "new"}-${initial?.version ?? 0}`,
    theme: { paragraph: "forms-lexical-paragraph", text: { bold: "forms-lexical-bold", italic: "forms-lexical-italic" }, heading: { h1: "forms-lexical-h1", h2: "forms-lexical-h2", h3: "forms-lexical-h3" }, link: "forms-lexical-link" },
    nodes: [HeadingNode, ListNode, ListItemNode, LinkNode, VariableNode],
    onError(error: Error) { throw error; },
    editorState: () => importDocument(document),
  }), [initial?.id, initial?.version]);

  async function save() {
    await onSave({ action, locale: locale.trim(), subject_template: subject.trim(), document, effective_from: new Date(effectiveFrom).toISOString(), effective_until: effectiveUntil ? new Date(effectiveUntil).toISOString() : undefined });
  }

  return <section className="forms-communication-editor" aria-labelledby="communication-editor-title">
    <div className="forms-task-heading"><div><span>New governed revision</span><h3 id="communication-editor-title">{initial ? `Edit ${initial.action} · ${initial.locale} · v${initial.version}` : "Create communication template"}</h3><p>Saving creates a new immutable draft revision. Protected variables are token nodes and cannot be edited character-by-character.</p></div></div>
    <div className="forms-task-grid">
      <label><span>Action</span><select value={action} onChange={(event) => setAction(event.target.value as CommunicationTemplate["action"])}><option>INVITATION</option><option>REMINDER</option><option>DUE_SOON</option><option>EXPIRED</option><option>CHANGE_REQUESTED</option><option>AMENDMENT</option><option>COMPLETION</option></select></label>
      <label><span>Locale</span><input value={locale} maxLength={20} onChange={(event) => setLocale(event.target.value)}/></label>
      <label className="forms-task-span"><span>Subject</span><input value={subject} maxLength={200} onChange={(event) => setSubject(event.target.value)}/><div className="forms-variable-palette">{variables.map((value) => <button type="button" key={value} onClick={() => setSubject((current) => `${current} {{${value}}}`.trim())}>{`{{${value}}}`}</button>)}</div></label>
      <label><span>Effective from</span><input type="datetime-local" value={effectiveFrom} onChange={(event) => setEffectiveFrom(event.target.value)}/></label>
      <label><span>Effective until</span><input type="datetime-local" value={effectiveUntil} onChange={(event) => setEffectiveUntil(event.target.value)}/></label>
    </div>
    <LexicalComposer initialConfig={initialConfig}>
      <LexicalToolbar/>
      <div className="forms-lexical-shell">
        <RichTextPlugin contentEditable={<ContentEditable className="forms-lexical-editor" aria-label="Communication body"/>} placeholder={<div className="forms-lexical-placeholder">Write governed recipient communication…</div>} ErrorBoundary={LexicalErrorBoundary}/>
        <HistoryPlugin/><ListPlugin/><LinkPlugin/>
        <OnChangePlugin onChange={(state) => setDocument(exportDocument(state))}/>
      </div>
      <VariableToolbar/>
    </LexicalComposer>
    <div className="forms-task-actions"><button className="forms-primary" type="button" disabled={busy || !locale.trim() || !subject.trim() || document.length === 0 || !effectiveFrom} onClick={() => void save()}>{busy ? "Saving…" : "Save new draft revision"}</button><small>Server validation remains authoritative for placeholders, secure-link requirements and effective dates.</small></div>
  </section>;
}

function LexicalToolbar() {
  const [editor, setEditor] = useState<LexicalEditor>();
  return <LexicalComposerConsumer onEditor={setEditor}>{editor && <div className="forms-lexical-toolbar"><button type="button" onClick={() => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "bold")}>Bold</button><button type="button" onClick={() => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "italic")}>Italic</button><button type="button" onClick={() => editor.dispatchCommand(INSERT_UNORDERED_LIST_COMMAND, undefined)}>List</button><button type="button" onClick={() => editor.dispatchCommand(TOGGLE_LINK_COMMAND, "https://example.invalid/")}>Link</button></div>}</LexicalComposerConsumer>;
}
function VariableToolbar() {
  const [editor, setEditor] = useState<LexicalEditor>();
  return <LexicalComposerConsumer onEditor={setEditor}>{editor && <div className="forms-variable-palette" aria-label="Protected communication variables">{variables.map((value) => <button type="button" key={value} onClick={() => editor.update(() => $insertNodes([$createVariableNode(`{{${value}}}`)]))}>{`{{${value}}}`}</button>)}</div>}</LexicalComposerConsumer>;
}

function LexicalComposerConsumer({ children, onEditor }: { children: React.ReactNode; onEditor: (editor: LexicalEditor) => void }) {
  const Consumer = requireComposerContext();
  return <Consumer onEditor={onEditor}>{children}</Consumer>;
}
function requireComposerContext() {
  // Kept in this Lexical-only chunk so no editor code is pulled into the main Forms workspace.
  return ComposerContextBridge;
}
function ComposerContextBridge({ children, onEditor }: { children: React.ReactNode; onEditor: (editor: LexicalEditor) => void }) {
  const [editor] = useLexicalComposerContextCompat();
  onEditor(editor);
  return <>{children}</>;
}

// Isolated import keeps every Lexical symbol behind the Communications lazy boundary.
import { useLexicalComposerContext as useLexicalComposerContextCompat } from "@lexical/react/LexicalComposerContext";

function LexicalErrorBoundary({ children }: { children?: React.ReactNode }) { return <>{children}</>; }

function importDocument(nodes: CommunicationNode[]) {
  const root = $getRoot(); root.clear();
  for (const node of nodes) {
    const type = node.type.toLowerCase();
    if (type === "divider") { const paragraph = $createParagraphNode(); paragraph.append($createTextNode("—")); root.append(paragraph); continue; }
    if (type === "list") {
      const list = $createListNode("bullet");
      for (const item of node.items ?? []) { const li = $createListItemNode(); appendProtectedText(li, item); list.append(li); }
      root.append(list); continue;
    }
    const container = type === "heading" ? $createHeadingNode(`h${Math.min(3, Math.max(1, node.level ?? 2))}` as "h1" | "h2" | "h3") : $createParagraphNode();
    if (type === "link" || type === "primary-action") { const link = $createLinkNode(node.href || "https://example.invalid/"); appendProtectedText(link, node.text ?? "Open"); container.append(link); }
    else { appendProtectedText(container, node.text ?? ""); if (type === "strong") container.getChildren().forEach((child) => $isTextNode(child) && child.toggleFormat("bold")); if (type === "emphasis") container.getChildren().forEach((child) => $isTextNode(child) && child.toggleFormat("italic")); }
    root.append(container);
  }
  if (root.getChildrenSize() === 0) root.append($createParagraphNode());
}
function appendProtectedText(parent: { append: (...nodes: TextNode[]) => unknown }, value: string) {
  const pattern = /(\{\{[a-z_]+\}\})/g; let offset = 0;
  for (const match of value.matchAll(pattern)) { const index = match.index ?? 0; if (index > offset) parent.append($createTextNode(value.slice(offset, index))); parent.append($createVariableNode(match[0])); offset = index + match[0].length; }
  if (offset < value.length) parent.append($createTextNode(value.slice(offset)));
}
function exportDocument(state: EditorState): CommunicationNode[] {
  let result: CommunicationNode[] = [];
  state.read(() => {
    result = $getRoot().getChildren().flatMap((node): CommunicationNode[] => {
      if ($isHeadingNode(node)) return [{ type: "heading", level: Number(node.getTag().slice(1)), text: node.getTextContent() }];
      if ($isListNode(node)) return [{ type: "list", items: node.getChildren().filter($isListItemNode).map((item) => item.getTextContent()) }];
      if (!$isParagraphNode(node)) return [];
      const children = node.getChildren();
      if (children.length === 1 && $isLinkNode(children[0])) return [{ type: children[0].getURL() === "{{secure_form_link}}" ? "primary-action" : "link", text: children[0].getTextContent(), href: children[0].getURL() }];
      const text = node.getTextContent(); if (!text.trim()) return [];
      const textNodes = node.getAllTextNodes();
      if (textNodes.length && textNodes.every((child) => child.hasFormat("bold"))) return [{ type: "strong", text }];
      if (textNodes.length && textNodes.every((child) => child.hasFormat("italic"))) return [{ type: "emphasis", text }];
      return [{ type: "paragraph", text }];
    });
  });
  return result;
}
function toLocal(value?: string) { if (!value) return ""; const date = new Date(value); if (Number.isNaN(date.getTime())) return ""; const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000); return local.toISOString().slice(0, 16); }
