// ENE_WISHLIST_PAGE_V1
// Location: src/app/(storefront)/account/wishlist/page.tsx

import type { Metadata } from "next";

import { WishlistClient } from "@/components/account/components/WishlistClient";

export const metadata: Metadata = {
  title: "Wishlist",
  robots: { index: false, follow: false },
};

export const dynamic = "force-dynamic";

export default function WishlistPage() {
  return <WishlistClient />;
}
