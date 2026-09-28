"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import { AdminRequestError, adminFetch } from "@/lib/admin/api";
import type {
  AdminProductRequest,
  AdminProductRequestListResponse,
  AdminProductRequestMessage,
  AdminProductRequestMessagesResponse,
  AdminProductRequestResponse,
  AdminProductRequestStatus,
  AdminSourcingConfirmation,
  AdminSourcingConfirmationResponse,
  AdminSourcingOffer,
  AdminSourcingOfferMutationPayload,
  AdminSourcingOfferResponse,
  AdminSourcingOffersResponse,
} from "@/lib/admin/product-request-types";

import AdminProductRequestNegotiation from "./AdminProductRequestNegotiation";
import styles from "../css/AdminProductRequests.module.css";

type Props = {
  portal: string;
};

type RequestDetailState = {
  request: AdminProductRequest;
  messages: AdminProductRequestMessage[];
  offers: AdminSourcingOffer[];
  confirmation: AdminSourcingConfirmation | null;
};

const PAGE_SIZE = 40;

const statusOptions: Array<{
  value: "" | AdminProductRequestStatus;
  label: string;
}> = [
  { value: "", label: "All requests" },
  { value: "pending_review", label: "Under review" },
  { value: "on_hold", label: "On hold" },
  { value: "accepted", label: "Accepted for sourcing" },
  { value: "negotiating", label: "Negotiating" },
  { value: "agreed", label: "Agreement confirmed" },
  { value: "converted_to_order", label: "Order created" },
  { value: "cancelled", label: "Cancelled" },
];

