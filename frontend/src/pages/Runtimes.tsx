import { RuntimesList } from "../components/RuntimesList";

export function Runtimes() {
  return (
    <div className="p-6 overflow-auto h-full scrollbar-thin">
      <div className="mx-auto w-full max-w-3xl">
        <h1 className="text-lg font-semibold mb-1">Runtimes</h1>
        <p className="text-[12px] text-[var(--orbit-muted)] mb-5">
          Bundled language runtimes. Downloaded on demand, resolved per project.
        </p>
        <RuntimesList />
      </div>
    </div>
  );
}
