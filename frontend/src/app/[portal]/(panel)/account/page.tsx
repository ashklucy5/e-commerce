import type { Metadata } from "next";

import AdminAccountSecurity from "@/components/admin/account/components/AdminAccountSecurity";

export const metadata: Metadata = {
  title: "Account & Security | Ene dei Operations",
  robots: {
    index: false,
    follow: false,
  },
};

type AccountPageProps = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function AccountPage({
  params,
}: AccountPageProps) {
  const { portal } = await params;

  return <AdminAccountSecurity portal={portal} />;
}
