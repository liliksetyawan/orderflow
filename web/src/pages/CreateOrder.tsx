import { useNavigate } from "react-router-dom";
import { toast } from "sonner";

import { Container } from "@/components/layout/Container";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { OrderForm } from "@/components/orders/OrderForm";
import { useCreateOrderMutation } from "@/features/orders/ordersApi";
import { pushRecent } from "@/lib/recent-orders";
import type { CreateOrderForm } from "@/lib/schemas";

export function CreateOrder() {
  const navigate = useNavigate();
  const [createOrder, { isLoading }] = useCreateOrderMutation();

  const onSubmit = async (data: CreateOrderForm) => {
    try {
      const order = await createOrder(data).unwrap();
      pushRecent({
        id: order.id,
        customer_id: order.customer_id,
        total: order.total,
        created_at: new Date().toISOString(),
      });
      toast.success("Order created", {
        description: `Saga running · status ${order.status}`,
      });
      navigate(`/orders/${order.id}`);
    } catch (err: unknown) {
      const msg =
        (err as { data?: { error?: string } })?.data?.error ??
        "Failed to create order";
      toast.error("Could not create order", { description: msg });
    }
  };

  return (
    <Container className="max-w-3xl">
      <Card>
        <CardHeader>
          <CardTitle className="text-xl">Place a new order</CardTitle>
          <CardDescription>
            POST <code className="font-mono">/v1/orders</code> on the order
            service. The saga starts immediately — head to the order detail
            page to watch it progress.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <OrderForm isSubmitting={isLoading} onSubmit={onSubmit} />
        </CardContent>
      </Card>
    </Container>
  );
}
