import { useMemo, useState } from "react";
import { LexicalComposer } from "@lexical/react/LexicalComposer";
import { useLexicalComposerContext } from "@lexical/react/LexicalComposerContext";
import { ContentEditable } from "@lexical/react/LexicalContentEditable";
import { LexicalErrorBoundary } from "@lexical/react/LexicalErrorBoundary";
import { HistoryPlugin } from "@lexical/react/LexicalHistoryPlugin";
import { LinkPlugin } from "@lexical/react/LexicalLinkPlugin";
import { ListPlugin } from "@lexical/react/LexicalListPlugin";
import { OnChangePlugin } from "@lexical/react/LexicalOnChangePlugin";
import { RichTextPlugin } from "@lexical/react/LexicalRichTextPlugin";
import {
  $createLinkNode,
  $isLinkNode,
  LinkNode,
  TOGGLE_LINK_COMMAND,
} from "@lexical/link";
import {
  $createListItemNode,
  $createListNode,
  $isListItemNode,
  $isListNode,
  ListItemNode,
  ListNode,
  INSERT_UNORDERED_LIST_COMMAND,
} from "@lexical/list";
import {
  $createHeadingNode,
  $isHeadingNode,
  HeadingNode,
} from "@lexical/rich-text";
import {
  $createParagraphNode,
  $createTextNode,
  $getRoot,
  $isParagraphNode,
  $isTextNode,
  $insertNodes,
  FORMAT_TEXT_COMMAND,
  TextNode,
  type EditorConfig,
  type EditorState,
  type NodeKey,
  type SerializedTextNode,
} from "lexical";
import type {
  CommunicationNode,
  CommunicationTemplate,
} from "../../formsCommunicationApi";
import { Button, PopoverDialog, SelectField, TextField } from "../ui";

const variables = [
  "recipient_name",
  "bank_name",
  "form_title",
  "task_summary",
  "due_time",
  "link_expiry",
  "access_instructions",
  "support_contact",
  "secure_form_link",
] as const;
const actionOptions = [
  "INVITATION",
  "REMINDER",
  "DUE_SOON",
  "EXPIRED",
  "CHANGE_REQUESTED",
  "AMENDMENT",
  "COMPLETION",
].map((id) => ({
  id,
  label: id
    .toLowerCase()
    .replaceAll("_", " ")
    .replace(/(^|\s)\S/g, (part) => part.toUpperCase()),
})) as Array<{ id: CommunicationTemplate["action"]; label: string }>;
type SerializedVariableNode = SerializedTextNode & {
  type: "communication-variable";
  version: 1;
};

class VariableNode extends TextNode {
  static getType() {
    return "communication-variable";
  }
  static clone(node: VariableNode) {
    return new VariableNode(node.__text, node.__key);
  }
  constructor(text: string, key?: NodeKey) {
    super(text, key);
  }
  createDOM(config: EditorConfig) {
    const element = super.createDOM(config);
    element.classList.add("forms-variable-token");
    return element;
  }
  static importJSON(value: SerializedVariableNode) {
    return $createVariableNode(value.text);
  }
  exportJSON(): SerializedVariableNode {
    return {
      ...super.exportJSON(),
      type: "communication-variable",
      version: 1,
    };
  }
  isTextEntity() {
    return true;
  }
  canInsertTextBefore() {
    return false;
  }
  canInsertTextAfter() {
    return false;
  }
}
function $createVariableNode(value: string) {
  const node = new VariableNode(value);
  node.setMode("token");
  return node;
}

export type CommunicationTemplateDraft = Pick<
  CommunicationTemplate,
  | "action"
  | "locale"
  | "subject_template"
  | "document"
  | "effective_from"
  | "effective_until"
>;
type Props = {
  initial?: CommunicationTemplate;
  onSave: (draft: CommunicationTemplateDraft) => Promise<void>;
  onCancel: () => void;
  busy?: boolean;
};

