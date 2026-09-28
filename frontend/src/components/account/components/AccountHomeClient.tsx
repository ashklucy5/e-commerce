// Location: src/components/account/components/AccountHomeClient.tsx
"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { ChangeEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";

import { Icon, type IconName } from "@/components/ui/Icon";
import type {
  AccountAddress,
  AccountCustomer,
  AccountDataResponse,
  AccountOrderHistoryItem,
  AccountOrderHistoryResponse,
  AccountReturnItem,
  AccountReturnsResponse,
  AccountSecurityOverview,
  AccountSupportCase,
  AccountSupportCasesResponse,
  AccountWishlistItem,
  AccountWishlistResponse,
} from "@/lib/api/contracts/account";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/AccountDashboard.module.css";

type DashboardData = {
  customer: AccountCustomer;
  orders: AccountOrderHistoryItem[];
  addresses: AccountAddress[];
  wishlist: AccountWishlistItem[];
  returns: AccountReturnItem[];
  support: AccountSupportCase[];
  security: AccountSecurityOverview | null;
};

type State =
  | { kind: "loading" }
  | { kind: "signed-out" }
  | { kind: "error"; message: string }
  | { kind: "ready"; data: DashboardData };

type ApiError = Error & { status?: number };

type UploadTarget = {
  key: string;
  method: string;
  url: string;
  headers: Record<string, string>;
};

const sidebarItems: Array<{
  href: string;
  label: string;
  icon: IconName;
  count?: (data: DashboardData) => number | undefined;
}> = [
  { href: "/account", label: "Overview", icon: "homeFilled" },
  { href: "/account/orders", label: "Orders", icon: "orders", count: (data) => data.orders.length },
  {
    href: "/account/watchlist",
    label: "Watchlist",
    icon: "bell",
    count: (data) => data.orders.filter((order) => isWaitingForDelivery(order.status)).length,
  },
  { href: "/account/wishlist", label: "Wishlist", icon: "heart", count: (data) => data.wishlist.length },
  { href: "/account/addresses", label: "Addresses", icon: "address", count: (data) => data.addresses.length },
  {
    href: "/account/returns",
    label: "Returns",
    icon: "returns",
    count: (data) => activeReturns(data.returns),
  },
  { href: "/account/settings", label: "Profile & settings", icon: "account" },
  {
    href: "/account/security",
    label: "Security",
    icon: "secureCheckout",
    count: (data) => data.security?.active_session_count ?? data.security?.session_count,
  },
  { href: "/account/settings#notifications", label: "Notifications", icon: "bell" },
  { href: "/account/support", label: "Support", icon: "support" },
];

const quickActions: Array<{ href: string; icon: IconName; label: string }> = [
  { href: "/account/orders", icon: "delivery", label: "Track order" },
  { href: "/account/addresses", icon: "address", label: "Add address" },
  { href: "/account/request", icon: "request", label: "Request product" },
  { href: "/account/security", icon: "secureCheckout", label: "Account security" },
];

function initials(name: string) {
  return (
    name
      .trim()
      .split(/\s+/)
      .slice(0, 2)
      .map((part) => part[0]?.toUpperCase() || "")
      .join("") || "ED"
  );
}

function normalizeStatus(value: string) {
  return value.trim().toLowerCase().replaceAll("-", "_").replaceAll(" ", "_");
}

