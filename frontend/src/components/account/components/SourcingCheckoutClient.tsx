"use client";

import Link from "next/link";

import {
  useEffect,
  useMemo,
  useState,
} from "react";

import { Icon } from "@/components/ui/Icon";

import type {
  AccountAddress,
  AccountCustomer,
  AccountDataResponse,
} from "@/lib/api/contracts/account";

import type {
  PlaceSourcingOrderResponse,
  SourcingConfirmation,
  SourcingConfirmationResponse,
} from "@/lib/api/contracts/product-request";

import { formatMoney } from "@/lib/money/format";

import styles from "../css/SourcingCheckout.module.css";

type Props = {
  requestID: string;
};

type FormState = {
  customer_name: string;
  customer_phone: string;
  customer_email: string;

  shipping_address_line1: string;
  shipping_address_line2: string;
  shipping_city: string;
  shipping_area: string;
  shipping_postal_code: string;

  payment_method: string;
};

const paymentChoices = [
  {
    code: "cod",
    label: "Cash on Delivery",
    note: "Pay when the sourced order is delivered.",
  },
  {
    code: "bkash",
    label: "bKash",
    note: "Continue to secure online payment after the order is created.",
  },
  {
    code: "nagad",
    label: "Nagad",
    note: "Continue to secure online payment after the order is created.",
  },
  {
    code: "rocket",
    label: "Rocket",
    note: "Continue to secure online payment after the order is created.",
  },
  {
    code: "bank_transfer",
    label: "Bank Transfer",
    note: "Continue with bank-transfer payment instructions.",
  },
];

function initialForm(): FormState {
  return {
    customer_name: "",
    customer_phone: "",
    customer_email: "",

    shipping_address_line1: "",
    shipping_address_line2: "",
    shipping_city: "",
    shipping_area: "",
    shipping_postal_code: "",

    payment_method: "cod",
  };
}

async function readError(
  response: Response,
  fallback: string,
) {
  try {
    const payload = (await response.json()) as {
      error?: {
        message?: string;
      };
    };

    return (
      payload.error?.message?.trim() ||
      fallback
    );
  } catch {
    return fallback;
  }
}

type UnknownRecord =
  Record<string, unknown>;

function findPaymentURL(
  value: unknown,
  depth = 0,
): string | null {
  if (
    depth > 4 ||
    !value ||
    typeof value !== "object"
  ) {
    return null;
  }

  const record = value as UnknownRecord;

  const preferred = [
    "redirect_url",
    "payment_url",
    "checkout_url",
    "url",
  ];

  for (const key of preferred) {
    const candidate = record[key];

    if (
      typeof candidate === "string" &&
      /^https?:\/\//i.test(candidate)
    ) {
      return candidate;
    }
  }

  for (const nested of Object.values(record)) {
    const candidate = findPaymentURL(
      nested,
      depth + 1,
    );

    if (candidate) {
      return candidate;
    }
  }

  return null;
}

