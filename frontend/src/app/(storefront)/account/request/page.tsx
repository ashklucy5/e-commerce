// Location: src/app/(storefront)/account/request/page.tsx

import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { ProductRequestCenterClient } from "@/components/account/components/ProductRequestCenterClient";
import {
  getCustomerAccessToken,
  getCustomerRefreshToken,
} from "@/lib/account/session";

export const metadata: Metadata = {
  title: "Product Requests",

  robots: {
    index: false,
    follow: false,
  },
};

export const dynamic =
  "force-dynamic";

export default async function ProductRequestPage() {
  const [
    accessToken,
    refreshToken,
  ] =
    await Promise.all([
      getCustomerAccessToken(),
      getCustomerRefreshToken(),
    ]);

  if (
    !accessToken &&
    !refreshToken
  ) {
    redirect(
      "/account/sign-in?next=%2Faccount%2Frequest",
    );
  }

  return (
    <ProductRequestCenterClient />
  );
}