function humanize(value: string) {
  return value
    .replaceAll("_", " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function isDelivered(status: string) {
  return ["delivered", "completed"].includes(normalizeStatus(status));
}

function isTransit(status: string) {
  return ["shipped", "in_transit", "out_for_delivery", "delivering"].includes(
    normalizeStatus(status),
  );
}

function isCancelled(status: string) {
  return ["cancelled", "canceled", "refunded", "returned", "failed"].includes(
    normalizeStatus(status),
  );
}

function isWaitingForDelivery(status: string) {
  return !isDelivered(status) && !isTransit(status) && !isCancelled(status);
}

function activeReturns(items: AccountReturnItem[]) {
  return items.filter(
    (item) => !["completed", "closed", "rejected", "cancelled", "canceled"].includes(normalizeStatus(item.status)),
  ).length;
}

function formatDate(value?: string) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("en-BD", {
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(date);
}

async function readError(response: Response, fallback: string) {
  try {
    const payload = (await response.json()) as {
      error?: { message?: string } | string;
      message?: string;
    };

    if (typeof payload.error === "string" && payload.error.trim()) return payload.error;
    if (typeof payload.error === "object" && payload.error?.message?.trim()) {
      return payload.error.message.trim();
    }
    return payload.message?.trim() || fallback;
  } catch {
    return fallback;
  }
}

async function getJSON<T>(url: string, fallback: string) {
  const response = await fetch(url, { cache: "no-store" });

  if (!response.ok) {
    const error = new Error(await readError(response, fallback)) as ApiError;
    error.status = response.status;
    throw error;
  }

  return (await response.json()) as T;
}

async function optionalJSON<T>(url: string): Promise<T | null> {
  try {
    return await getJSON<T>(url, "Unable to load this account section.");
  } catch {
    return null;
  }
}

function objectValue(value: unknown): Record<string, unknown> | null {
  return value && typeof value === "object" ? (value as Record<string, unknown>) : null;
}

function resolveUploadTarget(payload: unknown): UploadTarget | null {
  const root = objectValue(payload);
  const data = objectValue(root?.data) ?? root;
  const candidate = objectValue(data?.target) ?? objectValue(data?.upload) ?? data;

  if (!candidate) return null;

  const key = typeof candidate.key === "string" ? candidate.key : "";
  const url = typeof candidate.url === "string" ? candidate.url : "";
  const method = typeof candidate.method === "string" ? candidate.method : "PUT";
  const headerValue = objectValue(candidate.headers);
  const headers: Record<string, string> = {};

  if (headerValue) {
    for (const [name, value] of Object.entries(headerValue)) {
      if (typeof value === "string") headers[name] = value;
    }
  }

  return key && url ? { key, url, method, headers } : null;
}

export function AccountHomeClient() {
  const router = useRouter();
  const [state, setState] = useState<State>({ kind: "loading" });
  const [signingOut, setSigningOut] = useState(false);

  const load = useCallback(async (showLoading = true) => {
    if (showLoading) setState({ kind: "loading" });

    try {
      const me = await getJSON<AccountDataResponse<AccountCustomer>>(
        "/api/storefront/account/me",
        "Unable to load your account.",
      );

      const [orders, addresses, wishlist, returns, support, security] = await Promise.all([
        optionalJSON<AccountOrderHistoryResponse>("/api/storefront/account/orders?page=1&limit=100"),
        optionalJSON<AccountDataResponse<AccountAddress[]>>("/api/storefront/account/addresses"),
        optionalJSON<AccountWishlistResponse>("/api/storefront/account/wishlist?page=1&limit=100"),
        optionalJSON<AccountReturnsResponse>("/api/storefront/account/returns?page=1&limit=100"),
        optionalJSON<AccountSupportCasesResponse>("/api/storefront/account/support"),
        optionalJSON<AccountDataResponse<AccountSecurityOverview>>("/api/storefront/account/security"),
      ]);

      setState({
        kind: "ready",
        data: {
          customer: me.data,
          orders: orders?.data ?? [],
          addresses: addresses?.data ?? [],
          wishlist: wishlist?.data ?? [],
          returns: returns?.data ?? [],
          support: support?.data ?? [],
          security: security?.data ?? null,
        },
      });
    } catch (caught) {
      const error = caught as ApiError;
      if (error.status === 401) {
        router.replace("/account/sign-in?next=%2Faccount");
        return;
      }

      setState({
        kind: "error",
        message: caught instanceof Error ? caught.message : "Unable to load your account.",
      });
    }
  }, [router]);

  useEffect(() => {
    let cancelled = false;

    void Promise.resolve().then(() => {
      if (!cancelled) {
        void load();
      }
    });

    return () => {
      cancelled = true;
    };
  }, [load]);

  async function signOut() {
    if (signingOut) return;
    setSigningOut(true);

    try {
      await fetch("/api/storefront/auth/logout", { method: "POST" });
    } finally {
      router.replace("/account/sign-in");
    }
  }

  if (state.kind === "loading") {
    return (
      <main className={styles.page} aria-busy="true">
        <div className={styles.shell}>
          <div className={styles.loadingLayout}>
            <div className={styles.skeletonSidebar} />
            <div>
              <div className={styles.skeletonHero} />
              <div className={styles.skeletonGrid}><div /><div /><div /><div /></div>
            </div>
          </div>
        </div>
      </main>
    );
  }

  if (state.kind === "signed-out") {
    return null;
  }

  if (state.kind === "error") {
    return (
      <main className={styles.page}>
        <section className={styles.stateCard} role="alert">
          <span className={styles.stateIcon}><Icon name="support" size={24} /></span>
          <h1>Account is temporarily unavailable.</h1>
          <p>{state.message}</p>
          <button className={styles.secondaryAction} type="button" onClick={() => void load()}>Try again</button>
        </section>
      </main>
    );
  }

  return (
    <AccountDashboard
      data={state.data}
      signingOut={signingOut}
      onSignOut={signOut}
      onRefresh={() => load(false)}
    />
  );
}

function AccountDashboard({
  data,
  signingOut,
  onSignOut,
  onRefresh,
}: {
  data: DashboardData;
  signingOut: boolean;
  onSignOut: () => Promise<void>;
  onRefresh: () => Promise<void>;
}) {
  const { customer, orders, wishlist, returns, support, security } = data;

  const waitingOrders = useMemo(
    () => orders.filter((order) => isWaitingForDelivery(order.status)),
    [orders],
  );
  const deliveredCount = useMemo(
    () => orders.filter((order) => isDelivered(order.status)).length,
    [orders],
  );
  const activeReturnCount = useMemo(() => activeReturns(returns), [returns]);
  const recentOrders = orders.slice(0, 4);
  const activeSupport = support.filter(
    (item) => !["resolved", "closed"].includes(normalizeStatus(item.status)),
  ).length;
  const sessionCount = security?.active_session_count ?? security?.session_count;

  return (
    <main className={styles.page}>
      <div className={styles.liquidBackdrop} aria-hidden="true">
        <span className={styles.liquidOne} />
        <span className={styles.liquidTwo} />
        <span className={styles.liquidThree} />
      </div>

      <div className={styles.shell}>
        <div className={styles.accountLayout}>
          <aside className={`${styles.glassSurface} ${styles.sidebar}`} aria-label="Account navigation">
            <div className={styles.sidebarProfile}>
              <AvatarEditor customer={customer} onChanged={onRefresh} />
              <div className={styles.sidebarIdentity}>
                <strong>{customer.full_name}</strong>
                <span>{customer.phone}</span>
              </div>
            </div>

            <nav className={styles.sidebarNav}>
              {sidebarItems.map((item) => {
                const count = item.count?.(data);
                const active = item.href === "/account";
                return (
                  <Link
                    key={item.href + item.label}
                    href={item.href}
                    className={active ? styles.sidebarActive : undefined}
                    aria-current={active ? "page" : undefined}
                  >
                    <span className={styles.sidebarLabel}>
                      <Icon name={item.icon} size={17} />
                      {item.label}
                    </span>
                    {count !== undefined ? <span className={styles.sidebarCount}>{count}</span> : null}
                  </Link>
                );
              })}
            </nav>

            <button
              type="button"
              className={styles.logoutButton}
              onClick={() => void onSignOut()}
              disabled={signingOut}
            >
              <Icon name="arrowLeft" size={16} />
              {signingOut ? "Signing out…" : "Sign out"}
            </button>
          </aside>

          <section className={styles.mainColumn}>
            <div className={`${styles.glassSurface} ${styles.mobileProfile}`}>
              <AvatarEditor customer={customer} onChanged={onRefresh} compact />
              <div>
                <span className={styles.eyebrow}>Your account</span>
                <h1>{customer.full_name}</h1>
                <p>{customer.phone}</p>
              </div>
              <Link href="/account/settings" aria-label="Open profile settings">
                <Icon name="chevronRight" size={17} />
              </Link>
            </div>

            <section className={`${styles.glassSurface} ${styles.heroPanel}`}>
              <div className={styles.heroCopy}>
                <span className={styles.eyebrow}>Account overview</span>
                <h1>Welcome back, {customer.full_name.split(/\s+/)[0] || "Customer"}.</h1>
                <p>Orders, saved items, delivery progress and account controls in one place.</p>
              </div>
              <div className={styles.heroLiquid} aria-hidden="true" />
            </section>

            <section className={styles.summaryGrid} aria-label="Account summary">
              <SummaryCard icon="orders" label="Orders" value={orders.length} detail="Total orders" href="/account/orders" />
              <SummaryCard icon="bell" label="Watchlist" value={waitingOrders.length} detail="Before delivery" href="/account/watchlist" />
              <SummaryCard icon="heart" label="Wishlist" value={wishlist.length} detail="Saved items" href="/account/wishlist" />
              <SummaryCard icon="returns" label="Returns" value={activeReturnCount} detail="Active requests" href="/account/returns" />
            </section>

            <section className={`${styles.glassSurface} ${styles.quickActions}`}>
              <div className={styles.sectionHeading}>
                <div><span className={styles.eyebrow}>Shortcuts</span><h2>Quick actions</h2></div>
              </div>
              <div className={styles.quickGrid}>
                {quickActions.map((item) => (
                  <Link key={item.href + item.label} href={item.href}>
                    <span className={styles.quickIcon}><Icon name={item.icon} size={18} /></span>
                    <span>{item.label}</span>
                    <Icon name="chevronRight" size={14} />
                  </Link>
                ))}
              </div>
            </section>

            <section className={`${styles.glassSurface} ${styles.watchlistPanel}`}>
              <div className={styles.sectionHeading}>
                <div>
                  <span className={styles.eyebrow}>Before delivery starts</span>
                  <h2>Watchlist</h2>
                  <p>Orders stay here while they are waiting to enter the delivery stage.</p>
                </div>
                <Link href="/account/watchlist">View all</Link>
              </div>

              {waitingOrders.length ? (
                <div className={styles.watchlistRows}>
                  {waitingOrders.slice(0, 3).map((order) => (
                    <WatchlistOrder key={order.id} order={order} />
                  ))}
                </div>
              ) : (
                <div className={styles.watchlistEmpty}>
                  <span><Icon name="delivery" size={22} /></span>
                  <div>
                    <strong>Nothing is waiting for delivery.</strong>
                    <p>New orders will appear here until shipping or delivery begins.</p>
                  </div>
                  <Link href="/">Continue shopping</Link>
                </div>
              )}
            </section>

            <section className={`${styles.glassSurface} ${styles.ordersPanel}`}>
              <div className={styles.sectionHeading}>
                <div><span className={styles.eyebrow}>Purchase history</span><h2>Recent orders</h2></div>
                <Link href="/account/orders">View all orders</Link>
              </div>

              {recentOrders.length ? (
                <div className={styles.orderTable}>
                  <div className={styles.orderTableHead} aria-hidden="true">
                    <span>Order</span><span>Date</span><span>Status</span><span>Total</span><span />
                  </div>
                  {recentOrders.map((order) => <RecentOrder key={order.id} order={order} />)}
                </div>
              ) : (
                <div className={styles.orderEmpty}>
                  <strong>No orders yet.</strong>
                  <Link href="/">Browse products</Link>
                </div>
              )}
            </section>

            <section className={styles.bottomUtilityGrid}>
              <UtilityCard icon="delivery" title="Delivered" detail={`${deliveredCount} completed orders`} href="/account/orders?status=delivered" />
              <UtilityCard icon="support" title="Support" detail={`${activeSupport} open cases`} href="/account/support" />
              <UtilityCard icon="secureCheckout" title="Security" detail={sessionCount !== undefined ? `${sessionCount} active sessions` : "Password & sessions"} href="/account/security" />
              <UtilityCard icon="settings" title="Settings" detail="Profile & preferences" href="/account/settings" />
            </section>
          </section>
        </div>
      </div>
    </main>
  );
}

function AvatarEditor({
  customer,
  onChanged,
  compact = false,
}: {
  customer: AccountCustomer;
  onChanged: () => Promise<void>;
  compact?: boolean;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function upload(file: File) {
    if (busy) return;
    setBusy(true);
    setMessage("");

    try {
      const targetResponse = await fetch("/api/storefront/account/avatar/upload-target", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          filename: file.name,
          content_type: file.type || "application/octet-stream",
          content_length: file.size,
        }),
      });

      if (!targetResponse.ok) {
        throw new Error(await readError(targetResponse, "Unable to prepare avatar upload."));
      }

      const target = resolveUploadTarget(await targetResponse.json());
      if (!target) throw new Error("The avatar upload target was incomplete.");

      const uploadResponse = await fetch(target.url, {
        method: target.method || "PUT",
        headers: target.headers,
        body: file,
      });

      if (!uploadResponse.ok) {
        throw new Error("The avatar image could not be uploaded to storage.");
      }

      const completeResponse = await fetch("/api/storefront/account/avatar/complete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ storage_key: target.key }),
      });

      if (!completeResponse.ok) {
        throw new Error(await readError(completeResponse, "Unable to save your new avatar."));
      }

      setMessage("Photo updated");
      await onChanged();
      window.dispatchEvent(new Event("customer-auth-changed"));
    } catch (caught) {
      setMessage(caught instanceof Error ? caught.message : "Unable to update avatar.");
    } finally {
      setBusy(false);
      if (inputRef.current) inputRef.current.value = "";
    }
  }

  async function removeAvatar() {
    if (busy || !customer.avatar_url) return;
    setBusy(true);
    setMessage("");

    try {
      const response = await fetch("/api/storefront/account/avatar", { method: "DELETE" });
      if (!response.ok) throw new Error(await readError(response, "Unable to remove avatar."));
      setMessage("Photo removed");
      await onChanged();
      window.dispatchEvent(new Event("customer-auth-changed"));
    } catch (caught) {
      setMessage(caught instanceof Error ? caught.message : "Unable to remove avatar.");
    } finally {
      setBusy(false);
    }
  }

  function onFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;

    if (!["image/jpeg", "image/png", "image/webp"].includes(file.type)) {
      setMessage("Choose a JPEG, PNG or WebP image.");
      event.target.value = "";
      return;
    }

    if (file.size <= 0 || file.size > 5 * 1024 * 1024) {
      setMessage("Profile photos must be 5 MB or smaller.");
      event.target.value = "";
      return;
    }

    void upload(file);
  }

  return (
    <div className={`${styles.avatarEditor} ${compact ? styles.avatarCompact : ""}`}>
      <div className={styles.avatarFrame}>
        {customer.avatar_url ? (
          // Backend-issued avatar URLs may be short-lived signed URLs.
          // eslint-disable-next-line @next/next/no-img-element
          <img src={customer.avatar_url} alt={`${customer.full_name} avatar`} className={styles.avatarImage} />
        ) : (
          <span className={styles.avatarFallback}>{initials(customer.full_name)}</span>
        )}

        <button
          type="button"
          className={styles.avatarChange}
          aria-label="Change profile photo"
          onClick={() => inputRef.current?.click()}
          disabled={busy}
        >
          {busy ? <span className={styles.avatarSpinner} aria-hidden="true" /> : <Icon name="plus" size={14} />}
        </button>
      </div>

      <input
        ref={inputRef}
        className={styles.avatarInput}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        onChange={onFileChange}
        tabIndex={-1}
      />

      {!compact ? (
        <div className={styles.avatarActions}>
          <button type="button" onClick={() => inputRef.current?.click()} disabled={busy}>
            {busy ? "Updating…" : "Change photo"}
          </button>
          {customer.avatar_url ? (
            <button type="button" onClick={() => void removeAvatar()} disabled={busy}>Remove</button>
          ) : null}
        </div>
      ) : null}

      {message ? <span className={styles.avatarMessage} role="status">{message}</span> : null}
    </div>
  );
}

