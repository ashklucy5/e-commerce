"use client";

import Link from "next/link";

import {
  Icon,
} from "@/components/ui/Icon";

import styles from "../css/SiteHeader.module.css";

export function CustomerAccountAction() {
  return (
    <Link
      href="/account"
      className={
        styles.action
      }
      aria-label="Account"
    >
      <Icon
        name="account"
        size={20}
      />

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