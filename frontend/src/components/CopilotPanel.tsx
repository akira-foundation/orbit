import { useEffect, useState } from "react";
import { Loader2, Sparkles, Timer } from "lucide-react";
import type { Project } from "../types";
import {
  copilotApi,
  type CopilotAnswer,
  type CopilotConfigType,
  type CopilotProvider,
} from "../copilotApi";
import { Button } from "./ui/button";

interface Entry {
  action: string;
  answer: CopilotAnswer;
}

export function CopilotPanel({ project }: { project: Project }) {
  const [config, setConfig] = useState<CopilotConfigType | null>(null);
  const [providers, setProviders] = useState<CopilotProvider[]>([]);
  const [entries, setEntries] = useState<Entry[]>([]);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [question, setQuestion] = useState("");

  useEffect(() => {
    copilotApi.config().then(setConfig);
    copilotApi.providers().then(setProviders);
  }, []);

  const run = async (action: string, fn: () => Promise<CopilotAnswer>) => {
    setBusy(action);
    setError(null);
    try {
      const answer = await fn();
      setEntries((xs) => [{ action, answer }, ...xs]);
    } catch (e: any) {
      setError(e?.message ?? String(e));
    } finally {
      setBusy(null);
    }
  };

  const ask = () => {
    const q = question.trim();
    if (!q || busy) return;
    run("Question", () => copilotApi.ask(project.id, q));
    setQuestion("");
  };

  if (config && !config.enabled) {
    return (
      <p className="px-1 py-2 text-[12px] text-[var(--orbit-muted)] leading-relaxed">
        Orbit AI is off. Enable it in Settings, General to explain errors,
        diagnose slowness, and ask about this project.
      </p>
    );
  }
  if (config && providers.length === 0) {
    return (
      <p className="px-1 py-2 text-[12px] text-amber-300/90 leading-relaxed">
        No AI CLI found. Install the Claude Code CLI (claude) or the Codex CLI
        (codex) and sign in, Orbit AI uses your existing subscription.
      </p>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={!!busy}
          onClick={() => run("Explain last error", () => copilotApi.explain(project.id))}
        >
          <Sparkles className="size-3.5 mr-1.5" />
          Explain last error
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={!!busy}
          onClick={() => run("Why slow", () => copilotApi.whySlow(project.id))}
        >
          <Timer className="size-3.5 mr-1.5" />
          Why slow
        </Button>
      </div>

      <div className="rounded-lg border border-white/10 bg-white/5 focus-within:border-[var(--orbit-accent-2)]/50 transition-colors">
        <textarea
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              ask();
            }
          }}
          rows={3}
          placeholder="Ask about this project… (Enter to send, Shift+Enter for a new line)"
          className="w-full px-3 pt-2.5 bg-transparent resize-none text-[13px] leading-relaxed outline-none placeholder:text-[var(--orbit-subtle)]"
        />
        <div className="flex justify-end px-2 pb-2">
          <Button
            variant="outline"
            size="sm"
            disabled={!question.trim() || !!busy}
            onClick={ask}
          >
            {busy === "Question" ? (
              <>
                <Loader2 className="size-3.5 mr-1.5 animate-spin" />
                Asking…
              </>
            ) : (
              "Ask"
            )}
          </Button>
        </div>
      </div>

      {error ? (
        <p className="text-[11.5px] text-rose-300 font-mono whitespace-pre-wrap">
          {error}
        </p>
      ) : null}

      <div className="space-y-2.5">
        {busy ? (
          <div className="rounded-lg border border-white/10 bg-white/[0.03] px-3.5 py-3 flex items-center gap-2 text-[12px] text-[var(--orbit-muted)]">
            <Loader2 className="size-3.5 animate-spin text-[var(--orbit-accent-2)]" />
            {busy}…
          </div>
        ) : null}
        {entries.map((e, i) => (
          <AnswerCard key={entries.length - i} entry={e} />
        ))}
        {entries.length === 0 && !busy ? (
          <p className="text-[11.5px] text-[var(--orbit-subtle)]">
            Answers use your local Claude Code or Codex CLI, running inside
            this project's folder with its recent logs as context.
          </p>
        ) : null}
      </div>
    </div>
  );
}

function AnswerCard({ entry }: { entry: Entry }) {
  const a = entry.answer;
  return (
    <div className="rounded-lg border border-white/10 bg-white/[0.03]">
      <div className="px-3.5 pt-2.5 pb-1.5 flex items-center gap-2 text-[11px] text-[var(--orbit-subtle)]">
        <Sparkles className="size-3 text-[var(--orbit-accent-2)]" />
        <span className="font-medium text-[var(--orbit-muted)]">
          {entry.action}
        </span>
        <span className="ml-auto">
          {a.provider} · {(a.durationMs / 1000).toFixed(1)}s
        </span>
      </div>
      <div className="px-3.5 pb-3 text-[12.5px] leading-relaxed whitespace-pre-wrap text-[var(--orbit-text)]/90">
        {a.text}
      </div>
    </div>
  );
}
