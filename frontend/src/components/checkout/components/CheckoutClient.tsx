"use client";
/* eslint-disable @next/next/no-img-element */
// Location: src/components/checkout/components/CheckoutClient.tsx

import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  type FormEvent,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import type {
  AccountAddress,
  AccountCustomer,
  AccountDataResponse,
} from "@/lib/api/contracts/account";
import type {
  CheckoutSession,
  DeliveryOption,
  PaymentOption,
  PlaceOrderResponse,
} from "@/lib/api/contracts/commerce";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/Checkout.module.css";

type Props = {
  customer: AccountCustomer;
  initialAddresses: AccountAddress[];
  initialCheckout: CheckoutSession;
  deliveryOptions: DeliveryOption[];
  paymentOptions: PaymentOption[];
};

type Stage = "checkout" | "review" | "payment" | "confirmed";

type AddressDraft = {
  label: string;
  recipient_name: string;
  phone: string;
  address_line1: string;
  address_line2: string;
  city: string;
  area: string;
  postal_code: string;
  is_default: boolean;
};

type ErrorPayload = {
  error?: {
    message?: string;
  };
};

type UnknownRecord = Record<string, unknown>;

type DeliveryDisplayOption = {
  key: string;
  label: string;
  description: string;
  backendOption: DeliveryOption | null;
};

type PaymentDisplayOption = {
  key: string;
  label: string;
  description: string;
  mark: string;
  backendOption: PaymentOption | null;
};

const DELIVERY_SLOTS = [
  {
    key: "standard",
    label: "Standard Delivery",
    description: "Standard doorstep delivery",
    aliases: ["standard", "regular", "courier"],
  },
  {
    key: "express",
    label: "Express Delivery",
    description: "Faster delivery option",
    aliases: ["express", "priority"],
  },
  {
    key: "self-pickup",
    label: "Self Pickup",
    description: "Collect from the configured pickup location",
    aliases: ["self_pickup", "self-pickup", "pickup", "collection"],
  },
] as const;

const PAYMENT_SLOTS = [
  { key: "cod", label: "Cash on Delivery", description: "Pay when you receive your order", mark: "COD", aliases: ["cod", "cash_on_delivery", "cash-on-delivery"] },
  { key: "bkash", label: "bKash", description: "Pay securely with bKash", mark: "bK", aliases: ["bkash"] },
  { key: "nagad", label: "Nagad", description: "Pay securely with Nagad", mark: "N", aliases: ["nagad"] },
  { key: "rocket", label: "Rocket", description: "Pay securely with Rocket", mark: "R", aliases: ["rocket"] },
  { key: "bank-transfer", label: "Bank Transfer", description: "Pay by bank transfer", mark: "BANK", aliases: ["bank_transfer", "bank-transfer", "bank"] },
] as const;

const PAYMENT_BRAND_ICON_URLS: Record<string, string> = {
  bkash: "https://images.seeklogo.com/logo-png/47/1/bkash-logo-png_seeklogo-471379.png",
  nagad: "https://static.freepnglogo.com/images/all_img/1725618513nagad-logo.png",
  rocket: "https://upload.wikimedia.org/wikipedia/commons/4/45/Rocket_mobile_banking_logo.svg",
};

function normalizedOptionValue(value: string) {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, "_");
}

function optionMatches(
  code: string,
  label: string,
  aliases: readonly string[],
) {
  const normalizedCode = normalizedOptionValue(code);
  const normalizedLabel = normalizedOptionValue(label);

  return aliases.some((alias) => {
    const normalizedAlias = normalizedOptionValue(alias);
    return normalizedCode === normalizedAlias || normalizedLabel.includes(normalizedAlias);
  });
}


const EMPTY_ADDRESS: AddressDraft = {
  label: "Home",
  recipient_name: "",
  phone: "",
  address_line1: "",
  address_line2: "",
  city: "",
  area: "",
  postal_code: "",
  is_default: false,
};

function Icon({ name }: { name: "pin" | "truck" | "wallet" | "shield" | "check" | "plus" | "arrow" | "document" }) {
  const common = {
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: 1.8,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    "aria-hidden": true,
  };

  if (name === "pin") {
    return <svg {...common}><path d="M20 10c0 5-8 11-8 11S4 15 4 10a8 8 0 1 1 16 0Z"/><circle cx="12" cy="10" r="2.5"/></svg>;
  }
  if (name === "truck") {
    return <svg {...common}><path d="M3 6h11v10H3z"/><path d="M14 10h4l3 3v3h-7z"/><circle cx="7" cy="18" r="2"/><circle cx="17" cy="18" r="2"/></svg>;
  }
  if (name === "wallet") {
    return <svg {...common}><path d="M4 7h15a2 2 0 0 1 2 2v9H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h13"/><path d="M16 11h5v4h-5a2 2 0 0 1 0-4Z"/></svg>;
  }
  if (name === "shield") {
    return <svg {...common}><path d="m12 3 7 3v5c0 4.7-2.7 8-7 10-4.3-2-7-5.3-7-10V6l7-3Z"/><path d="m9 12 2 2 4-4"/></svg>;
  }
  if (name === "check") {
    return <svg {...common}><path d="m5 12 4 4L19 6"/></svg>;
  }
  if (name === "plus") {
    return <svg {...common}><path d="M12 5v14M5 12h14"/></svg>;
  }
  if (name === "document") {
    return <svg {...common}><path d="M6 3h9l3 3v15H6z"/><path d="M15 3v4h4M9 12h6M9 16h6"/></svg>;
  }
  return <svg {...common}><path d="M5 12h14M14 7l5 5-5 5"/></svg>;
}

