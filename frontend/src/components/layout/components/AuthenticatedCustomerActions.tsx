"use client";

import Link from "next/link";

import { Icon } from "@/components/ui/Icon";
import { useCustomerAuth } from "@/lib/account/use-customer-auth";

import styles from "../css/SiteHeader.module.css";

export function AuthenticatedCustomerActions() {
  const {
    isAuthenticated,
    isLoading,
  } = useCustomerAuth();

  if (
    isLoading ||
    !isAuthenticated
  ) {
    return null;
  }

  return (
    <>
      <Link
        href="/account/request"
        className={`${styles.action} ${styles.desktopOnlyAction}`}
        aria-label="Product Request"
      >
        <Icon
          name="request"
          size={20}
        />

        <span className={styles.actionLabel}>
          Request
        </span>
      </Link>

      <Link
        href="/account/support"
        className={`${styles.action} ${styles.desktopOnlyAction}`}
        aria-label="Customer Support"
      >
        <Icon
          name="support"
          size={20}
        />

        <span className={styles.actionLabel}>
          Support
        </span>
      </Link>
    </>
  );
}