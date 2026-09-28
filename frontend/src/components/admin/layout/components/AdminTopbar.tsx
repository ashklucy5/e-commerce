"use client";

import Image from "next/image";
import Link from "next/link";

import type { AdminPrincipal } from "@/lib/admin/types";
import AdminNotificationCenter from "@/components/admin/notifications/components/AdminNotificationCenter";
import { siteConfig } from "@/lib/config/site";

import styles from "../css/AdminTopbar.module.css";

type AdminTopbarProps = {
  principal: AdminPrincipal;
  portal: string;
};

const roleLabels: Record<string, string> = {
  admin_superuser: "Super Admin",
  admin_administrator: "Administrator",
  admin_security: "Security",
  admin_sourcing: "Sourcing",
  admin_catalog: "Catalog",
  admin_inventory: "Inventory",
  admin_warehouse: "Warehouse",
  admin_fulfillment: "Fulfillment",
  admin_finance: "Finance",
  admin_returns: "Returns",
  admin_analytics: "Analytics",
  support_agent: "Support Agent",
  support_supervisor: "Support Supervisor",
};

function humanizeRole(role: string): string {
  if (roleLabels[role]) {
    return roleLabels[role];
  }

  return role
    .replace(/^admin_/, "")
    .replace(/_/g, " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

export default function AdminTopbar({
  principal,
  portal,
}: AdminTopbarProps) {
  const roles = principal.staff.roles
    .map(humanizeRole)
    .slice(0, 2);

  return (
    <header className={styles.topbar}>
      <div className={styles.identity}>
        <div className={styles.desktopIdentity}>
          <p className={styles.eyebrow}>
            Ene dei commerce
          </p>

          <p className={styles.title}>
            Operations console
          </p>
        </div>

        <Link
          href={`/${portal}`}
          className={styles.mobileBrand}
          aria-label="Ene Dei operations home"
        >
          <Image
            src={siteConfig.assets.logo}
            alt="Ene Dei"
            width={1100}
            height={439}
            priority
            sizes="8rem"
            className={styles.mobileBrandLogo}
          />
        </Link>
      </div>

      <div className={styles.actions}>
        <AdminNotificationCenter
          principal={principal}
          portal={portal}
        />

        <div className={styles.session}>
          <span className={styles.sessionDot} />

          <span className={styles.sessionText}>
            Secure session
          </span>
        </div>

        <div
          className={styles.operator}
          title={principal.staff.email}
        >
          <span className={styles.operatorName}>
            {principal.staff.full_name}
          </span>

          <span className={styles.operatorRole}>
            {roles.length > 0
              ? roles.join(" · ")
              : "Staff"}
          </span>
        </div>
      </div>
    </header>
  );
}