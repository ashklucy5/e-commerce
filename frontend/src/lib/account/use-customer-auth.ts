"use client";

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import { usePathname } from "next/navigation";

import type {
  AccountCustomer,
  AccountDataResponse,
} from "@/lib/api/contracts/account";

type CustomerAuthState = {
  customer: AccountCustomer | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  refresh: () => Promise<void>;
};

export function useCustomerAuth(): CustomerAuthState {
  const pathname = usePathname();

  const [customer, setCustomer] =
    useState<AccountCustomer | null>(null);

  const [isLoading, setIsLoading] =
    useState(true);

  const checkSession =
    useCallback(async () => {
      setIsLoading(true);

      try {
        const response = await fetch(
          "/api/storefront/account/me",
          {
            method: "GET",
            cache: "no-store",
          },
        );

        if (!response.ok) {
          setCustomer(null);
          return;
        }

        const payload =
          (await response.json()) as AccountDataResponse<AccountCustomer>;

        setCustomer(
          payload?.data ?? null,
        );
      } catch {
        setCustomer(null);
      } finally {
        setIsLoading(false);
      }
    }, []);

  /*
   * Important:
   *
   * The storefront layout can remain mounted while
   * Next.js navigates between account pages.
   *
   * Rechecking on pathname changes means the header
   * and navigation update immediately after login
   * and logout instead of keeping stale auth state.
   */
  useEffect(() => {
    void checkSession();
  }, [
    pathname,
    checkSession,
  ]);

  /*
   * Also refresh when the customer comes back to
   * the browser tab. This helps after authentication
   * in another tab/window.
   */
  useEffect(() => {
    function handleFocus() {
      void checkSession();
    }

    function handleVisibilityChange() {
      if (
        document.visibilityState ===
        "visible"
      ) {
        void checkSession();
      }
    }

    function handleAuthChanged() {
      void checkSession();
    }

    window.addEventListener(
      "focus",
      handleFocus,
    );

    window.addEventListener(
      "customer-auth-changed",
      handleAuthChanged,
    );

    document.addEventListener(
      "visibilitychange",
      handleVisibilityChange,
    );

    return () => {
      window.removeEventListener(
        "focus",
        handleFocus,
      );

      window.removeEventListener(
        "customer-auth-changed",
        handleAuthChanged,
      );

      document.removeEventListener(
        "visibilitychange",
        handleVisibilityChange,
      );
    };
  }, [checkSession]);

  return {
    customer,

    isAuthenticated:
      customer !== null,

    isLoading,

    refresh: checkSession,
  };
}