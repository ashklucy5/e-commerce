import AdminDashboard from "@/components/admin/dashboard/components/AdminDashboard";

type AdminDashboardPageProps = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function AdminDashboardPage({
  params,
}: AdminDashboardPageProps) {
  const { portal } = await params;

  return <AdminDashboard portal={portal} />;
}
