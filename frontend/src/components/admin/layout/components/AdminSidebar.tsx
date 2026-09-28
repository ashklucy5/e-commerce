"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";

import type { AdminPrincipal } from "@/lib/admin/types";
import { siteConfig } from "@/lib/config/site";

import styles from "../css/AdminSidebar.module.css";

type AdminSidebarProps = {
  principal: AdminPrincipal | null;
  portal: string;
  sessionReady?: boolean;
};

type NavItem = {
  path: string;
  label: string;
  mobileLabel?: string;
  icon: string;
  exact?: boolean;
};

type NavGroup = {
  label: string;
  items: NavItem[];
};

const navigation: NavGroup[] = [
  {
    label: "Operations",
    items: [
      {
        path: "",
        label: "Dashboard",
        mobileLabel: "Home",
        icon: "◈",
        exact: true,
      },
      {
        path: "orders",
        label: "Orders",
        mobileLabel: "Orders",
        icon: "▤",
      },
      {
        path: "fulfillment",
        label: "Fulfillment",
        mobileLabel: "Fulfill",
        icon: "⌁",
      },
      {
        path: "shipments",
        label: "Shipments",
        mobileLabel: "Ship",
        icon: "⇢",
      },
      {
        path: "products",
        label: "Products",
        mobileLabel: "Products",
        icon: "◇",
      },
      {
        path: "inventory",
        label: "Inventory",
        mobileLabel: "Stock",
        icon: "▦",
      },
    ],
  },
  {
    label: "Commerce",
    items: [
      {
        path: "product-requests",
        label: "Product requests",
        mobileLabel: "Requests",
        icon: "⌕",
      },
      {
        path: "promotions",
        label: "Promotions & discounts",
        mobileLabel: "Promos",
        icon: "%",
      },
      {
        path: "customers",
        label: "Customers",
        mobileLabel: "Customers",
        icon: "◎",
      },
      {
        path: "returns",
        label: "Returns & refunds",
        mobileLabel: "Returns",
        icon: "↺",
      },
      {
        path: "reviews",
        label: "Reviews",
        mobileLabel: "Reviews",
        icon: "✦",
      },
      {
        path: "support",
        label: "Support",
        mobileLabel: "Support",
        icon: "◌",
      },
    ],
  },
  {
    label: "Management",
    items: [
      {
        path: "analytics",
        label: "Analytics",
        mobileLabel: "Analytics",
        icon: "⌁",
      },
      {
        path: "staff",
        label: "Staff & roles",
        mobileLabel: "Staff",
        icon: "♙",
      },
    ],
  },
  {
    label: "Personal",
    items: [
      {
        path: "account",
        label: "Account & security",
        mobileLabel: "Account",
        icon: "⚙",
      },
    ],
  },
];

function buildHref(portal: string, path: string): string {
  const base = `/${portal}`;
  return path ? `${base}/${path}` : base;
}

function isActive(
  pathname: string,
  href: string,
  exact = false,
): boolean {
  if (exact) {
    return pathname === href;
  }

  return pathname === href || pathname.startsWith(`${href}/`);
}

export default function AdminSidebar({
  principal,
  portal,
  sessionReady = true,
}: AdminSidebarProps) {
  const pathname = usePathname();

  const dashboardHref = buildHref(portal, "");
  const name = principal?.staff.full_name || "Administrator";
  const staffCode = principal?.staff.staff_code || "Loading account";

  const initials = principal
    ? name
        .split(/\s+/)
        .filter(Boolean)
        .slice(0, 2)
        .map((part) => part.charAt(0))
        .join("")
        .toUpperCase() || "AD"
    : "ED";

  return (
    <aside className={styles.sidebar}>
      <div className={styles.ambient} aria-hidden="true" />

      <div className={styles.brand}>
        <Link href={dashboardHref} className={styles.brandLink}>
          <Image
            src={siteConfig.assets.logo}
            alt=""
            width={1100}
            height={439}
            priority
            sizes="(max-width: 67.5rem) 2.7rem, 8.5rem"
            className={styles.brandLogo}
          />

          <span className={styles.brandMeta}>Operations</span>
        </Link>
      </div>

      <nav
        className={styles.navigation}
        aria-label="Operations navigation"
      >
        <div className={styles.navigationTrack}>
          {navigation.map((group) => (
            <div key={group.label} className={styles.group}>
              <p className={styles.groupLabel}>{group.label}</p>

              <div className={styles.groupItems}>
                {group.items.map((item) => {
                  const href = buildHref(portal, item.path);
                  const active = isActive(
                    pathname,
                    href,
                    item.exact,
                  );

                  return (
                    <Link
                      key={item.path}
                      href={href}
                      title={item.label}
                      aria-current={active ? "page" : undefined}
                      className={[
                        styles.navItem,
                        active ? styles.active : "",
                      ]
                        .filter(Boolean)
                        .join(" ")}
                    >
                      {active ? (
                        <span
                          className={styles.activeBar}
                          aria-hidden="true"
                        />
                      ) : null}

                      <span className={styles.navIcon} aria-hidden="true">
                        {item.icon}
                      </span>

                      <span className={styles.navLabel}>{item.label}</span>

                      <span
                        className={styles.mobileNavLabel}
                        aria-hidden="true"
                      >
                        {item.mobileLabel ?? item.label}
                      </span>
                    </Link>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      </nav>

      <div className={styles.footer}>
        <div
          className={[
            styles.profile,
            !sessionReady ? styles.profileLoading : "",
          ]
            .filter(Boolean)
            .join(" ")}
        >
          <div className={styles.avatar}>{initials}</div>

          <div className={styles.profileCopy}>
            <p className={styles.profileName}>{name}</p>
            <p className={styles.profileMeta}>{staffCode}</p>
          </div>

          <span
            className={styles.statusDot}
            title={sessionReady ? "Active" : "Checking session"}
          />
        </div>
      </div>
    </aside>
  );
}
