import type {
  Metadata,
} from "next";

import CatalogImportWorkspace from "@/components/admin/products/components/CatalogImportWorkspace";

export const metadata:
  Metadata = {
    title:
      "Catalog Import | Ene dei Operations",

    robots: {
      index: false,
      follow: false,
    },
  };

type CatalogImportPageProps = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function CatalogImportPage({
  params,
}: CatalogImportPageProps) {
  const {
    portal,
  } = await params;

  return (
    <CatalogImportWorkspace
      portal={portal}
    />
  );
}