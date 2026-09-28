"use client";

import Image from "next/image";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";

import { Icon } from "@/components/ui/Icon";
import styles from "../css/StorefrontInstallPrompt.module.css";

const DISMISS_KEY =
  "ene-dei-install-prompt-dismissed-at-v1";

const DISMISS_FOR_MS =
  7 * 24 * 60 * 60 * 1000;

type InstallChoice = {
  outcome:
    | "accepted"
    | "dismissed";

  platform?: string;
};

type BeforeInstallPromptEvent =
  Event & {
    prompt: () =>
      Promise<InstallChoice>;

    userChoice:
      Promise<InstallChoice>;
  };

type IOSNavigator =
  Navigator & {
    standalone?: boolean;
  };

type Platform =
  | "ios"
  | "android"
  | "other-mobile"
  | null;

function isStandalone() {
  return (
    window.matchMedia(
      "(display-mode: standalone)",
    ).matches ||
    Boolean(
      (
        navigator as IOSNavigator
      ).standalone,
    )
  );
}

function detectPlatform():
  Platform {
  const ua =
    navigator.userAgent;

  const iPadDesktopMode =
    navigator.platform ===
      "MacIntel" &&
    navigator.maxTouchPoints >
      1;

  if (
    /iPad|iPhone|iPod/i.test(
      ua,
    ) ||
    iPadDesktopMode
  ) {
    return "ios";
  }

  if (
    /Android/i.test(
      ua,
    )
  ) {
    return "android";
  }

  if (
    /Mobile/i.test(
      ua,
    )
  ) {
    return "other-mobile";
  }

  return null;
}

function recentlyDismissed() {
  try {
    const value =
      Number(
        window.localStorage.getItem(
          DISMISS_KEY,
        ),
      );

    return (
      Number.isFinite(
        value,
      ) &&
      Date.now() -
        value <
        DISMISS_FOR_MS
    );
  } catch {
    return false;
  }
}

function rememberDismissal() {
  try {
    window.localStorage.setItem(
      DISMISS_KEY,
      String(
        Date.now(),
      ),
    );
  } catch {
    /*
     * The browser's own install
     * controls remain available.
     */
  }
}

function hiddenRoute(
  pathname: string,
) {
  return (
    pathname ===
      "/account/support" ||
    pathname.startsWith(
      "/account/support/",
    ) ||
    pathname.startsWith(
      "/checkout",
    ) ||
    pathname.startsWith(
      "/order/success",
    ) ||
    pathname.startsWith(
      "/order-success",
    ) ||
    (
      pathname.startsWith(
        "/account/request/",
      ) &&
      pathname.endsWith(
        "/checkout",
      )
    )
  );
}

