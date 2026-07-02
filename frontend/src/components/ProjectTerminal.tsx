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
    const observer = new ResizeObserver(() => {
      fitRef.current?.fit();
      api.terminalResize(projectId, term.cols, term.rows);
    });
    observer.observe(containerRef.current);
    return () => observer.disconnect();
  }, [term, projectId]);

  return (
    <div
      ref={containerRef}
      className="h-[420px] rounded-lg overflow-hidden border border-white/10 bg-[#0c0d12] p-2"
    />
  );
}
