"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import AdminPageHeader from "@/components/admin/layout/components/AdminPageHeader";
import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";
import type {
  AdminInventoryListResponse,
  AdminInventoryStockItem,
} from "@/lib/admin/inventory-types";

import AdminInventoryManageDrawer from "./AdminInventoryManageDrawer";

import styles from "../css/AdminInventory.module.css";

const PAGE_SIZE = 40;
const REQUEST_LIMIT =
  PAGE_SIZE + 1;

const LIVE_SYNC_MS =
  8_000;

type Props = {
  portal?: string;
};

type PageStatus =
  | "all"
  | "low"
  | "out"
  | "reserved"
  | "healthy";

function formatNumber(
  value: number,
) {
  return new Intl.NumberFormat(
    "en",
  ).format(value);
}

function formatDateTime(
  value?: string,
) {
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
      hour: "numeric",
      minute: "2-digit",
    },
  ).format(date);
}

function errorMessage(
  value: unknown,
) {
  if (
    value instanceof
    AdminRequestError
  ) {
    return value.message;
  }

  if (
    value instanceof Error &&
    value.name !==
      "AbortError"
  ) {
    return value.message;
  }

  return "Inventory could not be loaded.";
}

function stockState(
  item:
    AdminInventoryStockItem,
): PageStatus {
  if (
    item.available_quantity <=
    0
  ) {
    return "out";
  }

  if (item.low_stock) {
    return "low";
  }

  if (
    item.quantity_reserved >
    0
  ) {
    return "reserved";
  }

  return "healthy";
}

function stateLabel(
  item:
    AdminInventoryStockItem,
) {
  switch (
    stockState(item)
  ) {
    case "out":
      return "No available stock";

    case "low":
      return "Low stock";

    case "reserved":
      return "Stock reserved";

    case "healthy":
      return "Healthy";

    default:
      return "Inventory";
  }
}

function stateTone(
  item:
    AdminInventoryStockItem,
) {
  switch (
    stockState(item)
  ) {
    case "out":
      return styles.danger;

    case "low":
      return styles.attention;

    case "reserved":
      return styles.info;

    case "healthy":
      return styles.positive;

    default:
      return styles.neutral;
  }
}

