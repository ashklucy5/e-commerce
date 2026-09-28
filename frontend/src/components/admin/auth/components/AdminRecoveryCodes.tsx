"use client";

import { useState } from "react";

import AdminButton from "@/components/admin/ui/components/AdminButton";

import styles from "../css/AdminRecoveryCodes.module.css";

type AdminRecoveryCodesProps = {
  codes: string[];
  onContinue: () => void;
};

export default function AdminRecoveryCodes({
  codes,
  onContinue,
}: AdminRecoveryCodesProps) {
  const [copied, setCopied] =
    useState(false);

  async function copyCodes() {
    await navigator.clipboard.writeText(
      codes.join("\n"),
    );

    setCopied(true);

    window.setTimeout(() => {
      setCopied(false);
    }, 1800);
  }

  return (
    <div className={styles.root}>
      <div className={styles.warning}>
        Store these recovery codes
        somewhere private. They may
        not be shown again.
      </div>

      <div className={styles.codes}>
        {codes.map((code) => (
          <code
            key={code}
            className={styles.code}
          >
            {code}
          </code>
        ))}
      </div>

      <div className={styles.actions}>
        <AdminButton
          type="button"
          variant="secondary"
          onClick={() => {
            void copyCodes();
          }}
        >
          {copied
            ? "Copied"
            : "Copy codes"}
        </AdminButton>

        <AdminButton
          type="button"
          onClick={onContinue}
        >
          I saved them
        </AdminButton>
      </div>
    </div>
  );
}