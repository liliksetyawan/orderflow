import { useState } from "react";
import { Link } from "react-router-dom";
import { ArrowRight, Filter, Inbox, Plus } from "lucide-react";

import { Container } from "@/components/layout/Container";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { LoadingSpinner } from "@/components/feedback/LoadingSpinner";
import { ErrorState } from "@/components/feedback/ErrorState";
import { EmptyState } from "@/components/feedback/EmptyState";
import { StatusBadge } from "@/components/saga/StatusBadge";
import { useListOrdersQuery } from "@/features/orders/ordersApi";
import type { OrderStatus } from "@/lib/types";
import { cn, formatAmount, timeAgo } from "@/lib/utils";

const FILTERS: Array<{ key: "" | OrderStatus; label: string }> = [
  { key: "", label: "All" },
  { key: "PENDING", label: "Pending" },
  { key: "AUTHORIZED", label: "Authorized" },
  { key: "CONFIRMED", label: "Confirmed" },
  { key: "CANCELED", label: "Canceled" },
];

export function Orders() {
  const [filter, setFilter] = useState<"" | OrderStatus>("");
  const { data, error, isFetching } = useListOrdersQuery({ status: filter });

  return (
    <Container className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Orders</h1>
          <p className="text-sm text-muted-foreground">
            All orders across the system, newest first.
          </p>
        </div>
        <Button asChild>
          <Link to="/orders/new">
            <Plus className="h-4 w-4" />
            New order
          </Link>
        </Button>
      </div>

      <div className="flex items-center gap-2 overflow-x-auto pb-1">
        <Filter className="h-4 w-4 shrink-0 text-muted-foreground" />
        {FILTERS.map((f) => (
          <button
            key={f.key || "all"}
            onClick={() => setFilter(f.key)}
            className={cn(
              "shrink-0 rounded-full border px-3 py-1 text-xs font-medium transition-colors",
              filter === f.key
                ? "border-primary bg-primary text-primary-foreground"
                : "border-border bg-background text-muted-foreground hover:text-foreground",
            )}
          >
            {f.label}
          </button>
        ))}
      </div>

      {error && (
        <ErrorState
          title="Couldn't load orders"
          message="Make sure the order service is running on localhost:8081 and CORS is configured."
        />
      )}

      {!error && data && data.data.length === 0 && (
        <EmptyState
          icon={<Inbox className="h-8 w-8" />}
          title="No orders match this filter"
          action={
            <Button variant="outline" size="sm" onClick={() => setFilter("")}>
              Clear filter
            </Button>
          }
        />
      )}

      {!error && data && data.data.length > 0 && (
        <Card>
          <CardHeader className="flex-row items-center justify-between space-y-0">
            <CardTitle className="text-sm uppercase tracking-wide text-muted-foreground">
              {data.total} order{data.total === 1 ? "" : "s"}
            </CardTitle>
            {isFetching && (
              <Badge variant="outline" className="font-normal">
                refreshing…
              </Badge>
            )}
          </CardHeader>
          <CardContent className="p-0">
            <ul className="divide-y">
              {data.data.map((o) => (
                <li key={o.id}>
                  <Link
                    to={`/orders/${o.id}`}
                    className="flex items-center justify-between gap-3 px-6 py-3 transition-colors hover:bg-accent/40"
                  >
                    <div className="min-w-0 flex-1">
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
                    <span className="font-mono text-sm">
                      {formatAmount(o.total)}
                    </span>
                    <StatusBadge status={o.status} />
                    <ArrowRight className="h-4 w-4 text-muted-foreground" />
                  </Link>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      {!error && !data && <LoadingSpinner label="Loading orders…" />}
    </Container>
  );
}
