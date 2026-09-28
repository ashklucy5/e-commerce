"use client";

import {
  createContext,
  type ReactNode,
  useContext,
} from "react";

import type { AdminPrincipal } from "@/lib/admin/types";

const AdminSessionContext =
  createContext<AdminPrincipal | null>(null);

type AdminSessionProviderProps = {
  children: ReactNode;
  principal: AdminPrincipal;
};

export function AdminSessionProvider({
  children,
  principal,
}: AdminSessionProviderProps) {
  return (
    <AdminSessionContext.Provider value={principal}>
      {children}
    </AdminSessionContext.Provider>
  );
}

export function useAdminSession(): AdminPrincipal {
  const principal = useContext(AdminSessionContext);

  if (!principal) {
    throw new Error(
      "useAdminSession must be used inside AdminSessionProvider",
    );
  }

  return principal;
}
