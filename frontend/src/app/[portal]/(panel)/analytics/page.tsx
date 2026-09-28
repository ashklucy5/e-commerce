import type { Metadata } from "next";

import CommerceIntelligenceWorkspace from "@/components/admin/analytics/components/CommerceIntelligenceWorkspace";

export const metadata: Metadata = {
  title: "Commerce Intelligence",
  robots: {
    index: false,
    follow: false,
  },
};

export const dynamic = "force-dynamic";

type Props = {
  params: Promise<{
    portal: string;
  }>;
};

export default async function AnalyticsPage({
  params,
}: Props) {
  const { portal } = await params;

  return (
    <CommerceIntelligenceWorkspace
      portal={portal}
    />
  );
}