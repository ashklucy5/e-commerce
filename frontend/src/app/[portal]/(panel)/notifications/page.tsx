import type {
  Metadata,
} from "next";

import AdminNotificationsWorkspace from "@/components/admin/notifications/components/AdminNotificationsWorkspace";

export const metadata: Metadata = {
  title:
    "Notifications | Ene dei Operations",

  robots: {
    index: false,
    follow: false,
    nocache: true,
  },
};

type NotificationsPageProps = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function NotificationsPage({
  params,
}: NotificationsPageProps) {
  const {
    portal,
  } =
    await params;

  return (
    <AdminNotificationsWorkspace
      portal={portal}
    />
  );
}