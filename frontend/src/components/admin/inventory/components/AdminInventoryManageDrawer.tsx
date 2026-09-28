"use client";

import type { FormEvent } from "react";
import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { createPortal } from "react-dom";

import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";
import type {
  AdminInventoryAdjustmentResponse,
  AdminInventoryItemResponse,
  AdminInventoryMovement,
  AdminInventoryMovementListResponse,
  AdminInventoryStockItem,
} from "@/lib/admin/inventory-types";

import styles from "../css/AdminInventoryManageDrawer.module.css";

const MOVEMENT_PAGE_SIZE = 25;

const MOVEMENT_REQUEST_LIMIT =
  MOVEMENT_PAGE_SIZE + 1;

const adjustmentReasons = [
  [
    "manual_count_correction",
    "Stock count correction",
  ],
  [
    "damaged_stock",
    "Damaged stock",
  ],
  [
    "found_stock",
    "Found stock",
  ],
  [
    "administrative_correction",
    "Administrative correction",
  ],
] as const;

type AdjustmentReason =
  (typeof adjustmentReasons)[number][0];

type Props = {
  variantId: string | null;
  onClose: () => void;
  onUpdated: () => void;
};

function formatNumber(
  value: number,
) {
  return new Intl.NumberFormat(
    "en",
  ).format(value);
}

function formatSigned(
  value: number,
) {
  return value > 0
    ? `+${formatNumber(value)}`
    : formatNumber(value);
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
      year: "numeric",
      hour: "numeric",
      minute: "2-digit",
    },
  ).format(date);
}

function titleCase(
  value?: string,
) {
  if (!value) {
    return "—";
  }

  return value
    .replace(
      /[._-]+/g,
      " ",
    )
    .replace(
      /\b\w/g,
      (character) =>
        character.toUpperCase(),
    );
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

  return "Inventory operation failed.";
}

function deltaTone(
  value: number,
) {
  if (value > 0) {
    return styles.deltaPositive;
  }

  if (value < 0) {
    return styles.deltaNegative;
  }

  return styles.deltaNeutral;
}

