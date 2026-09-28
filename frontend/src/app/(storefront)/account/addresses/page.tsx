// Location: src/app/(storefront)/account/addresses/page.tsx
import type { Metadata } from "next";

import { AddressesClient } from "@/components/account/components/AddressesClient";

export const metadata: Metadata = {
  title: "Saved Addresses",
  robots: { index: false, follow: false },
};

export const dynamic = "force-dynamic";

export default function AddressesPage() {
  return <AddressesClient />;
}
