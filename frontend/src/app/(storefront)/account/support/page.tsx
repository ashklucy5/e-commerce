import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { SupportCenterClient } from "@/components/account/components/SupportCenterClient";
import {
  getCustomerAccessToken,
  getCustomerRefreshToken,
} from "@/lib/account/session";

export const metadata: Metadata = {
  title: "Customer Support",
  robots: {
    index: false,
    follow: false,
  },
};

export const dynamic = "force-dynamic";

export default async function SupportPage() {
  const [accessToken, refreshToken] = await Promise.all([
    getCustomerAccessToken(),
    getCustomerRefreshToken(),
  ]);

  if (!accessToken && !refreshToken) {
    redirect("/account/sign-in?next=%2Faccount%2Fsupport");
  }

  return <SupportCenterClient />;
}