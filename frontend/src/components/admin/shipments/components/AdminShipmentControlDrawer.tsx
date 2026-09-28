"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import { adminFetch, AdminRequestError } from "@/lib/admin/api";
import type {
  AdminDeliveryTrackingDetail,
  AdminDeliveryTrackingResponse,
  AdminOrderDetail,
  AdminOrderResponse,
  AdminWarehouseFulfillment,
  AdminWarehouseFulfillmentsResponse,
} from "@/lib/admin/order-types";
import { formatMoney } from "@/lib/money/format";

import orderStyles from "@/components/admin/orders/css/AdminOrders.module.css";

type Props = {
  orderId: string | null;
  onClose: () => void;
  onUpdated: () => void;
};

type AdminWarehouse = {
  id: string;
  code: string;
  name: string;
  country_code: string;
  city?: string;
  address_line1?: string;
  status: string;
  is_default: boolean;
  allows_self_pickup: boolean;
};

type AdminWarehousesResponse = {
  data: AdminWarehouse[];
};

type FulfillmentAction = {
  label: string;
  path: "start-picking" | "mark-packed" | "ready-for-handoff";
};

const LIVE_SYNC_MS = 5_000;

function titleCase(value: string): string {
  return value
    .replace(/_/g, " ")
    .replace(/\b\w/g, (character) => character.toUpperCase());
}

function formatDateTime(value?: string): string {
  if (!value) return "—";

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";

  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

function errorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) return value.message;
  if (value instanceof Error) return value.message;
  return "Shipment update failed.";
}

function statusTone(status: string): string {
  switch (status) {
    case "completed":
    case "delivered":
    case "paid":
    case "cod_collected":
    case "handed_off":
      return orderStyles.tonePositive;

    case "confirmed":
    case "processing":
    case "shipped":
    case "picking":
    case "packed":
    case "ready_for_handoff":
      return orderStyles.toneInfo;

    case "awaiting_procurement":
    case "pending_payment":
    case "cod_pending":
    case "allocated":
    case "waiting_inbound":
    case "received":
    case "pending":
    case "awaiting_confirmation":
      return orderStyles.toneAttention;

    case "cancelled":
    case "failed":
      return orderStyles.toneDanger;

    default:
      return orderStyles.toneNeutral;
  }
}

