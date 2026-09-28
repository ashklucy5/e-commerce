import AdminSupportWorkspace from "@/components/admin/support/components/AdminSupportWorkspace";

type Props = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function AdminSupportPage({ params }: Props) {
  const { portal } = await params;
  return <AdminSupportWorkspace portal={portal} />;
}
