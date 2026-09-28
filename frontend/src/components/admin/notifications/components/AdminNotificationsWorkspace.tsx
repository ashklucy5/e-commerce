"use client";

import { useRouter } from "next/navigation";
import {
  useCallback,
  useEffect,
  useState,
} from "react";

import AdminPageHeader from "@/components/admin/layout/components/AdminPageHeader";
import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import { Icon } from "@/components/ui/Icon";
import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";
import type {
  AdminNotification,
  AdminNotificationListResponse,
  AdminNotificationSummaryResponse,
} from "@/lib/admin/notification-types";

import styles from "../css/AdminNotifications.module.css";

const PAGE_SIZE = 40;
const LIVE_SYNC_MS = 20_000;

type Props = {
  portal: string;
};

function titleCase(value: string): string {
  return value
    .replace(/[._-]+/g, " ")
    .replace(
      /\b\w/g,
      (character) =>
        character.toUpperCase(),
    );
}

function actionHref(
  portal: string,
  value?: string,
): string {
  if (
    !value ||
    !value.startsWith("/") ||
    value.startsWith("//")
  ) {
    return `/${portal}/notifications`;
  }

  if (value === "/admin") {
    return `/${portal}`;
  }

  if (
    value.startsWith(
      "/admin/",
    )
  ) {
    return `/${portal}${value.slice(
      "/admin".length,
    )}`;
  }

  return value;
}

