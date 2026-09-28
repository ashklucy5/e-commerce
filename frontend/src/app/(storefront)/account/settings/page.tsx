import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { AccountSettingsClient } from "@/components/account/settings/components/AccountSettingsClient";
import {
  getCustomerAccessToken,
  getCustomerRefreshToken,
} from "@/lib/account/session";

export const metadata: Metadata = {
  title: "Profile & Settings",
  robots: {
    index: false,
    follow: false,
  },
};

export const dynamic = "force-dynamic";

export default async function AccountSettingsPage() {
  const [accessToken, refreshToken] = await Promise.all([
    getCustomerAccessToken(),
    getCustomerRefreshToken(),
  ]);

  if (!accessToken && !refreshToken) {
    redirect("/account/sign-in?next=%2Faccount%2Fsettings");
  }

  return <AccountSettingsClient />;
}
