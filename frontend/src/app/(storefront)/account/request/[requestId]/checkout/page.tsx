import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { SourcingCheckoutClient } from "@/components/account/components/SourcingCheckoutClient";

import {
  getCustomerAccessToken,
  getCustomerRefreshToken,
} from "@/lib/account/session";

export const metadata: Metadata = {
  title: "Sourcing Checkout",
  robots: {
    index: false,
    follow: false,
  },
};

export const dynamic = "force-dynamic";

type PageProps = {
  params: Promise<{
    requestId: string;
  }>;
};

export default async function SourcingCheckoutPage({
  params,
}: PageProps) {
  const { requestId } = await params;

  const [accessToken, refreshToken] =
    await Promise.all([
      getCustomerAccessToken(),
      getCustomerRefreshToken(),
    ]);

  if (!accessToken && !refreshToken) {
    const next = encodeURIComponent(
      `/account/request/${encodeURIComponent(
        requestId,
      )}/checkout`,
    );

    redirect(
      `/account/sign-in?next=${next}`,
    );
  }

  return (
    <SourcingCheckoutClient
      requestID={requestId}
    />
  );
}