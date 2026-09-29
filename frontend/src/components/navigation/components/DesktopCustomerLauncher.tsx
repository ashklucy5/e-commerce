"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import {
  useEffect,
  useRef,
  useState,
} from "react";

import { Icon } from "@/components/ui/Icon";
import { useCustomerSession } from "@/lib/account/use-customer-session";

import styles from "../css/DesktopCustomerLauncher.module.css";

const TUTORIAL_STORAGE_KEY =
  "ene-dei-customer-launcher-tutorial-seen";

function signInHref(
  destination: string,
) {
  return (
    "/account/sign-in?next=" +
    encodeURIComponent(
      destination,
    )
  );
}

export function DesktopCustomerLauncher() {
  const pathname =
    usePathname();

  const {
    isAuthenticated,
    isReady,
  } = useCustomerSession();

  const [
    open,
    setOpen,
  ] = useState(false);

  const [
    showTutorial,
    setShowTutorial,
  ] = useState(false);

  const containerRef =
    useRef<HTMLDivElement | null>(
      null,
    );

  /*
   * Show the tutorial only once
   * per browser/device.
   */
  useEffect(() => {
    try {
      const seen =
        window.localStorage.getItem(
          TUTORIAL_STORAGE_KEY,
        );

      if (!seen) {
        setShowTutorial(
          true,
        );
      }
    } catch {
      /*
       * localStorage can be unavailable
       * in restrictive/private modes.
       *
       * The launcher itself still works.
       */
    }
  }, []);

  /*
   * Close the launcher panel whenever
   * navigation occurs.
   */
  useEffect(() => {
    setOpen(false);
  }, [pathname]);

  /*
   * Clicking outside closes the panel.
   */
  useEffect(() => {
    function handlePointerDown(
      event: PointerEvent,
    ) {
      if (
        !open ||
        !containerRef.current
      ) {
        return;
      }

      if (
        !containerRef.current.contains(
          event.target as Node,
        )
      ) {
        setOpen(false);
      }
    }

    document.addEventListener(
      "pointerdown",
      handlePointerDown,
    );

    return () => {
      document.removeEventListener(
        "pointerdown",
        handlePointerDown,
      );
    };
  }, [open]);

  /*
   * Escape closes the launcher.
   */
  useEffect(() => {
    function handleKeyDown(
      event: KeyboardEvent,
    ) {
      if (
        event.key ===
        "Escape"
      ) {
        setOpen(false);
      }
    }

    document.addEventListener(
      "keydown",
      handleKeyDown,
    );

    return () => {
      document.removeEventListener(
        "keydown",
        handleKeyDown,
      );
    };
  }, []);

  function dismissTutorial() {
    setShowTutorial(
      false,
    );

    try {
      window.localStorage.setItem(
        TUTORIAL_STORAGE_KEY,
        "true",
      );
    } catch {
      /*
       * Nothing else is required.
       */
    }
  }

  function handleLauncherClick() {
    if (
      showTutorial
    ) {
      dismissTutorial();
    }

    setOpen(
      (current) =>
        !current,
    );
  }

  function protectedHref(
    destination: string,
  ) {
    /*
     * While the auth check is running,
     * use the protected destination.
     *
     * The destination itself is already
     * protected server-side.
     */
    if (!isReady) {
      return destination;
    }

    if (
      !isAuthenticated
    ) {
      return signInHref(
        destination,
      );
    }

    return destination;
  }

  /*
   * Hide the floating launcher while the
   * customer is already inside Support.
   *
   * Showing another Support button inside
   * the Support experience is redundant.
   */
  const isSupport =
    pathname ===
      "/account/support" ||
    pathname.startsWith(
      "/account/support/",
    );

  /*
   * Keep checkout experiences focused.
   */
  const isCommerceCheckout =
    pathname.startsWith(
      "/checkout",
    );

  const isOrderSuccess =
    pathname.startsWith(
      "/order/success",
    ) ||
    pathname.startsWith(
      "/order-success",
    );

  const isSourcingCheckout =
    pathname.startsWith(
      "/account/request/",
    ) &&
    pathname.endsWith(
      "/checkout",
    );

  const hidden =
    isSupport ||
    isCommerceCheckout ||
    isOrderSuccess ||
    isSourcingCheckout;

  if (hidden) {
    return null;
  }

  return (
    <div
      ref={containerRef}
      className={
        styles.root
      }
    >
      {showTutorial &&
      !open ? (
        <aside
          className={
            styles.tutorial
          }
          aria-label="Customer services introduction"
        >
          <button
            type="button"
            className={
              styles.tutorialClose
            }
            aria-label="Dismiss tip"
            onClick={
              dismissTutorial
            }
          >
            ×
          </button>

          <strong>
            Need help or looking
            for a product?
          </strong>

          <p>
            Customer Support and
            Product Request are
            available here.
          </p>

          <button
            type="button"
            className={
              styles.tutorialAction
            }
            onClick={
              dismissTutorial
            }
          >
            Got it
          </button>

          <span
            className={
              styles.tutorialArrow
            }
            aria-hidden="true"
          />
        </aside>
      ) : null}

      <div
        className={[
          styles.panel,

          open
            ? styles.panelOpen
            : "",
        ]
          .filter(Boolean)
          .join(" ")}
        aria-hidden={!open}
      >
        <div
          className={
            styles.panelHeader
          }
        >
          <span
            className={
              styles.eyebrow
            }
          >
            Customer services
          </span>

          <strong>
            What do you need?
          </strong>
        </div>

        <div
          className={
            styles.actions
          }
        >
          <Link
            href={protectedHref(
              "/account/support",
            )}
            className={
              styles.action
            }
            tabIndex={
              open
                ? 0
                : -1
            }
            onClick={() =>
              setOpen(false)
            }
          >
            <span
              className={
                styles.actionIcon
              }
            >
              <Icon
                name="support"
                size={20}
              />
            </span>

            <span
              className={
                styles.actionContent
              }
            >
              <strong>
                Customer Support
              </strong>

              <small>
                Orders, payments,
                delivery and account
                assistance
              </small>
            </span>

            <span
              className={
                styles.arrow
              }
              aria-hidden="true"
            >
              ›
            </span>
          </Link>

          <Link
            href={protectedHref(
              "/account/request",
            )}
            className={
              styles.action
            }
            tabIndex={
              open
                ? 0
                : -1
            }
            onClick={() =>
              setOpen(false)
            }
          >
            <span
              className={
                styles.actionIcon
              }
            >
              <Icon
                name="request"
                size={20}
              />
            </span>

            <span
              className={
                styles.actionContent
              }
            >
              <strong>
                Product Request
              </strong>

              <small>
                Ask us to source a
                product or quantity
              </small>
            </span>

            <span
              className={
                styles.arrow
              }
              aria-hidden="true"
            >
              ›
            </span>
          </Link>
        </div>

        {isReady &&
        !isAuthenticated ? (
          <p
            className={
              styles.authNote
            }
          >
            Sign in is required
            before opening Support
            or Product Request.
          </p>
        ) : null}
      </div>

      <button
        type="button"
        className={[
          styles.launcher,

          open
            ? styles.launcherOpen
            : "",
        ]
          .filter(Boolean)
          .join(" ")}
        aria-expanded={
          open
        }
        aria-haspopup="menu"
        aria-label={
          open
            ? "Close customer services"
            : "Open customer services"
        }
        onClick={
          handleLauncherClick
        }
      >
        {open ? (
          <span
            className={
              styles.closeIcon
            }
            aria-hidden="true"
          >
            ×
          </span>
        ) : (
          <Icon
            name="support"
            size={23}
          />
        )}

        {!showTutorial &&
        !open ? (
          <span
            className={
              styles.attentionDot
            }
            aria-hidden="true"
          />
        ) : null}
      </button>
    </div>
  );
}