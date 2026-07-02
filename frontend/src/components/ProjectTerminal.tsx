import { useEffect, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { api } from "../api";
import { useProjectTerminal } from "../hooks/useProjectTerminal";

export function ProjectTerminal({ projectId }: { projectId: string }) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const [term, setTerm] = useState<Terminal | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    const instance = new Terminal({
      convertEol: true,
      fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
      fontSize: 12.5,
      theme: { background: "#0c0d12" },
    });
    const fit = new FitAddon();
    instance.loadAddon(fit);
    instance.open(containerRef.current);
    fit.fit();
    fitRef.current = fit;
    setTerm(instance);

    return () => {
      instance.dispose();
      setTerm(null);
    };
  }, []);

  useProjectTerminal(projectId, term);

  useEffect(() => {
    if (!term || !containerRef.current) return;

    const lastSize = { cols: term.cols, rows: term.rows };
    let debounce: ReturnType<typeof setTimeout> | null = null;

    const applyResize = () => {
      fitRef.current?.fit();
      if (term.cols === lastSize.cols && term.rows === lastSize.rows) return;
      lastSize.cols = term.cols;
      lastSize.rows = term.rows;
      api.terminalResize(projectId, term.cols, term.rows);
    };

    const observer = new ResizeObserver(() => {
      if (debounce) clearTimeout(debounce);
      debounce = setTimeout(applyResize, 120);
    });
    observer.observe(containerRef.current);

    return () => {
      observer.disconnect();
      if (debounce) clearTimeout(debounce);
    };
  }, [term, projectId]);

  return (
    <div
      ref={containerRef}
      className="h-[420px] rounded-lg overflow-hidden border border-white/10 bg-[#0c0d12] p-2"
    />
  );
}