export function CommunicationTemplateEditor({
  initial,
  onSave,
  onCancel,
  busy,
}: Props) {
  const [action, setAction] = useState<CommunicationTemplate["action"]>(
    initial?.action ?? "INVITATION",
  );
  const [locale, setLocale] = useState(initial?.locale ?? "en");
  const [subject, setSubject] = useState(
    initial?.subject_template ?? "{{form_title}} — action required",
  );
  const [document, setDocument] = useState<CommunicationNode[]>(
    initial?.document ?? [
      { type: "paragraph", text: "Hello {{recipient_name}}," },
      { type: "paragraph", text: "{{task_summary}}" },
      {
        type: "primary-action",
        text: "Open secure form",
        href: "{{secure_form_link}}",
      },
    ],
  );
  const [effectiveFrom, setEffectiveFrom] = useState(
    toLocal(initial?.effective_from) || toLocal(new Date().toISOString()),
  );
  const [effectiveUntil, setEffectiveUntil] = useState(
    toLocal(initial?.effective_until),
  );
  const initialConfig = useMemo(
    () => ({
      namespace: `forms-communication-${initial?.id ?? "new"}-${initial?.version ?? 0}`,
      theme: {
        paragraph: "forms-lexical-paragraph",
        text: { bold: "forms-lexical-bold", italic: "forms-lexical-italic" },
        heading: {
          h1: "forms-lexical-h1",
          h2: "forms-lexical-h2",
          h3: "forms-lexical-h3",
        },
        link: "forms-lexical-link",
      },
      nodes: [HeadingNode, ListNode, ListItemNode, LinkNode, VariableNode],
      onError(error: Error) {
        throw error;
      },
      editorState: () => importDocument(document),
    }),
    [initial?.id, initial?.version],
  );

  return (
    <div
      className="forms-communication-editor forms-dialog-editor"
      aria-labelledby="communication-editor-title"
    >
      <div className="forms-task-heading">
        <div>
          <span>New message version</span>
          <h3 id="communication-editor-title">
            {initial
              ? `Edit ${initial.action} · ${initial.locale} · v${initial.version}`
              : "Create communication template"}
          </h3>
          <p>
            Saving creates a new draft version. Protected variables remain fixed
            so secure links and recipient details render safely.
          </p>
        </div>
      </div>
      <div className="forms-task-grid">
        <SelectField
          label="Message action"
          value={action}
          placeholder="Choose message action"
          options={actionOptions}
          allowsEmpty={false}
          onChange={(value) => {
            if (value) setAction(value);
          }}
        />
        <TextField
          label="Locale"
          value={locale}
          maxLength={20}
          onChange={setLocale}
        />
        <div className="forms-task-span">
          <TextField
            label="Subject"
            value={subject}
            maxLength={200}
            isRequired
            onChange={setSubject}
          />
          <div
            className="forms-variable-palette"
            aria-label="Subject variables"
          >
            {variables.map((value) => (
              <Button
                variant="quiet"
                size="compact"
                key={value}
                onPress={() =>
                  setSubject((current) => `${current} {{${value}}}`.trim())
                }
              >{`{{${value}}}`}</Button>
            ))}
          </div>
        </div>
        <TextField
          label="Effective from"
          type="datetime-local"
          value={effectiveFrom}
          isRequired
          onChange={setEffectiveFrom}
        />
        <TextField
          label="Effective until"
          type="datetime-local"
          value={effectiveUntil}
          onChange={setEffectiveUntil}
        />
      </div>
      <LexicalComposer initialConfig={initialConfig}>
        <LexicalToolbar />
        <div className="forms-lexical-shell">
          <RichTextPlugin
            contentEditable={
              <ContentEditable
                className="forms-lexical-editor"
                aria-label="Communication body"
              />
            }
            placeholder={
              <div className="forms-lexical-placeholder">
                Write governed recipient communication…
              </div>
            }
            ErrorBoundary={LexicalErrorBoundary}
          />
          <HistoryPlugin />
          <ListPlugin />
          <LinkPlugin />
          <OnChangePlugin
            onChange={(state) => setDocument(exportDocument(state))}
          />
        </div>
        <VariableToolbar />
      </LexicalComposer>
      <div className="forms-task-actions">
        <Button onPress={onCancel}>Cancel</Button>
        <Button
          variant="primary"
          isDisabled={
            busy ||
            !locale.trim() ||
            !subject.trim() ||
            document.length === 0 ||
            !effectiveFrom
          }
          isLoading={busy}
          onPress={() =>
            void onSave({
              action,
              locale: locale.trim(),
              subject_template: subject.trim(),
              document,
              effective_from: new Date(effectiveFrom).toISOString(),
              effective_until: effectiveUntil
                ? new Date(effectiveUntil).toISOString()
                : undefined,
            })
          }
        >
          Save template revision
        </Button>
        <small>
          Complete the required variables and effective dates before saving.
        </small>
      </div>
    </div>
  );
}

