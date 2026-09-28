// Location: src/app/(storefront)/(auth)/account/sign-up/page.tsx
import type { Metadata } from "next";

import { AccountAuthClient } from "@/components/account/components/AccountAuthClient";

export const metadata: Metadata = {
  title: "Create Account",
  robots: { index: false, follow: false },
};

type Props = {
  searchParams: Promise<{ next?: string | string[] }>;
};

function safeReturnTo(value: string | string[] | undefined) {
  const candidate = Array.isArray(value) ? value[0] ?? "" : value ?? "";

  if (!candidate.startsWith("/") || candidate.startsWith("//") || candidate.includes("\\")) {
    return "/account";
  }

  return candidate;
}

export default async function SignUpPage({ searchParams }: Props) {
  const params = await searchParams;
  return <AccountAuthClient mode="sign-up" returnTo={safeReturnTo(params.next)} />;
}
