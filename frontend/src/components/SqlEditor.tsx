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
      border: "1px solid rgba(255,255,255,0.10)",
    },
    "&.cm-focused": { outline: "none", borderColor: "rgba(92,200,255,0.5)" },
    ".cm-gutters": { backgroundColor: "transparent", border: "none", color: "var(--orbit-subtle)" },
    ".cm-content": { fontFamily: "ui-monospace, SFMono-Regular, monospace", caretColor: "#fff" },
    ".cm-activeLine, .cm-activeLineGutter": { backgroundColor: "rgba(255,255,255,0.03)" },
    ".cm-tooltip-autocomplete": {
      background: "rgba(28,28,34,0.98)",
      border: "1px solid rgba(255,255,255,0.10)",
      borderRadius: "8px",
    },
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