function statusLabel(status: string): string {
  const match = statusOptions.find((option) => option.value === status);
  if (match) return match.label;
  return status.replace(/_/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function statusTone(status: string): string {
  switch (status) {
    case "accepted":
    case "agreed":
    case "converted_to_order":
      return styles.good;
    case "negotiating":
      return styles.info;
    case "on_hold":
      return styles.warn;
    case "cancelled":
      return styles.danger;
    default:
      return styles.neutral;
  }
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

function requestErrorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) return value.message;
  if (value instanceof Error) return value.message;
  return "Unable to complete the request.";
}

async function optionalConfirmation(requestId: string): Promise<AdminSourcingConfirmation | null> {
  try {
    const response = await adminFetch<AdminSourcingConfirmationResponse>(
      `/product-requests/${requestId}/confirmation`,
    );
    return response.data;
  } catch (value: unknown) {
    if (value instanceof AdminRequestError && (value.status === 403 || value.status === 404)) {
      return null;
    }
    throw value;
  }
}

export default function AdminProductRequestsWorkspace({ portal }: Props) {
  const principal = useAdminSession();
  const isSuperAdmin = principal.staff.roles.includes("admin_superuser");
  const permissions = principal.staff.permissions;

  const hasPermission = useCallback(
    (permission: string) => isSuperAdmin || permissions.includes(permission),
    [isSuperAdmin, permissions],
  );

  const canRead = hasPermission("admin.sourcing.read");
  const canReview = hasPermission("admin.sourcing.review");
  const canManageOffers = hasPermission("admin.sourcing.offer.manage");
  const canFinalize = hasPermission("admin.sourcing.finalize");

  const [requests, setRequests] = useState<AdminProductRequest[]>([]);
  const [queryInput, setQueryInput] = useState("");
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"" | AdminProductRequestStatus>("");
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [loading, setLoading] = useState(true);
  const [listError, setListError] = useState("");
  const [lastSyncAt, setLastSyncAt] = useState<Date | null>(null);

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<RequestDetailState | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState("");
  const [busyAction, setBusyAction] = useState("");

  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const listAbort = useRef<AbortController | null>(null);

  useEffect(() => {
    if (searchTimer.current) clearTimeout(searchTimer.current);
    searchTimer.current = setTimeout(() => {
      setOffset(0);
      setQuery(queryInput.trim());
    }, 280);

    return () => {
      if (searchTimer.current) clearTimeout(searchTimer.current);
    };
  }, [queryInput]);

  const loadRequests = useCallback(
    async (nextOffset = 0, append = false, quiet = false) => {
      if (!canRead) {
        setLoading(false);
        return;
      }

      listAbort.current?.abort();
      const controller = new AbortController();
      listAbort.current = controller;

      if (!quiet) setLoading(true);
      setListError("");

      try {
        const params = new URLSearchParams({
          limit: String(PAGE_SIZE),
          offset: String(nextOffset),
        });
        if (query) params.set("q", query);
        if (status) params.set("status", status);

        const response = await adminFetch<AdminProductRequestListResponse>(
          `/product-requests?${params.toString()}`,
          { signal: controller.signal },
        );

        if (controller.signal.aborted) return;
        setRequests((current) => (append ? [...current, ...response.data] : response.data));
        setHasMore(response.data.length === PAGE_SIZE);
        setOffset(nextOffset);
        setLastSyncAt(new Date());
      } catch (value: unknown) {
        if (value instanceof DOMException && value.name === "AbortError") return;
        setListError(requestErrorMessage(value));
      } finally {
        if (!quiet) setLoading(false);
      }
    },
    [canRead, query, status],
  );

  const loadDetail = useCallback(
    async (requestId: string, quiet = false) => {
      if (!canRead) return;
      if (!quiet) setDetailLoading(true);
      setDetailError("");

      try {
        const [requestResponse, messageResponse, offerResponse, confirmation] = await Promise.all([
          adminFetch<AdminProductRequestResponse>(`/product-requests/${requestId}`),
          adminFetch<AdminProductRequestMessagesResponse>(
            `/product-requests/${requestId}/messages?limit=200&offset=0`,
          ),
          adminFetch<AdminSourcingOffersResponse>(`/product-requests/${requestId}/offers`),
          optionalConfirmation(requestId),
        ]);

        setDetail({
          request: requestResponse.data,
          messages: messageResponse.data,
          offers: offerResponse.data,
          confirmation,
        });
        setRequests((current) =>
          current.map((item) => (item.id === requestId ? requestResponse.data : item)),
        );
      } catch (value: unknown) {
        setDetailError(requestErrorMessage(value));
      } finally {
        if (!quiet) setDetailLoading(false);
      }
    },
    [canRead],
  );

  useEffect(() => {
    void loadRequests(0, false);
    return () => listAbort.current?.abort();
  }, [loadRequests]);

  useEffect(() => {
    if (!selectedId) {
      setDetail(null);
      setDetailError("");
      return;
    }

    setDetail(null);
    setDetailError("");
    void loadDetail(selectedId);
  }, [loadDetail, selectedId]);

  useEffect(() => {
    const timer = window.setInterval(() => {
      if (document.visibilityState !== "visible" || busyAction) return;
      void loadRequests(0, false, true);
      if (selectedId) void loadDetail(selectedId, true);
    }, 10_000);

    return () => window.clearInterval(timer);
  }, [busyAction, loadDetail, loadRequests, selectedId]);

  const visibleStats = useMemo(() => {
    const needsReview = requests.filter((item) => item.status === "pending_review").length;
    const active = requests.filter((item) => ["accepted", "negotiating"].includes(item.status)).length;
    const agreed = requests.filter((item) => item.status === "agreed").length;
    return { needsReview, active, agreed };
  }, [requests]);

  const afterMutation = useCallback(async () => {
    if (!selectedId) return;
    await Promise.all([loadDetail(selectedId, true), loadRequests(0, false, true)]);
  }, [loadDetail, loadRequests, selectedId]);

  async function updateStatus(
    targetStatus: "accepted" | "on_hold" | "cancelled",
    reason: string,
  ) {
    if (!selectedId) return;
    if ((targetStatus === "on_hold" || targetStatus === "cancelled") && !reason.trim()) {
      setDetailError("A reason is required for hold or cancellation.");
      return;
    }

    setBusyAction(`status:${targetStatus}`);
    setDetailError("");
    try {
      await adminFetch<AdminProductRequestResponse>(`/product-requests/${selectedId}/status`, {
        method: "PATCH",
        body: JSON.stringify({ status: targetStatus, reason: reason.trim() }),
      });
      await afterMutation();
    } catch (value: unknown) {
      setDetailError(requestErrorMessage(value));
      throw value;
    } finally {
      setBusyAction("");
    }
  }

  async function sendMessage(
    body: string,
    visibility: "customer" | "internal",
    attachments: unknown[],
  ) {
    if (!selectedId || !body.trim()) return;

    setBusyAction("message");
    setDetailError("");
    try {
      await adminFetch(`/product-requests/${selectedId}/messages`, {
        method: "POST",
        body: JSON.stringify({ message: body.trim(), visibility, attachments }),
      });
      await afterMutation();
    } catch (value: unknown) {
      setDetailError(requestErrorMessage(value));
      throw value;
    } finally {
      setBusyAction("");
    }
  }

  async function saveOffer(
    offerId: string | null,
    payload: AdminSourcingOfferMutationPayload,
    sendAfterSave: boolean,
  ) {
    if (!selectedId) return;

    setBusyAction("offer-save");
    setDetailError("");
    try {
      const saved = offerId
        ? await adminFetch<AdminSourcingOfferResponse>(
            `/product-requests/${selectedId}/offers/${offerId}`,
            { method: "PATCH", body: JSON.stringify(payload) },
          )
        : await adminFetch<AdminSourcingOfferResponse>(`/product-requests/${selectedId}/offers`, {
            method: "POST",
            body: JSON.stringify(payload),
          });

      if (sendAfterSave) {
        try {
          await adminFetch<AdminSourcingOfferResponse>(
            `/product-requests/${selectedId}/offers/${saved.data.id}/send`,
            { method: "POST" },
          );
        } catch (value: unknown) {
          const message = `Offer saved as a draft, but it could not be sent: ${requestErrorMessage(value)}`;
          await afterMutation();
          setDetailError(message);
          return;
        }
      }

      await afterMutation();
    } catch (value: unknown) {
      setDetailError(requestErrorMessage(value));
      throw value;
    } finally {
      setBusyAction("");
    }
  }

  async function sendOffer(offerId: string) {
    if (!selectedId) return;
    setBusyAction(`offer-send:${offerId}`);
    setDetailError("");
    try {
      await adminFetch<AdminSourcingOfferResponse>(
        `/product-requests/${selectedId}/offers/${offerId}/send`,
        { method: "POST" },
      );
      await afterMutation();
    } catch (value: unknown) {
      setDetailError(requestErrorMessage(value));
      throw value;
    } finally {
      setBusyAction("");
    }
  }

  async function finalizeOffer(offerId: string) {
    if (!selectedId) return;
    setBusyAction(`offer-finalize:${offerId}`);
    setDetailError("");
    try {
      await adminFetch<AdminSourcingConfirmationResponse>(
        `/product-requests/${selectedId}/offers/${offerId}/finalize`,
        { method: "POST" },
      );
      await afterMutation();
    } catch (value: unknown) {
      setDetailError(requestErrorMessage(value));
      throw value;
    } finally {
      setBusyAction("");
    }
  }

  if (!canRead) {
    return (
      <main className={styles.page}>
        <div className={styles.permissionCard}>
          <span>Product requests</span>
          <h1>Access restricted</h1>
          <p>Your current staff role does not include sourcing read access.</p>
        </div>
      </main>
    );
  }

  return (
    <main className={styles.page}>
      <header className={styles.hero}>
        <div>
          <p className={styles.eyebrow}>Sourcing operations</p>
          <h1>Product requests</h1>
          <p className={styles.heroCopy}>
            Review incoming sourcing work, negotiate with customers and turn accepted terms into locked sourcing agreements.
          </p>
        </div>
        <div className={styles.liveState}>
          <span className={styles.liveDot} />
          <span>{lastSyncAt ? `Live · ${formatDateTime(lastSyncAt.toISOString())}` : "Connecting"}</span>
        </div>
      </header>

      <section className={styles.summaryGrid} aria-label="Visible sourcing workload">
        <article className={styles.summaryCard}>
          <span>Needs review</span>
          <strong>{visibleStats.needsReview}</strong>
          <small>Visible in the current result set</small>
        </article>
        <article className={styles.summaryCard}>
          <span>Active sourcing</span>
          <strong>{visibleStats.active}</strong>
          <small>Accepted or negotiating</small>
        </article>
        <article className={styles.summaryCard}>
          <span>Agreement confirmed</span>
          <strong>{visibleStats.agreed}</strong>
          <small>Ready for customer sourcing checkout</small>
        </article>
      </section>

      <section className={styles.queueCard}>
        <div className={styles.queueHeader}>
          <div>
            <p className={styles.sectionEyebrow}>Request queue</p>
            <h2>{requests.length} visible requests</h2>
          </div>
          <button type="button" className={styles.secondaryButton} onClick={() => void loadRequests(0, false)} disabled={loading}>Refresh</button>
        </div>

        <div className={styles.filters}>
          <label className={styles.searchField}>
            <span className={styles.srOnly}>Search product requests</span>
            <input value={queryInput} onChange={(event) => setQueryInput(event.target.value)} placeholder="Request, product, customer, phone, email..." />
          </label>
          <label className={styles.selectField}>
            <span className={styles.srOnly}>Request status</span>
            <select value={status} onChange={(event) => { setOffset(0); setStatus(event.target.value as "" | AdminProductRequestStatus); }}>
              {statusOptions.map((option) => <option key={option.value || "all"} value={option.value}>{option.label}</option>)}
            </select>
          </label>
        </div>

        {listError ? <div className={styles.errorBanner}>{listError}</div> : null}

        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr><th>Request</th><th>Customer</th><th>Qty</th><th>Status</th><th>CRM</th><th>Updated</th><th aria-label="Action" /></tr>
            </thead>
            <tbody>
              {requests.map((item) => (
                <tr key={item.id} onClick={() => setSelectedId(item.id)}>
                  <td><strong>{item.request_number}</strong><span>{item.requested_product_name}</span></td>
                  <td><strong>{item.customer.full_name}</strong><span>{item.customer.phone}</span></td>
                  <td>{item.requested_quantity}</td>
                  <td><span className={`${styles.statusPill} ${statusTone(item.status)}`}>{statusLabel(item.status)}</span></td>
                  <td><strong>{statusLabel(item.crm_status)}</strong><span>{statusLabel(item.crm_priority)}</span></td>
                  <td><strong>{formatDateTime(item.updated_at)}</strong><span>{item.assignment?.queue_name ?? "Sourcing"}</span></td>
                  <td>
                    <button type="button" className={styles.rowButton} onClick={(event) => { event.stopPropagation(); setSelectedId(item.id); }}>Open</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          <div className={styles.mobileList}>
            {requests.map((item) => (
              <button type="button" key={item.id} className={styles.mobileCard} onClick={() => setSelectedId(item.id)}>
                <span className={styles.mobileTopline}><strong>{item.request_number}</strong><span className={`${styles.statusPill} ${statusTone(item.status)}`}>{statusLabel(item.status)}</span></span>
                <b>{item.requested_product_name}</b>
                <span>{item.customer.full_name} · {item.customer.phone}</span>
                <span>Qty {item.requested_quantity} · {formatDateTime(item.updated_at)}</span>
              </button>
            ))}
          </div>
        </div>

        {!loading && requests.length === 0 ? <div className={styles.emptyState}><strong>No matching product requests</strong><span>Try another search or status filter.</span></div> : null}
        {loading ? <div className={styles.loadingState}>Loading sourcing requests…</div> : null}

        {hasMore && !loading ? (
          <div className={styles.loadMoreWrap}>
            <button type="button" className={styles.secondaryButton} onClick={() => void loadRequests(offset + PAGE_SIZE, true)}>Load more</button>
          </div>
        ) : null}
      </section>

      {selectedId ? (
        <AdminProductRequestNegotiation
          portal={portal}
          detail={detail}
          loading={detailLoading}
          error={detailError}
          busyAction={busyAction}
          staffName={principal.staff.full_name}
          canReview={canReview}
          canManageOffers={canManageOffers}
          canFinalize={canFinalize}
          onClose={() => setSelectedId(null)}
          onRefresh={() => loadDetail(selectedId)}
          onUpdateStatus={updateStatus}
          onSendMessage={sendMessage}
          onSaveOffer={saveOffer}
          onSendOffer={sendOffer}
          onFinalizeOffer={finalizeOffer}
        />
      ) : null}
    </main>
  );
}
