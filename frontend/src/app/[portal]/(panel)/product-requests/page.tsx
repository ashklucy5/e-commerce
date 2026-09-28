import AdminProductRequestsWorkspace from "@/components/admin/product-requests/components/AdminProductRequestsWorkspace";

type Props = {
  params: Promise<{ portal: string }>;
};

export default async function AdminProductRequestsPage({ params }: Props) {
  const { portal } = await params;

  return <AdminProductRequestsWorkspace portal={portal} />;
}