function shipmentStage(status?: string): string {
  switch (status) {
    case undefined:
      return "Not prepared";
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

function fulfillmentAction(
  fulfillment: AdminWarehouseFulfillment,
): FulfillmentAction | null {
  switch (fulfillment.status) {
    case "allocated":
    case "received":
      return { label: "Start preparing", path: "start-picking" };
    case "picking":
      return { label: "Mark packed", path: "mark-packed" };
    case "packed":
      return { label: "Mark ready", path: "ready-for-handoff" };
    default:
      return null;
  }
}

async function optionalFetch<T>(path: string): Promise<T | null> {
  try {
    return await adminFetch<T>(path);
  } catch (value: unknown) {
    if (
      value instanceof AdminRequestError &&
      (value.status === 403 || value.status === 404)
    ) {
      return null;
    }
    throw value;
  }
}

export default function AdminShipmentControlDrawer({
  orderId,
  onClose,
  onUpdated,
}: Props) {
  const principal = useAdminSession();
  const permissions = principal.staff.permissions;
  const isSuperAdmin = principal.staff.roles.includes("admin_superuser");
  const hasPermission = useCallback(
    (permission: string) => isSuperAdmin || permissions.includes(permission),
    [isSuperAdmin, permissions],
  );

  const canWarehouseRead = hasPermission("admin.warehouse.read");
  const canWarehouseManage = hasPermission("admin.warehouse.manage");
  const canDeliveryRead = hasPermission("admin.delivery.read");
  const canDeliveryManage = hasPermission("admin.delivery.manage");
  const canConfirmReceipt = hasPermission("admin.delivery.confirm_receipt");

  const [order, setOrder] = useState<AdminOrderDetail | null>(null);
  const [fulfillments, setFulfillments] = useState<AdminWarehouseFulfillment[]>([]);
  const [tracking, setTracking] = useState<AdminDeliveryTrackingDetail | null>(null);
  const [warehouses, setWarehouses] = useState<AdminWarehouse[]>([]);
  const [warehouseId, setWarehouseId] = useState("");

  const [deliveryMode, setDeliveryMode] = useState<
    "courier" | "community_rider" | "self_pickup"
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

  const [trackingStatus, setTrackingStatus] = useState("in_transit");
  const [trackingCity, setTrackingCity] = useState("");
  const [trackingLocation, setTrackingLocation] = useState("");
  const [trackingPublicMessage, setTrackingPublicMessage] = useState("");
  const [trackingInternalMessage, setTrackingInternalMessage] = useState("");

  const [providerMessage, setProviderMessage] = useState("");
  const [receiptNote, setReceiptNote] = useState("");

  const [loading, setLoading] = useState(false);
  const [operationLoading, setOperationLoading] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [lastSyncedAt, setLastSyncedAt] = useState<Date | null>(null);

  const loadControl = useCallback(
    async (options?: { silent?: boolean }) => {
      if (!orderId) return;

      if (!options?.silent) setLoading(true);
      setError("");

      try {
        const [orderResponse, fulfillmentResponse, trackingResponse, warehouseResponse] =
          await Promise.all([
            adminFetch<AdminOrderResponse>(`/orders/${orderId}`),
            canWarehouseRead
              ? optionalFetch<AdminWarehouseFulfillmentsResponse>(
                  `/warehouse/orders/${orderId}/fulfillments`,
                )
              : Promise.resolve(null),
            canDeliveryRead
              ? optionalFetch<AdminDeliveryTrackingResponse>(
                  `/delivery/orders/${orderId}/tracking`,
                )
              : Promise.resolve(null),
            canWarehouseRead
              ? optionalFetch<AdminWarehousesResponse>("/warehouse/locations")
              : Promise.resolve(null),
          ]);

        const activeWarehouses = (warehouseResponse?.data ?? []).filter(
          (warehouse) => warehouse.status === "active",
        );

        setOrder(orderResponse.data);
        setFulfillments(fulfillmentResponse?.data ?? []);
        setTracking(trackingResponse?.data ?? null);
        setWarehouses(activeWarehouses);
        setWarehouseId((current) => {
          if (current && activeWarehouses.some((warehouse) => warehouse.id === current)) {
            return current;
          }
          return (
            activeWarehouses.find((warehouse) => warehouse.is_default)?.id ??
            activeWarehouses[0]?.id ??
            ""
          );
        });
        setLastSyncedAt(new Date());
      } catch (value: unknown) {
        setError(errorMessage(value));
      } finally {
        if (!options?.silent) setLoading(false);
      }
    },
    [canDeliveryRead, canWarehouseRead, orderId],
  );

  useEffect(() => {
    setOrder(null);
    setFulfillments([]);
    setTracking(null);
    setWarehouses([]);
    setWarehouseId("");
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
    setTrackingStatus("in_transit");
    setTrackingCity("");
    setTrackingLocation("");
    setTrackingPublicMessage("");
    setTrackingInternalMessage("");
    setProviderMessage("");
    setReceiptNote("");
    setError("");
    setNotice("");
    setLastSyncedAt(null);

    if (orderId) void loadControl();
  }, [loadControl, orderId]);

  useEffect(() => {
    const shipment = tracking?.shipment;
    if (!shipment || shipment.status !== "pending") return;

    setDeliveryMode(
      shipment.delivery_mode as "courier" | "community_rider" | "self_pickup",
    );
    setCourierName(shipment.courier_name ?? "");
    setProviderCode(shipment.provider_code ?? "");
    setProviderShipmentId(shipment.provider_shipment_id ?? "");
    setCourierReference(shipment.courier_reference ?? "");
    setRiderReference(shipment.rider_reference ?? "");
    setTrackingNumber(shipment.tracking_number ?? "");
    setTrackingUrl(shipment.tracking_url ?? "");
  }, [tracking?.shipment.id]);

  useEffect(() => {
    if (!orderId) return;

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [onClose, orderId]);

  useEffect(() => {
    if (!orderId) return;

    const sync = () => {
      if (document.visibilityState !== "visible" || operationLoading) return;
      void loadControl({ silent: true });
    };

    const intervalId = window.setInterval(sync, LIVE_SYNC_MS);
    document.addEventListener("visibilitychange", sync);

    return () => {
      window.clearInterval(intervalId);
      document.removeEventListener("visibilitychange", sync);
    };
  }, [loadControl, operationLoading, orderId]);

  const activeFulfillments = useMemo(
    () => fulfillments.filter((fulfillment) => fulfillment.status !== "cancelled"),
    [fulfillments],
  );

  const remainingItems = useMemo(() => {
    if (!order) return [];

    return order.items
      .map((item) => {
        const allocated = activeFulfillments
          .filter((fulfillment) => fulfillment.order_item_id === item.id)
          .reduce((total, fulfillment) => total + fulfillment.quantity, 0);

        return { item, remaining: Math.max(0, item.quantity - allocated) };
      })
      .filter((entry) => entry.remaining > 0);
  }, [activeFulfillments, order]);

  const readyFulfillments = activeFulfillments.filter(
    (fulfillment) => fulfillment.status === "ready_for_handoff",
  );
  const allReady =
    activeFulfillments.length > 0 &&
    remainingItems.length === 0 &&
    activeFulfillments.every(
      (fulfillment) => fulfillment.status === "ready_for_handoff",
    );
  const allHandedOff =
    activeFulfillments.length > 0 &&
    activeFulfillments.every((fulfillment) => fulfillment.status === "handed_off");

  const shipmentWarehouseId =
    tracking?.shipment.origin_warehouse_id ??
    activeFulfillments[0]?.warehouse_id ??
    warehouseId;
  const shipmentWarehouse = warehouses.find(
    (warehouse) => warehouse.id === shipmentWarehouseId,
  );
  const selfPickupCapabilityKnown = Boolean(shipmentWarehouse);
  const selfPickupAllowed = shipmentWarehouse?.allows_self_pickup ?? true;
  const shipmentModeInvalid =
    tracking?.shipment.status === "pending" &&
    tracking.shipment.delivery_mode === "self_pickup" &&
    selfPickupCapabilityKnown &&
    !selfPickupAllowed;

  const nextStage = useMemo(() => {
    if (!order) return { title: "Loading", hint: "Loading shipment state." };

    if (!tracking) {
      if (remainingItems.length > 0) {
        return {
          title: "Allocate items for preparation",
          hint: "Choose the warehouse and allocate the remaining order quantities.",
        };
      }

      const nextFulfillment = activeFulfillments.find((fulfillment) =>
        fulfillmentAction(fulfillment),
      );

      if (nextFulfillment) {
        return {
          title: fulfillmentAction(nextFulfillment)?.label ?? "Prepare fulfillment",
          hint: "Advance warehouse preparation until every item is ready for handoff.",
        };
      }

      if (allReady) {
        return {
          title: "Mark shipment prepared",
          hint: "Enter delivery details and create the prepared shipment.",
        };
      }

      return {
        title: "Shipment cannot be prepared yet",
        hint: "Check warehouse preparation and order status.",
      };
    }

    if (tracking.shipment.status === "pending") {
      if (shipmentModeInvalid) {
        return {
          title: "Correct prepared shipment",
          hint: "Self pickup is disabled for this warehouse. Change the delivery mode before handoff.",
        };
      }

      return allHandedOff
        ? {
            title: "Mark shipment shipped",
            hint: "Dispatch updates both the shipment and the order to Shipped.",
          }
        : {
            title: "Record physical handoff",
            hint: "Hand the prepared shipment to the courier, rider or pickup flow.",
          };
    }

    if (tracking.shipment.status === "shipped") {
      return {
        title: "Update delivery progress",
        hint: "Record live tracking events or mark the provider delivery complete.",
      };
    }

    if (tracking.shipment.status === "awaiting_confirmation") {
      return {
        title: "Mark delivered",
        hint: "Confirm receipt to update the order to Delivered.",
      };
    }

    return {
      title: "Delivery complete",
      hint: "This shipment has no further delivery-state update.",
    };
  }, [
    activeFulfillments,
    allHandedOff,
    allReady,
    order,
    remainingItems.length,
    shipmentModeInvalid,
    tracking,
  ]);

  const finishMutation = useCallback(async (message: string) => {
    setNotice(message);
    await loadControl({ silent: true });
    onUpdated();
  }, [loadControl, onUpdated]);

  async function allocateRemaining() {
    if (!order || !warehouseId || remainingItems.length === 0) return;

    setOperationLoading("allocate");
    setError("");
    setNotice("");

    try {
      for (const entry of remainingItems) {
        await adminFetch("/warehouse/fulfillments", {
          method: "POST",
          body: JSON.stringify({
            order_item_id: entry.item.id,
            warehouse_id: warehouseId,
            inbound_shipment_id: "",
            source: "bangladesh_stock",
            quantity: entry.remaining,
          }),
        });
      }

      await finishMutation("Order items allocated. Start preparation next.");
    } catch (value: unknown) {
      setError(errorMessage(value));
      await loadControl({ silent: true });
    } finally {
      setOperationLoading("");
    }
  }

  async function advanceFulfillment(
    fulfillment: AdminWarehouseFulfillment,
    action: FulfillmentAction,
  ) {
    setOperationLoading(`fulfillment:${fulfillment.id}`);
    setError("");
    setNotice("");

    try {
      await adminFetch(
        `/warehouse/fulfillments/${fulfillment.id}/${action.path}`,
        { method: "POST", body: JSON.stringify({}) },
      );
      await finishMutation(`${action.label} completed.`);
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  async function prepareShipment() {
    if (!order) return;

    if (deliveryMode === "courier" && !courierName.trim() && !providerCode.trim()) {
      setError("Enter a courier name or provider code before marking the shipment prepared.");
      return;
    }

    if (deliveryMode === "community_rider" && !riderReference.trim()) {
      setError("Enter the rider reference before marking the shipment prepared.");
      return;
    }

    if (deliveryMode === "self_pickup" && selfPickupCapabilityKnown && !selfPickupAllowed) {
      setError("Self pickup is not enabled for the selected warehouse. Choose courier or community rider.");
      return;
    }

    setOperationLoading("prepare");
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

      await finishMutation("Shipment marked Prepared.");
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  async function savePreparedShipment() {
    if (!tracking || tracking.shipment.status !== "pending") return;

    if (deliveryMode === "courier" && !courierName.trim() && !providerCode.trim()) {
      setError("Enter a courier name or provider code before updating the prepared shipment.");
      return;
    }

    if (deliveryMode === "community_rider" && !riderReference.trim()) {
      setError("Enter the rider reference before updating the prepared shipment.");
      return;
    }

    if (deliveryMode === "self_pickup" && selfPickupCapabilityKnown && !selfPickupAllowed) {
      setError("Self pickup is not enabled for this warehouse. Choose courier or community rider.");
      return;
    }

    setOperationLoading("update-prepared");
    setError("");
    setNotice("");

    try {
      await adminFetch(`/delivery/shipments/${tracking.shipment.id}`, {
        method: "PATCH",
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

      await finishMutation("Prepared shipment details updated.");
    } catch (value: unknown) {
      setError(errorMessage(value));
      await loadControl({ silent: true });
    } finally {
      setOperationLoading("");
    }
  }

  async function recordHandoff() {
    if (!tracking || readyFulfillments.length === 0) return;

    if (shipmentModeInvalid) {
      setError("This prepared shipment is set to self pickup, but the warehouse does not allow self pickup. Change the delivery mode first.");
      return;
    }

    setOperationLoading("handoff");
    setError("");
    setNotice("");

    try {
      await adminFetch("/warehouse/handoffs", {
        method: "POST",
        body: JSON.stringify({
          shipment_id: tracking.shipment.id,
          handoff_type: tracking.shipment.delivery_mode,
          reference: handoffReference.trim(),
          items: readyFulfillments.map((fulfillment) => ({
            fulfillment_id: fulfillment.id,
            quantity: fulfillment.quantity,
          })),
        }),
      });

      setHandoffReference("");
      await finishMutation("Physical handoff recorded.");
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  async function markShipped() {
    if (!tracking) return;

    setOperationLoading("dispatch");
    setError("");
    setNotice("");

    try {
      await adminFetch(`/delivery/shipments/${tracking.shipment.id}/dispatch`, {
        method: "POST",
        body: JSON.stringify({ message: dispatchMessage.trim() }),
      });

      setDispatchMessage("");
      await finishMutation("Shipment marked Shipped. The order is now Shipped too.");
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  async function saveTrackingUpdate() {
    if (!tracking || !trackingStatus.trim()) return;

    setOperationLoading("tracking");
    setError("");
    setNotice("");

    try {
      const normalized = trackingStatus.trim().toLowerCase();
      await adminFetch(
        `/delivery/shipments/${tracking.shipment.id}/tracking-events`,
        {
          method: "POST",
          body: JSON.stringify({
            source: "admin",
            event_code: normalized,
            status: normalized,
            message: trackingInternalMessage.trim(),
            public_message: trackingPublicMessage.trim(),
            city: trackingCity.trim(),
            location_name: trackingLocation.trim(),
            customer_visible: true,
          }),
        },
      );

      setTrackingPublicMessage("");
      setTrackingInternalMessage("");
      await finishMutation(`Tracking updated: ${titleCase(normalized)}.`);
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  async function markProviderDelivered() {
    if (!tracking) return;

    setOperationLoading("provider-delivered");
    setError("");
    setNotice("");

    try {
      await adminFetch(
        `/delivery/shipments/${tracking.shipment.id}/provider-delivered`,
        {
          method: "POST",
          body: JSON.stringify({
            provider_status: "delivered",
            message: providerMessage.trim(),
          }),
        },
      );

      setProviderMessage("");
      await finishMutation("Provider delivery recorded. Receipt confirmation is next.");
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  async function markDelivered() {
    if (!order) return;

    setOperationLoading("delivered");
    setError("");
    setNotice("");

    try {
      await adminFetch(`/delivery/orders/${order.id}/confirm-receipt`, {
        method: "POST",
        body: JSON.stringify({ note: receiptNote.trim() }),
      });

      setReceiptNote("");
      await finishMutation("Receipt confirmed. Shipment and order are Delivered.");
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setOperationLoading("");
    }
  }

  if (!orderId) return null;

  return (
    <div className={orderStyles.drawerLayer} role="presentation">
      <button
        type="button"
        className={orderStyles.drawerBackdrop}
        onClick={onClose}
        aria-label="Close shipment control"
      />

      <aside
        className={orderStyles.drawer}
        role="dialog"
        aria-modal="true"
        aria-labelledby="shipment-control-title"
      >
        <header className={orderStyles.drawerHeader}>
          <div>
            <span className={orderStyles.drawerEyebrow}>Shipment control</span>
            <h2 id="shipment-control-title">{order?.order_number ?? "Loading shipment…"}</h2>
          </div>
          <button
            type="button"
            className={orderStyles.drawerClose}
            onClick={onClose}
            aria-label="Close shipment control"
          >
            ×
          </button>
        </header>

        <div className={orderStyles.drawerScroll}>
          {loading && !order ? (
            <div className={orderStyles.detailSkeleton} aria-hidden="true">
              <span />
              <span />
              <span />
              <span />
            </div>
          ) : error && !order ? (
            <div className={orderStyles.drawerError} role="alert">
              <strong>Shipment could not be opened</strong>
              <p>{error}</p>
              <button type="button" onClick={() => void loadControl()}>
                Try again
              </button>
            </div>
          ) : order ? (
            <>
              <section className={orderStyles.orderHero}>
                <div className={orderStyles.heroStatusRow}>
                  <span className={`${orderStyles.statusPill} ${statusTone(order.status)}`}>
                    Order · {titleCase(order.status)}
                  </span>
                  <span className={`${orderStyles.statusPill} ${statusTone(order.payment_status)}`}>
                    {titleCase(order.payment_status)}
                  </span>
                  <span className={`${orderStyles.statusPill} ${statusTone(tracking?.shipment.status ?? "pending")}`}>
                    Shipment · {shipmentStage(tracking?.shipment.status)}
                  </span>
                </div>

                <div className={orderStyles.heroAmount}>
                  <span>Total</span>
                  <strong>{formatMoney(order.total_amount, order.currency)}</strong>
                </div>

                <div className={orderStyles.heroFacts}>
                  <span>
                    <small>Customer</small>
                    <strong>{order.customer_name || "Customer"}</strong>
                  </span>
                  <span>
                    <small>Destination</small>
                    <strong>{order.shipping_area || order.shipping_city || "—"}</strong>
                  </span>
                  <span>
                    <small>Payment</small>
                    <strong>{titleCase(order.payment_method)}</strong>
                  </span>
                </div>
              </section>

              {notice ? (
                <div className={orderStyles.noticeBanner} role="status">
                  <span aria-hidden="true">✓</span>
                  {notice}
                </div>
              ) : null}

              {error ? (
                <div className={orderStyles.inlineError} role="alert">{error}</div>
              ) : null}

              <section className={`${orderStyles.detailSection} ${orderStyles.deliveryControlSection}`}>
                <div className={orderStyles.sectionHeading}>
                  <div>
                    <span>Manual shipment update</span>
                    <h3>Update this shipment now</h3>
                  </div>
                  <span className={`${orderStyles.statusPill} ${statusTone(tracking?.shipment.status ?? "pending")}`}>
                    {shipmentStage(tracking?.shipment.status)}
                  </span>
                </div>

                <div className={orderStyles.shipmentStagePanel}>
                  <div className={orderStyles.shipmentStageCurrent}>
                    <span>Current stage</span>
                    <strong>{shipmentStage(tracking?.shipment.status)}</strong>
                    <small>Order: {titleCase(order.status)}</small>
                  </div>
                  <div className={orderStyles.shipmentStageNext}>
                    <span>Next update</span>
                    <strong>{nextStage.title}</strong>
                    <small>{nextStage.hint}</small>
                  </div>
                  <div className={orderStyles.shipmentSyncState}>
                    <span className={orderStyles.liveDot} aria-hidden="true" />
                    <span>
                      Live
                      {lastSyncedAt
                        ? ` · ${formatDateTime(lastSyncedAt.toISOString())}`
                        : ""}
                    </span>
                  </div>
                </div>

                {!canDeliveryManage && !canWarehouseManage && !canConfirmReceipt ? (
                  <div className={orderStyles.quietState}>
                    <strong>Read-only access</strong>
                    <p>
                      This account can view shipments but does not have shipment or warehouse management permissions.
                    </p>
                  </div>
                ) : null}

                {!tracking && remainingItems.length > 0 ? (
                  <div className={orderStyles.deliveryActionCard}>
                    <div>
                      <span className={orderStyles.deliveryStep}>1 · Allocate</span>
                      <strong>Allocate order items to a warehouse</strong>
                      <p>
                        This is the missing first action for a Not prepared order. After allocation you can move it through preparation and then mark the shipment Prepared.
                      </p>
                    </div>

                    <label className={orderStyles.fieldLabel}>
                      Warehouse
                      <select
                        value={warehouseId}
                        onChange={(event) => setWarehouseId(event.target.value)}
                        disabled={!canWarehouseManage || Boolean(operationLoading)}
                      >
                        <option value="">Select warehouse</option>
                        {warehouses.map((warehouse) => (
                          <option key={warehouse.id} value={warehouse.id}>
                            {warehouse.name} ({warehouse.code}){warehouse.is_default ? " · Default" : ""}
                          </option>
                        ))}
                      </select>
                    </label>

                    <div className={orderStyles.fulfillmentList}>
                      {remainingItems.map(({ item, remaining }) => (
                        <div key={item.id} className={orderStyles.fulfillmentCard}>
                          <div className={orderStyles.fulfillmentTop}>
                            <div>
                              <strong>{item.product_name}</strong>
                              <span>{item.sku} · Qty to allocate {remaining}</span>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>

                    <button
                      type="button"
                      className={orderStyles.primaryActionButton}
                      onClick={() => void allocateRemaining()}
                      disabled={
                        !canWarehouseManage ||
                        !warehouseId ||
                        Boolean(operationLoading)
                      }
                    >
                      {operationLoading === "allocate"
                        ? "Allocating…"
                        : "Allocate for preparation"}
                    </button>
                  </div>
                ) : null}

                {!tracking && activeFulfillments.length > 0 ? (
                  <div className={orderStyles.deliveryControlGroup}>
                    <div className={orderStyles.deliveryControlGroupHeading}>
                      <div>
                        <span>Warehouse preparation</span>
                        <strong>Move each item to Ready for handoff</strong>
                      </div>
                      <span className={orderStyles.sectionCount}>{activeFulfillments.length}</span>
                    </div>

                    <div className={orderStyles.fulfillmentList}>
                      {activeFulfillments.map((fulfillment) => {
                        const action = fulfillmentAction(fulfillment);
                        return (
                          <article key={fulfillment.id} className={orderStyles.fulfillmentCard}>
                            <div className={orderStyles.fulfillmentTop}>
                              <div>
                                <strong>{fulfillment.product_name || fulfillment.order_item_sku || "Order item"}</strong>
                                <span>Qty {fulfillment.quantity} · {fulfillment.warehouse_name || fulfillment.warehouse_code || "Warehouse"}</span>
                              </div>
                              <span className={`${orderStyles.statusPill} ${statusTone(fulfillment.status)}`}>
                                {titleCase(fulfillment.status)}
                              </span>
                            </div>

                            {action ? (
                              <button
                                type="button"
                                className={orderStyles.primaryActionButton}
                                onClick={() => void advanceFulfillment(fulfillment, action)}
                                disabled={!canWarehouseManage || Boolean(operationLoading)}
                              >
                                {operationLoading === `fulfillment:${fulfillment.id}`
                                  ? "Updating…"
                                  : action.label}
                              </button>
                            ) : null}
                          </article>
                        );
                      })}
                    </div>
                  </div>
                ) : null}

                {!tracking && allReady ? (
                  <div className={orderStyles.deliveryActionCard}>
                    <div>
                      <span className={orderStyles.deliveryStep}>2 · Prepared</span>
                      <strong>Mark shipment prepared</strong>
                      <p>This creates the real backend shipment in Prepared state.</p>
                    </div>

                    <div className={orderStyles.deliveryFormGrid}>
                      <label className={orderStyles.fieldLabel}>
                        Delivery mode
                        <select
                          value={deliveryMode}
                          onChange={(event) =>
                            setDeliveryMode(
                              event.target.value as "courier" | "community_rider" | "self_pickup",
                            )
                          }
                        >
                          <option value="courier">Courier</option>
                          <option value="community_rider">Community rider</option>
                          <option
                            value="self_pickup"
                            disabled={selfPickupCapabilityKnown && !selfPickupAllowed}
                          >
                            Self pickup{
                              selfPickupCapabilityKnown && !selfPickupAllowed
                                ? " · unavailable at this warehouse"
                                : ""
                            }
                          </option>
                        </select>
                      </label>

                      {deliveryMode === "courier" ? (
                        <>
                          <label className={orderStyles.fieldLabel}>
                            Courier name
                            <input value={courierName} onChange={(event) => setCourierName(event.target.value)} placeholder="Pathao / Steadfast / etc." />
                          </label>
                          <label className={orderStyles.fieldLabel}>
                            Provider code
                            <input value={providerCode} onChange={(event) => setProviderCode(event.target.value)} placeholder="Optional if courier name is filled" />
                          </label>
                          <label className={orderStyles.fieldLabel}>
                            Tracking number
                            <input value={trackingNumber} onChange={(event) => setTrackingNumber(event.target.value)} placeholder="Optional" />
                          </label>
                          <label className={orderStyles.fieldLabel}>
                            Courier reference
                            <input value={courierReference} onChange={(event) => setCourierReference(event.target.value)} placeholder="Optional" />
                          </label>
                          <label className={orderStyles.fieldLabel}>
                            Provider shipment ID
                            <input value={providerShipmentId} onChange={(event) => setProviderShipmentId(event.target.value)} placeholder="Optional" />
                          </label>
                          <label className={orderStyles.fieldLabel}>
                            Tracking URL
                            <input value={trackingUrl} onChange={(event) => setTrackingUrl(event.target.value)} placeholder="Optional" />
                          </label>
                        </>
                      ) : null}

                      {deliveryMode === "community_rider" ? (
                        <label className={orderStyles.fieldLabel}>
                          Rider reference
                          <input value={riderReference} onChange={(event) => setRiderReference(event.target.value)} placeholder="Required rider reference" />
                        </label>
                      ) : null}
                    </div>

                    <button
                      type="button"
                      className={orderStyles.positiveActionButton}
                      onClick={() => void prepareShipment()}
                      disabled={!canDeliveryManage || Boolean(operationLoading)}
                    >
                      {operationLoading === "prepare" ? "Preparing…" : "Mark prepared"}
                    </button>
                  </div>
                ) : null}

                {tracking?.shipment.status === "pending" && !allHandedOff ? (
                  <div className={orderStyles.deliveryActionCard}>
                    <div>
                      <span className={orderStyles.deliveryStep}>Prepared shipment details</span>
                      <strong>Delivery mode and carrier</strong>
                      <p>
                        You can correct these details until warehouse handoff starts.
                      </p>
                    </div>

                    {shipmentModeInvalid ? (
                      <div className={orderStyles.inlineWarning}>
                        This shipment is currently set to Self pickup, but {
                          shipmentWarehouse?.name ?? "the origin warehouse"
                        } does not allow self pickup. Choose Courier or Community rider, then save before recording handoff.
                      </div>
                    ) : null}

                    <div className={orderStyles.deliveryFormGrid}>
                      <label className={orderStyles.fieldLabel}>
                        Delivery mode
                        <select
                          value={deliveryMode}
                          onChange={(event) =>
                            setDeliveryMode(
                              event.target.value as
                                | "courier"
                                | "community_rider"
                                | "self_pickup",
                            )
                          }
                        >
                          <option value="courier">Courier</option>
                          <option value="community_rider">Community rider</option>
                          <option
                            value="self_pickup"
                            disabled={selfPickupCapabilityKnown && !selfPickupAllowed}
                          >
                            Self pickup{
                              selfPickupCapabilityKnown && !selfPickupAllowed
                                ? " · unavailable at this warehouse"
                                : ""
                            }
                          </option>
                        </select>
                      </label>

                      {deliveryMode === "courier" ? (
                        <>
                          <label className={orderStyles.fieldLabel}>
                            Courier name
                            <input
                              value={courierName}
                              onChange={(event) => setCourierName(event.target.value)}
                              placeholder="Pathao / Steadfast / etc."
                            />
                          </label>
                          <label className={orderStyles.fieldLabel}>
                            Provider code
                            <input
                              value={providerCode}
                              onChange={(event) => setProviderCode(event.target.value)}
                              placeholder="Optional if courier name is filled"
                            />
                          </label>
                          <label className={orderStyles.fieldLabel}>
                            Tracking number
                            <input
                              value={trackingNumber}
                              onChange={(event) => setTrackingNumber(event.target.value)}
                              placeholder="Optional"
                            />
                          </label>
                          <label className={orderStyles.fieldLabel}>
                            Courier reference
                            <input
                              value={courierReference}
                              onChange={(event) => setCourierReference(event.target.value)}
                              placeholder="Optional"
                            />
                          </label>
                          <label className={orderStyles.fieldLabel}>
                            Provider shipment ID
                            <input
                              value={providerShipmentId}
                              onChange={(event) => setProviderShipmentId(event.target.value)}
                              placeholder="Optional"
                            />
                          </label>
                          <label className={orderStyles.fieldLabel}>
                            Tracking URL
                            <input
                              value={trackingUrl}
                              onChange={(event) => setTrackingUrl(event.target.value)}
                              placeholder="Optional"
                            />
                          </label>
                        </>
                      ) : null}

                      {deliveryMode === "community_rider" ? (
                        <label className={orderStyles.fieldLabel}>
                          Rider reference
                          <input
                            value={riderReference}
                            onChange={(event) => setRiderReference(event.target.value)}
                            placeholder="Required rider reference"
                          />
                        </label>
                      ) : null}
                    </div>

                    <button
                      type="button"
                      className={orderStyles.positiveActionButton}
                      onClick={() => void savePreparedShipment()}
                      disabled={!canDeliveryManage || Boolean(operationLoading)}
                    >
                      {operationLoading === "update-prepared"
                        ? "Saving…"
                        : "Update prepared shipment"}
                    </button>
                  </div>
                ) : null}

                {tracking?.shipment.status === "pending" && !allHandedOff ? (
                  <div className={orderStyles.deliveryActionCard}>
                    <div>
                      <span className={orderStyles.deliveryStep}>3 · Handoff</span>
                      <strong>Record physical handoff</strong>
                      <p>Required before the shipment can be marked Shipped.</p>
                    </div>
                    <label className={orderStyles.fieldLabel}>
                      Handoff reference
                      <input
                        value={handoffReference}
                        onChange={(event) => setHandoffReference(event.target.value)}
                        placeholder="Optional courier/rider reference"
                      />
                    </label>
                    <button
                      type="button"
                      className={orderStyles.primaryActionButton}
                      onClick={() => void recordHandoff()}
                      disabled={
                        !canWarehouseManage ||
                        readyFulfillments.length === 0 ||
                        shipmentModeInvalid ||
                        Boolean(operationLoading)
                      }
                    >
                      {operationLoading === "handoff" ? "Recording…" : "Record handoff"}
                    </button>
                  </div>
                ) : null}

                {tracking?.shipment.status === "pending" && allHandedOff ? (
                  <div className={orderStyles.deliveryActionCard}>
                    <div>
                      <span className={orderStyles.deliveryStep}>4 · Shipped</span>
                      <strong>Mark shipment shipped</strong>
                      <p>This updates the shipment and the linked order to Shipped immediately.</p>
                    </div>
                    <label className={orderStyles.fieldLabel}>
                      Dispatch note
                      <input
                        value={dispatchMessage}
                        onChange={(event) => setDispatchMessage(event.target.value)}
                        placeholder="Optional"
                      />
                    </label>
                    <button
                      type="button"
                      className={orderStyles.positiveActionButton}
                      onClick={() => void markShipped()}
                      disabled={!canDeliveryManage || Boolean(operationLoading)}
                    >
                      {operationLoading === "dispatch" ? "Updating…" : "Mark shipped"}
                    </button>
                  </div>
                ) : null}

                {tracking?.shipment.status === "shipped" ? (
                  <>
                    <div className={orderStyles.deliveryActionCard}>
                      <div>
                        <span className={orderStyles.deliveryStep}>5 · Live delivery update</span>
                        <strong>Update delivery status</strong>
                        <p>Record the real-world delivery progress. It will also appear in the Order Lifecycle.</p>
                      </div>

                      <div className={orderStyles.deliveryFormGrid}>
                        <label className={orderStyles.fieldLabel}>
                          Delivery status
                          <select value={trackingStatus} onChange={(event) => setTrackingStatus(event.target.value)}>
                            <option value="in_transit">In transit</option>
                            <option value="arrived_at_hub">Arrived at hub</option>
                            <option value="out_for_delivery">Out for delivery</option>
                            <option value="delivery_attempted">Delivery attempted</option>
                            <option value="delayed">Delayed</option>
                          </select>
                        </label>
                        <label className={orderStyles.fieldLabel}>
                          City
                          <input value={trackingCity} onChange={(event) => setTrackingCity(event.target.value)} placeholder="Optional" />
                        </label>
                        <label className={orderStyles.fieldLabel}>
                          Location
                          <input value={trackingLocation} onChange={(event) => setTrackingLocation(event.target.value)} placeholder="Optional" />
                        </label>
                        <label className={`${orderStyles.fieldLabel} ${orderStyles.fieldWide}`}>
                          Customer message
                          <textarea value={trackingPublicMessage} onChange={(event) => setTrackingPublicMessage(event.target.value)} rows={2} placeholder="Optional customer-visible update" />
                        </label>
                        <label className={`${orderStyles.fieldLabel} ${orderStyles.fieldWide}`}>
                          Internal note
                          <textarea value={trackingInternalMessage} onChange={(event) => setTrackingInternalMessage(event.target.value)} rows={2} placeholder="Optional internal note" />
                        </label>
                      </div>

                      <button
                        type="button"
                        className={orderStyles.primaryActionButton}
                        onClick={() => void saveTrackingUpdate()}
                        disabled={!canDeliveryManage || Boolean(operationLoading)}
                      >
                        {operationLoading === "tracking" ? "Saving…" : "Save delivery update"}
                      </button>
                    </div>

                    {tracking.shipment.delivery_mode !== "self_pickup" ? (
                      <div className={orderStyles.deliveryActionCard}>
                        <div>
                          <span className={orderStyles.deliveryStep}>6 · Provider delivered</span>
                          <strong>Courier reports delivered</strong>
                          <p>Moves the shipment to Awaiting confirmation.</p>
                        </div>
                        <label className={orderStyles.fieldLabel}>
                          Provider message
                          <input value={providerMessage} onChange={(event) => setProviderMessage(event.target.value)} placeholder="Optional" />
                        </label>
                        <button
                          type="button"
                          className={orderStyles.primaryActionButton}
                          onClick={() => void markProviderDelivered()}
                          disabled={!canDeliveryManage || Boolean(operationLoading)}
                        >
                          {operationLoading === "provider-delivered" ? "Updating…" : "Mark provider delivered"}
                        </button>
                      </div>
                    ) : null}
                  </>
                ) : null}

                {tracking &&
                order.status === "shipped" &&
                (tracking.shipment.status === "shipped" ||
                  tracking.shipment.status === "awaiting_confirmation") ? (
                  <div className={`${orderStyles.deliveryActionCard} ${orderStyles.deliveryConfirmCard}`}>
                    <div>
                      <span className={orderStyles.deliveryStep}>7 · Delivered</span>
                      <strong>Confirm receipt and mark delivered</strong>
                      <p>This is the authoritative Shipped → Delivered update for the linked order.</p>
                    </div>
                    <label className={orderStyles.fieldLabel}>
                      Confirmation note
                      <textarea value={receiptNote} onChange={(event) => setReceiptNote(event.target.value)} rows={2} placeholder="Optional" />
                    </label>
                    <button
                      type="button"
                      className={orderStyles.positiveActionButton}
                      onClick={() => void markDelivered()}
                      disabled={!canConfirmReceipt || Boolean(operationLoading)}
                    >
                      {operationLoading === "delivered" ? "Updating…" : "Mark delivered"}
                    </button>
                  </div>
                ) : null}

                {tracking?.events?.length ? (
                  <div className={orderStyles.deliveryActivity}>
                    <div className={orderStyles.deliveryControlGroupHeading}>
                      <div>
                        <span>Recent delivery activity</span>
                        <strong>Live shipment history</strong>
                      </div>
                    </div>
                    <div className={orderStyles.trackingTimeline}>
                      {[...tracking.events]
                        .sort(
                          (left, right) =>
                            new Date(right.occurred_at).getTime() -
                            new Date(left.occurred_at).getTime(),
                        )
                        .slice(0, 8)
                        .map((event) => (
                          <div key={event.id} className={orderStyles.trackingEvent}>
                            <span className={orderStyles.liveDot} aria-hidden="true" />
                            <div>
                              <strong>{titleCase(event.event_code)}</strong>
                              <span>{event.message || titleCase(event.status || "updated")}</span>
                              <small>{formatDateTime(event.occurred_at)}</small>
                            </div>
                          </div>
                        ))}
                    </div>
                  </div>
                ) : null}
              </section>
            </>
          ) : null}
        </div>
      </aside>
    </div>
  );
}
