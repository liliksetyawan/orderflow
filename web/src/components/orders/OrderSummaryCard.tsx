import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { StatusBadge } from "@/components/saga/StatusBadge";
import type { Order } from "@/lib/types";
import { formatAmount } from "@/lib/utils";

export function OrderSummaryCard({ order }: { order: Order }) {
  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between space-y-0">
        <div>
          <CardTitle className="text-sm uppercase tracking-wide text-muted-foreground">
            Order
          </CardTitle>
          <p className="mt-1 font-mono text-xs text-muted-foreground break-all">
            {order.id}
          </p>
        </div>
        <StatusBadge status={order.status} />
      </CardHeader>
      <CardContent className="space-y-4">
        <div>
          <p className="text-xs uppercase tracking-wide text-muted-foreground">
            Customer
          </p>
          <p className="mt-0.5 font-medium">{order.customer_id}</p>
        </div>

        <div>
          <p className="text-xs uppercase tracking-wide text-muted-foreground">
            Items
          </p>
          <ul className="mt-1 divide-y rounded-lg border">
            {order.items.map((it, i) => (
              <li
                key={`${it.sku}-${i}`}
                className="flex items-center justify-between px-3 py-2 text-sm"
              >
                <div>
                  <span className="font-mono">{it.sku}</span>
                  <span className="ml-2 text-muted-foreground">
                    × {it.quantity}
                  </span>
                </div>
                <span className="font-mono">{formatAmount(it.price)}</span>
              </li>
            ))}
          </ul>
        </div>

        <div className="flex items-center justify-between border-t pt-4">
          <span className="text-sm text-muted-foreground">Total</span>
          <span className="font-mono text-base font-semibold">
            {formatAmount(order.total)}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}
