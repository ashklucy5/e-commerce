import AdminCustomersWorkspace from "@/components/admin/customers/components/AdminCustomersWorkspace";

type Props = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function AdminCustomersPage({ params }: Props) {
  const { portal } = await params;

  return <AdminCustomersWorkspace portal={portal} />;
}
