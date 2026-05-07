/**
 * Wire types matching the Go backend's HTTP DTOs.
 * Do not couple to internal domain shape — only what the API actually returns.
 */

export type OrderStatus =
  | "PENDING"
  | "AUTHORIZED"
  | "CONFIRMED"
  | "CANCELED";

export const TERMINAL_STATUSES: OrderStatus[] = ["CONFIRMED", "CANCELED"];

export interface OrderItem {
  sku: string;
  quantity: number;
  price: number;
}

export interface Order {
  id: string;
  customer_id: string;
  total: number;
  status: OrderStatus;
  items: OrderItem[];
}

export interface OrderListItem {
  id: string;
  customer_id: string;
  total: number;
  status: OrderStatus;
  created_at: string;
  updated_at: string;
}

export interface OrderListResponse {
  data: OrderListItem[];
  total: number;
  limit: number;
  offset: number;
}

export interface CreateOrderRequest {
  customer_id: string;
  items: OrderItem[];
}

export interface HealthResponse {
  status: string;
}

export interface ApiError {
  error: string;
}
