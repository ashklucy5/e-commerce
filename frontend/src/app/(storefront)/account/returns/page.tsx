// Location: src/app/(storefront)/account/returns/page.tsx
import type { Metadata } from "next";

import { ReturnsClient } from "@/components/account/components/ReturnsClient";

export const metadata: Metadata = {
  title: "Returns & Refunds",
  robots: { index: false, follow: false },
};

export const dynamic = "force-dynamic";

export default function ReturnsPage() {
  return <ReturnsClient />;
}
