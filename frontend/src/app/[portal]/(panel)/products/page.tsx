import type {
  Metadata,
} from "next";

import ProductsManagement from "@/components/admin/products/components/ProductsManagement";

export const metadata:
  Metadata = {
    title:
      "Products | Ene dei Operations",

    robots: {
      index: false,
      follow: false,
    },
  };

type ProductsPageProps = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function ProductsPage({
  params,
}: ProductsPageProps) {
  const {
    portal,
  } = await params;

  return (
    <ProductsManagement
      portal={portal}
    />
  );
}