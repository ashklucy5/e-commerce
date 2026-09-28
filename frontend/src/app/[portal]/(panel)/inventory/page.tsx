import type { Metadata } from "next";

import AdminInventoryWorkspace from "@/components/admin/inventory/components/AdminInventoryWorkspace";

export const metadata: Metadata = {
  title: "Inventory | Ene dei Operations",

  robots: {
    index: false,
    follow: false,
    nocache: true,
  },
};

type InventoryPageProps = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function InventoryPage({
  params,
}: InventoryPageProps) {
  const { portal } = await params;

  return (
    <AdminInventoryWorkspace
      portal={portal}
    />
  );
}