"use client";

import {
  type FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import { AdminRequestError, adminFetch } from "@/lib/admin/api";
import { formatMoney } from "@/lib/money/format";
import type {
  AdminPaginationMeta,
  AdminRefundSummary,
  AdminReturnDetail,
  AdminReturnDetailResponse,
  AdminReturnInspectionStatus,
  AdminReturnItem,
  AdminReturnListItem,
  AdminReturnListResponse,
  AdminReturnStatus,
} from "@/lib/admin/returns-types";

import styles from "../css/AdminReturns.module.css";

type Props = {
  portal: string;
};

type StatusFilter = "" | AdminReturnStatus;

type InspectionDraft = {
  inspection_status: AdminReturnInspectionStatus;
  inspection_note: string;
  restock_quantity: string;
};

const PAGE_SIZE = 24;
const LIVE_REFRESH_MS = 8_000;

const statusOptions: Array<{
  value: StatusFilter;
  label: string;
}> = [
  { value: "", label: "All returns" },
  { value: "requested", label: "Requested" },
  { value: "approved", label: "Approved" },
  { value: "received", label: "Received" },
  { value: "inspected", label: "Inspected" },
  { value: "completed", label: "Completed" },
  { value: "rejected", label: "Rejected" },
  { value: "cancelled", label: "Cancelled" },
];

function errorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) {
    return value.message;
  }

  if (value instanceof Error) {
    return value.message;
  }

  return "Unable to complete the request.";
}

function titleCase(value?: string): string {
  if (!value) {
    return "—";
  }

  return value
    .replace(/[_-]+/g, " ")
    .replace(/\b\w/g, (character) =>
      character.toUpperCase(),
    );
}

function formatDateTime(
  value?: string,
): string {
  if (!value) {
    return "—";
  }

  const date =
    new Date(value);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return "—";
  }

  return new Intl.DateTimeFormat(
    "en",
    {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "numeric",
      minute: "2-digit",
    },
  ).format(date);
}

function statusTone(
  status: string,
): string {
  switch (status) {
    case "requested":
      return styles.statusRequested;

    case "approved":
      return styles.statusApproved;

    case "received":
      return styles.statusReceived;

    case "inspected":
      return styles.statusInspected;

    case "completed":
      return styles.statusCompleted;

    case "rejected":
    case "cancelled":
      return styles.statusClosed;

    default:
      return styles.statusNeutral;
  }
}

function refundTone(
  status: string,
): string {
  switch (status) {
    case "requested":
      return styles.refundRequested;

    case "approved":
    case "processing":
      return styles.refundProcessing;

    case "succeeded":
      return styles.refundSucceeded;

    case "failed":
    case "cancelled":
      return styles.refundFailed;

    default:
      return styles.statusNeutral;
  }
}

function nextReturnStep(
  status: string,
): string {
  switch (status) {
    case "requested":
      return "Review the request and approve or reject it.";

    case "approved":
      return "Record the physical quantities received from the customer.";

    case "received":
      return "Inspect each received item and decide what can be restocked.";

    case "inspected":
      return "Inspection is complete. Create or continue the refund when applicable.";

    case "completed":
      return "Return processing is complete.";

    case "rejected":
      return "This return request was rejected.";

    case "cancelled":
      return "This return request was cancelled.";

    default:
      return "Review the current return state before taking action.";
  }
}

function integerValue(
  value: string,
): number | null {
  if (
    !/^\d+$/.test(
      value.trim(),
    )
  ) {
    return null;
  }

  const parsed =
    Number.parseInt(
      value,
      10,
    );

  return Number.isSafeInteger(
    parsed,
  )
    ? parsed
    : null;
}

