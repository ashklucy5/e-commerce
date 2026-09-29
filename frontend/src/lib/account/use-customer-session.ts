"use client";

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import {
  usePathname,
} from "next/navigation";

import {
  CUSTOMER_AUTH_CHANGED_EVENT,
  CUSTOMER_SESSION_HINT_COOKIE,
} from "./session-contract";

type CustomerSessionState = {
  isAuthenticated: boolean;
  isReady: boolean;
};

function hasCustomerSessionHint() {
  if (
    typeof document ===
    "undefined"
  ) {
    return false;
  }

  const prefix =
    `${encodeURIComponent(
      CUSTOMER_SESSION_HINT_COOKIE,
    )}=`;

  return document.cookie
    .split(";")
    .some(
      (part) =>
        part
          .trim()
          .startsWith(prefix),
    );
}

export function useCustomerSession():
  CustomerSessionState {
  const pathname =
    usePathname();

  const [
    state,
    setState,
  ] =
    useState<CustomerSessionState>({
      isAuthenticated:
        false,

      isReady:
        false,
    });

  const sync =
    useCallback(() => {
      const authenticated =
        hasCustomerSessionHint();

      setState(
        (current) => {
          if (
            current.isReady &&
            current.isAuthenticated ===
              authenticated
          ) {
            return current;
          }

          return {
            isAuthenticated:
              authenticated,

            isReady:
              true,
          };
        },
      );
    }, []);

  /*
   * Login/logout navigates to another route.
   * Re-read the local session hint when that
   * happens instead of calling /account/me.
   */
  useEffect(() => {
    sync();
  }, [
    pathname,
    sync,
  ]);

  /*
   * Account-specific UI can explicitly signal
   * that authentication changed.
   */
  useEffect(() => {
    function handleAuthChanged() {
      sync();
    }

    function handleFocus() {
      /*
       * Cheap local cookie read only.
       * No API call.
       */
      sync();
    }

    window.addEventListener(
      CUSTOMER_AUTH_CHANGED_EVENT,
      handleAuthChanged,
    );

    window.addEventListener(
      "focus",
      handleFocus,
    );

    return () => {
      window.removeEventListener(
        CUSTOMER_AUTH_CHANGED_EVENT,
        handleAuthChanged,
      );

      window.removeEventListener(
        "focus",
        handleFocus,
      );
    };
  }, [
    sync,
  ]);

  return state;
}