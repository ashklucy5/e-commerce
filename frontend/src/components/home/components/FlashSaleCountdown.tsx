"use client";

import {
  useEffect,
  useState,
} from "react";

import styles from "../css/FlashSaleCountdown.module.css";

type Props = {
  target?: number;
};

type ActiveCountdownProps = {
  target: number;
};

function ActiveFlashSaleCountdown({
  target,
}: ActiveCountdownProps) {
  const [
    remaining,
    setRemaining,
  ] = useState(() =>
    Math.max(
      0,
      target - Date.now(),
    ),
  );

  useEffect(() => {
    const timer =
      window.setInterval(
        () => {
          setRemaining(
            Math.max(
              0,
              target -
                Date.now(),
            ),
          );
        },
        1000,
      );

    return () => {
      window.clearInterval(
        timer,
      );
    };
  }, [target]);

  if (remaining <= 0) {
    return (
      <span
        className={
          styles.ended
        }
      >
        Ended
      </span>
    );
  }

  const totalSeconds =
    Math.floor(
      remaining / 1000,
    );

  const days =
    Math.floor(
      totalSeconds / 86400,
    );

  const hours =
    Math.floor(
      (totalSeconds %
        86400) /
        3600,
    );

  const minutes =
    Math.floor(
      (totalSeconds %
        3600) /
        60,
    );

  const seconds =
    totalSeconds % 60;

  const clock =
    `${String(hours).padStart(
      2,
      "0",
    )}:${String(
      minutes,
    ).padStart(
      2,
      "0",
    )}:${String(
      seconds,
    ).padStart(
      2,
      "0",
    )}`;

  return (
    <span
      className={
        styles.countdown
      }
      role="timer"
      aria-label={
        days > 0
          ? `Flash sale ends in ${days} days and ${clock}`
          : `Flash sale ends in ${clock}`
      }
    >
      <span>
        Ends in
      </span>

      <strong>
        {days > 0
          ? `${days}d ${clock}`
          : clock}
      </strong>
    </span>
  );
}

export function FlashSaleCountdown({
  target,
}: Props) {
  if (!target) {
    return null;
  }

  /*
   * Using target as the key deliberately remounts
   * the timer if a different promotion becomes
   * active. This avoids synchronizing local state
   * with props inside an effect.
   */
  return (
    <ActiveFlashSaleCountdown
      key={target}
      target={target}
    />
  );
}