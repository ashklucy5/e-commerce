import type {
  Metadata,
} from "next";

import AdminAuthFlow from "@/components/admin/auth/components/AdminAuthFlow";

export const metadata: Metadata = {
  title: "Sign in",
};

type AdminLoginPageProps = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function AdminLoginPage({
  params,
}: AdminLoginPageProps) {
  const { portal } = await params;

  return (
    <AdminAuthFlow portal={portal} />
  );
}