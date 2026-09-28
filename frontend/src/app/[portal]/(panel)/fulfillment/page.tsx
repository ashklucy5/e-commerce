import type { Metadata } from "next";

import AdminOrdersWorkspace from "@/components/admin/orders/components/AdminOrdersWorkspace";

export const metadata: Metadata = {
  title: "Fulfillment | Ene dei Operations",
  robots: {
    index: false,
    follow: false,
  },
};

type FulfillmentPageProps = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function FulfillmentPage({
  params,
}: FulfillmentPageProps) {
  const { portal } = await params;

  return (
    <AdminOrdersWorkspace
      portal={portal}
      mode="fulfillment"
    />
  );
}
