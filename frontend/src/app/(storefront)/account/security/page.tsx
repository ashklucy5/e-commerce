// Location: src/app/(storefront)/account/security/page.tsx
import type { Metadata } from "next";

import { SecurityClient } from "@/components/account/components/SecurityClient";

export const metadata: Metadata = {
  title: "Account Security",
  robots: { index: false, follow: false },
};

export const dynamic = "force-dynamic";

export default function SecurityPage() {
  return <SecurityClient />;
}
