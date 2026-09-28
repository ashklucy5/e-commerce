// Location: src/app/(storefront)/account/page.tsx

import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { AccountHomeClient } from "@/components/account/components/AccountHomeClient";
import {
  getCustomerAccessToken,
  getCustomerRefreshToken,
} from "@/lib/account/session";

export const metadata: Metadata = {
  title: "Account",
  robots: { index: false, follow: false },
};

export const dynamic = "force-dynamic";

export default async function AccountPage() {
  const [accessToken, refreshToken] = await Promise.all([
    getCustomerAccessToken(),
    getCustomerRefreshToken(),
  ]);

  if (!accessToken && !refreshToken) {
    redirect("/account/sign-in?next=%2Faccount");
  }

  return <AccountHomeClient />;
}
