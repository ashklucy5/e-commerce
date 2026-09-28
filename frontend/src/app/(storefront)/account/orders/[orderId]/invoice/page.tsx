// ENE_INVOICE_PAGE_ROUTE_V6
// Location: src/app/(storefront)/account/orders/[orderId]/invoice/page.tsx

import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { InvoiceClient } from "@/components/account/components/InvoiceClient";
import {
  getCustomerAccessToken,
  getCustomerRefreshToken,
} from "@/lib/account/session";

export const metadata: Metadata = {
  title: "Invoice",
  robots: {
    index: false,
    follow: false,
  },
};

export const dynamic = "force-dynamic";

type InvoicePageProps = {
  params: Promise<{
    orderId: string;
  }>;
};

export default async function InvoicePage({ params }: InvoicePageProps) {
  const { orderId } = await params;

  const [accessToken, refreshToken] = await Promise.all([
    getCustomerAccessToken(),
    getCustomerRefreshToken(),
  ]);

  if (!accessToken && !refreshToken) {
    const nextPath = encodeURIComponent(`/account/orders/${orderId}/invoice`);
    redirect(`/account/sign-in?next=${nextPath}`);
  }

  return <InvoiceClient orderId={orderId} />;
}