export default function AdminInventoryWorkspace({
  portal: _portal,
}: Props) {
  const principal =
    useAdminSession();

  const isSuperAdmin =
    principal.staff.roles.includes(
      "admin_superuser",
    );

  const canRead =
    isSuperAdmin ||
    principal.staff.permissions.includes(
      "admin.inventory.read",
    );

  const [
    items,
    setItems,
  ] =
    useState<
      AdminInventoryStockItem[]
    >([]);

  const [
    offset,
    setOffset,
  ] =
    useState(0);

  const [
    hasNext,
    setHasNext,
  ] =
    useState(false);

  const [
    loading,
    setLoading,
  ] =
    useState(true);

  const [
    error,
    setError,
  ] =
    useState("");

  const [
    lastSyncedAt,
    setLastSyncedAt,
  ] =
    useState<Date | null>(
      null,
    );

  const [
    refreshKey,
    setRefreshKey,
  ] =
    useState(0);

  const [
    pageFilter,
    setPageFilter,
  ] =
    useState<PageStatus>(
      "all",
    );

  const [
    localQuery,
    setLocalQuery,
  ] =
    useState("");

  const [
    selectedVariantId,
    setSelectedVariantId,
  ] =
    useState<
      string | null
    >(null);

  const requestSequence =
    useRef(0);

  const loadInventory =
    useCallback(
      async (
        nextOffset:
          number,

        options?: {
          silent?:
            boolean;

          signal?:
            AbortSignal;

          preserveFilters?:
            boolean;
        },
      ) => {
        if (!canRead) {
          setLoading(
            false,
          );

          return;
        }

        const sequence =
          ++requestSequence.current;

        if (
          !options?.silent
        ) {
          setLoading(
            true,
          );
        }

        setError("");

        try {
          const params =
            new URLSearchParams(
              {
                limit:
                  String(
                    REQUEST_LIMIT,
                  ),

                offset:
                  String(
                    nextOffset,
                  ),
              },
            );

          const response =
            await adminFetch<AdminInventoryListResponse>(
              `/inventory?${params.toString()}`,
              {
                signal:
                  options
                    ?.signal,
              },
            );

          if (
            sequence !==
            requestSequence.current
          ) {
            return;
          }

          const received =
            Array.isArray(
              response.data
                ?.items,
            )
              ? response.data
                  .items
              : [];

          setHasNext(
            received.length >
              PAGE_SIZE,
          );

          setItems(
            received.slice(
              0,
              PAGE_SIZE,
            ),
          );

          setOffset(
            nextOffset,
          );

          setLastSyncedAt(
            new Date(),
          );

          if (
            !options
              ?.preserveFilters
          ) {
            setPageFilter(
              "all",
            );

            setLocalQuery(
              "",
            );
          }
        } catch (
          value: unknown
        ) {
          if (
            value instanceof
              DOMException &&
            value.name ===
              "AbortError"
          ) {
            return;
          }

          if (
            sequence ===
            requestSequence.current
          ) {
            setError(
              errorMessage(
                value,
              ),
            );
          }
        } finally {
          if (
            sequence ===
              requestSequence.current &&
            !options?.silent
          ) {
            setLoading(
              false,
            );
          }
        }
      },
      [
        canRead,
      ],
    );

  useEffect(() => {
    if (!canRead) {
      return;
    }

    const controller =
      new AbortController();

    void loadInventory(
      offset,
      {
        signal:
          controller.signal,

        preserveFilters:
          true,
      },
    );

    return () =>
      controller.abort();
  }, [
    canRead,
    loadInventory,
    offset,
    refreshKey,
  ]);

  useEffect(() => {
    if (!canRead) {
      return;
    }

    const sync =
      () => {
        if (
          document.visibilityState !==
          "visible"
        ) {
          return;
        }

        void loadInventory(
          offset,
          {
            silent:
              true,

            preserveFilters:
              true,
          },
        );
      };

    const intervalId =
      window.setInterval(
        sync,
        LIVE_SYNC_MS,
      );

    document.addEventListener(
      "visibilitychange",
      sync,
    );

    return () => {
      window.clearInterval(
        intervalId,
      );

      document.removeEventListener(
        "visibilitychange",
        sync,
      );
    };
  }, [
    canRead,
    loadInventory,
    offset,
  ]);

  const pageSummary =
    useMemo(
      () => {
        let low = 0;
        let out = 0;
        let reserved = 0;
        let healthy = 0;

        let onHand = 0;
        let available = 0;
        let reservedUnits =
          0;

        for (
          const item of
          items
        ) {
          onHand +=
            item.quantity_on_hand;

          available +=
            item.available_quantity;

          reservedUnits +=
            item.quantity_reserved;

          switch (
            stockState(
              item,
            )
          ) {
            case "out":
              out += 1;
              break;

            case "low":
              low += 1;
              break;

            case "reserved":
              reserved += 1;
              break;

            default:
              healthy +=
                1;
          }
        }

        return {
          low,
          out,
          reserved,
          healthy,
          onHand,
          available,
          reservedUnits,
        };
      },
      [
        items,
      ],
    );

  const visibleItems =
    useMemo(
      () => {
        const query =
          localQuery
            .trim()
            .toLowerCase();

        return items.filter(
          (
            item,
          ) => {
            if (
              pageFilter !==
                "all" &&
              stockState(
                item,
              ) !==
                pageFilter
            ) {
              return false;
            }

            if (!query) {
              return true;
            }

            return [
              item.sku,
              item.product_name,
              item.product_code,
              item.variant_id,
            ].some(
              (
                value,
              ) =>
                value
                  .toLowerCase()
                  .includes(
                    query,
                  ),
            );
          },
        );
      },
      [
        items,
        localQuery,
        pageFilter,
      ],
    );

  const pageNumber =
    Math.floor(
      offset /
        PAGE_SIZE,
    ) + 1;

  function refresh() {
    setRefreshKey(
      (
        value,
      ) =>
        value + 1,
    );
  }

  if (!canRead) {
    return (
      <div
        className={
          styles.page
        }
      >
        <AdminPageHeader
          eyebrow="Stock operations"
          title="Inventory"
          description="Your staff role does not include permission to view inventory."
        />

        <section
          className={
            styles.permissionPanel
          }
        >
          <strong>
            Inventory access
            is restricted
          </strong>

          <p>
            This workspace
            requires the
            admin.inventory.read
            permission.
          </p>
        </section>
      </div>
    );
  }

  return (
    <div
      className={
        styles.page
      }
    >
      <AdminPageHeader
        eyebrow="Stock operations"
        title="Inventory"
        description="Monitor available stock, reservations and reorder thresholds across catalog variants."
        actions={
          <button
            type="button"
            className={
              styles.refreshButton
            }
            onClick={
              refresh
            }
            disabled={
              loading
            }
          >
            <span
              aria-hidden="true"
            >
              ↻
            </span>

            Refresh
          </button>
        }
      />

      <section
        className={
          styles.summaryGrid
        }
        aria-label="Inventory summary for loaded page"
      >
        <button
          type="button"
          className={`${styles.summaryCard} ${styles.summaryBlue}`}
          onClick={() =>
            setPageFilter(
              "all",
            )
          }
        >
          <span>
            Loaded variants
          </span>

          <strong>
            {formatNumber(
              items.length,
            )}
          </strong>

          <small>
            Page{" "}
            {pageNumber}
            {" · "}
            {formatNumber(
              pageSummary.onHand,
            )}{" "}
            units on hand
          </small>
        </button>

        <button
          type="button"
          className={`${styles.summaryCard} ${styles.summaryEmerald}`}
          onClick={() =>
            setPageFilter(
              "healthy",
            )
          }
        >
          <span>
            Healthy
          </span>

          <strong>
            {formatNumber(
              pageSummary.healthy,
            )}
          </strong>

          <small>
            Above reorder
            threshold
          </small>
        </button>

        <button
          type="button"
          className={`${styles.summaryCard} ${styles.summaryGold}`}
          onClick={() =>
            setPageFilter(
              "low",
            )
          }
        >
          <span>
            Low stock
          </span>

          <strong>
            {formatNumber(
              pageSummary.low,
            )}
          </strong>

          <small>
            Available stock at
            or below reorder
            level
          </small>
        </button>

        <button
          type="button"
          className={`${styles.summaryCard} ${styles.summaryRose}`}
          onClick={() =>
            setPageFilter(
              "out",
            )
          }
        >
          <span>
            No availability
          </span>

          <strong>
            {formatNumber(
              pageSummary.out,
            )}
          </strong>

          <small>
            Zero or negative
            available quantity
          </small>
        </button>

        <button
          type="button"
          className={`${styles.summaryCard} ${styles.summaryViolet}`}
          onClick={() =>
            setPageFilter(
              "reserved",
            )
          }
        >
          <span>
            Reserved
          </span>

          <strong>
            {formatNumber(
              pageSummary.reservedUnits,
            )}
          </strong>

          <small>
            Units reserved on
            this loaded page
          </small>
        </button>

        <div
          className={`${styles.summaryCard} ${styles.summarySky}`}
        >
          <span>
            Available units
          </span>

          <strong>
            {formatNumber(
              pageSummary.available,
            )}
          </strong>

          <small>
            On hand minus
            reserved quantity
          </small>
        </div>
      </section>

      <section
        className={
          styles.inventoryPanel
        }
      >
        <div
          className={
            styles.panelHeader
          }
        >
          <div>
            <span
              className={
                styles.eyebrow
              }
            >
              Variant stock
            </span>

            <h2>
              Inventory queue
            </h2>

            <p>
              Showing up to{" "}
              {PAGE_SIZE} variants
              per page. Search and
              stock-state filters
              apply only to the
              currently loaded
              page.
            </p>
          </div>

          <div
            className={
              styles.panelMeta
            }
          >
            <span
              className={
                styles.liveMeta
              }
            >
              <span
                className={
                  styles.liveDot
                }
                aria-hidden="true"
              />

              Live

              {lastSyncedAt
                ? ` · ${formatDateTime(
                    lastSyncedAt.toISOString(),
                  )}`
                : ""}
            </span>

            <span
              className={
                styles.pageBadge
              }
            >
              Page{" "}
              {pageNumber}
            </span>
          </div>
        </div>

        <div
          className={
            styles.filters
          }
        >
          <label
            className={
              styles.searchField
            }
          >
            <span
              aria-hidden="true"
            >
              ⌕
            </span>

            <input
              value={
                localQuery
              }
              onChange={(
                event,
              ) =>
                setLocalQuery(
                  event.target
                    .value,
                )
              }
              placeholder="Filter loaded page by product, SKU or code…"
              maxLength={
                120
              }
              autoComplete="off"
            />

            {localQuery ? (
              <button
                type="button"
                aria-label="Clear inventory filter"
                onClick={() =>
                  setLocalQuery(
                    "",
                  )
                }
              >
                ×
              </button>
            ) : null}
          </label>

          <select
            value={
              pageFilter
            }
            onChange={(
              event,
            ) =>
              setPageFilter(
                event.target
                  .value as PageStatus,
              )
            }
            aria-label="Inventory status filter"
          >
            <option value="all">
              All loaded stock
            </option>

            <option value="healthy">
              Healthy
            </option>

            <option value="low">
              Low stock
            </option>

            <option value="out">
              No availability
            </option>

            <option value="reserved">
              Reserved stock
            </option>
          </select>
        </div>

        {error ? (
          <div
            className={
              styles.errorBanner
            }
            role="alert"
          >
            {error}
          </div>
        ) : null}

        <div
          className={
            styles.tableWrap
          }
        >
          <div
            className={
              styles.tableHeader
            }
            aria-hidden="true"
          >
            <span>
              Product / variant
            </span>

            <span>
              Availability
            </span>

            <span>
              On hand
            </span>

            <span>
              Reserved
            </span>

            <span>
              Reorder level
            </span>

            <span>
              Status
            </span>

            <span>
              Updated
            </span>

            <span>
              Action
            </span>
          </div>

          <div
            className={
              styles.rows
            }
          >
            {loading &&
            items.length ===
              0
              ? Array.from(
                  {
                    length:
                      7,
                  },
                  (
                    _,
                    index,
                  ) => (
                    <div
                      key={
                        index
                      }
                      className={
                        styles.loadingRow
                      }
                    >
                      <span />
                      <span />
                      <span />
                      <span />
                      <span />
                    </div>
                  ),
                )
              : visibleItems.map(
                  (
                    item,
                  ) => (
                    <article
                      key={
                        item.variant_id
                      }
                      className={
                        styles.row
                      }
                    >
                      <span
                        className={
                          styles.productCell
                        }
                      >
                        <strong>
                          {
                            item.product_name
                          }
                        </strong>

                        <small>
                          SKU{" "}
                          {
                            item.sku
                          }
                        </small>

                        <small>
                          {
                            item.product_code
                          }
                        </small>
                      </span>

                      <span
                        className={
                          styles.quantityCell
                        }
                      >
                        <strong>
                          {formatNumber(
                            item.available_quantity,
                          )}
                        </strong>

                        <small>
                          Available
                        </small>
                      </span>

                      <span
                        className={
                          styles.quantityCell
                        }
                      >
                        <strong>
                          {formatNumber(
                            item.quantity_on_hand,
                          )}
                        </strong>

                        <small>
                          Physical
                          stock
                        </small>
                      </span>

                      <span
                        className={
                          styles.quantityCell
                        }
                      >
                        <strong>
                          {formatNumber(
                            item.quantity_reserved,
                          )}
                        </strong>

                        <small>
                          Committed
                          quantity
                        </small>
                      </span>

                      <span
                        className={
                          styles.quantityCell
                        }
                      >
                        <strong>
                          {formatNumber(
                            item.reorder_level,
                          )}
                        </strong>

                        <small>
                          Alert
                          threshold
                        </small>
                      </span>

                      <span
                        className={
                          styles.statusCell
                        }
                      >
                        <span
                          className={`${styles.statusPill} ${stateTone(
                            item,
                          )}`}
                        >
                          {stateLabel(
                            item,
                          )}
                        </span>

                        <small>
                          {item.low_stock
                            ? "Reorder attention required"
                            : "Stock within threshold"}
                        </small>
                      </span>

                      <span
                        className={
                          styles.updatedCell
                        }
                      >
                        <strong>
                          {formatDateTime(
                            item.updated_at,
                          )}
                        </strong>

                        <small>
                          Inventory
                          record
                        </small>
                      </span>

                      <span
                        className={
                          styles.actionCell
                        }
                      >
                        <button
                          type="button"
                          className={
                            styles.rowAction
                          }
                          onClick={() =>
                            setSelectedVariantId(
                              item.variant_id,
                            )
                          }
                        >
                          Manage →
                        </button>
                      </span>
                    </article>
                  ),
                )}

            {!loading &&
            visibleItems.length ===
              0 ? (
              <div
                className={
                  styles.emptyState
                }
              >
                <strong>
                  No inventory
                  matches this
                  loaded-page view
                </strong>

                <p>
                  Clear the local
                  filter or choose
                  another stock
                  state. Use page
                  navigation to
                  inspect other
                  variants.
                </p>
              </div>
            ) : null}
          </div>
        </div>

        <div
          className={
            styles.pagination
          }
        >
          <button
            type="button"
            disabled={
              offset === 0 ||
              loading
            }
            onClick={() =>
              setOffset(
                Math.max(
                  0,
                  offset -
                    PAGE_SIZE,
                ),
              )
            }
          >
            ← Previous
          </button>

          <span>
            Page{" "}
            {pageNumber}
          </span>

          <button
            type="button"
            disabled={
              !hasNext ||
              loading
            }
            onClick={() =>
              setOffset(
                offset +
                  PAGE_SIZE,
              )
            }
          >
            Next →
          </button>
        </div>
      </section>

      <AdminInventoryManageDrawer
        variantId={
          selectedVariantId
        }
        onClose={() =>
          setSelectedVariantId(
            null,
          )
        }
        onUpdated={
          refresh
        }
      />
    </div>
  );
}