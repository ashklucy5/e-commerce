import type { Metadata } from "next";

import AdminStaffActivationFlow from "@/components/admin/auth/components/AdminStaffActivationFlow";

export const metadata: Metadata = {
  title: "Activate staff account",
};

type Props = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function AdminStaffActivationPage({ params }: Props) {
  const { portal } = await params;
  return <AdminStaffActivationFlow portal={portal} />;
}
