import { useEffect, useRef, useState } from "react";
import * as pdfjs from "pdfjs-dist";
import workerUrl from "pdfjs-dist/build/pdf.worker.min.mjs?url";

pdfjs.GlobalWorkerOptions.workerSrc = workerUrl;

export function PdfViewer({ data }: { data: Uint8Array }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const container = containerRef.current;
    if (!container) return;

    const render = async () => {
      try {
        const pdf = await pdfjs.getDocument({ data }).promise;
        if (cancelled) return;
        container.replaceChildren();
        const dpr = Math.min(window.devicePixelRatio || 1, 2);
        for (let n = 1; n <= pdf.numPages; n++) {
          const page = await pdf.getPage(n);
          if (cancelled) return;
          const base = page.getViewport({ scale: 1 });
          const viewport = page.getViewport({ scale: dpr });
          const canvas = document.createElement("canvas");
          canvas.width = viewport.width;
          canvas.height = viewport.height;
          canvas.style.width = `${base.width}px`;
          canvas.style.height = `${base.height}px`;
          canvas.className = "shadow-lg shrink-0";
          const ctx = canvas.getContext("2d");
          if (!ctx) continue;
          container.appendChild(canvas);
          await page.render({ canvas, canvasContext: ctx, viewport }).promise;
        }
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      }
    };
    void render();
    return () => {
      cancelled = true;
    };
  }, [data]);

  if (error) {
    return (
      <div className="h-full grid place-items-center text-[12px] text-[var(--orbit-muted)]">
        Could not render PDF: {error}
      </div>
    );
  }
  return (
    <div
      ref={containerRef}
      className="h-full overflow-auto scrollbar-thin bg-neutral-800 p-4 flex flex-col items-center gap-4"
    />
  );
}
