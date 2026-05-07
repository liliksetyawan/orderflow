import { useGetHealthQuery, type ServiceName } from "@/features/health/healthApi";
import { Card, CardContent } from "@/components/ui/card";
import { Activity, AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";

const labels: Record<ServiceName, string> = {
  order: "Order",
  payment: "Payment",
  inventory: "Inventory",
  notification: "Notification",
};

export function ServiceHealthBadge({ name }: { name: ServiceName }) {
  const { data, error, isLoading, isFetching } = useGetHealthQuery(name, {
    pollingInterval: 5000,
  });

  const ok = !!data && !error;
  const status = isLoading
    ? "checking…"
    : ok
      ? "online"
      : "offline";

  return (
    <Card className="overflow-hidden">
      <CardContent className="flex items-center gap-3 p-4">
        <div
          className={cn(
            "relative flex h-9 w-9 items-center justify-center rounded-lg",
            ok
              ? "bg-success/15 text-success"
              : "bg-destructive/15 text-destructive",
          )}
        >
          {ok ? (
            <Activity className="h-4 w-4" />
          ) : (
            <AlertTriangle className="h-4 w-4" />
          )}
          {isFetching && (
            <span className="absolute -top-0.5 -right-0.5 h-2 w-2 animate-pulse rounded-full bg-primary" />
          )}
        </div>

        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium leading-tight">
            {labels[name]}
          </p>
          <p
            className={cn(
              "text-xs leading-tight",
              ok ? "text-success" : "text-destructive",
            )}
          >
            {status}
          </p>
        </div>
      </CardContent>
    </Card>
  );
}
