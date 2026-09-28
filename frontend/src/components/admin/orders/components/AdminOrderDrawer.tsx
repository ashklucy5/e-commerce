"use client";

import Link from "next/link";

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";
import type {
  AdminDeliveryTrackingDetail,
  AdminDeliveryTrackingResponse,
  AdminInvoice,
  AdminInvoiceResponse,
  AdminOrderDetail,
  AdminOrderResponse,
  AdminWarehouseFulfillment,
  AdminWarehouseFulfillmentsResponse,
} from "@/lib/admin/order-types";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/AdminOrders.module.css";

type WorkspaceMode = "orders" | "fulfillment" | "shipments";

type AdminOrderDrawerProps = {
  orderId: string | null;
  portal: string;
  mode: WorkspaceMode;
  onClose: () => void;
  onUpdated: () => void;
};

type OrderOperation = {
  label: string;
  targetStatus: string;
  description: string;
  tone: "primary" | "positive";
};

type FulfillmentOperation = {
  label: string;
  path: string;
};

function titleCase(value: string): string {
  return value
    .replace(/_/g, " ")
    .replace(/\b\w/g, (character) => character.toUpperCase());
}

function formatDateTime(value?: string): string {
  if (!value) {
    return "—";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "—";
  }

  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

function errorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) {
    return value.message;
  }

  if (value instanceof Error) {
    return value.message;
  }

  return "Order details could not be loaded.";
}

function statusTone(status: string): string {
  switch (status) {
    case "completed":
    case "delivered":
    case "paid":
    case "cod_collected":
    case "handed_off":
      return styles.tonePositive;

    case "confirmed":
    case "processing":
    case "shipped":
    case "picking":
    case "packed":
    case "ready_for_handoff":
      return styles.toneInfo;

    case "awaiting_procurement":
    case "pending_payment":
    case "cod_pending":
    case "allocated":
    case "waiting_inbound":
    case "received":
    case "awaiting_confirmation":
      return styles.toneAttention;

    case "refunded":
      return styles.toneViolet;

    case "cancelled":
    case "payment_expired":
    case "failed":
    case "expired":
      return styles.toneDanger;

    default:
      return styles.toneNeutral;
  }
}

function shipmentStatusLabel(status?: string): string {
  if (!status) {
    return "Not prepared";
  }

  switch (status) {
    case "pending":
      return "Prepared";
    case "shipped":
      return "Shipped";
    case "awaiting_confirmation":
      return "Awaiting confirmation";
    case "delivered":
      return "Delivered";
    case "cancelled":
      return "Cancelled";
    default:
      return titleCase(status);
  }
}

function orderOperation(order: AdminOrderDetail): OrderOperation | null {
  if (
    order.status === "awaiting_procurement" &&
    order.order_type === "sourcing"
  ) {
    return {
      label: "Confirm procurement received",
      targetStatus: "confirmed",
      description:
        "Use this only after the sourced stock has physically arrived and inventory has been received.",
      tone: "positive",
    };
  }

  if (order.status === "confirmed") {
    return {
      label: "Move to processing",
      targetStatus: "processing",
      description:
        "Marks the order ready for warehouse and fulfillment work.",
      tone: "primary",
    };
  }

  if (order.status === "delivered") {
    return {
      label: "Complete order",
      targetStatus: "completed",
      description:
        "Closes the commercial order after delivery has been confirmed.",
      tone: "positive",
    };
  }

  return null;
}

function fulfillmentOperation(
  fulfillment: AdminWarehouseFulfillment,
): FulfillmentOperation | null {
  switch (fulfillment.status) {
    case "allocated":
    case "received":
      return {
        label: "Start picking",
        path: "start-picking",
      };

    case "picking":
      return {
        label: "Mark packed",
        path: "mark-packed",
      };

    case "packed":
      return {
        label: "Ready for handoff",
        path: "ready-for-handoff",
      };

    default:
      return null;
  }
}

async function optionalAdminFetch<T>(path: string): Promise<T | null> {
  try {
    return await adminFetch<T>(path);
  } catch (value: unknown) {
    if (
      value instanceof AdminRequestError &&
      (value.status === 404 || value.status === 403)
    ) {
      return null;
    }

    throw value;
  }
}

