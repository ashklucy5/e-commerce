import type {
  Metadata,
  Viewport,
} from "next";

import type { ReactNode } from "react";

import { notFound } from "next/navigation";

import { getAdminPortalSlug } from "@/lib/admin/portal";

export const runtime = "nodejs";

export const dynamic =
  "force-dynamic";

export const metadata: Metadata = {
  title: {
    default: "Ene dei — Operations",
    template: "%s | Ene dei Operations",
  },

  description:
    "Private Ene dei commerce operations console.",

  robots: {
    index: false,
    follow: false,
    nocache: true,
  },

  referrer: "no-referrer",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
  colorScheme: "dark",
  themeColor: "#070709",
};

type AdminPortalLayoutProps = {
  children: ReactNode;

  params: Promise<{
    portal: string;
  }>;
};

export default async function AdminPortalLayout({
  children,
  params,
}: AdminPortalLayoutProps) {
  const { portal } = await params;

  const expectedPortal =
    await getAdminPortalSlug();

  if (portal !== expectedPortal) {
    notFound();
  }

  return children;
}