async function errorMessage(response: Response, fallback: string) {
  try {
    const payload = (await response.json()) as ErrorPayload;
    return payload.error?.message?.trim() || fallback;
  } catch {
    return fallback;
  }
}

function etaLabel(option: DeliveryOption) {
  const minHours = Math.max(1, Math.ceil(option.estimated_min_minutes / 60));
  const maxHours = Math.max(minHours, Math.ceil(option.estimated_max_minutes / 60));

  if (maxHours >= 48) {
    const minDays = Math.max(1, Math.ceil(minHours / 24));
    const maxDays = Math.max(minDays, Math.ceil(maxHours / 24));
    return `${minDays}–${maxDays} working days`;
  }

  return `${minHours}–${maxHours} hours`;
}

function paymentDescription(option: PaymentOption) {
  const code = option.code.toLowerCase();

  if (code === "cod") return "Pay when you receive your order";
  if (code.includes("bkash")) return "Pay securely with bKash";
  if (code.includes("nagad")) return "Pay securely with Nagad";
  if (code.includes("rocket")) return "Pay securely with Rocket";
  if (code.includes("bank")) return "Pay by bank transfer";
  if (code.includes("card") || code.includes("ssl")) return "Pay securely with card";
  if (code.includes("upay")) return "Pay securely with Upay";

  return option.requires_immediate_payment
    ? "Continue to the payment provider after confirmation"
    : "Complete payment using this method";
}

function paymentMark(option: PaymentOption) {
  const code = option.code.toLowerCase();
  if (code === "cod") return "COD";
  if (code.includes("bkash")) return "bK";
  if (code.includes("nagad")) return "N";
  if (code.includes("rocket")) return "R";
  if (code.includes("bank")) return "BANK";
  if (code.includes("ssl")) return "SSL";
  if (code.includes("card")) return "CARD";
  if (code.includes("upay")) return "U";
  return option.label.slice(0, 4).toUpperCase();
}

function PaymentMethodIcon({ option }: { option: PaymentDisplayOption }) {
  const brandURL = PAYMENT_BRAND_ICON_URLS[option.key];

  if (brandURL) {
    return (
      <img
        className={styles.paymentLogo}
        src={brandURL}
        alt=""
        aria-hidden="true"
        width={72}
        height={40}
        loading="lazy"
        decoding="async"
      />
    );
  }

  if (option.key === "cod") {
    return (
      <svg className={styles.paymentGenericIcon} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true">
        <rect x="3" y="6" width="18" height="12" rx="2" />
        <path d="M7 10h4M7 14h3M16 9.5c1.3 0 2.3 1 2.3 2.5s-1 2.5-2.3 2.5-2.3-1-2.3-2.5 1-2.5 2.3-2.5Z" />
      </svg>
    );
  }

  if (option.key === "bank-transfer") {
    return (
      <svg className={styles.paymentGenericIcon} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="m3 9 9-5 9 5" />
        <path d="M5 10v7M9 10v7M15 10v7M19 10v7M3 20h18" />
      </svg>
    );
  }

  return <span className={styles.paymentFallbackMark}>{option.mark}</span>;
}

function findPaymentURL(value: unknown, depth = 0): string | null {
  if (depth > 4 || !value || typeof value !== "object") return null;

  const record = value as UnknownRecord;
  const preferred = ["redirect_url", "payment_url", "checkout_url", "url"];

  for (const key of preferred) {
    const candidate = record[key];
    if (typeof candidate === "string" && /^https?:\/\//i.test(candidate)) {
      return candidate;
    }
  }

  for (const nested of Object.values(record)) {
    const candidate = findPaymentURL(nested, depth + 1);
    if (candidate) return candidate;
  }

  return null;
}

function addressMatchesCheckout(address: AccountAddress, checkout: CheckoutSession) {
  return Boolean(
    checkout.shipping_address_line1 &&
      address.address_line1 === checkout.shipping_address_line1 &&
      address.city === checkout.shipping_city &&
      address.area === checkout.shipping_area,
  );
}

