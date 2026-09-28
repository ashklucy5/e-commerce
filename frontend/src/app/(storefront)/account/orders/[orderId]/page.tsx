// Location: src/app/(storefront)/account/orders/[orderId]/page.tsx

import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { OrderDetailClient } from "@/components/account/components/OrderDetailClient";
import {
  getCustomerAccessToken,
  getCustomerRefreshToken,
} from "@/lib/account/session";

export const metadata: Metadata = {
  title: "Order Details",
  robots: { index: false, follow: false },
};

export const dynamic = "force-dynamic";

type Props = {
  params: Promise<{ orderId: string }>;
};

export default async function OrderDetailPage({ params }: Props) {
  const { orderId } = await params;
  const [accessToken, refreshToken] = await Promise.all([
    getCustomerAccessToken(),
    getCustomerRefreshToken(),
  ]);

  if (!accessToken && !refreshToken) {
    const next = encodeURIComponent(`/account/orders/${orderId}`);
    redirect(`/account/sign-in?next=${next}`);
  }

  return <OrderDetailClient orderId={orderId} />;
}
