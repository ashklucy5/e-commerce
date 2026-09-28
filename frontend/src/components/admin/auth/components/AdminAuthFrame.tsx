import Image from "next/image";
import type { ReactNode } from "react";

import AdminBackground from "@/components/admin/layout/components/AdminBackground";
import AdminCard from "@/components/admin/ui/components/AdminCard";
import { siteConfig } from "@/lib/config/site";

import styles from "../css/AdminAuthFrame.module.css";

type AdminAuthFrameProps = {
  eyebrow: string;
  title: string;
  description: string;
  children: ReactNode;
};

export default function AdminAuthFrame({
  eyebrow,
  title,
  description,
  children,
}: AdminAuthFrameProps) {
  return (
    <main className={styles.page}>
      <AdminBackground />

      <section className={styles.stage}>
        <AdminCard className={styles.card}>
          <div className={styles.brand}>
            <Image
              src={siteConfig.assets.logo}
              alt=""
              width={1100}
              height={439}
              priority
              sizes="9.5rem"
              className={styles.brandLogo}
            />

            <p className={styles.brandMeta}>
              Operations
            </p>
          </div>

          <div className={styles.heading}>
            <p className={styles.eyebrow}>
              {eyebrow}
            </p>

            <h1 className={styles.title}>
              {title}
            </h1>

            <p className={styles.description}>
              {description}
            </p>
          </div>

          {children}

          <footer className={styles.footer}>
            <span className={styles.secureDot} />

            Private commerce operations
          </footer>
        </AdminCard>
      </section>
    </main>
  );
}