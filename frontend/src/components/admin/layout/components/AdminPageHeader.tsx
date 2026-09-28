import type {
  ReactNode,
} from "react";

import styles from "../css/AdminPageHeader.module.css";

type AdminPageHeaderProps = {
  eyebrow?: string;
  title: string;
  description?: string;
  actions?: ReactNode;
};

export default function AdminPageHeader({
  eyebrow,
  title,
  description,
  actions,
}: AdminPageHeaderProps) {
  return (
    <header className={styles.header}>
      <div className={styles.copy}>
        {eyebrow ? (
          <p className={styles.eyebrow}>
            {eyebrow}
          </p>
        ) : null}

        <h1 className={styles.title}>
          {title}
        </h1>

        {description ? (
          <p
            className={
              styles.description
            }
          >
            {description}
          </p>
        ) : null}
      </div>

      {actions ? (
        <div className={styles.actions}>
          {actions}
        </div>
      ) : null}
    </header>
  );
}