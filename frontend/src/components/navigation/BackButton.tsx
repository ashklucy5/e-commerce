"use client";

import { useRouter } from "next/navigation";

import { Icon } from "@/components/ui/Icon";

import styles from "./BackButton.module.css";

type BackButtonProps = {
  fallbackHref?: string;
  label?: string;
};

export function BackButton({
  fallbackHref = "/",
  label = "Back",
}: BackButtonProps) {
  const router = useRouter();

  function handleBack() {
    if (typeof window !== "undefined" && window.history.length > 1) {
      router.back();
      return;
    }

    router.push(fallbackHref);
  }

  return (
    <button
      type="button"
      className={styles.button}
      onClick={handleBack}
      aria-label="Go back to the previous page"
    >
      <Icon name="arrowLeft" size={17} />
      <span>{label}</span>
    </button>
  );
}