import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { ArrowLeft, RefreshCw } from "lucide-react";

import { Container } from "@/components/layout/Container";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { LoadingSpinner } from "@/components/feedback/LoadingSpinner";
import { ErrorState } from "@/components/feedback/ErrorState";
import { OrderSummaryCard } from "@/components/orders/OrderSummaryCard";
import { SagaTimeline } from "@/components/saga/SagaTimeline";
import { useGetOrderQuery, isTerminal } from "@/features/orders/ordersApi";

export function OrderDetail() {
  const { id } = useParams<{ id: string }>();

  // Polling cadence: 1s while the saga progresses, 0 (off) once the order
  // reaches a terminal state. We can't reference `order` inside the same
  // destructuring statement, so we mirror it through state.
  const [pollingInterval, setPollingInterval] = useState(1000);

  const { data: order, error, isLoading, isFetching, refetch } =
    useGetOrderQuery(id!, { skip: !id, pollingInterval });

  useEffect(() => {
    if (order && isTerminal(order.status)) setPollingInterval(0);
    else setPollingInterval(1000);
  }, [order]);

  if (isLoading) {
    return (
      <Container className="py-10">
        <LoadingSpinner label="Loading order…" />
      </Container>
    );
  }

  if (error || !order) {
    return (
      <Container className="max-w-3xl space-y-4">
        <BackLink />
        <ErrorState
          title="Order not found"
          message={
            (error as { status?: number } | undefined)?.status === 404
              ? "We couldn't find an order with that id."
              : "Something went wrong fetching this order."
          }
        />
      </Container>
    );
  }

  return (
    <Container className="space-y-6">
      <div className="flex items-center justify-between">
        <BackLink />
        <Button
          variant="ghost"
          size="sm"
          onClick={() => refetch()}
          disabled={isFetching}
        >
          <RefreshCw className={isFetching ? "h-4 w-4 animate-spin" : "h-4 w-4"} />
          Refresh
        </Button>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[2fr_1fr]">
        <Card>
          <CardHeader>
            <CardTitle>Saga timeline</CardTitle>
            <CardDescription>
              {isTerminal(order.status)
                ? "Saga complete — no further events expected"
                : "Auto-refreshing every second until the saga reaches a terminal state"}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <SagaTimeline status={order.status} />
          </CardContent>
        </Card>

        <OrderSummaryCard order={order} />
      </div>
    </Container>
  );
}

function BackLink() {
  return (
    <Button variant="ghost" size="sm" asChild>
      <Link to="/orders">
        <ArrowLeft className="h-4 w-4" />
        All orders
      </Link>
    </Button>
  );
}
