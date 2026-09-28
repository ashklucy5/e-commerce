import type { Metadata } from "next";

import AdminStaffWorkspace from "@/components/admin/staff/components/AdminStaffWorkspace";

export const metadata: Metadata = {
  title: "Staff & roles",
};

type Props = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function AdminStaffPage({ params }: Props) {
  const { portal } = await params;
  return <AdminStaffWorkspace portal={portal} />;
}
