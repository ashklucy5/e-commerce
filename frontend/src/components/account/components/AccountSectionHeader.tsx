// Location: src/components/account/components/AccountSectionHeader.tsx
import Link from "next/link";

import { Icon } from "@/components/ui/Icon";

import styles from "../css/AccountManagement.module.css";

export function AccountSectionHeader({
  eyebrow,
  title,
  description,
}: {
  eyebrow: string;
  title: string;
  description: string;
}) {
  return (
    <>
      <Link className={styles.back} href="/account">
        <Icon name="chevronLeft" size={14} />
        Account overview
      </Link>
      <header className={styles.hero}>
        <div>
          <span className={styles.eyebrow}>{eyebrow}</span>
          <h1>{title}</h1>
          <p>{description}</p>
        </div>
      </header>
    </>
  );
}
