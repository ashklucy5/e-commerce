import type {
  Metadata,
} from "next";

import ProductCreateForm from "@/components/admin/products/components/ProductCreateForm";

export const metadata: Metadata = {
  title: "Create product",
};

export default function NewProductPage() {
  return <ProductCreateForm />;
}