export default function AdminReturnsWorkspace({
  portal,
}: Props) {
  const principal =
    useAdminSession();

  const isSuperAdmin =
    principal.staff.roles.includes(
      "admin_superuser",
    );

  const permissions =
    principal.staff.permissions;

  const hasPermission =
    useCallback(
      (permission: string) =>
        isSuperAdmin ||
        permissions.includes(
          permission,
        ),
      [
        isSuperAdmin,
        permissions,
      ],
    );

  const canRead =
    hasPermission(
      "admin.return.read",
    );

  const canManage =
    hasPermission(
      "admin.return.manage",
    );

  /*
   * Keep this aligned with the currently
   * enforced refund mutation permission.
   */
  const canRefund =
    hasPermission(
      "admin.payment.refund",
    );

  const [
    items,
    setItems,
  ] =
    useState<
      AdminReturnListItem[]
    >([]);

  const [
    meta,
    setMeta,
  ] =
    useState<
      AdminPaginationMeta | null
    >(null);

  const [
    page,
    setPage,
  ] =
    useState(1);

  const [
    status,
    setStatus,
  ] =
    useState<StatusFilter>("");

  const [
    queryInput,
    setQueryInput,
  ] =
    useState("");

  const [
    query,
    setQuery,
  ] =
    useState("");

  const [
    selectedId,
    setSelectedId,
  ] =
    useState<string | null>(
      null,
    );

  const [
    detail,
    setDetail,
  ] =
    useState<
      AdminReturnDetail | null
    >(null);

  const [
    loading,
    setLoading,
  ] =
    useState(true);

  const [
    detailLoading,
    setDetailLoading,
  ] =
    useState(false);

  const [
    mutating,
    setMutating,
  ] =
    useState(false);

  const [
    error,
    setError,
  ] =
    useState("");

  const [
    detailError,
    setDetailError,
  ] =
    useState("");

  const [
    liveAt,
    setLiveAt,
  ] =
    useState<Date | null>(
      null,
    );

  const [
    rejectReason,
    setRejectReason,
  ] =
    useState("");

  const [
    receiveQuantities,
    setReceiveQuantities,
  ] =
    useState<
      Record<
        string,
        string
      >
    >({});

  const [
    inspectionDrafts,
    setInspectionDrafts,
  ] =
    useState<
      Record<
        string,
        InspectionDraft
      >
    >({});

  const [
    refundReason,
    setRefundReason,
  ] =
    useState("");

  const [
    refundProviderReference,
    setRefundProviderReference,
  ] =
    useState<
      Record<
        string,
        string
      >
    >({});

  const [
    refundFailureCode,
    setRefundFailureCode,
  ] =
    useState<
      Record<
        string,
        string
      >
    >({});

  const [
    refundFailureMessage,
    setRefundFailureMessage,
  ] =
    useState<
      Record<
        string,
        string
      >
    >({});

  const listAbortRef =
    useRef<
      AbortController | null
    >(null);

  const drawerRef =
    useRef<
      HTMLDivElement | null
    >(null);

  const loadReturns =
    useCallback(
      async (
        silent = false,
      ) => {
        if (!canRead) {
          setLoading(false);
          return;
        }

        listAbortRef.current?.
          abort();

        const controller =
          new AbortController();

        listAbortRef.current =
          controller;

        if (!silent) {
          setLoading(true);
          setError("");
        }

        const params =
          new URLSearchParams({
            page:
              String(page),
            limit:
              String(
                PAGE_SIZE,
              ),
          });

        if (status) {
          params.set(
            "status",
            status,
          );
        }

        if (
          query.trim()
        ) {
          params.set(
            "q",
            query.trim(),
          );
        }

        try {
          const response =
            await adminFetch<AdminReturnListResponse>(
              `/returns?${params.toString()}`,
              {
                signal:
                  controller.signal,
              },
            );

          if (
            controller.signal
              .aborted
          ) {
            return;
          }

          setItems(
            response.data ?? [],
          );

          setMeta(
            response.meta ??
              null,
          );

          setLiveAt(
            new Date(),
          );
        } catch (
          value: unknown
        ) {
          if (
            controller.signal
              .aborted
          ) {
            return;
          }

          if (!silent) {
            setError(
              errorMessage(
                value,
              ),
            );

            setItems([]);
            setMeta(null);
          }
        } finally {
          if (
            !controller.signal
              .aborted &&
            !silent
          ) {
            setLoading(false);
          }
        }
      },
      [
        canRead,
        page,
        query,
        status,
      ],
    );

  const initializeActionDrafts =
    useCallback(
      (
        next:
          AdminReturnDetail,
      ) => {
        const receive:
          Record<
            string,
            string
          > = {};

        const inspection:
          Record<
            string,
            InspectionDraft
          > = {};

        for (
          const item
          of next.items ?? []
        ) {
          receive[
            item.order_item_id
          ] =
            String(
              item.received_quantity ??
                0,
            );

          inspection[
            item.order_item_id
          ] = {
            inspection_status:
              item.inspection_status ===
                "restockable" ||
              item.inspection_status ===
                "damaged" ||
              item.inspection_status ===
                "non_restockable"
                ? item.inspection_status
                : "pending",

            inspection_note:
              item.inspection_note ??
              "",

            restock_quantity:
              String(
                item.restock_quantity ??
                  0,
              ),
          };
        }

        setReceiveQuantities(
          receive,
        );

        setInspectionDrafts(
          inspection,
        );
      },
      [],
    );

  const loadDetail =
    useCallback(
      async (
        returnId: string,
        silent = false,
      ) => {
        if (!canRead) {
          return;
        }

        if (!silent) {
          setDetailLoading(
            true,
          );

          setDetailError("");
        }

        try {
          const response =
            await adminFetch<AdminReturnDetailResponse>(
              `/returns/${returnId}`,
            );

          setDetail(
            response.data,
          );

          initializeActionDrafts(
            response.data,
          );

          setItems(
            (
              current,
            ) =>
              current.map(
                (item) =>
                  item.id ===
                  response
                    .data.id
                    ? {
                        ...item,

                        status:
                          response
                            .data
                            .status,

                        updated_at:
                          response
                            .data
                            .updated_at,
                      }
                    : item,
              ),
          );
        } catch (
          value: unknown
        ) {
          if (!silent) {
            setDetailError(
              errorMessage(
                value,
              ),
            );
          }
        } finally {
          if (!silent) {
            setDetailLoading(
              false,
            );
          }
        }
      },
      [
        canRead,
        initializeActionDrafts,
      ],
    );

  useEffect(() => {
    void loadReturns();

    return () => {
      listAbortRef.current?.
        abort();
    };
  }, [loadReturns]);

  useEffect(() => {
    const timer =
      window.setTimeout(
        () => {
          setQuery(
            queryInput.trim(),
          );

          setPage(1);
        },
        320,
      );

    return () =>
      window.clearTimeout(
        timer,
      );
  }, [queryInput]);

  useEffect(() => {
    if (!selectedId) {
      setDetail(null);
      setDetailError("");
      setRejectReason("");
      setRefundReason("");

      return;
    }

    void loadDetail(
      selectedId,
    );
  }, [
    loadDetail,
    selectedId,
  ]);

  useEffect(() => {
    const interval =
      window.setInterval(
        () => {
          if (
            document
              .visibilityState !==
              "visible" ||
            mutating
          ) {
            return;
          }

          void loadReturns(
            true,
          );

          if (selectedId) {
            void loadDetail(
              selectedId,
              true,
            );
          }
        },
        LIVE_REFRESH_MS,
      );

    return () =>
      window.clearInterval(
        interval,
      );
  }, [
    loadDetail,
    loadReturns,
    mutating,
    selectedId,
  ]);

  useEffect(() => {
    if (!selectedId) {
      return;
    }

    const onKeyDown =
      (
        event:
          KeyboardEvent,
      ) => {
        if (
          event.key ===
            "Escape" &&
          !mutating
        ) {
          setSelectedId(
            null,
          );
        }
      };

    window.addEventListener(
      "keydown",
      onKeyDown,
    );

    return () =>
      window.removeEventListener(
        "keydown",
        onKeyDown,
      );
  }, [
    mutating,
    selectedId,
  ]);

  useEffect(() => {
    if (selectedId) {
      drawerRef.current?.
        focus();
    }
  }, [selectedId]);

  const refreshAfterMutation =
    useCallback(
      async () => {
        await Promise.all([
          loadReturns(true),

          selectedId
            ? loadDetail(
                selectedId,
                true,
              )
            : Promise.resolve(),
        ]);
      },
      [
        loadDetail,
        loadReturns,
        selectedId,
      ],
    );

  const mutateReturn =
    useCallback(
      async (
        path: string,
        init?: RequestInit,
      ) => {
        if (!selectedId) {
          return;
        }

        setMutating(true);
        setDetailError("");

        try {
          await adminFetch(
            path,
            init ?? {
              method: "POST",
            },
          );

          await refreshAfterMutation();
        } catch (
          value: unknown
        ) {
          setDetailError(
            errorMessage(
              value,
            ),
          );

          throw value;
        } finally {
          setMutating(false);
        }
      },
      [
        refreshAfterMutation,
        selectedId,
      ],
    );

  async function approveReturn() {
    if (!selectedId) {
      return;
    }

    try {
      await mutateReturn(
        `/returns/${selectedId}/approve`,
        {
          method: "POST",
        },
      );
    } catch {
      // Error shown in drawer.
    }
  }

  async function rejectReturn(
    event:
      FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (
      !selectedId ||
      !rejectReason.trim()
    ) {
      return;
    }

    try {
      await mutateReturn(
        `/returns/${selectedId}/reject`,
        {
          method: "POST",

          body:
            JSON.stringify({
              reason:
                rejectReason.trim(),
            }),
        },
      );

      setRejectReason("");
    } catch {
      // Error shown in drawer.
    }
  }

  async function receiveReturn(
    event:
      FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (
      !selectedId ||
      !detail
    ) {
      return;
    }

    const payload =
      detail.items.map(
        (item) => ({
          order_item_id:
            item.order_item_id,

          received_quantity:
            integerValue(
              receiveQuantities[
                item
                  .order_item_id
              ] ?? "",
            ),
        }),
      );

    const invalid =
      payload.some(
        (
          item,
          index,
        ) => {
          const requested =
            detail.items[
              index
            ]?.quantity ??
            0;

          return (
            item.received_quantity ===
              null ||
            item.received_quantity <
              0 ||
            item.received_quantity >
              requested
          );
        },
      );

    if (invalid) {
      setDetailError(
        "Received quantity must be a whole number between 0 and the requested quantity for every item.",
      );

      return;
    }

    try {
      await mutateReturn(
        `/returns/${selectedId}/receive`,
        {
          method: "POST",

          body:
            JSON.stringify({
              items:
                payload.map(
                  (item) => ({
                    order_item_id:
                      item.order_item_id,

                    received_quantity:
                      item.received_quantity,
                  }),
                ),
            }),
        },
      );
    } catch {
      // Error shown in drawer.
    }
  }

  async function inspectReturn(
    event:
      FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (
      !selectedId ||
      !detail
    ) {
      return;
    }

    const payload =
      detail.items.map(
        (item) => {
          const draft =
            inspectionDrafts[
              item
                .order_item_id
            ];

          return {
            item,

            inspection_status:
              draft?.
                inspection_status ??
              "pending",

            inspection_note:
              draft?.
                inspection_note.
                trim() ??
              "",

            restock_quantity:
              integerValue(
                draft?.
                  restock_quantity ??
                  "",
              ),
          };
        },
      );

    const invalid =
      payload.some(
        ({
          item,
          inspection_status,
          restock_quantity,
        }) =>
          inspection_status ===
            "pending" ||
          restock_quantity ===
            null ||
          restock_quantity <
            0 ||
          restock_quantity >
            item.received_quantity ||
          (
            inspection_status !==
              "restockable" &&
            restock_quantity !==
              0
          ),
      );

    if (invalid) {
      setDetailError(
        "Choose an inspection result for every item. Restock quantity must be between 0 and received quantity, and only restockable items may have a restock quantity above zero.",
      );

      return;
    }

    try {
      await mutateReturn(
        `/returns/${selectedId}/inspect`,
        {
          method: "POST",

          body:
            JSON.stringify({
              items:
                payload.map(
                  ({
                    item,
                    inspection_status,
                    inspection_note,
                    restock_quantity,
                  }) => ({
                    order_item_id:
                      item.order_item_id,

                    inspection_status,

                    inspection_note,

                    restock_quantity,
                  }),
                ),
            }),
        },
      );
    } catch {
      // Error shown in drawer.
    }
  }

  async function createRefund(
    event:
      FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (
      !selectedId ||
      !refundReason.trim()
    ) {
      return;
    }

    try {
      await mutateReturn(
        `/returns/${selectedId}/refunds`,
        {
          method: "POST",

          body:
            JSON.stringify({
              reason:
                refundReason.trim(),
            }),
        },
      );

      setRefundReason("");
    } catch {
      // Error shown in drawer.
    }
  }

  async function mutateRefund(
    refund:
      AdminRefundSummary,
    action:
      | "approve"
      | "process"
      | "succeed"
      | "fail",
  ) {
    let body:
      string |
      undefined;

    if (
      action ===
      "succeed"
    ) {
      const providerRefundId =
        refundProviderReference[
          refund.id
        ]?.trim() ??
        "";

      if (
        !providerRefundId
      ) {
        setDetailError(
          "Enter the provider refund reference before marking the refund succeeded.",
        );

        return;
      }

      body =
        JSON.stringify({
          provider_refund_id:
            providerRefundId,
        });
    }

    if (
      action ===
      "fail"
    ) {
      const failureCode =
        refundFailureCode[
          refund.id
        ]?.trim() ??
        "";

      const failureMessage =
        refundFailureMessage[
          refund.id
        ]?.trim() ??
        "";

      if (
        !failureCode ||
        !failureMessage
      ) {
        setDetailError(
          "Enter both a failure code and failure message before marking the refund failed.",
        );

        return;
      }

      body =
        JSON.stringify({
          failure_code:
            failureCode,

          failure_message:
            failureMessage,
        });
    }

    setMutating(true);
    setDetailError("");

    try {
      await adminFetch(
        `/refunds/${refund.id}/${action}`,
        {
          method: "POST",

          ...(body
            ? { body }
            : {}),
        },
      );

      await refreshAfterMutation();
    } catch (
      value: unknown
    ) {
      setDetailError(
        errorMessage(
          value,
        ),
      );
    } finally {
      setMutating(false);
    }
  }

  const visibleRange =
    useMemo(
      () => {
        if (
          !meta ||
          meta.total === 0
        ) {
          return "0 returns";
        }

        const start =
          (
            meta.page - 1
          ) *
            meta.limit +
          1;

        const end =
          Math.min(
            meta.total,
            start +
              items.length -
              1,
          );

        return `${start}–${end} of ${meta.total}`;
      },
      [
        items.length,
        meta,
      ],
    );

  if (!canRead) {
    return (
      <section
        className={
          styles.workspace
        }
      >
        <div
          className={
            styles.permissionState
          }
        >
          <span>
            Returns & refunds
          </span>

          <h1>
            Access not available
          </h1>

          <p>
            Your staff account does
            not have permission to
            read return operations.
          </p>
        </div>
      </section>
    );
  }

  return (
    <section
      className={
        styles.workspace
      }
    >
      <header
        className={
          styles.pageHeader
        }
      >
        <div>
          <p
            className={
              styles.eyebrow
            }
          >
            After-sales operations
          </p>

          <h1>
            Returns & refunds
          </h1>

          <p
            className={
              styles.headerCopy
            }
          >
            Review return requests,
            receive physical items,
            record inspection outcomes,
            and continue authorized
            refunds from one
            operational queue.
          </p>
        </div>

        <div
          className={
            styles.liveCard
          }
        >
          <span
            className={
              styles.liveDot
            }
            aria-hidden="true"
          />

          <div>
            <strong>
              Live operations
            </strong>

            <span>
              {liveAt
                ? `Updated ${formatDateTime(
                    liveAt.toISOString(),
                  )}`
                : "Connecting…"}
            </span>
          </div>
        </div>
      </header>

      <div
        className={
          styles.toolbar
        }
      >
        <label
          className={
            styles.searchField
          }
        >
          <span>
            Search
          </span>

          <input
            value={
              queryInput
            }
            onChange={(
              event,
            ) =>
              setQueryInput(
                event.target
                  .value,
              )
            }
            placeholder="Return no., order no. or customer phone"
            autoComplete="off"
          />
        </label>

        <label
          className={
            styles.filterField
          }
        >
          <span>
            Status
          </span>

          <select
            value={
              status
            }
            onChange={(
              event,
            ) => {
              setStatus(
                event.target
                  .value as
                  StatusFilter,
              );

              setPage(1);
            }}
          >
            {statusOptions.map(
              (
                option,
              ) => (
                <option
                  key={
                    option.value ||
                    "all"
                  }
                  value={
                    option.value
                  }
                >
                  {
                    option.label
                  }
                </option>
              ),
            )}
          </select>
        </label>

        <button
          type="button"
          className={
            styles.refreshButton
          }
          onClick={() =>
            void loadReturns()
          }
          disabled={
            loading
          }
        >
          {loading
            ? "Refreshing…"
            : "Refresh"}
        </button>
      </div>

      {error ? (
        <div
          className={
            styles.bannerError
          }
        >
          {error}
        </div>
      ) : null}

      <div
        className={
          styles.queueHeader
        }
      >
        <div>
          <span>
            Return queue
          </span>

          <strong>
            {visibleRange}
          </strong>
        </div>

        <p>
          Server-filtered and
          paginated for the full
          return dataset.
        </p>
      </div>

      <div
        className={
          styles.tableShell
        }
      >
        <div
          className={
            styles.tableHead
          }
          aria-hidden="true"
        >
          <span>
            Return
          </span>

          <span>
            Order
          </span>

          <span>
            Customer
          </span>

          <span>
            Status
          </span>

          <span>
            Requested
          </span>

          <span />
        </div>

        <div
          className={
            styles.rows
          }
        >
          {loading ? (
            Array.from({
              length: 6,
            }).map(
              (
                _,
                index,
              ) => (
                <div
                  key={
                    index
                  }
                  className={
                    styles.skeletonRow
                  }
                />
              ),
            )
          ) : items.length ===
            0 ? (
            <div
              className={
                styles.emptyState
              }
            >
              <strong>
                No returns matched
                this view
              </strong>

              <p>
                Change the status
                filter or search
                value and try again.
              </p>
            </div>
          ) : (
            items.map(
              (
                item,
              ) => (
                <button
                  key={
                    item.id
                  }
                  type="button"
                  className={
                    styles.returnRow
                  }
                  onClick={() =>
                    setSelectedId(
                      item.id,
                    )
                  }
                >
                  <span
                    className={
                      styles.primaryCell
                    }
                  >
                    <strong>
                      {
                        item.return_number
                      }
                    </strong>

                    <small>
                      {item.id.slice(
                        0,
                        8,
                      )}
                    </small>
                  </span>

                  <span
                    className={
                      styles.standardCell
                    }
                  >
                    <strong>
                      {
                        item.order_number
                      }
                    </strong>

                    <small>
                      Order
                    </small>
                  </span>

                  <span
                    className={
                      styles.standardCell
                    }
                  >
                    <strong>
                      {item.customer_name ||
                        "Customer"}
                    </strong>

                    <small>
                      {item.customer_phone ||
                        "—"}
                    </small>
                  </span>

                  <span>
                    <span
                      className={[
                        styles.statusPill,
                        statusTone(
                          item.status,
                        ),
                      ].join(
                        " ",
                      )}
                    >
                      {titleCase(
                        item.status,
                      )}
                    </span>
                  </span>

                  <span
                    className={
                      styles.dateCell
                    }
                  >
                    {formatDateTime(
                      item.requested_at,
                    )}
                  </span>

                  <span
                    className={
                      styles.openCell
                    }
                  >
                    Open{" "}
                    <b
                      aria-hidden="true"
                    >
                      ›
                    </b>
                  </span>
                </button>
              ),
            )
          )}
        </div>
      </div>

      {meta &&
      meta.total_pages >
        1 ? (
        <div
          className={
            styles.pagination
          }
        >
          <button
            type="button"
            onClick={() =>
              setPage(
                (
                  current,
                ) =>
                  Math.max(
                    1,
                    current -
                      1,
                  ),
              )
            }
            disabled={
              page <= 1 ||
              loading
            }
          >
            Previous
          </button>

          <span>
            Page {meta.page} of{" "}
            {
              meta.total_pages
            }
          </span>

          <button
            type="button"
            onClick={() =>
              setPage(
                (
                  current,
                ) =>
                  Math.min(
                    meta.total_pages,
                    current +
                      1,
                  ),
              )
            }
            disabled={
              page >=
                meta.total_pages ||
              loading
            }
          >
            Next
          </button>
        </div>
      ) : null}

      {selectedId ? (
        <div
          className={
            styles.drawerBackdrop
          }
          onMouseDown={(
            event,
          ) => {
            if (
              event.currentTarget ===
                event.target &&
              !mutating
            ) {
              setSelectedId(
                null,
              );
            }
          }}
        >
          <aside
            ref={
              drawerRef
            }
            className={
              styles.drawer
            }
            tabIndex={-1}
            aria-label="Return details"
          >
            <div
              className={
                styles.drawerTopbar
              }
            >
              <div>
                <span>
                  Return operations
                </span>

                <strong>
                  {detail?.
                    return_number ??
                    "Loading…"}
                </strong>
              </div>

              <button
                type="button"
                className={
                  styles.closeButton
                }
                onClick={() =>
                  setSelectedId(
                    null,
                  )
                }
                disabled={
                  mutating
                }
                aria-label="Close return details"
              >
                ×
              </button>
            </div>

            {detailError ? (
              <div
                className={
                  styles.drawerError
                }
              >
                {
                  detailError
                }
              </div>
            ) : null}

            {detailLoading &&
            !detail ? (
              <div
                className={
                  styles.detailLoading
                }
              >
                Loading return
                details…
              </div>
            ) : detail ? (
              <div
                className={
                  styles.drawerBody
                }
              >
                <section
                  className={
                    styles.heroCard
                  }
                >
                  <div>
                    <span
                      className={[
                        styles.statusPill,
                        statusTone(
                          detail.status,
                        ),
                      ].join(
                        " ",
                      )}
                    >
                      {titleCase(
                        detail.status,
                      )}
                    </span>

                    <h2>
                      {
                        detail.return_number
                      }
                    </h2>

                    <p>
                      {nextReturnStep(
                        detail.status,
                      )}
                    </p>
                  </div>

                  <a
                    href={`/${portal}/orders?order=${encodeURIComponent(
                      detail.order_id,
                    )}`}
                  >
                    Open order{" "}
                    {
                      detail.order_number
                    }{" "}
                    ↗
                  </a>
                </section>

                <section
                  className={
                    styles.detailSection
                  }
                >
                  <div
                    className={
                      styles.sectionHeading
                    }
                  >
                    <div>
                      <span>
                        Customer &
                        request
                      </span>

                      <h3>
                        Return context
                      </h3>
                    </div>
                  </div>

                  <div
                    className={
                      styles.infoGrid
                    }
                  >
                    <span>
                      <small>
                        Customer
                      </small>

                      <strong>
                        {detail.customer_name ||
                          "Customer"}
                      </strong>
                    </span>

                    <span>
                      <small>
                        Phone
                      </small>

                      <strong>
                        {detail.customer_phone ||
                          "—"}
                      </strong>
                    </span>

                    <span>
                      <small>
                        Requested
                      </small>

                      <strong>
                        {formatDateTime(
                          detail.requested_at,
                        )}
                      </strong>
                    </span>

                    <span>
                      <small>
                        Updated
                      </small>

                      <strong>
                        {formatDateTime(
                          detail.updated_at,
                        )}
                      </strong>
                    </span>

                    {detail.customer_note ? (
                      <span
                        className={
                          styles.infoWide
                        }
                      >
                        <small>
                          Customer note
                        </small>

                        <strong>
                          {
                            detail.customer_note
                          }
                        </strong>
                      </span>
                    ) : null}

                    {detail.rejection_reason ? (
                      <span
                        className={
                          styles.infoWide
                        }
                      >
                        <small>
                          Rejection reason
                        </small>

                        <strong>
                          {
                            detail.rejection_reason
                          }
                        </strong>
                      </span>
                    ) : null}
                  </div>
                </section>

                <section
                  className={
                    styles.detailSection
                  }
                >
                  <div
                    className={
                      styles.sectionHeading
                    }
                  >
                    <div>
                      <span>
                        Returned
                        merchandise
                      </span>

                      <h3>
                        {
                          detail.items
                            .length
                        }{" "}
                        item
                        {detail
                          .items
                          .length ===
                        1
                          ? ""
                          : "s"}
                      </h3>
                    </div>
                  </div>

                  <div
                    className={
                      styles.itemStack
                    }
                  >
                    {detail.items.map(
                      (
                        item,
                      ) => (
                        <ReturnItemCard
                          key={
                            item.id
                          }
                          item={
                            item
                          }
                        />
                      ),
                    )}
                  </div>
                </section>

                {canManage &&
                detail.status ===
                  "requested" ? (
                  <section
                    className={
                      styles.actionSection
                    }
                  >
                    <div
                      className={
                        styles.sectionHeading
                      }
                    >
                      <div>
                        <span>
                          Decision
                        </span>

                        <h3>
                          Review return
                          request
                        </h3>
                      </div>
                    </div>

                    <button
                      type="button"
                      className={
                        styles.primaryAction
                      }
                      onClick={() =>
                        void approveReturn()
                      }
                      disabled={
                        mutating
                      }
                    >
                      {mutating
                        ? "Updating…"
                        : "Approve return"}
                    </button>

                    <form
                      className={
                        styles.actionForm
                      }
                      onSubmit={
                        rejectReturn
                      }
                    >
                      <label>
                        <span>
                          Reject with
                          reason
                        </span>

                        <textarea
                          value={
                            rejectReason
                          }
                          onChange={(
                            event,
                          ) =>
                            setRejectReason(
                              event
                                .target
                                .value,
                            )
                          }
                          rows={
                            3
                          }
                          maxLength={
                            500
                          }
                          placeholder="Explain why this return cannot be accepted"
                        />
                      </label>

                      <button
                        type="submit"
                        className={
                          styles.dangerAction
                        }
                        disabled={
                          mutating ||
                          !rejectReason.trim()
                        }
                      >
                        Reject return
                      </button>
                    </form>
                  </section>
                ) : null}

                {canManage &&
                detail.status ===
                  "approved" ? (
                  <section
                    className={
                      styles.actionSection
                    }
                  >
                    <div
                      className={
                        styles.sectionHeading
                      }
                    >
                      <div>
                        <span>
                          Physical
                          receipt
                        </span>

                        <h3>
                          Record received
                          quantities
                        </h3>
                      </div>
                    </div>

                    <form
                      className={
                        styles.quantityForm
                      }
                      onSubmit={
                        receiveReturn
                      }
                    >
                      {detail.items.map(
                        (
                          item,
                        ) => (
                          <label
                            key={
                              item.order_item_id
                            }
                          >
                            <span>
                              <strong>
                                {
                                  item.product_name
                                }
                              </strong>

                              <small>
                                Requested{" "}
                                {
                                  item.quantity
                                }
                              </small>
                            </span>

                            <input
                              type="number"
                              inputMode="numeric"
                              min={
                                0
                              }
                              max={
                                item.quantity
                              }
                              step={
                                1
                              }
                              value={
                                receiveQuantities[
                                  item
                                    .order_item_id
                                ] ??
                                "0"
                              }
                              onChange={(
                                event,
                              ) =>
                                setReceiveQuantities(
                                  (
                                    current,
                                  ) => ({
                                    ...current,

                                    [item.order_item_id]:
                                      event
                                        .target
                                        .value,
                                  }),
                                )
                              }
                            />
                          </label>
                        ),
                      )}

                      <button
                        type="submit"
                        className={
                          styles.primaryAction
                        }
                        disabled={
                          mutating
                        }
                      >
                        {mutating
                          ? "Recording…"
                          : "Confirm received items"}
                      </button>
                    </form>
                  </section>
                ) : null}

                {canManage &&
                detail.status ===
                  "received" ? (
                  <section
                    className={
                      styles.actionSection
                    }
                  >
                    <div
                      className={
                        styles.sectionHeading
                      }
                    >
                      <div>
                        <span>
                          Quality
                          control
                        </span>

                        <h3>
                          Inspect
                          received items
                        </h3>
                      </div>
                    </div>

                    <form
                      className={
                        styles.inspectionForm
                      }
                      onSubmit={
                        inspectReturn
                      }
                    >
                      {detail.items.map(
                        (
                          item,
                        ) => {
                          const draft =
                            inspectionDrafts[
                              item
                                .order_item_id
                            ] ?? {
                              inspection_status:
                                "pending" as const,

                              inspection_note:
                                "",

                              restock_quantity:
                                "0",
                            };

                          return (
                            <article
                              key={
                                item.order_item_id
                              }
                              className={
                                styles.inspectionCard
                              }
                            >
                              <div
                                className={
                                  styles.inspectionTitle
                                }
                              >
                                <div>
                                  <strong>
                                    {
                                      item.product_name
                                    }
                                  </strong>

                                  <span>
                                    {
                                      item.sku
                                    }
                                  </span>
                                </div>

                                <small>
                                  Received{" "}
                                  {
                                    item.received_quantity
                                  }
                                </small>
                              </div>

                              <label>
                                <span>
                                  Inspection
                                  result
                                </span>

                                <select
                                  value={
                                    draft.inspection_status
                                  }
                                  onChange={(
                                    event,
                                  ) =>
                                    setInspectionDrafts(
                                      (
                                        current,
                                      ) => ({
                                        ...current,

                                        [item.order_item_id]:
                                          {
                                            ...draft,

                                            inspection_status:
                                              event
                                                .target
                                                .value as
                                                AdminReturnInspectionStatus,

                                            restock_quantity:
                                              event
                                                .target
                                                .value ===
                                              "restockable"
                                                ? draft.restock_quantity
                                                : "0",
                                          },
                                      }),
                                    )
                                  }
                                >
                                  <option value="pending">
                                    Choose
                                    result
                                  </option>

                                  <option value="restockable">
                                    Restockable
                                  </option>

                                  <option value="damaged">
                                    Damaged
                                  </option>

                                  <option value="non_restockable">
                                    Non-restockable
                                  </option>
                                </select>
                              </label>

                              <label>
                                <span>
                                  Restock
                                  quantity
                                </span>

                                <input
                                  type="number"
                                  min={
                                    0
                                  }
                                  max={
                                    item.received_quantity
                                  }
                                  step={
                                    1
                                  }
                                  disabled={
                                    draft.inspection_status !==
                                    "restockable"
                                  }
                                  value={
                                    draft.restock_quantity
                                  }
                                  onChange={(
                                    event,
                                  ) =>
                                    setInspectionDrafts(
                                      (
                                        current,
                                      ) => ({
                                        ...current,

                                        [item.order_item_id]:
                                          {
                                            ...draft,

                                            restock_quantity:
                                              event
                                                .target
                                                .value,
                                          },
                                      }),
                                    )
                                  }
                                />
                              </label>

                              <label
                                className={
                                  styles.inspectionNote
                                }
                              >
                                <span>
                                  Inspection
                                  note
                                </span>

                                <textarea
                                  rows={
                                    2
                                  }
                                  maxLength={
                                    500
                                  }
                                  value={
                                    draft.inspection_note
                                  }
                                  onChange={(
                                    event,
                                  ) =>
                                    setInspectionDrafts(
                                      (
                                        current,
                                      ) => ({
                                        ...current,

                                        [item.order_item_id]:
                                          {
                                            ...draft,

                                            inspection_note:
                                              event
                                                .target
                                                .value,
                                          },
                                      }),
                                    )
                                  }
                                  placeholder="Condition, packaging, defects or other observations"
                                />
                              </label>
                            </article>
                          );
                        },
                      )}

                      <button
                        type="submit"
                        className={
                          styles.primaryAction
                        }
                        disabled={
                          mutating
                        }
                      >
                        {mutating
                          ? "Saving inspection…"
                          : "Complete inspection"}
                      </button>
                    </form>
                  </section>
                ) : null}

                <section
                  className={
                    styles.detailSection
                  }
                >
                  <div
                    className={
                      styles.sectionHeading
                    }
                  >
                    <div>
                      <span>
                        Financial
                        resolution
                      </span>

                      <h3>
                        Refunds
                      </h3>
                    </div>
                  </div>

                  {detail.refunds
                    .length ===
                  0 ? (
                    <div
                      className={
                        styles.quietState
                      }
                    >
                      <strong>
                        No refund has
                        been created
                      </strong>

                      <p>
                        When this return
                        requires a refund,
                        an authorized
                        operator can start
                        it after
                        inspection.
                      </p>
                    </div>
                  ) : (
                    <div
                      className={
                        styles.refundStack
                      }
                    >
                      {detail.refunds.map(
                        (
                          refund,
                        ) => (
                          <RefundCard
                            key={
                              refund.id
                            }
                            refund={
                              refund
                            }
                            canRefund={
                              canRefund
                            }
                            mutating={
                              mutating
                            }
                            providerReference={
                              refundProviderReference[
                                refund
                                  .id
                              ] ??
                              ""
                            }
                            failureCode={
                              refundFailureCode[
                                refund
                                  .id
                              ] ??
                              ""
                            }
                            failureMessage={
                              refundFailureMessage[
                                refund
                                  .id
                              ] ??
                              ""
                            }
                            onProviderReference={(
                              value,
                            ) =>
                              setRefundProviderReference(
                                (
                                  current,
                                ) => ({
                                  ...current,

                                  [refund.id]:
                                    value,
                                }),
                              )
                            }
                            onFailureCode={(
                              value,
                            ) =>
                              setRefundFailureCode(
                                (
                                  current,
                                ) => ({
                                  ...current,

                                  [refund.id]:
                                    value,
                                }),
                              )
                            }
                            onFailureMessage={(
                              value,
                            ) =>
                              setRefundFailureMessage(
                                (
                                  current,
                                ) => ({
                                  ...current,

                                  [refund.id]:
                                    value,
                                }),
                              )
                            }
                            onAction={(
                              action,
                            ) =>
                              void mutateRefund(
                                refund,
                                action,
                              )
                            }
                          />
                        ),
                      )}
                    </div>
                  )}

                  {canRefund &&
                  (
                    detail.status ===
                      "inspected" ||
                    detail.status ===
                      "completed"
                  ) ? (
                    <form
                      className={
                        styles.actionForm
                      }
                      onSubmit={
                        createRefund
                      }
                    >
                      <label>
                        <span>
                          Refund reason
                        </span>

                        <textarea
                          rows={
                            3
                          }
                          maxLength={
                            500
                          }
                          value={
                            refundReason
                          }
                          onChange={(
                            event,
                          ) =>
                            setRefundReason(
                              event
                                .target
                                .value,
                            )
                          }
                          placeholder="Reason recorded with this refund"
                        />
                      </label>

                      <button
                        type="submit"
                        className={
                          styles.secondaryAction
                        }
                        disabled={
                          mutating ||
                          !refundReason.trim()
                        }
                      >
                        Create return
                        refund
                      </button>
                    </form>
                  ) : null}
                </section>
              </div>
            ) : (
              <div
                className={
                  styles.detailLoading
                }
              >
                Return details are
                unavailable.
              </div>
            )}
          </aside>
        </div>
      ) : null}
    </section>
  );
}

