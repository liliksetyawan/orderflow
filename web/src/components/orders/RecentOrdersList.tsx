import { Link } from "react-router-dom";
import { ArrowRight, Inbox } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { EmptyState } from "@/components/feedback/EmptyState";
import { loadRecent } from "@/lib/recent-orders";
import { formatAmount, timeAgo } from "@/lib/utils";

export function RecentOrdersList() {
  const items = loadRecent();

  if (items.length === 0) {
    return (
      <EmptyState
        icon={<Inbox className="h-8 w-8" />}
        title="No recent orders yet"
        description="Orders you create from this browser will show up here. Tap the Create tab to place your first one."
      />
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Your recent orders</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <ul className="divide-y">
          {items.map((o) => (
            <li key={o.id}>
              <Link
                to={`/orders/${o.id}`}
                className="flex items-center justify-between px-6 py-3 transition-colors hover:bg-accent/40"
              >
                <div className="min-w-0">
                  <p className="truncate font-mono text-xs text-muted-foreground">
                    {o.id}
                  </p>
                  <p className="mt-0.5 text-sm">
                    <span className="font-medium">{o.customer_id}</span>
                    <span className="text-muted-foreground">
                      {" · "}
                      {timeAgo(o.created_at)}
                    </span>
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  <span className="font-mono text-sm">
                    {formatAmount(o.total)}
                  </span>
                  <ArrowRight className="h-4 w-4 text-muted-foreground" />
                </div>
              </Link>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}