function SummaryCard({
  icon,
  label,
  value,
  detail,
  href,
}: {
  icon: IconName;
  label: string;
  value: number;
  detail: string;
  href: string;
}) {
  return (
    <Link className={`${styles.glassSurface} ${styles.summaryCard}`} href={href}>
      <span className={styles.summaryIcon}><Icon name={icon} size={19} /></span>
      <div>
        <span>{label}</span>
        <strong>{value.toLocaleString()}</strong>
        <small>{detail}</small>
      </div>
      <span className={styles.summaryLink}>View <Icon name="chevronRight" size={11} /></span>
    </Link>
  );
}

function WatchlistOrder({ order }: { order: AccountOrderHistoryItem }) {
  return (
    <Link href={`/account/orders/${order.id}`} className={styles.watchlistRow}>
      <span className={styles.watchlistThumb}><Icon name="orders" size={20} /></span>
      <span className={styles.watchlistMain}>
        <strong>{order.first_product_name || `Order #${order.order_number}`}</strong>
        <small>Order #{order.order_number} · {order.quantity_total} units</small>
        <span className={styles.waitingBadge}><Icon name="bell" size={11} /> {humanize(order.status)}</span>
      </span>
      <span className={styles.watchlistPrice}>{formatMoney(order.total_amount, order.currency)}</span>
      <Icon name="chevronRight" size={14} />
    </Link>
  );
}

function RecentOrder({ order }: { order: AccountOrderHistoryItem }) {
  const delivered = isDelivered(order.status);
  const transit = isTransit(order.status);
  return (
    <Link className={styles.orderRow} href={`/account/orders/${order.id}`}>
      <span><strong>#{order.order_number}</strong><small>{order.first_product_name || `${order.item_count} product lines`}</small></span>
      <span>{formatDate(order.created_at)}</span>
      <span className={`${styles.statusBadge} ${delivered ? styles.statusDelivered : transit ? styles.statusTransit : styles.statusWaiting}`}>{humanize(order.status)}</span>
      <strong className={styles.orderTotal}>{formatMoney(order.total_amount, order.currency)}</strong>
      <Icon name="chevronRight" size={14} />
    </Link>
  );
}

function UtilityCard({
  icon,
  title,
  detail,
  href,
}: {
  icon: IconName;
  title: string;
  detail: string;
  href: string;
}) {
  return (
    <Link className={`${styles.glassSurface} ${styles.utilityCard}`} href={href}>
      <span><Icon name={icon} size={18} /></span>
      <div><strong>{title}</strong><small>{detail}</small></div>
      <Icon name="chevronRight" size={14} />
    </Link>
  );
}
