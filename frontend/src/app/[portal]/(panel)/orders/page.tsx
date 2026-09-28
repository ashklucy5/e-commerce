import type { Metadata } from "next";

import AdminOrdersWorkspace from "@/components/admin/orders/components/AdminOrdersWorkspace";

export const metadata: Metadata = {
  title: "Orders | Ene dei Operations",
  robots: {
    index: false,
    follow: false,
  },
};

type OrdersPageProps = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function OrdersPage({
  params,
}: OrdersPageProps) {
  const { portal } = await params;

  return (
    <AdminOrdersWorkspace
      portal={portal}
      mode="orders"
    />
  );
}
