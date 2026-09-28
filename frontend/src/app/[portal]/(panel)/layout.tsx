import type {
  ReactNode,
} from "react";

import AdminShell from "@/components/admin/layout/components/AdminShell";

type AdminPanelLayoutProps = {
  children: ReactNode;

  params: Promise<{
    portal: string;
  }>;
};

export default async function AdminPanelLayout({
  children,
  params,
}: AdminPanelLayoutProps) {
  const { portal } = await params;

  return (
    <AdminShell portal={portal}>
      {children}
    </AdminShell>
  );
}