import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import type { HealthResponse } from "@/lib/types";

export type ServiceName = "order" | "payment" | "inventory" | "notification";

export const SERVICE_BASE_URLS: Record<ServiceName, string> = {
  order:
    import.meta.env.VITE_ORDER_BASE_URL ?? "http://localhost:8081",
  payment:
    import.meta.env.VITE_PAYMENT_BASE_URL ?? "http://localhost:8082",
  inventory:
    import.meta.env.VITE_INVENTORY_BASE_URL ?? "http://localhost:8083",
  notification:
    import.meta.env.VITE_NOTIFICATION_BASE_URL ?? "http://localhost:8084",
};

/**
 * Single RTK Query "service" with one endpoint that takes the target service
 * URL as its arg. Lets us cache, refetch, and refresh per-service health
 * independently while keeping the boilerplate low.
 */
export const healthApi = createApi({
  reducerPath: "healthApi",
  baseQuery: fetchBaseQuery({ baseUrl: "" }),
  endpoints: (builder) => ({
    getHealth: builder.query<HealthResponse, ServiceName>({
      query: (svc) => ({ url: `${SERVICE_BASE_URLS[svc]}/health` }),
    }),
  }),
});

export const { useGetHealthQuery } = healthApi;
