// Location: src/components/account/components/SecurityClient.tsx
"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { Icon } from "@/components/ui/Icon";
import type {
  AccountDataResponse,
  AccountSecurityOverview,
  AccountSession,
  AccountSessionsResponse,
} from "@/lib/api/contracts/account";

import { AccountSectionHeader } from "./AccountSectionHeader";
import styles from "../css/AccountManagement.module.css";

function dateLabel(value?: string) {
  if (!value) return "Not available";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat("en", { dateStyle: "medium", timeStyle: "short" }).format(date);
}

async function readError(response: Response, fallback: string) {
  try {
    const payload = (await response.json()) as { error?: { message?: string } };
    return payload.error?.message?.trim() || fallback;
  } catch {
    return fallback;
  }
}

export function SecurityClient() {
  const [security, setSecurity] = useState<AccountSecurityOverview | null>(null);
  const [sessions, setSessions] = useState<AccountSession[]>([]);
  const [state, setState] = useState<"loading" | "ready" | "signed-out" | "error">("loading");
  const [busy, setBusy] = useState("");
  const [message, setMessage] = useState("");

  const load = useCallback(async () => {
    setState("loading");
    setMessage("");
    try {
      const securityResponse = await fetch("/api/storefront/account/security", { cache: "no-store" });
      if (securityResponse.status === 401) { setState("signed-out"); return; }
      if (!securityResponse.ok) throw new Error(await readError(securityResponse, "Unable to load security."));
      const securityPayload = (await securityResponse.json()) as AccountDataResponse<AccountSecurityOverview>;
      setSecurity(securityPayload.data);

      const sessionsResponse = await fetch("/api/storefront/account/security/sessions", { cache: "no-store" });
      if (!sessionsResponse.ok) throw new Error(await readError(sessionsResponse, "Unable to load sessions."));
      const sessionsPayload = (await sessionsResponse.json()) as AccountSessionsResponse;
      setSessions(sessionsPayload.data ?? []);
      setState("ready");
    } catch (caught) {
      setMessage(caught instanceof Error ? caught.message : "Unable to load account security.");
      setState("error");
    }
  }, []);

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

  async function revokeSession(sessionID: string) {
    if (busy) return;
    setBusy(sessionID);
    setMessage("");
    try {
      const response = await fetch(`/api/storefront/account/security/sessions/${encodeURIComponent(sessionID)}`, { method: "DELETE" });
      if (!response.ok) throw new Error(await readError(response, "Unable to revoke session."));
      await load();
    } catch (caught) {
      setMessage(caught instanceof Error ? caught.message : "Unable to revoke session.");
    } finally {
      setBusy("");
    }
  }

  async function revokeOthers() {
    if (busy) return;
    setBusy("others");
    setMessage("");
    try {
      const response = await fetch("/api/storefront/account/security/sessions/revoke-others", { method: "POST" });
      if (!response.ok) throw new Error(await readError(response, "Unable to sign out other devices."));
      await load();
      setMessage("Other sessions were revoked.");
    } catch (caught) {
      setMessage(caught instanceof Error ? caught.message : "Unable to sign out other devices.");
    } finally {
      setBusy("");
    }
  }

  if (state === "signed-out") {
    return <main className={styles.page}><section className={styles.state}><Icon name="secureCheckout" size={24} /><h1>Sign in to open security.</h1><Link className={styles.primary} href="/account/sign-in?next=%2Faccount%2Fsecurity">Sign in</Link></section></main>;
  }

  return (
    <main className={styles.page}>
      <div className={styles.shell}>
        <AccountSectionHeader eyebrow="Security" title="Your sessions, under control." description="Review account security and revoke sessions you no longer recognize or use." />
        <div className={styles.grid}>
          <section className={`${styles.card} ${styles.span5}`}>
            <div className={styles.cardHeader}><div><h2>Security overview</h2><p>Backend-authoritative account security state.</p></div></div>
            {state === "loading" ? <p className={styles.message}>Loading security…</p> : null}
            {security ? (
              <div className={styles.list}>
                <div className={styles.row}><div className={styles.rowMain}><strong>Password</strong><span>Last changed</span></div><div className={styles.rowMeta}><strong>{dateLabel(security.password_updated_at)}</strong></div></div>
                <div className={styles.row}><div className={styles.rowMain}><strong>Last sign in</strong><span>Latest recorded login</span></div><div className={styles.rowMeta}><strong>{dateLabel(security.last_login_at)}</strong></div></div>
                <div className={styles.row}><div className={styles.rowMain}><strong>Active sessions</strong><span>Signed-in browsers and devices</span></div><div className={styles.rowMeta}><strong>{(security.active_session_count ?? security.session_count ?? sessions.length).toLocaleString()}</strong></div></div>
              </div>
            ) : null}
          </section>

          <section className={`${styles.card} ${styles.span7}`}>
            <div className={styles.cardHeader}><div><h2>Sessions</h2><p>Revoke any session you no longer want active.</p></div><button className={styles.danger} type="button" disabled={Boolean(busy) || sessions.length <= 1} onClick={() => void revokeOthers()}>{busy === "others" ? "Signing out…" : "Sign out other devices"}</button></div>
            {state === "error" ? <p className={`${styles.message} ${styles.error}`} role="alert">{message || "Security is temporarily unavailable."}</p> : null}
            {message && state === "ready" ? <p className={styles.message} role="status">{message}</p> : null}
            <div className={styles.list}>
              {sessions.map((session) => (
                <article className={`${styles.row} ${session.is_current ? styles.sessionCurrent : ""}`} key={session.id}>
                  <div className={styles.rowMain}>
                    <strong>{session.device_name || "Signed-in device"} {session.is_current ? <span className={styles.positiveBadge}>Current</span> : null}</strong>
                    <span>{session.user_agent || "Device details unavailable"}</span>
                    <p>Last seen {dateLabel(session.last_seen_at || session.created_at)}</p>
                  </div>
                  {!session.is_current ? <button className={styles.danger} type="button" disabled={Boolean(busy)} onClick={() => void revokeSession(session.id)}>{busy === session.id ? "Revoking…" : "Revoke"}</button> : null}
                </article>
              ))}
              {state === "ready" && sessions.length === 0 ? <div className={styles.empty}>No active sessions were returned.</div> : null}
            </div>
          </section>
        </div>
      </div>
    </main>
  );
}
