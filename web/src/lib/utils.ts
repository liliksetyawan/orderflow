import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/** Format minor units (rupiah/cents) as a currency-style display string. */
export function formatAmount(minor: number, currency = "IDR"): string {
  if (currency === "IDR") {
    return `Rp ${minor.toLocaleString("id-ID")}`;
  }
  return `${(minor / 100).toFixed(2)} ${currency}`;
}

/** Short relative time: "12s ago", "3m ago", "2h ago", or full date. */
export function timeAgo(iso: string): string {
  const ts = new Date(iso).getTime();
  if (Number.isNaN(ts)) return "—";
  const diff = Math.floor((Date.now() - ts) / 1000);
  if (diff < 5) return "just now";
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return new Date(iso).toLocaleString();
}
