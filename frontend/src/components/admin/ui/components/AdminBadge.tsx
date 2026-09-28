import type { ReactNode } from "react";

import styles from "../css/AdminBadge.module.css";

type AdminBadgeTone =
  | "neutral"
  | "success"
  | "warning"
  | "danger"
  | "info";

type AdminBadgeProps = {
  children: ReactNode;
  tone?: AdminBadgeTone;
};

export default function AdminBadge({
  children,
  tone = "neutral",
}: AdminBadgeProps) {
  return (
    <span
      className={[
        styles.badge,
        styles[tone],
      ].join(" ")}
    >
      {children}
    </span>
  );
}