export function CheckoutClient({
  customer,
  initialAddresses,
  initialCheckout,
  deliveryOptions,
  paymentOptions,
}: Props) {
  const router = useRouter();
  const termsDialogRef = useRef<HTMLDialogElement>(null);

  const [stage, setStage] = useState<Stage>("checkout");
  const [checkout, setCheckout] = useState(initialCheckout);
  const [addresses, setAddresses] = useState(initialAddresses);
  const [showAddressForm, setShowAddressForm] = useState(false);
  const [addressDraft, setAddressDraft] = useState<AddressDraft>({
    ...EMPTY_ADDRESS,
    recipient_name: customer.full_name,
    phone: customer.phone,
  });
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [termsAccepted, setTermsAccepted] = useState(false);
  const [placedOrder, setPlacedOrder] = useState<PlaceOrderResponse["data"] | null>(null);
  const [paymentMessage, setPaymentMessage] = useState("");

  const initialAddressID = useMemo(() => {
    return (
      initialAddresses.find((address) => addressMatchesCheckout(address, initialCheckout))?.id ??
      initialAddresses.find((address) => address.is_default)?.id ??
      initialAddresses[0]?.id ??
      ""
    );
  }, [initialAddresses, initialCheckout]);

  const [selectedAddressID, setSelectedAddressID] = useState(initialAddressID);

  const selectedAddress = useMemo(
    () => addresses.find((address) => address.id === selectedAddressID) ?? null,
    [addresses, selectedAddressID],
  );

  const firstPayment = useMemo(
    () =>
      paymentOptions.find((option) => option.code === initialCheckout.payment_method) ??
      paymentOptions.find((option) => !option.requires_immediate_payment) ??
      paymentOptions[0],
    [initialCheckout.payment_method, paymentOptions],
  );

  const [deliveryCode, setDeliveryCode] = useState(
    initialCheckout.delivery_method ||
      deliveryOptions.find((option) => option.selected)?.code ||
      deliveryOptions[0]?.code ||
      "",
  );
  const [paymentCode, setPaymentCode] = useState(
    initialCheckout.payment_method || firstPayment?.code || "",
  );

  const selectedDelivery = useMemo(
    () => deliveryOptions.find((option) => option.code === deliveryCode) ?? null,
    [deliveryCode, deliveryOptions],
  );
  const selectedPayment = useMemo(
    () => paymentOptions.find((option) => option.code === paymentCode) ?? null,
    [paymentCode, paymentOptions],
  );

  const displayDeliveryOptions = useMemo<DeliveryDisplayOption[]>(() => {
    const usedCodes = new Set<string>();

    const planned = DELIVERY_SLOTS.map((slot) => {
      const backendOption =
        deliveryOptions.find((option) =>
          !usedCodes.has(option.code) &&
          optionMatches(option.code, option.label, slot.aliases),
        ) ?? null;

      if (backendOption) {
        usedCodes.add(backendOption.code);
      }

      return {
        key: slot.key,
        label: backendOption?.label ?? slot.label,
        description: backendOption ? etaLabel(backendOption) : slot.description,
        backendOption,
      };
    });

    const extras = deliveryOptions
      .filter((option) => !usedCodes.has(option.code))
      .map((option) => ({
        key: `backend:${option.code}`,
        label: option.label,
        description: etaLabel(option),
        backendOption: option,
      }));

    return [...planned, ...extras];
  }, [deliveryOptions]);

  const displayPaymentOptions = useMemo<PaymentDisplayOption[]>(() => {
    const usedCodes = new Set<string>();

    const planned = PAYMENT_SLOTS.map((slot) => {
      const backendOption =
        paymentOptions.find((option) =>
          !usedCodes.has(option.code) &&
          optionMatches(option.code, option.label, slot.aliases),
        ) ?? null;

      if (backendOption) {
        usedCodes.add(backendOption.code);
      }

      return {
        key: slot.key,
        label: backendOption?.label ?? slot.label,
        description: backendOption ? paymentDescription(backendOption) : slot.description,
        mark: backendOption ? paymentMark(backendOption) : slot.mark,
        backendOption,
      };
    });

    const extras = paymentOptions
      .filter((option) => !usedCodes.has(option.code))
      .map((option) => ({
        key: `backend:${option.code}`,
        label: option.label,
        description: paymentDescription(option),
        mark: paymentMark(option),
        backendOption: option,
      }));

    return [...planned, ...extras];
  }, [paymentOptions]);

  useEffect(() => {
    if (addresses.length > 0) return;

    let cancelled = false;

    void fetch("/api/storefront/account/addresses", { cache: "no-store" })
      .then(async (response) => {
        if (!response.ok) return null;
        return (await response.json()) as AccountDataResponse<AccountAddress[]>;
      })
      .then((payload) => {
        if (cancelled || !payload?.data?.length) return;
        setAddresses(payload.data);
        const preferred = payload.data.find((address) => address.is_default) ?? payload.data[0];
        setSelectedAddressID((current) => current || preferred.id);
      })
      .catch(() => undefined);

    return () => {
      cancelled = true;
    };
  }, [addresses.length]);

  function clearError() {
    setError("");
  }

  async function patchCheckout(payload: Record<string, string | null>) {
    const response = await fetch("/api/storefront/checkout", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    if (response.status === 401) {
      router.replace("/account/sign-in?next=%2Fcheckout");
      throw new Error("Authentication required.");
    }

    if (!response.ok) {
      throw new Error(await errorMessage(response, "Unable to update checkout."));
    }

    const payloadResult = (await response.json()) as { data: CheckoutSession };
    setCheckout(payloadResult.data);
    return payloadResult.data;
  }

  async function chooseAddress(address: AccountAddress) {
    if (busy) return;

    setSelectedAddressID(address.id);
    clearError();
    setBusy(`address:${address.id}`);

    try {
      await patchCheckout({
        customer_name: address.recipient_name || customer.full_name,
        customer_phone: address.phone || customer.phone,
        customer_email: customer.email ?? "",
        shipping_address_line1: address.address_line1,
        shipping_address_line2: address.address_line2 ?? "",
        shipping_city: address.city,
        shipping_area: address.area,
        shipping_postal_code: address.postal_code ?? "",
      });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to select address.");
    } finally {
      setBusy("");
    }
  }

  async function addAddress(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;

    if (
      !addressDraft.label.trim() ||
      !addressDraft.recipient_name.trim() ||
      !addressDraft.phone.trim() ||
      !addressDraft.address_line1.trim() ||
      !addressDraft.city.trim() ||
      !addressDraft.area.trim()
    ) {
      setError("Complete the required address fields.");
      return;
    }

    setBusy("new-address");
    clearError();

    try {
      const response = await fetch("/api/storefront/account/addresses", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(addressDraft),
      });

      if (response.status === 401) {
        router.replace("/account/sign-in?next=%2Fcheckout");
        return;
      }

      if (!response.ok) {
        throw new Error(await errorMessage(response, "Unable to save address."));
      }

      const payload = (await response.json()) as AccountDataResponse<AccountAddress>;
      const address = payload.data;
      setAddresses((current) => {
        const normalized = address.is_default
          ? current.map((item) => ({ ...item, is_default: false }))
          : current;
        return [address, ...normalized.filter((item) => item.id !== address.id)];
      });
      setSelectedAddressID(address.id);
      setShowAddressForm(false);
      setAddressDraft({
        ...EMPTY_ADDRESS,
        recipient_name: customer.full_name,
        phone: customer.phone,
      });
      await patchCheckout({
        customer_name: address.recipient_name || customer.full_name,
        customer_phone: address.phone || customer.phone,
        customer_email: customer.email ?? "",
        shipping_address_line1: address.address_line1,
        shipping_address_line2: address.address_line2 ?? "",
        shipping_city: address.city,
        shipping_area: address.area,
        shipping_postal_code: address.postal_code ?? "",
      });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to save address.");
    } finally {
      setBusy("");
    }
  }

  async function chooseDelivery(code: string) {
    if (busy) return;
    setDeliveryCode(code);
    clearError();
    setBusy(`delivery:${code}`);

    try {
      await patchCheckout({ delivery_method: code });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to update delivery method.");
    } finally {
      setBusy("");
    }
  }

  async function choosePayment(code: string) {
    if (busy) return;
    setPaymentCode(code);
    clearError();
    setBusy(`payment:${code}`);

    try {
      await patchCheckout({ payment_method: code });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to update payment method.");
    } finally {
      setBusy("");
    }
  }

  function validateCheckout() {
    if (!selectedAddress) return "Choose or add a delivery address.";
    if (!selectedDelivery) return "Choose a delivery method.";
    if (!selectedPayment) return "Choose a payment method.";
    if (!termsAccepted) return "You must agree to the delivery terms before continuing.";
    return "";
  }

  async function continueToReview() {
    if (busy) return;

    const validationError = validateCheckout();
    if (validationError) {
      setError(validationError);
      return;
    }

    if (!selectedAddress || !selectedDelivery || !selectedPayment) return;

    setBusy("review");
    clearError();

    try {
      await patchCheckout({
        customer_name: selectedAddress.recipient_name || customer.full_name,
        customer_phone: selectedAddress.phone || customer.phone,
        customer_email: customer.email ?? "",
        shipping_address_line1: selectedAddress.address_line1,
        shipping_address_line2: selectedAddress.address_line2 ?? "",
        shipping_city: selectedAddress.city,
        shipping_area: selectedAddress.area,
        shipping_postal_code: selectedAddress.postal_code ?? "",
        delivery_method: selectedDelivery.code,
        payment_method: selectedPayment.code,
      });
      setStage("review");
      window.scrollTo({ top: 0, behavior: "smooth" });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to prepare order review.");
    } finally {
      setBusy("");
    }
  }

  async function initiatePayment(orderID: string) {
    setBusy("payment");
    setPaymentMessage("");

    try {
      const response = await fetch(
        `/api/storefront/order/${encodeURIComponent(orderID)}/payment/initiate`,
        { method: "POST" },
      );

      if (response.status === 401) {
        router.replace("/account/sign-in?next=%2Fcheckout");
        return;
      }

      if (!response.ok) {
        throw new Error(await errorMessage(response, "Unable to start payment."));
      }

      const payload: unknown = await response.json();
      const redirectURL = findPaymentURL(payload);

      if (redirectURL) {
        window.location.assign(redirectURL);
        return;
      }

      setStage("payment");
      setPaymentMessage(
        "The payment request was created. Follow the provider instructions returned for this order, or retry if your provider did not open.",
      );
    } catch (caught) {
      setStage("payment");
      setPaymentMessage(caught instanceof Error ? caught.message : "Unable to start payment.");
    } finally {
      setBusy("");
    }
  }

  async function confirmOrder() {
    if (busy) return;

    const validationError = validateCheckout();
    if (validationError) {
      setError(validationError);
      setStage("checkout");
      return;
    }

    setBusy("confirm");
    clearError();

    try {
      const response = await fetch("/api/storefront/order", { method: "POST" });

      if (response.status === 401) {
        router.replace("/account/sign-in?next=%2Fcheckout");
        return;
      }

      if (!response.ok) {
        throw new Error(await errorMessage(response, "Unable to confirm order."));
      }

      const payload = (await response.json()) as PlaceOrderResponse;
      setPlacedOrder(payload.data);

      if (selectedPayment?.requires_immediate_payment) {
        setStage("payment");
        await initiatePayment(payload.data.id);
        return;
      }

      setStage("confirmed");
      window.scrollTo({ top: 0, behavior: "smooth" });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to confirm order.");
    } finally {
      setBusy("");
    }
  }

  const stepState = stage === "confirmed" ? 5 : stage === "payment" ? 4 : stage === "review" ? 3 : 2;
  const actionLabel = stage === "review"
    ? selectedPayment?.requires_immediate_payment
      ? "Confirm & continue to payment"
      : "Confirm order"
    : "Continue to review";

  const summaryCheckout = checkout;

  if (stage === "confirmed" && placedOrder) {
    return (
      <section className={styles.confirmedShell} aria-labelledby="confirmed-title">
        <div className={styles.confirmedIcon}><Icon name="check" /></div>
        <p className={styles.eyebrow}>Order confirmed</p>
        <h1 id="confirmed-title">Thank you. Your order is confirmed.</h1>
        <p className={styles.confirmedLead}>
          Order <strong>{placedOrder.order_number}</strong> has been created. Your immutable invoice is ready now.
        </p>
        <div className={styles.confirmedActions}>
          <a
            className={styles.primaryButton}
            href={`/api/storefront/order/${encodeURIComponent(placedOrder.id)}/invoice`}
            target="_blank"
            rel="noreferrer"
          >
            <Icon name="document" /> View invoice
          </a>
          <Link className={styles.secondaryButton} href="/account/orders">View orders</Link>
          <Link className={styles.textButton} href="/">Continue shopping</Link>
        </div>
        <p className={styles.confirmedNote}>
          Order and invoice notifications are generated by the commerce backend. Delivery depends on the configured notification provider.
        </p>
      </section>
    );
  }

  if (stage === "payment" && placedOrder) {
    return (
      <section className={styles.confirmedShell} aria-labelledby="payment-title">
        <div className={styles.confirmedIcon}><Icon name="wallet" /></div>
        <p className={styles.eyebrow}>Payment</p>
        <h1 id="payment-title">Complete your {selectedPayment?.label ?? "payment"} payment.</h1>
        <p className={styles.confirmedLead}>{paymentMessage || "Opening your payment provider…"}</p>
        <div className={styles.confirmedActions}>
          <button
            type="button"
            className={styles.primaryButton}
            disabled={busy === "payment"}
            onClick={() => void initiatePayment(placedOrder.id)}
          >
            {busy === "payment" ? "Starting payment…" : "Try payment again"}
          </button>
          <Link className={styles.secondaryButton} href="/account/orders">View order</Link>
        </div>
      </section>
    );
  }

  return (
    <>
      <div className={styles.progress} aria-label="Checkout progress">
        {["Cart", "Checkout", "Review", "Payment", "Confirmed"].map((label, index) => {
          const number = index + 1;
          const current = number === stepState;
          const complete = number < stepState;
          return (
            <div key={label} className={`${styles.progressStep} ${current ? styles.progressCurrent : ""} ${complete ? styles.progressComplete : ""}`}>
              <span>{complete ? <Icon name="check" /> : number}</span>
              <strong>{label}</strong>
            </div>
          );
        })}
      </div>

      {stage === "review" ? (
        <div className={styles.reviewBanner}>
          <div>
            <p className={styles.eyebrow}>Final review</p>
            <h1>Review your order before confirmation.</h1>
            <p>No order is placed until you press the confirmation button.</p>
          </div>
          <button type="button" className={styles.secondaryButton} onClick={() => setStage("checkout")}>Edit checkout</button>
        </div>
      ) : null}

      <div className={styles.layout}>
        <div className={styles.mainColumn}>
          <section className={styles.glassSection} aria-labelledby="address-heading">
            <header className={styles.sectionHeader}>
              <span className={styles.sectionNumber}>1</span>
              <div>
                <h2 id="address-heading">Delivery address</h2>
                <p>Choose a saved address from your account or add a new one.</p>
              </div>
              <Link className={styles.manageLink} href="/account/addresses"><Icon name="pin" /> Manage addresses</Link>
            </header>

            <div className={styles.addressGrid}>
              {addresses.map((address) => {
                const selected = selectedAddressID === address.id;
                return (
                  <button
                    type="button"
                    key={address.id}
                    className={`${styles.addressCard} ${selected ? styles.selectedCard : ""}`}
                    aria-pressed={selected}
                    disabled={Boolean(busy)}
                    onClick={() => void chooseAddress(address)}
                  >
                    <span className={styles.radio} aria-hidden="true" />
                    <span className={styles.addressTitleRow}>
                      <strong>{address.label}</strong>
                      {address.is_default ? <small>Default</small> : null}
                    </span>
                    <span>{address.recipient_name}</span>
                    <span>{address.phone}</span>
                    <span>{address.address_line1}</span>
                    {address.address_line2 ? <span>{address.address_line2}</span> : null}
                    <span>{[address.area, address.city, address.postal_code].filter(Boolean).join(", ")}</span>
                  </button>
                );
              })}

              <button
                type="button"
                className={styles.addAddressCard}
                onClick={() => {
                  setShowAddressForm((current) => !current);
                  clearError();
                }}
              >
                <span><Icon name="plus" /></span>
                <strong>Add new address</strong>
                <small>Save it to your account</small>
              </button>
            </div>

            {addresses.length === 0 && !showAddressForm ? (
              <p className={styles.inlineNotice}>You do not have a saved delivery address yet. Add one to continue.</p>
            ) : null}

            {showAddressForm ? (
              <form className={styles.addressForm} onSubmit={addAddress}>
                <div className={styles.formHeading}>
                  <div>
                    <h3>New delivery address</h3>
                    <p>This address will also be saved in your account.</p>
                  </div>
                  <button type="button" onClick={() => setShowAddressForm(false)}>Cancel</button>
                </div>

                <label>
                  <span>Label</span>
                  <input value={addressDraft.label} onChange={(event) => setAddressDraft((current) => ({ ...current, label: event.target.value }))} placeholder="Home, Office…" />
                </label>
                <label>
                  <span>Recipient name</span>
                  <input value={addressDraft.recipient_name} onChange={(event) => setAddressDraft((current) => ({ ...current, recipient_name: event.target.value }))} autoComplete="name" />
                </label>
                <label>
                  <span>Phone</span>
                  <input value={addressDraft.phone} onChange={(event) => setAddressDraft((current) => ({ ...current, phone: event.target.value }))} autoComplete="tel" inputMode="tel" />
                </label>
                <label className={styles.formWide}>
                  <span>Address line 1</span>
                  <input value={addressDraft.address_line1} onChange={(event) => setAddressDraft((current) => ({ ...current, address_line1: event.target.value }))} autoComplete="address-line1" />
                </label>
                <label className={styles.formWide}>
                  <span>Address line 2 <small>Optional</small></span>
                  <input value={addressDraft.address_line2} onChange={(event) => setAddressDraft((current) => ({ ...current, address_line2: event.target.value }))} autoComplete="address-line2" />
                </label>
                <label>
                  <span>Area / Thana</span>
                  <input value={addressDraft.area} onChange={(event) => setAddressDraft((current) => ({ ...current, area: event.target.value }))} />
                </label>
                <label>
                  <span>City / District</span>
                  <input value={addressDraft.city} onChange={(event) => setAddressDraft((current) => ({ ...current, city: event.target.value }))} autoComplete="address-level2" />
                </label>
                <label>
                  <span>Postal code <small>Optional</small></span>
                  <input value={addressDraft.postal_code} onChange={(event) => setAddressDraft((current) => ({ ...current, postal_code: event.target.value }))} autoComplete="postal-code" inputMode="numeric" />
                </label>
                <label className={styles.defaultCheck}>
                  <input type="checkbox" checked={addressDraft.is_default} onChange={(event) => setAddressDraft((current) => ({ ...current, is_default: event.target.checked }))} />
                  <span>Make this my default address</span>
                </label>
                <button className={`${styles.primaryButton} ${styles.saveAddressButton}`} type="submit" disabled={busy === "new-address"}>
                  {busy === "new-address" ? "Saving…" : "Save & use address"}
                </button>
              </form>
            ) : null}
          </section>

          <section className={styles.glassSection} aria-labelledby="delivery-heading">
            <header className={styles.sectionHeader}>
              <span className={styles.sectionNumber}>2</span>
              <div>
                <h2 id="delivery-heading">Delivery method</h2>
                <p>Available methods are selectable. Planned delivery types stay visible until enabled.</p>
              </div>
            </header>

            <div className={styles.optionGrid}>
              {displayDeliveryOptions.map((displayOption) => {
                const option = displayOption.backendOption;
                const unavailable = !option;
                const selected = option ? deliveryCode === option.code : false;

                return (
                  <button
                    type="button"
                    key={displayOption.key}
                    className={`${styles.optionCard} ${selected ? styles.selectedCard : ""} ${unavailable ? styles.unavailableCard : ""}`}
                    aria-pressed={selected}
                    aria-disabled={unavailable}
                    disabled={Boolean(busy) || unavailable}
                    onClick={() => {
                      if (option) void chooseDelivery(option.code);
                    }}
                  >
                    <span className={styles.radio} aria-hidden="true" />
                    <span className={styles.optionIcon}><Icon name="truck" /></span>
                    <span className={styles.optionCopy}>
                      <strong>{displayOption.label}</strong>
                      <small>{displayOption.description}</small>
                    </span>
                    <strong className={`${styles.optionPrice} ${unavailable ? styles.unavailableText : ""}`}>
                      {unavailable
                        ? "Unavailable"
                        : option.shipping_amount === 0
                          ? "Free"
                          : formatMoney(option.shipping_amount, option.currency)}
                    </strong>
                  </button>
                );
              })}
            </div>

            {deliveryOptions.length === 0 ? <p className={styles.inlineNotice}>Delivery methods are shown for layout completeness, but none are enabled by the backend yet.</p> : null}
          </section>

          <section className={styles.glassSection} aria-labelledby="payment-heading">
            <header className={styles.sectionHeader}>
              <span className={styles.sectionNumber}>3</span>
              <div>
                <h2 id="payment-heading">Payment method</h2>
                <p>Enabled methods are selectable. Backend-supported payment methods remain visible while unavailable.</p>
              </div>
            </header>

            <div className={styles.paymentGrid}>
              {displayPaymentOptions.map((displayOption) => {
                const option = displayOption.backendOption;
                const unavailable = !option;
                const selected = option ? paymentCode === option.code : false;

                return (
                  <button
                    type="button"
                    key={displayOption.key}
                    className={`${styles.paymentCard} ${selected ? styles.selectedCard : ""} ${unavailable ? styles.unavailableCard : ""}`}
                    aria-pressed={selected}
                    aria-disabled={unavailable}
                    disabled={Boolean(busy) || unavailable}
                    onClick={() => {
                      if (option) void choosePayment(option.code);
                    }}
                  >
                    <span className={styles.radio} aria-hidden="true" />
                    <span className={styles.paymentMark}><PaymentMethodIcon option={displayOption} /></span>
                    <span className={styles.optionCopy}>
                      <strong>{displayOption.label}</strong>
                      <small>{displayOption.description}</small>
                    </span>
                    {unavailable ? <span className={styles.unavailableBadge}>Unavailable</span> : null}
                  </button>
                );
              })}
            </div>

            {paymentOptions.length === 0 ? <p className={styles.inlineNotice}>Payment methods are shown for layout completeness, but none are enabled by the backend yet.</p> : null}
          </section>

          <section className={styles.glassSection} aria-labelledby="terms-heading">
            <header className={styles.sectionHeader}>
              <span className={styles.sectionNumber}>4</span>
              <div>
                <h2 id="terms-heading">Delivery terms agreement</h2>
                <p>You must agree before the order can be confirmed.</p>
              </div>
            </header>

            <div className={styles.termsRow}>
              <label>
                <input
                  type="checkbox"
                  checked={termsAccepted}
                  onChange={(event) => {
                    setTermsAccepted(event.target.checked);
                    clearError();
                  }}
                />
                <span>I have read and agree to the <strong>Delivery Terms and Conditions.</strong></span>
              </label>
              <button type="button" onClick={() => termsDialogRef.current?.showModal()}>View terms</button>
            </div>
          </section>

          {stage === "review" ? (
            <section className={styles.reviewDetails} aria-label="Order review selections">
              <div><span>Ship to</span><strong>{selectedAddress?.label}</strong><small>{selectedAddress ? [selectedAddress.address_line1, selectedAddress.area, selectedAddress.city].filter(Boolean).join(", ") : "—"}</small></div>
              <div><span>Delivery</span><strong>{selectedDelivery?.label ?? "—"}</strong><small>{selectedDelivery ? etaLabel(selectedDelivery) : ""}</small></div>
              <div><span>Payment</span><strong>{selectedPayment?.label ?? "—"}</strong><small>{selectedPayment ? paymentDescription(selectedPayment) : ""}</small></div>
              <div><span>Terms</span><strong>Accepted</strong><small>Required delivery agreement</small></div>
            </section>
          ) : null}

          <div className={styles.trustStrip} aria-label="Checkout assurances">
            <div><Icon name="shield" /><span><strong>Secure checkout</strong><small>Your account session is protected</small></span></div>
            <div><Icon name="truck" /><span><strong>Backend delivery quotes</strong><small>Fees and ETA come from checkout data</small></span></div>
            <div><Icon name="wallet" /><span><strong>Verified payment options</strong><small>Enabled methods are selectable; unavailable methods are clearly marked</small></span></div>
          </div>
        </div>

        <aside className={styles.summaryColumn}>
          <div className={styles.summaryCard}>
            <header className={styles.summaryHeader}>
              <h2>Order summary</h2>
              <span>{summaryCheckout.item_count} {summaryCheckout.item_count === 1 ? "item" : "items"}</span>
            </header>

            <div className={styles.summaryItems}>
              {summaryCheckout.items.map((item) => (
                <div className={styles.summaryItem} key={item.id}>
                  <span className={styles.itemFallback} aria-hidden="true">{item.product_name.slice(0, 1).toUpperCase()}</span>
                  <span className={styles.itemCopy}>
                    <strong>{item.product_name}</strong>
                    <small>SKU {item.sku}</small>
                    <small>Qty: {item.quantity}</small>
                  </span>
                  <strong>{formatMoney(item.line_total_amount, item.currency)}</strong>
                </div>
              ))}
            </div>

            <dl className={styles.totals}>
              <div><dt>Subtotal</dt><dd>{formatMoney(summaryCheckout.subtotal_amount, summaryCheckout.currency)}</dd></div>
              <div><dt>Delivery fee</dt><dd>{summaryCheckout.shipping_amount === 0 ? "Free" : formatMoney(summaryCheckout.shipping_amount, summaryCheckout.currency)}</dd></div>
              <div className={styles.discountRow}><dt>Discount</dt><dd>− {formatMoney(summaryCheckout.discount_amount, summaryCheckout.currency)}</dd></div>
              <div className={styles.totalRow}><dt>Total</dt><dd>{formatMoney(summaryCheckout.total_amount, summaryCheckout.currency)}</dd></div>
            </dl>

            {error ? <div className={styles.error} role="alert">{error}</div> : null}

            <button
              type="button"
              className={styles.primaryButton}
              disabled={Boolean(busy) || paymentOptions.length === 0 || deliveryOptions.length === 0}
              onClick={() => void (stage === "review" ? confirmOrder() : continueToReview())}
            >
              <span>{busy === "review" ? "Preparing review…" : busy === "confirm" ? "Confirming order…" : actionLabel}</span>
              <Icon name="arrow" />
            </button>

            <p className={styles.summaryFootnote}>
              {stage === "review"
                ? selectedPayment?.requires_immediate_payment
                  ? "Your order is created first, then the configured payment provider is opened."
                  : "Cash on Delivery confirms the order without an online charge."
                : "Nothing is ordered until you review and confirm."}
            </p>
          </div>
        </aside>
      </div>

      <div className={styles.mobileDock}>
        <span><small>Total</small><strong>{formatMoney(summaryCheckout.total_amount, summaryCheckout.currency)}</strong></span>
        <button
          type="button"
          disabled={Boolean(busy) || paymentOptions.length === 0 || deliveryOptions.length === 0}
          onClick={() => void (stage === "review" ? confirmOrder() : continueToReview())}
        >
          {busy ? "Please wait…" : actionLabel}
          <Icon name="arrow" />
        </button>
      </div>

      <dialog className={styles.termsDialog} ref={termsDialogRef}>
        <div className={styles.dialogHeader}>
          <div>
            <p className={styles.eyebrow}>Delivery terms</p>
            <h2>Terms and conditions</h2>
          </div>
          <button type="button" onClick={() => termsDialogRef.current?.close()} aria-label="Close terms">×</button>
        </div>
        <p>
          The final delivery terms will be added here before production. The checkout already requires explicit agreement so the legal copy can be inserted later without changing the purchase flow.
        </p>
        <button className={styles.primaryButton} type="button" onClick={() => termsDialogRef.current?.close()}>Close</button>
      </dialog>
    </>
  );
}