function formatDateTime(
  value: string,
): string {
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

function errorMessage(
  value: unknown,
): string {
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

  return "Notifications could not be loaded.";
}

export default function AdminNotificationsWorkspace({
  portal,
}: Props) {
  const router =
    useRouter();

  const principal =
    useAdminSession();

  const permissions =
    principal.staff.permissions;

  const isSuperAdmin =
    principal.staff.roles.includes(
      "admin_superuser",
    );

  const canRead =
    isSuperAdmin ||
    permissions.includes(
      "admin.notification.read",
    );

  const [
    items,
    setItems,
  ] =
    useState<
      AdminNotification[]
    >([]);

  const [
    unread,
    setUnread,
  ] =
    useState(0);

  const [
    state,
    setState,
  ] =
    useState<
      | "loading"
      | "ready"
      | "error"
    >("loading");

  const [
    error,
    setError,
  ] =
    useState("");

  const [
    hasMore,
    setHasMore,
  ] =
    useState(false);

  const [
    loadingMore,
    setLoadingMore,
  ] =
    useState(false);

  const [
    markingAll,
    setMarkingAll,
  ] =
    useState(false);

  const [
    refreshing,
    setRefreshing,
  ] =
    useState(false);

  const loadInitial =
    useCallback(
      async () => {
        const [
          list,
          summary,
        ] =
          await Promise.all([
            adminFetch<AdminNotificationListResponse>(
              `/notifications?limit=${PAGE_SIZE}&offset=0`,
            ),

            adminFetch<AdminNotificationSummaryResponse>(
              "/notifications/summary",
            ),
          ]);

        const next =
          Array.isArray(
            list.data,
          )
            ? list.data
            : [];

        setItems(next);

        setUnread(
          Math.max(
            0,
            Number(
              summary.data
                ?.unread_count ??
                0,
            ),
          ),
        );

        setHasMore(
          next.length ===
            PAGE_SIZE,
        );

        setError("");
        setState("ready");
      },
      [],
    );

  const refreshHead =
    useCallback(
      async () => {
        const [
          list,
          summary,
        ] =
          await Promise.all([
            adminFetch<AdminNotificationListResponse>(
              `/notifications?limit=${PAGE_SIZE}&offset=0`,
            ),

            adminFetch<AdminNotificationSummaryResponse>(
              "/notifications/summary",
            ),
          ]);

        const next =
          Array.isArray(
            list.data,
          )
            ? list.data
            : [];

        setItems(
          (current) => {
            if (
              current.length <=
              PAGE_SIZE
            ) {
              return next;
            }

            const headIDs =
              new Set(
                next.map(
                  (item) =>
                    item.id,
                ),
              );

            const older =
              current.filter(
                (item) =>
                  !headIDs.has(
                    item.id,
                  ),
              );

            return [
              ...next,
              ...older,
            ];
          },
        );

        setUnread(
          Math.max(
            0,
            Number(
              summary.data
                ?.unread_count ??
                0,
            ),
          ),
        );

        setError("");
      },
      [],
    );

  useEffect(() => {
    if (!canRead) {
      setState("ready");
      return;
    }

    let active = true;

    void loadInitial().catch(
      (caught) => {
        if (!active) {
          return;
        }

        setError(
          errorMessage(
            caught,
          ),
        );

        setState(
          "error",
        );
      },
    );

    return () => {
      active = false;
    };
  }, [
    canRead,
    loadInitial,
  ]);

  useEffect(() => {
    if (
      !canRead ||
      state !== "ready"
    ) {
      return;
    }

    const timer =
      window.setInterval(
        () => {
          if (
            document.visibilityState !==
            "visible"
          ) {
            return;
          }

          void refreshHead().catch(
            () => undefined,
          );
        },
        LIVE_SYNC_MS,
      );

    return () =>
      window.clearInterval(
        timer,
      );
  }, [
    canRead,
    refreshHead,
    state,
  ]);

  async function manualRefresh() {
    if (refreshing) {
      return;
    }

    setRefreshing(true);
    setError("");

    try {
      await refreshHead();
    } catch (caught) {
      setError(
        errorMessage(
          caught,
        ),
      );
    } finally {
      setRefreshing(false);
    }
  }

  async function openItem(
    item: AdminNotification,
  ) {
    if (!item.read_at) {
      setItems(
        (current) =>
          current.map(
            (entry) =>
              entry.id ===
              item.id
                ? {
                    ...entry,

                    read_at:
                      new Date().toISOString(),
                  }
                : entry,
          ),
      );

      setUnread(
        (current) =>
          Math.max(
            0,
            current - 1,
          ),
      );

      try {
        await adminFetch(
          `/notifications/${encodeURIComponent(
            item.id,
          )}/read`,
          {
            method:
              "POST",
          },
        );
      } catch {
        /*
         * Navigation remains useful
         * even if read-state syncing
         * temporarily fails.
         *
         * The next summary refresh
         * reconciles server state.
         */
      }
    }

    router.push(
      actionHref(
        portal,
        item.action_url,
      ),
    );
  }

  async function markAllRead() {
    if (
      markingAll ||
      unread === 0
    ) {
      return;
    }

    setMarkingAll(true);
    setError("");

    try {
      await adminFetch(
        "/notifications/read-all",
        {
          method:
            "POST",
        },
      );

      const now =
        new Date().toISOString();

      setItems(
        (current) =>
          current.map(
            (item) => ({
              ...item,

              read_at:
                item.read_at ??
                now,
            }),
          ),
      );

      setUnread(0);
    } catch (caught) {
      setError(
        errorMessage(
          caught,
        ),
      );
    } finally {
      setMarkingAll(false);
    }
  }

  async function loadMore() {
    if (
      loadingMore ||
      !hasMore
    ) {
      return;
    }

    setLoadingMore(true);
    setError("");

    try {
      const response =
        await adminFetch<AdminNotificationListResponse>(
          `/notifications?limit=${PAGE_SIZE}&offset=${items.length}`,
        );

      const next =
        Array.isArray(
          response.data,
        )
          ? response.data
          : [];

      setItems(
        (current) => {
          const existing =
            new Set(
              current.map(
                (item) =>
                  item.id,
              ),
            );

          const unique =
            next.filter(
              (item) =>
                !existing.has(
                  item.id,
                ),
            );

          return [
            ...current,
            ...unique,
          ];
        },
      );

      setHasMore(
        next.length ===
          PAGE_SIZE,
      );
    } catch (caught) {
      setError(
        errorMessage(
          caught,
        ),
      );
    } finally {
      setLoadingMore(false);
    }
  }

  if (!canRead) {
    return (
      <div
        className={
          styles.page
        }
      >
        <AdminPageHeader
          eyebrow="Operations alerts"
          title="Notifications"
          description="Your staff role does not include permission to view operational notifications."
        />

        <section
          className={
            styles.permissionPanel
          }
        >
          <Icon
            name="bell"
            size={22}
          />

          <strong>
            Notification access
            is restricted
          </strong>

          <p>
            This workspace
            requires the{" "}
            <code>
              admin.notification.read
            </code>{" "}
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
        eyebrow="Operations alerts"
        title="Notifications"
        description="Orders, sourcing, inventory, support, delivery, security and system events targeted to your staff permissions."
        actions={
          <button
            type="button"
            className={
              styles.refreshButton
            }
            onClick={() =>
              void manualRefresh()
            }
            disabled={
              refreshing
            }
          >
            <span
              className={
                refreshing
                  ? styles.spinning
                  : ""
              }
              aria-hidden="true"
            >
              ↻
            </span>

            {refreshing
              ? "Refreshing…"
              : "Refresh"}
          </button>
        }
      />

      <section
        className={
          styles.summaryGrid
        }
        aria-label="Notification summary"
      >
        <article
          className={`${styles.summaryCard} ${styles.summaryPrimary}`}
        >
          <span>
            Unread
          </span>

          <strong>
            {state ===
            "loading"
              ? "—"
              : unread}
          </strong>

          <small>
            Operational events
            still needing your
            attention
          </small>
        </article>

        <article
          className={
            styles.summaryCard
          }
        >
          <span>
            Loaded activity
          </span>

          <strong>
            {state ===
            "loading"
              ? "—"
              : items.length}
          </strong>

          <small>
            Newest targeted
            notifications in
            this workspace
          </small>
        </article>
      </section>

      <section
        className={
          styles.panel
        }
        aria-busy={
          state ===
          "loading"
        }
      >
        <div
          className={
            styles.toolbar
          }
        >
          <div>
            <strong>
              Operational
              activity
            </strong>

            <span>
              Newest events
              first
            </span>
          </div>

          <button
            type="button"
            onClick={() =>
              void markAllRead()
            }
            disabled={
              markingAll ||
              unread === 0
            }
          >
            {markingAll
              ? "Updating…"
              : "Mark all as read"}
          </button>
        </div>

        {error ? (
          <div
            className={
              styles.error
            }
            role="alert"
          >
            <strong>
              Notification
              update failed.
            </strong>

            <span>
              {error}
            </span>
          </div>
        ) : null}

        {state ===
        "loading" ? (
          <div
            className={
              styles.skeleton
            }
            aria-label="Loading notifications"
          >
            <span />
            <span />
            <span />
            <span />
          </div>
        ) : state ===
          "error" ? (
          <div
            className={
              styles.empty
            }
          >
            <Icon
              name="bell"
              size={22}
            />

            <strong>
              Notification
              center unavailable
            </strong>

            <span>
              The operational
              inbox could not
              be loaded.
            </span>

            <button
              type="button"
              onClick={() => {
                setState(
                  "loading",
                );

                setError("");

                void loadInitial().catch(
                  (
                    caught,
                  ) => {
                    setError(
                      errorMessage(
                        caught,
                      ),
                    );

                    setState(
                      "error",
                    );
                  },
                );
              }}
            >
              Try again
            </button>
          </div>
        ) : items.length ===
          0 ? (
          <div
            className={
              styles.empty
            }
          >
            <Icon
              name="bell"
              size={22}
            />

            <strong>
              No operational
              notifications
            </strong>

            <span>
              New orders,
              sourcing requests,
              stock alerts,
              support activity
              and other targeted
              events will appear
              here.
            </span>
          </div>
        ) : (
          <div
            className={
              styles.list
            }
          >
            {items.map(
              (item) => (
                <button
                  type="button"
                  key={
                    item.id
                  }
                  className={[
                    styles.item,

                    item.read_at
                      ? styles.read
                      : styles.unread,
                  ]
                    .filter(
                      Boolean,
                    )
                    .join(" ")}
                  onClick={() =>
                    void openItem(
                      item,
                    )
                  }
                >
                  <span
                    className={[
                      styles.priority,

                      styles[
                        `priority_${item.priority}`
                      ] ?? "",
                    ]
                      .filter(
                        Boolean,
                      )
                      .join(" ")}
                    aria-hidden="true"
                  />

                  <span
                    className={
                      styles.body
                    }
                  >
                    <span
                      className={
                        styles.topline
                      }
                    >
                      <span>
                        {titleCase(
                          item.category,
                        )}
                      </span>

                      <time
                        dateTime={
                          item.created_at
                        }
                      >
                        {formatDateTime(
                          item.created_at,
                        )}
                      </time>
                    </span>

                    <strong>
                      {
                        item.title
                      }
                    </strong>

                    <span
                      className={
                        styles.message
                      }
                    >
                      {
                        item.message
                      }
                    </span>

                    <span
                      className={
                        styles.meta
                      }
                    >
                      {titleCase(
                        item.event_type,
                      )}

                      {item.entity_type
                        ? ` · ${titleCase(
                            item.entity_type,
                          )}`
                        : ""}
                    </span>
                  </span>

                  <span
                    className={
                      styles.chevron
                    }
                  >
                    <Icon
                      name="chevronRight"
                      size={14}
                    />
                  </span>
                </button>
              ),
            )}
          </div>
        )}

        {state ===
          "ready" &&
        hasMore ? (
          <button
            className={
              styles.loadMore
            }
            type="button"
            onClick={() =>
              void loadMore()
            }
            disabled={
              loadingMore
            }
          >
            {loadingMore
              ? "Loading…"
              : "Load more notifications"}
          </button>
        ) : null}
      </section>
    </div>
  );
}