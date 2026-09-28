"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import { Icon } from "@/components/ui/Icon";
import { adminFetch } from "@/lib/admin/api";
import type { AdminPrincipal } from "@/lib/admin/types";
import type {
  AdminNotification,
  AdminNotificationListResponse,
  AdminNotificationSummaryResponse,
} from "@/lib/admin/notification-types";

import styles from "../css/AdminNotificationCenter.module.css";

const POLL_MS = 20_000;

function actionHref(
  portal: string,
  value?: string,
) {
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

  if (value.startsWith("/admin/")) {
    return `/${portal}${value.slice("/admin".length)}`;
  }

  return value;
}

function relativeTime(value: string) {
  const time =
    new Date(value).getTime();

  if (!Number.isFinite(time)) {
    return "";
  }

  const minutes =
    Math.floor(
      Math.max(
        0,
        Date.now() - time,
      ) / 60_000,
    );

  if (minutes < 1) {
    return "Now";
  }

  if (minutes < 60) {
    return `${minutes}m`;
  }

  const hours =
    Math.floor(
      minutes / 60,
    );

  if (hours < 24) {
    return `${hours}h`;
  }

  const days =
    Math.floor(
      hours / 24,
    );

  return days < 7
    ? `${days}d`
    : new Date(value).toLocaleDateString();
}

