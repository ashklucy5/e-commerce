import type { Metadata } from "next";

import AdminPromotionsWorkspace from "@/components/admin/promotions/components/AdminPromotionsWorkspace";

export const metadata: Metadata = {
  title: "Promotions & discounts | Ene dei Operations",
  robots: {
    index: false,
    follow: false,
  },
};

export default function AdminPromotionsPage() {
  return <AdminPromotionsWorkspace />;
}