export default function AdminOrderDrawer({
  orderId,
  portal,
  mode,
  onClose,
  onUpdated,
}: AdminOrderDrawerProps) {
  const principal = useAdminSession();

  const permissions = principal.staff.permissions;
  const canOrderManage = permissions.includes("admin.order.manage");
  const canWarehouseRead = permissions.includes("admin.warehouse.read");
  const canWarehouseManage = permissions.includes("admin.warehouse.manage");
  const canDeliveryRead = permissions.includes("admin.delivery.read");
  const canDeliveryManage = permissions.includes("admin.delivery.manage");
  const canConfirmReceipt = permissions.includes(
    "admin.delivery.confirm_receipt",
  );
  const canInvoiceRead = permissions.includes("admin.invoice.read");

  const [order, setOrder] = useState<AdminOrderDetail | null>(null);
  const [fulfillments, setFulfillments] = useState<AdminWarehouseFulfillment[]>([]);
  const [tracking, setTracking] = useState<AdminDeliveryTrackingDetail | null>(null);
  const [invoice, setInvoice] = useState<AdminInvoice | null>(null);
  const [invoiceLoaded, setInvoiceLoaded] = useState(false);
  const [loading, setLoading] = useState(false);
  const [invoiceLoading, setInvoiceLoading] = useState(false);
  const [operationLoading, setOperationLoading] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [lastSyncedAt, setLastSyncedAt] = useState<Date | null>(null);

  const [deliveryMode, setDeliveryMode] = useState<
    "courier" | "self_pickup" | "community_rider"
  >("courier");
  const [courierName, setCourierName] = useState("");
  const [providerCode, setProviderCode] = useState("");
  const [providerShipmentId, setProviderShipmentId] = useState("");
  const [courierReference, setCourierReference] = useState("");
  const [riderReference, setRiderReference] = useState("");
  const [trackingNumber, setTrackingNumber] = useState("");
  const [trackingUrl, setTrackingUrl] = useState("");

  const [handoffReference, setHandoffReference] = useState("");
  const [dispatchMessage, setDispatchMessage] = useState("");

  const [trackingEventCode, setTrackingEventCode] = useState("in_transit");
  const [trackingEventStatus, setTrackingEventStatus] = useState("in_transit");
  const [trackingPublicMessage, setTrackingPublicMessage] = useState("");
  const [trackingInternalMessage, setTrackingInternalMessage] = useState("");
  const [trackingCity, setTrackingCity] = useState("");
  const [trackingLocation, setTrackingLocation] = useState("");

  const [providerDeliveredStatus, setProviderDeliveredStatus] =
    useState("delivered");
  const [providerDeliveredMessage, setProviderDeliveredMessage] = useState("");
  const [receiptNote, setReceiptNote] = useState("");

  const loadDetail = useCallback(async (options?: { silent?: boolean }) => {
    if (!orderId) {
      return;
    }

    if (!options?.silent) {
      setLoading(true);
      setError("");
      setInvoice(null);
      setInvoiceLoaded(false);
    }

    try {
      const [orderResponse, fulfillmentResponse, trackingResponse] =
        await Promise.all([
          adminFetch<AdminOrderResponse>(`/orders/${orderId}`),
          canWarehouseRead
            ? optionalAdminFetch<AdminWarehouseFulfillmentsResponse>(
                `/warehouse/orders/${orderId}/fulfillments`,
              )
            : Promise.resolve(null),
          canDeliveryRead
            ? optionalAdminFetch<AdminDeliveryTrackingResponse>(
                `/delivery/orders/${orderId}/tracking`,
              )
            : Promise.resolve(null),
        ]);

      setOrder(orderResponse.data);
      setFulfillments(fulfillmentResponse?.data ?? []);
      setTracking(trackingResponse?.data ?? null);
      setLastSyncedAt(new Date());
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      if (!options?.silent) {
        setLoading(false);
      }
    }
  }, [canDeliveryRead, canWarehouseRead, orderId]);

  useEffect(() => {
    setDeliveryMode("courier");
    setCourierName("");
    setProviderCode("");
    setProviderShipmentId("");
    setCourierReference("");
    setRiderReference("");
    setTrackingNumber("");
    setTrackingUrl("");
    setHandoffReference("");
    setDispatchMessage("");
    setTrackingEventCode("in_transit");
    setTrackingEventStatus("in_transit");
    setTrackingPublicMessage("");
    setTrackingInternalMessage("");
    setTrackingCity("");
    setTrackingLocation("");
    setProviderDeliveredStatus("delivered");
    setProviderDeliveredMessage("");
    setReceiptNote("");
    setError("");
    setNotice("");
    setLastSyncedAt(null);

    if (!orderId) {
      setOrder(null);
      setFulfillments([]);
      setTracking(null);
      setInvoice(null);
      setInvoiceLoaded(false);
      setError("");
      setNotice("");
      setLastSyncedAt(null);
      return;
    }

    void loadDetail();
  }, [loadDetail, orderId]);

  useEffect(() => {
    if (!orderId) {
      return;
    }

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onClose();
      }
    }

    window.addEventListener("keydown", handleKeyDown);

    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [onClose, orderId]);

  useEffect(() => {
    if (!orderId || mode !== "shipments") {
      return;
    }

    const sync = () => {
      if (document.visibilityState !== "visible" || operationLoading) {
        return;
      }

      void loadDetail({ silent: true });
    };

    const intervalId = window.setInterval(sync, 5_000);
    const handleVisibility = () => {
      if (document.visibilityState === "visible") {
        sync();
      }
    };

    document.addEventListener("visibilitychange", handleVisibility);

    return () => {
      window.clearInterval(intervalId);
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, [loadDetail, mode, operationLoading, orderId]);

  const operation = useMemo(
    () => (order ? orderOperation(order) : null),
    [order],
  );

  async function transitionOrder(nextStatus: string) {
    if (!order) {
      return;
    }

    setOperationLoading(`order:${nextStatus}`);
    setError("");
    setNotice("");

    try {
      await adminFetch(`/orders/${order.id}/fulfillment`, {
        method: "PATCH",
        body: JSON.stringify({
          status: nextStatus,
          courier_name: "",
          courier_reference: "",
          tracking_number: "",
          tracking_url: "",
        }),
      });

      setNotice(`Order moved to ${titleCase(nextStatus)}.`);
      await loadDetail({ silent: true });
      onUpdated();
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  async function transitionFulfillment(
    fulfillment: AdminWarehouseFulfillment,
    action: FulfillmentOperation,
  ) {
    setOperationLoading(`fulfillment:${fulfillment.id}`);
    setError("");
    setNotice("");

    try {
      await adminFetch(
        `/warehouse/fulfillments/${fulfillment.id}/${action.path}`,
        {
          method: "POST",
          body: JSON.stringify({}),
        },
      );

      setNotice(`${action.label} completed.`);
      await loadDetail({ silent: true });
      onUpdated();
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  const readyForHandoffFulfillments = fulfillments.filter(
    (fulfillment) => fulfillment.status === "ready_for_handoff",
  );

  const allFulfillmentsReadyForShipment =
    fulfillments.length > 0 &&
    fulfillments.every(
      (fulfillment) => fulfillment.status === "ready_for_handoff",
    );

  const allFulfillmentsHandedOff =
    fulfillments.length > 0 &&
    fulfillments.every((fulfillment) => fulfillment.status === "handed_off");

  const shipmentStage = shipmentStatusLabel(tracking?.shipment.status);

  let shipmentNextAction = "No further shipment update";
  let shipmentNextHint = "This shipment does not currently require another delivery action.";

  if (!tracking) {
    if (!allFulfillmentsReadyForShipment) {
      shipmentNextAction = "Prepare warehouse fulfillment";
      shipmentNextHint =
        "Finish picking, packing and Ready for handoff first. The shipment can then be marked prepared.";
    } else if (canDeliveryManage) {
      shipmentNextAction = "Mark shipment prepared";
      shipmentNextHint =
        "Choose the delivery mode and shipment details below, then mark this shipment prepared.";
    } else {
      shipmentNextAction = "Shipment preparation requires permission";
      shipmentNextHint =
        "A staff member with delivery-management permission must mark this shipment prepared.";
    }
  } else if (tracking.shipment.status === "pending") {
    if (!allFulfillmentsHandedOff) {
      shipmentNextAction = "Record warehouse handoff";
      shipmentNextHint =
        "The shipment is prepared. Record the physical handoff before it can be marked shipped.";
    } else {
      shipmentNextAction = "Mark shipment shipped";
      shipmentNextHint =
        "Warehouse handoff is complete. Dispatching will update both shipment and order to Shipped.";
    }
  } else if (tracking.shipment.status === "shipped") {
    if (tracking.shipment.delivery_mode === "self_pickup") {
      shipmentNextAction = "Confirm receipt";
      shipmentNextHint =
        "Confirm the customer received the pickup to mark the order delivered.";
    } else {
      shipmentNextAction = "Update delivery progress";
      shipmentNextHint =
        "Add live tracking updates, or mark provider delivered when the courier reports delivery.";
    }
  } else if (tracking.shipment.status === "awaiting_confirmation") {
    shipmentNextAction = "Confirm delivery";
    shipmentNextHint =
      "The provider reports delivered. Confirm customer receipt to mark the order Delivered.";
  } else if (tracking.shipment.status === "delivered") {
    shipmentNextAction = order?.status === "delivered" ? "Complete order" : "Delivery complete";
    shipmentNextHint =
      order?.status === "delivered"
        ? "Receipt is confirmed. The commercial order can now be completed."
        : "The shipment delivery lifecycle is complete.";
  }

  async function prepareShipment() {
    if (!order) {
      return;
    }

    if (
      deliveryMode === "courier" &&
      !courierName.trim() &&
      !providerCode.trim()
    ) {
      setError("Courier shipments require a courier name or provider code.");
      return;
    }

    if (deliveryMode === "community_rider" && !riderReference.trim()) {
      setError("Community-rider shipments require a rider reference.");
      return;
    }

    setOperationLoading("delivery:prepare");
    setError("");
    setNotice("");

    try {
      await adminFetch(`/delivery/orders/${order.id}/shipments`, {
        method: "POST",
        body: JSON.stringify({
          delivery_mode: deliveryMode,
          provider_code: deliveryMode === "courier" ? providerCode.trim() : "",
          provider_shipment_id:
            deliveryMode === "courier" ? providerShipmentId.trim() : "",
          provider_status: "",
          courier_name: deliveryMode === "courier" ? courierName.trim() : "",
          courier_reference:
            deliveryMode === "courier" ? courierReference.trim() : "",
          rider_reference:
            deliveryMode === "community_rider" ? riderReference.trim() : "",
          tracking_number:
            deliveryMode === "courier" ? trackingNumber.trim() : "",
          tracking_url: deliveryMode === "courier" ? trackingUrl.trim() : "",
        }),
      });

      setNotice("Shipment prepared. Record the warehouse handoff next.");
      await loadDetail({ silent: true });
      onUpdated();
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  async function recordHandoff() {
    if (!tracking || readyForHandoffFulfillments.length === 0) {
      return;
    }

    setOperationLoading("delivery:handoff");
    setError("");
    setNotice("");

    try {
      await adminFetch("/warehouse/handoffs", {
        method: "POST",
        body: JSON.stringify({
          shipment_id: tracking.shipment.id,
          handoff_type: tracking.shipment.delivery_mode,
          reference: handoffReference.trim(),
          items: readyForHandoffFulfillments.map((fulfillment) => ({
            fulfillment_id: fulfillment.id,
            quantity: fulfillment.quantity,
          })),
        }),
      });

      setNotice("Warehouse handoff recorded.");
      setHandoffReference("");
      await loadDetail({ silent: true });
      onUpdated();
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  async function dispatchShipment() {
    if (!tracking) {
      return;
    }

    setOperationLoading("delivery:dispatch");
    setError("");
    setNotice("");

    try {
      await adminFetch(`/delivery/shipments/${tracking.shipment.id}/dispatch`, {
        method: "POST",
        body: JSON.stringify({
          message: dispatchMessage.trim(),
        }),
      });

      setNotice("Shipment dispatched. The order is now shipped.");
      setDispatchMessage("");
      await loadDetail({ silent: true });
      onUpdated();
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  async function addTrackingUpdate() {
    if (!tracking || !trackingEventCode.trim()) {
      setError("A tracking event code is required.");
      return;
    }

    setOperationLoading("delivery:event");
    setError("");
    setNotice("");

    try {
      await adminFetch(
        `/delivery/shipments/${tracking.shipment.id}/tracking-events`,
        {
          method: "POST",
          body: JSON.stringify({
            source: "admin",
            event_code: trackingEventCode.trim().toLowerCase(),
            status: trackingEventStatus.trim().toLowerCase(),
            message: trackingInternalMessage.trim(),
            public_message: trackingPublicMessage.trim(),
            city: trackingCity.trim(),
            location_name: trackingLocation.trim(),
            customer_visible: true,
          }),
        },
      );

      setNotice("Delivery tracking update recorded.");
      setTrackingPublicMessage("");
      setTrackingInternalMessage("");
      await loadDetail({ silent: true });
      onUpdated();
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  async function markProviderDelivered() {
    if (!tracking) {
      return;
    }

    setOperationLoading("delivery:provider-delivered");
    setError("");
    setNotice("");

    try {
      await adminFetch(
        `/delivery/shipments/${tracking.shipment.id}/provider-delivered`,
        {
          method: "POST",
          body: JSON.stringify({
            provider_status: providerDeliveredStatus.trim(),
            message: providerDeliveredMessage.trim(),
          }),
        },
      );

      setNotice(
        "Provider delivery recorded. Customer receipt confirmation is still required.",
      );
      setProviderDeliveredMessage("");
      await loadDetail({ silent: true });
      onUpdated();
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  async function confirmReceipt() {
    if (!order) {
      return;
    }

    setOperationLoading("delivery:confirm-receipt");
    setError("");
    setNotice("");

    try {
      await adminFetch(`/delivery/orders/${order.id}/confirm-receipt`, {
        method: "POST",
        body: JSON.stringify({
          note: receiptNote.trim(),
        }),
      });

      setNotice("Receipt confirmed. The order is now delivered.");
      setReceiptNote("");
      await loadDetail({ silent: true });
      onUpdated();
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  async function loadInvoice() {
    if (!order || !canInvoiceRead) {
      return;
    }

    setInvoiceLoading(true);
    setError("");

    try {
      const response = await optionalAdminFetch<AdminInvoiceResponse>(
        `/orders/${order.id}/invoice`,
      );

      setInvoice(response?.data ?? null);
      setInvoiceLoaded(true);
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setInvoiceLoading(false);
    }
  }

  const timeline = useMemo(() => {
    if (!order) {
      return [];
    }

    const entries: Array<{
      key: string;
      label: string;
      value: string;
      detail?: string;
      delivery?: boolean;
    }> = [
      { key: "placed", label: "Order placed", value: order.created_at },
      { key: "paid", label: "Payment received", value: order.paid_at ?? "" },
      { key: "confirmed", label: "Confirmed", value: order.confirmed_at ?? "" },
      { key: "processing", label: "Processing", value: order.processing_at ?? "" },
      { key: "shipped", label: "Shipped", value: order.shipped_at ?? "" },
      { key: "delivered", label: "Delivered", value: order.delivered_at ?? "" },
      { key: "completed", label: "Completed", value: order.completed_at ?? "" },
      { key: "cancelled", label: "Cancelled", value: order.cancelled_at ?? "" },
    ].filter((entry) => Boolean(entry.value));

    for (const event of tracking?.events ?? []) {
      entries.push({
        key: `delivery:${event.id}`,
        label: `Delivery · ${titleCase(event.event_code)}`,
        value: event.occurred_at || event.created_at,
        detail:
          event.message ||
          (event.status ? `Status: ${titleCase(event.status)}` : "Shipment update"),
        delivery: true,
      });
    }

    return entries.sort(
      (left, right) =>
        new Date(left.value).getTime() - new Date(right.value).getTime(),
    );
  }, [order, tracking]);

  if (!orderId) {
    return null;
  }

  return (
    <div className={styles.drawerLayer} role="presentation">
      <button
        type="button"
        className={styles.drawerBackdrop}
        onClick={onClose}
        aria-label="Close order details"
      />

      <aside
        className={styles.drawer}
        role="dialog"
        aria-modal="true"
        aria-labelledby="order-drawer-title"
      >
        <header className={styles.drawerHeader}>
          <div>
            <span className={styles.drawerEyebrow}>
              {mode === "shipments"
                ? "Shipment control"
                : mode === "fulfillment"
                  ? "Fulfillment workspace"
                  : "Order details"}
            </span>
            <h2 id="order-drawer-title">
              {order?.order_number ?? "Loading order…"}
            </h2>
          </div>

          <button
            type="button"
            className={styles.drawerClose}
            onClick={onClose}
            aria-label="Close order details"
          >
            ×
          </button>
        </header>

        <div className={styles.drawerScroll}>
          {loading && !order ? (
            <div className={styles.detailSkeleton} aria-hidden="true">
              <span />
              <span />
              <span />
              <span />
              <span />
            </div>
          ) : error && !order ? (
            <div className={styles.drawerError} role="alert">
              <strong>Order could not be opened</strong>
              <p>{error}</p>
              <button type="button" onClick={() => void loadDetail()}>
                Try again
              </button>
            </div>
          ) : order ? (
            <>
              <section className={styles.orderHero}>
                <div className={styles.heroStatusRow}>
                  <span className={`${styles.statusPill} ${statusTone(order.status)}`}>
                    {titleCase(order.status)}
                  </span>
                  <span className={`${styles.statusPill} ${statusTone(order.payment_status)}`}>
                    {titleCase(order.payment_status)}
                  </span>
                  <span className={`${styles.statusPill} ${order.order_type === "sourcing" ? styles.toneViolet : styles.toneNeutral}`}>
                    {order.order_type === "sourcing" ? "Sourcing order" : "Standard order"}
                  </span>
                </div>

                <div className={styles.heroAmount}>
                  <span>Total</span>
                  <strong>{formatMoney(order.total_amount, order.currency)}</strong>
                </div>

                <div className={styles.heroFacts}>
                  <span>
                    <small>Customer</small>
                    <strong>{order.customer_name || "Guest customer"}</strong>
                  </span>
                  <span>
                    <small>Payment</small>
                    <strong>{titleCase(order.payment_method)}</strong>
                  </span>
                  <span>
                    <small>Delivery</small>
                    <strong>{titleCase(order.delivery_method)}</strong>
                  </span>
                </div>
              </section>

              {notice ? (
                <div className={styles.noticeBanner} role="status">
                  <span aria-hidden="true">✓</span>
                  {notice}
                </div>
              ) : null}

              {error ? (
                <div className={styles.inlineError} role="alert">
                  {error}
                </div>
              ) : null}

              {mode !== "shipments" && operation && canOrderManage ? (
                <section className={`${styles.detailSection} ${styles.actionSection}`}>
                  <div className={styles.sectionHeading}>
                    <div>
                      <span>Next order action</span>
                      <h3>{operation.label}</h3>
                    </div>
                  </div>

                  <p className={styles.actionDescription}>{operation.description}</p>

                  <button
                    type="button"
                    className={
                      operation.tone === "positive"
                        ? styles.positiveActionButton
                        : styles.primaryActionButton
                    }
                    onClick={() => void transitionOrder(operation.targetStatus)}
                    disabled={Boolean(operationLoading)}
                  >
                    {operationLoading === `order:${operation.targetStatus}`
                      ? "Updating…"
                      : operation.label}
                  </button>
                </section>
              ) : null}

              {mode === "shipments" && (canWarehouseRead || canDeliveryRead) ? (
                <section className={`${styles.detailSection} ${styles.deliveryControlSection}`}>
                  <div className={styles.sectionHeading}>
                    <div>
                      <span>Delivery control</span>
                      <h3>Fulfillment, shipment & delivery updates</h3>
                    </div>
                    <span className={`${styles.statusPill} ${statusTone(order.status)}`}>
                      {titleCase(order.status)}
                    </span>
                  </div>

                  <div className={styles.shipmentStagePanel}>
                    <div className={styles.shipmentStageCurrent}>
                      <span>Current shipment stage</span>
                      <strong>{shipmentStage}</strong>
                      <small>
                        Order: {titleCase(order.status)}
                        {tracking?.shipment.tracking_number
                          ? ` · ${tracking.shipment.tracking_number}`
                          : ""}
                      </small>
                    </div>
                    <div className={styles.shipmentStageNext}>
                      <span>Next update</span>
                      <strong>{shipmentNextAction}</strong>
                      <small>{shipmentNextHint}</small>
                    </div>
                    <div className={styles.shipmentSyncState}>
                      <span className={styles.liveDot} aria-hidden="true" />
                      <span>
                        Live sync
                        {lastSyncedAt
                          ? ` · ${formatDateTime(lastSyncedAt.toISOString())}`
                          : ""}
                      </span>
                    </div>
                  </div>

                  <div className={styles.deliveryProgress} aria-label="Delivery progress">
                    <div
                      className={`${styles.deliveryProgressStep} ${
                        fulfillments.length > 0 ? styles.progressDone : styles.progressCurrent
                      }`}
                    >
                      <span>1</span>
                      <strong>Fulfillment</strong>
                      <small>
                        {fulfillments.length > 0
                          ? `${fulfillments.length} record${fulfillments.length === 1 ? "" : "s"}`
                          : "Waiting"}
                      </small>
                    </div>
                    <div
                      className={`${styles.deliveryProgressStep} ${
                        tracking ? styles.progressDone : styles.progressPending
                      }`}
                    >
                      <span>2</span>
                      <strong>Shipment</strong>
                      <small>{shipmentStatusLabel(tracking?.shipment.status)}</small>
                    </div>
                    <div
                      className={`${styles.deliveryProgressStep} ${
                        allFulfillmentsHandedOff
                          ? styles.progressDone
                          : tracking
                            ? styles.progressCurrent
                            : styles.progressPending
                      }`}
                    >
                      <span>3</span>
                      <strong>Handoff</strong>
                      <small>{allFulfillmentsHandedOff ? "Complete" : "Pending"}</small>
                    </div>
                    <div
                      className={`${styles.deliveryProgressStep} ${
                        ["shipped", "delivered", "completed"].includes(order.status)
                          ? styles.progressDone
                          : allFulfillmentsHandedOff
                            ? styles.progressCurrent
                            : styles.progressPending
                      }`}
                    >
                      <span>4</span>
                      <strong>Dispatch</strong>
                      <small>
                        {["shipped", "delivered", "completed"].includes(order.status)
                          ? "Shipped"
                          : "Pending"}
                      </small>
                    </div>
                    <div
                      className={`${styles.deliveryProgressStep} ${
                        ["delivered", "completed"].includes(order.status)
                          ? styles.progressDone
                          : order.status === "shipped"
                            ? styles.progressCurrent
                            : styles.progressPending
                      }`}
                    >
                      <span>5</span>
                      <strong>Receipt</strong>
                      <small>
                        {["delivered", "completed"].includes(order.status)
                          ? "Confirmed"
                          : tracking?.shipment.status === "awaiting_confirmation"
                            ? "Confirm now"
                            : "Pending"}
                      </small>
                    </div>
                    <div
                      className={`${styles.deliveryProgressStep} ${
                        order.status === "completed"
                          ? styles.progressDone
                          : order.status === "delivered"
                            ? styles.progressCurrent
                            : styles.progressPending
                      }`}
                    >
                      <span>6</span>
                      <strong>Complete</strong>
                      <small>{order.status === "completed" ? "Done" : "Pending"}</small>
                    </div>
                  </div>

                  {canWarehouseRead ? (
                    <div className={styles.deliveryControlGroup}>
                      <div className={styles.deliveryControlGroupHeading}>
                        <div>
                          <span>Warehouse preparation</span>
                          <strong>Prepare every fulfillment for handoff</strong>
                        </div>
                        <span className={styles.sectionCount}>{fulfillments.length}</span>
                      </div>

                      {fulfillments.length === 0 ? (
                        <div className={styles.quietState}>
                          <strong>No warehouse fulfillment yet</strong>
                          <p>
                            Allocation appears here once order quantities are assigned to warehouse stock or inbound supply.
                          </p>
                        </div>
                      ) : (
                        <div className={styles.fulfillmentList}>
                          {fulfillments.map((fulfillment) => {
                            const nextAction = fulfillmentOperation(fulfillment);

                            return (
                              <article key={fulfillment.id} className={styles.fulfillmentCard}>
                                <div className={styles.fulfillmentTop}>
                                  <div>
                                    <strong>
                                      {fulfillment.product_name ||
                                        fulfillment.order_item_sku ||
                                        "Order item"}
                                    </strong>
                                    <span>
                                      Qty {fulfillment.quantity} · {fulfillment.warehouse_name ||
                                        fulfillment.warehouse_code ||
                                        "Warehouse"}
                                    </span>
                                  </div>
                                  <span
                                    className={`${styles.statusPill} ${statusTone(
                                      fulfillment.status,
                                    )}`}
                                  >
                                    {titleCase(fulfillment.status)}
                                  </span>
                                </div>

                                <div className={styles.fulfillmentMeta}>
                                  <span>
                                    <small>Source</small>
                                    <strong>{titleCase(fulfillment.source)}</strong>
                                  </span>
                                  {fulfillment.inbound_reference ? (
                                    <span>
                                      <small>Inbound</small>
                                      <strong>{fulfillment.inbound_reference}</strong>
                                    </span>
                                  ) : null}
                                </div>

                                {nextAction && canWarehouseManage ? (
                                  <button
                                    type="button"
                                    className={styles.compactActionButton}
                                    onClick={() =>
                                      void transitionFulfillment(fulfillment, nextAction)
                                    }
                                    disabled={Boolean(operationLoading)}
                                  >
                                    {operationLoading === `fulfillment:${fulfillment.id}`
                                      ? "Updating…"
                                      : nextAction.label}
                                  </button>
                                ) : null}
                              </article>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  ) : null}

                  {canDeliveryRead ? (
                    <div className={styles.deliveryControlGroup}>
                      <div className={styles.deliveryControlGroupHeading}>
                        <div>
                          <span>Shipment</span>
                          <strong>
                            {tracking ? "Shipment & tracking" : "Prepare delivery shipment"}
                          </strong>
                        </div>
                        {tracking ? (
                          <span
                            className={`${styles.statusPill} ${statusTone(
                              tracking.shipment.status,
                            )}`}
                          >
                            {titleCase(tracking.shipment.status)}
                          </span>
                        ) : null}
                      </div>

                      {tracking ? (
                        <article className={styles.shipmentCard}>
                          <div className={styles.fulfillmentTop}>
                            <div>
                              <strong>
                                {tracking.shipment.courier_name ||
                                  tracking.shipment.provider_code ||
                                  titleCase(tracking.shipment.delivery_mode)}
                              </strong>
                              <span>
                                {tracking.shipment.tracking_number ||
                                  tracking.shipment.courier_reference ||
                                  tracking.shipment.rider_reference ||
                                  "Tracking reference pending"}
                              </span>
                            </div>
                          </div>

                          <div className={styles.fulfillmentMeta}>
                            <span>
                              <small>Mode</small>
                              <strong>{titleCase(tracking.shipment.delivery_mode)}</strong>
                            </span>
                            <span>
                              <small>Provider status</small>
                              <strong>{tracking.shipment.provider_status || "—"}</strong>
                            </span>
                          </div>

                          {tracking.shipment.tracking_url ? (
                            <a
                              className={styles.trackingLink}
                              href={tracking.shipment.tracking_url}
                              target="_blank"
                              rel="noreferrer"
                            >
                              Open courier tracking ↗
                            </a>
                          ) : null}
                        </article>
                      ) : (
                        <div className={styles.quietState}>
                          <strong>No shipment yet</strong>
                          <p>
                            Shipment creation is controlled by backend readiness rules. The order must be Processing, payment-ready, and every warehouse fulfillment must be Ready for handoff.
                          </p>
                        </div>
                      )}

                      {!tracking && order.status === "processing" && !allFulfillmentsReadyForShipment ? (
                        <div className={styles.deliveryActionCard}>
                          <div>
                            <span className={styles.deliveryStep}>Shipment setup blocked</span>
                            <strong>Warehouse preparation is not complete</strong>
                            <p>
                              This order is Processing, but the shipment cannot be created until all allocated quantities reach Ready for handoff.
                            </p>
                          </div>

                          <div className={styles.fulfillmentMeta}>
                            <span>
                              <small>Order</small>
                              <strong>{titleCase(order.status)}</strong>
                            </span>
                            <span>
                              <small>Payment</small>
                              <strong>{titleCase(order.payment_status)}</strong>
                            </span>
                            <span>
                              <small>Warehouse records</small>
                              <strong>{fulfillments.length}</strong>
                            </span>
                            <span>
                              <small>Ready for handoff</small>
                              <strong>{readyForHandoffFulfillments.length}/{fulfillments.length || 0}</strong>
                            </span>
                          </div>

                          <Link
                            className={styles.openShipmentButton}
                            href={`/${portal}/fulfillment`}
                          >
                            Open fulfillment →
                          </Link>
                        </div>
                      ) : null}

                      {!tracking && order.status !== "processing" ? (
                        <div className={styles.deliveryActionCard}>
                          <div>
                            <span className={styles.deliveryStep}>Shipment setup blocked</span>
                            <strong>Order is not ready for shipment</strong>
                            <p>
                              Shipment creation starts only after the order reaches Processing. Current order status: {titleCase(order.status)}.
                            </p>
                          </div>
                        </div>
                      ) : null}

                      {!tracking && order.status === "processing" && allFulfillmentsReadyForShipment && !canDeliveryManage ? (
                        <div className={styles.deliveryActionCard}>
                          <div>
                            <span className={styles.deliveryStep}>Permission required</span>
                            <strong>Shipment is ready to prepare</strong>
                            <p>
                              Your current staff permissions allow shipment viewing but not shipment management.
                            </p>
                          </div>
                        </div>
                      ) : null}

                      {!tracking &&
                      canDeliveryManage &&
                      order.status === "processing" &&
                      allFulfillmentsReadyForShipment ? (
                        <div className={styles.deliveryActionCard}>
                          <div>
                            <span className={styles.deliveryStep}>1 · Shipment setup</span>
                            <strong>Prepare delivery shipment</strong>
                            <p>
                              Choose the physical delivery mode. Checkout speed such as Standard or Same day remains separate.
                            </p>
                          </div>

                          <div className={styles.deliveryFormGrid}>
                            <label className={styles.fieldLabel}>
                              Delivery mode
                              <select
                                value={deliveryMode}
                                onChange={(event) =>
                                  setDeliveryMode(
                                    event.target.value as
                                      | "courier"
                                      | "self_pickup"
                                      | "community_rider",
                                  )
                                }
                              >
                                <option value="courier">Courier</option>
                                <option value="community_rider">Community rider</option>
                                <option value="self_pickup">Self pickup</option>
                              </select>
                            </label>

                            {deliveryMode === "courier" ? (
                              <>
                                <label className={styles.fieldLabel}>
                                  Courier name
                                  <input
                                    value={courierName}
                                    onChange={(event) => setCourierName(event.target.value)}
                                    placeholder="Pathao, RedX, etc."
                                    maxLength={120}
                                  />
                                </label>
                                <label className={styles.fieldLabel}>
                                  Provider code
                                  <input
                                    value={providerCode}
                                    onChange={(event) => setProviderCode(event.target.value)}
                                    placeholder="Optional integration code"
                                    maxLength={60}
                                  />
                                </label>
                                <label className={styles.fieldLabel}>
                                  Provider shipment ID
                                  <input
                                    value={providerShipmentId}
                                    onChange={(event) => setProviderShipmentId(event.target.value)}
                                    maxLength={160}
                                  />
                                </label>
                                <label className={styles.fieldLabel}>
                                  Courier reference
                                  <input
                                    value={courierReference}
                                    onChange={(event) => setCourierReference(event.target.value)}
                                    maxLength={160}
                                  />
                                </label>
                                <label className={styles.fieldLabel}>
                                  Tracking number
                                  <input
                                    value={trackingNumber}
                                    onChange={(event) => setTrackingNumber(event.target.value)}
                                    maxLength={160}
                                  />
                                </label>
                                <label className={`${styles.fieldLabel} ${styles.fieldWide}`}>
                                  Tracking URL
                                  <input
                                    type="url"
                                    value={trackingUrl}
                                    onChange={(event) => setTrackingUrl(event.target.value)}
                                    placeholder="https://…"
                                    maxLength={1000}
                                  />
                                </label>
                              </>
                            ) : null}

                            {deliveryMode === "community_rider" ? (
                              <label className={`${styles.fieldLabel} ${styles.fieldWide}`}>
                                Rider reference
                                <input
                                  value={riderReference}
                                  onChange={(event) => setRiderReference(event.target.value)}
                                  placeholder="Required rider reference"
                                  maxLength={160}
                                />
                              </label>
                            ) : null}
                          </div>

                          <button
                            type="button"
                            className={styles.primaryActionButton}
                            onClick={() => void prepareShipment()}
                            disabled={Boolean(operationLoading)}
                          >
                            {operationLoading === "delivery:prepare"
                              ? "Preparing…"
                              : "Mark prepared"}
                          </button>
                        </div>
                      ) : null}

                      {tracking &&
                      canWarehouseManage &&
                      tracking.shipment.status === "pending" &&
                      readyForHandoffFulfillments.length > 0 ? (
                        <div className={styles.deliveryActionCard}>
                          <div>
                            <span className={styles.deliveryStep}>2 · Warehouse handoff</span>
                            <strong>Record physical handoff</strong>
                            <p>
                              Assign every ready warehouse quantity to this shipment before dispatch.
                            </p>
                          </div>

                          <label className={styles.fieldLabel}>
                            Handoff reference
                            <input
                              value={handoffReference}
                              onChange={(event) => setHandoffReference(event.target.value)}
                              placeholder="Courier bag, rider or pickup reference"
                              maxLength={160}
                            />
                          </label>

                          <button
                            type="button"
                            className={styles.primaryActionButton}
                            onClick={() => void recordHandoff()}
                            disabled={Boolean(operationLoading)}
                          >
                            {operationLoading === "delivery:handoff"
                              ? "Recording…"
                              : `Record handoff (${readyForHandoffFulfillments.length})`}
                          </button>
                        </div>
                      ) : null}

                      {tracking &&
                      canDeliveryManage &&
                      tracking.shipment.status === "pending" &&
                      allFulfillmentsHandedOff ? (
                        <div className={styles.deliveryActionCard}>
                          <div>
                            <span className={styles.deliveryStep}>3 · Dispatch</span>
                            <strong>Dispatch shipment</strong>
                            <p>
                              This is the backend-owned transition from Processing to Shipped.
                            </p>
                          </div>

                          <label className={styles.fieldLabel}>
                            Dispatch note
                            <input
                              value={dispatchMessage}
                              onChange={(event) => setDispatchMessage(event.target.value)}
                              placeholder="Optional dispatch note"
                              maxLength={1000}
                            />
                          </label>

                          <button
                            type="button"
                            className={styles.positiveActionButton}
                            onClick={() => void dispatchShipment()}
                            disabled={Boolean(operationLoading)}
                          >
                            {operationLoading === "delivery:dispatch"
                              ? "Dispatching…"
                              : "Mark shipped"}
                          </button>
                        </div>
                      ) : null}

                      {tracking &&
                      canDeliveryManage &&
                      tracking.shipment.status !== "cancelled" ? (
                        <details className={styles.deliveryDetails}>
                          <summary>4 · Add manual tracking update</summary>

                          <div className={styles.deliveryFormGrid}>
                            <label className={styles.fieldLabel}>
                              Event code
                              <input
                                value={trackingEventCode}
                                onChange={(event) => setTrackingEventCode(event.target.value)}
                                placeholder="in_transit"
                                maxLength={80}
                              />
                            </label>

                            <label className={styles.fieldLabel}>
                              Status
                              <input
                                value={trackingEventStatus}
                                onChange={(event) => setTrackingEventStatus(event.target.value)}
                                placeholder="in_transit"
                                maxLength={80}
                              />
                            </label>

                            <label className={styles.fieldLabel}>
                              City
                              <input
                                value={trackingCity}
                                onChange={(event) => setTrackingCity(event.target.value)}
                                placeholder="Dhaka"
                                maxLength={120}
                              />
                            </label>

                            <label className={styles.fieldLabel}>
                              Location
                              <input
                                value={trackingLocation}
                                onChange={(event) => setTrackingLocation(event.target.value)}
                                placeholder="Local hub / delivery area"
                                maxLength={160}
                              />
                            </label>

                            <label className={`${styles.fieldLabel} ${styles.fieldWide}`}>
                              Customer message
                              <textarea
                                value={trackingPublicMessage}
                                onChange={(event) => setTrackingPublicMessage(event.target.value)}
                                placeholder="Visible delivery update for the customer"
                                maxLength={500}
                                rows={3}
                              />
                            </label>

                            <label className={`${styles.fieldLabel} ${styles.fieldWide}`}>
                              Internal note
                              <textarea
                                value={trackingInternalMessage}
                                onChange={(event) => setTrackingInternalMessage(event.target.value)}
                                placeholder="Optional internal Admin note"
                                maxLength={1000}
                                rows={3}
                              />
                            </label>
                          </div>

                          <button
                            type="button"
                            className={styles.primaryActionButton}
                            onClick={() => void addTrackingUpdate()}
                            disabled={Boolean(operationLoading)}
                          >
                            {operationLoading === "delivery:event"
                              ? "Saving update…"
                              : "Save tracking update"}
                          </button>
                        </details>
                      ) : null}

                      {tracking &&
                      canDeliveryManage &&
                      tracking.shipment.status === "shipped" &&
                      tracking.shipment.delivery_mode !== "self_pickup" ? (
                        <div className={styles.deliveryActionCard}>
                          <div>
                            <span className={styles.deliveryStep}>5 · Provider delivery</span>
                            <strong>Courier reports delivered</strong>
                            <p>
                              This moves the shipment to Awaiting confirmation without finalizing the order.
                            </p>
                          </div>

                          <div className={styles.deliveryFormGrid}>
                            <label className={styles.fieldLabel}>
                              Provider status
                              <input
                                value={providerDeliveredStatus}
                                onChange={(event) =>
                                  setProviderDeliveredStatus(event.target.value)
                                }
                                maxLength={80}
                              />
                            </label>

                            <label className={styles.fieldLabel}>
                              Provider message
                              <input
                                value={providerDeliveredMessage}
                                onChange={(event) =>
                                  setProviderDeliveredMessage(event.target.value)
                                }
                                placeholder="Optional provider message"
                                maxLength={1000}
                              />
                            </label>
                          </div>

                          <button
                            type="button"
                            className={styles.primaryActionButton}
                            onClick={() => void markProviderDelivered()}
                            disabled={Boolean(operationLoading)}
                          >
                            {operationLoading === "delivery:provider-delivered"
                              ? "Updating…"
                              : "Mark provider delivered"}
                          </button>
                        </div>
                      ) : null}

                      {tracking &&
                      canConfirmReceipt &&
                      order.status === "shipped" &&
                      (tracking.shipment.status === "shipped" ||
                        tracking.shipment.status === "awaiting_confirmation") ? (
                        <div
                          className={`${styles.deliveryActionCard} ${styles.deliveryConfirmCard}`}
                        >
                          <div>
                            <span className={styles.deliveryStep}>6 · Receipt confirmation</span>
                            <strong>Confirm customer received the order</strong>
                            <p>
                              This is the authoritative Shipped → Delivered transition. COD collection is recorded here too.
                            </p>
                          </div>

                          <label className={styles.fieldLabel}>
                            Confirmation note
                            <textarea
                              value={receiptNote}
                              onChange={(event) => setReceiptNote(event.target.value)}
                              placeholder="Optional confirmation note"
                              maxLength={1000}
                              rows={3}
                            />
                          </label>

                          <button
                            type="button"
                            className={styles.positiveActionButton}
                            onClick={() => void confirmReceipt()}
                            disabled={Boolean(operationLoading)}
                          >
                            {operationLoading === "delivery:confirm-receipt"
                              ? "Confirming…"
                              : "Mark delivered"}
                          </button>
                        </div>
                      ) : null}

                      {order.status === "delivered" && canOrderManage ? (
                        <div
                          className={`${styles.deliveryActionCard} ${styles.deliveryCompleteCard}`}
                        >
                          <div>
                            <span className={styles.deliveryStep}>7 · Completion</span>
                            <strong>Complete order</strong>
                            <p>
                              Finalize the order after customer receipt has been confirmed.
                            </p>
                          </div>

                          <button
                            type="button"
                            className={styles.positiveActionButton}
                            onClick={() => void transitionOrder("completed")}
                            disabled={Boolean(operationLoading)}
                          >
                            {operationLoading === "order:completed"
                              ? "Completing…"
                              : "Complete order"}
                          </button>
                        </div>
                      ) : null}

                      {tracking && tracking.events.length > 0 ? (
                        <details className={styles.deliveryActivity}>
                          <summary>
                            Recent delivery activity
                            <span>{tracking.events.length}</span>
                          </summary>
                          <div className={styles.trackingTimeline}>
                            {tracking.events
                              .slice()
                              .reverse()
                              .slice(0, 10)
                              .map((event) => (
                                <article key={event.id} className={styles.trackingEvent}>
                                  <span className={styles.timelineDot} />
                                  <div>
                                    <strong>
                                      {event.message || titleCase(event.event_code)}
                                    </strong>
                                    <span>
                                      {formatDateTime(event.occurred_at)}
                                      {event.status ? ` · ${titleCase(event.status)}` : ""}
                                    </span>
                                  </div>
                                </article>
                              ))}
                          </div>
                        </details>
                      ) : null}
                    </div>
                  ) : null}
                </section>
              ) : null}

              <section className={styles.detailSection}>
                <div className={styles.sectionHeading}>
                  <div>
                    <span>Order contents</span>
                    <h3>{order.items.length} line item{order.items.length === 1 ? "" : "s"}</h3>
                  </div>
                </div>

                <div className={styles.itemList}>
                  {order.items.map((item) => (
                    <article key={item.id} className={styles.itemRow}>
                      <div>
                        <strong>{item.product_name}</strong>
                        <span>
                          {item.sku || "No SKU"} · Qty {item.quantity}
                          {item.minimum_order_quantity > 1
                            ? ` · MOQ ${item.minimum_order_quantity}`
                            : ""}
                        </span>
                      </div>
                      <strong>{formatMoney(item.line_total_amount, item.currency)}</strong>
                    </article>
                  ))}
                </div>

                <div className={styles.moneyBreakdown}>
                  <span><small>Subtotal</small><strong>{formatMoney(order.subtotal_amount, order.currency)}</strong></span>
                  <span><small>Discount</small><strong>−{formatMoney(order.discount_amount, order.currency)}</strong></span>
                  <span><small>Shipping</small><strong>{formatMoney(order.shipping_amount, order.currency)}</strong></span>
                  <span className={styles.moneyTotal}><small>Total</small><strong>{formatMoney(order.total_amount, order.currency)}</strong></span>
                </div>
              </section>

              {mode === "orders" && canDeliveryRead ? (
                <section className={styles.detailSection}>
                  <div className={styles.sectionHeading}>
                    <div>
                      <span>Shipment</span>
                      <h3>Delivery status</h3>
                    </div>
                    <span
                      className={`${styles.statusPill} ${statusTone(
                        tracking?.shipment.status ?? "not_prepared",
                      )}`}
                    >
                      {tracking ? titleCase(tracking.shipment.status) : "Not prepared"}
                    </span>
                  </div>

                  <div className={styles.shipmentSummaryGrid}>
                    <span>
                      <small>Carrier / mode</small>
                      <strong>
                        {tracking?.shipment.courier_name ||
                          tracking?.shipment.provider_code ||
                          (tracking?.shipment.delivery_mode
                            ? titleCase(tracking.shipment.delivery_mode)
                            : "—")}
                      </strong>
                    </span>
                    <span>
                      <small>Tracking</small>
                      <strong>{tracking?.shipment.tracking_number || "—"}</strong>
                    </span>
                    <span>
                      <small>Latest delivery update</small>
                      <strong>
                        {tracking?.events?.length
                          ? titleCase(
                              tracking.events[tracking.events.length - 1].event_code,
                            )
                          : "No shipment event yet"}
                      </strong>
                    </span>
                  </div>

                  <Link
                    className={styles.openShipmentButton}
                    href={`/${portal}/shipments?q=${encodeURIComponent(
                      order.order_number,
                    )}`}
                  >
                    Open shipment control →
                  </Link>
                </section>
              ) : null}

              {mode === "fulfillment" && canWarehouseRead ? (
                <section className={styles.detailSection}>
                  <div className={styles.sectionHeading}>
                    <div>
                      <span>Warehouse fulfillment</span>
                      <h3>Picking, packing & handoff readiness</h3>
                    </div>
                    <span className={styles.sectionCount}>{fulfillments.length}</span>
                  </div>

                  {fulfillments.length === 0 ? (
                    <div className={styles.quietState}>
                      <strong>No warehouse fulfillment yet</strong>
                      <p>
                        Allocation appears here once the order is assigned to warehouse stock or inbound supply.
                      </p>
                    </div>
                  ) : (
                    <div className={styles.fulfillmentList}>
                      {fulfillments.map((fulfillment) => {
                        const nextAction = fulfillmentOperation(fulfillment);

                        return (
                          <article key={fulfillment.id} className={styles.fulfillmentCard}>
                            <div className={styles.fulfillmentTop}>
                              <div>
                                <strong>
                                  {fulfillment.product_name ||
                                    fulfillment.order_item_sku ||
                                    "Order item"}
                                </strong>
                                <span>
                                  Qty {fulfillment.quantity} · {fulfillment.warehouse_name ||
                                    fulfillment.warehouse_code ||
                                    "Warehouse"}
                                </span>
                              </div>
                              <span
                                className={`${styles.statusPill} ${statusTone(
                                  fulfillment.status,
                                )}`}
                              >
                                {titleCase(fulfillment.status)}
                              </span>
                            </div>

                            {nextAction && canWarehouseManage ? (
                              <button
                                type="button"
                                className={styles.compactActionButton}
                                onClick={() =>
                                  void transitionFulfillment(fulfillment, nextAction)
                                }
                                disabled={Boolean(operationLoading)}
                              >
                                {operationLoading === `fulfillment:${fulfillment.id}`
                                  ? "Updating…"
                                  : nextAction.label}
                              </button>
                            ) : null}
                          </article>
                        );
                      })}
                    </div>
                  )}
                </section>
              ) : null}



              <section className={styles.detailSection}>
                <div className={styles.sectionHeading}>
                  <div>
                    <span>Customer</span>
                    <h3>Contact & delivery</h3>
                  </div>
                </div>

                <div className={styles.infoGrid}>
                  <span>
                    <small>Name</small>
                    <strong>{order.customer_name || "Guest customer"}</strong>
                  </span>
                  <span>
                    <small>Phone</small>
                    <strong>{order.customer_phone || "—"}</strong>
                  </span>
                  <span>
                    <small>Email</small>
                    <strong>{order.customer_email || "—"}</strong>
                  </span>
                  <span className={styles.infoWide}>
                    <small>Shipping address</small>
                    <strong>
                      {[
                        order.shipping_address_line1,
                        order.shipping_address_line2,
                        order.shipping_area,
                        order.shipping_city,
                        order.shipping_postal_code,
                      ]
                        .filter(Boolean)
                        .join(", ")}
                    </strong>
                  </span>
                </div>
              </section>

              <section className={styles.detailSection}>
                <div className={styles.sectionHeading}>
                  <div>
                    <span>Lifecycle</span>
                    <h3>Order timeline</h3>
                  </div>
                </div>

                <div className={styles.orderTimeline}>
                  {timeline.map((entry) => (
                    <article
                      key={entry.key}
                      className={`${styles.orderTimelineItem} ${
                        entry.delivery ? styles.deliveryTimelineItem : ""
                      }`}
                    >
                      <span className={styles.timelineDot} />
                      <div>
                        <strong>{entry.label}</strong>
                        <span>{formatDateTime(entry.value)}</span>
                        {entry.detail ? <small>{entry.detail}</small> : null}
                      </div>
                    </article>
                  ))}
                </div>
              </section>

              {(order.payments.length > 0 || order.returns.length > 0 || order.refunds.length > 0) ? (
                <section className={styles.detailSection}>
                  <div className={styles.sectionHeading}>
                    <div>
                      <span>Financial & after-sales</span>
                      <h3>Payments, returns & refunds</h3>
                    </div>
                  </div>

                  {order.payments.length > 0 ? (
                    <div className={styles.compactRecords}>
                      {order.payments.map((payment) => (
                        <article key={payment.id}>
                          <div>
                            <strong>{titleCase(payment.provider)}</strong>
                            <span>{formatDateTime(payment.paid_at || payment.created_at)}</span>
                          </div>
                          <div className={styles.recordRight}>
                            <strong>{formatMoney(payment.amount, payment.currency)}</strong>
                            <span className={`${styles.statusPill} ${statusTone(payment.status)}`}>
                              {titleCase(payment.status)}
                            </span>
                          </div>
                        </article>
                      ))}
                    </div>
                  ) : null}

                  {order.returns.length > 0 ? (
                    <div className={styles.linkRecords}>
                      {order.returns.map((item) => (
                        <Link key={item.id} href={`/${portal}/returns`}>
                          <span>
                            <strong>{item.return_number}</strong>
                            <small>{formatDateTime(item.requested_at)}</small>
                          </span>
                          <span className={`${styles.statusPill} ${statusTone(item.status)}`}>
                            {titleCase(item.status)}
                          </span>
                        </Link>
                      ))}
                    </div>
                  ) : null}

                  {order.refunds.length > 0 ? (
                    <div className={styles.compactRecords}>
                      {order.refunds.map((refund) => (
                        <article key={refund.id}>
                          <div>
                            <strong>{refund.refund_number}</strong>
                            <span>{formatDateTime(refund.requested_at)}</span>
                          </div>
                          <div className={styles.recordRight}>
                            <strong>{formatMoney(refund.amount, refund.currency)}</strong>
                            <span className={`${styles.statusPill} ${statusTone(refund.status)}`}>
                              {titleCase(refund.status)}
                            </span>
                          </div>
                        </article>
                      ))}
                    </div>
                  ) : null}
                </section>
              ) : null}

              {canInvoiceRead ? (
                <section className={styles.detailSection}>
                  <div className={styles.sectionHeading}>
                    <div>
                      <span>Document</span>
                      <h3>Invoice</h3>
                    </div>
                  </div>

                  {!invoiceLoaded ? (
                    <button
                      type="button"
                      className={styles.secondaryActionButton}
                      onClick={() => void loadInvoice()}
                      disabled={invoiceLoading}
                    >
                      {invoiceLoading ? "Loading invoice…" : "Load invoice"}
                    </button>
                  ) : invoice ? (
                    <article className={styles.invoiceCard}>
                      <div>
                        <small>Invoice number</small>
                        <strong>{invoice.invoice_number}</strong>
                      </div>
                      <div>
                        <small>Issued</small>
                        <strong>{formatDateTime(invoice.issued_at)}</strong>
                      </div>
                      <div>
                        <small>Total</small>
                        <strong>{formatMoney(invoice.total_amount, invoice.currency)}</strong>
                      </div>
                    </article>
                  ) : (
                    <div className={styles.quietState}>
                      <strong>No invoice is available yet</strong>
                      <p>The invoice will appear here once one exists for this order.</p>
                    </div>
                  )}
                </section>
              ) : null}
            </>
          ) : null}
        </div>
      </aside>
    </div>
  );
}
