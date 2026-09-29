"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import {
  Icon,
  type IconName,
} from "@/components/ui/Icon";

import { useCustomerSession } from "@/lib/account/use-customer-session";

import styles from "../css/MobileBottomNav.module.css";

type NavKey =
  | "home"
  | "support"
  | "request"
  | "orders"
  | "account";

type NavItem = {
  key: NavKey;
  label: string;
  href: string;
  icon: IconName;
  activeIcon?: IconName;
  requiresAuth?: boolean;
};

const navigationItems: NavItem[] = [
  {
    key: "home",
    label: "Home",
    href: "/",
    icon: "home",
    activeIcon: "homeFilled",
  },
  {
    key: "support",
    label: "Support",
    href: "/account/support",
    icon: "support",
    requiresAuth: true,
  },
  {
    key: "request",
    label: "Request",
    href: "/account/request",
    icon: "request",
    requiresAuth: true,
  },
  {
    key: "orders",
    label: "Orders",
    href: "/account/orders",
    icon: "orders",
  },
  {
    key: "account",
    label: "Account",
    href: "/account",
    icon: "account",
  },
];

function signInDestination(
  destination: string,
) {
  return (
    "/account/sign-in?next=" +
    encodeURIComponent(destination)
  );
}

export function MobileBottomNav() {
  const pathname = usePathname();

  const {
    isAuthenticated,
    isReady,
  } = useCustomerSession();

  function getItemHref(
    item: NavItem,
  ) {
    if (
  item.requiresAuth &&
  isReady &&
  !isAuthenticated
) {
      return signInDestination(
        item.href,
      );
    }

    return item.href;
  }

  function isItemActive(
    item: NavItem,
  ) {
    switch (item.key) {
      case "home":
        return pathname === "/";

      case "support":
        return (
          pathname ===
            "/account/support" ||
          pathname.startsWith(
            "/account/support/",
          )
        );

      case "request":
        return (
          pathname ===
            "/account/request" ||
          pathname.startsWith(
            "/account/request/",
          )
        );

      case "orders":
        return (
          pathname ===
            "/account/orders" ||
          pathname.startsWith(
            "/account/orders/",
          )
        );

      case "account":
        return (
          pathname ===
            "/account" ||
          (
            pathname.startsWith(
              "/account/",
            ) &&
            !pathname.startsWith(
              "/account/support",
            ) &&
            !pathname.startsWith(
              "/account/request",
            ) &&
            !pathname.startsWith(
              "/account/orders",
            )
          )
        );

      default:
        return false;
    }
  }

  /*
   * Support is an immersive messaging
   * experience on mobile.
   *
   * Do not place the global bottom
   * navigation above the keyboard.
   */
  const isSupport =
    pathname ===
      "/account/support" ||
    pathname.startsWith(
      "/account/support/",
    );

  /*
   * Commerce checkout is also focused
   * and should not show global nav.
   */
  const isCommerceCheckout =
    pathname.startsWith(
      "/checkout",
    );

  const isOrderSuccess =
    pathname.startsWith(
      "/order/success",
    ) ||
    pathname.startsWith(
      "/order-success",
    );

  /*
   * Dedicated sourcing checkout should
   * remain focused as well.
   */
  const isSourcingCheckout =
    pathname.startsWith(
      "/account/request/",
    ) &&
    pathname.endsWith(
      "/checkout",
    );

  if (
    isSupport ||
    isCommerceCheckout ||
    isOrderSuccess ||
    isSourcingCheckout
  ) {
    return null;
  }

  return (
    <nav
      className={styles.shell}
      aria-label="Primary mobile navigation"
    >
      <div className={styles.glass}>
        <div
          className={styles.highlight}
          aria-hidden="true"
        />

        <div
          className={styles.navigation}
        >
          {navigationItems.map(
            (item) => {
              const active =
                isItemActive(item);

              const icon =
                active &&
                item.activeIcon
                  ? item.activeIcon
                  : item.icon;

              return (
                <Link
                  key={item.key}
                  href={getItemHref(
                    item,
                  )}
                  aria-current={
                    active
                      ? "page"
                      : undefined
                  }
                  aria-label={
                    item.label
                  }
                  className={[
                    styles.item,

                    active
                      ? styles.active
                      : "",
                  ]
                    .filter(Boolean)
                    .join(" ")}
                >
                  <span
                    className={
                      styles.activeSurface
                    }
                    aria-hidden="true"
                  />

                  <span
                    className={
                      styles.iconContainer
                    }
                  >
                    <Icon
                      name={icon}
                      size={21}
                      className={
                        styles.icon
                      }
                    />
                  </span>

                  <span
                    className={
                      styles.label
                    }
                  >
                    {item.label}
                  </span>

                  <span
                    className={
                      styles.activeIndicator
                    }
                    aria-hidden="true"
                  />
                </Link>
              );
            },
          )}
        </div>
      </div>
    </nav>
  );
}