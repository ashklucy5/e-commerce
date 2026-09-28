// Location: src/lib/orders/presentation.ts

export type OrderFilter = "all" | "processing" | "transit" | "delivered";
export type OrderStatusTone = "active" | "delivered" | "warning" | "danger" | "neutral";

const PROCESSING_STATUSES = new Set([
  "pending_payment",
  "confirmed",
  "processing",
]);

const TRANSIT_STATUSES = new Set([
  "shipped",
  "in_transit",
  "out_for_delivery",
  "awaiting_confirmation",
  "delivering",
]);

const DELIVERED_STATUSES = new Set([
  "delivered",
  "completed",
]);

const DANGER_STATUSES = new Set([
  "cancelled",
  "canceled",
  "payment_expired",
  "failed",
  "refunded",
  "returned",
]);

export function normalizeOrderValue(value?: string) {
  return String(value ?? "")
    .trim()
    .toLowerCase()
    .replaceAll("-", "_")
    .replaceAll(" ", "_");
}

export function humanizeOrderValue(value?: string) {
  const normalized = normalizeOrderValue(value);

  if (!normalized) return "—";

  if (normalized === "cod") return "Cash on delivery";
  if (normalized === "cod_collected") return "COD collected";

  return normalized
    .replaceAll("_", " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

export function orderFilterForStatus(status: string): Exclude<OrderFilter, "all"> | "other" {
  const normalized = normalizeOrderValue(status);

  if (PROCESSING_STATUSES.has(normalized)) return "processing";
  if (TRANSIT_STATUSES.has(normalized)) return "transit";
  if (DELIVERED_STATUSES.has(normalized)) return "delivered";

  return "other";
}

export function orderMatchesFilter(status: string, filter: OrderFilter) {
  return filter === "all" || orderFilterForStatus(status) === filter;
}

export function isDeliveredOrder(status: string) {
  return DELIVERED_STATUSES.has(normalizeOrderValue(status));
}

export function orderStatusTone(status: string): OrderStatusTone {
  const normalized = normalizeOrderValue(status);

  if (DELIVERED_STATUSES.has(normalized)) return "delivered";
  if (DANGER_STATUSES.has(normalized)) return "danger";
  if (normalized === "pending_payment") return "warning";
  if (PROCESSING_STATUSES.has(normalized) || TRANSIT_STATUSES.has(normalized)) return "active";

  return "neutral";
}

export function formatOrderDate(value?: string) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;

  return new Intl.DateTimeFormat("en-BD", {
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(date);
}

export function formatOrderDateTime(value?: string) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;

  return new Intl.DateTimeFormat("en-BD", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

export function deliveryProgressIndex(orderStatus: string, trackingStage?: string) {
  const status = normalizeOrderValue(orderStatus);
  const stage = normalizeOrderValue(trackingStage);

  if (DELIVERED_STATUSES.has(status) || ["delivered", "completed"].includes(stage)) return 4;
  if (["delivering", "out_for_delivery", "awaiting_confirmation", "in_transit"].includes(stage)) return 3;
  if (status === "shipped" || ["shipped", "dispatched", "handoff"].includes(stage)) return 2;
  if (status === "processing" || ["processing", "warehouse", "packing", "ready_for_handoff"].includes(stage)) return 1;
  if (["confirmed", "pending_payment"].includes(status) || ["confirmed", "placed"].includes(stage)) return 0;

  return -1;
}

export const DELIVERY_STEPS = [
  "Confirmed",
  "Processing",
  "Shipped",
  "In transit",
  "Delivered",
] as const;
