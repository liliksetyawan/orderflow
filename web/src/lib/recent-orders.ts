/**
 * localStorage helper to remember orders the current browser created.
 * Used on the dashboard for "your recent orders" since the backend list
 * endpoint is unfiltered. Capped to 25 entries (newest first).
 */
const KEY = "orderflow:recent-orders";
const MAX = 25;

export interface RecentOrder {
  id: string;
  customer_id: string;
  total: number;
  created_at: string;
}

export function loadRecent(): RecentOrder[] {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.slice(0, MAX);
  } catch {
    return [];
  }
}

export function pushRecent(o: RecentOrder): void {
  try {
    const list = loadRecent();
    const next = [o, ...list.filter((x) => x.id !== o.id)].slice(0, MAX);
    localStorage.setItem(KEY, JSON.stringify(next));
  } catch {
    /* swallow quota errors */
  }
}

export function clearRecent(): void {
  try {
    localStorage.removeItem(KEY);
  } catch {
    /* noop */
  }
}
