import { Badge } from "@/components/ui/badge";
import type { OrderStatus } from "@/lib/types";

const map: Record<
  OrderStatus,
  { label: string; variant: "default" | "secondary" | "success" | "warning" | "destructive" }
> = {
  PENDING: { label: "Pending", variant: "warning" },
  AUTHORIZED: { label: "Authorized", variant: "default" },
  CONFIRMED: { label: "Confirmed", variant: "success" },
  CANCELED: { label: "Canceled", variant: "destructive" },
};

export function StatusBadge({ status }: { status: OrderStatus }) {
  const cfg = map[status] ?? { label: status, variant: "secondary" as const };
  return <Badge variant={cfg.variant}>{cfg.label}</Badge>;
}
