// Location: src/components/account/components/ReturnsClient.tsx
"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { Icon } from "@/components/ui/Icon";
import type { AccountReturnItem, AccountReturnsResponse } from "@/lib/api/contracts/account";
import { formatMoney } from "@/lib/money/format";

import { AccountSectionHeader } from "./AccountSectionHeader";
import styles from "../css/AccountManagement.module.css";

function humanize(value: string) {
  return value.replaceAll("_", " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function dateLabel(value?: string) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(date);
}

export function ReturnsClient() {
  const [items, setItems] = useState<AccountReturnItem[]>([]);
  const [state, setState] = useState<"loading" | "ready" | "signed-out" | "error">("loading");

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const response = await fetch("/api/storefront/account/returns?page=1&limit=100", { cache: "no-store" });
        if (cancelled) return;
        if (response.status === 401) { setState("signed-out"); return; }
        if (!response.ok) { setState("error"); return; }
        const payload = (await response.json()) as AccountReturnsResponse;
        setItems(payload.data ?? []);
        setState("ready");
      } catch {
        if (!cancelled) setState("error");
      }
    }
    void load();
    return () => { cancelled = true; };
  }, []);

  if (state === "signed-out") {
    return <main className={styles.page}><section className={styles.state}><Icon name="returns" size={24} /><h1>Sign in to view returns.</h1><Link className={styles.primary} href="/account/sign-in?next=%2Faccount%2Freturns">Sign in</Link></section></main>;
  }

  return (
    <main className={styles.page}>
      <div className={styles.shell}>
        <AccountSectionHeader eyebrow="Activity" title="Returns and refunds." description="Follow return requests and refund-related activity from the same account that placed the order." />
        <section className={styles.card}>
          <div className={styles.cardHeader}><div><h2>Return activity</h2><p>{items.length.toLocaleString()} requests</p></div></div>
          {state === "loading" ? <p className={styles.message}>Loading return activity…</p> : null}
          {state === "error" ? <p className={`${styles.message} ${styles.error}`} role="alert">Returns are temporarily unavailable.</p> : null}
          {state === "ready" && items.length === 0 ? <div className={styles.empty}>No return requests yet.</div> : null}
          <div className={styles.list}>
            {items.map((item) => (
              <article className={styles.row} key={item.id}>
                <div className={styles.rowMain}>
                  <strong>{item.return_number ? `#${item.return_number}` : "Return request"}</strong>
                  <span>Order {item.order_number ? `#${item.order_number}` : item.order_id}</span>
                  {item.reason ? <p>{item.reason}</p> : null}
                </div>
                <div className={styles.rowMeta}>
                  <span className={styles.statusBadge}>{humanize(item.status)}</span>
                  {item.refund_amount !== undefined && item.currency ? <strong>{formatMoney(item.refund_amount, item.currency)}</strong> : null}
                  <small>{dateLabel(item.requested_at || item.created_at)}</small>
                </div>
              </article>
            ))}
          </div>
        </section>
      </div>
    </main>
  );
}
