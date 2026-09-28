import type {
  Metadata,
} from "next";

import Link from "next/link";

import {
  redirect,
} from "next/navigation";

import {
  Icon,
} from "@/components/ui/Icon";

import {
  getCurrentOrder,
} from "@/lib/commerce/server";

import {
  formatMoney,
} from "@/lib/money/format";

import styles from "@/components/order/css/OrderSuccess.module.css";

export const metadata: Metadata = {
  title:
    "Order Confirmed",

  robots: {
    index: false,
    follow: false,
  },
};

export const dynamic =
  "force-dynamic";

function humanize(
  value: string,
) {
  if (!value) {
    return "";
  }

  const normalized =
    value
      .trim()
      .toLowerCase();

  const specialLabels:
    Record<
      string,
      string
    > = {
      cod:
        "COD",

      cod_pending:
        "COD Pending",

      cash_on_delivery:
        "Cash on Delivery",
    };

  if (
    specialLabels[
      normalized
    ]
  ) {
    return specialLabels[
      normalized
    ];
  }

  return normalized
    .replaceAll(
      "_",
      " ",
    )
    .replace(
      /\b\w/g,
      (
        character,
      ) =>
        character.toUpperCase(),
    );
}

function paymentMessage(
  method: string,
  status: string,
) {
  if (
    method ===
    "cod"
  ) {
    return "Cash on Delivery selected. Payment will be collected according to the delivery process.";
  }

  if (
    status ===
    "pending"
  ) {
    return "Payment is currently pending for this order.";
  }

  return "Your latest payment status is shown here.";
}

export default async function OrderSuccessPage() {
  const order =
    await getCurrentOrder()
      .catch(
        () => null,
      );

  if (
    !order
  ) {
    redirect(
      "/",
    );
  }

  return (
    <main
      className={
        styles.page
      }
    >
      <div
        className={
          styles.shell
        }
      >
        <section
          className={
            styles.hero
          }
        >
          <span
            className={
              styles.icon
            }
            aria-hidden="true"
          >
            <Icon
              name="secureCheckout"
              size={31}
            />
          </span>

          <span
            className={
              styles.eyebrow
            }
          >
            Order confirmed
          </span>

          <h1>
            Thank you.
            Your order is in.
          </h1>

          <p>
            We received your
            order and saved the
            confirmed totals.
          </p>

          <div
            className={
              styles.orderNumber
            }
          >
            <span>
              Order
            </span>

            <strong>
              {
                order.order_number
              }
            </strong>
          </div>
        </section>

        <div
          className={
            styles.statusGrid
          }
        >
          <section
            className={
              styles.card
            }
          >
            <span
              className={
                styles.cardEyebrow
              }
            >
              Order
            </span>

            <h2>
              Current status
            </h2>

            <div
              className={
                styles.rows
              }
            >
              <div>
                <span>
                  Status
                </span>

                <strong>
                  {humanize(
                    order.status,
                  )}
                </strong>
              </div>

              <div>
                <span>
                  Payment
                </span>

                <strong>
                  {humanize(
                    order.payment_status,
                  )}
                </strong>
              </div>

              <div>
                <span>
                  Method
                </span>

                <strong>
                  {humanize(
                    order.payment_method,
                  )}
                </strong>
              </div>

              <div>
                <span>
                  Delivery
                </span>

                <strong>
                  {humanize(
                    order.delivery_method,
                  )}
                </strong>
              </div>
            </div>

            <p
              className={
                styles.note
              }
            >
              {paymentMessage(
                order.payment_method,
                order.payment_status,
              )}
            </p>
          </section>

          <section
            className={
              styles.card
            }
          >
            <span
              className={
                styles.cardEyebrow
              }
            >
              Delivery
            </span>

            <h2>
              Shipping to
            </h2>

            <address
              className={
                styles.address
              }
            >
              <strong>
                {
                  order.customer_name
                }
              </strong>

              <span>
                {
                  order.shipping_address_line1
                }
              </span>

              {order.shipping_address_line2 ? (
                <span>
                  {
                    order.shipping_address_line2
                  }
                </span>
              ) : null}

              <span>
                {
                  order.shipping_area
                }
                ,{" "}
                {
                  order.shipping_city
                }
              </span>

              {order.shipping_postal_code ? (
                <span>
                  {
                    order.shipping_postal_code
                  }
                </span>
              ) : null}

              <span>
                {
                  order.customer_phone
                }
              </span>
            </address>
          </section>
        </div>

        <section
          className={`${styles.card} ${styles.productsCard}`}
        >
          <div
            className={
              styles.productsHeading
            }
          >
            <div>
              <span
                className={
                  styles.cardEyebrow
                }
              >
                Products
              </span>

              <h2>
                What you ordered
              </h2>
            </div>

            <span>
              {
                order.quantity_total
              }{" "}
              units
            </span>
          </div>

          <div
            className={
              styles.products
            }
          >
            {order.items.map(
              (
                item,
              ) => (
                <div
                  key={
                    item.id
                  }
                  className={
                    styles.product
                  }
                >
                  <span>
                    <strong>
                      {
                        item.product_name
                      }
                    </strong>

                    <small>
                      {
                        item.sku
                      }{" "}
                      · MOQ{" "}
                      {
                        item.minimum_order_quantity
                      }
                    </small>
                  </span>

                  <span
                    className={
                      styles.quantity
                    }
                  >
                    {
                      item.quantity
                    }{" "}
                    units
                  </span>

                  <strong
                    className={
                      styles.productTotal
                    }
                  >
                    {formatMoney(
                      item.line_total_amount,
                      item.currency,
                    )}
                  </strong>
                </div>
              ),
            )}
          </div>
        </section>

        <section
          className={
            styles.totalCard
          }
        >
          <div
            className={
              styles.totalRows
            }
          >
            <div>
              <span>
                Subtotal
              </span>

              <strong>
                {formatMoney(
                  order.subtotal_amount,
                  order.currency,
                )}
              </strong>
            </div>

            {order.discount_amount >
            0 ? (
              <div
                className={
                  styles.discount
                }
              >
                <span>
                  Discount
                </span>

                <strong>
                  -
                  {formatMoney(
                    order.discount_amount,
                    order.currency,
                  )}
                </strong>
              </div>
            ) : null}

            <div>
              <span>
                Delivery
              </span>

              <strong>
                {formatMoney(
                  order.shipping_amount,
                  order.currency,
                )}
              </strong>
            </div>
          </div>

          <div
            className={
              styles.grandTotal
            }
          >
            <span>
              Total
            </span>

            <strong>
              {formatMoney(
                order.total_amount,
                order.currency,
              )}
            </strong>
          </div>
        </section>

        <div
          className={
            styles.actions
          }
        >
          <Link
            href="/search"
            className={
              styles.primary
            }
          >
            Continue shopping

            <span
              aria-hidden="true"
            >
              →
            </span>
          </Link>

          <Link
            href="/"
            className={
              styles.secondary
            }
          >
            Back to home
          </Link>
        </div>
      </div>
    </main>
  );
}