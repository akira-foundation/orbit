import { useMemo, useRef } from "react";
import CodeMirror from "@uiw/react-codemirror";
import { sql, PostgreSQL } from "@codemirror/lang-sql";
import { keymap, EditorView } from "@codemirror/view";

const appTheme = EditorView.theme(
  {
    "&": {
      backgroundColor: "rgba(255,255,255,0.05)",
      fontSize: "12px",
      borderRadius: "8px",
      border: "1px solid var(--orbit-border)",
      transition: "border-color 0.15s",
    },
    "&.cm-focused": { outline: "none", borderColor: "color-mix(in oklab, var(--orbit-accent-2) 50%, transparent)" },
    ".cm-scroller": { fontFamily: "ui-monospace, SFMono-Regular, monospace", lineHeight: "1.6" },
    ".cm-gutters": { backgroundColor: "transparent", border: "none", color: "var(--orbit-subtle)" },
    ".cm-content": { padding: "8px 12px", caretColor: "var(--orbit-text)" },
    ".cm-line": { padding: "0" },
    ".cm-activeLine": { backgroundColor: "transparent" },
    ".cm-cursor": { borderLeftColor: "var(--orbit-text)" },
    ".cm-selectionBackground, &.cm-focused .cm-selectionBackground": {
      backgroundColor: "color-mix(in oklab, var(--orbit-accent-2) 25%, transparent)",
    },
    ".cm-tooltip-autocomplete": {
      background: "rgba(28,28,34,0.98)",
      border: "1px solid var(--orbit-border)",
      borderRadius: "10px",
      overflow: "hidden",
      boxShadow: "0 12px 32px rgba(0,0,0,0.5)",
    },
    ".cm-tooltip-autocomplete ul li[aria-selected]": {
      background: "color-mix(in oklab, var(--orbit-accent-2) 20%, transparent)",
      color: "var(--orbit-text)",
    },
    ".cm-tooltip-autocomplete ul li": { fontFamily: "ui-monospace, monospace", fontSize: "11.5px", padding: "3px 8px" },
    ".cm-placeholder": { color: "var(--orbit-subtle)" },
  },
  { dark: true },
);

export function SqlEditor({
  value,
  onChange,
  onRun,
  schema,
}: {
  value: string;
  onChange: (v: string) => void;
  onRun: () => void;
  schema: Record<string, string[]>;
}) {
  const runRef = useRef(onRun);
  runRef.current = onRun;

  const extensions = useMemo(
    () => [
      sql({ dialect: PostgreSQL, schema, upperCaseKeywords: true }),
      keymap.of([
        {
          key: "Mod-Enter",
          run: () => {
            runRef.current();
            return true;
          },
        },
      ]),
      EditorView.lineWrapping,
      appTheme,
    ],
    [schema],
  );

  return (
    <CodeMirror
      value={value}
      onChange={onChange}
      extensions={extensions}
      theme="dark"
      basicSetup={{ lineNumbers: false, foldGutter: false, highlightActiveLine: true }}
      height="80px"
      placeholder="SELECT * FROM …  (Cmd+Enter to run)"
    />
  );
}
