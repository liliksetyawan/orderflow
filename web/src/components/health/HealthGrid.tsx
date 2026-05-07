import type { ServiceName } from "@/features/health/healthApi";
import { ServiceHealthBadge } from "./ServiceHealthBadge";

const services: ServiceName[] = ["order", "payment", "inventory", "notification"];

export function HealthGrid() {
  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      {services.map((s) => (
        <ServiceHealthBadge key={s} name={s} />
      ))}
    </div>
  );
}