function ReturnItemCard({
  item,
}: {
  item: AdminReturnItem;
}) {
  return (
    <article
      className={
        styles.itemCard
      }
    >
      <div
        className={
          styles.itemTopline
        }
      >
        <div>
          <strong>
            {
              item.product_name
            }
          </strong>

          <span>
            {item.sku}
          </span>
        </div>

        <strong>
          {formatMoney(
            item.unit_price_amount,
            item.currency,
          )}
        </strong>
      </div>

      <div
        className={
          styles.itemFacts
        }
      >
        <span>
          <small>
            Requested
          </small>

          <strong>
            {item.quantity}
          </strong>
        </span>

        <span>
          <small>
            Received
          </small>

          <strong>
            {
              item.received_quantity
            }
          </strong>
        </span>

        <span>
          <small>
            Restock
          </small>

          <strong>
            {
              item.restock_quantity
            }
          </strong>
        </span>

        <span>
          <small>
            Inspection
          </small>

          <strong>
            {titleCase(
              item.inspection_status,
            )}
          </strong>
        </span>
      </div>

      <div
        className={
          styles.itemReason
        }
      >
        <small>
          {titleCase(
            item.reason_code,
          )}
        </small>

        {item.reason_note ? (
          <p>
            {
              item.reason_note
            }
          </p>
        ) : null}

        {item.inspection_note ? (
          <p>
            Inspection:{" "}
            {
              item.inspection_note
            }
          </p>
        ) : null}
      </div>
    </article>
  );
}

