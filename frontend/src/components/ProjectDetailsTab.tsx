import { FolderOpen } from "lucide-react";
import { api } from "../api";
import type { Project, RuntimeSnapshot } from "../types";
import { Detail } from "./ProjectDetailParts";
import { ComposePanel } from "./ComposePanel";

export function ProjectDetailsTab({
  project,
  snapshot,
  onCopy,
}: {
  project: Project;
  snapshot: RuntimeSnapshot | null;
  onCopy: (key: string, value: string) => void;
}) {
  return (
    <>
      <dl className="divide-y divide-white/[0.04]">
        <Detail label="Dev command">
          <code className="font-mono text-[11.5px]">{project.devCommand}</code>
        </Detail>
        <Detail label="PID">
          <span className="font-mono text-[11.5px]">
            {snapshot?.pid ? snapshot.pid : "-"}
          </span>
        </Detail>
        <Detail label="Path">
          <div className="group inline-flex items-center gap-1.5 min-w-0 max-w-full justify-end">
            <button
              onClick={() => onCopy("path", project.path)}
              className="font-mono text-[11.5px] text-[var(--orbit-text)] hover:text-[var(--orbit-accent-2)] transition-colors truncate text-right"
              title="Copy path"
            >
              {project.path}
            </button>
            <button
              onClick={() => api.revealInFinder(project.path)}
              className="text-[var(--orbit-muted)] hover:text-[var(--orbit-text)]"
              title="Reveal in Finder"
            >
              <FolderOpen className="size-3.5" />
            </button>
          </div>
        </Detail>
        <Detail label="Slug">
          <code className="font-mono text-[11.5px]">{project.slug}</code>
        </Detail>
        <Detail label="Created">
          <span className="text-[11.5px] text-[var(--orbit-muted)]">
            {new Date(project.createdAt).toLocaleString()}
          </span>
        </Detail>
      </dl>
      <ComposePanel projectId={project.id} />
    </>
  );
}
