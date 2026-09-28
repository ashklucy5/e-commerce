import AdminReturnsWorkspace from "@/components/admin/returns/components/AdminReturnsWorkspace";

type Props = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function AdminReturnsPage({
  params,
}: Props) {
  const { portal } =
    await params;

  return (
    <AdminReturnsWorkspace
      portal={portal}
    />
  );
}