type RefundAction =
  | "approve"
  | "process"
  | "succeed"
  | "fail";

type RefundCardProps = {
  refund:
    AdminRefundSummary;

  canRefund:
    boolean;

  mutating:
    boolean;

  providerReference:
    string;

  failureCode:
    string;

  failureMessage:
    string;

  onProviderReference:
    (
      value: string,
    ) => void;

  onFailureCode:
    (
      value: string,
    ) => void;

  onFailureMessage:
    (
      value: string,
    ) => void;

  onAction:
    (
      action:
        RefundAction,
    ) => void;
};

function RefundCard({
  refund,
  canRefund,
  mutating,
  providerReference,
  failureCode,
  failureMessage,
  onProviderReference,
  onFailureCode,
  onFailureMessage,
  onAction,
}: RefundCardProps) {
  return (
    <article
      className={
        styles.refundCard
      }
    >
      <div
        className={
          styles.refundHeader
        }
      >
        <div>
          <strong>
            {
              refund.refund_number
            }
          </strong>

          <span>
            {titleCase(
              refund.source_type,
            )}{" "}
            refund
          </span>
        </div>

        <div
          className={
            styles.refundAmount
          }
        >
          <strong>
            {formatMoney(
              refund.amount,
              refund.currency,
            )}
          </strong>

          <span
            className={[
              styles.statusPill,
              refundTone(
                refund.status,
              ),
            ].join(" ")}
          >
            {titleCase(
              refund.status,
            )}
          </span>
        </div>
      </div>

      <div
        className={
          styles.refundMeta
        }
      >
        <span>
          Requested{" "}
          {formatDateTime(
            refund.requested_at,
          )}
        </span>

        {refund.provider ? (
          <span>
            {titleCase(
              refund.provider,
            )}
          </span>
        ) : null}

        {refund.provider_refund_id ? (
          <span>
            Ref{" "}
            {
              refund.provider_refund_id
            }
          </span>
        ) : null}
      </div>

      {canRefund &&
      refund.status ===
        "requested" ? (
        <button
          type="button"
          className={
            styles.compactAction
          }
          onClick={() =>
            onAction(
              "approve",
            )
          }
          disabled={
            mutating
          }
        >
          Approve refund
        </button>
      ) : null}

      {canRefund &&
      (
        refund.status ===
          "approved" ||
        refund.status ===
          "failed"
      ) ? (
        <button
          type="button"
          className={
            styles.compactAction
          }
          onClick={() =>
            onAction(
              "process",
            )
          }
          disabled={
            mutating
          }
        >
          Start processing
        </button>
      ) : null}

      {canRefund &&
      refund.status ===
        "processing" ? (
        <div
          className={
            styles.refundResolution
          }
        >
          <label>
            <span>
              Provider refund
              reference
            </span>

            <input
              value={
                providerReference
              }
              onChange={(
                event,
              ) =>
                onProviderReference(
                  event.target
                    .value,
                )
              }
              maxLength={
                160
              }
              placeholder="Provider transaction / refund ID"
            />
          </label>

          <button
            type="button"
            className={
              styles.compactAction
            }
            disabled={
              mutating ||
              !providerReference.trim()
            }
            onClick={() =>
              onAction(
                "succeed",
              )
            }
          >
            Mark succeeded
          </button>

          <div
            className={
              styles.failureGrid
            }
          >
            <label>
              <span>
                Failure code
              </span>

              <input
                value={
                  failureCode
                }
                onChange={(
                  event,
                ) =>
                  onFailureCode(
                    event.target
                      .value,
                  )
                }
                maxLength={
                  100
                }
                placeholder="provider_error"
              />
            </label>

            <label>
              <span>
                Failure message
              </span>

              <input
                value={
                  failureMessage
                }
                onChange={(
                  event,
                ) =>
                  onFailureMessage(
                    event.target
                      .value,
                  )
                }
                maxLength={
                  500
                }
                placeholder="What prevented the refund"
              />
            </label>
          </div>

          <button
            type="button"
            className={
              styles.dangerCompactAction
            }
            disabled={
              mutating ||
              !failureCode.trim() ||
              !failureMessage.trim()
            }
            onClick={() =>
              onAction(
                "fail",
              )
            }
          >
            Mark failed
          </button>
        </div>
      ) : null}
    </article>
  );
}