"use client";

import {
  useEffect,
  useRef,
} from "react";

import styles from "../css/AdminBackground.module.css";

export default function AdminBackground() {
  const rootRef =
    useRef<HTMLDivElement>(null);

  useEffect(() => {
    const root = rootRef.current;

    if (!root) {
      return;
    }

    const reducedMotion =
      window.matchMedia(
        "(prefers-reduced-motion: reduce)",
      );

    const finePointer =
      window.matchMedia(
        "(pointer: fine)",
      );

    if (
      reducedMotion.matches ||
      !finePointer.matches
    ) {
      return;
    }

    let frame = 0;

    function handlePointerMove(
      event: PointerEvent,
    ) {
      cancelAnimationFrame(frame);

      frame = requestAnimationFrame(() => {
        const current =
          rootRef.current;

        if (!current) {
          return;
        }

        const x =
          event.clientX /
            window.innerWidth -
          0.5;

        const y =
          event.clientY /
            window.innerHeight -
          0.5;

        current.style.setProperty(
          "--admin-parallax-x",
          `${x}`,
        );

        current.style.setProperty(
          "--admin-parallax-y",
          `${y}`,
        );
      });
    }

    function handlePointerLeave() {
      const current =
        rootRef.current;

      if (!current) {
        return;
      }

      current.style.setProperty(
        "--admin-parallax-x",
        "0",
      );

      current.style.setProperty(
        "--admin-parallax-y",
        "0",
      );
    }

    window.addEventListener(
      "pointermove",
      handlePointerMove,
      {
        passive: true,
      },
    );

    document.addEventListener(
      "mouseleave",
      handlePointerLeave,
    );

    return () => {
      cancelAnimationFrame(frame);

      window.removeEventListener(
        "pointermove",
        handlePointerMove,
      );

      document.removeEventListener(
        "mouseleave",
        handlePointerLeave,
      );
    };
  }, []);

  return (
    <div
      ref={rootRef}
      className={styles.background}
      aria-hidden="true"
    >
      <div className={styles.baseGlow} />

      <div className={styles.grid} />

      <div
        className={[
          styles.orb,
          styles.orbRed,
        ].join(" ")}
      />

      <div
        className={[
          styles.orb,
          styles.orbViolet,
        ].join(" ")}
      />

      <div
        className={[
          styles.orb,
          styles.orbSilver,
        ].join(" ")}
      />

      <div className={styles.ringOne} />

      <div className={styles.ringTwo} />

      <div className={styles.noise} />

      <div className={styles.vignette} />
    </div>
  );
}