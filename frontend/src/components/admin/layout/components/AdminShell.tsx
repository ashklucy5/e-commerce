"use client";

import type { ReactNode } from "react";
import {
  useEffect,
  useState,
} from "react";
import { useRouter } from "next/navigation";

import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";
import type {
  AdminMeResponse,
  AdminPrincipal,
} from "@/lib/admin/types";

import { AdminSessionProvider } from "../context/AdminSessionContext";
import AdminBackground from "./AdminBackground";
import AdminSidebar from "./AdminSidebar";
import AdminTopbar from "./AdminTopbar";

import styles from "../css/AdminShell.module.css";

type AdminShellProps = {
  children: ReactNode;
  portal: string;
};

type SessionState =
  | "loading"
  | "ready"
  | "error";

export default function AdminShell({
  children,
  portal,
}: AdminShellProps) {
  const router = useRouter();

  const [principal, setPrincipal] =
    useState<AdminPrincipal | null>(
      null,
    );

  const [state, setState] =
    useState<SessionState>(
      "loading",
    );

  useEffect(() => {
    const controller =
      new AbortController();

    async function loadSession() {
      try {
        const response =
          await adminFetch<AdminMeResponse>(
            "/auth/me",
            {
              signal:
                controller.signal,
            },
          );

        if (
          controller.signal.aborted
        ) {
          return;
        }

        setPrincipal(
          response.data.principal,
        );

        setState(
          "ready",
        );
      } catch (error) {
        if (
          controller.signal.aborted
        ) {
          return;
        }

        if (
          error instanceof
            AdminRequestError &&
          (
            error.status === 401 ||
            error.status === 403
          )
        ) {
          router.replace(
            `/${portal}/login`,
          );

          return;
        }

        setState(
          "error",
        );
      }
    }

    void loadSession();

    return () => {
      controller.abort();
    };
  }, [
    portal,
    router,
  ]);

  const readyPrincipal =
    state === "ready"
      ? principal
      : null;

  return (
    <main className={styles.shell}>
      <AdminBackground />

      <div className={styles.layout}>
        <AdminSidebar
          principal={principal}
          portal={portal}
          sessionReady={
            readyPrincipal !== null
          }
        />

        <section
          className={styles.main}
          aria-busy={
            readyPrincipal === null &&
            state === "loading"
          }
        >
          {readyPrincipal ? (
            <AdminSessionProvider
              principal={
                readyPrincipal
              }
            >
              <AdminTopbar
                principal={
                  readyPrincipal
                }
                portal={portal}
              />

              <div
                className={
                  styles.content
                }
              >
                {children}
              </div>
            </AdminSessionProvider>
          ) : state === "error" ? (
            <div
              className={
                styles.errorStage
              }
            >
              <p
                className={
                  styles.errorEyebrow
                }
              >
                Operations unavailable
              </p>

              <h1
                className={
                  styles.errorTitle
                }
              >
                We could not load
                the Admin session.
              </h1>

              <p
                className={
                  styles.errorText
                }
              >
                Check that the
                commerce API is
                running, then
                refresh this page.
              </p>
            </div>
          ) : (
            <div
              className={
                styles.sessionStage
              }
            >
              <div
                className={
                  styles.sessionPulse
                }
                aria-hidden="true"
              />

              <div
                className={
                  styles.sessionCopy
                }
              >
                <p
                  className={
                    styles.loadingText
                  }
                >
                  Opening
                  operations…
                </p>
              </div>
            </div>
          )}
        </section>
      </div>
    </main>
  );
}