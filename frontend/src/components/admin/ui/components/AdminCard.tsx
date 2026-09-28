import type { ReactNode } from "react";

import styles from "../css/AdminCard.module.css";

type AdminCardProps = {
  children: ReactNode;
  className?: string;
};

export default function AdminCard({
  children,
  className = "",
}: AdminCardProps) {
  return (
    <section
      className={[
        styles.card,
        className,
      ]
        .filter(Boolean)
        .join(" ")}
    >
      {children}
    </section>
  );
}