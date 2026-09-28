"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import type {
  CheckoutSession,
  DeliveryOption,
  PaymentOption,
} from "@/lib/api/contracts/commerce";
import { formatMoney } from "@/lib/money/format";

type Props = {
  initialCheckout: CheckoutSession;
  deliveryOptions: DeliveryOption[];
  paymentOptions: PaymentOption[];
};

type ErrorPayload = {
  error?: { message?: string };
};

async function getErrorMessage(response: Response) {
  try {
    const payload = (await response.json()) as ErrorPayload;
    return payload.error?.message ?? "Unable to update checkout.";
  } catch {
    return "Unable to update checkout.";
  }
}

function etaLabel(option: DeliveryOption) {
  const minHours = Math.ceil(option.estimated_min_minutes / 60);
  const maxHours = Math.ceil(option.estimated_max_minutes / 60);

  if (maxHours >= 48) {
    const minDays = Math.max(1, Math.ceil(minHours / 24));
    const maxDays = Math.max(minDays, Math.ceil(maxHours / 24));
    return `${minDays}–${maxDays} days`;
  }

  return `${minHours}–${maxHours} hours`;
}

export function CheckoutClient({
  initialCheckout,
  deliveryOptions,
  paymentOptions,
}: Props) {
  const router = useRouter();
  const [checkout, setCheckout] = useState(initialCheckout);
  const [busy, setBusy] = useState(false);
  const [optionBusy, setOptionBusy] = useState(false);
  const [error, setError] = useState("");
  const [promotionCode, setPromotionCode] = useState(initialCheckout.promotion_code ?? "");

  const firstUsablePayment = useMemo(
    () => paymentOptions.find((option) => !option.requires_immediate_payment),
    [paymentOptions],
  );

  const [form, setForm] = useState({
    customer_name: initialCheckout.customer_name ?? "",
    customer_phone: initialCheckout.customer_phone ?? "",
    customer_email: initialCheckout.customer_email ?? "",
    shipping_address_line1: initialCheckout.shipping_address_line1 ?? "",
    shipping_address_line2: initialCheckout.shipping_address_line2 ?? "",
    shipping_city: initialCheckout.shipping_city ?? "",
    shipping_area: initialCheckout.shipping_area ?? "",
    shipping_postal_code: initialCheckout.shipping_postal_code ?? "",
    delivery_method:
      initialCheckout.delivery_method ||
      deliveryOptions.find((option) => option.selected)?.code ||
      deliveryOptions[0]?.code ||
      "",
    payment_method:
      initialCheckout.payment_method || firstUsablePayment?.code || "",
  });

  function setField(name: keyof typeof form, value: string) {
    setForm((current) => ({ ...current, [name]: value }));
    setError("");
  }

  async function patchCheckout(payload: Record<string, string | null>) {
    const response = await fetch("/api/storefront/checkout", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      throw new Error(await getErrorMessage(response));
    }

    const data = (await response.json()) as { data: CheckoutSession };
    setCheckout(data.data);
    return data.data;
  }

  async function selectDelivery(code: string) {
    if (optionBusy) {
      return;
    }

    setField("delivery_method", code);
    setOptionBusy(true);

    try {
      await patchCheckout({ delivery_method: code });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to update delivery.");
    } finally {
      setOptionBusy(false);
    }
  }

  async function applyPromotion() {
    if (optionBusy) {
      return;
    }

    setOptionBusy(true);
    setError("");

    try {
      await patchCheckout({ promotion_code: promotionCode.trim() });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to apply promotion.");
    } finally {
      setOptionBusy(false);
    }
  }

  async function placeOrder() {
    if (busy) {
      return;
    }

    const required = [
      form.customer_name,
      form.customer_phone,
      form.shipping_address_line1,
      form.shipping_city,
      form.shipping_area,
      form.delivery_method,
      form.payment_method,
    ];

    if (required.some((value) => !value.trim())) {
      setError("Complete your contact, shipping, delivery and payment details.");
      return;
    }

    const selectedPayment = paymentOptions.find(
      (option) => option.code === form.payment_method,
    );

    if (selectedPayment?.requires_immediate_payment) {
      setError(
        `${selectedPayment.label} requires the payment-provider handoff. Use an available non-immediate method for this checkout until that provider step is connected.`,
      );
      return;
    }

    setBusy(true);
    setError("");

    try {
      await patchCheckout({
        customer_name: form.customer_name,
        customer_phone: form.customer_phone,
        customer_email: form.customer_email,
        shipping_address_line1: form.shipping_address_line1,
        shipping_address_line2: form.shipping_address_line2,
        shipping_city: form.shipping_city,
        shipping_area: form.shipping_area,
        shipping_postal_code: form.shipping_postal_code,
        delivery_method: form.delivery_method,
        payment_method: form.payment_method,
        promotion_code: promotionCode.trim(),
      });

      const response = await fetch("/api/storefront/order", {
        method: "POST",
      });

      if (!response.ok) {
        throw new Error(await getErrorMessage(response));
      }

      router.push("/order/success");
      router.refresh();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to place order.");
      setBusy(false);
    }
  }

  return (
    <div className="checkout-layout">
      <section className="checkout-form-column">
        <header className="commerce-page-heading checkout-heading">
          <div>
            <span>Secure checkout</span>
            <h1>Complete your order</h1>
          </div>
          <p>{checkout.quantity_total} units</p>
        </header>

        {error && (
          <div className="commerce-alert commerce-alert--error" role="alert">
            {error}
          </div>
        )}

        <section className="checkout-card">
          <div className="checkout-card__heading">
            <span>01</span>
            <div>
              <h2>Contact</h2>
              <p>Who should we contact about this order?</p>
            </div>
          </div>

          <div className="checkout-fields checkout-fields--2">
            <label>
              <span>Name *</span>
              <input
                value={form.customer_name}
                autoComplete="name"
                onChange={(event) => setField("customer_name", event.target.value)}
              />
            </label>
            <label>
              <span>Phone *</span>
              <input
                value={form.customer_phone}
                autoComplete="tel"
                inputMode="tel"
                onChange={(event) => setField("customer_phone", event.target.value)}
              />
            </label>
            <label className="checkout-field-wide">
              <span>Email</span>
              <input
                value={form.customer_email}
                autoComplete="email"
                inputMode="email"
                onChange={(event) => setField("customer_email", event.target.value)}
              />
            </label>
          </div>
        </section>

        <section className="checkout-card">
          <div className="checkout-card__heading">
            <span>02</span>
            <div>
              <h2>Shipping address</h2>
              <p>Required delivery details for your order.</p>
            </div>
          </div>

          <div className="checkout-fields checkout-fields--2">
            <label className="checkout-field-wide">
              <span>Address line 1 *</span>
              <input
                value={form.shipping_address_line1}
                autoComplete="address-line1"
                onChange={(event) => setField("shipping_address_line1", event.target.value)}
              />
            </label>
            <label className="checkout-field-wide">
              <span>Address line 2</span>
              <input
                value={form.shipping_address_line2}
                autoComplete="address-line2"
                onChange={(event) => setField("shipping_address_line2", event.target.value)}
              />
            </label>
            <label>
              <span>City *</span>
              <input
                value={form.shipping_city}
                autoComplete="address-level2"
                onChange={(event) => setField("shipping_city", event.target.value)}
              />
            </label>
            <label>
              <span>Area *</span>
              <input
                value={form.shipping_area}
                onChange={(event) => setField("shipping_area", event.target.value)}
              />
            </label>
            <label>
              <span>Postal code</span>
              <input
                value={form.shipping_postal_code}
                autoComplete="postal-code"
                onChange={(event) => setField("shipping_postal_code", event.target.value)}
              />
            </label>
          </div>
        </section>

        <section className="checkout-card">
          <div className="checkout-card__heading">
            <span>03</span>
            <div>
              <h2>Delivery</h2>
              <p>Options are loaded from the current checkout configuration.</p>
            </div>
          </div>

          <div className="checkout-option-list">
            {deliveryOptions.map((option) => (
              <label
                key={option.code}
                className={
                  form.delivery_method === option.code
                    ? "checkout-option is-selected"
                    : "checkout-option"
                }
              >
                <input
                  type="radio"
                  name="delivery_method"
                  value={option.code}
                  checked={form.delivery_method === option.code}
                  disabled={optionBusy}
                  onChange={() => void selectDelivery(option.code)}
                />
                <span>
                  <strong>{option.label}</strong>
                  <small>{etaLabel(option)}</small>
                </span>
                <b>
                  {option.shipping_amount === 0
                    ? "No charge"
                    : formatMoney(option.shipping_amount, option.currency)}
                </b>
              </label>
            ))}
          </div>
        </section>

        <section className="checkout-card">
          <div className="checkout-card__heading">
            <span>04</span>
            <div>
              <h2>Payment</h2>
              <p>Only enabled backend payment methods are shown.</p>
            </div>
          </div>

          <div className="checkout-option-list">
            {paymentOptions.map((option) => {
              const blocked = option.requires_immediate_payment;

              return (
                <label
                  key={option.code}
                  className={[
                    "checkout-option",
                    form.payment_method === option.code ? "is-selected" : "",
                    blocked ? "is-disabled" : "",
                  ]
                    .filter(Boolean)
                    .join(" ")}
                >
                  <input
                    type="radio"
                    name="payment_method"
                    value={option.code}
                    checked={form.payment_method === option.code}
                    disabled={blocked}
                    onChange={() => setField("payment_method", option.code)}
                  />
                  <span>
                    <strong>{option.label}</strong>
                    <small>
                      {blocked
                        ? "Provider payment handoff will be connected in the payment integration phase"
                        : "Available for this checkout"}
                    </small>
                  </span>
                  <b>{formatMoney(option.payable_amount, option.currency)}</b>
                </label>
              );
            })}
          </div>
        </section>
      </section>

      <aside className="checkout-summary">
        <span className="cart-summary__eyebrow">Order summary</span>
        <h2>{checkout.item_count} {checkout.item_count === 1 ? "product" : "products"}</h2>

        <div className="checkout-summary__items">
          {checkout.items.map((item) => (
            <div key={item.id}>
              <span>
                {item.product_name}
                <small>{item.quantity} × {formatMoney(item.unit_price_amount, item.currency)}</small>
              </span>
              <strong>{formatMoney(item.line_total_amount, item.currency)}</strong>
            </div>
          ))}
        </div>

        <div className="checkout-summary__totals">
          <div>
            <span>Subtotal</span>
            <strong>{formatMoney(checkout.subtotal_amount, checkout.currency)}</strong>
          </div>
          {checkout.discount_amount > 0 && (
            <div className="is-discount">
              <span>Discount</span>
              <strong>-{formatMoney(checkout.discount_amount, checkout.currency)}</strong>
            </div>
          )}
          <div>
            <span>Shipping</span>
            <strong>{formatMoney(checkout.shipping_amount, checkout.currency)}</strong>
          </div>
          <div className="checkout-summary__total">
            <span>Total</span>
            <strong>{formatMoney(checkout.total_amount, checkout.currency)}</strong>
          </div>
        </div>

        <div className="checkout-promo">
          <label htmlFor="promotion-code">Promotion code</label>
          <div>
            <input
              id="promotion-code"
              value={promotionCode}
              onChange={(event) => setPromotionCode(event.target.value)}
              placeholder="Enter code"
            />
            <button type="button" disabled={optionBusy} onClick={() => void applyPromotion()}>
              Apply
            </button>
          </div>
        </div>

        <button
          type="button"
          className="checkout-place-order"
          disabled={busy}
          onClick={() => void placeOrder()}
        >
          <span>{busy ? "Placing order…" : "Place order"}</span>
          <span aria-hidden="true">→</span>
        </button>

        <p className="checkout-summary__note">
          Price, MOQ, stock, discounts and shipping are revalidated by the commerce backend when the order is placed.
        </p>
      </aside>
    </div>
  );
}