export function StorefrontInstallPrompt() {
  const pathname =
    usePathname();

  const [
    platform,
    setPlatform,
  ] =
    useState<Platform>(
      null,
    );

  const [
    deferredPrompt,
    setDeferredPrompt,
  ] =
    useState<
      BeforeInstallPromptEvent | null
    >(
      null,
    );

  const [
    installed,
    setInstalled,
  ] =
    useState(
      false,
    );

  const [
    dismissed,
    setDismissed,
  ] =
    useState(
      true,
    );

  const [
    guideOpen,
    setGuideOpen,
  ] =
    useState(
      false,
    );

  const [
    ready,
    setReady,
  ] =
    useState(
      false,
    );

  useEffect(
    () => {
      setPlatform(
        detectPlatform(),
      );

      setInstalled(
        isStandalone(),
      );

      setDismissed(
        recentlyDismissed(),
      );

      const timer =
        window.setTimeout(
          () =>
            setReady(
              true,
            ),
          650,
        );

      const onBeforeInstallPrompt =
        (
          event: Event,
        ) => {
          event.preventDefault();

          setDeferredPrompt(
            event as BeforeInstallPromptEvent,
          );
        };

      const onInstalled =
        () => {
          setInstalled(
            true,
          );

          setDeferredPrompt(
            null,
          );

          setGuideOpen(
            false,
          );
        };

      const media =
        window.matchMedia(
          "(display-mode: standalone)",
        );

      const onDisplayModeChange =
        () => {
          if (
            isStandalone()
          ) {
            onInstalled();
          }
        };

      window.addEventListener(
        "beforeinstallprompt",
        onBeforeInstallPrompt,
      );

      window.addEventListener(
        "appinstalled",
        onInstalled,
      );

      media.addEventListener?.(
        "change",
        onDisplayModeChange,
      );

      return () => {
        window.clearTimeout(
          timer,
        );

        window.removeEventListener(
          "beforeinstallprompt",
          onBeforeInstallPrompt,
        );

        window.removeEventListener(
          "appinstalled",
          onInstalled,
        );

        media.removeEventListener?.(
          "change",
          onDisplayModeChange,
        );
      };
    },
    [],
  );

  useEffect(
    () => {
      if (
        !guideOpen
      ) {
        return;
      }

      const onKeyDown =
        (
          event:
            KeyboardEvent,
        ) => {
          if (
            event.key ===
            "Escape"
          ) {
            setGuideOpen(
              false,
            );
          }
        };

      document.addEventListener(
        "keydown",
        onKeyDown,
      );

      return () =>
        document.removeEventListener(
          "keydown",
          onKeyDown,
        );
    },
    [
      guideOpen,
    ],
  );

  function dismiss() {
    rememberDismissal();

    setDismissed(
      true,
    );

    setGuideOpen(
      false,
    );
  }

  async function install() {
    if (
      platform ===
      "ios"
    ) {
      setGuideOpen(
        true,
      );

      return;
    }

    if (
      !deferredPrompt
    ) {
      return;
    }

    try {
      const choice =
        await deferredPrompt.prompt();

      setDeferredPrompt(
        null,
      );

      if (
        choice.outcome ===
        "accepted"
      ) {
        setInstalled(
          true,
        );
      }
    } catch {
      setDeferredPrompt(
        null,
      );
    }
  }

  const nativeInstallAvailable =
    platform !== null &&
    platform !==
      "ios" &&
    deferredPrompt !==
      null;

  const showPrompt =
    ready &&
    !installed &&
    !dismissed &&
    !hiddenRoute(
      pathname,
    ) &&
    (
      platform ===
        "ios" ||
      nativeInstallAvailable
    );

  return (
    <>
      {showPrompt ? (
        <aside
          className={
            styles.prompt
          }
          aria-label="Install Ene Dei"
        >
          <span
            className={
              styles.promptIcon
            }
            aria-hidden="true"
          >
            <Image
              src="/Ene-Dei-Symbol-Fixed.svg"
              alt=""
              width={44}
              height={44}
            />
          </span>

          <span
            className={
              styles.promptCopy
            }
          >
            <strong>
              Install Ene Dei
            </strong>

            <span>
              Add the website
              to your Home
              Screen.
            </span>
          </span>

          <button
            type="button"
            className={
              styles.installButton
            }
            onClick={
              install
            }
          >
            Install
          </button>

          <button
            type="button"
            className={
              styles.dismissButton
            }
            aria-label="Hide install suggestion"
            onClick={
              dismiss
            }
          >
            <Icon
              name="close"
              size={14}
            />
          </button>
        </aside>
      ) : null}

      {guideOpen ? (
        <div
          className={
            styles.backdrop
          }
          role="presentation"
          onPointerDown={(
            event,
          ) => {
            if (
              event.target ===
              event.currentTarget
            ) {
              setGuideOpen(
                false,
              );
            }
          }}
        >
          <section
            className={
              styles.sheet
            }
            role="dialog"
            aria-modal="true"
            aria-labelledby="ene-install-title"
          >
            <div
              className={
                styles.sheetHandle
              }
              aria-hidden="true"
            />

            <header
              className={
                styles.sheetHeader
              }
            >
              <span
                className={
                  styles.sheetBrand
                }
                aria-hidden="true"
              >
                <Image
                  src="/Ene-Dei-Symbol-Fixed.svg"
                  alt=""
                  width={54}
                  height={54}
                />
              </span>

              <div>
                <h2
                  id="ene-install-title"
                >
                  Install Ene Dei
                </h2>

                <p>
                  Keep Ene Dei
                  on your iPhone
                  Home Screen and
                  open it like an
                  app.
                </p>
              </div>

              <button
                type="button"
                className={
                  styles.sheetClose
                }
                aria-label="Close install instructions"
                onClick={() =>
                  setGuideOpen(
                    false,
                  )
                }
              >
                <Icon
                  name="close"
                  size={17}
                />
              </button>
            </header>

            <ol
              className={
                styles.steps
              }
            >
              <li>
                <span
                  className={
                    styles.stepIcon
                  }
                >
                  <Icon
                    name="share"
                    size={20}
                  />
                </span>

                <div>
                  <strong>
                    Tap Share
                  </strong>

                  <p>
                    Use the Share
                    button in your
                    browser.
                  </p>
                </div>
              </li>

              <li>
                <span
                  className={
                    styles.stepNumber
                  }
                >
                  2
                </span>

                <div>
                  <strong>
                    Add to Home
                    Screen
                  </strong>

                  <p>
                    Scroll the Share
                    menu and choose
                    Add to Home
                    Screen.
                  </p>
                </div>
              </li>

              <li>
                <span
                  className={
                    styles.stepNumber
                  }
                >
                  3
                </span>

                <div>
                  <strong>
                    Add Ene Dei
                  </strong>

                  <p>
                    Keep Open as
                    Web App enabled
                    when shown, then
                    tap Add.
                  </p>
                </div>
              </li>
            </ol>

            <p
              className={
                styles.note
              }
            >
              No App Store download
              is required. Ene Dei
              remains the same live
              website.
            </p>

            <button
              type="button"
              className={
                styles.doneButton
              }
              onClick={() =>
                setGuideOpen(
                  false,
                )
              }
            >
              Got it
            </button>
          </section>
        </div>
      ) : null}
    </>
  );
}