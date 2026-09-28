import {
  Icon,
} from "@/components/ui/Icon";

import styles from "../css/PromoBar.module.css";

const items = [
  {
    icon: "coupon" as const,
    text: "Wholesale prices",
  },
  {
    icon: "orders" as const,
    text: "Bulk discounts",
  },
  {
    icon: "delivery" as const,
    text: "Global shipping",
  },
];

export function PromoBar() {
  return (
    <div
      className={styles.bar}
    >
      <div
        className={`site-container ${styles.inner}`}
      >
        <div
          className={
            styles.messages
          }
        >
          {items.map(
            (
              item,
              index,
            ) => (
              <span
                key={
                  item.text
                }
                className={
                  styles.message
                }
              >
                {index >
                0 ? (
                  <span
                    className={
                      styles.dot
                    }
                  />
                ) : null}

                <Icon
                  name={
                    item.icon
                  }
                  size={12}
                />

                {
                  item.text
                }
              </span>
            ),
          )}
        </div>

        <span
          className={
            styles.support
          }
        >
          <Icon
            name="support"
            size={12}
          />
          Need help? 24/7
          support
        </span>
      </div>
    </div>
  );
}