export default function AdminInventoryManageDrawer({
  variantId,
  onClose,
  onUpdated,
}: Props) {
  const principal =
    useAdminSession();

  const canAdjust =
    principal.staff.roles.includes(
      "admin_superuser",
    ) ||
    principal.staff.permissions.includes(
      "admin.inventory.adjust",
    );

  const [
    stock,
    setStock,
  ] =
    useState<AdminInventoryStockItem | null>(
      null,
    );

  const [
    movements,
    setMovements,
  ] =
    useState<
      AdminInventoryMovement[]
    >([]);

  const [
    movementOffset,
    setMovementOffset,
  ] =
    useState(0);

  const [
    hasMoreMovements,
    setHasMoreMovements,
  ] =
    useState(false);

  const [
    loading,
    setLoading,
  ] =
    useState(false);

  const [
    loadingMore,
    setLoadingMore,
  ] =
    useState(false);

  const [
    operation,
    setOperation,
  ] =
    useState<
      "" |
      "reorder" |
      "adjust"
    >("");

  const [
    error,
    setError,
  ] =
    useState("");

  const [
    notice,
    setNotice,
  ] =
    useState("");

  const [
    reorderLevel,
    setReorderLevel,
  ] =
    useState("");

  const [
    quantityDelta,
    setQuantityDelta,
  ] =
    useState("");

  const [
    adjustmentReason,
    setAdjustmentReason,
  ] =
    useState<AdjustmentReason>(
      adjustmentReasons[0][0],
    );

  const [
    adjustmentNote,
    setAdjustmentNote,
  ] =
    useState("");

  const loadMovementPage =
    useCallback(
      async (
        nextOffset: number,
        append: boolean,
      ) => {
        if (!variantId) {
          return;
        }

        const response =
          await adminFetch<AdminInventoryMovementListResponse>(
            `/inventory/${encodeURIComponent(
              variantId,
            )}/movements?limit=${MOVEMENT_REQUEST_LIMIT}&offset=${nextOffset}`,
          );

        const received =
          Array.isArray(
            response.data
              ?.items,
          )
            ? response.data
                .items
            : [];

        const visible =
          received.slice(
            0,
            MOVEMENT_PAGE_SIZE,
          );

        setMovements(
          (current) => {
            if (!append) {
              return visible;
            }

            const existing =
              new Set(
                current.map(
                  (
                    movement,
                  ) =>
                    movement.id,
                ),
              );

            return [
              ...current,

              ...visible.filter(
                (
                  movement,
                ) =>
                  !existing.has(
                    movement.id,
                  ),
              ),
            ];
          },
        );

        setMovementOffset(
          nextOffset,
        );

        setHasMoreMovements(
          received.length >
            MOVEMENT_PAGE_SIZE,
        );
      },
      [
        variantId,
      ],
    );

  const loadDrawer =
    useCallback(
      async () => {
        if (!variantId) {
          return;
        }

        setLoading(true);
        setError("");
        setNotice("");

        try {
          const [
            detailResponse,
            movementResponse,
          ] =
            await Promise.all(
              [
                adminFetch<AdminInventoryItemResponse>(
                  `/inventory/${encodeURIComponent(
                    variantId,
                  )}`,
                ),

                adminFetch<AdminInventoryMovementListResponse>(
                  `/inventory/${encodeURIComponent(
                    variantId,
                  )}/movements?limit=${MOVEMENT_REQUEST_LIMIT}&offset=0`,
                ),
              ],
            );

          const received =
            Array.isArray(
              movementResponse
                .data?.items,
            )
              ? movementResponse
                  .data.items
              : [];

          setStock(
            detailResponse.data,
          );

          setReorderLevel(
            String(
              detailResponse
                .data
                .reorder_level,
            ),
          );

          setMovements(
            received.slice(
              0,
              MOVEMENT_PAGE_SIZE,
            ),
          );

          setMovementOffset(0);

          setHasMoreMovements(
            received.length >
              MOVEMENT_PAGE_SIZE,
          );
        } catch (
          value: unknown
        ) {
          setError(
            errorMessage(
              value,
            ),
          );
        } finally {
          setLoading(false);
        }
      },
      [
        variantId,
      ],
    );

  useEffect(() => {
    setStock(null);
    setMovements([]);
    setMovementOffset(0);
    setHasMoreMovements(
      false,
    );

    setReorderLevel("");
    setQuantityDelta("");

    setAdjustmentReason(
      adjustmentReasons[0][0],
    );

    setAdjustmentNote("");
    setError("");
    setNotice("");

    if (variantId) {
      void loadDrawer();
    }
  }, [
    loadDrawer,
    variantId,
  ]);

  useEffect(() => {
    if (!variantId) {
      return;
    }

    const previousOverflow =
      document.body.style
        .overflow;

    document.body.style.overflow =
      "hidden";

    const keyDown =
      (
        event: KeyboardEvent,
      ) => {
        if (
          event.key ===
            "Escape" &&
          !operation
        ) {
          onClose();
        }
      };

    document.addEventListener(
      "keydown",
      keyDown,
    );

    return () => {
      document.body.style.overflow =
        previousOverflow;

      document.removeEventListener(
        "keydown",
        keyDown,
      );
    };
  }, [
    onClose,
    operation,
    variantId,
  ]);

  const projectedOnHand =
    useMemo(
      () => {
        if (
          !stock ||
          !quantityDelta.trim()
        ) {
          return null;
        }

        const delta =
          Number(
            quantityDelta,
          );

        return Number.isInteger(
          delta,
        )
          ? stock.quantity_on_hand +
              delta
          : null;
      },
      [
        quantityDelta,
        stock,
      ],
    );

  const projectedAvailable =
    projectedOnHand ===
      null ||
    !stock
      ? null
      : projectedOnHand -
        stock.quantity_reserved;

  async function updateReorderLevel(
    event:
      FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (
      !variantId ||
      !stock ||
      !canAdjust ||
      operation
    ) {
      return;
    }

    const nextLevel =
      Number(
        reorderLevel,
      );

    if (
      !Number.isInteger(
        nextLevel,
      ) ||
      nextLevel < 0
    ) {
      setError(
        "Reorder level must be a non-negative whole number.",
      );

      return;
    }

    if (
      nextLevel ===
      stock.reorder_level
    ) {
      setError("");

      setNotice(
        "Reorder level is already up to date.",
      );

      return;
    }

    setOperation(
      "reorder",
    );

    setError("");
    setNotice("");

    try {
      const response =
        await adminFetch<AdminInventoryItemResponse>(
          `/inventory/${encodeURIComponent(
            variantId,
          )}`,
          {
            method:
              "PATCH",

            body:
              JSON.stringify(
                {
                  reorder_level:
                    nextLevel,
                },
              ),
          },
        );

      setStock(
        response.data,
      );

      setReorderLevel(
        String(
          response.data
            .reorder_level,
        ),
      );

      setNotice(
        "Reorder threshold updated.",
      );

      onUpdated();
    } catch (
      value: unknown
    ) {
      setError(
        errorMessage(
          value,
        ),
      );
    } finally {
      setOperation("");
    }
  }

  async function submitAdjustment(
    event:
      FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (
      !variantId ||
      !stock ||
      !canAdjust ||
      operation
    ) {
      return;
    }

    const delta =
      Number(
        quantityDelta,
      );

    const reason =
      adjustmentReason.trim();

    const note =
      adjustmentNote.trim();

    if (
      !Number.isInteger(
        delta,
      ) ||
      delta === 0
    ) {
      setError(
        "Adjustment must be a non-zero whole number.",
      );

      return;
    }

    if (!reason) {
      setError(
        "Choose a reason for the stock adjustment.",
      );

      return;
    }

    const nextOnHand =
      stock.quantity_on_hand +
      delta;

    if (
      nextOnHand <
      0
    ) {
      setError(
        "This adjustment would make on-hand stock negative.",
      );

      return;
    }

    if (
      nextOnHand <
      stock.quantity_reserved
    ) {
      setError(
        `This would leave ${formatNumber(
          stock.quantity_reserved,
        )} reserved units above the new on-hand balance of ${formatNumber(
          nextOnHand,
        )}.`,
      );

      return;
    }

    setOperation(
      "adjust",
    );

    setError("");
    setNotice("");

    try {
      const response =
        await adminFetch<AdminInventoryAdjustmentResponse>(
          `/inventory/${encodeURIComponent(
            variantId,
          )}/adjust`,
          {
            method:
              "POST",

            body:
              JSON.stringify(
                {
                  quantity_delta:
                    delta,

                  reason,

                  ...(note
                    ? {
                        note,
                      }
                    : {}),
                },
              ),
          },
        );

      setStock(
        response.data,
      );

      setReorderLevel(
        String(
          response.data
            .reorder_level,
        ),
      );

      setQuantityDelta(
        "",
      );

      setAdjustmentNote(
        "",
      );

      setNotice(
        `On-hand inventory adjusted by ${formatSigned(
          delta,
        )} units.`,
      );

      await loadMovementPage(
        0,
        false,
      );

      onUpdated();
    } catch (
      value: unknown
    ) {
      setError(
        errorMessage(
          value,
        ),
      );
    } finally {
      setOperation("");
    }
  }

  async function loadOlderMovements() {
    if (
      loadingMore ||
      !hasMoreMovements
    ) {
      return;
    }

    setLoadingMore(true);
    setError("");

    try {
      await loadMovementPage(
        movementOffset +
          MOVEMENT_PAGE_SIZE,
        true,
      );
    } catch (
      value: unknown
    ) {
      setError(
        errorMessage(
          value,
        ),
      );
    } finally {
      setLoadingMore(
        false,
      );
    }
  }

  if (
    !variantId ||
    typeof document ===
      "undefined"
  ) {
    return null;
  }

  return createPortal(
    <div
      className={
        styles.layer
      }
      role="presentation"
    >
      <button
        type="button"
        className={
          styles.backdrop
        }
        aria-label="Close inventory controls"
        onClick={() => {
          if (!operation) {
            onClose();
          }
        }}
      />

      <aside
        className={
          styles.drawer
        }
        role="dialog"
        aria-modal="true"
        aria-labelledby="inventory-manage-title"
      >
        <header
          className={
            styles.header
          }
        >
          <div
            className={
              styles.headerCopy
            }
          >
            <span>
              Inventory control
            </span>

            <h2 id="inventory-manage-title">
              {stock?.product_name ??
                "Loading inventory…"}
            </h2>

            {stock ? (
              <small>
                {stock.sku}
                {" · "}
                {
                  stock.product_code
                }
              </small>
            ) : null}
          </div>

          <div
            className={
              styles.headerActions
            }
          >
            <button
              type="button"
              className={
                styles.refreshButton
              }
              aria-label="Refresh inventory details"
              onClick={() =>
                void loadDrawer()
              }
              disabled={
                loading ||
                Boolean(
                  operation,
                )
              }
            >
              ↻
            </button>

            <button
              type="button"
              className={
                styles.closeButton
              }
              aria-label="Close inventory controls"
              onClick={
                onClose
              }
              disabled={
                Boolean(
                  operation,
                )
              }
            >
              ×
            </button>
          </div>
        </header>

        <div
          className={
            styles.scroll
          }
        >
          {loading &&
          !stock ? (
            <div
              className={
                styles.skeleton
              }
              aria-label="Loading inventory controls"
            >
              <span />
              <span />
              <span />
              <span />
            </div>
          ) : error &&
            !stock ? (
            <div
              className={
                styles.fatalError
              }
              role="alert"
            >
              <strong>
                Inventory could
                not be opened
              </strong>

              <p>
                {error}
              </p>

              <button
                type="button"
                onClick={() =>
                  void loadDrawer()
                }
              >
                Try again
              </button>
            </div>
          ) : stock ? (
            <>
              <section
                className={
                  styles.hero
                }
              >
                <div
                  className={
                    styles.heroTopline
                  }
                >
                  <span
                    className={`${styles.statusPill} ${
                      stock.available_quantity <=
                      0
                        ? styles.statusDanger
                        : stock.low_stock
                          ? styles.statusAttention
                          : styles.statusPositive
                    }`}
                  >
                    {stock.available_quantity <=
                    0
                      ? "No availability"
                      : stock.low_stock
                        ? "Low stock"
                        : "Stock healthy"}
                  </span>

                  <span
                    className={
                      styles.updatedAt
                    }
                  >
                    Updated{" "}
                    {formatDateTime(
                      stock.updated_at,
                    )}
                  </span>
                </div>

                <div
                  className={
                    styles.balanceGrid
                  }
                >
                  <div>
                    <span>
                      Available
                    </span>

                    <strong>
                      {formatNumber(
                        stock.available_quantity,
                      )}
                    </strong>
                  </div>

                  <div>
                    <span>
                      On hand
                    </span>

                    <strong>
                      {formatNumber(
                        stock.quantity_on_hand,
                      )}
                    </strong>
                  </div>

                  <div>
                    <span>
                      Reserved
                    </span>

                    <strong>
                      {formatNumber(
                        stock.quantity_reserved,
                      )}
                    </strong>
                  </div>

                  <div>
                    <span>
                      Reorder level
                    </span>

                    <strong>
                      {formatNumber(
                        stock.reorder_level,
                      )}
                    </strong>
                  </div>
                </div>
              </section>

              {notice ? (
                <div
                  className={
                    styles.notice
                  }
                  role="status"
                >
                  ✓ {notice}
                </div>
              ) : null}

              {error ? (
                <div
                  className={
                    styles.inlineError
                  }
                  role="alert"
                >
                  {error}
                </div>
              ) : null}

              <section
                className={
                  styles.section
                }
              >
                <div
                  className={
                    styles.sectionHeading
                  }
                >
                  <div>
                    <span>
                      Reorder
                      threshold
                    </span>

                    <h3>
                      Low-stock
                      trigger
                    </h3>
                  </div>

                  <small>
                    Current:{" "}
                    {formatNumber(
                      stock.reorder_level,
                    )}
                  </small>
                </div>

                {canAdjust ? (
                  <form
                    className={
                      styles.compactForm
                    }
                    onSubmit={
                      updateReorderLevel
                    }
                  >
                    <label>
                      <span>
                        Reorder level
                      </span>

                      <input
                        type="number"
                        inputMode="numeric"
                        min={0}
                        step={1}
                        value={
                          reorderLevel
                        }
                        onChange={(
                          event,
                        ) =>
                          setReorderLevel(
                            event
                              .target
                              .value,
                          )
                        }
                        disabled={
                          Boolean(
                            operation,
                          )
                        }
                      />
                    </label>

                    <button
                      type="submit"
                      disabled={
                        Boolean(
                          operation,
                        )
                      }
                    >
                      {operation ===
                      "reorder"
                        ? "Saving…"
                        : "Save threshold"}
                    </button>
                  </form>
                ) : (
                  <p
                    className={
                      styles.readOnlyNote
                    }
                  >
                    Your role can
                    inspect this
                    threshold but
                    cannot change
                    it.
                  </p>
                )}
              </section>

              <section
                className={
                  styles.section
                }
              >
                <div
                  className={
                    styles.sectionHeading
                  }
                >
                  <div>
                    <span>
                      Manual
                      correction
                    </span>

                    <h3>
                      Adjust
                      on-hand stock
                    </h3>
                  </div>

                  <small>
                    Reserved stock
                    is not edited
                    here.
                  </small>
                </div>

                {canAdjust ? (
                  <form
                    className={
                      styles.adjustForm
                    }
                    onSubmit={
                      submitAdjustment
                    }
                  >
                    <div
                      className={
                        styles.twoColumns
                      }
                    >
                      <label>
                        <span>
                          Quantity
                          delta
                        </span>

                        <input
                          type="number"
                          inputMode="numeric"
                          step={1}
                          placeholder="e.g. 5 or -2"
                          value={
                            quantityDelta
                          }
                          onChange={(
                            event,
                          ) =>
                            setQuantityDelta(
                              event
                                .target
                                .value,
                            )
                          }
                          disabled={
                            Boolean(
                              operation,
                            )
                          }
                        />
                      </label>

                      <label>
                        <span>
                          Reason
                        </span>

                        <select
                          value={
                            adjustmentReason
                          }
                          onChange={(
                            event,
                          ) =>
                            setAdjustmentReason(
                              event
                                .target
                                .value as AdjustmentReason,
                            )
                          }
                          disabled={
                            Boolean(
                              operation,
                            )
                          }
                        >
                          {adjustmentReasons.map(
                            ([
                              value,
                              label,
                            ]) => (
                              <option
                                key={
                                  value
                                }
                                value={
                                  value
                                }
                              >
                                {
                                  label
                                }
                              </option>
                            ),
                          )}
                        </select>
                      </label>
                    </div>

                    <label>
                      <span>
                        Internal
                        note ·
                        optional
                      </span>

                      <textarea
                        rows={3}
                        maxLength={
                          500
                        }
                        placeholder="Explain the physical count or exception that requires this correction."
                        value={
                          adjustmentNote
                        }
                        onChange={(
                          event,
                        ) =>
                          setAdjustmentNote(
                            event
                              .target
                              .value,
                          )
                        }
                        disabled={
                          Boolean(
                            operation,
                          )
                        }
                      />
                    </label>

                    <div
                      className={
                        styles.adjustPreview
                      }
                    >
                      <div>
                        <span>
                          Projected
                          on hand
                        </span>

                        <strong>
                          {projectedOnHand ===
                          null
                            ? "—"
                            : formatNumber(
                                projectedOnHand,
                              )}
                        </strong>
                      </div>

                      <div>
                        <span>
                          Projected
                          available
                        </span>

                        <strong>
                          {projectedAvailable ===
                          null
                            ? "—"
                            : formatNumber(
                                projectedAvailable,
                              )}
                        </strong>
                      </div>
                    </div>

                    <div
                      className={
                        styles.adjustFooter
                      }
                    >
                      <p>
                        Use this only
                        for verified
                        corrections or
                        stock
                        exceptions.
                        Normal order
                        reservations
                        and commits
                        remain
                        controlled by
                        their existing
                        workflows.
                      </p>

                      <button
                        type="submit"
                        disabled={
                          Boolean(
                            operation,
                          )
                        }
                      >
                        {operation ===
                        "adjust"
                          ? "Applying…"
                          : "Apply adjustment"}
                      </button>
                    </div>
                  </form>
                ) : (
                  <p
                    className={
                      styles.readOnlyNote
                    }
                  >
                    Manual stock
                    adjustment
                    requires the
                    admin.inventory.adjust
                    permission.
                  </p>
                )}
              </section>

              <section
                className={
                  styles.section
                }
              >
                <div
                  className={
                    styles.sectionHeading
                  }
                >
                  <div>
                    <span>
                      Audit trail
                    </span>

                    <h3>
                      Inventory
                      movements
                    </h3>
                  </div>

                  <small>
                    Newest first
                  </small>
                </div>

                {movements.length ? (
                  <div
                    className={
                      styles.movementList
                    }
                  >
                    {movements.map(
                      (
                        movement,
                      ) => (
                        <article
                          key={
                            movement.id
                          }
                          className={
                            styles.movement
                          }
                        >
                          <div
                            className={
                              styles.movementTopline
                            }
                          >
                            <div>
                              <strong>
                                {titleCase(
                                  movement.movement_type,
                                )}
                              </strong>

                              <span>
                                {formatDateTime(
                                  movement.created_at,
                                )}
                              </span>
                            </div>

                            <div
                              className={
                                styles.deltaGroup
                              }
                            >
                              <span
                                className={deltaTone(
                                  movement.quantity_on_hand_delta,
                                )}
                              >
                                On hand{" "}
                                {formatSigned(
                                  movement.quantity_on_hand_delta,
                                )}
                              </span>

                              {movement.quantity_reserved_delta !==
                              0 ? (
                                <span
                                  className={deltaTone(
                                    movement.quantity_reserved_delta,
                                  )}
                                >
                                  Reserved{" "}
                                  {formatSigned(
                                    movement.quantity_reserved_delta,
                                  )}
                                </span>
                              ) : null}
                            </div>
                          </div>

                          <div
                            className={
                              styles.movementBalances
                            }
                          >
                            <span>
                              <small>
                                On hand
                                after
                              </small>

                              <strong>
                                {formatNumber(
                                  movement.quantity_on_hand_after,
                                )}
                              </strong>
                            </span>

                            <span>
                              <small>
                                Reserved
                                after
                              </small>

                              <strong>
                                {formatNumber(
                                  movement.quantity_reserved_after,
                                )}
                              </strong>
                            </span>
                          </div>

                          {movement.reason ||
                          movement.note ? (
                            <div
                              className={
                                styles.movementDescription
                              }
                            >
                              {movement.reason ? (
                                <strong>
                                  {titleCase(
                                    movement.reason,
                                  )}
                                </strong>
                              ) : null}

                              {movement.note ? (
                                <p>
                                  {
                                    movement.note
                                  }
                                </p>
                              ) : null}
                            </div>
                          ) : null}

                          {movement.reference_type ||
                          movement.actor_type ? (
                            <div
                              className={
                                styles.movementMeta
                              }
                            >
                              {movement.reference_type ? (
                                <span>
                                  Reference ·{" "}
                                  {titleCase(
                                    movement.reference_type,
                                  )}

                                  {movement.reference_id
                                    ? ` · ${movement.reference_id}`
                                    : ""}
                                </span>
                              ) : null}

                              {movement.actor_type ? (
                                <span>
                                  Actor ·{" "}
                                  {titleCase(
                                    movement.actor_type,
                                  )}
                                </span>
                              ) : null}
                            </div>
                          ) : null}
                        </article>
                      ),
                    )}

                    {hasMoreMovements ? (
                      <button
                        type="button"
                        className={
                          styles.loadMore
                        }
                        onClick={() =>
                          void loadOlderMovements()
                        }
                        disabled={
                          loadingMore
                        }
                      >
                        {loadingMore
                          ? "Loading…"
                          : "Load older movements"}
                      </button>
                    ) : null}
                  </div>
                ) : (
                  <div
                    className={
                      styles.emptyMovements
                    }
                  >
                    <strong>
                      No inventory
                      movements yet
                    </strong>

                    <p>
                      Reservations,
                      commits,
                      releases and
                      authorized
                      adjustments
                      will appear
                      here when they
                      occur.
                    </p>
                  </div>
                )}
              </section>
            </>
          ) : null}
        </div>
      </aside>
    </div>,
    document.body,
  );
}