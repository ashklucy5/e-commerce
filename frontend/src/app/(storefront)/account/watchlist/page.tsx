// Location: src/app/(storefront)/account/watchlist/page.tsx

import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { WatchlistClient } from "@/components/account/components/WatchlistClient";
import {
  getCustomerAccessToken,
  getCustomerRefreshToken,
} from "@/lib/account/session";

export const metadata: Metadata = {
  title: "Watchlist",
  robots: { index: false, follow: false },
};

export const dynamic = "force-dynamic";

export default async function WatchlistPage() {
  const [accessToken, refreshToken] = await Promise.all([
    getCustomerAccessToken(),
    getCustomerRefreshToken(),
  ]);

  if (!accessToken && !refreshToken) {
    redirect("/account/sign-in?next=%2Faccount%2Fwatchlist");
  }

  return <WatchlistClient />;
}
