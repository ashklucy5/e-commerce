// Location: src/app/(storefront)/account/orders/page.tsx

import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { OrdersClient } from "@/components/account/components/OrdersClient";
import {
  getCustomerAccessToken,
  getCustomerRefreshToken,
} from "@/lib/account/session";

export const metadata: Metadata = {
  title: "Orders",
  robots: { index: false, follow: false },
};

export const dynamic = "force-dynamic";

export default async function OrdersPage() {
  const [accessToken, refreshToken] = await Promise.all([
    getCustomerAccessToken(),
    getCustomerRefreshToken(),
  ]);

  if (!accessToken && !refreshToken) {
    redirect("/account/sign-in?next=%2Faccount%2Forders");
  }

  return <OrdersClient />;
}
