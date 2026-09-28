"use client";

import Link from "next/link";

import { Icon } from "@/components/ui/Icon";
import { useCustomerAuth } from "@/lib/account/use-customer-auth";

import styles from "../css/SiteHeader.module.css";

function getInitials(
  fullName: string,
) {
  const parts = fullName
    .trim()
    .split(/\s+/)
    .filter(Boolean);

  if (parts.length === 0) {
    return "U";
  }

  if (parts.length === 1) {
    return parts[0]
      .slice(0, 2)
      .toUpperCase();
  }

  return (
    parts[0][0] +
    parts[parts.length - 1][0]
  ).toUpperCase();
}

export function CustomerAccountAction() {
  const {
    customer,
    isAuthenticated,
  } = useCustomerAuth();

  return (
    <Link
      href="/account"
      className={styles.action}
      aria-label={
        isAuthenticated &&
        customer
          ? `Account for ${customer.full_name}`
          : "Account"
      }
    >
      {isAuthenticated &&
      customer ? (
        <span
          className={
            styles.accountAvatar
          }
          aria-hidden="true"
        >
          {customer.avatar_url ? (
            /*
             * Avatar URLs can be backend-issued
             * signed URLs, so use a normal image
             * rather than requiring a Next/Image
             * remote-host configuration.
             */
            <img
              src={customer.avatar_url}
              alt=""
              className={
                styles.accountAvatarImage
              }
            />
          ) : (
            <span
              className={
                styles.accountAvatarFallback
              }
            >
              {getInitials(
                customer.full_name,
              )}
            </span>
          )}
        </span>
      ) : (
        <Icon
          name="account"
          size={20}
        />
      )}

      <span
        className={
          styles.actionLabel
        }
      >
        Account
      </span>
    </Link>
  );
}