import AdminShipmentsWorkspace from "@/components/admin/shipments/components/AdminShipmentsWorkspace";

type Props = {
  params: Promise<{ portal: string }>;
  searchParams: Promise<{ q?: string }>;
};

export default async function AdminShipmentsPage({ params, searchParams }: Props) {
  const [{ portal }, query] = await Promise.all([params, searchParams]);

  return (
    <AdminShipmentsWorkspace
      portal={portal}
      initialQuery={query.q ?? ""}
    />
  );
}
