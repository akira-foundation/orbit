import { ServicesList } from "../components/ServicesList";

export function Services() {
  return (
    <div className="p-6 overflow-auto h-full scrollbar-thin">
      <div className="mx-auto w-full max-w-3xl">
        <h1 className="text-lg font-semibold mb-1">Services</h1>
        <p className="text-[12px] text-[var(--orbit-muted)] mb-5">
          Managed local services. Downloaded on demand, started when a project
          needs them.
        </p>
        <ServicesList />
      </div>
    </div>
  );
}
