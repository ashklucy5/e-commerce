// Location: src/app/(storefront)/checkout/page.tsx

import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { CheckoutClient } from "@/components/checkout/components/CheckoutClient";
import styles from "@/components/checkout/css/Checkout.module.css";
import {
  getCustomerAccessToken,
  getCustomerRefreshToken,
} from "@/lib/account/session";
import type {
  AccountAddress,
  AccountCustomer,
  AccountDataResponse,
} from "@/lib/api/contracts/account";
import type {
  ApiData,
  CheckoutSession,
  DeliveryOption,
  PaymentOption,
} from "@/lib/api/contracts/commerce";
import { CommerceApiError } from "@/lib/api/error";
import { commerceFetch } from "@/lib/api/server";
import { getCheckoutKey } from "@/lib/commerce/session";

export const metadata: Metadata = {
  title: "Secure Checkout",
  robots: { index: false, follow: false },
};

export const dynamic = "force-dynamic";

const SIGN_IN_PATH = "/account/sign-in?next=%2Fcheckout";
const ENSURE_SESSION_PATH = "/api/storefront/auth/ensure?next=%2Fcheckout";

async function requireCheckoutCustomer() {
  const [accessToken, refreshToken] = await Promise.all([
    getCustomerAccessToken(),
    getCustomerRefreshToken(),
  ]);

  if (!accessToken) {
    if (refreshToken) {
      redirect(ENSURE_SESSION_PATH);
    }

    redirect(SIGN_IN_PATH);
  }

  try {
    const response = await commerceFetch<AccountDataResponse<AccountCustomer>>(
      "/api/v1/customers/me",
      {
        cache: "no-store",
        headers: {
          Authorization: `Bearer ${accessToken}`,
        },
      },
    );

    return {
      customer: response.data,
      accessToken,
    };
  } catch (error) {
    if (error instanceof CommerceApiError && error.status === 401) {
      if (refreshToken) {
        redirect(ENSURE_SESSION_PATH);
      }

      redirect(SIGN_IN_PATH);
    }

    throw error;
  }
}

function authHeaders(accessToken: string) {
  return {
    Authorization: `Bearer ${accessToken}`,
  };
}

async function getAuthenticatedCheckout(
  checkoutKey: string,
  accessToken: string,
) {
  const response = await commerceFetch<ApiData<CheckoutSession>>(
    `/api/v1/checkouts/${encodeURIComponent(checkoutKey)}`,
    {
      cache: "no-store",
      headers: authHeaders(accessToken),
    },
  );

  return response.data;
}

async function getAuthenticatedDeliveryOptions(
  checkoutKey: string,
  accessToken: string,
) {
  const response = await commerceFetch<ApiData<DeliveryOption[]>>(
    `/api/v1/checkouts/${encodeURIComponent(checkoutKey)}/delivery-methods`,
    {
      cache: "no-store",
      headers: authHeaders(accessToken),
    },
  );

  return response.data;
}

async function getAuthenticatedPaymentOptions(
  checkoutKey: string,
  accessToken: string,
) {
  const query = new URLSearchParams({ checkout_key: checkoutKey });
  const response = await commerceFetch<ApiData<PaymentOption[]>>(
    `/api/v1/checkout-options/payment-methods?${query.toString()}`,
    {
      cache: "no-store",
      headers: authHeaders(accessToken),
    },
  );

  return response.data;
}

async function getSavedAddresses(accessToken: string) {
  try {
    const response = await commerceFetch<AccountDataResponse<AccountAddress[]>>(
      "/api/v1/customers/me/addresses",
      {
        cache: "no-store",
        headers: authHeaders(accessToken),
      },
    );

    return response.data ?? [];
  } catch (error) {
    if (error instanceof CommerceApiError && error.status === 401) {
      throw error;
    }

    // Checkout can still recover this list through the account BFF in the client.
    return [];
  }
}

export default async function CheckoutPage() {
  const { customer, accessToken } = await requireCheckoutCustomer();
  const checkoutKey = await getCheckoutKey();

  if (!checkoutKey) {
    redirect("/cart");
  }

  let checkout: CheckoutSession;

  try {
    checkout = await getAuthenticatedCheckout(checkoutKey, accessToken);
  } catch (error) {
    if (
      error instanceof CommerceApiError &&
      (error.status === 404 ||
        ["CHECKOUT_NOT_FOUND", "CHECKOUT_NOT_ACTIVE", "EXPIRED"].includes(error.code ?? ""))
    ) {
      redirect("/cart");
    }

    if (error instanceof CommerceApiError && error.status === 401) {
      redirect(ENSURE_SESSION_PATH);
    }

    throw error;
  }

  if (checkout.status !== "active") {
    redirect("/cart");
  }

  const [deliveryOptions, paymentOptions, addresses] = await Promise.all([
    getAuthenticatedDeliveryOptions(checkout.checkout_key, accessToken),
    getAuthenticatedPaymentOptions(checkout.checkout_key, accessToken),
    getSavedAddresses(accessToken),
  ]);

  return (
    <main className={styles.page}>
      <div className={styles.ambient} aria-hidden="true">
        <span className={styles.orbOne} />
        <span className={styles.orbTwo} />
        <span className={styles.orbThree} />
      </div>

      <div className={styles.shell}>
        <CheckoutClient
          customer={customer}
          initialAddresses={addresses}
          initialCheckout={checkout}
          deliveryOptions={deliveryOptions}
          paymentOptions={paymentOptions}
        />
      </div>
    </main>
  );
}
