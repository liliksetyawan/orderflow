import { z } from "zod";

export const orderItemSchema = z.object({
  sku: z.string().min(1, "SKU required"),
  quantity: z.coerce
    .number({ invalid_type_error: "must be a number" })
    .int()
    .positive("must be > 0"),
  price: z.coerce
    .number({ invalid_type_error: "must be a number" })
    .int()
    .nonnegative("must be ≥ 0"),
});

export const createOrderSchema = z.object({
  customer_id: z.string().min(1, "customer_id required"),
  items: z.array(orderItemSchema).min(1, "at least one item required"),
});

export type CreateOrderForm = z.infer<typeof createOrderSchema>;
