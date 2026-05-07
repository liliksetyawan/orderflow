import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";

import type {
  CreateOrderRequest,
  Order,
  OrderListResponse,
  OrderStatus,
} from "@/lib/types";
import { TERMINAL_STATUSES } from "@/lib/types";

const ORDER_BASE_URL =
  import.meta.env.VITE_ORDER_BASE_URL ?? "http://localhost:8081";

export const ordersApi = createApi({
  reducerPath: "ordersApi",
  baseQuery: fetchBaseQuery({ baseUrl: ORDER_BASE_URL }),
  tagTypes: ["Order", "OrderList"],
  endpoints: (builder) => ({
    /**
     * Create a new order. Returns the created Order with status PENDING.
     * Saga progression happens server-side; poll getOrder to follow it.
     */
    createOrder: builder.mutation<Order, CreateOrderRequest>({
      query: (body) => ({
        url: "/v1/orders",
        method: "POST",
        body,
      }),
      invalidatesTags: ["OrderList"],
    }),

    /**
     * Fetch one order by id. The pollingInterval option (set per-component
     * via useGetOrderQuery's `pollingInterval`) drives the saga timeline
     * auto-refresh. Pass `skipPollingIfUnfocused: true` so we stop polling
     * when the tab is in the background.
     */
    getOrder: builder.query<Order, string>({
      query: (id) => `/v1/orders/${id}`,
      providesTags: (_r, _e, id) => [{ type: "Order", id }],
    }),

    /**
     * List orders, newest first. Items are not included — fetch detail
     * via getOrder when needed.
     */
    listOrders: builder.query<
      OrderListResponse,
      { status?: OrderStatus | ""; limit?: number; offset?: number }
    >({
      query: ({ status, limit = 20, offset = 0 } = {}) => ({
        url: "/v1/orders",
        params: {
          ...(status ? { status } : {}),
          limit,
          offset,
        },
      }),
      providesTags: ["OrderList"],
    }),
  }),
});

export const {
  useCreateOrderMutation,
  useGetOrderQuery,
  useListOrdersQuery,
} = ordersApi;

/** Returns true if a status is terminal (no further saga steps). */
export function isTerminal(status: OrderStatus): boolean {
  return TERMINAL_STATUSES.includes(status);
}
