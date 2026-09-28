// Location: src/app/(storefront)/cart/page.tsx
import type { Metadata } from "next";

import { CartClient } from "@/components/cart/components/CartClient";
import type { Cart } from "@/lib/api/contracts/commerce";
import { getCurrentCart } from "@/lib/commerce/server";

export const metadata: Metadata = {
  title: "Cart",
  robots: {
    index: false,
    follow: false,
  },
};

export const dynamic = "force-dynamic";

export default async function CartPage() {
  let cart: Cart | null = null;
  let initialLoadError = false;

  try {
    cart = await getCurrentCart();
  } catch {
    initialLoadError = true;
  }

  return (
    <main>
      <CartClient
        initialCart={cart}
        initialLoadError={initialLoadError}
      />
    </main>
  );
}
