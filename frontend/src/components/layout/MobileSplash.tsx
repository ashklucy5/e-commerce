"use client";

import Image from "next/image";

import {
  useEffect,
  useState,
} from "react";

import {
  siteConfig,
} from "@/lib/config/site";

const STORAGE_KEY =
  "ene-dei-mobile-splash-seen";

export function MobileSplash() {
  const [
    visible,
    setVisible,
  ] = useState(false);

  const [
    leaving,
    setLeaving,
  ] = useState(false);

  useEffect(() => {
    const mobile =
      window.matchMedia(
        "(max-width: 767px)",
      ).matches;

    if (!mobile) {
      return;
    }

    try {
      const alreadySeen =
        sessionStorage.getItem(
          STORAGE_KEY,
        ) === "1";

      if (alreadySeen) {
        return;
      }
    } catch {
      /*
       * Continue when browser storage is
       * blocked. The splash can still work.
       */
    }

    const reducedMotion =
      window.matchMedia(
        "(prefers-reduced-motion: reduce)",
      ).matches;

    const displayDuration =
      reducedMotion
        ? 150
        : 900;

    const exitDuration =
      reducedMotion
        ? 1
        : 280;

    let leaveTimer:
      | number
      | undefined;

    let hideTimer:
      | number
      | undefined;

    /*
     * Waiting until the next animation frame
     * avoids synchronously updating React
     * state inside the effect itself.
     */
    const showFrame =
      window.requestAnimationFrame(
        () => {
          setVisible(true);

          leaveTimer =
            window.setTimeout(
              () => {
                setLeaving(true);

                try {
                  sessionStorage.setItem(
                    STORAGE_KEY,
                    "1",
                  );
                } catch {
                  /*
                   * Storage is optional;
                   * complete the animation.
                   */
                }
              },
              displayDuration,
            );

          hideTimer =
            window.setTimeout(
              () => {
                setVisible(false);
              },
              displayDuration +
                exitDuration,
            );
        },
      );

    return () => {
      window.cancelAnimationFrame(
        showFrame,
      );

      if (
        leaveTimer !==
        undefined
      ) {
        window.clearTimeout(
          leaveTimer,
        );
      }

      if (
        hideTimer !==
        undefined
      ) {
        window.clearTimeout(
          hideTimer,
        );
      }
    };
  }, []);

  if (!visible) {
    return null;
  }

  return (
    <div
      className={[
        "mobile-splash",
        leaving
          ? "mobile-splash--leaving"
          : "",
      ]
        .filter(Boolean)
        .join(" ")}
      aria-hidden="true"
    >
      <div className="mobile-splash__glow" />

      <div className="mobile-splash__content">
        <Image
          src={
            siteConfig.assets
              .logo
          }
          alt=""
          width={320}
          height={110}
          priority
          className="mobile-splash__logo"
        />

        <div className="mobile-splash__accent">
          <span />
        </div>
      </div>

      <p className="mobile-splash__footer">
        Find what fits you.
      </p>
    </div>
  );
}