export default function AdminNotificationCenter({
  principal,
  portal,
}: {
  principal: AdminPrincipal;
  portal: string;
}) {
  const router = useRouter();
  const pathname = usePathname();

  const rootRef =
    useRef<HTMLDivElement | null>(
      null,
    );

  const canRead =
    principal.staff.permissions.includes(
      "admin.notification.read",
    );

  const [open, setOpen] =
    useState(false);

  const [items, setItems] =
    useState<AdminNotification[]>([]);

  const [unread, setUnread] =
    useState(0);

  const [busyAll, setBusyAll] =
    useState(false);

  const [error, setError] =
    useState("");

  const loadSummary =
    useCallback(
      async () => {
        if (!canRead) {
          return;
        }

        try {
          const payload =
            await adminFetch<AdminNotificationSummaryResponse>(
              "/notifications/summary",
            );

          setUnread(
            Math.max(
              0,
              Number(
                payload.data?.unread_count ??
                  0,
              ),
            ),
          );
        } catch {
          /*
           * Notification refresh must never
           * make the Admin topbar unusable.
           */
        }
      },
      [canRead],
    );

  const loadPreview =
    useCallback(
      async () => {
        if (!canRead) {
          return;
        }

        try {
          setError("");

          const payload =
            await adminFetch<AdminNotificationListResponse>(
              "/notifications?limit=6&offset=0",
            );

          setItems(
            Array.isArray(
              payload.data,
            )
              ? payload.data
              : [],
          );
        } catch (caught) {
          setError(
            caught instanceof Error
              ? caught.message
              : "Unable to load notifications.",
          );
        }
      },
      [canRead],
    );

  useEffect(() => {
    if (!canRead) {
      return;
    }

    void loadSummary();

    const timer =
      window.setInterval(
        () => {
          if (
            document.visibilityState ===
            "visible"
          ) {
            void loadSummary();
          }
        },
        POLL_MS,
      );

    return () =>
      window.clearInterval(
        timer,
      );
  }, [
    canRead,
    loadSummary,
  ]);

  useEffect(() => {
    setOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (!open) {
      return;
    }

    void loadPreview();
    void loadSummary();

    function pointerDown(
      event: PointerEvent,
    ) {
      if (
        rootRef.current &&
        !rootRef.current.contains(
          event.target as Node,
        )
      ) {
        setOpen(false);
      }
    }

    function keyDown(
      event: KeyboardEvent,
    ) {
      if (
        event.key === "Escape"
      ) {
        setOpen(false);
      }
    }

    document.addEventListener(
      "pointerdown",
      pointerDown,
    );

    document.addEventListener(
      "keydown",
      keyDown,
    );

    return () => {
      document.removeEventListener(
        "pointerdown",
        pointerDown,
      );

      document.removeEventListener(
        "keydown",
        keyDown,
      );
    };
  }, [
    loadPreview,
    loadSummary,
    open,
  ]);

  if (!canRead) {
    return null;
  }

  function openItem(
    item: AdminNotification,
  ) {
    if (!item.read_at) {
      setItems(
        (current) =>
          current.map(
            (entry) =>
              entry.id === item.id
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

      void adminFetch(
        `/notifications/${encodeURIComponent(
          item.id,
        )}/read`,
        {
          method: "POST",
        },
      )
        .then(loadSummary)
        .catch(
          () => undefined,
        );
    }

    setOpen(false);

    router.push(
      actionHref(
        portal,
        item.action_url,
      ),
    );
  }

  async function markAllRead() {
    if (
      busyAll ||
      unread === 0
    ) {
      return;
    }

    setBusyAll(true);
    setError("");

    try {
      await adminFetch(
        "/notifications/read-all",
        {
          method: "POST",
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
        caught instanceof Error
          ? caught.message
          : "Unable to update notifications.",
      );
    } finally {
      setBusyAll(false);
    }
  }

  return (
    <div
      className={styles.root}
      ref={rootRef}
    >
      <button
        type="button"
        className={styles.trigger}
        aria-label={
          unread > 0
            ? `Operations notifications, ${unread} unread`
            : "Operations notifications"
        }
        aria-expanded={open}
        aria-haspopup="dialog"
        onClick={() =>
          setOpen(
            (current) =>
              !current,
          )
        }
      >
        <Icon
          name="bell"
          size={17}
        />

        {unread > 0 ? (
          <span
            className={
              styles.badge
            }
          >
            {unread > 99
              ? "99+"
              : unread}
          </span>
        ) : null}
      </button>

      {open ? (
        <section
          className={
            styles.panel
          }
          role="dialog"
          aria-label="Operations notifications"
        >
          <div
            className={
              styles.header
            }
          >
            <div>
              <span>
                Operations inbox
              </span>

              <strong>
                {unread > 0
                  ? `${unread} unread`
                  : "All caught up"}
              </strong>
            </div>

            <button
              type="button"
              onClick={() =>
                void markAllRead()
              }
              disabled={
                busyAll ||
                unread === 0
              }
            >
              {busyAll
                ? "Updating…"
                : "Mark all read"}
            </button>
          </div>

          {error ? (
            <p
              className={
                styles.error
              }
            >
              {error}
            </p>
          ) : null}

          <div
            className={
              styles.list
            }
          >
            {items.length ? (
              items.map(
                (item) => (
                  <button
                    type="button"
                    key={item.id}
                    className={`${styles.item} ${
                      item.read_at
                        ? styles.read
                        : styles.unread
                    }`}
                    onClick={() =>
                      openItem(
                        item,
                      )
                    }
                  >
                    <span
                      className={`${styles.priority} ${
                        styles[
                          `priority_${item.priority}`
                        ] ?? ""
                      }`}
                      aria-hidden="true"
                    />

                    <span
                      className={
                        styles.copy
                      }
                    >
                      <span
                        className={
                          styles.topline
                        }
                      >
                        <span>
                          {
                            item.category
                          }
                        </span>

                        <time>
                          {relativeTime(
                            item.created_at,
                          )}
                        </time>
                      </span>

                      <strong>
                        {item.title}
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
                    </span>
                  </button>
                ),
              )
            ) : (
              <div
                className={
                  styles.empty
                }
              >
                <Icon
                  name="bell"
                  size={20}
                />

                <strong>
                  No operational alerts
                </strong>

                <span>
                  New orders,
                  sourcing,
                  support and
                  stock events
                  will appear
                  here.
                </span>
              </div>
            )}
          </div>

          <Link
            className={
              styles.viewAll
            }
            href={`/${portal}/notifications`}
            onClick={() =>
              setOpen(false)
            }
          >
            Open notification center

            <Icon
              name="chevronRight"
              size={13}
            />
          </Link>
        </section>
      ) : null}
    </div>
  );
}