export function SourcingCheckoutClient({
  requestID,
}: Props) {
  const [
    confirmation,
    setConfirmation,
  ] =
    useState<SourcingConfirmation | null>(
      null,
    );

  const [addresses, setAddresses] =
    useState<AccountAddress[]>([]);

  const [
    selectedAddressID,
    setSelectedAddressID,
  ] = useState("");

  const [form, setForm] =
    useState<FormState>(initialForm);

  const [loading, setLoading] =
    useState(true);

  const [placing, setPlacing] =
    useState(false);

  const [error, setError] =
    useState("");

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError("");

      try {
        const encoded =
          encodeURIComponent(requestID);

        const [
          confirmationResponse,
          customerResponse,
          addressesResponse,
        ] = await Promise.all([
          fetch(
            `/api/storefront/account/requests/${encoded}/confirmation`,
            {
              cache: "no-store",
            },
          ),

          fetch(
            "/api/storefront/account/me",
            {
              cache: "no-store",
            },
          ),

          fetch(
            "/api/storefront/account/addresses",
            {
              cache: "no-store",
            },
          ),
        ]);

        if (!confirmationResponse.ok) {
          throw new Error(
            await readError(
              confirmationResponse,
              "This sourcing agreement is not ready for checkout.",
            ),
          );
        }

        if (!customerResponse.ok) {
          throw new Error(
            await readError(
              customerResponse,
              "Unable to load your account details.",
            ),
          );
        }

        const confirmationPayload =
          (await confirmationResponse.json()) as SourcingConfirmationResponse;

        const customerPayload =
          (await customerResponse.json()) as AccountDataResponse<AccountCustomer>;

        let addressItems: AccountAddress[] =
          [];

        if (addressesResponse.ok) {
          const addressPayload =
            (await addressesResponse.json()) as AccountDataResponse<
              AccountAddress[]
            >;

          addressItems = Array.isArray(
            addressPayload.data,
          )
            ? addressPayload.data
            : [];
        }

        if (cancelled) {
          return;
        }

        setConfirmation(
          confirmationPayload.data,
        );

        setAddresses(addressItems);

        const preferredAddress =
          addressItems.find(
            (address) =>
              address.is_default,
          ) ??
          addressItems[0] ??
          null;

        if (preferredAddress) {
          setSelectedAddressID(
            preferredAddress.id,
          );
        }

        setForm((current) => ({
          ...current,

          customer_name:
            preferredAddress?.recipient_name ||
            customerPayload.data.full_name ||
            "",

          customer_phone:
            preferredAddress?.phone ||
            customerPayload.data.phone ||
            "",

          customer_email:
            customerPayload.data.email ||
            "",

          shipping_address_line1:
            preferredAddress?.address_line1 ||
            "",

          shipping_address_line2:
            preferredAddress?.address_line2 ||
            "",

          shipping_city:
            preferredAddress?.city || "",

          shipping_area:
            preferredAddress?.area || "",

          shipping_postal_code:
            preferredAddress?.postal_code ||
            "",
        }));
      } catch (caught) {
        if (!cancelled) {
          setError(
            caught instanceof Error
              ? caught.message
              : "Unable to prepare sourcing checkout.",
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [requestID]);

  const selectedPayment = useMemo(
    () =>
      paymentChoices.find(
        (choice) =>
          choice.code ===
          form.payment_method,
      ) ?? null,
    [form.payment_method],
  );

  function chooseAddress(
    address: AccountAddress,
  ) {
    setSelectedAddressID(address.id);

    setForm((current) => ({
      ...current,

      customer_name:
        address.recipient_name ||
        current.customer_name,

      customer_phone:
        address.phone ||
        current.customer_phone,

      shipping_address_line1:
        address.address_line1,

      shipping_address_line2:
        address.address_line2 || "",

      shipping_city:
        address.city,

      shipping_area:
        address.area,

      shipping_postal_code:
        address.postal_code || "",
    }));

    setError("");
  }

  function updateField<
    K extends keyof FormState,
  >(
    key: K,
    value: FormState[K],
  ) {
    setForm((current) => ({
      ...current,
      [key]: value,
    }));

    setError("");
  }

  async function placeOrder() {
    if (
      !confirmation ||
      placing
    ) {
      return;
    }

    if (
      !form.customer_name.trim() ||
      !form.customer_phone.trim() ||
      !form.shipping_address_line1.trim() ||
      !form.shipping_city.trim() ||
      !form.shipping_area.trim() ||
      !form.payment_method.trim()
    ) {
      setError(
        "Complete the required contact, delivery and payment fields.",
      );
      return;
    }

    setPlacing(true);
    setError("");

    try {
      /*
       * IMPORTANT:
       *
       * We send ONLY delivery/contact/payment
       * information.
       *
       * Product name, quantity, MOQ, unit price,
       * shipping price, currency and total are
       * intentionally NOT sent from the browser.
       *
       * The backend reads those immutable values
       * from the finalized sourcing confirmation.
       */
      const response = await fetch(
        `/api/storefront/account/requests/${encodeURIComponent(
          requestID,
        )}/order`,
        {
          method: "POST",

          headers: {
            "Content-Type":
              "application/json",
          },

          body: JSON.stringify({
            customer_name:
              form.customer_name.trim(),

            customer_phone:
              form.customer_phone.trim(),

            customer_email:
              form.customer_email.trim(),

            shipping_address_line1:
              form.shipping_address_line1.trim(),

            shipping_address_line2:
              form.shipping_address_line2.trim(),

            shipping_city:
              form.shipping_city.trim(),

            shipping_area:
              form.shipping_area.trim(),

            shipping_postal_code:
              form.shipping_postal_code.trim(),

            payment_method:
              form.payment_method,
          }),
        },
      );

      if (!response.ok) {
        throw new Error(
          await readError(
            response,
            "Unable to create your sourcing order.",
          ),
        );
      }

      const payload =
        (await response.json()) as PlaceSourcingOrderResponse;

      const orderID =
        payload.data.order.id;

      if (
        form.payment_method !== "cod"
      ) {
        const paymentResponse =
          await fetch(
            `/api/storefront/order/${encodeURIComponent(
              orderID,
            )}/payment/initiate`,
            {
              method: "POST",
            },
          );

        if (!paymentResponse.ok) {
          throw new Error(
            await readError(
              paymentResponse,
              "Your sourcing order was created, but payment could not be started. Open the order to continue payment.",
            ),
          );
        }

        const paymentPayload: unknown =
          await paymentResponse.json();

        const redirectURL =
          findPaymentURL(
            paymentPayload,
          );

        if (redirectURL) {
          window.location.assign(
            redirectURL,
          );
          return;
        }
      }

      window.location.assign(
        `/account/orders/${encodeURIComponent(
          orderID,
        )}`,
      );
    } catch (caught) {
      setError(
        caught instanceof Error
          ? caught.message
          : "Unable to create your sourcing order.",
      );

      setPlacing(false);
    }
  }

  if (loading) {
    return (
      <main className={styles.page}>
        <div className={styles.shell}>
          <div
            className={styles.loadingCard}
          >
            Preparing your sourcing
            checkout…
          </div>
        </div>
      </main>
    );
  }

  if (!confirmation) {
    return (
      <main className={styles.page}>
        <div className={styles.shell}>
          <section
            className={styles.emptyCard}
          >
            <span
              className={styles.eyebrow}
            >
              Sourcing checkout
            </span>

            <h1>
              Checkout is not available yet.
            </h1>

            <p>
              {error ||
                "The sourcing agreement is not ready to become an order."}
            </p>

            <Link
              className={
                styles.secondaryButton
              }
              href="/account/request"
            >
              Back to product requests
            </Link>
          </section>
        </div>
      </main>
    );
  }

  if (
    confirmation.created_order_id ||
    confirmation.status ===
      "order_created"
  ) {
    return (
      <main className={styles.page}>
        <div className={styles.shell}>
          <section
            className={styles.emptyCard}
          >
            <span
              className={styles.eyebrow}
            >
              Sourcing checkout
            </span>

            <h1>
              This agreement already has
              an order.
            </h1>

            <p>
              Your finalized sourcing terms
              have already been converted
              into a customer order.
            </p>

            {confirmation.created_order_id ? (
              <Link
                className={
                  styles.primaryButton
                }
                href={`/account/orders/${confirmation.created_order_id}`}
              >
                View order

                <Icon
                  name="chevronRight"
                  size={14}
                />
              </Link>
            ) : (
              <Link
                className={
                  styles.secondaryButton
                }
                href="/account/orders"
              >
                View my orders
              </Link>
            )}
          </section>
        </div>
      </main>
    );
  }

  return (
    <main className={styles.page}>
      <div
        className={styles.ambient}
        aria-hidden="true"
      >
        <span
          className={styles.orbOne}
        />

        <span
          className={styles.orbTwo}
        />
      </div>

      <div className={styles.shell}>
        <header className={styles.header}>
          <div>
            <span
              className={styles.eyebrow}
            >
              Final sourcing checkout
            </span>

            <h1>
              Confirm delivery and payment.
            </h1>

            <p>
              Product, quantity and
              commercial pricing are locked
              to the finalized sourcing
              agreement and cannot be edited
              here.
            </p>
          </div>

          <Link
            className={styles.backLink}
            href="/account/request"
          >
            <Icon
              name="arrowLeft"
              size={15}
            />

            Product requests
          </Link>
        </header>

        {error ? (
          <div
            className={styles.error}
            role="alert"
          >
            {error}
          </div>
        ) : null}

        <div className={styles.layout}>
          <div
            className={styles.mainColumn}
          >
            <section
              className={styles.glassCard}
              aria-labelledby="address-heading"
            >
              <div
                className={
                  styles.sectionHead
                }
              >
                <div>
                  <span
                    className={
                      styles.kicker
                    }
                  >
                    Delivery
                  </span>

                  <h2 id="address-heading">
                    Contact & address
                  </h2>
                </div>

                <Link href="/account/addresses">
                  Manage saved addresses
                </Link>
              </div>

              {addresses.length > 0 ? (
                <div
                  className={
                    styles.addressGrid
                  }
                >
                  {addresses.map(
                    (address) => {
                      const selected =
                        selectedAddressID ===
                        address.id;

                      return (
                        <button
                          type="button"
                          key={address.id}
                          className={`${styles.addressCard} ${
                            selected
                              ? styles.selectedCard
                              : ""
                          }`}
                          onClick={() =>
                            chooseAddress(
                              address,
                            )
                          }
                        >
                          <span
                            className={
                              styles.addressTopline
                            }
                          >
                            <strong>
                              {address.label}
                            </strong>

                            {address.is_default ? (
                              <small>
                                Default
                              </small>
                            ) : null}
                          </span>

                          <span>
                            {
                              address.recipient_name
                            }
                          </span>

                          <span>
                            {address.phone}
                          </span>

                          <span>
                            {
                              address.address_line1
                            }
                          </span>

                          <span>
                            {[
                              address.area,
                              address.city,
                              address.postal_code,
                            ]
                              .filter(Boolean)
                              .join(", ")}
                          </span>
                        </button>
                      );
                    },
                  )}
                </div>
              ) : (
                <p
                  className={
                    styles.inlineNote
                  }
                >
                  You do not have a saved
                  delivery address yet. Enter
                  one below or manage your
                  account addresses.
                </p>
              )}

              <div
                className={styles.formGrid}
              >
                <label>
                  <span>Name</span>

                  <input
                    value={
                      form.customer_name
                    }
                    onChange={(event) =>
                      updateField(
                        "customer_name",
                        event.target.value,
                      )
                    }
                    autoComplete="name"
                  />
                </label>

                <label>
                  <span>Phone</span>

                  <input
                    value={
                      form.customer_phone
                    }
                    onChange={(event) =>
                      updateField(
                        "customer_phone",
                        event.target.value,
                      )
                    }
                    autoComplete="tel"
                    inputMode="tel"
                  />
                </label>

                <label
                  className={
                    styles.fullField
                  }
                >
                  <span>
                    Email{" "}
                    <small>Optional</small>
                  </span>

                  <input
                    value={
                      form.customer_email
                    }
                    onChange={(event) =>
                      updateField(
                        "customer_email",
                        event.target.value,
                      )
                    }
                    autoComplete="email"
                    type="email"
                  />
                </label>

                <label
                  className={
                    styles.fullField
                  }
                >
                  <span>
                    Address line 1
                  </span>

                  <input
                    value={
                      form.shipping_address_line1
                    }
                    onChange={(event) =>
                      updateField(
                        "shipping_address_line1",
                        event.target.value,
                      )
                    }
                    autoComplete="address-line1"
                  />
                </label>

                <label
                  className={
                    styles.fullField
                  }
                >
                  <span>
                    Address line 2{" "}
                    <small>Optional</small>
                  </span>

                  <input
                    value={
                      form.shipping_address_line2
                    }
                    onChange={(event) =>
                      updateField(
                        "shipping_address_line2",
                        event.target.value,
                      )
                    }
                    autoComplete="address-line2"
                  />
                </label>

                <label>
                  <span>Area</span>

                  <input
                    value={
                      form.shipping_area
                    }
                    onChange={(event) =>
                      updateField(
                        "shipping_area",
                        event.target.value,
                      )
                    }
                  />
                </label>

                <label>
                  <span>City</span>

                  <input
                    value={
                      form.shipping_city
                    }
                    onChange={(event) =>
                      updateField(
                        "shipping_city",
                        event.target.value,
                      )
                    }
                    autoComplete="address-level2"
                  />
                </label>

                <label>
                  <span>
                    Postal code{" "}
                    <small>Optional</small>
                  </span>

                  <input
                    value={
                      form.shipping_postal_code
                    }
                    onChange={(event) =>
                      updateField(
                        "shipping_postal_code",
                        event.target.value,
                      )
                    }
                    autoComplete="postal-code"
                  />
                </label>
              </div>
            </section>

            <section
              className={styles.glassCard}
              aria-labelledby="payment-heading"
            >
              <div
                className={
                  styles.sectionHead
                }
              >
                <div>
                  <span
                    className={
                      styles.kicker
                    }
                  >
                    Payment
                  </span>

                  <h2 id="payment-heading">
                    Choose a payment method
                  </h2>
                </div>
              </div>

              <div
                className={
                  styles.paymentGrid
                }
              >
                {paymentChoices.map(
                  (choice) => {
                    const selected =
                      form.payment_method ===
                      choice.code;

                    return (
                      <button
                        key={choice.code}
                        type="button"
                        className={`${styles.paymentCard} ${
                          selected
                            ? styles.selectedCard
                            : ""
                        }`}
                        onClick={() =>
                          updateField(
                            "payment_method",
                            choice.code,
                          )
                        }
                      >
                        <span
                          className={
                            styles.radio
                          }
                          aria-hidden="true"
                        >
                          {selected ? (
                            <span />
                          ) : null}
                        </span>

                        <span>
                          <strong>
                            {choice.label}
                          </strong>

                          <small>
                            {choice.note}
                          </small>
                        </span>
                      </button>
                    );
                  },
                )}
              </div>

              <p
                className={
                  styles.inlineNote
                }
              >
                The backend remains the
                authority for payment-method
                availability. If a selected
                method is unavailable, the
                sourcing agreement remains
                intact.
              </p>
            </section>
          </div>

          <aside
            className={styles.summaryCard}
          >
            <span
              className={styles.kicker}
            >
              Locked agreement
            </span>

            <h2>
              {
                confirmation.accepted_product_name
              }
            </h2>

            <p>
              These values come directly
              from the finalized sourcing
              confirmation.
            </p>

            <dl
              className={
                styles.summaryFacts
              }
            >
              <div>
                <dt>Quantity</dt>

                <dd>
                  {confirmation.quantity.toLocaleString()}
                </dd>
              </div>

              <div>
                <dt>MOQ</dt>

                <dd>
                  {confirmation.minimum_order_quantity.toLocaleString()}
                </dd>
              </div>

              <div>
                <dt>Unit price</dt>

                <dd>
                  {formatMoney(
                    confirmation.unit_price_snapshot,
                    confirmation.currency,
                  )}
                </dd>
              </div>

              <div>
                <dt>Shipping</dt>

                <dd>
                  {formatMoney(
                    confirmation.shipping_price_snapshot,
                    confirmation.currency,
                  )}
                </dd>
              </div>

              <div
                className={styles.totalRow}
              >
                <dt>Total</dt>

                <dd>
                  {formatMoney(
                    confirmation.total_amount,
                    confirmation.currency,
                  )}
                </dd>
              </div>
            </dl>

            <div
              className={
                styles.selectionSummary
              }
            >
              <span>Payment</span>

              <strong>
                {selectedPayment?.label ??
                  "—"}
              </strong>
            </div>

            <button
              type="button"
              className={
                styles.placeButton
              }
              disabled={placing}
              onClick={() =>
                void placeOrder()
              }
            >
              {placing
                ? "Creating sourcing order…"
                : "Place sourcing order"}

              {!placing ? (
                <Icon
                  name="chevronRight"
                  size={15}
                />
              ) : null}
            </button>

            <p
              className={
                styles.securityNote
              }
            >
              <Icon
                name="secureCheckout"
                size={14}
              />

              Commercial terms are
              server-owned and are never
              resent from this form.
            </p>
          </aside>
        </div>
      </div>
    </main>
  );
}