function LexicalToolbar() {
  const [editor] = useLexicalComposerContext();
  return (
    <div className="forms-lexical-toolbar">
      <Button
        size="compact"
        onPress={() => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "bold")}
      >
        Bold
      </Button>
      <Button
        size="compact"
        onPress={() => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "italic")}
      >
        Italic
      </Button>
      <Button
        size="compact"
        onPress={() =>
          editor.dispatchCommand(INSERT_UNORDERED_LIST_COMMAND, undefined)
        }
      >
        List
      </Button>
      <Button
        size="compact"
        onPress={() =>
          editor.dispatchCommand(
            TOGGLE_LINK_COMMAND,
            "https://example.invalid/",
          )
        }
      >
        Link
      </Button>
    </div>
  );
}
function VariableToolbar() {
  const [editor] = useLexicalComposerContext();
  return (
    <PopoverDialog label="Insert protected variable" placement="top start" trigger={<Button size="compact">Insert protected variable</Button>}>
      <div className="forms-variable-palette" aria-label="Protected communication variables">
        {variables.map((value) => (
          <Button variant="quiet" size="compact" key={value} onPress={() => editor.update(() => $insertNodes([$createVariableNode(`{{${value}}}`)]))}>
            {`{{${value}}}`}
          </Button>
        ))}
      </div>
    </PopoverDialog>
  );
}

function importDocument(nodes: CommunicationNode[]) {
  const root = $getRoot();
  root.clear();
  for (const node of nodes) {
    const type = node.type.toLowerCase();
    if (type === "divider") {
      const paragraph = $createParagraphNode();
      paragraph.append($createTextNode("—"));
      root.append(paragraph);
      continue;
    }
    if (type === "list") {
      const list = $createListNode("bullet");
      for (const item of node.items ?? []) {
        const li = $createListItemNode();
        appendProtectedText(li, item);
        list.append(li);
      }
      root.append(list);
      continue;
    }
    const container =
      type === "heading"
        ? $createHeadingNode(
            `h${Math.min(3, Math.max(1, node.level ?? 2))}` as
              "h1" | "h2" | "h3",
          )
        : $createParagraphNode();
    if (type === "link" || type === "primary-action") {
      const link = $createLinkNode(node.href || "https://example.invalid/");
      appendProtectedText(link, node.text ?? "Open");
      container.append(link);
    } else {
      appendProtectedText(container, node.text ?? "");
      if (type === "strong")
        container
          .getAllTextNodes()
          .forEach((child) => child.toggleFormat("bold"));
      if (type === "emphasis")
        container
          .getAllTextNodes()
          .forEach((child) => child.toggleFormat("italic"));
    }
    root.append(container);
  }
  if (root.getChildrenSize() === 0) root.append($createParagraphNode());
}
function appendProtectedText(
  parent: { append: (...nodes: TextNode[]) => unknown },
  value: string,
) {
  const pattern = /(\{\{[a-z_]+\}\})/g;
  let offset = 0;
  for (const match of value.matchAll(pattern)) {
    const index = match.index ?? 0;
    if (index > offset)
      parent.append($createTextNode(value.slice(offset, index)));
    parent.append($createVariableNode(match[0]));
    offset = index + match[0].length;
  }
  if (offset < value.length)
    parent.append($createTextNode(value.slice(offset)));
}
function exportDocument(state: EditorState): CommunicationNode[] {
  let result: CommunicationNode[] = [];
  state.read(() => {
    result = $getRoot()
      .getChildren()
      .flatMap((node): CommunicationNode[] => {
        if ($isHeadingNode(node))
          return [
            {
              type: "heading",
              level: Number(node.getTag().slice(1)),
              text: node.getTextContent(),
            },
          ];
        if ($isListNode(node))
          return [
            {
              type: "list",
              items: node
                .getChildren()
                .filter($isListItemNode)
                .map((item) => item.getTextContent()),
            },
          ];
        if (!$isParagraphNode(node)) return [];
        const children = node.getChildren();
        if (children.length === 1 && $isLinkNode(children[0]))
          return [
            {
              type:
                children[0].getURL() === "{{secure_form_link}}"
                  ? "primary-action"
                  : "link",
              text: children[0].getTextContent(),
              href: children[0].getURL(),
            },
          ];
        const text = node.getTextContent();
        if (!text.trim()) return [];
        const textNodes = node.getAllTextNodes();
        if (
          textNodes.length &&
          textNodes.every((child) => child.hasFormat("bold"))
        )
          return [{ type: "strong", text }];
        if (
          textNodes.length &&
          textNodes.every((child) => child.hasFormat("italic"))
        )
          return [{ type: "emphasis", text }];
        return [{ type: "paragraph", text }];
      });
  });
  return result;
}
function toLocal(value?: string) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}
